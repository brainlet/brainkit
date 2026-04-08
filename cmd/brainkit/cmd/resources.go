package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newResourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "resources",
		Short: "List all resources (tools, agents, workflows)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadConfig()
			if err != nil {
				return err
			}
			client, err := config.Connect(cfg)
			if err != nil {
				return err
			}
			defer client.Close()

			tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "TYPE\tNAME\tDESCRIPTION")

			if tools, err := httpBusRequest[sdk.ToolListMsg, sdk.ToolListResp](client, sdk.ToolListMsg{}); err == nil {
				for _, t := range tools.Tools {
					fmt.Fprintf(tw, "tool\t%s\t%s\n", t.ShortName, t.Description)
				}
			}
			if agents, err := httpBusRequest[sdk.AgentListMsg, sdk.AgentListResp](client, sdk.AgentListMsg{}); err == nil {
				for _, a := range agents.Agents {
					fmt.Fprintf(tw, "agent\t%s\t%s\n", a.Name, a.Status)
				}
			}
			if wfs, err := httpBusRequest[sdk.WorkflowListMsg, sdk.WorkflowListResp](client, sdk.WorkflowListMsg{}); err == nil {
				for _, wf := range wfs.Workflows {
					fmt.Fprintf(tw, "workflow\t%s\t%s\n", wf.Name, wf.Source)
				}
			}

			tw.Flush()
			return nil
		},
	}
}
