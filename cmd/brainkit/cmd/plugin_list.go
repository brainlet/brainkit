package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/brainlet/brainkit/cmd/brainkit/config"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func addPluginListCmd(parent *cobra.Command) {
	parent.AddCommand(&cobra.Command{
		Use: "list", Short: "List installed and running plugins",
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

			installed, installErr := httpBusRequest[messages.PackagesListMsg, messages.PackagesListResp](client, messages.PackagesListMsg{})
			running, runErr := httpBusRequest[messages.PluginListRunningMsg, messages.PluginListRunningResp](client, messages.PluginListRunningMsg{})

			runningByName := map[string]messages.RunningPluginInfo{}
			if runErr == nil && running != nil {
				for _, p := range running.Plugins {
					runningByName[p.Name] = p
				}
			}

			tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tVERSION\tSTATUS\tPID\tUPTIME")

			if installErr == nil && installed != nil {
				for _, p := range installed.Plugins {
					if rp, ok := runningByName[p.Name]; ok {
						fmt.Fprintf(tw, "%s\t%s\trunning\t%d\t%s\n", p.Name, p.Version, rp.PID, rp.Uptime)
						delete(runningByName, p.Name)
					} else {
						fmt.Fprintf(tw, "%s\t%s\tinstalled\t-\t-\n", p.Name, p.Version)
					}
				}
			}
			for _, rp := range runningByName {
				fmt.Fprintf(tw, "%s\t-\trunning\t%d\t%s\n", rp.Name, rp.PID, rp.Uptime)
			}
			tw.Flush()
			return nil
		},
	})
}
