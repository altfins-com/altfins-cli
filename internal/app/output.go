package app

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/altfins-com/altfins-cli/internal/altfins"
)

type tableData struct {
	Headers []string
	Rows    [][]string
}

func WriteOutput(w io.Writer, data any, format string, fields []string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "table":
		tabular, err := toTableData(data, fields)
		if err != nil {
			return err
		}
		return writeTable(w, tabular)
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case "jsonl":
		return writeJSONL(w, data)
	case "csv":
		tabular, err := toTableData(data, fields)
		if err != nil {
			return err
		}
		return writeCSV(w, tabular)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeJSONL(w io.Writer, data any) error {
	items := toJSONItems(data)
	enc := json.NewEncoder(w)
	for _, item := range items {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	return nil
}

func writeCSV(w io.Writer, data tableData) error {
	writer := csv.NewWriter(w)
	if err := writer.Write(data.Headers); err != nil {
		return err
	}
	for _, row := range data.Rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeTable(w io.Writer, data tableData) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, strings.Join(data.Headers, "\t")); err != nil {
		return err
	}
	for _, row := range data.Rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func toJSONItems(data any) []any {
	switch value := data.(type) {
	case altfins.Page[altfins.ScreenerSearchResult]:
		return toAnySlice(value.Content)
	case altfins.Page[altfins.SignalFeedItem]:
		return toAnySlice(value.Content)
	case altfins.Page[altfins.NewsSummary]:
		return toAnySlice(value.Content)
	case altfins.Page[altfins.OHLCVData]:
		return toAnySlice(value.Content)
	case altfins.Page[altfins.AnalyticsHistoryData]:
		return toAnySlice(value.Content)
	case altfins.Page[altfins.TechnicalAnalysisSummary]:
		return toAnySlice(value.Content)
	case []altfins.AssetInfo:
		return toAnySlice(value)
	case []string:
		return toAnySlice(value)
	case []altfins.ValueType:
		return toAnySlice(value)
	case []altfins.SignalLabel:
		return toAnySlice(value)
	case []altfins.AnalyticsType:
		return toAnySlice(value)
	case []altfins.OHLCVData:
		return toAnySlice(value)
	default:
		return []any{data}
	}
}

func toTableData(data any, fields []string) (tableData, error) {
	switch value := data.(type) {
	case map[string]any:
		headers := []string{"key", "value"}
		keys := make([]string, 0, len(value))
		for key := range value {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		rows := make([][]string, 0, len(keys))
		for _, key := range keys {
			rows = append(rows, []string{key, stringify(value[key])})
		}
		return projectTable(tableData{Headers: headers, Rows: rows}, fields), nil
	case []map[string]any:
		return tabularMaps(value, fields), nil
	case []altfins.AssetInfo:
		return projectTable(tableData{
			Headers: []string{"name", "friendlyName"},
			Rows:    mapAssets(value),
		}, fields), nil
	case []string:
		rows := make([][]string, 0, len(value))
		for _, item := range value {
			rows = append(rows, []string{item})
		}
		return projectTable(tableData{Headers: []string{"value"}, Rows: rows}, fields), nil
	case []altfins.ValueType:
		rows := make([][]string, 0, len(value))
		for _, item := range value {
			rows = append(rows, []string{item.ID, item.FriendlyName})
		}
		return projectTable(tableData{Headers: []string{"id", "friendlyName"}, Rows: rows}, fields), nil
	case []altfins.SignalLabel:
		rows := make([][]string, 0, len(value))
		for _, item := range value {
			rows = append(rows, []string{item.SignalKey, item.SignalType, fmt.Sprintf("%t", item.TrendSensitive), item.NameBullish, item.NameBearish})
		}
		return projectTable(tableData{Headers: []string{"signalKey", "signalType", "trendSensitive", "nameBullish", "nameBearish"}, Rows: rows}, fields), nil
	case []altfins.AnalyticsType:
		rows := make([][]string, 0, len(value))
		for _, item := range value {
			rows = append(rows, []string{item.ID, item.FriendlyName, fmt.Sprintf("%t", item.IsNumerical)})
		}
		return projectTable(tableData{Headers: []string{"id", "friendlyName", "isNumerical"}, Rows: rows}, fields), nil
	case int64:
		return tableData{Headers: []string{"value"}, Rows: [][]string{{fmt.Sprintf("%d", value)}}}, nil
	case altfins.PermitsInfo:
		return projectTable(tableData{
			Headers: []string{"availablePermits", "monthlyAvailablePermits"},
			Rows: [][]string{{fmt.Sprintf("%d", value.AvailablePermits), fmt.Sprintf("%d", value.MonthlyAvailablePermits)}},
		}, fields), nil
	case altfins.Page[altfins.ScreenerSearchResult]:
		return tabularScreenerPage(value, fields), nil
	case altfins.Page[altfins.SignalFeedItem]:
		rows := make([][]string, 0, len(value.Content))
		for _, item := range value.Content {
			rows = append(rows, []string{
				item.Timestamp.Format(time.RFC3339),
				item.Symbol,
				item.SymbolName,
				item.Direction,
				item.SignalKey,
				item.SignalName,
				item.LastPrice,
				item.PriceChange,
				item.MarketCap,
			})
		}
		return projectTable(tableData{Headers: []string{"timestamp", "symbol", "symbolName", "direction", "signalKey", "signalName", "lastPrice", "priceChange", "marketCap"}, Rows: rows}, fields), nil
	case altfins.Page[altfins.NewsSummary]:
		rows := make([][]string, 0, len(value.Content))
		for _, item := range value.Content {
			rows = append(rows, []string{
				fmt.Sprintf("%d", item.MessageID),
				fmt.Sprintf("%d", item.SourceID),
				item.Timestamp.Format(time.RFC3339),
				item.SourceName,
				item.Title,
				item.URL,
			})
		}
		return projectTable(tableData{Headers: []string{"messageId", "sourceId", "timestamp", "sourceName", "title", "url"}, Rows: rows}, fields), nil
	case altfins.NewsSummary:
		return projectTable(tableData{
			Headers: []string{"messageId", "sourceId", "timestamp", "sourceName", "title", "url", "content"},
			Rows: [][]string{{fmt.Sprintf("%d", value.MessageID), fmt.Sprintf("%d", value.SourceID), value.Timestamp.Format(time.RFC3339), value.SourceName, value.Title, value.URL, value.Content}},
		}, fields), nil
	case []altfins.OHLCVData:
		return ohlcvRows(value, fields), nil
	case altfins.Page[altfins.OHLCVData]:
		return ohlcvRows(value.Content, fields), nil
	case altfins.Page[altfins.AnalyticsHistoryData]:
		rows := make([][]string, 0, len(value.Content))
		for _, item := range value.Content {
			rows = append(rows, []string{
				item.Symbol,
				item.Time.Format(time.RFC3339),
				item.Value,
				item.NonNumericalValue,
			})
		}
		return projectTable(tableData{Headers: []string{"symbol", "time", "value", "nonNumericalValue"}, Rows: rows}, fields), nil
	case altfins.Page[altfins.TechnicalAnalysisSummary]:
		rows := make([][]string, 0, len(value.Content))
		for _, item := range value.Content {
			rows = append(rows, []string{
				item.Symbol,
				item.FriendlyName,
				item.UpdatedDate.Format(time.RFC3339),
				item.NearTermOutlook,
				item.PatternType,
				item.PatternStage,
				item.Description,
				item.ImgChartURL,
			})
		}
		return projectTable(tableData{Headers: []string{"symbol", "friendlyName", "updatedDate", "nearTermOutlook", "patternType", "patternStage", "description", "imgChartUrl"}, Rows: rows}, fields), nil
	default:
		payload, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return tableData{}, fmt.Errorf("render table fallback: %w", err)
		}
		return tableData{Headers: []string{"json"}, Rows: [][]string{{string(payload)}}}, nil
	}
}

func mapAssets(items []altfins.AssetInfo) [][]string {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.Name, item.FriendlyName})
	}
	return rows
}

