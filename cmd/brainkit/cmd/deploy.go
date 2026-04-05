package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	var deployRole string

	c := &cobra.Command{
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
				return deployDirectory(cmd, path)
			}
			return deployFile(cmd, path)
		},
	}
	c.Flags().StringVar(&deployRole, "role", "", "RBAC role for the deployment")
	return c
}

// deployFile wraps a single .ts file as a single-service package.
// echo.ts → package "echo", service "echo", deployed as echo/echo.ts.
// Same namespace convention as directory packages — no special case.
func deployFile(cmd *cobra.Command, path string) error {
	code, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	manifest := fmt.Sprintf(`{"name":%q,"version":"0.0.0","services":{%q:{"entry":%q}}}`,
		name, name, filename)
	return connectAndPublish(cmd, messages.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{filename: string(code)},
	},
		func(resp *messages.PackageDeployResp) {
			cmd.Printf("Deployed %s\n", name)
			for _, svc := range resp.Services {
				cmd.Printf("  service: %s\n", svc)
			}
		},
	)
}

func deployDirectory(cmd *cobra.Command, path string) error {
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
	return connectAndPublish(cmd, messages.PackageDeployMsg{Manifest: json.RawMessage(manifestData), Files: files},
		func(resp *messages.PackageDeployResp) {
			cmd.Printf("Deployed package %s v%s\n", resp.Name, resp.Version)
			for _, svc := range resp.Services {
				cmd.Printf("  service: %s\n", svc)
			}
		},
	)
}
