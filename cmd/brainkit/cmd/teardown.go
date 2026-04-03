package cmd

import (
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newTeardownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teardown <source>",
		Short: "Remove a deployment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.KitTeardownMsg{Source: args[0]},
				func(resp *messages.KitTeardownResp) {
					cmd.Printf("Removed %d resources from %s\n", resp.Removed, args[0])
				},
			)
		},
	}
}
