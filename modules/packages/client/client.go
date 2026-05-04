// Package packageclient contains caller-side helpers for package.deploy without
// importing the hot-mount packages module.
package packageclient

import (
	"context"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	packagesource "github.com/brainlet/brainkit/modules/packages/source"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Package describes a deployment unit.
type Package = packagesource.Package

// Inline builds a Package from an inline source string.
func Inline(name, entry, source string) Package {
	return packagesource.Inline(name, entry, source)
}

// FromDir loads a package from a directory containing manifest.json and source files.
func FromDir(dir string) (Package, error) {
	return packagesource.FromDir(dir)
}

// FromFile loads a single .ts file as a virtual package.
func FromFile(path string) (Package, error) {
	return packagesource.FromFile(path)
}

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

// Deploy deploys a package through the package.deploy module command.
func Deploy(ctx context.Context, rt sdk.CallerRuntime, pkg Package) (DeployResult, error) {
	if !hasPackageCommand(rt, (packagemsg.PackageDeployMsg{}).BusTopic()) {
		return DeployResult{}, &sdkerrors.NotConfiguredError{Feature: "packages"}
	}
	payload, err := pkg.DeployPayload()
	if err != nil {
		return DeployResult{}, err
	}
	resp, err := sdk.Call[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](rt, ctx, packagemsg.PackageDeployMsg{
		Path:     payload.Path,
		Manifest: payload.Manifest,
		Files:    payload.Files,
	}, sdk.WithCallTimeout(30*time.Second))
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
