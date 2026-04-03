package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var deployRole string

var deployCmd = &cobra.Command{
	Use:   "deploy <file-or-dir>",
	Short: "Deploy a .ts file or module directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}

		if info.IsDir() {
			return deployDirectory(path)
		}
		return deployFile(path)
	},
}

func deployFile(path string) error {
	code, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	source := filepath.Base(path)

	return connectAndPublish(
		messages.KitDeployMsg{Source: source, Code: string(code)},
		func(resp *messages.KitDeployResp) {
			fmt.Printf("Deployed %s\n", source)
			for _, r := range resp.Resources {
				fmt.Printf("  %s: %s\n", r.Type, r.Name)
			}
		},
	)
}

func deployDirectory(path string) error {
	manifestData, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	files := make(map[string]string)
	entries, _ := os.ReadDir(path)
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".ts" {
			data, err := os.ReadFile(filepath.Join(path, e.Name()))
			if err != nil {
				return err
			}
			files[e.Name()] = string(data)
		}
	}

	return connectAndPublish(
		messages.PackageDeployMsg{
			Manifest: json.RawMessage(manifestData),
			Files:    files,
		},
		func(resp *messages.PackageDeployResp) {
			fmt.Printf("Deployed package %s v%s\n", resp.Name, resp.Version)
			for _, svc := range resp.Services {
				fmt.Printf("  service: %s\n", svc)
			}
		},
	)
}

func init() {
	deployCmd.Flags().StringVar(&deployRole, "role", "", "RBAC role for the deployment")
	rootCmd.AddCommand(deployCmd)
}
