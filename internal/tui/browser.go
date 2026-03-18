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
}

func (i browserItem) Title() string       { return i.title }
func (i browserItem) Description() string { return i.description }
func (i browserItem) FilterValue() string { return i.filterValue }

type itemsMsg struct {
	items []browserItem
}

type quotaMsg struct {
	quota altfins.PermitsInfo
	err   error
}

type chartMsg struct {
	symbol string
	view   string
	err    error
}

type errMsg struct {
	err error
}

type loadFn func(context.Context, *altfins.Client, map[string]any) ([]browserItem, error)

type browserModel struct {
	title       string
	client      *altfins.Client
	authSource  string
	filter      map[string]any
	filterJSON  string
	load        loadFn
	list        list.Model
	loading     bool
	err         error
	quota       altfins.PermitsInfo
	width       int
	height      int
	showFilter  bool
	detailOnly  bool
	focus       string
	lastRefresh time.Time
	charts      map[string]string
}

func newBrowserModel(title string, deps *Dependencies, loader loadFn) browserModel {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true

	listModel := list.New([]list.Item{}, delegate, 0, 0)
	listModel.Title = title
	listModel.SetFilteringEnabled(true)
	listModel.SetShowHelp(false)
	listModel.SetShowStatusBar(true)
	listModel.SetShowPagination(true)

	return browserModel{
		title:      title,
		client:     deps.Client,
		authSource: deps.AuthSource,
		filter:     deps.Filter,
		filterJSON: deps.FilterJSON,
		load:       loader,
		list:       listModel,
		loading:    true,
		focus:      "list",
		charts:     map[string]string{},
	}
}

func (m browserModel) Init() tea.Cmd {
	return tea.Batch(m.loadCmd(), m.quotaCmd())
}

func (m browserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		return m, nil
	case itemsMsg:
		items := make([]list.Item, 0, len(msg.items))
		for _, item := range msg.items {
			items = append(items, item)
		}
		m.list.SetItems(items)
		m.loading = false
		m.err = nil
		m.lastRefresh = time.Now()
		return m, m.loadChartForSelection()
	case quotaMsg:
		if msg.err == nil {
			m.quota = msg.quota
		}
		return m, nil
	case chartMsg:
		if msg.err == nil && msg.symbol != "" {
			m.charts[msg.symbol] = msg.view
		}
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.loadCmd(), m.quotaCmd())
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
		case "esc", "backspace":
			if m.detailOnly {
				m.detailOnly = false
				return m, nil
			}
		}
	}

	beforeIndex := m.list.Index()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	if m.list.Index() != beforeIndex {
		return m, tea.Batch(cmd, m.loadChartForSelection())
	}
	return m, cmd
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
	detail := m.renderDetail()

	if m.detailOnly {
		blocks := []string{detail}
		if m.showFilter {
			blocks = append(blocks, m.renderFilter())
		}
		return lipgloss.JoinVertical(lipgloss.Left, blocks...)
	}

	leftWidth := max(30, m.width/2-1)
	rightWidth := max(30, m.width-leftWidth-1)

	left := lipgloss.NewStyle().
		Width(leftWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.borderColor("list")).
		Render(m.list.View())

	rightParts := []string{
		lipgloss.NewStyle().
			Width(rightWidth).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.borderColor("detail")).
			Render(detail),
	}
	if m.showFilter {
		rightParts = append(rightParts,
			lipgloss.NewStyle().
				Width(rightWidth).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("12")).
				Render(m.renderFilter()),
		)
	}
	right := lipgloss.JoinVertical(lipgloss.Left, rightParts...)

	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m browserModel) renderDetail() string {
	selected := m.selected()
	if selected == nil {
		return "No items loaded."
	}

	lines := []string{
		lipgloss.NewStyle().Bold(true).Render(selected.title),
		selected.description,
		"",
	}
	if chart, ok := m.charts[selected.symbol]; ok && chart != "" {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Render("OHLCV Sparkline"),
			chart,
			"",
		)
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

func (m browserModel) renderFilter() string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("10")).
		Render("Active Filter") + "\n" + m.filterJSON
}

func (m browserModel) selected() *browserItem {
	item, ok := m.list.SelectedItem().(browserItem)
	if !ok {
		return nil
	}
	return &item
}

func (m browserModel) loadCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := m.load(context.Background(), m.client, m.filter)
		if err != nil {
			return errMsg{err: err}
		}
		return itemsMsg{items: items}
	}
}

func (m browserModel) quotaCmd() tea.Cmd {
	return func() tea.Msg {
		quota, err := m.client.AllAvailablePermits(context.Background())
		return quotaMsg{quota: quota, err: err}
	}
}

func (m browserModel) loadChartForSelection() tea.Cmd {
	selected := m.selected()
	if selected == nil || selected.symbol == "" {
		return nil
	}
	if _, ok := m.charts[selected.symbol]; ok {
		return nil
	}
	symbol := selected.symbol
	return func() tea.Msg {
		page, err := m.client.OHLCVHistory(context.Background(), altfins.Paging{Size: 30}, map[string]any{
			"symbol":       symbol,
			"timeInterval": "DAILY",
		})
		if err != nil {
			return chartMsg{symbol: symbol, err: err}
		}
		return chartMsg{symbol: symbol, view: renderSparkline(page.Content)}
	}
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
		"Enter detail",
		"Esc back",
		"Tab focus",
		"f filter",
		"r refresh",
		fmt.Sprintf("permits %d/%d", m.quota.AvailablePermits, m.quota.MonthlyAvailablePermits),
	}
	if !m.lastRefresh.IsZero() {
		parts = append(parts, "updated "+m.lastRefresh.Format("15:04:05"))
	}
	if m.authSource != "" {
		parts = append(parts, "auth "+m.authSource)
	}
	return strings.Join(parts, "  |  ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
