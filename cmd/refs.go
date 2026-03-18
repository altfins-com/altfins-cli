package cmd

import "github.com/spf13/cobra"

func newRefsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs",
		Short: "Reference data and enums",
	}

	symbolsCmd := &cobra.Command{
		Use:   "symbols",
		Short: "List supported altFINS symbols",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.Symbols(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(symbolsCmd, "GET", "/api/v2/public/symbols")

	intervalsCmd := &cobra.Command{
		Use:   "intervals",
		Short: "List supported time intervals",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.Intervals(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(intervalsCmd, "GET", "/api/v2/public/intervals")

	cmd.AddCommand(symbolsCmd, intervalsCmd)
	return cmd
}
