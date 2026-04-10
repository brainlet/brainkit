package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func addPluginListCmd(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use: "list", Short: "List running plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.PluginListRunningMsg{},
				func(resp *sdk.PluginListRunningResp) {
					tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tSTATUS\tPID\tUPTIME")
					for _, p := range resp.Plugins {
						fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", p.Name, p.Status, p.PID, p.Uptime)
					}
					tw.Flush()
				},
			)
		},
	})
}
