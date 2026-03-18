package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

const lazyLoadThreshold = 10

type Dependencies struct {
	Client     *altfins.Client
	AuthSource string
	Filter     map[string]any
	FilterJSON string
}

type Runner interface {
	Run() error
}

type runner struct {
	model tea.Model
}

func (r runner) Run() error {
	_, err := tea.NewProgram(r.model, tea.WithAltScreen()).Run()
	return err
}

type browserItem struct {
	title       string
	description string
	filterValue string
	symbol      string
	details     map[string]string
	pageNumber  int
}

func (i browserItem) Title() string       { return i.title }
func (i browserItem) Description() string { return i.description }
func (i browserItem) FilterValue() string { return i.filterValue }

type browserPage struct {
	Items            []browserItem
	Page             int
	TotalPages       int
	TotalElements    int64
	NumberOfElements int
	First            bool
	Last             bool
}

type pageMsg struct {
	generation    int
	requestedPage int
	result        browserPage
	err           error
}

type quotaMsg struct {
	quota altfins.PermitsInfo
	err   error
}

type chartMsg struct {
	key   string
	state chartState
}

type loadFn func(context.Context, *altfins.Client, map[string]any, altfins.Paging) (browserPage, error)

type browserModel struct {
	title                 string
	client                *altfins.Client
	authSource            string
	filter                map[string]any
	filterJSON            string
	load                  loadFn
	list                  list.Model
	loading               bool
	err                   error
	quota                 altfins.PermitsInfo
	width                 int
	height                int
	showFilter            bool
	detailOnly            bool
	focus                 string
	lastRefresh           time.Time
	chartConfig           chartConfig
	chartMode             chartMode
	chartPresetIndex      int
	chartZoom             bool
	zoomRestoreDetailOnly bool
	charts                map[string]chartState
	chartLoading          map[string]bool

	generation           int
	items                []browserItem
	loadedPages          map[int]bool
	pageStarts           map[int]int
	nextPage             int
	totalPages           int
	totalElements        int64
	loadingPage          int
	incrementalLoading   bool
	incrementalError     error
	incrementalErrorPage int
	pendingJumpPage      int
}

func newBrowserModel(title string, deps *Dependencies, loader loadFn, chartCfg chartConfig) browserModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.Title = title
	listModel.SetFilteringEnabled(true)
	listModel.SetShowHelp(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowPagination(false)

	defaultIndex := chartCfg.DefaultIndex
	if defaultIndex < 0 || defaultIndex >= len(chartCfg.Presets) {
		defaultIndex = 0
	}

	return browserModel{
		title:                title,
		client:               deps.Client,
		authSource:           deps.AuthSource,
		filter:               deps.Filter,
		filterJSON:           deps.FilterJSON,
		load:                 loader,
		list:                 listModel,
		loading:              true,
		focus:                "list",
		chartConfig:          chartCfg,
		chartMode:            chartModeCandles,
		chartPresetIndex:     defaultIndex,
		charts:               map[string]chartState{},
		chartLoading:         map[string]bool{},
		generation:           1,
		loadedPages:          map[int]bool{},
		pageStarts:           map[int]int{},
		nextPage:             0,
		loadingPage:          0,
		incrementalErrorPage: -1,
		pendingJumpPage:      -1,
	}
}

