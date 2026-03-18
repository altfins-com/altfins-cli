package cmd

import "github.com/spf13/cobra"

func newQuotaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "Inspect current permit quotas",
	}

	currentCmd := &cobra.Command{
		Use:   "current",
		Short: "Show currently available permits",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.AvailablePermits(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(currentCmd, "GET", "/api/v2/public/available-permits")

	monthlyCmd := &cobra.Command{
		Use:   "monthly",
		Short: "Show monthly remaining permits",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.MonthlyAvailablePermits(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(monthlyCmd, "GET", "/api/v2/public/monthly-available-permits")

	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Show both current and monthly permit values",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			data, err := client.AllAvailablePermits(cmd.Context())
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(allCmd, "GET", "/api/v2/public/all-available-permits")

	cmd.AddCommand(currentCmd, monthlyCmd, allCmd)
	return cmd
}
