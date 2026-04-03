package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List active deployments",
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.KitListMsg{},
			func(resp *messages.KitListResp) {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "SOURCE\tCREATED\tRESOURCES")
				for _, d := range resp.Deployments {
					fmt.Fprintf(w, "%s\t%s\t%d resources\n", d.Source, d.CreatedAt, len(d.Resources))
				}
				w.Flush()
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
