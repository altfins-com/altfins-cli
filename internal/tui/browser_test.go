package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

func TestBrowserModelToggleModes(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 1, 1, testBrowserItem("BTC", "Bitcoin")))
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
	model, _ := newTestBrowserModel(nil, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 1, 1, testBrowserItem("BTC", "Bitcoin")))
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

func TestLoadPageCmdRequestsPageZero(t *testing.T) {
	model, recorder := newTestBrowserModel(map[int]browserPage{
		0: testBrowserPage(0, 2, 100, testBrowserItem("BTC", "Bitcoin")),
	}, nil)

	msg := model.loadPageCmd(0)().(pageMsg)
	if len(recorder.calls) != 1 {
		t.Fatalf("expected one loader call, got %d", len(recorder.calls))
	}
	if got, want := recorder.calls[0].Page, 0; got != want {
		t.Fatalf("page mismatch: got %d want %d", got, want)
	}
	if got, want := recorder.calls[0].Size, tuiAPIPageSize; got != want {
		t.Fatalf("size mismatch: got %d want %d", got, want)
	}
	if got, want := msg.requestedPage, 0; got != want {
		t.Fatalf("requested page mismatch: got %d want %d", got, want)
	}
}

func TestAutoLoadNearEndLoadsNextPageOnce(t *testing.T) {
	model, recorder := newTestBrowserModel(map[int]browserPage{
		0: testBrowserPage(0, 3, 120, testBrowserItems(50)...),
		1: testBrowserPage(1, 3, 120, testBrowserItemsWithPrefix("P1", 50)...),
	}, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 3, 120, testBrowserItems(50)...))
	m := updated.(browserModel)
	m.list.Select(40)

	cmd := m.autoLoadCmd()
	if cmd == nil {
		t.Fatalf("expected auto-load command near the end of loaded rows")
	}
	msg := cmd().(pageMsg)
	if len(recorder.calls) != 1 {
		t.Fatalf("expected one incremental loader call, got %d", len(recorder.calls))
	}
	if got, want := msg.requestedPage, 1; got != want {
		t.Fatalf("expected page 1 to be requested, got %d", got)
	}
	if got, want := recorder.calls[0].Page, 1; got != want {
		t.Fatalf("expected loader to fetch page %d, got %d", want, got)
	}

	if duplicate := m.autoLoadCmd(); duplicate != nil {
		t.Fatalf("expected duplicate auto-load to be blocked while loading")
	}
}

func TestExplicitNextPageLoadsAndJumps(t *testing.T) {
	model, recorder := newTestBrowserModel(map[int]browserPage{
		1: testBrowserPage(1, 3, 120, testBrowserItemsWithPrefix("P1", 50)...),
	}, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 3, 120, testBrowserItems(50)...))
	m := updated.(browserModel)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(browserModel)
	if !m.incrementalLoading {
		t.Fatalf("expected incremental loading after pressing n")
	}
	if cmd == nil {
		t.Fatalf("expected next-page command after pressing n")
	}

	msg := cmd().(pageMsg)
	if len(recorder.calls) != 1 || recorder.calls[0].Page != 1 {
		t.Fatalf("expected explicit next page to request page 1, got %+v", recorder.calls)
	}

	updated, _ = m.Update(msg)
	m = updated.(browserModel)
	selected := m.selected()
	if selected == nil {
		t.Fatalf("expected a selected item after page jump")
	}
	if got, want := selected.pageNumber, 1; got != want {
		t.Fatalf("expected selection to jump to page %d, got %d", want, got)
	}
}

func TestPreviousPageJumpMovesToPreviousBoundary(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 2, 100, testBrowserItems(50)...))
	m := updated.(browserModel)
	updated, _ = m.Update(testPageMsg(m.generation, 1, 2, 100, testBrowserItemsWithPrefix("P1", 50)...))
	m = updated.(browserModel)
	m.list.Select(50)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(browserModel)
	selected := m.selected()
	if selected == nil {
		t.Fatalf("expected a selected item after jumping to previous page")
	}
	if got, want := selected.pageNumber, 0; got != want {
		t.Fatalf("expected previous page selection, got %d", got)
	}
}

