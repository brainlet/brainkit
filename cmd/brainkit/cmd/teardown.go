package cmd

import (
	"strings"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newTeardownCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "teardown <name>",
		Short: "Remove a deployed package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSuffix(args[0], ".ts")
			return connectAndPublish(cmd, sdk.PackageTeardownMsg{Name: name},
				func(resp *sdk.PackageTeardownResp) {
					cmd.Printf("Removed %s\n", name)
				},
			)
		},
	}
}
