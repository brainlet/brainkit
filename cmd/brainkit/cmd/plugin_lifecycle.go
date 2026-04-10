package cmd

import (
	"strings"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func addPluginLifecycleCmds(parent *cobra.Command) {
	var (
		pluginStartBinary string
		pluginStartEnv    []string
	)

	startCmd := &cobra.Command{
		Use: "start <name>", Short: "Start an installed plugin", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := sdk.PluginStartMsg{Name: args[0], Binary: pluginStartBinary}
			if len(pluginStartEnv) > 0 {
				msg.Env = make(map[string]string)
				for _, kv := range pluginStartEnv {
					parts := strings.SplitN(kv, "=", 2)
					if len(parts) == 2 {
						msg.Env[parts[0]] = parts[1]
					}
				}
			}
			return connectAndPublish(cmd, msg,
				func(resp *sdk.PluginStartResp) { cmd.Printf("Started %s (pid %d)\n", resp.Name, resp.PID) },
			)
		},
	}
	startCmd.Flags().StringVar(&pluginStartBinary, "binary", "", "explicit binary path (overrides installed path)")
	startCmd.Flags().StringArrayVar(&pluginStartEnv, "env", nil, "environment variables (KEY=VALUE, repeatable)")

	stopCmd := &cobra.Command{
		Use: "stop <name>", Short: "Stop a running plugin", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.PluginStopMsg{Name: args[0]},
				func(resp *sdk.PluginStopResp) { cmd.Printf("Stopped %s\n", args[0]) },
			)
		},
	}

	parent.AddCommand(startCmd, stopCmd)
}
