package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print brainkit version",
		Run: func(cmd *cobra.Command, args []string) {
			if jsonOutput {
				cmd.Printf(`{"version":"%s","commit":"%s"}`, Version, Commit)
				cmd.Println()
			} else {
				cmd.Println(fmt.Sprintf("brainkit version %s (commit %s)", Version, Commit))
			}
		},
	}
}
