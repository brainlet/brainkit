package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var resourcesCmd = &cobra.Command{
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

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "TYPE\tNAME\tDESCRIPTION")

		if tools, err := busRequest[messages.ToolListMsg, messages.ToolListResp](client, messages.ToolListMsg{}); err == nil {
			for _, t := range tools.Tools {
				fmt.Fprintf(w, "tool\t%s\t%s\n", t.ShortName, t.Description)
			}
		}

		if agents, err := busRequest[messages.AgentListMsg, messages.AgentListResp](client, messages.AgentListMsg{}); err == nil {
			for _, a := range agents.Agents {
				fmt.Fprintf(w, "agent\t%s\t%s\n", a.Name, a.Status)
			}
		}

		if wfs, err := busRequest[messages.WorkflowListMsg, messages.WorkflowListResp](client, messages.WorkflowListMsg{}); err == nil {
			for _, wf := range wfs.Workflows {
				fmt.Fprintf(w, "workflow\t%s\t%s\n", wf.Name, wf.Source)
			}
		}

		w.Flush()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resourcesCmd)
}
