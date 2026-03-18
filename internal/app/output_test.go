package app

import (
	"bytes"
	"os"
	"path/filepath"
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
					"MARKET_CAP":  "1000000",
					"PRICE_CHANGE": "2.5",
				},
			},
			{
				Symbol:    "ETH",
				Name:      "Ethereum",
				LastPrice: "4000",
				AdditionalData: map[string]any{
					"MARKET_CAP":  "500000",
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
			if got := buf.String(); got != string(want) {
				t.Fatalf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
			}
		})
	}
}
