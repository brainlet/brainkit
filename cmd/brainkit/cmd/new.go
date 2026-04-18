package cmd

import "github.com/spf13/cobra"

func newNewCmd() *cobra.Command {
	c := &cobra.Command{Use: "new", Short: "Scaffold a new project"}
	c.AddCommand(newPackageSubCmd())
	c.AddCommand(newPluginSubCmd())
	c.AddCommand(newServerSubCmd())
	return c
}
