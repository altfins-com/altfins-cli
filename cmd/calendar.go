package cmd

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// calendarCategories are the fixed category enum values accepted by the MCP
// getCryptoCalendarEvents tool. They are local constants from the MCP schema, not
// a remote endpoint, so `af calendar categories` serves them without a network call.
var calendarCategories = []string{
	"RELEASE", "BRANDING", "TOKENOMICS", "EXCHANGE", "CONFERENCE", "COMMUNITY",
	"OTHER", "AIRDROP", "AMA", "PARTNERSHIP", "ROADMAP_UPDATE", "FORK_SWAP",
	"WHITEPAPER_UPDATE", "DEV_UPDATE",
}

var calendarMyData = []string{
	"PORTFOLIO_SYMBOLS", "OPEN_ORDERS_IDENTIFIERS", "OPEN_ORDERS_SYMBOLS",
	"WATCHLIST_IDENTIFIERS", "WATCHLIST_SYMBOLS_LEVEL_1",
}

var calendarSortFields = []string{"dateEvent", "createdDate"}
var calendarSortDirections = []string{"ASC", "DESC"}

// calendarDefaultColumns is the curated default table/csv column set per FINAL.md
// (dateEvent, title, category, symbols, hot, trending, significant). The response
// names the category field coinMarketCalCategories, so the command aliases it to
// `category` before projection. The output layer keeps only present columns, so
// the verbose nested securityIdentifier object is excluded.
var calendarDefaultColumns = []string{
	"dateEvent", "title", "category", "assetSymbols", "symbols",
	"hot", "trending", "significant",
}

// calendarAllowedKeys is the full set of arguments getCryptoCalendarEvents
// accepts. Raw --filter/--stdin-json JSON is validated against it because the MCP
// input schema is additionalProperties:false.
var calendarAllowedKeys = map[string]struct{}{
	"quickSearch": {}, "releaseFrom": {}, "releaseTo": {}, "eventFrom": {}, "eventTo": {},
	"titleKeyword": {}, "assetSymbols": {}, "voteIsHot": {}, "voteIsTrending": {},
	"voteIsSignificant": {}, "category": {}, "descriptionKeyword": {}, "myData": {},
	"page": {}, "size": {}, "sortField": {}, "sortDirection": {},
}

func newCalendarCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "Crypto calendar event queries",
	}

	var bodyFlags jsonBodyFlags
	var (
		quickSearch, releaseFrom, releaseTo, eventFrom, eventTo string
		titleKeyword, descriptionKeyword, symbols, category    string
		myData, sortField, sortDirection                       string
		hot, trending, significant                             bool
		page, size                                             int
	)

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List crypto calendar events",
		Long: "List crypto calendar events (AMAs, listings, airdrops, partnerships, conferences).\n\n" +
			"Dates accept ISO-8601 (2026-06-20T00:00:00.000) or natural language ('today', 'last 7 days',\n" +
			"'this week', 'this month'); they are passed to the server verbatim. Use --event-from/--event-to\n" +
			"for when an event happens and --release-from/--release-to for when it was announced.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := mcpClientFor(cmd)
			if err != nil {
				return err
			}
			callArgs, err := loadBodyFlags(cmd, bodyFlags)
			if err != nil {
				return err
			}
			for key := range callArgs {
				if _, ok := calendarAllowedKeys[key]; !ok {
					return fmt.Errorf("unknown calendar filter key %q (allowed: %s)", key, allowedKeyList())
				}
			}

			// First-class flags override raw --filter JSON. Strings are sent only
			// when non-empty; booleans only when true (false == "no filter").
			setStr := func(key, value string) {
				if value != "" {
					callArgs[key] = value
				}
			}
			setStr("quickSearch", quickSearch)
			setStr("releaseFrom", releaseFrom)
			setStr("releaseTo", releaseTo)
			setStr("eventFrom", eventFrom)
			setStr("eventTo", eventTo)
			setStr("titleKeyword", titleKeyword)
			setStr("descriptionKeyword", descriptionKeyword)
			setStr("assetSymbols", symbols)
			setStr("category", category)
			setStr("myData", myData)
			setStr("sortField", sortField)
			setStr("sortDirection", sortDirection)
			if hot {
				callArgs["voteIsHot"] = true
			}
			if trending {
				callArgs["voteIsTrending"] = true
			}
			if significant {
				callArgs["voteIsSignificant"] = true
			}
			if cmd.Flags().Changed("page") {
				callArgs["page"] = page
			}
			if cmd.Flags().Changed("size") {
				callArgs["size"] = size
			}

			// Normalize + validate enum/CSV values uniformly, whether they came
			// from a first-class flag or from raw --filter/--stdin-json JSON.
			if err := normalizeAndValidateCalendarArgs(callArgs); err != nil {
				return err
			}

			var raw json.RawMessage
			callErr := client.CallTool(cmd.Context(), "getCryptoCalendarEvents", callArgs, &raw)
			if callErr != nil {
				return handleResult(cmd, nil, callErr)
			}
			value := mcpListValue(raw)
			if factory, ferr := factoryFor(cmd); ferr == nil {
				// Only reshape the default table/csv view; json/jsonl and explicit
				// --fields keep the full response untouched.
				if isTableMode(factory.Options.Output) && len(factory.Options.Fields) == 0 {
					value = aliasCalendarCategory(value)
					value = applyDefaultColumns(value, factory.Options.Output, nil, calendarDefaultColumns)
				}
			}
			return handleResult(cmd, value, nil)
		},
	}
	markMCPQuery(listCmd, "getCryptoCalendarEvents")
	bodyFlags.bind(listCmd)
	listCmd.Flags().StringVar(&quickSearch, "quick-search", "", "Full-text search across all event fields")
	listCmd.Flags().StringVar(&releaseFrom, "release-from", "", "Filter by release (announcement) date from (ISO or natural language)")
	listCmd.Flags().StringVar(&releaseTo, "release-to", "", "Filter by release (announcement) date to")
	listCmd.Flags().StringVar(&eventFrom, "event-from", "", "Filter by event date from (when it happens)")
	listCmd.Flags().StringVar(&eventTo, "event-to", "", "Filter by event date to")
	listCmd.Flags().StringVar(&titleKeyword, "title-keyword", "", "Match a keyword in the event title")
	listCmd.Flags().StringVar(&descriptionKeyword, "description-keyword", "", "Match a keyword in the event description")
	listCmd.Flags().StringVar(&symbols, "symbols", "", "Comma-separated asset symbols (e.g. BTC,ETH)")
	listCmd.Flags().StringVar(&category, "category", "", "Comma-separated categories (see `af calendar categories`)")
	listCmd.Flags().StringVar(&myData, "my-data", "", "Account-scoped filter; comma-separated (e.g. PORTFOLIO_SYMBOLS). Privacy-sensitive, opt-in.")
	listCmd.Flags().BoolVar(&hot, "hot", false, "Only events voted hot")
	listCmd.Flags().BoolVar(&trending, "trending", false, "Only events voted trending")
	listCmd.Flags().BoolVar(&significant, "significant", false, "Only events voted significant")
	listCmd.Flags().IntVar(&page, "page", 0, "Zero-based page index")
	listCmd.Flags().IntVar(&size, "size", 0, "Page size (server default when 0)")
	listCmd.Flags().StringVar(&sortField, "sort-field", "", "Sort field: dateEvent or createdDate")
	listCmd.Flags().StringVar(&sortDirection, "sort-direction", "", "Sort direction: ASC or DESC")

	categoriesCmd := &cobra.Command{
		Use:   "categories",
		Short: "List valid calendar event categories (local)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleResult(cmd, append([]string(nil), calendarCategories...), nil)
		},
	}
	markLocalRead(categoriesCmd)

	cmd.AddCommand(listCmd, categoriesCmd)
	return cmd
}

