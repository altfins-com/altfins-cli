package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

func TestBrowserModelToggleModes(t *testing.T) {
	model := newBrowserModel("Test", &Dependencies{
		FilterJSON: "{}",
		Filter:     map[string]any{},
	}, func(_ context.Context, _ *altfins.Client, _ map[string]any) ([]browserItem, error) {
		return []browserItem{
			{
				title:       "Bitcoin",
				description: "BTC",
				filterValue: "btc",
				symbol:      "BTC",
				details: map[string]string{
					"symbol": "BTC",
				},
			},
		}, nil
	}, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 30},
		},
		DefaultIndex: 2,
	})

	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(itemsMsg{
		items: []browserItem{
			{
				title:       "Bitcoin",
				description: "BTC",
				filterValue: "btc",
				symbol:      "BTC",
				details: map[string]string{
					"symbol": "BTC",
				},
			},
		},
	})
	m := updated.(browserModel)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	m = updated.(browserModel)
	if !m.showFilter {
		t.Fatalf("expected filter drawer to be visible")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(browserModel)
	if !m.detailOnly {
		t.Fatalf("expected detail mode after enter")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(browserModel)
	if m.detailOnly {
		t.Fatalf("expected detail mode to close on esc")
	}
}

func TestBrowserModelChartControls(t *testing.T) {
	model := newBrowserModel("Test", &Dependencies{
		FilterJSON: "{}",
		Filter:     map[string]any{},
	}, func(_ context.Context, _ *altfins.Client, _ map[string]any) ([]browserItem, error) {
		return []browserItem{
			{
				title:       "Bitcoin",
				description: "BTC",
				filterValue: "btc",
				symbol:      "BTC",
				details: map[string]string{
					"symbol": "BTC",
				},
			},
		}, nil
	}, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 30},
		},
		DefaultIndex: 0,
	})

	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(itemsMsg{
		items: []browserItem{
			{
				title:       "Bitcoin",
				description: "BTC",
				filterValue: "btc",
				symbol:      "BTC",
				details: map[string]string{
					"symbol": "BTC",
				},
			},
		},
	})
	m := updated.(browserModel)

	if m.chartMode != chartModeCandles {
		t.Fatalf("expected candle mode by default")
	}
	if preset := m.activeChartPreset(); preset.Interval != "HOURLY" {
		t.Fatalf("expected hourly default, got %s", preset.Interval)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	m = updated.(browserModel)
	if m.chartMode != chartModeBraille {
		t.Fatalf("expected braille mode after c toggle")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	m = updated.(browserModel)
	if preset := m.activeChartPreset(); preset.Interval != "HOURS4" {
		t.Fatalf("expected hours4 preset after interval cycle, got %s", preset.Interval)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = updated.(browserModel)
	if !m.chartZoom || !m.detailOnly {
		t.Fatalf("expected zoom mode to enable detail-only chart view")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(browserModel)
	if m.chartZoom || m.detailOnly {
		t.Fatalf("expected esc to exit chart zoom mode back to split view")
	}
}

func TestChartCacheKeySeparatesIntervals(t *testing.T) {
	hourly := chartCacheKey("btc", chartPreset{Interval: "HOURLY", Bars: 48})
	daily := chartCacheKey("BTC", chartPreset{Interval: "DAILY", Bars: 30})
	if hourly == daily {
		t.Fatalf("expected distinct cache keys for different chart presets")
	}
}
