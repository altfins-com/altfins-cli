package cmd

import "github.com/spf13/cobra"

func newTACommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ta",
		Short: "Curated technical analysis summaries",
	}

	var paging pagingFlags
	var symbol string
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List curated technical analysis entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.TechnicalAnalysis(cmd.Context(), paging.value(), symbol)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(listCmd, "GET", "/api/v2/public/technical-analysis/data")
	paging.bind(listCmd)
	listCmd.Flags().StringVar(&symbol, "symbol", "", "Symbol filter")

	cmd.AddCommand(listCmd)
	return cmd
}
