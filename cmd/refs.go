package cmd

import (
	"github.com/altfins-com/altfins-cli/internal/altfins"
	"github.com/spf13/cobra"
)

func newRefsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refs",
		Short: "Reference data and enums",
	}

	var class, search string
	symbolsCmd := &cobra.Command{
		Use:   "symbols",
		Short: "List supported altFINS symbols",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			if _, err := altfins.FilterSymbols(nil, class, search); err != nil {
				return err
			}
			data, err := client.Symbols(cmd.Context())
			if err == nil {
				data, err = altfins.FilterSymbols(data, class, search)
			}
			return handleResult(cmd, data, err)
		},
	}
	symbolsCmd.Flags().StringVar(&search, "search", "", "Match an identifier or instrument name")
	symbolsCmd.Flags().StringVar(&class, "asset-class", "", "Limit to stock or forex identifiers")
	markRemoteQuery(symbolsCmd, "GET", "/api/v2/public/symbols")

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
	markRemoteQuery(intervalsCmd, "GET", "/api/v2/public/intervals")

	cmd.AddCommand(symbolsCmd, intervalsCmd)
	return cmd
}
