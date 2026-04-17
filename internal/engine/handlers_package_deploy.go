package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/deploy"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// deployerAdapter adapts the engine.Deployer interface to deploy.Deployer.
type deployerAdapter struct {
	deployer    Deployer
	packageName string
}

func (d *deployerAdapter) Deploy(ctx context.Context, source, code string) error {
	var opts []types.DeployOption
	if d.packageName != "" {
		opts = append(opts, types.WithPackageName(d.packageName))
	}
	_, err := d.deployer.Deploy(ctx, source, code, opts...)
	return err
}

func (d *deployerAdapter) Teardown(ctx context.Context, source string) error {
	_, err := d.deployer.Teardown(ctx, source)
	return err
}

// PackageDeployDomain handles package.deploy/teardown/list/info bus commands.
type PackageDeployDomain struct {
	deployer             Deployer
	secretStore          secrets.SecretStore
	pluginCheckerFactory func() deploy.PluginChecker

	bus       BusPublisher
	audit     *auditpkg.Recorder
	runtimeID string

	mu       syncx.Mutex
	deployed map[string]*deploy.Package
}

func newPackageDeployDomain(deployer Deployer, secretStore secrets.SecretStore, pluginCheckerFactory func() deploy.PluginChecker) *PackageDeployDomain {
	return &PackageDeployDomain{
		deployer:             deployer,
		secretStore:          secretStore,
		pluginCheckerFactory: pluginCheckerFactory,
		deployed:             make(map[string]*deploy.Package),
	}
}

// attachLifecycle wires the bus + audit + runtimeID for emitting deploy/teardown
// events. Called after the kernel's remote bus and audit recorder are initialized.
func (d *PackageDeployDomain) attachLifecycle(bus BusPublisher, audit *auditpkg.Recorder, runtimeID string) {
	d.bus = bus
	d.audit = audit
	d.runtimeID = runtimeID
}

func (d *PackageDeployDomain) emitDeployed(ctx context.Context, source string, resources []types.ResourceInfo) {
	if d.bus == nil {
		return
	}
	evt := sdk.KitDeployedEvent{
		Source:    source,
		RuntimeID: d.runtimeID,
		Resources: resourceInfosToMessages(resources),
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}
	_, _ = d.bus.PublishRaw(ctx, evt.BusTopic(), payload)
	if d.audit != nil {
		d.audit.Deployed(source, len(resources))
	}
}

func (d *PackageDeployDomain) emitTeardowned(ctx context.Context, source string, removed int) {
	if d.bus == nil {
		return
	}
	evt := sdk.KitTeardownedEvent{
		Source:    source,
		RuntimeID: d.runtimeID,
		Removed:   removed,
	}
	payload, err := json.Marshal(evt)
	if err != nil {
		return
	}
	_, _ = d.bus.PublishRaw(ctx, evt.BusTopic(), payload)
	if d.audit != nil {
		d.audit.Teardown(source)
	}
}

