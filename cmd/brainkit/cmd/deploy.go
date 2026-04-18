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
	var endpoint string
	c := &cobra.Command{
		Use:   "deploy <file-or-dir>",
		Short: "Deploy a .ts file or package directory to a running server",
		Long: `Deploy posts a package to a running brainkit server via
POST /api/bus. A plain .ts file ships as a single-entry inline
package; a directory ships with its manifest.json plus every
.ts under it (excluding node_modules, .git, and types/).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("cannot read %s: %w", path, err)
			}

			msg, err := buildDeployMsg(path, info)
			if err != nil {
				return err
			}

			ctx, cancel := withTimeout(cmd.Context())
			defer cancel()

			payload, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("marshal deploy msg: %w", err)
			}

			client := newBusClient(endpoint)
			reply, err := client.call(ctx, msg.BusTopic(), payload)
			if err != nil {
				return err
			}

			var resp sdk.PackageDeployResp
			if err := json.Unmarshal(reply, &resp); err != nil {
				return fmt.Errorf("decode response: %w (body: %s)", err, string(reply))
			}

			if jsonOutput {
				return writeJSONPretty(cmd.OutOrStdout(), reply)
			}
			version := resp.Version
			if version == "" {
				version = "0.0.0"
			}
			cmd.Printf("Deployed %s v%s (%s)\n", resp.Name, version, resp.Source)
			return nil
		},
	}
	c.Flags().StringVarP(&endpoint, "endpoint", "e", "", "server endpoint (default http://127.0.0.1:8080)")
	return c
}

// buildDeployMsg wraps a filesystem path as a PackageDeployMsg.
// Files send as inline sources so the server doesn't need
// filesystem access to the caller's machine.
func buildDeployMsg(path string, info os.FileInfo) (sdk.PackageDeployMsg, error) {
	if info.IsDir() {
		return buildDirectoryMsg(path)
	}
	return buildFileMsg(path)
}

// buildFileMsg wraps a single .ts file as a one-entry package.
// hello.ts → package "hello", deployed as hello.ts, namespace
// ts.hello.*.
func buildFileMsg(path string) (sdk.PackageDeployMsg, error) {
	code, err := os.ReadFile(path)
	if err != nil {
		return sdk.PackageDeployMsg{}, err
	}
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	manifest := fmt.Sprintf(`{"name":%q,"version":"0.0.0","entry":%q}`, name, filename)
	return sdk.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{filename: string(code)},
	}, nil
}

// buildDirectoryMsg reads manifest.json + every .ts file in the
// directory tree (skipping node_modules / .git / types).
func buildDirectoryMsg(path string) (sdk.PackageDeployMsg, error) {
	manifestData, err := os.ReadFile(filepath.Join(path, "manifest.json"))
	if err != nil {
		return sdk.PackageDeployMsg{}, fmt.Errorf("read manifest: %w", err)
	}
	files := make(map[string]string)
	walkErr := filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", ".git", "types":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(d.Name()) != ".ts" {
			return nil
		}
		rel, _ := filepath.Rel(path, p)
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		files[rel] = string(data)
		return nil
	})
	if walkErr != nil {
		return sdk.PackageDeployMsg{}, fmt.Errorf("walk package: %w", walkErr)
	}
	return sdk.PackageDeployMsg{
		Manifest: json.RawMessage(manifestData),
		Files:    files,
	}, nil
}
