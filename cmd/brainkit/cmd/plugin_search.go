package cmd

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func addPluginSearchCmds(parent *cobra.Command) {
	searchCmd := &cobra.Command{
		Use: "search <query>", Short: "Search the plugin registry", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.PackagesSearchMsg{Query: args[0]},
				func(resp *sdk.PackagesSearchResp) {
					tw := tabwriter.NewWriter(w(cmd), 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tVERSION\tDESCRIPTION")
					for _, p := range resp.Plugins {
						name := p.Name
						if p.Owner != "" {
							name = p.Owner + "/" + p.Name
						}
						fmt.Fprintf(tw, "%s\t%s\t%s\n", name, p.Version, p.Description)
					}
					tw.Flush()
				},
			)
		},
	}

	infoCmd := &cobra.Command{
		Use: "info <name>", Short: "Show detailed info about an installed plugin", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return connectAndPublish(cmd, sdk.PackagesInfoMsg{Name: args[0]},
				func(resp *sdk.PackagesInfoResp) {
					var manifest struct {
						Name         string   `json:"name"`
						Owner        string   `json:"owner"`
						Version      string   `json:"version"`
						Description  string   `json:"description"`
						Capabilities []string `json:"capabilities"`
					}
					json.Unmarshal(resp.Manifest, &manifest)
					cmd.Printf("Name:         %s\n", manifest.Name)
					if manifest.Owner != "" {
						cmd.Printf("Owner:        %s\n", manifest.Owner)
					}
					cmd.Printf("Version:      %s\n", manifest.Version)
					if manifest.Description != "" {
						cmd.Printf("Description:  %s\n", manifest.Description)
					}
					if len(manifest.Capabilities) > 0 {
						cmd.Printf("Capabilities: %v\n", manifest.Capabilities)
					}
				},
			)
		},
	}

	parent.AddCommand(searchCmd, infoCmd)
}
