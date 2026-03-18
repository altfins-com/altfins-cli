package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

const tuiAPIPageSize = 50

func NewMarketsScreen(deps *Dependencies) (Runner, error) {
	model := newBrowserModel("altFINS Markets", deps, loadMarkets, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 30},
		},
		DefaultIndex: 2,
	})
	return runner{model: model}, nil
}

func NewSignalsScreen(deps *Dependencies) (Runner, error) {
	model := newBrowserModel("altFINS Signals", deps, loadSignals, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 30},
		},
		DefaultIndex: 0,
	})
	return runner{model: model}, nil
}

func NewTechnicalAnalysisScreen(deps *Dependencies) (Runner, error) {
	model := newBrowserModel("altFINS Technical Analysis", deps, loadTechnicalAnalysis, chartConfig{
		Enabled: true,
		Presets: []chartPreset{
			{Interval: "HOURLY", Bars: 48},
			{Interval: "HOURS4", Bars: 42},
			{Interval: "DAILY", Bars: 60},
		},
		DefaultIndex: 2,
	})
	return runner{model: model}, nil
}

func NewNewsScreen(deps *Dependencies) (Runner, error) {
	model := newBrowserModel("altFINS News", deps, loadNews, chartConfig{})
	return runner{model: model}, nil
}

func loadMarkets(ctx context.Context, client *altfins.Client, filter map[string]any, paging altfins.Paging) (browserPage, error) {
	filter = copyFilter(filter)
	if symbol, ok := filter["symbol"].(string); ok && symbol != "" {
		filter["symbols"] = []string{symbol}
		delete(filter, "symbol")
	}
	paging.Size = resolveTUIPageSize(paging.Size)
	page, err := client.MarketsSearch(ctx, paging, filter)
	if err != nil {
		return browserPage{}, err
	}
	items := make([]browserItem, 0, len(page.Content))
	for _, item := range page.Content {
		details := map[string]string{
			"symbol":    item.Symbol,
			"name":      item.Name,
			"lastPrice": item.LastPrice,
		}
		keys := make([]string, 0, len(item.AdditionalData))
		for key := range item.AdditionalData {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			details[key] = stringify(item.AdditionalData[key])
		}
		items = append(items, browserItem{
			title:       fmt.Sprintf("%s (%s)", item.Name, item.Symbol),
			description: "Last price " + item.LastPrice,
			filterValue: item.Name + " " + item.Symbol,
			symbol:      item.Symbol,
			details:     details,
		})
	}
	return browserPageFrom(page, items), nil
}

func loadSignals(ctx context.Context, client *altfins.Client, filter map[string]any, paging altfins.Paging) (browserPage, error) {
	filter = copyFilter(filter)
	if symbol, ok := filter["symbol"].(string); ok && symbol != "" {
		filter["symbols"] = []string{symbol}
		delete(filter, "symbol")
	}
	paging.Size = resolveTUIPageSize(paging.Size)
	page, err := client.SignalsSearch(ctx, paging, filter)
	if err != nil {
		return browserPage{}, err
	}
	items := make([]browserItem, 0, len(page.Content))
	for _, item := range page.Content {
		items = append(items, browserItem{
			title:       fmt.Sprintf("%s %s", item.Symbol, item.SignalName),
			description: fmt.Sprintf("%s | %s | %s", item.Direction, item.LastPrice, item.Timestamp.Format("2006-01-02 15:04")),
			filterValue: strings.Join([]string{item.Symbol, item.SymbolName, item.SignalKey, item.SignalName}, " "),
			symbol:      item.Symbol,
			details: map[string]string{
				"timestamp":   item.Timestamp.Format(timeLayout),
				"direction":   item.Direction,
				"signalKey":   item.SignalKey,
				"signalName":  item.SignalName,
				"symbol":      item.Symbol,
				"symbolName":  item.SymbolName,
				"lastPrice":   item.LastPrice,
				"priceChange": item.PriceChange,
				"marketCap":   item.MarketCap,
			},
		})
	}
	return browserPageFrom(page, items), nil
}

func loadTechnicalAnalysis(ctx context.Context, client *altfins.Client, filter map[string]any, paging altfins.Paging) (browserPage, error) {
	paging.Size = resolveTUIPageSize(paging.Size)
	symbol, _ := filter["symbol"].(string)
	page, err := client.TechnicalAnalysis(ctx, paging, symbol)
	if err != nil {
		return browserPage{}, err
	}
	items := make([]browserItem, 0, len(page.Content))
	for _, item := range page.Content {
		items = append(items, browserItem{
			title:       fmt.Sprintf("%s (%s)", item.FriendlyName, item.Symbol),
			description: fmt.Sprintf("%s | %s", item.NearTermOutlook, item.PatternType),
			filterValue: strings.Join([]string{item.Symbol, item.FriendlyName, item.PatternType, item.Description}, " "),
			symbol:      item.Symbol,
			details: map[string]string{
				"symbol":          item.Symbol,
				"friendlyName":    item.FriendlyName,
				"updatedDate":     item.UpdatedDate.Format(timeLayout),
				"nearTermOutlook": item.NearTermOutlook,
				"patternType":     item.PatternType,
				"patternStage":    item.PatternStage,
				"description":     item.Description,
				"imgChartUrl":     item.ImgChartURL,
				"imgChartUrlDark": item.ImgChartURLDark,
				"logoUrl":         item.LogoURL,
			},
		})
	}
	return browserPageFrom(page, items), nil
}

func loadNews(ctx context.Context, client *altfins.Client, filter map[string]any, paging altfins.Paging) (browserPage, error) {
	paging.Size = resolveTUIPageSize(paging.Size)
	page, err := client.NewsSearch(ctx, paging, filter)
	if err != nil {
		return browserPage{}, err
	}
	items := make([]browserItem, 0, len(page.Content))
	for _, item := range page.Content {
		items = append(items, browserItem{
			title:       item.Title,
			description: fmt.Sprintf("%s | %s", item.SourceName, item.Timestamp.Format("2006-01-02 15:04")),
			filterValue: strings.Join([]string{item.Title, item.SourceName, item.Content}, " "),
			details: map[string]string{
				"messageId":  fmt.Sprintf("%d", item.MessageID),
				"sourceId":   fmt.Sprintf("%d", item.SourceID),
				"sourceName": item.SourceName,
				"timestamp":  item.Timestamp.Format(timeLayout),
				"url":        item.URL,
				"content":    item.Content,
			},
		})
	}
	return browserPageFrom(page, items), nil
}

const timeLayout = "2006-01-02 15:04:05 MST"

func browserPageFrom[T any](page altfins.Page[T], items []browserItem) browserPage {
	return browserPage{
		Items:            items,
		Page:             page.Number,
		TotalPages:       page.TotalPages,
		TotalElements:    page.TotalElements,
		NumberOfElements: page.NumberOfElements,
		First:            page.First,
		Last:             page.Last,
	}
}

func resolveTUIPageSize(size int) int {
	if size > 0 {
		return size
	}
	return tuiAPIPageSize
}

func copyFilter(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}
