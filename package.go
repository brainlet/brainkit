package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/deploy"
	"github.com/brainlet/brainkit/sdk"
)

// DeployResult is returned by (*Kit).Deploy. Mirrors sdk.PackageDeployResp.
type DeployResult struct {
	Name      string
	Version   string
	Source    string
	Resources []sdk.ResourceInfo
}

// DeploymentInfo describes a deployed package.
type DeploymentInfo struct {
	Name    string
	Version string
	Source  string
	Status  string
}

// PackageInline builds a Package from an inline source string. No bundling.
// Use for small single-file deployments; name becomes the package identifier,
// entry is the filename the runtime sees (e.g. "agents.ts"), source is the code.
func PackageInline(name, entry, source string) Package {
	return Package{
		Name:  name,
		Entry: entry,
		Files: map[string]string{entry: source},
	}
}

// PackageFromDir loads a package from a directory containing manifest.json and
// source files. The dir path is carried to the handler which bundles via esbuild.
func PackageFromDir(dir string) (Package, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Package{}, fmt.Errorf("brainkit.PackageFromDir: read manifest: %w", err)
	}
	var m deploy.PackageManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Package{}, fmt.Errorf("brainkit.PackageFromDir: parse manifest: %w", err)
	}
	return Package{
		Name:    m.Name,
		Version: m.Version,
		Entry:   m.Entry,
		path:    dir,
	}, nil
}

// PackageFromFile loads a single .ts file as a virtual package. Imports are
// resolved via esbuild at deploy time. The package name is the filename stem.
func PackageFromFile(path string) (Package, error) {
	if _, err := os.Stat(path); err != nil {
		return Package{}, fmt.Errorf("brainkit.PackageFromFile: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Package{
		Name:  name,
		Entry: filepath.Base(path),
		path:  path,
	}, nil
}

// path is set by PackageFromDir/PackageFromFile; package_path.go adds it
// through a dedicated field. Kept here as an unexported field on Package.

// Deploy deploys a package into the Kit. Hot-replaces an existing deployment
// with the same name.
func (k *Kit) Deploy(ctx context.Context, pkg Package) (DeployResult, error) {
	msg, err := pkg.toDeployMsg()
	if err != nil {
		return DeployResult{}, err
	}
	resp, err := Call[sdk.PackageDeployMsg, sdk.PackageDeployResp](k, ctx, msg, WithCallTimeout(30*time.Second))
	if err != nil {
		return DeployResult{}, err
	}
	return DeployResult{
		Name:      resp.Name,
		Version:   resp.Version,
		Source:    resp.Source,
		Resources: resp.Resources,
	}, nil
}

// Teardown removes a deployed package by name.
func (k *Kit) Teardown(ctx context.Context, name string) error {
	_, err := Call[sdk.PackageTeardownMsg, sdk.PackageTeardownResp](
		k, ctx, sdk.PackageTeardownMsg{Name: name}, WithCallTimeout(15*time.Second))
	return err
}

// Get returns info about a deployed package by name.
func (k *Kit) Get(ctx context.Context, name string) (DeploymentInfo, bool, error) {
	resp, err := Call[sdk.PackageDeployInfoMsg, sdk.PackageDeployInfoResp](
		k, ctx, sdk.PackageDeployInfoMsg{Name: name}, WithCallTimeout(5*time.Second))
	if err != nil {
		return DeploymentInfo{}, false, nil
	}
	return DeploymentInfo{
		Name:    resp.Name,
		Version: resp.Version,
		Source:  resp.Source,
		Status:  "active",
	}, true, nil
}

// List returns all deployed packages.
func (k *Kit) List(ctx context.Context) ([]DeploymentInfo, error) {
	resp, err := Call[sdk.PackageListDeployedMsg, sdk.PackageListDeployedResp](
		k, ctx, sdk.PackageListDeployedMsg{}, WithCallTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	out := make([]DeploymentInfo, 0, len(resp.Packages))
	for _, p := range resp.Packages {
		out = append(out, DeploymentInfo{
			Name: p.Name, Version: p.Version, Source: p.Source, Status: p.Status,
		})
	}
	return out, nil
}

func (p Package) toDeployMsg() (sdk.PackageDeployMsg, error) {
	// Filesystem path: handler reads manifest.json + bundles.
	if p.path != "" {
		return sdk.PackageDeployMsg{Path: p.path}, nil
	}
	// Inline: name + entry + files.
	if p.Name == "" {
		return sdk.PackageDeployMsg{}, fmt.Errorf("brainkit: Package.Name is required for inline deploy")
	}
	if p.Entry == "" {
		return sdk.PackageDeployMsg{}, fmt.Errorf("brainkit: Package.Entry is required for inline deploy")
	}
	if len(p.Files) == 0 {
		return sdk.PackageDeployMsg{}, fmt.Errorf("brainkit: Package.Files is required for inline deploy")
	}
	manifest := map[string]string{"name": p.Name, "entry": p.Entry}
	if p.Version != "" {
		manifest["version"] = p.Version
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return sdk.PackageDeployMsg{}, err
	}
	return sdk.PackageDeployMsg{Manifest: raw, Files: p.Files}, nil
}
