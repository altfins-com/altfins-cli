package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/altfins-com/altfins-cli/internal/app"
)

func newNewsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "news",
		Short: "News summary queries",
	}

	var paging pagingFlags
	var bodyFlags jsonBodyFlags
	var from string
	var to string

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List summarized news items",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			filter, err := loadBodyFlags(cmd, bodyFlags)
			if err != nil {
				return err
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
			data, err := client.NewsSearch(cmd.Context(), paging.value(), filter)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(listCmd, "POST", "/api/v2/public/news-summary/search-requests")
	paging.bind(listCmd)
	bodyFlags.bind(listCmd)
	listCmd.Flags().StringVar(&from, "from", "", "From date/datetime")
	listCmd.Flags().StringVar(&to, "to", "", "To date/datetime")

	var messageID int64
	var sourceID int64
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Fetch a single news summary by message and source ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := clientFor(cmd)
			if err != nil {
				return err
			}
			if messageID == 0 || sourceID == 0 {
				return fmt.Errorf("both --message-id and --source-id are required")
			}
			data, err := client.NewsGet(cmd.Context(), messageID, sourceID)
			return handleResult(cmd, data, err)
		},
	}
	annotateEndpoint(getCmd, "POST", "/api/v2/public/news-summary/find-summary")
	getCmd.Flags().Int64Var(&messageID, "message-id", 0, "Message id")
	getCmd.Flags().Int64Var(&sourceID, "source-id", 0, "Source id")

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}
