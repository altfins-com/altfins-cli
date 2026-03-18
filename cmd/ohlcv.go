package cmd

import (
	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newOHLCVCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ohlcv",
		Short: "Snapshot and historical OHLCV data",
	}

	var snapshotFlags jsonBodyFlags
	var snapshotSymbols string
	var snapshotInterval string

	snapshotCmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Get latest OHLCV snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			filter, err := loadBodyFlags(cmd, snapshotFlags)
			if err != nil {
				return err
			}
			if items := csvValues(snapshotSymbols); len(items) > 0 {
				filter["symbols"] = items
			}
			if snapshotInterval != "" {
				filter["timeInterval"] = snapshotInterval
			}
			data, err := client.OHLCVSnapshot(cmd.Context(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(snapshotCmd, "POST", "/api/v2/public/ohlcv/snapshot-requests")
	snapshotFlags.bind(snapshotCmd)
	snapshotCmd.Flags().StringVar(&snapshotSymbols, "symbols", "", "Comma-separated symbols")
	snapshotCmd.Flags().StringVar(&snapshotInterval, "interval", "", "Time interval, e.g. DAILY")

	var historyPaging pagingFlags
	var historyFlags jsonBodyFlags
	var historySymbol string
	var historyInterval string
	var historyFrom string
	var historyTo string

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Get historical OHLCV candles",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			filter, err := loadBodyFlags(cmd, historyFlags)
			if err != nil {
				return err
			}
			if historySymbol != "" {
				filter["symbol"] = historySymbol
			}
			if historyInterval != "" {
				filter["timeInterval"] = historyInterval
			}
			if normalized, err := app.NormalizeTimeInput(historyFrom, false); err != nil {
				return err
			} else if normalized != "" {
				filter["from"] = normalized
			}
			if normalized, err := app.NormalizeTimeInput(historyTo, true); err != nil {
				return err
			} else if normalized != "" {
				filter["to"] = normalized
			}
			data, err := client.OHLCVHistory(cmd.Context(), historyPaging.value(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(historyCmd, "POST", "/api/v2/public/ohlcv/history-requests")
	historyPaging.bind(historyCmd)
	historyFlags.bind(historyCmd)
	historyCmd.Flags().StringVar(&historySymbol, "symbol", "", "Symbol, e.g. BTC")
	historyCmd.Flags().StringVar(&historyInterval, "interval", "", "Time interval, e.g. DAILY")
	historyCmd.Flags().StringVar(&historyFrom, "from", "", "From date/datetime")
	historyCmd.Flags().StringVar(&historyTo, "to", "", "To date/datetime")

	cmd.AddCommand(snapshotCmd, historyCmd)
	return cmd
}
