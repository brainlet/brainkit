package cmd

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

// --- start ---

var (
	pluginStartBinary string
	pluginStartEnv    []string
	pluginStartRole   string
)

var pluginStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		msg := messages.PluginStartMsg{
			Name:   args[0],
			Binary: pluginStartBinary,
			Role:   pluginStartRole,
		}

		if len(pluginStartEnv) > 0 {
			msg.Env = make(map[string]string)
			for _, kv := range pluginStartEnv {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) == 2 {
					msg.Env[parts[0]] = parts[1]
				}
			}
		}

		return connectAndPublish(msg,
			func(resp *messages.PluginStartResp) {
				fmt.Printf("Started %s (pid %d)\n", resp.Name, resp.PID)
			},
		)
	},
}

// --- stop ---

var pluginStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PluginStopMsg{Name: args[0]},
			func(resp *messages.PluginStopResp) {
				fmt.Printf("Stopped %s\n", args[0])
			},
		)
	},
}

func init() {
	pluginStartCmd.Flags().StringVar(&pluginStartBinary, "binary", "", "explicit binary path (overrides installed path)")
	pluginStartCmd.Flags().StringArrayVar(&pluginStartEnv, "env", nil, "environment variables (KEY=VALUE, repeatable)")
	pluginStartCmd.Flags().StringVar(&pluginStartRole, "role", "", "RBAC role assignment")
	pluginCmd.AddCommand(pluginStartCmd, pluginStopCmd)
}
