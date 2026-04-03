package cmd

import (
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var teardownCmd = &cobra.Command{
	Use:   "teardown <source>",
	Short: "Remove a deployment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.KitTeardownMsg{Source: args[0]},
			func(resp *messages.KitTeardownResp) {
				fmt.Printf("Removed %d resources from %s\n", resp.Removed, args[0])
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(teardownCmd)
}