func (m browserModel) Init() tea.Cmd {
	return tea.Batch(m.loadPageCmd(0), m.quotaCmd())
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case pageMsg:
		return m.handlePageMsg(msg)
	case quotaMsg:
		if msg.err == nil {
			m.quota = msg.quota
		}
		return m, nil
	case chartMsg:
		delete(m.chartLoading, msg.key)
		m.charts[msg.key] = msg.state
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if !m.list.SettingFilter() {
			switch msg.String() {
			case "q":
				return m, tea.Quit
			case "r":
				m.generation++
				m.resetPageState()
				m.loading = true
				m.err = nil
				m.loadingPage = 0
				m.charts = map[string]chartState{}
				m.chartLoading = map[string]bool{}
				return m, tea.Batch(m.loadPageCmd(0), m.quotaCmd())
			case "n":
				return m, m.startPageLoad(m.nextPage, true)
			case "p":
				if m.jumpToPreviousLoadedPage() {
					return m, m.loadChartForSelection()
				}
				return m, nil
			case "tab":
				if m.focus == "list" {
					m.focus = "detail"
				} else {
					m.focus = "list"
				}
				return m, nil
			case "f":
				m.showFilter = !m.showFilter
				return m, nil
			case "enter":
				if m.selected() != nil {
					m.detailOnly = true
				}
				return m, nil
			case "z":
				if !m.chartConfig.Enabled || m.selected() == nil || m.selected().symbol == "" {
					return m, nil
				}
				if m.chartZoom {
					m.chartZoom = false
					m.detailOnly = m.zoomRestoreDetailOnly
				} else {
					m.zoomRestoreDetailOnly = m.detailOnly
					m.detailOnly = true
					m.chartZoom = true
				}
				return m, m.loadChartForSelection()
			case "c":
				if !m.chartConfig.Enabled {
					return m, nil
				}
				if m.chartMode == chartModeCandles {
					m.chartMode = chartModeBraille
				} else {
					m.chartMode = chartModeCandles
				}
				return m, nil
			case "i":
				if !m.chartConfig.Enabled || len(m.chartConfig.Presets) == 0 {
					return m, nil
				}
				m.chartPresetIndex = (m.chartPresetIndex + 1) % len(m.chartConfig.Presets)
				return m, m.loadChartForSelection()
			case "esc", "backspace":
				if m.chartZoom {
					m.chartZoom = false
					m.detailOnly = m.zoomRestoreDetailOnly
					return m, nil
				}
				if m.detailOnly {
					m.detailOnly = false
					return m, nil
				}
			}
		}
	}

	beforeSelection := m.selectedKey()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	afterSelection := m.selectedKey()

	cmds := []tea.Cmd{cmd}
	if beforeSelection != afterSelection {
		cmds = append(cmds, m.loadChartForSelection())
		if autoCmd := m.autoLoadCmd(); autoCmd != nil {
			cmds = append(cmds, autoCmd)
		}
	}
	return m, batchCmds(cmds...)
}

func (m browserModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	header := lipgloss.NewStyle().Bold(true).Padding(0, 1).Render(m.title)
	status := m.statusLine()
	content := ""

	switch {
	case m.err != nil:
		content = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.err.Error())
	case m.loading:
		content = "Loading data..."
	default:
		content = m.renderBody()
	}

	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Padding(0, 1).
		Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}

