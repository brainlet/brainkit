package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/brainlet/brainkit/sdk"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	var deployRole string

	c := &cobra.Command{
		Use:   "deploy <file-or-dir>",
		Short: "Deploy a .ts file or package directory",
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

// deployFile wraps a single .ts file as a virtual package.
// hello.ts → package "hello", deployed as hello.ts, namespace ts.hello.*
func deployFile(cmd *cobra.Command, path string) error {
	code, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	manifest := fmt.Sprintf(`{"name":%q,"version":"0.0.0","entry":%q}`, name, filename)
	return connectAndPublish(cmd, sdk.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{filename: string(code)},
	},
		func(resp *sdk.PackageDeployResp) {
			cmd.Printf("Deployed %s (%s)\n", resp.Name, resp.Source)
		},
	)
}

// deployDirectory reads manifest + recursively collects all .ts files for inline deploy.
func deployDirectory(cmd *cobra.Command, path string) error {
	manifestData, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	files := make(map[string]string)
	filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "types" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) == ".ts" {
			rel, _ := filepath.Rel(path, p)
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return readErr
			}
			files[rel] = string(data)
		}
		return nil
	})
	return connectAndPublish(cmd, sdk.PackageDeployMsg{
		Manifest: json.RawMessage(manifestData),
		Files:    files,
	},
		func(resp *sdk.PackageDeployResp) {
			cmd.Printf("Deployed %s v%s (%s)\n", resp.Name, resp.Version, resp.Source)
		},
	)
}
