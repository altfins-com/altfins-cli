package tui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas"
	"github.com/NimbleMarkets/ntcharts/canvas/graph"
	"github.com/NimbleMarkets/ntcharts/linechart"
	"github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

type chartMode string

const (
	chartModeCandles chartMode = "candles"
	chartModeBraille chartMode = "braille"
)

type chartPreset struct {
	Interval string
	Bars     int
}

type chartConfig struct {
	Enabled      bool
	Presets      []chartPreset
	DefaultIndex int
}

type parsedCandle struct {
	Time  time.Time
	Open  float64
	High  float64
	Low   float64
	Close float64
}

type chartState struct {
	Symbol    string
	Interval  string
	Bars      int
	Raw       []altfins.OHLCVData
	Candles   []parsedCandle
	Err       error
	FetchedAt time.Time
}

type renderedChart struct {
	Title         string
	Summary       string
	Body          string
	EffectiveMode chartMode
	Notice        string
}

var (
	chartBullStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	chartBearStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	chartDimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func newChartState(symbol string, preset chartPreset, items []altfins.OHLCVData, err error) chartState {
	raw := append([]altfins.OHLCVData(nil), items...)
	sort.Slice(raw, func(i, j int) bool {
		return raw[i].Time.Before(raw[j].Time)
	})
	return chartState{
		Symbol:    strings.TrimSpace(symbol),
		Interval:  preset.Interval,
		Bars:      preset.Bars,
		Raw:       raw,
		Candles:   parseCandles(raw),
		Err:       err,
		FetchedAt: time.Now(),
	}
}

func parseCandles(items []altfins.OHLCVData) []parsedCandle {
	candles := make([]parsedCandle, 0, len(items))
	for _, item := range items {
		openValue, err := strconv.ParseFloat(item.Open, 64)
		if err != nil {
			continue
		}
		highValue, err := strconv.ParseFloat(item.High, 64)
		if err != nil {
			continue
		}
		lowValue, err := strconv.ParseFloat(item.Low, 64)
		if err != nil {
			continue
		}
		closeValue, err := strconv.ParseFloat(item.Close, 64)
		if err != nil {
			continue
		}
		candles = append(candles, parsedCandle{
			Time:  item.Time.UTC(),
			Open:  openValue,
			High:  highValue,
			Low:   lowValue,
			Close: closeValue,
		})
	}
	sort.Slice(candles, func(i, j int) bool {
		return candles[i].Time.Before(candles[j].Time)
	})
	return candles
}

func renderChart(state chartState, width, height int, mode chartMode) renderedChart {
	title := fmt.Sprintf("%s  %s  %d candles", fallback(state.Symbol, "OHLCV"), state.Interval, len(state.Candles))
	if state.Err != nil {
		return renderedChart{
			Title:         title,
			Summary:       "Unable to load OHLCV history.",
			Body:          state.Err.Error(),
			EffectiveMode: mode,
		}
	}
	if len(state.Candles) == 0 {
		return renderedChart{
			Title:         title,
			Summary:       "No valid OHLCV candles available.",
			Body:          "No chart data available.",
			EffectiveMode: mode,
		}
	}

	latest := state.Candles[len(state.Candles)-1]
	summary := fmt.Sprintf(
		"O %s  H %s  L %s  C %s  @ %s",
		formatPrice(latest.Open),
		formatPrice(latest.High),
		formatPrice(latest.Low),
		formatPrice(latest.Close),
		latest.Time.Format("2006-01-02 15:04 UTC"),
	)

	actualMode := mode
	notice := ""
	switch {
	case width < 28 || height < 8:
		return renderedChart{
			Title:         title,
			Summary:       summary,
			Body:          "Not enough room to render a chart.",
			EffectiveMode: mode,
		}
	case mode == chartModeCandles && len(state.Candles) < 8:
		if len(state.Candles) >= 2 {
			actualMode = chartModeBraille
			notice = "Fewer than 8 valid candles; showing braille fallback."
		} else {
			return renderedChart{
				Title:         title,
				Summary:       summary,
				Body:          "Not enough OHLCV data to draw a chart.",
				EffectiveMode: mode,
			}
		}
	case mode == chartModeCandles && (width < 44 || height < 12):
		actualMode = chartModeBraille
		notice = "Pane is too narrow for candles; showing braille fallback."
	}

	bodyWidth := max(12, width)
	bodyHeight := max(5, height)
	body := ""
	switch actualMode {
	case chartModeBraille:
		body = renderBrailleChart(state.Candles, bodyWidth, bodyHeight)
	default:
		body = renderCandleChart(state, bodyWidth, bodyHeight)
	}
	if strings.TrimSpace(body) == "" {
		body = "No chart data available."
	}

	return renderedChart{
		Title:         title,
		Summary:       summary,
		Body:          body,
		EffectiveMode: actualMode,
		Notice:        notice,
	}
}

func renderCandleChart(state chartState, width, height int) string {
	if len(state.Candles) == 0 {
		return ""
	}

	minTime := state.Candles[0].Time
	maxTime := state.Candles[len(state.Candles)-1].Time.Add(intervalDuration(state.Interval))
	minPrice := state.Candles[0].Low
	maxPrice := state.Candles[0].High
	for _, candle := range state.Candles[1:] {
		if candle.Low < minPrice {
			minPrice = candle.Low
		}
		if candle.High > maxPrice {
			maxPrice = candle.High
		}
	}
	if minPrice == maxPrice {
		padding := math.Abs(minPrice) * 0.02
		if padding == 0 {
			padding = 1
		}
		minPrice -= padding
		maxPrice += padding
	} else {
		padding := (maxPrice - minPrice) * 0.05
		minPrice -= padding
		maxPrice += padding
	}

	chart := timeserieslinechart.New(
		width,
		height,
		timeserieslinechart.WithTimeRange(minTime, maxTime),
		timeserieslinechart.WithYRange(minPrice, maxPrice),
		timeserieslinechart.WithXYSteps(3, 3),
		timeserieslinechart.WithAxesStyles(chartDimStyle, chartDimStyle),
		timeserieslinechart.WithXLabelFormatter(chartTimeLabelFormatter(state.Interval)),
		timeserieslinechart.WithYLabelFormatter(func(_ int, value float64) string {
			return formatAxisPrice(value)
		}),
	)

	for _, candle := range state.Candles {
		openPoint := timeserieslinechart.TimePoint{Time: candle.Time, Value: candle.Open}
		highPoint := timeserieslinechart.TimePoint{Time: candle.Time, Value: candle.High}
		lowPoint := timeserieslinechart.TimePoint{Time: candle.Time, Value: candle.Low}
		closePoint := timeserieslinechart.TimePoint{Time: candle.Time, Value: candle.Close}

		chart.PushDataSet("open", openPoint)
		chart.PushDataSet("high", highPoint)
		chart.PushDataSet("low", lowPoint)
		chart.PushDataSet("close", closePoint)
	}

	chart.DrawCandle("open", "high", "low", "close", chartBullStyle, chartBearStyle)
	return chart.View()
}

func renderBrailleChart(candles []parsedCandle, width, height int) string {
	if len(candles) == 0 || width < 10 || height < 5 {
		return "No data"
	}

	prices := make([]float64, 0, len(candles))
	for _, candle := range candles {
		prices = append(prices, candle.Close)
	}

	minPrice, maxPrice := prices[0], prices[0]
	for _, price := range prices[1:] {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}
	if maxPrice == minPrice {
		maxPrice = minPrice + 1
	}

	midPrice := (minPrice + maxPrice) / 2
	yHigh := formatAxisPrice(maxPrice)
	yMid := formatAxisPrice(midPrice)
	yLow := formatAxisPrice(minPrice)
	yWidth := max(max(lipgloss.Width(yHigh), lipgloss.Width(yMid)), lipgloss.Width(yLow)) + 1

	chartWidth := max(4, width-yWidth)
	chartHeight := max(3, height-2)
	bg := graph.NewBrailleGrid(chartWidth, chartHeight, 0, float64(len(prices)-1), minPrice, maxPrice)

	for i := 1; i < len(prices); i++ {
		p1 := canvas.Float64Point{X: float64(i - 1), Y: prices[i-1]}
		p2 := canvas.Float64Point{X: float64(i), Y: prices[i]}
		for _, point := range graph.GetLinePoints(bg.GridPoint(p1), bg.GridPoint(p2)) {
			bg.Set(point)
		}
	}

	lineStyle := chartBullStyle
	if prices[len(prices)-1] < prices[0] {
		lineStyle = chartBearStyle
	}

	var out strings.Builder
	rows := bg.BraillePatterns()
	for i, row := range rows {
		switch i {
		case 0:
			out.WriteString(chartDimStyle.Render(fmt.Sprintf("%*s", yWidth, yHigh)))
		case len(rows) / 2:
			out.WriteString(chartDimStyle.Render(fmt.Sprintf("%*s", yWidth, yMid)))
		case len(rows) - 1:
			out.WriteString(chartDimStyle.Render(fmt.Sprintf("%*s", yWidth, yLow)))
		default:
			out.WriteString(strings.Repeat(" ", yWidth))
		}
		out.WriteString(lineStyle.Render(string(row)))
		out.WriteString("\n")
	}

	start := candles[0].Time.Format("01/02")
	mid := candles[len(candles)/2].Time.Format("01/02")
	end := candles[len(candles)-1].Time.Format("01/02")
	gap := chartWidth - lipgloss.Width(start) - lipgloss.Width(mid) - lipgloss.Width(end)
	if gap < 2 {
		gap = 2
	}
	leftGap := gap / 2
	rightGap := gap - leftGap
	xAxis := strings.Repeat(" ", yWidth) +
		start + strings.Repeat(" ", leftGap) +
		mid + strings.Repeat(" ", rightGap) +
		end
	out.WriteString(chartDimStyle.Render(xAxis))

	return out.String()
}

func chartTimeLabelFormatter(interval string) linechart.LabelFormatter {
	return func(_ int, value float64) string {
		t := time.Unix(int64(value), 0).UTC()
		switch interval {
		case "HOURLY", "HOURS4":
			return t.Format("01/02 15h")
		default:
			return t.Format("01/02")
		}
	}
}

func intervalDuration(interval string) time.Duration {
	switch interval {
	case "HOURLY":
		return time.Hour
	case "HOURS4":
		return 4 * time.Hour
	default:
		return 24 * time.Hour
	}
}

func formatAxisPrice(value float64) string {
	return trimZeroes(func() string {
		abs := math.Abs(value)
		switch {
		case abs >= 1000:
			return fmt.Sprintf("%.0f", value)
		case abs >= 100:
			return fmt.Sprintf("%.1f", value)
		case abs >= 1:
			return fmt.Sprintf("%.2f", value)
		case abs >= 0.01:
			return fmt.Sprintf("%.4f", value)
		case abs >= 0.0001:
			return fmt.Sprintf("%.6f", value)
		default:
			return fmt.Sprintf("%.8f", value)
		}
	}())
}

func formatPrice(value float64) string {
	return formatAxisPrice(value)
}

func trimZeroes(value string) string {
	if !strings.Contains(value, ".") {
		return value
	}
	value = strings.TrimRight(value, "0")
	value = strings.TrimRight(value, ".")
	return value
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
