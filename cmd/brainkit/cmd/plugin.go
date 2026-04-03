package cmd

import (
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
}

// --- install ---

var pluginInstallVersion string

var pluginInstallCmd = &cobra.Command{
	Use:   "install <name>",
	Short: "Install a plugin from the registry",
	Long:  "Install a plugin binary. Name format: owner/name or just name (defaults to brainlet owner).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PackagesInstallMsg{Name: args[0], Version: pluginInstallVersion},
			func(resp *messages.PackagesInstallResp) {
				fmt.Printf("Installed %s v%s\n", resp.Name, resp.Version)
				fmt.Printf("  binary: %s\n", resp.Path)
			},
		)
	},
}

// --- remove ---

var pluginRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PackagesRemoveMsg{Name: args[0]},
			func(resp *messages.PackagesRemoveResp) {
				fmt.Printf("Removed %s\n", args[0])
			},
		)
	},
}

// --- update ---

var pluginUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Update a plugin to the latest version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PackagesUpdateMsg{Name: args[0]},
			func(resp *messages.PackagesUpdateResp) {
				if resp.Updated {
					fmt.Printf("Updated %s: %s → %s\n", args[0], resp.OldVersion, resp.NewVersion)
				} else {
					fmt.Printf("%s already at latest (%s)\n", args[0], resp.OldVersion)
				}
			},
		)
	},
}

func init() {
	pluginInstallCmd.Flags().StringVar(&pluginInstallVersion, "version", "", "specific version to install (default: latest)")
	pluginCmd.AddCommand(pluginInstallCmd, pluginRemoveCmd, pluginUpdateCmd)
	rootCmd.AddCommand(pluginCmd)
}
