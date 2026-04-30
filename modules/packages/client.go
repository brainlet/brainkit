package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// DeployResult is returned by Deploy. Mirrors packagemsg.PackageDeployResp.
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

// Package describes a deployment unit.
type Package struct {
	Name    string            `json:"name"`
	Version string            `json:"version,omitempty"`
	Entry   string            `json:"entry,omitempty"`
	Files   map[string]string `json:"files,omitempty"`

	path string `json:"-"`
}

// Inline builds a Package from an inline source string. No bundling.
func Inline(name, entry, source string) Package {
	return Package{
		Name:  name,
		Entry: entry,
		Files: map[string]string{entry: source},
	}
}

// FromDir loads a package from a directory containing manifest.json and source files.
func FromDir(dir string) (Package, error) {
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return Package{}, fmt.Errorf("packages.FromDir: read manifest: %w", err)
	}
	var m struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Entry   string `json:"entry"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return Package{}, fmt.Errorf("packages.FromDir: parse manifest: %w", err)
	}
	return Package{
		Name:    m.Name,
		Version: m.Version,
		Entry:   m.Entry,
		path:    dir,
	}, nil
}

// FromFile loads a single .ts file as a virtual package.
func FromFile(path string) (Package, error) {
	if _, err := os.Stat(path); err != nil {
		return Package{}, fmt.Errorf("packages.FromFile: %w", err)
	}
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return Package{
		Name:  name,
		Entry: filepath.Base(path),
		path:  path,
	}, nil
}

// Deploy deploys a package through the package.deploy module command.
func Deploy(ctx context.Context, rt sdk.CallerRuntime, pkg Package) (DeployResult, error) {
	if !hasPackageCommand(rt, (packagemsg.PackageDeployMsg{}).BusTopic()) {
		return DeployResult{}, &sdkerrors.NotConfiguredError{Feature: "packages"}
	}
	msg, err := pkg.toDeployMsg()
	if err != nil {
		return DeployResult{}, err
	}
	resp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](rt, ctx, msg, sdk.WithCallTimeout(30*time.Second))
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

// Teardown removes a deployed package by name through package.teardown.
func Teardown(ctx context.Context, rt sdk.CallerRuntime, name string) error {
	if !hasPackageCommand(rt, (packagemsg.PackageTeardownMsg{}).BusTopic()) {
		return &sdkerrors.NotConfiguredError{Feature: "packages"}
	}
	_, err := sdk.Call[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](
		rt, ctx, packagemsg.PackageTeardownMsg{Name: name}, sdk.WithCallTimeout(15*time.Second))
	return err
}

// Get returns info about a deployed package by name.
func Get(ctx context.Context, rt sdk.CallerRuntime, name string) (DeploymentInfo, bool, error) {
	if !hasPackageCommand(rt, (packagemsg.PackageDeployInfoMsg{}).BusTopic()) {
		return DeploymentInfo{}, false, &sdkerrors.NotConfiguredError{Feature: "packages"}
	}
	resp, err := sdk.Call[packagemsg.PackageDeployInfoMsg, packagemsg.PackageDeployInfoResp](
		rt, ctx, packagemsg.PackageDeployInfoMsg{Name: name}, sdk.WithCallTimeout(5*time.Second))
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
func List(ctx context.Context, rt sdk.CallerRuntime) ([]DeploymentInfo, error) {
	if !hasPackageCommand(rt, (packagemsg.PackageListDeployedMsg{}).BusTopic()) {
		return nil, &sdkerrors.NotConfiguredError{Feature: "packages"}
	}
	resp, err := sdk.Call[packagemsg.PackageListDeployedMsg, packagemsg.PackageListDeployedResp](
		rt, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	out := make([]DeploymentInfo, 0, len(resp.Packages))
	for _, p := range resp.Packages {
		out = append(out, DeploymentInfo{
			Name:    p.Name,
			Version: p.Version,
			Source:  p.Source,
			Status:  p.Status,
		})
	}
	return out, nil
}

func (p Package) toDeployMsg() (packagemsg.PackageDeployMsg, error) {
	if p.path != "" {
		return packagemsg.PackageDeployMsg{Path: p.path}, nil
	}
	if p.Name == "" {
		return packagemsg.PackageDeployMsg{}, fmt.Errorf("packages: Package.Name is required for inline deploy")
	}
	if p.Entry == "" {
		return packagemsg.PackageDeployMsg{}, fmt.Errorf("packages: Package.Entry is required for inline deploy")
	}
	if len(p.Files) == 0 {
		return packagemsg.PackageDeployMsg{}, fmt.Errorf("packages: Package.Files is required for inline deploy")
	}
	manifest := map[string]string{"name": p.Name, "entry": p.Entry}
	if p.Version != "" {
		manifest["version"] = p.Version
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return packagemsg.PackageDeployMsg{}, err
	}
	return packagemsg.PackageDeployMsg{Manifest: raw, Files: p.Files}, nil
}

type mountedModuleLister interface {
	MountedModules() []bkmodule.Descriptor
}

func hasPackageCommand(rt sdk.CallerRuntime, topic string) bool {
	lister, ok := rt.(mountedModuleLister)
	if !ok {
		return true
	}
	for _, desc := range lister.MountedModules() {
		for _, cmd := range desc.Commands {
			if cmd.Topic == topic {
				return true
			}
		}
	}
	return false
}
