package cmd

import (
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newPluginCmd() *cobra.Command {
	var pluginInstallVersion string

	pluginCmd := &cobra.Command{Use: "plugin", Short: "Manage plugins"}

	installCmd := &cobra.Command{
		Use: "install <name>", Short: "Install a plugin from the registry",
		Long: "Install a plugin binary. Name format: owner/name or just name (defaults to brainlet owner).",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.PackagesInstallMsg{Name: args[0], Version: pluginInstallVersion},
				func(resp *messages.PackagesInstallResp) {
					cmd.Printf("Installed %s v%s\n", resp.Name, resp.Version)
					cmd.Printf("  binary: %s\n", resp.Path)
				},
			)
		},
	}
	installCmd.Flags().StringVar(&pluginInstallVersion, "version", "", "specific version to install (default: latest)")

	removeCmd := &cobra.Command{
		Use: "remove <name>", Short: "Remove an installed plugin", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.PackagesRemoveMsg{Name: args[0]},
				func(resp *messages.PackagesRemoveResp) { cmd.Printf("Removed %s\n", args[0]) },
			)
		},
	}

	updateCmd := &cobra.Command{
		Use: "update <name>", Short: "Update a plugin to the latest version", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, messages.PackagesUpdateMsg{Name: args[0]},
				func(resp *messages.PackagesUpdateResp) {
					if resp.Updated {
						cmd.Printf("Updated %s: %s → %s\n", args[0], resp.OldVersion, resp.NewVersion)
					} else {
						cmd.Printf("%s already at latest (%s)\n", args[0], resp.OldVersion)
					}
				},
			)
		},
	}

	pluginCmd.AddCommand(installCmd, removeCmd, updateCmd)
	addPluginListCmd(pluginCmd)
	addPluginLifecycleCmds(pluginCmd)
	addPluginSearchCmds(pluginCmd)
	return pluginCmd
}
