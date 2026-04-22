package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DeployPackage reads a manifest, resolves entry, validates deps, bundles, and deploys.
func DeployPackage(ctx context.Context, deployer Deployer, dir string, plugins PluginChecker, secrets SecretChecker) (*Package, error) {
	// 1. Read manifest
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("package.deploy: read manifest: %w", err)
	}

	var manifest PackageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("package.deploy: parse manifest: %w", err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("package.deploy: manifest missing 'name'")
	}

	// 2. Resolve entry point
	entryPath, err := ResolveEntry(dir, manifest)
	if err != nil {
		return nil, fmt.Errorf("package.deploy: %w", err)
	}

	// 3. Validate dependencies. Callers must provide non-nil checkers;
	// use denyAll* fallbacks when the backing subsystem is absent.
	if err := ValidateDeps(manifest, plugins, secrets); err != nil {
		return nil, err
	}

	// 4. Bundle from entry point
	bundled, err := Bundle(entryPath)
	if err != nil {
		return nil, fmt.Errorf("package.deploy: bundle: %w", err)
	}

	// 5. Deploy as {name}.ts
	source := manifest.Name + ".ts"
	if err := deployer.Deploy(ctx, source, bundled); err != nil {
		return nil, fmt.Errorf("package.deploy: deploy %q: %w", source, err)
	}

	return &Package{
		Name:    manifest.Name,
		Version: manifest.Version,
		Dir:     dir,
		Source:  source,
	}, nil
}

// DeployFile deploys a single .ts file as a virtual package.
// The package name is derived from the filename (hello.ts → "hello").
func DeployFile(ctx context.Context, deployer Deployer, path string) (*Package, error) {
	filename := filepath.Base(path)
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	bundled, err := Bundle(path)
	if err != nil {
		return nil, fmt.Errorf("package.deploy: bundle %s: %w", path, err)
	}
	if len(strings.TrimSpace(bundled)) == 0 {
		return nil, fmt.Errorf("package.deploy: bundle %s: produced empty output — check for unsupported import patterns", path)
	}

	source := name + ".ts"
	if err := deployer.Deploy(ctx, source, bundled); err != nil {
		return nil, fmt.Errorf("package.deploy: deploy %q: %w", source, err)
	}

	return &Package{
		Name:    name,
		Version: "0.0.0",
		Source:  source,
	}, nil
}

// TeardownPackage removes the package's deployment.
func TeardownPackage(ctx context.Context, deployer Deployer, pkg *Package) error {
	return deployer.Teardown(ctx, pkg.Source)
}
