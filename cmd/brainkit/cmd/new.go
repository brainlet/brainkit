package cmd

import "github.com/spf13/cobra"

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Scaffold a new project",
}

func init() {
	rootCmd.AddCommand(newCmd)
}
