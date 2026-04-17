package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active deployments",
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.PackageListDeployedMsg{},
				func(resp *sdk.PackageListDeployedResp) {
					tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tVERSION\tSOURCE\tSTATUS")
					for _, p := range resp.Packages {
						fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", p.Name, p.Version, p.Source, p.Status)
					}
					tw.Flush()
				},
			)
		},
	}
}
