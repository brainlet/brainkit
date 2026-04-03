package cmd

import "github.com/spf13/cobra"

func newNewCmd() *cobra.Command {
	c := &cobra.Command{Use: "new", Short: "Scaffold a new project"}
	c.AddCommand(newModuleSubCmd())
	c.AddCommand(newPluginSubCmd())
	return c
}
