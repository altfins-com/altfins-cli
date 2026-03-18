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
