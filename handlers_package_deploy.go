package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"github.com/brainlet/brainkit/internal/syncx"

	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/packages"
	"github.com/brainlet/brainkit/sdk/messages"
)

// kernelDeployer adapts Kernel to packages.Deployer interface.
type kernelDeployer struct {
	kernel      *Kernel
	packageName string
}

func (d *kernelDeployer) Deploy(ctx context.Context, source, code string) error {
	var opts []DeployOption
	if d.packageName != "" {
		opts = append(opts, WithPackageName(d.packageName))
	}
	_, err := d.kernel.Deploy(ctx, source, code, opts...)
	return err
}

func (d *kernelDeployer) Teardown(ctx context.Context, source string) error {
	_, err := d.kernel.Teardown(ctx, source)
	return err
}

// PackageDeployDomain handles package.deploy/teardown/list/info bus commands.
type PackageDeployDomain struct {
	kit *Kernel

	mu       syncx.Mutex
	deployed map[string]*packages.Package // name → deployed package
}

func newPackageDeployDomain(k *Kernel) *PackageDeployDomain {
	return &PackageDeployDomain{
		kit:      k,
		deployed: make(map[string]*packages.Package),
	}
}

func (d *PackageDeployDomain) Deploy(ctx context.Context, req messages.PackageDeployMsg) (*messages.PackageDeployResp, error) {
	// Inline deploy mode: write files to temp dir, then deploy from there
	if req.Path == "" && len(req.Files) > 0 {
		tmpDir, err := os.MkdirTemp("", "brainkit-pkg-*")
		if err != nil {
			return nil, fmt.Errorf("package.deploy: create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		// Write manifest
		if len(req.Manifest) > 0 {
			os.WriteFile(filepath.Join(tmpDir, "manifest.json"), req.Manifest, 0644)
		}
		// Write files
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

	// Read manifest first to get package name for persistence tagging
	manifestData, _ := os.ReadFile(filepath.Join(req.Path, "manifest.json"))
	var pkgName string
	if len(manifestData) > 0 {
		var m struct{ Name string `json:"name"` }
		json.Unmarshal(manifestData, &m)
		pkgName = m.Name
	}

	deployer := &kernelDeployer{kernel: d.kit, packageName: pkgName}

	// Build plugin/secret checkers from kernel state
	var pluginChecker packages.PluginChecker
	var secretChecker packages.SecretChecker
	if d.kit.packages != nil && d.kit.secretStore != nil {
		pluginChecker = &kernelPluginChecker{kit: d.kit}
		secretChecker = &kernelSecretChecker{kit: d.kit}
	}

	pkg, err := packages.DeployPackage(ctx, deployer, req.Path, pluginChecker, secretChecker)
	if err != nil {
		return nil, err
	}

	d.mu.Lock()
	d.deployed[pkg.Name] = pkg
	d.mu.Unlock()

	return &messages.PackageDeployResp{
		Deployed: true,
		Name:     pkg.Name,
		Version:  pkg.Version,
		Source:   pkg.Source,
	}, nil
}

func (d *PackageDeployDomain) Teardown(ctx context.Context, req messages.PackageTeardownMsg) (*messages.PackageTeardownResp, error) {
	d.mu.Lock()
	pkg, ok := d.deployed[req.Name]
	if !ok {
		d.mu.Unlock()
		return nil, &sdkerrors.NotFoundError{Resource: "package", Name: req.Name}
	}
	delete(d.deployed, req.Name)
	d.mu.Unlock()

	deployer := &kernelDeployer{kernel: d.kit}
	packages.TeardownPackage(ctx, deployer, pkg)

	return &messages.PackageTeardownResp{Removed: true}, nil
}

func (d *PackageDeployDomain) List(_ context.Context, _ messages.PackageListDeployedMsg) (*messages.PackageListDeployedResp, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	pkgs := make([]messages.DeployedPackageInfo, 0, len(d.deployed))
	for _, pkg := range d.deployed {
		pkgs = append(pkgs, messages.DeployedPackageInfo{
			Name:    pkg.Name,
			Version: pkg.Version,
			Source:  pkg.Source,
			Status:  "active",
		})
	}
	return &messages.PackageListDeployedResp{Packages: pkgs}, nil
}

func (d *PackageDeployDomain) Info(_ context.Context, req messages.PackageDeployInfoMsg) (*messages.PackageDeployInfoResp, error) {
	d.mu.Lock()
	pkg, ok := d.deployed[req.Name]
	d.mu.Unlock()
	if !ok {
		return nil, &sdkerrors.NotFoundError{Resource: "package", Name: req.Name}
	}
	return &messages.PackageDeployInfoResp{
		Name:    pkg.Name,
		Version: pkg.Version,
		Source:  pkg.Source,
	}, nil
}

// kernelPluginChecker checks installed/running plugins via the Kernel's package manager.
type kernelPluginChecker struct {
	kit *Kernel
}

func (c *kernelPluginChecker) IsPluginInstalled(name string) bool {
	installed, err := c.kit.packages.ListInstalled()
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

func (c *kernelPluginChecker) IsPluginRunning(name string) bool {
	if c.kit.node == nil {
		return false // standalone Kernel — no plugins
	}
	for _, p := range c.kit.node.ListRunningPlugins() {
		if p.Name == name {
			return true
		}
	}
	return false
}

func (c *kernelPluginChecker) InstalledVersion(name string) string {
	installed, err := c.kit.packages.ListInstalled()
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

// kernelSecretChecker checks secrets via the Kernel's secret store.
type kernelSecretChecker struct {
	kit *Kernel
}

func (c *kernelSecretChecker) HasSecret(name string) bool {
	if c.kit.secretStore == nil {
		return false
	}
	val, err := c.kit.secretStore.Get(context.Background(), name)
	return err == nil && val != ""
}

// DeployFile deploys a single .ts file with import resolution via esbuild.
// Bundles with esbuild, deploys as {name}.ts where name is derived from filename.
func DeployFile(ctx context.Context, k *Kernel, filePath string) ([]ResourceInfo, error) {
	deployer := &kernelDeployer{kernel: k}
	pkg, err := packages.DeployFile(ctx, deployer, filePath)
	if err != nil {
		return nil, err
	}
	resources, _ := k.ResourcesFrom(pkg.Source)
	return resources, nil
}
