package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// newPluginCmd groups plugin-lifecycle verbs — list / restart /
// start / stop. Each lands on `plugin.*` bus topics on the running
// server. `inspect plugins` overlaps with `plugin list` and is kept
// for consistency with the other inspect subjects.
func newPluginCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "plugin",
		Short: "Manage brainkit plugins",
		Long: `Plugin subcommands control the lifecycle of installed plugins on
a running server. Use these when you need to restart a plugin after
a secret rotation, stop a misbehaving plugin, or confirm which
plugins are currently running.`,
	}
	c.AddCommand(newPluginListSubCmd(), newPluginRestartSubCmd(), newPluginStartSubCmd(), newPluginStopSubCmd())
	return c
}

func newPluginListSubCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "list",
		Short: "List running plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			reply, err := client.call(ctx, "plugin.list", json.RawMessage("{}"))
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			return renderPluginList(cmd.OutOrStdout(), reply)
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func newPluginRestartSubCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "restart <name>",
		Short: "Restart a running plugin",
		Long: `Restart stops and re-launches a plugin without restarting the
server. Used after a secret rotation or config change that the
plugin picks up at init time.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": args[0]})
			reply, err := client.call(ctx, "plugin.restart", body)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Printf("Plugin %s restarted\n", args[0])
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func newPluginStartSubCmd() *cobra.Command {
	var (
		endpoint string
		binary   string
		envArgs  []string
	)
	c := &cobra.Command{
		Use:   "start <name>",
		Short: "Start an installed plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			msg := map[string]any{"name": args[0]}
			if binary != "" {
				msg["binary"] = binary
			}
			if len(envArgs) > 0 {
				env := map[string]string{}
				for _, kv := range envArgs {
					parts := strings.SplitN(kv, "=", 2)
					if len(parts) == 2 {
						env[parts[0]] = parts[1]
					}
				}
				msg["env"] = env
			}
			body, err := json.Marshal(msg)
			if err != nil {
				return err
			}
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			reply, err := client.call(ctx, "plugin.start", body)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			var resp struct {
				Name string `json:"name"`
				PID  int    `json:"pid"`
			}
			_ = json.Unmarshal(reply, &resp)
			cmd.Printf("Started %s (pid %d)\n", nonEmpty(resp.Name, args[0]), resp.PID)
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	c.Flags().StringVar(&binary, "binary", "", "explicit binary path (overrides installed path)")
	c.Flags().StringArrayVar(&envArgs, "env", nil, "environment variables (KEY=VALUE, repeatable)")
	return c
}

func newPluginStopSubCmd() *cobra.Command {
	var endpoint string
	c := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running plugin",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()
			client := newBusClient(endpoint)
			body, _ := json.Marshal(map[string]string{"name": args[0]})
			reply, err := client.call(ctx, "plugin.stop", body)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			cmd.Printf("Stopped %s\n", args[0])
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

func renderPluginList(w io.Writer, payload json.RawMessage) error {
	var resp struct {
		Plugins []struct {
			Name     string `json:"name"`
			Version  string `json:"version"`
			PID      int    `json:"pid"`
			Identity string `json:"identity"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(payload, &resp); err != nil {
		return writeJSONPretty(w, payload)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tPID\tIDENTITY")
	for _, p := range resp.Plugins {
		fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n",
			nonEmpty(p.Name, "-"),
			nonEmpty(p.Version, "-"),
			p.PID,
			nonEmpty(p.Identity, "-"))
	}
	return tw.Flush()
}
