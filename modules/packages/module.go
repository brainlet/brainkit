// Package packages owns package.deploy, package.teardown, package.list, and
// package.info as a hot-mountable Kit module.
package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	coredeploy "github.com/brainlet/brainkit/internal/deploy"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modulecap/runtime"
	_ "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/sdk/systemmsg"
)

// Module exposes package deployment commands. Construct via New and include in
// brainkit.Config.Modules when a Kit should accept package.deploy traffic.
type Module struct {
	domain *Domain
}

// New creates the packages module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "packages" }

// Dependencies reports modules that must mount before package deployment.
func (m *Module) Dependencies() []string { return []string{"jsruntime"} }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers package.* command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	deployer, err := bkmodule.RequireCapability[runtimecap.Deployer](host, bkmodule.CapabilityDeployer)
	if err != nil {
		return fmt.Errorf("packages: %w", err)
	}
	secretStore, _ := bkmodule.Capability[secrets.SecretStore](host, bkmodule.CapabilitySecretStore)
	pluginCheckerFactory, _ := bkmodule.Capability[func() bkmodule.PluginChecker](host, bkmodule.CapabilityPluginChecker)
	runtimeID, _ := bkmodule.Capability[string](host, bkmodule.CapabilityRuntimeID)
	audit, _ := bkmodule.Capability[*auditpkg.Recorder](host, bkmodule.CapabilityAuditRecorder)

	m.domain = NewDomain(deployer, secretStore, pluginCheckerFactory)
	m.domain.attachLifecycle(host.Runtime(), audit, runtimeID)
	host.Scope().Defer(func(context.Context) error {
		m.domain = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.domain.Deploy),
		bkmodule.Command(m.domain.Teardown),
		bkmodule.Command(m.domain.List),
		bkmodule.Command(m.domain.Info),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module domain. Command handles are owned by the module
// scope, so unmounting automatically unregisters them.
func (m *Module) Close() error {
	m.domain = nil
	return nil
}

// Factory is the registered ModuleFactory for packages.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the packages module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "packages",
		Status:  bkmodule.StatusStable,
		Summary: "Package deployment bus commands (package.deploy, teardown, list, info).",
		Requires: []string{
			"jsruntime",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[packagemsg.PackageDeployInfoMsg, packagemsg.PackageDeployInfoResp](),
			bkmodule.CommandMessage[packagemsg.PackageDeployMsg, packagemsg.PackageDeployResp](),
			bkmodule.CommandMessage[packagemsg.PackageListDeployedMsg, packagemsg.PackageListDeployedResp](),
			bkmodule.CommandMessage[packagemsg.PackageTeardownMsg, packagemsg.PackageTeardownResp](),
		},
		Events: []bkmodule.MessageDescriptor{
			bkmodule.EventMessage[systemmsg.KitDeployedEvent](),
			bkmodule.EventMessage[systemmsg.KitTeardownedEvent](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[runtimecap.Deployer](bkmodule.CapabilityDeployer),
		},
	}
}

func init() { bkmodule.Register("packages", Factory{}) }

type busPublisher interface {
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error)
}

