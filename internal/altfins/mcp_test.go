package altfins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

const testSessionID = "test-session-abc"

// newMCPTestServer handles the initialize/initialized handshake generically and
// delegates the tools/call response to toolCall, asserting that the session id
// from initialize is propagated to the tool call.
func newMCPTestServer(t *testing.T, toolCall http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("every MCP request must carry a User-Agent (WAF requirement)")
		}
		body, _ := io.ReadAll(r.Body)
		var env struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &env)
		switch env.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", testSessionID)
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"test","version":"1"}}}`)
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			if got := r.Header.Get("Mcp-Session-Id"); got != testSessionID {
				t.Errorf("tools/call Mcp-Session-Id = %q, want %q", got, testSessionID)
			}
			toolCall(w, r)
		default:
			t.Errorf("unexpected MCP method %q", env.Method)
		}
	}))
}

func newTestMCPClient(url string) *MCPClient {
	return NewMCPClient(MCPClientConfig{URL: url, APIKey: "TESTKEY", AuthSource: "config"})
}

func TestMCPCallToolDecodesContentText(t *testing.T) {
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// tool payload is a JSON string inside content[].text
		io.WriteString(w, `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"[{\"symbol\":\"BTC\"}]"}]}}`)
	})
	defer srv.Close()

	var raw json.RawMessage
	if err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, &raw); err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode payload: %v (raw=%s)", err, raw)
	}
	if len(got) != 1 || got[0]["symbol"] != "BTC" {
		t.Errorf("payload = %v, want [{symbol:BTC}]", got)
	}
}

func TestMCPCallToolSSEAndPrettyJSON(t *testing.T) {
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// pretty-printed JSON inside an SSE data event (multi-line)
		io.WriteString(w, "event: message\ndata: {\n  \"jsonrpc\": \"2.0\",\n  \"id\": 2,\n  \"result\": { \"structuredContent\": [ {\"id\": 7} ] }\n}\n\n")
	})
	defer srv.Close()

	var raw json.RawMessage
	if err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, &raw); err != nil {
		t.Fatalf("CallTool over SSE: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("decode SSE payload: %v (raw=%s)", err, raw)
	}
	if len(got) != 1 || got[0]["id"].(float64) != 7 {
		t.Errorf("payload = %v, want [{id:7}]", got)
	}
}

func TestMCPCallToolHTTP403ExitCode3(t *testing.T) {
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, "blocked")
	})
	defer srv.Close()

	err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, new(json.RawMessage))
	mcpErr, ok := IsMCPError(err)
	if !ok {
		t.Fatalf("expected *MCPError, got %T: %v", err, err)
	}
	if mcpErr.ExitCode() != 3 {
		t.Errorf("HTTP 403 exit code = %d, want 3", mcpErr.ExitCode())
	}
}

func TestMCPCallToolJSONRPCErrorExitCode4(t *testing.T) {
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"bad request"}}`)
	})
	defer srv.Close()

	err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, new(json.RawMessage))
	mcpErr, ok := IsMCPError(err)
	if !ok {
		t.Fatalf("expected *MCPError, got %T: %v", err, err)
	}
	if mcpErr.ExitCode() != 4 {
		t.Errorf("JSON-RPC error exit code = %d, want 4", mcpErr.ExitCode())
	}
}

func TestMCPCallToolIsError(t *testing.T) {
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":2,"result":{"isError":true,"content":[{"type":"text","text":"tool failed"}]}}`)
	})
	defer srv.Close()

	err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, new(json.RawMessage))
	if _, ok := IsMCPError(err); !ok {
		t.Fatalf("expected *MCPError for isError result, got %T: %v", err, err)
	}
}

func TestMCPRetryOn500ThenSucceeds(t *testing.T) {
	var calls int32
	srv := newMCPTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"[]"}]}}`)
	})
	defer srv.Close()

	if err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, new(json.RawMessage)); err != nil {
		t.Fatalf("CallTool should succeed after one 500 retry: %v", err)
	}
	if atomic.LoadInt32(&calls) < 2 {
		t.Errorf("tools/call attempts = %d, want >= 2 (retry on 500)", calls)
	}
}

func TestMCPInitializeErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32602,"message":"unsupported protocol"}}`)
	}))
	defer srv.Close()

	err := newTestMCPClient(srv.URL).CallTool(context.Background(), "x", nil, new(json.RawMessage))
	if _, ok := IsMCPError(err); !ok {
		t.Fatalf("initialize error must surface as *MCPError, got %T: %v", err, err)
	}
}

func TestMCPDryRunReturnsPreview(t *testing.T) {
	client := NewMCPClient(MCPClientConfig{URL: "https://mcp.example/mcp", APIKey: "K", DryRun: true})
	err := client.CallTool(context.Background(), "getUserPortfolio", nil, new(json.RawMessage))
	dry, ok := IsDryRun(err)
	if !ok {
		t.Fatalf("dry-run CallTool must return *DryRunError, got %T: %v", err, err)
	}
	if dry.Preview.Headers["X-Api-Key"] != "redacted" {
		t.Errorf("dry-run preview must redact the API key, got %q", dry.Preview.Headers["X-Api-Key"])
	}
	if dry.Preview.Headers["User-Agent"] == "" {
		t.Error("dry-run preview must include a User-Agent")
	}
}

func TestExtractRPCBytes(t *testing.T) {
	cases := map[string]string{
		`{"a":1}`:                              `{"a":1}`,
		"data: {\"a\":1}\n\n":                  `{"a":1}`,
		"event: x\ndata: {\n  \"a\": 1\n}\n\n": "{\n  \"a\": 1\n}",
	}
	for input, want := range cases {
		got := string(extractRPCBytes([]byte(input)))
		if got != want {
			t.Errorf("extractRPCBytes(%q) = %q, want %q", input, got, want)
		}
	}
}
