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
			// when non-empty; booleans only when explicitly set.
			setStr := func(flag, key, value string) {
				if cmd.Flags().Changed(flag) || value != "" {
					if value != "" {
						callArgs[key] = value
					}
				}
			}
			setStr("quick-search", "quickSearch", quickSearch)
			setStr("release-from", "releaseFrom", releaseFrom)
			setStr("release-to", "releaseTo", releaseTo)
			setStr("event-from", "eventFrom", eventFrom)
			setStr("event-to", "eventTo", eventTo)
			setStr("title-keyword", "titleKeyword", titleKeyword)
			setStr("description-keyword", "descriptionKeyword", descriptionKeyword)
			setStr("symbols", "assetSymbols", symbols)

			if category != "" {
				values := csvValues(category)
				if err := validateEnum("category", values, calendarCategories); err != nil {
					return err
				}
				callArgs["category"] = strings.Join(values, ",")
			}
			if myData != "" {
				values := csvValues(myData)
				if err := validateEnum("my-data", values, calendarMyData); err != nil {
					return err
				}
				callArgs["myData"] = strings.Join(values, ",")
			}
			if cmd.Flags().Changed("hot") {
				callArgs["voteIsHot"] = hot
			}
			if cmd.Flags().Changed("trending") {
				callArgs["voteIsTrending"] = trending
			}
			if cmd.Flags().Changed("significant") {
				callArgs["voteIsSignificant"] = significant
			}
			if cmd.Flags().Changed("page") {
				callArgs["page"] = page
			}
			if cmd.Flags().Changed("size") {
				callArgs["size"] = size
			}
			if sortField != "" {
				if err := validateEnum("sort-field", []string{sortField}, calendarSortFields); err != nil {
					return err
				}
				callArgs["sortField"] = sortField
			}
			if sortDirection != "" {
				normalized := strings.ToUpper(sortDirection)
				if err := validateEnum("sort-direction", []string{normalized}, calendarSortDirections); err != nil {
					return err
				}
				callArgs["sortDirection"] = normalized
			}

			var raw json.RawMessage
			callErr := client.CallTool(cmd.Context(), "getCryptoCalendarEvents", callArgs, &raw)
			if callErr != nil {
				return handleResult(cmd, nil, callErr)
			}
			return handleResult(cmd, mcpListValue(raw), nil)
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

func allowedKeyList() string {
	keys := make([]string, 0, len(calendarAllowedKeys))
	for key := range calendarAllowedKeys {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
