package engine

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/brainlet/brainkit/internal/packages"
	"github.com/brainlet/brainkit/sdk/messages"
)

// PackagesDomain handles packages.search/install/remove/update/list/info bus commands.
type PackagesDomain struct {
	packages *packages.Manager
}

func newPackagesDomain(mgr *packages.Manager) *PackagesDomain {
	return &PackagesDomain{packages: mgr}
}

func (d *PackagesDomain) Search(_ context.Context, req messages.PackagesSearchMsg) (*messages.PackagesSearchResp, error) {
	results, err := d.packages.Search(req.Query, req.Capabilities)
	if err != nil {
		return nil, err
	}
	summaries := make([]messages.PluginSummary, len(results))
	for i, r := range results {
		summaries[i] = messages.PluginSummary{
			Name: r.Name, Owner: r.Owner, Version: r.Version,
			Description: r.Description, Capabilities: r.Capabilities,
		}
	}
	return &messages.PackagesSearchResp{Plugins: summaries}, nil
}

func (d *PackagesDomain) Install(_ context.Context, req messages.PackagesInstallMsg) (*messages.PackagesInstallResp, error) {
	owner, name := parsePluginName(req.Name)
	installed, err := d.packages.Install(owner, name, req.Version)
	if err != nil {
		return nil, err
	}
	return &messages.PackagesInstallResp{
		Installed: true, Name: installed.Name,
		Version: installed.Version, Path: installed.BinaryPath,
	}, nil
}

func (d *PackagesDomain) Remove(_ context.Context, req messages.PackagesRemoveMsg) (*messages.PackagesRemoveResp, error) {
	if err := d.packages.Remove(req.Name); err != nil {
		return nil, err
	}
	return &messages.PackagesRemoveResp{Removed: true}, nil
}

func (d *PackagesDomain) Update(_ context.Context, req messages.PackagesUpdateMsg) (*messages.PackagesUpdateResp, error) {
	owner, name := parsePluginName(req.Name)
	old, newVer, err := d.packages.Update(owner, name)
	if err != nil {
		return nil, err
	}
	return &messages.PackagesUpdateResp{Updated: old != newVer, OldVersion: old, NewVersion: newVer}, nil
}

func (d *PackagesDomain) List(_ context.Context, _ messages.PackagesListMsg) (*messages.PackagesListResp, error) {
	installed, err := d.packages.ListInstalled()
	if err != nil {
		return nil, err
	}
	infos := make([]messages.InstalledPluginInfo, len(installed))
	for i, p := range installed {
		infos[i] = messages.InstalledPluginInfo{
			Name: p.Name, Owner: p.Owner, Version: p.Version,
			BinaryPath: p.BinaryPath, InstalledAt: p.InstalledAt.Format(time.RFC3339),
		}
	}
	return &messages.PackagesListResp{Plugins: infos}, nil
}

func (d *PackagesDomain) Info(_ context.Context, req messages.PackagesInfoMsg) (*messages.PackagesInfoResp, error) {
	installed, err := d.packages.GetInstalled(req.Name)
	if err != nil {
		return nil, err
	}
	return &messages.PackagesInfoResp{Manifest: json.RawMessage(installed.Manifest)}, nil
}

func parsePluginName(fullName string) (owner, name string) {
	parts := strings.SplitN(fullName, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "brainlet", fullName // default owner
}

// kitStoreAdapter bridges KitStore (kit package) to packages.PluginStore interface.
type kitStoreAdapter struct {
	store KitStore
}

func (a *kitStoreAdapter) SaveInstalled(name, owner, version, binaryPath, manifest string, installedAt time.Time) error {
	return a.store.SaveInstalledPlugin(InstalledPlugin{
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
