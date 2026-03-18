package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

var version = "dev"

type exitCoder interface {
	ExitCode() int
}

type rootOptions struct {
	output  string
	dryRun  bool
	noColor bool
	fields  string
}

func Execute() int {
	root := NewRootCommand()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetIn(os.Stdin)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, app.FormatError(err))
		return exitCode(err)
	}
	return 0
}

func NewRootCommand() *cobra.Command {
	opts := &rootOptions{}

	root := &cobra.Command{
		Use:           "af",
		Short:         "altFINS CLI for market data, signals, analytics, and TUI workflows",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			output := strings.ToLower(strings.TrimSpace(opts.output))
			switch output {
			case "", "table", "json", "jsonl", "csv":
			default:
				return fmt.Errorf("unsupported output format %q", opts.output)
			}
			factory, err := app.NewFactory(app.RootOptions{
				Output:  output,
				DryRun:  opts.dryRun,
				NoColor: opts.noColor,
				Fields:  app.ParseCSV(opts.fields),
			}, cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			cmd.SetContext(app.WithFactory(cmd.Context(), factory))
			return nil
		},
	}

	root.PersistentFlags().StringVarP(&opts.output, "output", "o", "table", "Output format: table, json, jsonl, csv")
	root.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "Show the API request without executing it")
	root.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "Disable ANSI color output")
	root.PersistentFlags().StringVar(&opts.fields, "fields", "", "Comma-separated output field selection")

	root.AddCommand(
		newAuthCommand(),
		newRefsCommand(),
		newQuotaCommand(),
		newMarketsCommand(),
		newOHLCVCommand(),
		newAnalyticsCommand(),
		newTACommand(),
		newSignalsCommand(),
		newNewsCommand(),
		newTUICommand(),
	)
	root.AddCommand(newCommandsCommand(root))

	return root
}

func exitCode(err error) int {
	var coder exitCoder
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return 1
}
