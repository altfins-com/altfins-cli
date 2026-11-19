package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func mockPublic(t *testing.T, h http.HandlerFunc) {
	t.Helper()
	isolatedConfig(t)
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	t.Setenv("ALTFINS_API_KEY", "test")
	t.Setenv("ALTFINS_BASE_URL", srv.URL)
}

func TestClassSelectionAndPaging(t *testing.T) {
	calls := 0
	mockPublic(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/symbols") {
			fmt.Fprint(w, `[{"name":"CS_AAPL","friendlyName":"Apple Inc."},{"name":"AAPLX","friendlyName":"Tokenized Apple"},{"name":"FX_EURUSD"},{"name":"USD"},{"name":"CS_MSFT"}]`)
			return
		}
		calls++
		var body struct{ Symbols []string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(body.Symbols, []string{"CS_AAPL", "CS_MSFT"}) || r.URL.Query().Get("page") != "1" || !reflect.DeepEqual(r.URL.Query()["sort"], []string{"RSI14,desc", "LAST_PRICE,asc"}) {
			t.Errorf("wrong selection or paging %v %v", body, r.URL.Query())
		}
		fmt.Fprint(w, `{"content":[{"symbol":"CS_AAPL"}],"totalElements":2,"number":1}`)
	})
	out, err := runCLI(t, nil, "markets", "search", "--asset-class", "stock", "--page", "1", "--sort", "RSI14,desc", "--sort", "LAST_PRICE,asc", "-o", "json")
	if err != nil || !strings.Contains(out, `"totalElements": 2`) {
		t.Fatal(out, err)
	}
	for _, args := range [][]string{{"--asset-class", "stock", "--symbols", "FX_EURUSD,AAPLX,unknown"}, {"--filter", `{"symbols":[]}`}} {
		out, err := runCLI(t, nil, append([]string{"markets", "search", "-o", "json"}, args...)...)
		if err != nil || !strings.Contains(out, `"content": []`) {
			t.Fatal(out, err)
		}
	}
	if calls != 1 {
		t.Fatalf("search calls %d", calls)
	}
	out, err = runCLI(t, nil, "refs", "symbols", "--search", "EUR/USD", "--asset-class", "forex", "-o", "jsonl")
	if err != nil || !strings.Contains(out, "FX_EURUSD") || strings.Contains(out, "CS_") {
		t.Fatal(out, err)
	}
}

func TestFilterCompatibility(t *testing.T) {
	isolatedConfig(t)
	f := t.TempDir() + "/filter.json"
	if err := os.WriteFile(f, []byte(`{"exchangeFilter":["BINANCE"],"supportResistanceLookBackIntervals":"3"}`), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := runCLI(t, nil, "markets", "search", "--filter", "@"+f, "--trading-types", "SPOT", "--dry-run")
	if err != nil || !strings.Contains(out, `"exchangeFilter": "BINANCE"`) || !strings.Contains(out, `"tradingTypeFilter": "SPOT"`) || !strings.Contains(out, `"supportResistanceLookBackIntervals": 3`) {
		t.Fatal(out, err)
	}
	for _, args := range [][]string{{"markets", "search", "--exchange", "A,B"}, {"markets", "search", "--support-resistance-lookback", "6"}, {"markets", "search", "--stdin-json"}, {"calendar", "list", "--filter", `{"myData":"PORTFOLIO_SYMBOLS"}`}, {"calendar", "list", "--my-data", "PORTFOLIO_SYMBOLS"}, {"news", "list", "--filter", `{"symbol":"BTC"}`}, {"tui", "news", "--symbol", "BTC"}} {
		if _, err := runCLI(t, []byte(`{"tradingTypeFilter":["A","B"]}`), append(args, "--dry-run")...); err == nil {
			t.Errorf("accepted %v", args)
		}
	}
}

func TestPatternPageOutput(t *testing.T) {
	fixture, err := os.ReadFile("testdata/pattern-page.json")
	if err != nil {
		t.Fatal(err)
	}
	mockPublic(t, func(w http.ResponseWriter, r *http.Request) { w.Write(fixture) })
	for _, format := range []string{"json", "jsonl", "csv", "table"} {
		out, err := runCLI(t, nil, "patterns", "list", "--symbol", "BTC", "--fields", "patternSubType,relevant", "-o", format)
		if err != nil || !strings.Contains(out, "CHANNEL_DOWN") || !strings.Contains(out, "false") || strings.Contains(out, "analysisText") {
			t.Fatalf("%s %s %v", format, out, err)
		}
		if (format == "json") != strings.Contains(out, "totalPages") {
			t.Fatal(out)
		}
		if format == "csv" {
			rows, err := csv.NewReader(strings.NewReader(out)).ReadAll()
			if err != nil || len(rows) != 2 {
				t.Fatal(rows, err)
			}
		}
	}
}
func TestGenerationGateAndPreview(t *testing.T) {
	calls := 0
	mockPublic(t, func(w http.ResponseWriter, r *http.Request) { calls++ })
	if _, err := runCLI(t, nil, "setups", "generate", "--symbol", "SOL"); err == nil || exitCode(err) != 5 {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"setups", "generate", "--symbol", "SOL"}, {"markets", "search", "--asset-class", "forex"}} {
		out, err := runCLI(t, nil, append(args, "--dry-run")...)
		if err != nil || !strings.Contains(out, "/api/v2/public/") {
			t.Fatal(out, err)
		}
	}
	if calls != 0 {
		t.Fatalf("unexpected network calls %d", calls)
	}
}

func TestLevelsAndLegacyDirection(t *testing.T) {
	levels, err := os.ReadFile("testdata/levels-page.json")
	if err != nil {
		t.Fatal(err)
	}
	mockPublic(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "analytics/"):
			w.Write(levels)
		case strings.Contains(r.URL.Path, "signals-feed/"):
			fmt.Fprint(w, `{"content":[{"bullish":false},{"direction":"BULLISH","bullish":false}]}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	})
	for _, format := range []string{"table", "csv", "json", "jsonl"} {
		out, err := runCLI(t, nil, "analytics", "history", "--symbol", "CS_AAPL", "--type", "SUPPORT_RESISTANCE", "--fields", "supportResistance", "-o", format)
		if err != nil || !strings.Contains(out, "325.32999") || !strings.Contains(out, "null") || !strings.Contains(out, "resistances") {
			t.Fatalf("%s %s %v", format, out, err)
		}
		out, err = runCLI(t, nil, "signals", "list", "--fields", "direction", "-o", format)
		if err != nil || !strings.Contains(out, "BEARISH") || !strings.Contains(out, "BULLISH") {
			t.Fatalf("%s %s %v", format, out, err)
		}
	}
}