func (m browserModel) renderBody() string {
	bodyHeight := max(10, m.height-4)

	if m.chartZoom && m.chartConfig.Enabled {
		return m.renderZoomBody(m.width, bodyHeight)
	}

	if m.detailOnly {
		if m.chartConfig.Enabled {
			return m.renderChartDetailColumn(m.width, bodyHeight)
		}
		blocks := []string{m.renderStandardDetailBox(m.width, bodyHeight)}
		if m.showFilter {
			blocks = append(blocks, m.renderFilterBox(m.width, 10))
		}
		return lipgloss.JoinVertical(lipgloss.Left, blocks...)
	}

	leftWidth := max(34, m.width/2-1)
	rightWidth := max(42, m.width-leftWidth-1)

	left := lipgloss.NewStyle().
		Width(leftWidth).
		Height(bodyHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.borderColor("list")).
		Render(m.list.View())

	var right string
	if m.chartConfig.Enabled {
		right = m.renderChartDetailColumn(rightWidth, bodyHeight)
	} else {
		rightParts := []string{
			m.renderStandardDetailBox(rightWidth, bodyHeight),
		}
		if m.showFilter {
			rightParts = append(rightParts, m.renderFilterBox(rightWidth, 10))
		}
		right = lipgloss.JoinVertical(lipgloss.Left, rightParts...)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m browserModel) renderChartDetailColumn(width, height int) string {
	filterHeight := 0
	if m.showFilter {
		filterHeight = 10
	}
	available := max(14, height-filterHeight)
	metadataHeight := max(10, available/3)
	chartHeight := max(14, available-metadataHeight-1)

	blocks := []string{
		m.renderChartBox(width, chartHeight),
		m.renderMetadataBox(width, metadataHeight),
	}
	if m.showFilter {
		blocks = append(blocks, m.renderFilterBox(width, filterHeight))
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

func (m browserModel) renderZoomBody(width, height int) string {
	filterHeight := 0
	if m.showFilter {
		filterHeight = 10
	}
	summaryHeight := 7
	available := max(16, height-filterHeight)
	chartHeight := max(16, available-summaryHeight-1)

	blocks := []string{
		m.renderChartBox(width, chartHeight),
		m.renderZoomSummaryBox(width, summaryHeight),
	}
	if m.showFilter {
		blocks = append(blocks, m.renderFilterBox(width, filterHeight))
	}
	return lipgloss.JoinVertical(lipgloss.Left, blocks...)
}

func (m browserModel) renderStandardDetailBox(width, height int) string {
	return m.panel(width, height, m.borderColor("detail"), m.renderMetadata())
}

func (m browserModel) renderChartBox(width, height int) string {
	selected := m.selected()
	contentLines := []string{}
	title := "Chart"

	if selected == nil {
		contentLines = append(contentLines, "No selection.")
	} else if selected.symbol == "" {
		contentLines = append(contentLines, "This item does not expose a tradable symbol.")
	} else {
		chartWidth := max(16, width-4)
		chartHeight := max(6, height-8)
		rendered := m.currentRenderedChart(chartWidth, chartHeight)
		contentLines = append(contentLines, lipgloss.NewStyle().Bold(true).Render(rendered.Title))
		if rendered.Summary != "" {
			contentLines = append(contentLines, chartDimStyle.Render(rendered.Summary))
		}
		if rendered.Notice != "" {
			contentLines = append(contentLines, chartDimStyle.Render(rendered.Notice))
		}
		contentLines = append(contentLines, "", rendered.Body)
	}

	return m.panel(width, height, m.borderColor("detail"), title+"\n"+strings.Join(contentLines, "\n"))
}

func (m browserModel) renderMetadataBox(width, height int) string {
	return m.panel(width, height, m.borderColor("detail"), m.renderMetadata())
}

func (m browserModel) renderZoomSummaryBox(width, height int) string {
	selected := m.selected()
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render("Chart Focus"),
	}
	if selected != nil {
		lines = append(lines, selected.title)
		if selected.description != "" {
			lines = append(lines, chartDimStyle.Render(selected.description))
		}
	}
	preset := m.activeChartPreset()
	if preset.Interval != "" {
		lines = append(lines, chartDimStyle.Render(fmt.Sprintf("Mode %s  |  Interval %s x%d  |  c toggle  |  i cycle  |  Esc back", strings.ToUpper(string(m.chartMode)), preset.Interval, preset.Bars)))
	}
	return m.panel(width, height, lipgloss.Color("12"), strings.Join(lines, "\n"))
}

func (m browserModel) renderMetadata() string {
	selected := m.selected()
	if selected == nil {
		return "No items loaded."
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(selected.title),
	}
	if selected.description != "" {
		lines = append(lines, selected.description, "")
	}

	keys := make([]string, 0, len(selected.details))
	for key := range selected.details {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s: %s", key, selected.details[key]))
	}
	return strings.Join(lines, "\n")
}

func (m browserModel) renderFilterBox(width, height int) string {
	return m.panel(width, height, lipgloss.Color("12"), lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Render("Active Filter")+"\n"+m.filterJSON)
}

func (m browserModel) panel(width, height int, border lipgloss.Color, content string) string {
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Render(content)
}

func (m browserModel) selected() *browserItem {
	item, ok := m.list.SelectedItem().(browserItem)
	if !ok {
		return nil
	}
	return &item
}

func (m browserModel) selectedKey() string {
	selected := m.selected()
	if selected == nil {
		return ""
	}
	return fmt.Sprintf("%d|%s|%s", selected.pageNumber, selected.symbol, selected.title)
}

func (m browserModel) quotaCmd() tea.Cmd {
	return func() tea.Msg {
		quota, err := m.client.AllAvailablePermits(context.Background())
		return quotaMsg{quota: quota, err: err}
	}
}

func (m *browserModel) loadPageCmd(page int) tea.Cmd {
	generation := m.generation
	return func() tea.Msg {
		result, err := m.load(context.Background(), m.client, m.filter, altfins.Paging{
			Page: page,
			Size: tuiAPIPageSize,
		})
		return pageMsg{
			generation:    generation,
			requestedPage: page,
			result:        result,
			err:           err,
		}
	}
}

func (m browserModel) handlePageMsg(msg pageMsg) (tea.Model, tea.Cmd) {
	if msg.generation != m.generation {
		return m, nil
	}

	isInitialLoad := msg.requestedPage == 0 && len(m.items) == 0
	m.loadingPage = -1

	if msg.err != nil {
		if isInitialLoad {
			m.loading = false
			m.err = msg.err
		} else {
			m.incrementalLoading = false
			m.incrementalError = msg.err
			m.incrementalErrorPage = msg.requestedPage
			if m.pendingJumpPage == msg.requestedPage {
				m.pendingJumpPage = -1
			}
		}
		return m, nil
	}

	m.loading = false
	m.err = nil
	m.incrementalLoading = false
	m.incrementalError = nil
	m.incrementalErrorPage = -1
	if m.loadedPages[msg.result.Page] {
		return m, nil
	}

	m.appendPage(msg.result)
	m.syncListItems()

	selectionChanged := false
	if isInitialLoad {
		m.list.ResetSelected()
		selectionChanged = true
	}
	if m.pendingJumpPage >= 0 {
		if m.jumpToPage(m.pendingJumpPage) {
			selectionChanged = true
		}
		m.pendingJumpPage = -1
	}

	if selectionChanged {
		return m, m.loadChartForSelection()
	}
	return m, nil
}

func (m *browserModel) startPageLoad(page int, jump bool) tea.Cmd {
	if page < 0 || m.loadingPage >= 0 {
		return nil
	}
	if m.loadedPages[page] {
		return nil
	}
	if m.totalPages > 0 && page >= m.totalPages {
		return nil
	}

	if len(m.items) == 0 && page == 0 {
		m.loading = true
	} else {
		m.incrementalLoading = true
	}
	m.incrementalError = nil
	m.incrementalErrorPage = -1
	m.loadingPage = page
	if jump {
		m.pendingJumpPage = page
	}
	return m.loadPageCmd(page)
}

func (m *browserModel) appendPage(page browserPage) {
	start := len(m.items)
	items := make([]browserItem, 0, len(page.Items))
	for _, item := range page.Items {
		item.pageNumber = page.Page
		items = append(items, item)
	}

	m.items = append(m.items, items...)
	m.loadedPages[page.Page] = true
	m.pageStarts[page.Page] = start
	m.totalPages = page.TotalPages
	m.totalElements = page.TotalElements

	switch {
	case page.Last:
		m.nextPage = -1
	case page.TotalPages == 0:
		m.nextPage = -1
	case page.Page+1 >= page.TotalPages:
		m.nextPage = -1
	default:
		m.nextPage = page.Page + 1
	}
}

func (m *browserModel) syncListItems() {
	items := make([]list.Item, 0, len(m.items))
	for _, item := range m.items {
		items = append(items, item)
	}
	if cmd := m.list.SetItems(items); cmd != nil {
		if msg := cmd(); msg != nil {
			var next tea.Cmd
			m.list, next = m.list.Update(msg)
			_ = next
		}
	}
}

func (m *browserModel) loadChartForSelection() tea.Cmd {
	if !m.chartConfig.Enabled || m.client == nil {
		return nil
	}
	selected := m.selected()
	if selected == nil || selected.symbol == "" {
		return nil
	}
	preset := m.activeChartPreset()
	if preset.Interval == "" || preset.Bars <= 0 {
		return nil
	}
	key := chartCacheKey(selected.symbol, preset)
	if _, ok := m.charts[key]; ok {
		return nil
	}
	if m.chartLoading[key] {
		return nil
	}
	m.chartLoading[key] = true

	symbol := selected.symbol
	return func() tea.Msg {
		page, err := m.client.OHLCVHistory(context.Background(), altfins.Paging{Size: preset.Bars}, map[string]any{
			"symbol":       symbol,
			"timeInterval": preset.Interval,
		})
		return chartMsg{
			key:   key,
			state: newChartState(symbol, preset, page.Content, err),
		}
	}
}

func (m *browserModel) autoLoadCmd() tea.Cmd {
	if m.loading || m.incrementalLoading || m.loadingPage >= 0 || m.nextPage < 0 {
		return nil
	}
	if m.hasLocalFilter() {
		return nil
	}
	if len(m.items) == 0 || m.selected() == nil {
		return nil
	}
	globalIndex := m.list.GlobalIndex()
	if globalIndex < 0 {
		return nil
	}
	if len(m.items)-globalIndex <= lazyLoadThreshold {
		return m.startPageLoad(m.nextPage, false)
	}
	return nil
}

func (m *browserModel) jumpToPreviousLoadedPage() bool {
	selected := m.selected()
	if selected == nil {
		return false
	}
	if selected.pageNumber <= 0 {
		return false
	}
	return m.jumpToPage(selected.pageNumber - 1)
}

func (m *browserModel) jumpToPage(page int) bool {
	if page < 0 {
		return false
	}
	if !m.loadedPages[page] {
		return false
	}
	if !m.hasLocalFilter() {
		start, ok := m.pageStarts[page]
		if !ok {
			return false
		}
		m.list.Select(start)
		return true
	}
	for index, item := range m.list.VisibleItems() {
		typed, ok := item.(browserItem)
		if !ok {
			continue
		}
		if typed.pageNumber == page {
			m.list.Select(index)
			return true
		}
	}
	return false
}

func (m *browserModel) resetPageState() {
	m.items = nil
	m.loadedPages = map[int]bool{}
	m.pageStarts = map[int]int{}
	m.nextPage = 0
	m.totalPages = 0
	m.totalElements = 0
	m.loadingPage = -1
	m.incrementalLoading = false
	m.incrementalError = nil
	m.incrementalErrorPage = -1
	m.pendingJumpPage = -1
	m.list.ResetSelected()
	m.syncListItems()
}

func (m *browserModel) resize() {
	height := m.height - 4
	if height < 10 {
		height = 10
	}
	width := max(30, m.width/2-4)
	m.list.SetSize(width, height)
}

func (m browserModel) borderColor(section string) lipgloss.Color {
	if m.focus == section {
		return lipgloss.Color("10")
	}
	return lipgloss.Color("8")
}

func (m browserModel) statusLine() string {
	parts := []string{
		"j/k or arrows move",
		"/ search",
		"n next api page",
		"p prev api page",
		"Enter detail",
		"Esc back",
		"Tab focus",
		"f filter",
	}
	if m.chartConfig.Enabled {
		parts = append(parts, "z zoom", "c chart mode", "i interval")
	}
	parts = append(parts,
		"r refresh",
		m.loadedStatus(),
		m.apiPageStatus(),
	)
	if m.hasLocalFilter() {
		parts = append(parts, "searching loaded rows only")
	}
	if m.incrementalLoading && m.loadingPage >= 0 {
		totalPages := max(1, m.totalPages)
		parts = append(parts, fmt.Sprintf("loading page %d/%d", m.loadingPage+1, totalPages))
	}
	if m.incrementalError != nil && m.incrementalErrorPage >= 0 {
		parts = append(parts, fmt.Sprintf("page %d load failed", m.incrementalErrorPage+1))
	}
	parts = append(parts, fmt.Sprintf("permits %d/%d", m.quota.AvailablePermits, m.quota.MonthlyAvailablePermits))
	if m.chartConfig.Enabled {
		preset := m.activeChartPreset()
		if preset.Interval != "" {
			parts = append(parts, fmt.Sprintf("chart %s", strings.ToUpper(string(m.chartMode))))
			parts = append(parts, fmt.Sprintf("%s x%d", preset.Interval, preset.Bars))
		}
	}
	if !m.lastRefresh.IsZero() {
		parts = append(parts, "updated "+m.lastRefresh.Format("15:04:05"))
	}
	if m.authSource != "" {
		parts = append(parts, "auth "+m.authSource)
	}
	return strings.Join(parts, "  |  ")
}

func (m browserModel) loadedStatus() string {
	return fmt.Sprintf("loaded %d/%d", len(m.items), m.totalElements)
}

func (m browserModel) apiPageStatus() string {
	totalPages := m.totalPages
	if totalPages == 0 && (m.loading || m.loadingPage >= 0) {
		totalPages = 1
	}
	currentPage := 0
	if selected := m.selected(); selected != nil {
		currentPage = selected.pageNumber + 1
	} else if m.loadingPage >= 0 {
		currentPage = m.loadingPage + 1
	} else if len(m.loadedPages) > 0 {
		currentPage = len(m.loadedPages)
	}
	return fmt.Sprintf("api page %d/%d", currentPage, totalPages)
}

func (m browserModel) hasLocalFilter() bool {
	return m.list.FilterState() != list.Unfiltered
}

func (m browserModel) activeChartPreset() chartPreset {
	if !m.chartConfig.Enabled || len(m.chartConfig.Presets) == 0 {
		return chartPreset{}
	}
	index := m.chartPresetIndex
	if index < 0 || index >= len(m.chartConfig.Presets) {
		index = 0
	}
	return m.chartConfig.Presets[index]
}

func (m browserModel) currentRenderedChart(width, height int) renderedChart {
	selected := m.selected()
	if selected == nil || selected.symbol == "" {
		return renderedChart{
			Title:         "OHLCV",
			Summary:       "No symbol selected.",
			Body:          "Select a market item, signal, or technical analysis entry to load chart data.",
			EffectiveMode: m.chartMode,
		}
	}
	preset := m.activeChartPreset()
	key := chartCacheKey(selected.symbol, preset)
	if m.chartLoading[key] {
		return renderedChart{
			Title:         fmt.Sprintf("%s  %s  %d candles", selected.symbol, preset.Interval, preset.Bars),
			Summary:       "Loading OHLCV history...",
			Body:          "Fetching chart data...",
			EffectiveMode: m.chartMode,
		}
	}
	state, ok := m.charts[key]
	if !ok {
		return renderedChart{
			Title:         fmt.Sprintf("%s  %s  %d candles", selected.symbol, preset.Interval, preset.Bars),
			Summary:       "Waiting for chart data...",
			Body:          "Use j/k to move through symbols and load OHLCV history.",
			EffectiveMode: m.chartMode,
		}
	}
	return renderChart(state, width, height, m.chartMode)
}

func chartCacheKey(symbol string, preset chartPreset) string {
	return fmt.Sprintf("%s|%s|%d", strings.ToUpper(strings.TrimSpace(symbol)), preset.Interval, preset.Bars)
}

func batchCmds(cmds ...tea.Cmd) tea.Cmd {
	nonNil := make([]tea.Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			nonNil = append(nonNil, cmd)
		}
	}
	switch len(nonNil) {
	case 0:
		return nil
	case 1:
		return nonNil[0]
	default:
		return tea.Batch(nonNil...)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
