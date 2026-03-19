package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

func TestWriteOutputGolden(t *testing.T) {
	data := altfins.Page[altfins.ScreenerSearchResult]{
		Content: []altfins.ScreenerSearchResult{
			{
				Symbol:    "BTC",
				Name:      "Bitcoin",
				LastPrice: "70000",
				AdditionalData: map[string]any{
					"MARKET_CAP":   "1000000",
					"PRICE_CHANGE": "2.5",
				},
			},
			{
				Symbol:    "ETH",
				Name:      "Ethereum",
				LastPrice: "4000",
				AdditionalData: map[string]any{
					"MARKET_CAP":   "500000",
					"PRICE_CHANGE": "1.2",
				},
			},
		},
	}

	cases := []struct {
		name   string
		format string
		golden string
	}{
		{name: "table", format: "table", golden: "markets_table.golden"},
		{name: "csv", format: "csv", golden: "markets_csv.golden"},
		{name: "jsonl", format: "jsonl", golden: "markets_jsonl.golden"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteOutput(&buf, data, tc.format, nil); err != nil {
				t.Fatalf("write output: %v", err)
			}
			goldenPath := filepath.Join("testdata", tc.golden)
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden: %v", err)
			}
			if got := normalizeNewlines(buf.String()); got != normalizeNewlines(string(want)) {
				t.Fatalf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, normalizeNewlines(string(want)))
			}
		})
	}
}

func normalizeNewlines(value string) string {
	return strings.ReplaceAll(value, "\r\n", "\n")
}

func TestWriteOutputJSONFieldsProjectsPermitsInfo(t *testing.T) {
	var buf bytes.Buffer
	data := altfins.PermitsInfo{
		AvailablePermits:        12,
		MonthlyAvailablePermits: 99,
	}

	if err := WriteOutput(&buf, data, "json", []string{"availablePermits"}); err != nil {
		t.Fatalf("write output: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one projected field, got %v", got)
	}
	if got["availablePermits"] != float64(12) {
		t.Fatalf("expected projected permits value, got %v", got["availablePermits"])
	}
}

func TestWriteOutputJSONFieldsProjectsPageContent(t *testing.T) {
	var buf bytes.Buffer
	data := altfins.Page[altfins.ScreenerSearchResult]{
		Size:             1,
		Number:           0,
		TotalPages:       1,
		TotalElements:    1,
		NumberOfElements: 1,
		First:            true,
		Last:             true,
		Content: []altfins.ScreenerSearchResult{
			{
				Symbol:    "BTC",
				Name:      "Bitcoin",
				LastPrice: "70000",
				AdditionalData: map[string]any{
					"RSI14": "68.2",
				},
			},
		},
	}

	if err := WriteOutput(&buf, data, "json", []string{"symbol", "RSI14"}); err != nil {
		t.Fatalf("write output: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}

	content, ok := got["content"].([]any)
	if !ok || len(content) != 1 {
		t.Fatalf("expected projected page content, got %v", got["content"])
	}
	row, ok := content[0].(map[string]any)
	if !ok {
		t.Fatalf("expected row object, got %T", content[0])
	}
	if row["symbol"] != "BTC" || row["RSI14"] != "68.2" {
		t.Fatalf("unexpected projected row: %v", row)
	}
	if got["totalElements"] != float64(1) {
		t.Fatalf("expected page metadata to stay intact, got %v", got["totalElements"])
	}
}

func TestWriteOutputJSONLFieldsProjectsItems(t *testing.T) {
	var buf bytes.Buffer
	data := altfins.Page[altfins.ScreenerSearchResult]{
		Content: []altfins.ScreenerSearchResult{
			{
				Symbol: "BTC",
				AdditionalData: map[string]any{
					"RSI14": "68.2",
				},
			},
		},
	}

	if err := WriteOutput(&buf, data, "jsonl", []string{"symbol", "RSI14"}); err != nil {
		t.Fatalf("write output: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected one jsonl line, got %d", len(lines))
	}

	var row map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("decode row: %v", err)
	}
	if row["symbol"] != "BTC" || row["RSI14"] != "68.2" {
		t.Fatalf("unexpected projected jsonl row: %v", row)
	}
}
