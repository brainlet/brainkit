package cmd

import (
	"github.com/spf13/cobra"
)

func newPluginCmd() *cobra.Command {
	pluginCmd := &cobra.Command{Use: "plugin", Short: "Manage plugins"}

	addPluginListCmd(pluginCmd)
	addPluginLifecycleCmds(pluginCmd)
	return pluginCmd
}