func validateEnum(flag string, values, allowed []string) error {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		allowedSet[a] = struct{}{}
	}
	for _, v := range values {
		if _, ok := allowedSet[v]; !ok {
			return fmt.Errorf("invalid --%s value %q (allowed: %s)", flag, v, strings.Join(allowed, ", "))
		}
	}
	return nil
}

// normalizeAndValidateCalendarArgs validates enum values and string typing for the
// calendar args regardless of whether they came from a first-class flag or raw
// --filter JSON. It upper-cases category/myData/sortDirection (sortField stays
// exact-case) so the raw escape hatch gets the same validation the flags do.
func normalizeAndValidateCalendarArgs(args map[string]any) error {
	stringParams := []string{
		"quickSearch", "releaseFrom", "releaseTo", "eventFrom", "eventTo",
		"titleKeyword", "descriptionKeyword", "assetSymbols", "category", "myData",
		"sortField", "sortDirection",
	}
	for _, key := range stringParams {
		if v, ok := args[key]; ok {
			if _, isStr := v.(string); !isStr {
				return fmt.Errorf("calendar arg %q must be a string (got %T); the MCP schema is string/CSV, not an array", key, v)
			}
		}
	}
	if v, _ := args["category"].(string); v != "" {
		vals := csvValues(strings.ToUpper(v))
		if err := validateEnum("category", vals, calendarCategories); err != nil {
			return err
		}
		args["category"] = strings.Join(vals, ",")
	}
	if v, _ := args["myData"].(string); v != "" {
		vals := csvValues(strings.ToUpper(v))
		if err := validateEnum("my-data", vals, calendarMyData); err != nil {
			return err
		}
		args["myData"] = strings.Join(vals, ",")
	}
	if v, _ := args["sortField"].(string); v != "" {
		if err := validateEnum("sort-field", []string{v}, calendarSortFields); err != nil {
			return err
		}
	}
	if v, _ := args["sortDirection"].(string); v != "" {
		norm := strings.ToUpper(v)
		if err := validateEnum("sort-direction", []string{norm}, calendarSortDirections); err != nil {
			return err
		}
		args["sortDirection"] = norm
	}
	return nil
}

// aliasCalendarCategory copies the response's coinMarketCalCategories field into a
// `category` column (the FINAL.md column name) when no category key is present.
func aliasCalendarCategory(value any) any {
	items, ok := value.([]map[string]any)
	if !ok {
		return value
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		clone := make(map[string]any, len(item)+1)
		for k, v := range item {
			clone[k] = v
		}
		if _, has := clone["category"]; !has {
			if cat, ok := clone["coinMarketCalCategories"]; ok {
				clone["category"] = cat
			}
		}
		out = append(out, clone)
	}
	return out
}

// applyDefaultColumns projects each result row to the curated column set for the
// default table/csv view (no explicit --fields). Rows keep only the columns that
// exist; a row matching none is left untouched so a response-shape change degrades
// gracefully instead of rendering blank.
func applyDefaultColumns(value any, output string, fields, columns []string) any {
	if !isTableMode(output) || len(fields) > 0 {
		return value
	}
	items, ok := value.([]map[string]any)
	if !ok {
		return value
	}
	allow := make(map[string]struct{}, len(columns))
	for _, c := range columns {
		allow[c] = struct{}{}
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		projected := make(map[string]any, len(columns))
		for key, val := range item {
			if _, ok := allow[key]; ok {
				projected[key] = val
			}
		}
		if len(projected) == 0 {
			out = append(out, item)
			continue
		}
		out = append(out, projected)
	}
	return out
}

func allowedKeyList() string {
	keys := make([]string, 0, len(calendarAllowedKeys))
	for key := range calendarAllowedKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
