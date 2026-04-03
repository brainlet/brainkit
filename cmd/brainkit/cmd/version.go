package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print brainkit version",
	Run: func(cmd *cobra.Command, args []string) {
		if jsonOutput {
			fmt.Printf(`{"version":"%s","commit":"%s"}`, Version, Commit)
			fmt.Println()
		} else {
			fmt.Printf("brainkit version %s (commit %s)\n", Version, Commit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
