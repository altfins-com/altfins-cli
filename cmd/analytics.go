package cmd

import (
	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newAnalyticsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analytics",
		Short: "Historical analytics and metric metadata",
	}

	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "List supported analytics types",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.AnalyticsTypes(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(typesCmd, "GET", "/api/v2/public/analytics/types")

	var paging pagingFlags
	var bodyFlags jsonBodyFlags
	var symbol string
	var analyticsType string
	var interval string
	var from string
	var to string

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Fetch historical analytics data for a symbol and metric",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			filter, err := loadBodyFlags(cmd, bodyFlags)
			if err != nil {
				return err
			}
			if symbol != "" {
				filter["symbol"] = symbol
			}
			if analyticsType != "" {
				filter["analyticsType"] = analyticsType
			}
			if interval != "" {
				filter["timeInterval"] = interval
			}
			if normalized, err := app.NormalizeTimeInput(from, false); err != nil {
				return err
			} else if normalized != "" {
				filter["from"] = normalized
			}
			if normalized, err := app.NormalizeTimeInput(to, true); err != nil {
				return err
			} else if normalized != "" {
				filter["to"] = normalized
			}
			data, err := client.AnalyticsHistory(cmd.Context(), paging.value(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(historyCmd, "POST", "/api/v2/public/analytics/search-requests")
	paging.bind(historyCmd)
	bodyFlags.bind(historyCmd)
	historyCmd.Flags().StringVar(&symbol, "symbol", "", "Symbol, e.g. BTC")
	historyCmd.Flags().StringVar(&analyticsType, "type", "", "Analytics type id, e.g. RSI14")
	historyCmd.Flags().StringVar(&interval, "interval", "", "Time interval")
	historyCmd.Flags().StringVar(&from, "from", "", "From date/datetime")
	historyCmd.Flags().StringVar(&to, "to", "", "To date/datetime")

	cmd.AddCommand(typesCmd, historyCmd)
	return cmd
}
