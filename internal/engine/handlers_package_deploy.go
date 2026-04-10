package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/brainlet/brainkit/internal/packages"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// deployerAdapter adapts the engine.Deployer interface to packages.Deployer.
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
	packages             *packages.Manager
	secretStore          secrets.SecretStore
	pluginCheckerFactory func() packages.PluginChecker

	mu       syncx.Mutex
	deployed map[string]*packages.Package
}

func newPackageDeployDomain(deployer Deployer, pkgMgr *packages.Manager, secretStore secrets.SecretStore, pluginCheckerFactory func() packages.PluginChecker) *PackageDeployDomain {
	return &PackageDeployDomain{
		deployer:             deployer,
		packages:             pkgMgr,
		secretStore:          secretStore,
		pluginCheckerFactory: pluginCheckerFactory,
		deployed:             make(map[string]*packages.Package),
	}
}

func (d *PackageDeployDomain) Deploy(ctx context.Context, req sdk.PackageDeployMsg) (*sdk.PackageDeployResp, error) {
	if req.Path == "" && len(req.Files) > 0 {
		tmpDir, err := os.MkdirTemp("", "brainkit-pkg-*")
		if err != nil {
			return nil, fmt.Errorf("package.deploy: create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		if len(req.Manifest) > 0 {
			os.WriteFile(filepath.Join(tmpDir, "manifest.json"), req.Manifest, 0644)
		}
		for name, code := range req.Files {
			filePath := filepath.Join(tmpDir, name)
			os.MkdirAll(filepath.Dir(filePath), 0755)
			os.WriteFile(filePath, []byte(code), 0644)
		}
		req.Path = tmpDir
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

	var pluginChecker packages.PluginChecker
	if d.pluginCheckerFactory != nil {
		pluginChecker = d.pluginCheckerFactory()
	}

	pkg, err := packages.DeployPackage(ctx, adapter, req.Path, pluginChecker, d.newSecretChecker())
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	return &sdk.PackageDeployResp{
		Deployed: true,
		Name:     pkg.Name,
		Version:  pkg.Version,
		Source:   pkg.Source,
	}, nil
}

func (d *PackageDeployDomain) Teardown(ctx context.Context, req sdk.PackageTeardownMsg) (*sdk.PackageTeardownResp, error) {
	d.mu.Lock()
	pkg, ok := d.deployed[req.Name]
	if !ok {
		d.mu.Unlock()
		return nil, &sdkerrors.NotFoundError{Resource: "package", Name: req.Name}
	}
	delete(d.deployed, req.Name)
	d.mu.Unlock()

	adapter := &deployerAdapter{deployer: d.deployer}
	packages.TeardownPackage(ctx, adapter, pkg)

	return &sdk.PackageTeardownResp{Removed: true}, nil
}

func (d *PackageDeployDomain) List(_ context.Context, _ sdk.PackageListDeployedMsg) (*sdk.PackageListDeployedResp, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pkgs := make([]sdk.DeployedPackageInfo, 0, len(d.deployed))
	for _, pkg := range d.deployed {
		pkgs = append(pkgs, sdk.DeployedPackageInfo{
			Name:    pkg.Name,
			Version: pkg.Version,
			Source:  pkg.Source,
			Status:  "active",
		})
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

func (d *PackageDeployDomain) newSecretChecker() packages.SecretChecker {
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

// pluginCheckerImpl checks installed and running plugins.
type pluginCheckerImpl struct {
	packages *packages.Manager
	node     *Node
}

func (c *pluginCheckerImpl) IsPluginInstalled(name string) bool {
	installed, err := c.packages.ListInstalled()
	if err != nil {
		return false
	}
	for _, p := range installed {
		if p.Name == name {
			return true
		}
	}
	return false
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

func (c *pluginCheckerImpl) InstalledVersion(name string) string {
	installed, err := c.packages.ListInstalled()
	if err != nil {
		return ""
	}
	for _, p := range installed {
		if p.Name == name {
			return p.Version
		}
	}
	return ""
}

// DeployFile deploys a single .ts file with import resolution via esbuild.
func DeployFile(ctx context.Context, k *Kernel, filePath string) ([]types.ResourceInfo, error) {
	deployer := &deployerAdapter{deployer: k}
	pkg, err := packages.DeployFile(ctx, deployer, filePath)
	if err != nil {
		return nil, err
	}
	resources, _ := k.ResourcesFrom(pkg.Source)
	return resources, nil
}
