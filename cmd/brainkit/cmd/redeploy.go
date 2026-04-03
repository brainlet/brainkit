package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var redeployCmd = &cobra.Command{
	Use:   "redeploy <file>",
	Short: "Redeploy a .ts file (teardown + deploy, preserves metadata)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		code, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := filepath.Base(path)

		return connectAndPublish(
			messages.KitRedeployMsg{Source: source, Code: string(code)},
			func(resp *messages.KitRedeployResp) {
				fmt.Printf("Redeployed %s\n", source)
				for _, r := range resp.Resources {
					fmt.Printf("  %s: %s\n", r.Type, r.Name)
				}
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(redeployCmd)
}