func TestRefreshIgnoresStalePageResponses(t *testing.T) {
	model, _ := newTestBrowserModel(map[int]browserPage{
		1: testBrowserPage(1, 2, 100, testBrowserItemsWithPrefix("P1", 50)...),
	}, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 2, 100, testBrowserItems(50)...))
	m := updated.(browserModel)
	cmd := m.startPageLoad(1, false)
	if cmd == nil {
		t.Fatalf("expected incremental load command")
	}
	stale := cmd().(pageMsg)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(browserModel)
	if got, want := m.generation, model.generation+1; got != want {
		t.Fatalf("expected refresh generation %d, got %d", want, got)
	}

	updated, _ = m.Update(stale)
	m = updated.(browserModel)
	if got := len(m.items); got != 0 {
		t.Fatalf("expected stale page response to be ignored, got %d items", got)
	}
}

func TestLocalFilterDisablesAutoLoad(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 3, 120, testBrowserItems(50)...))
	m := updated.(browserModel)
	m.list.SetFilterText("coin-0")
	m.list.Select(0)

	if cmd := m.autoLoadCmd(); cmd != nil {
		t.Fatalf("expected auto-load to be disabled while local filter is active")
	}
}

func TestExplicitNextPageReappliesFilter(t *testing.T) {
	model, _ := newTestBrowserModel(map[int]browserPage{
		1: testBrowserPage(1, 2, 100,
			testBrowserItem("BTC", "Bitcoin next"),
			testBrowserItem("ETH", "Ethereum next"),
		),
	}, nil)
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 2, 100,
		testBrowserItem("BTC", "Bitcoin"),
		testBrowserItem("ETH", "Ethereum"),
	))
	m := updated.(browserModel)
	m.list.SetFilterText("btc")

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m = updated.(browserModel)
	if cmd == nil {
		t.Fatalf("expected explicit next-page command under active filter")
	}

	updated, _ = m.Update(cmd().(pageMsg))
	m = updated.(browserModel)
	if !m.list.IsFiltered() {
		t.Fatalf("expected local filter to remain active after loading another page")
	}
	if got, want := len(m.list.VisibleItems()), 2; got != want {
		t.Fatalf("expected filtered visible items %d, got %d", want, got)
	}
	selected := m.selected()
	if selected == nil || selected.pageNumber != 1 {
		t.Fatalf("expected selection to jump to the first visible item from page 1")
	}
}

func TestIncrementalPageErrorKeepsLoadedRows(t *testing.T) {
	model, _ := newTestBrowserModel(nil, map[int]error{
		1: errors.New("boom"),
	})
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 2, 100, testBrowserItems(50)...))
	m := updated.(browserModel)
	cmd := m.startPageLoad(1, false)
	if cmd == nil {
		t.Fatalf("expected page load command")
	}

	updated, _ = m.Update(cmd().(pageMsg))
	m = updated.(browserModel)
	if got, want := len(m.items), 50; got != want {
		t.Fatalf("expected loaded rows to be preserved after error, got %d", got)
	}
	if m.err != nil {
		t.Fatalf("expected incremental error to stay non-fatal, got %v", m.err)
	}
	if m.incrementalError == nil {
		t.Fatalf("expected incremental error to be recorded")
	}
	if retry := m.startPageLoad(1, false); retry == nil {
		t.Fatalf("expected retry command after incremental page error")
	}
}

func TestQuotaErrorShowsUnavailableStatus(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)

	updated, _ := model.Update(quotaMsg{err: errors.New("quota failed")})
	m := updated.(browserModel)

	if m.quotaErr == nil {
		t.Fatal("expected quota error to be recorded")
	}
	if got := m.statusLine(); !strings.Contains(got, "quota unavailable") {
		t.Fatalf("expected quota unavailable status, got %q", got)
	}
}

