package altfins

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Catalogue IDs are authoritative. Only the stock and FX namespaces identify a class.
func FilterSymbols(catalogue []AssetInfo, class, search string) ([]AssetInfo, error) {
	prefix := ""
	switch strings.ToLower(class) {
	case "":
	case "stock":
		prefix = "CS_"
	case "forex":
		prefix = "FX_"
	default:
		return nil, fmt.Errorf("unknown asset class %q (use stock or forex)", class)
	}
	needle := strings.ToUpper(strings.TrimSpace(search))
	pair := strings.ReplaceAll(needle, "/", "")
	matches := []AssetInfo{}
	for _, asset := range catalogue {
		if !strings.HasPrefix(asset.Name, prefix) {
			continue
		}
		if strings.Contains(strings.ToUpper(asset.FriendlyName), needle) || strings.Contains(strings.ToUpper(asset.Name), needle) || (strings.HasPrefix(asset.Name, "FX_") && strings.TrimPrefix(asset.Name, "FX_") == pair) {
			matches = append(matches, asset)
		}
	}
	return matches, nil
}

// Accept legacy single-item arrays, but never widen an unsupported multi-value filter.
func scalarFilter(v any) (string, error) {
	switch v := v.(type) {
	case string:
		if s := strings.TrimSpace(v); s != "" && !strings.Contains(s, ",") {
			return s, nil
		}
	case []any:
		if len(v) == 1 {
			if s, ok := v[0].(string); ok {
				return scalarFilter(s)
			}
		}
	}
	return "", fmt.Errorf("expected exactly one value")
}

func normalizeMarketFilter(filter map[string]any) error {
	for _, key := range []string{"tradingTypeFilter", "exchangeFilter"} {
		if v, ok := filter[key]; ok {
			s, err := scalarFilter(v)
			if err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
			filter[key] = s
		}
	}
	if v, ok := filter["supportResistanceLookBackIntervals"]; ok {
		n, err := strconv.Atoi(fmt.Sprint(v))
		if err != nil || n < 1 || n > 5 {
			return fmt.Errorf("supportResistanceLookBackIntervals must be an integer from 1 to 5")
		}
		filter["supportResistanceLookBackIntervals"] = n
	}
	return nil
}

func emptyMarketPage(paging Paging) Page[ScreenerSearchResult] {
	return Page[ScreenerSearchResult]{Size: paging.Size, Number: paging.Page, First: paging.Page == 0, Last: true, Sort: []OrderModel{}, Content: []ScreenerSearchResult{}}
}

func (c *Client) MarketsByClass(ctx context.Context, paging Paging, filter map[string]any, class string) (Page[ScreenerSearchResult], error) {
	empty := emptyMarketPage(paging)
	if _, err := FilterSymbols(nil, class, ""); err != nil {
		return empty, err
	}
	// Narrow to an explicitly supplied list, including an explicitly empty one.
	var requested map[string]bool
	if v, ok := filter["symbols"]; ok {
		requested = map[string]bool{}
		switch ids := v.(type) {
		case []string:
			for _, id := range ids {
				requested[id] = true
			}
		case []any:
			for _, v := range ids {
				id, ok := v.(string)
				if !ok {
					return empty, fmt.Errorf("symbols must be an array of identifiers")
				}
				requested[id] = true
			}
		default:
			return empty, fmt.Errorf("symbols must be an array of identifiers")
		}
		if len(requested) == 0 {
			return empty, nil
		}
	}
	catalogue, err := c.Symbols(ctx)
	if preview, ok := IsDryRun(err); ok {
		_, searchErr := c.MarketsSearch(ctx, paging, filter)
		if search, ok := IsDryRun(searchErr); ok {
			search.Preview.Discovery = &preview.Preview
			search.Preview.Selection = "Resolve " + class + " catalogue IDs, intersect any supplied symbols, then search once with all matching IDs. Skip search on an empty selection."
		}
		return empty, searchErr
	}
	if err != nil {
		return empty, err
	}
	matches, err := FilterSymbols(catalogue, class, "")
	if err != nil {
		return empty, err
	}
	ids := []string{}
	for _, asset := range matches {
		if requested == nil || requested[asset.Name] {
			ids = append(ids, asset.Name)
		}
	}
	if len(ids) == 0 {
		return empty, nil
	}
	filter["symbols"] = ids
	return c.MarketsSearch(ctx, paging, filter)
}
