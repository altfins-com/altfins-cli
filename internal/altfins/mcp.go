package altfins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// mcpUserAgent is sent on every MCP request. The altFINS MCP server sits behind
// a WAF that returns 403 to requests without a User-Agent, so this is mandatory
// (the REST client sends none and must stay that way).
const mcpUserAgent = "altfins-cli"

const mcpProtocolVersion = "2024-11-05"

// MCPClientConfig configures the thin MCP (Streamable HTTP / JSON-RPC) client used
// by the MCP-backed commands (calendar, portfolio). It shares the REST client's
// API key and dry-run semantics but speaks a different protocol to a different host.
type MCPClientConfig struct {
	URL        string
	APIKey     string
	AuthSource string
	DryRun     bool
	HTTPClient *http.Client
}

// MCPClient calls altFINS MCP tools via JSON-RPC tools/call. It is intentionally
// isolated from the REST Client so the REST transport stays untouched; a future
// REST backend for these capabilities can replace MCPClient behind CallTool.
type MCPClient struct {
	url        string
	apiKey     string
	authSource string
	dryRun     bool
	httpClient *http.Client
}

// MCPError is returned for both HTTP-layer (WAF/transport) and JSON-RPC-layer
// failures of an MCP call. HTTPStatus is 0 for pure JSON-RPC application errors.
type MCPError struct {
	HTTPStatus int    `json:"httpStatus,omitempty"`
	RPCCode    int    `json:"rpcCode,omitempty"`
	Message    string `json:"message"`
}

func (e *MCPError) Error() string {
	if e.HTTPStatus != 0 {
		return fmt.Sprintf("altFINS MCP error: HTTP %d: %s", e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("altFINS MCP error: %s", e.Message)
}

// ExitCode maps auth/permission failures to 3 (matching the REST APIError), and
// every other MCP failure to 4.
func (e *MCPError) ExitCode() int {
	switch e.HTTPStatus {
	case http.StatusUnauthorized, http.StatusForbidden:
		return 3
	}
	// JSON-RPC auth/permission codes are not standardized; -32001..-32003 are
	// commonly used for auth/permission. Treat them as auth failures.
	switch e.RPCCode {
	case -32001, -32002, -32003:
		return 3
	}
	return 4
}

func NewMCPClient(cfg MCPClientConfig) *MCPClient {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		url = "https://mcp.altfins.com/mcp"
	}
	return &MCPClient{
		url:        url,
		apiKey:     strings.TrimSpace(cfg.APIKey),
		authSource: strings.TrimSpace(cfg.AuthSource),
		dryRun:     cfg.DryRun,
		httpClient: httpClient,
	}
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

type mcpToolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent"`
	IsError           bool            `json:"isError"`
}

func rpcEnvelope(id int, method string, params any) map[string]any {
	env := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
	}
	if id > 0 {
		env["id"] = id
	}
	if params != nil {
		env["params"] = params
	}
	return env
}

func (c *MCPClient) previewFor(tool string, args map[string]any) RequestPreview {
	headers := map[string]string{
		"Accept":       "application/json, text/event-stream",
		"Content-Type": "application/json",
		"User-Agent":   mcpUserAgent,
	}
	if c.apiKey != "" {
		headers["X-Api-Key"] = "redacted"
	}
	return RequestPreview{
		Method:     http.MethodPost,
		URL:        c.url,
		Body:       rpcEnvelope(1, "tools/call", map[string]any{"name": tool, "arguments": args}),
		Headers:    headers,
		AuthSource: c.authSource,
	}
}

