package cmd

import "github.com/spf13/cobra"

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

	cmd.AddCommand(fieldsCmd, searchCmd)
	return cmd
}
