package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/tui"
)

func newTUICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Interactive terminal UI",
	}

	cmd.AddCommand(
		newTUIBrowserCommand("markets", "Markets TUI", "/api/v2/public/screener-data/search-requests", tui.NewMarketsScreen),
		newTUIBrowserCommand("signals", "Signals TUI", "/api/v2/public/signals-feed/search-requests", tui.NewSignalsScreen),
		newTUIBrowserCommand("ta", "Technical analysis TUI", "/api/v2/public/technical-analysis/data", tui.NewTechnicalAnalysisScreen),
		newTUIBrowserCommand("news", "News TUI", "/api/v2/public/news-summary/search-requests", tui.NewNewsScreen),
	)

	return cmd
}

func newTUIBrowserCommand(use, short, path string, constructor func(*tui.Dependencies) (tui.Runner, error)) *cobra.Command {
	var filterFlags jsonBodyFlags
	var symbol string
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			factory, err := factoryFor(cmd)
			if err != nil {
				return err
			}
			if factory.Options.DryRun {
				return fmt.Errorf("--dry-run is not supported for TUI commands")
			}

			client, err := factory.NewClient()
			if err != nil {
				return err
			}
			resolved, err := factory.ResolveConfig()
			if err != nil {
				return err
			}

			filter, err := loadBodyFlags(cmd, filterFlags)
			if err != nil {
				return err
			}
			if symbol != "" {
				filter["symbol"] = symbol
			}

			deps := &tui.Dependencies{
				Client:     client,
				AuthSource: resolved.AuthSource,
				FilterJSON: mustJSON(filter),
				Filter:     filter,
			}
			screen, err := constructor(deps)
			if err != nil {
				return err
			}
			return screen.Run()
		},
	}
	annotateEndpoint(cmd, "TUI", path)
	filterFlags.bind(cmd)
	cmd.Flags().StringVar(&symbol, "symbol", "", "Optional symbol seed filter")
	return cmd
}
