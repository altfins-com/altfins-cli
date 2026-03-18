package cmd

import (
	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newSignalsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signals",
		Short: "Trading signal feed and metadata",
	}

	keysCmd := &cobra.Command{
		Use:   "keys",
		Short: "List all signal keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.SignalKeys(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(keysCmd, "GET", "/api/v2/public/signals-feed/signal-keys")

	var paging pagingFlags
	var bodyFlags jsonBodyFlags
	var symbols string
	var signals string
	var direction string
	var from string
	var to string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Query the trading signal feed",
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
			if items := csvValues(signals); len(items) > 0 {
				filter["signals"] = items
			}
			if direction != "" {
				filter["signalDirection"] = direction
			}
			if normalized, err := app.NormalizeTimeInput(from, false); err != nil {
				return err
			} else if normalized != "" {
				filter["fromDate"] = normalized
			}
			if normalized, err := app.NormalizeTimeInput(to, true); err != nil {
				return err
			} else if normalized != "" {
				filter["toDate"] = normalized
			}
			data, err := client.SignalsSearch(cmd.Context(), paging.value(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(listCmd, "POST", "/api/v2/public/signals-feed/search-requests")
	paging.bind(listCmd)
	bodyFlags.bind(listCmd)
	listCmd.Flags().StringVar(&symbols, "symbols", "", "Comma-separated symbol list")
	listCmd.Flags().StringVar(&signals, "signals", "", "Comma-separated signal keys")
	listCmd.Flags().StringVar(&direction, "direction", "", "Signal direction: BULLISH or BEARISH")
	listCmd.Flags().StringVar(&from, "from", "", "From date/datetime")
	listCmd.Flags().StringVar(&to, "to", "", "To date/datetime")

	cmd.AddCommand(keysCmd, listCmd)
	return cmd
}
