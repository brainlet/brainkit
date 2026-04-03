package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

var redeployCmd = &cobra.Command{
	Use:   "redeploy <file-or-dir>",
	Short: "Redeploy a .ts file or module directory",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", path, err)
		}

		if info.IsDir() {
			return redeployDirectory(path)
		}
		return redeployFile(path)
	},
}

func redeployFile(path string) error {
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
}

func redeployDirectory(path string) error {
	manifestData, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var manifest struct {
		Name     string                       `json:"name"`
		Services map[string]json.RawMessage   `json:"services"`
	}
	json.Unmarshal(manifestData, &manifest)

	// Teardown each service by its deployed source name (packageName/serviceName.ts)
	for svcName := range manifest.Services {
		source := manifest.Name + "/" + svcName + ".ts"
		connectAndPublish(
			messages.KitTeardownMsg{Source: source},
			func(resp *messages.KitTeardownResp) {},
		)
	}

	// Deploy fresh
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
			fmt.Printf("Redeployed package %s v%s\n", resp.Name, resp.Version)
			for _, svc := range resp.Services {
				fmt.Printf("  service: %s\n", svc)
			}
		},
	)
}

func init() {
	rootCmd.AddCommand(redeployCmd)
}