func TestSuccessfulPageLoadUpdatesLastRefresh(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)

	updated, _ := model.Update(testPageMsg(model.generation, 0, 1, 1, testBrowserItem("BTC", "Bitcoin")))
	m := updated.(browserModel)

	if m.lastRefresh.IsZero() {
		t.Fatal("expected successful page load to set last refresh timestamp")
	}
}

func TestChartLoadRetriesAfterCachedError(t *testing.T) {
	model, _ := newTestBrowserModel(nil, nil)
	model.client = altfins.NewClient(altfins.ClientConfig{DryRun: true})
	model.width = 120
	model.height = 40
	model.resize()

	updated, _ := model.Update(testPageMsg(model.generation, 0, 1, 1, testBrowserItem("BTC", "Bitcoin")))
	m := updated.(browserModel)

	preset := m.activeChartPreset()
	key := chartCacheKey("BTC", preset)
	m.charts[key] = chartState{Symbol: "BTC", Interval: preset.Interval, Bars: preset.Bars, Err: errors.New("boom")}
	m.chartLoading = map[string]bool{}

	cmd := m.loadChartForSelection()
	if cmd == nil {
		t.Fatal("expected retry command for cached chart error")
	}
	if _, ok := m.charts[key]; ok {
		t.Fatal("expected cached error state to be cleared before retry")
	}
}

func TestChartCacheKeySeparatesIntervals(t *testing.T) {
	hourly := chartCacheKey("btc", chartPreset{Interval: "HOURLY", Bars: 48})
	daily := chartCacheKey("BTC", chartPreset{Interval: "DAILY", Bars: 30})
	if hourly == daily {
		t.Fatalf("expected distinct cache keys for different chart presets")
	}
}

type requestRecorder struct {
	calls []altfins.Paging
}

func newTestBrowserModel(pages map[int]browserPage, errs map[int]error) (browserModel, *requestRecorder) {
	recorder := &requestRecorder{}
	loader := func(_ context.Context, _ *altfins.Client, _ map[string]any, paging altfins.Paging) (browserPage, error) {
		recorder.calls = append(recorder.calls, paging)
		if errs != nil {
			if err, ok := errs[paging.Page]; ok {
				return browserPage{}, err
			}
		}
		if pages != nil {
			if page, ok := pages[paging.Page]; ok {
				return page, nil
			}
		}
		return browserPage{}, nil
	}

	return newBrowserModel("Test", &Dependencies{
		FilterJSON: "{}",
		Filter:     map[string]any{},
	}, loader, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 30},
		},
		DefaultIndex: 0,
	}), recorder
}

func testPageMsg(generation, page, totalPages int, totalElements int64, items ...browserItem) pageMsg {
	return pageMsg{
		generation:    generation,
		requestedPage: page,
		result:        testBrowserPage(page, totalPages, totalElements, items...),
	}
}

func testBrowserPage(page, totalPages int, totalElements int64, items ...browserItem) browserPage {
	return browserPage{
		Items:            items,
		Page:             page,
		TotalPages:       totalPages,
		TotalElements:    totalElements,
		NumberOfElements: len(items),
		First:            page == 0,
		Last:             page+1 >= totalPages,
	}
}

func testBrowserItems(count int) []browserItem {
	return testBrowserItemsWithPrefix("coin", count)
}

func testBrowserItemsWithPrefix(prefix string, count int) []browserItem {
	items := make([]browserItem, 0, count)
	for i := 0; i < count; i++ {
		items = append(items, testBrowserItem(fmt.Sprintf("%s-%02d", prefix, i), fmt.Sprintf("%s %02d", prefix, i)))
	}
	return items
}

func testBrowserItem(symbol, title string) browserItem {
	return browserItem{
		title:       title,
		description: symbol,
		filterValue: strings.ToLower(title + " " + symbol),
		symbol:      symbol,
		details: map[string]string{
			"symbol": symbol,
		},
	}
}
