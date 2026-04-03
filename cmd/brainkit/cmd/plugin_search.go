package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

// --- search ---

var pluginSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the plugin registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PackagesSearchMsg{Query: args[0]},
			func(resp *messages.PackagesSearchResp) {
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION")
				for _, p := range resp.Plugins {
					name := p.Name
					if p.Owner != "" {
						name = p.Owner + "/" + p.Name
					}
					fmt.Fprintf(w, "%s\t%s\t%s\n", name, p.Version, p.Description)
				}
				w.Flush()
			},
		)
	},
}

// --- info ---

var pluginInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed info about an installed plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectAndPublish(
			messages.PackagesInfoMsg{Name: args[0]},
			func(resp *messages.PackagesInfoResp) {
				var manifest struct {
					Name         string   `json:"name"`
					Owner        string   `json:"owner"`
					Version      string   `json:"version"`
					Description  string   `json:"description"`
					Capabilities []string `json:"capabilities"`
				}
				json.Unmarshal(resp.Manifest, &manifest)

				fmt.Printf("Name:         %s\n", manifest.Name)
				if manifest.Owner != "" {
					fmt.Printf("Owner:        %s\n", manifest.Owner)
				}
				fmt.Printf("Version:      %s\n", manifest.Version)
				if manifest.Description != "" {
					fmt.Printf("Description:  %s\n", manifest.Description)
				}
				if len(manifest.Capabilities) > 0 {
					fmt.Printf("Capabilities: %v\n", manifest.Capabilities)
				}
			},
		)
	},
}

func init() {
	pluginCmd.AddCommand(pluginSearchCmd, pluginInfoCmd)
}
