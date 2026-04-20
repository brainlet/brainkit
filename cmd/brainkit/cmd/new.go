package cmd

import "github.com/spf13/cobra"

// newNewCmd hosts the scaffolders that produce a separate folder with
// their own build graph — today `package` (a deployable .ts bundle)
// and `plugin` (a Go subprocess plugin). Setting up a brainkit runtime
// in the current working directory is `brainkit init` instead.
func newNewCmd() *cobra.Command {
	c := &cobra.Command{Use: "new", Short: "Scaffold a new project"}
	c.AddCommand(newPackageSubCmd())
	c.AddCommand(newPluginSubCmd())
	return c
}