// deployerAdapter adapts runtimecap.Deployer to the internal deploy package.
type deployerAdapter struct {
	deployer    runtimecap.Deployer
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

// Domain handles package.deploy/teardown/list/info bus commands.
type Domain struct {
	deployer             runtimecap.Deployer
	secretStore          secrets.SecretStore
	pluginCheckerFactory func() bkmodule.PluginChecker

	bus       busPublisher
	audit     *auditpkg.Recorder
	runtimeID string

	mu       syncx.Mutex
	deployed map[string]*coredeploy.Package
}

// NewDomain builds a package deployment command domain.
func NewDomain(deployer runtimecap.Deployer, secretStore secrets.SecretStore, pluginCheckerFactory func() bkmodule.PluginChecker) *Domain {
	return &Domain{
		deployer:             deployer,
		secretStore:          secretStore,
		pluginCheckerFactory: pluginCheckerFactory,
		deployed:             make(map[string]*coredeploy.Package),
	}
}

// attachLifecycle wires the bus, audit recorder, and runtime ID for deploy
// and teardown events.
func (d *Domain) attachLifecycle(bus busPublisher, audit *auditpkg.Recorder, runtimeID string) {
	d.bus = bus
	d.audit = audit
	d.runtimeID = runtimeID
}

func (d *Domain) emitDeployed(ctx context.Context, source string, resources []types.ResourceInfo) {
	if d.bus == nil {
		return
	}
	evt := systemmsg.KitDeployedEvent{
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

func (d *Domain) emitTeardowned(ctx context.Context, source string, removed int) {
	if d.bus == nil {
		return
	}
	evt := systemmsg.KitTeardownedEvent{
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

// Deploy handles package.deploy.
func (d *Domain) Deploy(ctx context.Context, req packagemsg.PackageDeployMsg) (*packagemsg.PackageDeployResp, error) {
	// Inline path: Files provided without a filesystem Path deploy directly.
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
		_ = json.Unmarshal(manifestData, &m)
		pkgName = m.Name
	}

	adapter := &deployerAdapter{deployer: d.deployer, packageName: pkgName}
	pkg, err := coredeploy.DeployPackage(ctx, adapter, req.Path, d.resolvePluginChecker(), d.newSecretChecker())
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	resources, _ := d.resourcesFrom(pkg.Source)
	d.emitDeployed(ctx, pkg.Source, resources)

	return &packagemsg.PackageDeployResp{
		Deployed:  true,
		Name:      pkg.Name,
		Version:   pkg.Version,
		Source:    pkg.Source,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

// resourcesFrom queries the deployer for resources from a source if possible.
func (d *Domain) resourcesFrom(source string) ([]types.ResourceInfo, error) {
	type resourcer interface {
		ResourcesFrom(source string) ([]types.ResourceInfo, error)
	}
	if r, ok := d.deployer.(resourcer); ok {
		return r.ResourcesFrom(source)
	}
	return nil, nil
}

// deployInline deploys a package from in-memory files.
func (d *Domain) deployInline(ctx context.Context, req packagemsg.PackageDeployMsg) (*packagemsg.PackageDeployResp, error) {
	var manifest struct {
		Name     string                   `json:"name"`
		Version  string                   `json:"version"`
		Entry    string                   `json:"entry"`
		Requires *coredeploy.Requirements `json:"requires,omitempty"`
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
	if _, ok := req.Files[manifest.Entry]; !ok {
		return nil, &sdkerrors.ValidationError{Field: "files", Message: fmt.Sprintf("entry %q not found", manifest.Entry)}
	}

	if manifest.Requires != nil {
		pm := coredeploy.PackageManifest{
			Name: manifest.Name, Version: manifest.Version, Entry: manifest.Entry, Requires: manifest.Requires,
		}
		if err := coredeploy.ValidateDeps(pm, d.resolvePluginChecker(), d.newSecretChecker()); err != nil {
			return nil, err
		}
	}

	// Bundle through esbuild so relative imports resolve against the sibling
	// files the client sent. Standalone .ts files still bundle to strip
	// TypeScript annotations.
	code, err := coredeploy.BundleInMemory(req.Files, manifest.Entry)
	if err != nil {
		return nil, &sdkerrors.DeployError{
			Source: manifest.Entry,
			Phase:  "transpile",
			Cause:  fmt.Errorf("bundle: %w", err),
		}
	}

	trimmed := strings.TrimSpace(code)
	if len(trimmed) == 0 || trimmed == ";" {
		return nil, &sdkerrors.DeployError{
			Source: manifest.Entry,
			Phase:  "bundle",
			Cause:  fmt.Errorf("bundle produced empty output; check for unsupported import patterns"),
		}
	}

	source := manifest.Name + filepath.Ext(manifest.Entry)
	var opts []types.DeployOption
	if manifest.Name != "" {
		opts = append(opts, types.WithPackageName(manifest.Name))
	}
	resources, err := d.deployer.Deploy(ctx, source, code, opts...)
	if err != nil {
		return nil, err
	}

	pkg := &coredeploy.Package{
		Name:    manifest.Name,
		Version: manifest.Version,
		Source:  source,
	}
	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	d.emitDeployed(ctx, source, resources)

	return &packagemsg.PackageDeployResp{
		Deployed:  true,
		Name:      pkg.Name,
		Version:   pkg.Version,
		Source:    pkg.Source,
		Resources: resourceInfosToMessages(resources),
	}, nil
}

// Teardown handles package.teardown.
func (d *Domain) Teardown(ctx context.Context, req packagemsg.PackageTeardownMsg) (*packagemsg.PackageTeardownResp, error) {
	d.mu.Lock()
	pkg, tracked := d.deployed[req.Name]
	if tracked {
		delete(d.deployed, req.Name)
	}
	d.mu.Unlock()

	source := req.Name + ".ts"
	if tracked {
		source = pkg.Source
	}

	removed, _ := d.deployer.Teardown(ctx, source)
	if removed > 0 || tracked {
		d.emitTeardowned(ctx, source, removed)
	}
	return &packagemsg.PackageTeardownResp{Removed: removed > 0 || tracked}, nil
}

// List handles package.list.
func (d *Domain) List(_ context.Context, _ packagemsg.PackageListDeployedMsg) (*packagemsg.PackageListDeployedResp, error) {
	deployments := d.deployer.ListDeployments()

	d.mu.Lock()
	meta := make(map[string]*coredeploy.Package, len(d.deployed))
	for _, p := range d.deployed {
		meta[p.Source] = p
	}
	d.mu.Unlock()

	pkgs := make([]packagemsg.DeployedPackageInfo, 0, len(deployments))
	for _, dep := range deployments {
		info := packagemsg.DeployedPackageInfo{Source: dep.Source, Status: "active"}
		if p, ok := meta[dep.Source]; ok {
			info.Name = p.Name
			info.Version = p.Version
		}
		pkgs = append(pkgs, info)
	}
	return &packagemsg.PackageListDeployedResp{Packages: pkgs}, nil
}

// Info handles package.info.
func (d *Domain) Info(_ context.Context, req packagemsg.PackageDeployInfoMsg) (*packagemsg.PackageDeployInfoResp, error) {
	d.mu.Lock()
	pkg, ok := d.deployed[req.Name]
	d.mu.Unlock()
	if !ok {
		return nil, &sdkerrors.NotFoundError{Resource: "package", Name: req.Name}
	}
	return &packagemsg.PackageDeployInfoResp{
		Name:    pkg.Name,
		Version: pkg.Version,
		Source:  pkg.Source,
	}, nil
}

func (d *Domain) newSecretChecker() coredeploy.SecretChecker {
	if d.secretStore == nil {
		return denyAllSecretChecker{}
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

// denyAllSecretChecker answers no to every secret-presence query. Used as the
// fallback when no secret store is configured.
type denyAllSecretChecker struct{}

func (denyAllSecretChecker) HasSecret(string) bool { return false }

// denyAllPluginChecker answers no to every plugin-presence query. Used as the
// fallback when no plugins module has registered a real checker.
type denyAllPluginChecker struct{}

func (denyAllPluginChecker) IsPluginRunning(string) bool { return false }

func (d *Domain) resolvePluginChecker() coredeploy.PluginChecker {
	if d.pluginCheckerFactory == nil {
		return denyAllPluginChecker{}
	}
	pc := d.pluginCheckerFactory()
	if pc == nil {
		return denyAllPluginChecker{}
	}
	return pc
}

func resourceInfosToMessages(resources []types.ResourceInfo) []sdk.ResourceInfo {
	out := make([]sdk.ResourceInfo, 0, len(resources))
	for _, resource := range resources {
		out = append(out, sdk.ResourceInfo{
			Type:      resource.Type,
			ID:        resource.ID,
			Name:      resource.Name,
			Source:    resource.Source,
			CreatedAt: resource.CreatedAt,
		})
	}
	return out
}
