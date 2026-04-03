package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed and running plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return err
		}
		client, err := config.Connect(cfg)
		if err != nil {
			return err
		}
		defer client.Close()

		installed, installErr := busRequest[messages.PackagesListMsg, messages.PackagesListResp](client, messages.PackagesListMsg{})
		running, runErr := busRequest[messages.PluginListRunningMsg, messages.PluginListRunningResp](client, messages.PluginListRunningMsg{})

		runningByName := map[string]messages.RunningPluginInfo{}
		if runErr == nil && running != nil {
			for _, p := range running.Plugins {
				runningByName[p.Name] = p
			}
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tSTATUS\tPID\tUPTIME")

		if installErr == nil && installed != nil {
			for _, p := range installed.Plugins {
				if rp, ok := runningByName[p.Name]; ok {
					fmt.Fprintf(w, "%s\t%s\trunning\t%d\t%s\n", p.Name, p.Version, rp.PID, rp.Uptime)
					delete(runningByName, p.Name)
				} else {
					fmt.Fprintf(w, "%s\t%s\tinstalled\t-\t-\n", p.Name, p.Version)
				}
			}
		}

		for _, rp := range runningByName {
			fmt.Fprintf(w, "%s\t-\trunning\t%d\t%s\n", rp.Name, rp.PID, rp.Uptime)
		}

		w.Flush()
		return nil
	},
}

func init() {
	pluginCmd.AddCommand(pluginListCmd)
}
