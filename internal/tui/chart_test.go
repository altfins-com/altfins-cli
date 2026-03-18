package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

func TestRenderChartUsesCandlesForMixedOHLC(t *testing.T) {
	state := newChartState("BTC", chartPreset{Interval: "DAILY", Bars: 8}, sampleOHLCV(
		[4]float64{100, 106, 98, 104},
		[4]float64{104, 108, 101, 102},
		[4]float64{102, 110, 100, 109},
		[4]float64{109, 111, 107, 108},
		[4]float64{108, 112, 105, 111},
		[4]float64{111, 114, 109, 110},
		[4]float64{110, 116, 108, 115},
		[4]float64{115, 118, 113, 117},
	), nil)

	rendered := renderChart(state, 80, 18, chartModeCandles)
	if rendered.EffectiveMode != chartModeCandles {
		t.Fatalf("expected candle mode, got %s", rendered.EffectiveMode)
	}
	if !strings.Contains(rendered.Title, "BTC") {
		t.Fatalf("expected symbol in chart title, got %q", rendered.Title)
	}
	if !strings.Contains(rendered.Summary, "O 115") || !strings.Contains(rendered.Summary, "C 117") {
		t.Fatalf("expected latest OHLC summary, got %q", rendered.Summary)
	}
	if strings.TrimSpace(rendered.Body) == "" {
		t.Fatalf("expected rendered candle chart body")
	}
}

func TestRenderChartFallsBackToBrailleWhenNarrow(t *testing.T) {
	state := newChartState("BTC", chartPreset{Interval: "DAILY", Bars: 8}, sampleOHLCV(
		[4]float64{100, 106, 98, 104},
		[4]float64{104, 108, 101, 102},
		[4]float64{102, 110, 100, 109},
		[4]float64{109, 111, 107, 108},
		[4]float64{108, 112, 105, 111},
		[4]float64{111, 114, 109, 110},
		[4]float64{110, 116, 108, 115},
		[4]float64{115, 118, 113, 117},
	), nil)

	rendered := renderChart(state, 32, 10, chartModeCandles)
	if rendered.EffectiveMode != chartModeBraille {
		t.Fatalf("expected braille fallback for narrow chart, got %s", rendered.EffectiveMode)
	}
	if !strings.Contains(rendered.Notice, "braille fallback") {
		t.Fatalf("expected fallback notice, got %q", rendered.Notice)
	}
}

func TestParseCandlesSkipsInvalidRows(t *testing.T) {
	now := time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC)
	items := []altfins.OHLCVData{
		{
			Symbol: "BTC",
			Time:   now,
			Open:   "100",
			High:   "105",
			Low:    "99",
			Close:  "103",
		},
		{
			Symbol: "BTC",
			Time:   now.Add(24 * time.Hour),
			Open:   "oops",
			High:   "107",
			Low:    "100",
			Close:  "104",
		},
		{
			Symbol: "BTC",
			Time:   now.Add(48 * time.Hour),
			Open:   "104",
			High:   "110",
			Low:    "102",
			Close:  "109",
		},
	}

	candles := parseCandles(items)
	if len(candles) != 2 {
		t.Fatalf("expected invalid rows to be skipped, got %d candles", len(candles))
	}
	if !candles[0].Time.Before(candles[1].Time) {
		t.Fatalf("expected candles to be sorted ascending")
	}
}

func sampleOHLCV(values ...[4]float64) []altfins.OHLCVData {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	items := make([]altfins.OHLCVData, 0, len(values))
	for i, item := range values {
		items = append(items, altfins.OHLCVData{
			Symbol: "BTC",
			Time:   start.Add(time.Duration(i) * 24 * time.Hour),
			Open:   trimZeroes(formatAxisPrice(item[0])),
			High:   trimZeroes(formatAxisPrice(item[1])),
			Low:    trimZeroes(formatAxisPrice(item[2])),
			Close:  trimZeroes(formatAxisPrice(item[3])),
		})
	}
	return items
}
