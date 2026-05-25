package cmd

import (
	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newMarketsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "markets",
		Short: "Screener market data queries",
	}

	fieldsCmd := &cobra.Command{
		Use:   "fields",
		Short: "List screener display field types",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.MarketsFields(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(fieldsCmd, "GET", "/api/v2/public/screener-data/value-types")

	var paging pagingFlags
	var bodyFlags jsonBodyFlags
	var symbols string
	var interval string
	var displayType string
	var coinType string
	var categories string
	var tradingTypes string
	var exchanges string
	var athBefore string
	var athAfter string
	var supportResistance string
	var supportResistanceLookback string
	var week52 string
	var rsiDivergence string
	var newLow string
	var newHigh string
	var macd string
	var macdHistogram string
	var minMarketCap float64

	searchCmd := &cobra.Command{
		Use:   "search",
		Short: "Run a screener market search",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			filter, err := loadBodyFlags(cmd, bodyFlags)
			if err != nil {
				return err
			}
			if items := csvValues(symbols); len(items) > 0 {
				filter["symbols"] = items
			}
			if interval != "" {
				filter["timeInterval"] = interval
			}
			if items := csvValues(displayType); len(items) > 0 {
				filter["displayType"] = items
			}
			if coinType != "" {
				filter["coinTypeFilter"] = coinType
			}
			if items := csvValues(categories); len(items) > 0 {
				filter["coinCategoryFilter"] = items
			}
			if items := csvValues(tradingTypes); len(items) > 0 {
				filter["tradingTypeFilter"] = items
			}
			if items := csvValues(exchanges); len(items) > 0 {
				filter["exchangeFilter"] = items
			}
			if normalized, err := app.NormalizeTimeInput(athBefore, true); err != nil {
				return err
			} else if normalized != "" {
				filter["athDateBeforeFilter"] = normalized
			}
			if normalized, err := app.NormalizeTimeInput(athAfter, false); err != nil {
				return err
			} else if normalized != "" {
				filter["athDateAfterFilter"] = normalized
			}
			if supportResistance != "" {
				filter["supportResistanceFilter"] = supportResistance
			}
			if supportResistanceLookback != "" {
				filter["supportResistanceLookBackIntervals"] = supportResistanceLookback
			}
			if week52 != "" {
				filter["weekAnalytics52Filter"] = week52
			}
			if rsiDivergence != "" {
				filter["rsiDivergenceFilter"] = rsiDivergence
			}
			if newLow != "" {
				filter["newLowInLastPeriodFilter"] = newLow
			}
			if newHigh != "" {
				filter["newHighInLastPeriodFilter"] = newHigh
			}
			if macd != "" {
				filter["macdFilter"] = macd
			}
			if macdHistogram != "" {
				filter["macdHistogramFilter"] = macdHistogram
			}
			if cmd.Flags().Changed("min-market-cap") {
				filter["minimumMarketCapValue"] = minMarketCap
			}
			data, err := client.MarketsSearch(cmd.Context(), paging.value(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(searchCmd, "POST", "/api/v2/public/screener-data/search-requests")
	paging.bind(searchCmd)
	bodyFlags.bind(searchCmd)
	searchCmd.Flags().StringVar(&symbols, "symbols", "", "Comma-separated symbols")
	searchCmd.Flags().StringVar(&interval, "interval", "", "Time interval, e.g. DAILY or HOURLY")
	searchCmd.Flags().StringVar(&displayType, "display-type", "", "Comma-separated screener display fields")
	searchCmd.Flags().StringVar(&coinType, "coin-type", "", "Coin type filter, e.g. REGULAR")
	searchCmd.Flags().StringVar(&categories, "categories", "", "Comma-separated coin category filters")
	searchCmd.Flags().StringVar(&tradingTypes, "trading-types", "", "Comma-separated trading type filters")
	searchCmd.Flags().StringVar(&exchanges, "exchanges", "", "Comma-separated exchange filters")
	searchCmd.Flags().StringVar(&athBefore, "ath-before", "", "ATH date on or before this date/datetime")
	searchCmd.Flags().StringVar(&athAfter, "ath-after", "", "ATH date on or after this date/datetime")
	searchCmd.Flags().StringVar(&supportResistance, "support-resistance", "", "Support/resistance filter")
	searchCmd.Flags().StringVar(&supportResistanceLookback, "support-resistance-lookback", "", "Support/resistance lookback intervals: 1-5")
	searchCmd.Flags().StringVar(&week52, "week-52", "", "52-week analytics filter")
	searchCmd.Flags().StringVar(&rsiDivergence, "rsi-divergence", "", "RSI divergence filter")
	searchCmd.Flags().StringVar(&newLow, "new-low", "", "New low period filter, e.g. PERIODS_30")
	searchCmd.Flags().StringVar(&newHigh, "new-high", "", "New high period filter, e.g. PERIODS_10")
	searchCmd.Flags().StringVar(&macd, "macd", "", "MACD crossover filter: BUY or SELL")
	searchCmd.Flags().StringVar(&macdHistogram, "macd-histogram", "", "MACD histogram filter")
	searchCmd.Flags().Float64Var(&minMarketCap, "min-market-cap", 0, "Minimum market cap value")

	cmd.AddCommand(fieldsCmd, searchCmd)
	return cmd
}
