package engine

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/packages"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// PackagesDomain handles packages.search/install/remove/update/list/info bus commands.
type PackagesDomain struct {
	packages *packages.Manager
}

func newPackagesDomain(mgr *packages.Manager) *PackagesDomain {
	return &PackagesDomain{packages: mgr}
}

func (d *PackagesDomain) Search(_ context.Context, req sdk.PackagesSearchMsg) (*sdk.PackagesSearchResp, error) {
	results, err := d.packages.Search(req.Query, req.Capabilities)
	if err != nil {
		return nil, err
	}
	summaries := make([]sdk.PluginSummary, len(results))
	for i, r := range results {
		summaries[i] = sdk.PluginSummary{
			Name: r.Name, Owner: r.Owner, Version: r.Version,
			Description: r.Description, Capabilities: r.Capabilities,
		}
	}
	return &sdk.PackagesSearchResp{Plugins: summaries}, nil
}

func (d *PackagesDomain) Install(_ context.Context, req sdk.PackagesInstallMsg) (*sdk.PackagesInstallResp, error) {
	owner, name := parsePluginName(req.Name)
	installed, err := d.packages.Install(owner, name, req.Version)
	if err != nil {
		return nil, err
	}
	return &sdk.PackagesInstallResp{
		Installed: true, Name: installed.Name,
		Version: installed.Version, Path: installed.BinaryPath,
	}, nil
}

func (d *PackagesDomain) Remove(_ context.Context, req sdk.PackagesRemoveMsg) (*sdk.PackagesRemoveResp, error) {
	if err := d.packages.Remove(req.Name); err != nil {
		return nil, err
	}
	return &sdk.PackagesRemoveResp{Removed: true}, nil
}

func (d *PackagesDomain) Update(_ context.Context, req sdk.PackagesUpdateMsg) (*sdk.PackagesUpdateResp, error) {
	owner, name := parsePluginName(req.Name)
	old, newVer, err := d.packages.Update(owner, name)
	if err != nil {
		return nil, err
	}
	return &sdk.PackagesUpdateResp{Updated: old != newVer, OldVersion: old, NewVersion: newVer}, nil
}

func (d *PackagesDomain) List(_ context.Context, _ sdk.PackagesListMsg) (*sdk.PackagesListResp, error) {
	installed, err := d.packages.ListInstalled()
	if err != nil {
		return nil, err
	}
	infos := make([]sdk.InstalledPluginInfo, len(installed))
	for i, p := range installed {
		infos[i] = sdk.InstalledPluginInfo{
			Name: p.Name, Owner: p.Owner, Version: p.Version,
			BinaryPath: p.BinaryPath, InstalledAt: p.InstalledAt.Format(time.RFC3339),
		}
	}
	return &sdk.PackagesListResp{Plugins: infos}, nil
}

func (d *PackagesDomain) Info(_ context.Context, req sdk.PackagesInfoMsg) (*sdk.PackagesInfoResp, error) {
	installed, err := d.packages.GetInstalled(req.Name)
	if err != nil {
		return nil, err
	}
	return &sdk.PackagesInfoResp{Manifest: json.RawMessage(installed.Manifest)}, nil
}

func parsePluginName(fullName string) (owner, name string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "brainlet", fullName // default owner
}

// kitStoreAdapter bridges types.KitStore (kit package) to packages.PluginStore interface.
type kitStoreAdapter struct {
	store types.KitStore
}

func (a *kitStoreAdapter) SaveInstalled(name, owner, version, binaryPath, manifest string, installedAt time.Time) error {
	return a.store.SaveInstalledPlugin(types.InstalledPlugin{
		Name: name, Owner: owner, Version: version,
		BinaryPath: binaryPath, Manifest: manifest, InstalledAt: installedAt,
	})
}

func (a *kitStoreAdapter) LoadInstalled() ([]packages.InstalledRecord, error) {
	plugins, err := a.store.LoadInstalledPlugins()
	if err != nil {
		return nil, err
	}
	records := make([]packages.InstalledRecord, len(plugins))
	for i, p := range plugins {
		records[i] = packages.InstalledRecord{
			Name: p.Name, Owner: p.Owner, Version: p.Version,
			BinaryPath: p.BinaryPath, Manifest: p.Manifest, InstalledAt: p.InstalledAt,
		}
	}
	return records, nil
}

func (a *kitStoreAdapter) DeleteInstalled(name string) error {
	return a.store.DeleteInstalledPlugin(name)
}