func ohlcvRows(items []altfins.OHLCVData, fields []string) tableData {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Symbol,
			item.Time.Format(time.RFC3339),
			item.Open,
			item.High,
			item.Low,
			item.Close,
			item.Volume,
		})
	}
	return projectTable(tableData{Headers: []string{"symbol", "time", "open", "high", "low", "close", "volume"}, Rows: rows}, fields)
}

func tabularScreenerPage(page altfins.Page[altfins.ScreenerSearchResult], fields []string) tableData {
	headers := []string{"symbol", "name", "lastPrice"}
	additional := make([]string, 0)
	keySet := map[string]struct{}{}
	for _, item := range page.Content {
		for key := range item.AdditionalData {
			if _, ok := keySet[key]; !ok {
				keySet[key] = struct{}{}
				additional = append(additional, key)
			}
		}
	}
	sort.Strings(additional)
	headers = append(headers, additional...)

	rows := make([][]string, 0, len(page.Content))
	for _, item := range page.Content {
		row := []string{item.Symbol, item.Name, item.LastPrice}
		for _, key := range additional {
			row = append(row, stringify(item.AdditionalData[key]))
		}
		rows = append(rows, row)
	}
	return projectTable(tableData{Headers: headers, Rows: rows}, fields)
}

func projectTable(in tableData, fields []string) tableData {
	if len(fields) == 0 {
		return in
	}
	indexByHeader := make(map[string]int, len(in.Headers))
	for idx, header := range in.Headers {
		indexByHeader[header] = idx
	}
	headers := make([]string, 0, len(fields))
	indices := make([]int, 0, len(fields))
	for _, field := range fields {
		if idx, ok := indexByHeader[field]; ok {
			headers = append(headers, field)
			indices = append(indices, idx)
		}
	}
	if len(headers) == 0 {
		return in
	}
	rows := make([][]string, 0, len(in.Rows))
	for _, row := range in.Rows {
		projected := make([]string, 0, len(indices))
		for _, idx := range indices {
			if idx < len(row) {
				projected = append(projected, row[idx])
			} else {
				projected = append(projected, "")
			}
		}
		rows = append(rows, projected)
	}
	return tableData{Headers: headers, Rows: rows}
}

func tabularMaps(items []map[string]any, fields []string) tableData {
	headers := make([]string, 0)
	seen := map[string]struct{}{}
	for _, item := range items {
		for key := range item {
			if _, ok := seen[key]; !ok {
				seen[key] = struct{}{}
				headers = append(headers, key)
			}
		}
	}
	sort.Strings(headers)
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		row := make([]string, 0, len(headers))
		for _, header := range headers {
			row = append(row, stringify(item[header]))
		}
		rows = append(rows, row)
	}
	return projectTable(tableData{Headers: headers, Rows: rows}, fields)
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case time.Time:
		return typed.Format(time.RFC3339)
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(payload)
	}
}

func toAnySlice[T any](items []T) []any {
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}
