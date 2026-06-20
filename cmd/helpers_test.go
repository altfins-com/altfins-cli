package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPListValueNormalization(t *testing.T) {
	asList := func(v any) []map[string]any {
		l, ok := v.([]map[string]any)
		if !ok {
			t.Fatalf("expected []map[string]any, got %T", v)
		}
		return l
	}

	// bare array
	if got := asList(mcpListValue([]byte(`[{"a":1}]`))); len(got) != 1 {
		t.Errorf("bare array: got %v", got)
	}
	// object wrapping a content array
	if got := asList(mcpListValue([]byte(`{"content":[{"a":1},{"a":2}]}`))); len(got) != 2 {
		t.Errorf("content wrapper: got %v", got)
	}
	// object wrapping a holdings array (portfolio-style)
	if got := asList(mcpListValue([]byte(`{"holdings":[{"symbol":"BTC"}]}`))); got[0]["symbol"] != "BTC" {
		t.Errorf("holdings wrapper: got %v", got)
	}
	// null / empty -> empty slice
	if got := asList(mcpListValue([]byte(`null`))); len(got) != 0 {
		t.Errorf("null: got %v", got)
	}
	// bare object (no known wrapper) -> map
	if _, ok := mcpListValue([]byte(`{"a":1}`)).(map[string]any); !ok {
		t.Error("bare object should remain a map")
	}
}

// newPortfolioMCPServer returns an httptest MCP server that answers the handshake
// and returns the given tool payload as result.content[].text.
func newPortfolioMCPServer(t *testing.T, payload string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var env struct {
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &env)
		w.Header().Set("Content-Type", "application/json")
		switch env.Method {
		case "initialize":
			w.Header().Set("Mcp-Session-Id", "s1")
			io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2024-11-05"}}`)
		case "notifications/initialized":
			w.WriteHeader(http.StatusAccepted)
		case "tools/call":
			resp := map[string]any{"jsonrpc": "2.0", "id": 2, "result": map[string]any{
				"content": []any{map[string]any{"type": "text", "text": payload}},
			}}
			b, _ := json.Marshal(resp)
			w.Write(b)
		}
	}))
}

func TestPortfolioShowFieldsProjection(t *testing.T) {
	srv := newPortfolioMCPServer(t, `[{"symbol":"ETH","walletAddress":"0xABCDEF0123","balance":"1.0"}]`)
	defer srv.Close()
	isolatedConfig(t)
	t.Setenv("ALTFINS_API_KEY", "TESTKEY")
	t.Setenv("ALTFINS_MCP_URL", srv.URL)

	out, err := runCLI(t, nil, "portfolio", "show", "--fields", "symbol", "-o", "json")
	if err != nil {
		t.Fatalf("portfolio show --fields symbol: %v (out=%s)", err, out)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("parse output: %v (out=%s)", err, out)
	}
	if len(got) != 1 || got[0]["symbol"] != "ETH" {
		t.Errorf("expected projected symbol ETH, got %v", got)
	}
	if _, leaked := got[0]["walletAddress"]; leaked {
		t.Error("--fields symbol must not include walletAddress")
	}
}