func (d *PackageDeployDomain) Deploy(ctx context.Context, req sdk.PackageDeployMsg) (*sdk.PackageDeployResp, error) {
	// Inline path: Files provided without a filesystem Path → deploy directly, no bundling.
	if req.Path == "" && len(req.Files) > 0 {
		return d.deployInline(ctx, req)
	}

	if req.Path == "" {
		return nil, &sdkerrors.ValidationError{Field: "path", Message: "path or files is required"}
	}

	manifestData, _ := os.ReadFile(filepath.Join(req.Path, "manifest.json"))
	var pkgName string
	if len(manifestData) > 0 {
		var m struct {
			Name string `json:"name"`
		}
		json.Unmarshal(manifestData, &m)
		pkgName = m.Name
	}

	adapter := &deployerAdapter{deployer: d.deployer, packageName: pkgName}

	var pluginChecker deploy.PluginChecker
	if d.pluginCheckerFactory != nil {
		pluginChecker = d.pluginCheckerFactory()
	}

	pkg, err := deploy.DeployPackage(ctx, adapter, req.Path, pluginChecker, d.newSecretChecker())
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	resources, _ := d.resourcesFrom(pkg.Source)
	d.emitDeployed(ctx, pkg.Source, resources)

	return &sdk.PackageDeployResp{
		Deployed:  true,
		Name:      pkg.Name,
		Version:   pkg.Version,
		Source:    pkg.Source,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

// resourcesFrom queries the deployer for resources from a source if possible.
func (d *PackageDeployDomain) resourcesFrom(source string) ([]types.ResourceInfo, error) {
	type resourcer interface {
		ResourcesFrom(source string) ([]types.ResourceInfo, error)
	}
	if r, ok := d.deployer.(resourcer); ok {
		return r.ResourcesFrom(source)
	}
	return nil, nil
}

// deployInline deploys a single-file package without esbuild bundling.
// Manifest provides {name, entry, version}; Files[entry] is the code.
func (d *PackageDeployDomain) deployInline(ctx context.Context, req sdk.PackageDeployMsg) (*sdk.PackageDeployResp, error) {
	var manifest struct {
		Name     string               `json:"name"`
		Version  string               `json:"version"`
		Entry    string               `json:"entry"`
		Requires *deploy.Requirements `json:"requires,omitempty"`
	}
	if len(req.Manifest) > 0 {
		if err := json.Unmarshal(req.Manifest, &manifest); err != nil {
			return nil, &sdkerrors.ValidationError{Field: "manifest", Message: err.Error()}
		}
	}
	if manifest.Name == "" {
		return nil, &sdkerrors.ValidationError{Field: "manifest.name", Message: "is required"}
	}
	if manifest.Entry == "" {
		return nil, &sdkerrors.ValidationError{Field: "manifest.entry", Message: "is required"}
	}
	code, ok := req.Files[manifest.Entry]
	if !ok {
		return nil, &sdkerrors.ValidationError{Field: "files", Message: fmt.Sprintf("entry %q not found", manifest.Entry)}
	}

	if manifest.Requires != nil && d.pluginCheckerFactory != nil {
		pc := d.pluginCheckerFactory()
		pm := deploy.PackageManifest{
			Name: manifest.Name, Version: manifest.Version, Entry: manifest.Entry, Requires: manifest.Requires,
		}
		if err := deploy.ValidateDeps(pm, pc, d.newSecretChecker()); err != nil {
			return nil, err
		}
	}

	source := manifest.Entry
	var opts []types.DeployOption
	if manifest.Name != "" {
		opts = append(opts, types.WithPackageName(manifest.Name))
	}
	resources, err := d.deployer.Deploy(ctx, source, code, opts...)
	if err != nil {
		return nil, err
	}

	pkg := &deploy.Package{
		Name:    manifest.Name,
		Version: manifest.Version,
		Source:  source,
	}
	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	d.emitDeployed(ctx, source, resources)

	return &sdk.PackageDeployResp{
		Deployed:  true,
		Name:      pkg.Name,
		Version:   pkg.Version,
		Source:    pkg.Source,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

func (d *PackageDeployDomain) Teardown(ctx context.Context, req sdk.PackageTeardownMsg) (*sdk.PackageTeardownResp, error) {
	d.mu.Lock()
	pkg, tracked := d.deployed[req.Name]
	if tracked {
		delete(d.deployed, req.Name)
	}
	d.mu.Unlock()

	// Resolve the runtime source: tracked package source if known, else derive
	// from the name (name+".ts"). Teardown is idempotent — missing sources are
	// a no-op, matching the prior kit.teardown semantics.
	source := req.Name + ".ts"
	if tracked {
		source = pkg.Source
	}

	removed := 0
	if t, ok := d.deployer.(interface {
		Teardown(context.Context, string) (int, error)
	}); ok {
		removed, _ = t.Teardown(ctx, source)
	} else if tracked {
		adapter := &deployerAdapter{deployer: d.deployer}
		deploy.TeardownPackage(ctx, adapter, pkg)
	}

	if removed > 0 || tracked {
		d.emitTeardowned(ctx, source, removed)
	}
	return &sdk.PackageTeardownResp{Removed: removed > 0 || tracked}, nil
}

func (d *PackageDeployDomain) List(_ context.Context, _ sdk.PackageListDeployedMsg) (*sdk.PackageListDeployedResp, error) {
	// Read from the authoritative deploymentMgr so restored-from-store deployments
	// (which bypass PackageDeployDomain) are also reported. Enrich with package
	// metadata from the deployed map when available.
	type lister interface {
		ListDeployments() []deploymentInfo
	}
	var deployments []deploymentInfo
	if l, ok := d.deployer.(lister); ok {
		deployments = l.ListDeployments()
	}

	d.mu.Lock()
	meta := make(map[string]*deploy.Package, len(d.deployed))
	for _, p := range d.deployed {
		meta[p.Source] = p
	}
	d.mu.Unlock()

	pkgs := make([]sdk.DeployedPackageInfo, 0, len(deployments))
	for _, dep := range deployments {
		info := sdk.DeployedPackageInfo{Source: dep.Source, Status: "active"}
		if p, ok := meta[dep.Source]; ok {
			info.Name = p.Name
			info.Version = p.Version
		}
		pkgs = append(pkgs, info)
	}
	return &sdk.PackageListDeployedResp{Packages: pkgs}, nil
}

func (d *PackageDeployDomain) Info(_ context.Context, req sdk.PackageDeployInfoMsg) (*sdk.PackageDeployInfoResp, error) {
	d.mu.Lock()
	pkg, ok := d.deployed[req.Name]
	d.mu.Unlock()
	if !ok {
		return nil, &sdkerrors.NotFoundError{Resource: "package", Name: req.Name}
	}
	return &sdk.PackageDeployInfoResp{
		Name:    pkg.Name,
		Version: pkg.Version,
		Source:  pkg.Source,
	}, nil
}

func (d *PackageDeployDomain) newSecretChecker() deploy.SecretChecker {
	if d.secretStore == nil {
		return nil
	}
	return &domainSecretChecker{store: d.secretStore}
}

type domainSecretChecker struct {
	store secrets.SecretStore
}

func (c *domainSecretChecker) HasSecret(name string) bool {
	val, err := c.store.Get(context.Background(), name)
	return err == nil && val != ""
}

// pluginCheckerImpl checks running plugins.
type pluginCheckerImpl struct {
	node *Node
}

func (c *pluginCheckerImpl) IsPluginRunning(name string) bool {
	if c.node == nil {
		return false
	}
	for _, p := range c.node.ListRunningPlugins() {
		if p.Name == name {
			return true
		}
	}
	return false
}

// DeployFile deploys a single .ts file with import resolution via esbuild.
func DeployFile(ctx context.Context, k *Kernel, filePath string) ([]types.ResourceInfo, error) {
	deployer := &deployerAdapter{deployer: k}
	pkg, err := deploy.DeployFile(ctx, deployer, filePath)
	if err != nil {
		return nil, err
	}
	resources, _ := k.ResourcesFrom(pkg.Source)
	return resources, nil
}
