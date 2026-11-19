package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/altfins-com/altfins-cli/internal/altfins"
	"github.com/altfins-com/altfins-cli/internal/app"
	"github.com/spf13/cobra"
)

func newPresetCommand() *cobra.Command {
	var preset, interval, display string
	var paging pagingFlags
	cmd := &cobra.Command{Use: "preset", Short: "Run a provider-defined screener preset", RunE: func(cmd *cobra.Command, _ []string) error {
		body := map[string]any{"presetFilterType": preset}
		if interval != "" {
			body["timeInterval"] = interval
		}
		if display != "" {
			body["displayType"] = csvValues(display)
		}
		c, err := clientFor(cmd)
		if err != nil {
			return err
		}
		data, err := c.MarketsPreset(cmd.Context(), paging.value(), body)
		return handleResult(cmd, data, err)
	}}
	cmd.Flags().StringVar(&preset, "preset", "", "Preset filter type")
	cmd.MarkFlagRequired("preset")
	cmd.Flags().StringVar(&interval, "interval", "", "Candle interval")
	cmd.Flags().StringVar(&display, "display-type", "", "Comma-separated screener fields")
	paging.bind(cmd)
	markRemoteQuery(cmd, "POST", altfins.PresetPath)
	return cmd
}

func newPatternsCommand() *cobra.Command {
	var bodyFlags jsonBodyFlags
	var paging pagingFlags
	var symbol, interval string
	cmd := &cobra.Command{Use: "list", Short: "Find crypto chart patterns (plan-dependent coverage)", RunE: func(cmd *cobra.Command, _ []string) error {
		body, err := loadBodyFlags(cmd, bodyFlags)
		if err != nil {
			return err
		}
		if symbol != "" {
			body["symbol"] = symbol
		}
		if interval != "" {
			body["timeInterval"] = interval
		}
		c, err := clientFor(cmd)
		if err != nil {
			return err
		}
		data, err := c.Patterns(cmd.Context(), paging.value(), body)
		return handleResult(cmd, data, err)
	}}
	cmd.Flags().StringVar(&symbol, "symbol", "", "Crypto identifier, e.g. BTC")
	cmd.Flags().StringVar(&interval, "interval", "", "Candle interval")
	paging.bind(cmd)
	bodyFlags.bind(cmd)
	markRemoteQuery(cmd, "POST", altfins.PatternsPath)
	group := &cobra.Command{Use: "patterns", Short: "Chart-pattern queries"}
	group.AddCommand(cmd)
	return group
}

func newSetupsCommand() *cobra.Command {
	group := &cobra.Command{Use: "setups", Short: "Stored and generated AI trade setups for supported coins"}
	var paging pagingFlags
	var symbols, from, to, interval, direction, confidence, creator, source string
	list := &cobra.Command{Use: "list", Short: "Read stored setups (defaults to the newest matching UTC day)", RunE: func(cmd *cobra.Command, _ []string) error {
		q := url.Values{}
		for k, v := range map[string]string{"symbols": symbols, "createdBy": creator, "generatedVia": source, "timeInterval": interval, "tradingDirection": direction, "confidence": confidence} {
			if v != "" {
				q.Set(k, v)
			}
		}
		for k, v := range map[string]string{"from": from, "to": to} {
			s, err := app.NormalizeTimeInput(v, k == "to")
			if err != nil {
				return err
			}
			if s != "" {
				q.Set(k, s)
			}
		}
		if source != "" && creator != "ME" {
			return fmt.Errorf("--generated-via requires --created-by ME")
		}
		c, err := clientFor(cmd)
		if err != nil {
			return err
		}
		data, err := c.Setups(cmd.Context(), paging.value(), q)
		return handleResult(cmd, data, err)
	}}
	list.Flags().StringVar(&symbols, "symbols", "", "Comma-separated symbols")
	list.Flags().StringVar(&from, "from", "", "Earliest setup creation date/time")
	list.Flags().StringVar(&to, "to", "", "Latest setup creation date/time")
	list.Flags().StringVar(&interval, "interval", "", "Candle interval")
	list.Flags().StringVar(&direction, "direction", "", "LONG, SHORT or NONE")
	list.Flags().StringVar(&confidence, "confidence", "", "LOW, MEDIUM or HIGH")
	list.Flags().StringVar(&creator, "created-by", "ALTFINS", "ALTFINS or ME")
	list.Flags().StringVar(&source, "generated-via", "", "GUI or REST for your own setups")
	paging.bind(list)
	markRemoteQuery(list, "GET", altfins.SetupsPath+"search-requests")
	var id int64
	report := &cobra.Command{Use: "report", Short: "Read a Markdown report or pending status (one API credit)", RunE: func(cmd *cobra.Command, _ []string) error {
		if id < 1 {
			return fmt.Errorf("--id must be a positive setup ID")
		}
		c, err := clientFor(cmd)
		if err != nil {
			return err
		}
		data, err := c.SetupReport(cmd.Context(), id)
		return handleResult(cmd, data, err)
	}}
	report.Flags().Int64Var(&id, "id", 0, "Setup ID from list or generate")
	report.MarkFlagRequired("id")
	markRemoteQuery(report, "GET", altfins.SetupsPath+"{id}/report")
	var symbol, genInterval string
	var yes bool
	gen := &cobra.Command{Use: "generate", Short: "Generate a setup (one monthly AI prompt, including reused results)", RunE: func(cmd *cobra.Command, _ []string) error {
		if strings.TrimSpace(symbol) == "" {
			return fmt.Errorf("--symbol is required")
		}
		f, err := factoryFor(cmd)
		if err != nil {
			return err
		}
		if !yes && !f.Options.DryRun {
			return &app.ConfirmationRequiredError{Command: "af setups generate", Flags: []string{"--yes"}, Message: "generation consumes one monthly AI prompt, including reused results. Confirm with --yes"}
		}
		c, err := f.NewClient()
		if err != nil {
			return err
		}
		data, err := c.GenerateSetup(cmd.Context(), symbol, genInterval)
		if err == nil && data["status"] == "PENDING" {
			fmt.Fprintf(cmd.ErrOrStderr(), "Pending. Check with af setups report --id %v (one API credit).\n", data["id"])
		}
		return handleResult(cmd, data, err)
	}}
	gen.Flags().StringVar(&symbol, "symbol", "", "Supported coin symbol")
	gen.MarkFlagRequired("symbol")
	gen.Flags().StringVar(&genInterval, "interval", "DAILY", "Candle interval")
	gen.Flags().BoolVar(&yes, "yes", false, "Confirm use of one monthly AI prompt")
	annotateEndpoint(gen, "GET", altfins.SetupsPath+"generate")
	annotateSafety(gen, safetyMeta{OperationType: OpRemoteWrite, MutatesRemoteState: true, DryRunSupported: true, ConfirmationRequired: true, ConfirmationFlags: []string{"--yes"}, ConfirmationExitCode: app.ConfirmationRequiredExitCode})
	group.AddCommand(list, report, gen)
	return group
}
