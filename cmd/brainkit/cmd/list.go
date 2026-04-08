package cmd

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.KitListMsg{},
				func(resp *sdk.KitListResp) {
					tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "SOURCE\tCREATED\tRESOURCES")
					for _, d := range resp.Deployments {
						name := strings.TrimSuffix(d.Source, ".ts")
						fmt.Fprintf(tw, "%s\t%s\t%d resources\n", name, d.CreatedAt, len(d.Resources))
					}
					tw.Flush()
				},
			)
		},
	}
}