// CallTool invokes an MCP tool and decodes its payload into out. On dry-run it
// returns a *DryRunError (the same type REST commands use) so the caller's
// HandleCommandResult prints an identical preview.
func (c *MCPClient) CallTool(ctx context.Context, tool string, args map[string]any, out any) error {
	if args == nil {
		args = map[string]any{}
	}
	if c.dryRun {
		return &DryRunError{Preview: c.previewFor(tool, args)}
	}

	// Streamable-HTTP MCP requires an initialize handshake before tool calls.
	initParams := map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": mcpUserAgent, "version": "1"},
	}
	_, sessionID, err := c.roundtrip(ctx, rpcEnvelope(1, "initialize", initParams), "")
	if err != nil {
		return err
	}
	// Best-effort initialized notification (no id, no result expected).
	_, _, _ = c.roundtrip(ctx, rpcEnvelope(0, "notifications/initialized", map[string]any{}), sessionID)

	callParams := map[string]any{"name": tool, "arguments": args}
	resp, _, err := c.roundtrip(ctx, rpcEnvelope(2, "tools/call", callParams), sessionID)
	if err != nil {
		return err
	}

	var rpc rpcResponse
	if err := json.Unmarshal(resp, &rpc); err != nil {
		return &MCPError{Message: fmt.Sprintf("decode MCP response: %v", err)}
	}
	if rpc.Error != nil {
		return &MCPError{RPCCode: rpc.Error.Code, Message: rpc.Error.Message}
	}

	var result mcpToolResult
	if err := json.Unmarshal(rpc.Result, &result); err != nil {
		// Some servers may return the payload directly; fall back to that.
		if out != nil && len(rpc.Result) > 0 {
			return json.Unmarshal(rpc.Result, out)
		}
		return &MCPError{Message: fmt.Sprintf("decode MCP tool result: %v", err)}
	}

	payload := result.StructuredContent
	if len(payload) == 0 {
		var sb strings.Builder
		for _, block := range result.Content {
			if block.Type == "text" || block.Type == "" {
				sb.WriteString(block.Text)
			}
		}
		payload = json.RawMessage(strings.TrimSpace(sb.String()))
	}
	if result.IsError {
		msg := strings.TrimSpace(string(payload))
		if msg == "" {
			msg = "tool reported an error"
		}
		return &MCPError{Message: msg}
	}
	if out == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return &MCPError{Message: fmt.Sprintf("decode MCP tool payload: %v", err)}
	}
	return nil
}

// roundtrip performs one JSON-RPC POST and returns the response body plus the
// session id the server assigned (read from the Mcp-Session-Id header). HTTP
// 429/5xx are retried with backoff; non-2xx becomes an MCPError.
func (c *MCPClient) roundtrip(ctx context.Context, payload map[string]any, sessionID string) ([]byte, string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", &MCPError{Message: fmt.Sprintf("marshal MCP request: %v", err)}
	}

	var lastErr error
	for attempt := range 3 {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, "", &MCPError{Message: fmt.Sprintf("build MCP request: %v", err)}
		}
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", mcpUserAgent)
		if c.apiKey != "" {
			req.Header.Set("X-Api-Key", c.apiKey)
		}
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, "", &MCPError{Message: fmt.Sprintf("execute MCP request: %v", err)}
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			lastErr = &MCPError{HTTPStatus: resp.StatusCode, Message: "MCP server temporarily unavailable"}
			if attempt < 2 {
				time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
				continue
			}
			return nil, "", lastErr
		}

		newSession := resp.Header.Get("Mcp-Session-Id")
		if newSession == "" {
			newSession = sessionID
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, newSession, &MCPError{HTTPStatus: resp.StatusCode, Message: mcpErrorBody(raw)}
		}
		return extractRPCBytes(raw), newSession, nil
	}
	return nil, "", lastErr
}

// extractRPCBytes returns the JSON object from an MCP response body, which may be
// either a plain JSON object or a Server-Sent-Events stream (`data: {...}` lines).
func extractRPCBytes(raw []byte) []byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		return trimmed
	}
	var last []byte
	for _, line := range bytes.Split(raw, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimSpace(line[len("data:"):])
		}
		if len(line) > 0 && line[0] == '{' {
			last = line
		}
	}
	return last
}

func mcpErrorBody(raw []byte) string {
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		return "request rejected"
	}
	if len(msg) > 300 {
		msg = msg[:300]
	}
	return msg
}

// IsMCPError reports whether err is an *MCPError.
func IsMCPError(err error) (*MCPError, bool) {
	var out *MCPError
	if errors.As(err, &out) {
		return out, true
	}
	return nil, false
}
