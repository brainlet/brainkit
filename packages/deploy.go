package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Package describes a deployed package and its services.
type Package struct {
	Name     string   `json:"name"`
	Version  string   `json:"version"`
	Dir      string   `json:"dir"`
	Services []string `json:"services"` // deployed service source names
}

// Deployer handles package deployment via a Kernel.
// Abstracted to avoid import cycles with the kit package.
type Deployer interface {
	Deploy(ctx context.Context, source, code string) error
	Teardown(ctx context.Context, source string) error
}

// DeployPackage reads a manifest, validates dependencies, bundles each service
// with esbuild, and deploys them as individual .ts Compartments.
func DeployPackage(ctx context.Context, deployer Deployer, dir string, plugins PluginChecker, secrets SecretChecker) (*Package, error) {
	// 1. Read manifest
	manifestPath := filepath.Join(dir, "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("package.deploy: read manifest: %w", err)
	}

	var manifest PackageManifestV2
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("package.deploy: parse manifest: %w", err)
	}

	if manifest.Name == "" {
		return nil, fmt.Errorf("package.deploy: manifest missing 'name'")
	}
	if len(manifest.Services) == 0 {
		return nil, fmt.Errorf("package.deploy: manifest has no services")
	}

	// 2. Validate dependencies
	if plugins != nil && secrets != nil {
		if err := ValidateDeps(manifest, plugins, secrets); err != nil {
			return nil, err
		}
	}

	// 3. Bundle and deploy each service
	var deployed []string
	for name, svc := range manifest.Services {
		entryPath := filepath.Join(dir, svc.Entry)
		if _, err := os.Stat(entryPath); err != nil {
			// Teardown already-deployed services on failure
			for _, d := range deployed {
				deployer.Teardown(ctx, d)
			}
			return nil, fmt.Errorf("package.deploy: service %q entry %q not found: %w", name, svc.Entry, err)
		}

		bundled, err := Bundle(entryPath)
		if err != nil {
			for _, d := range deployed {
				deployer.Teardown(ctx, d)
			}
			return nil, fmt.Errorf("package.deploy: bundle service %q: %w", name, err)
		}

		sourceName := manifest.Name + "/" + name + ".ts"
		if err := deployer.Deploy(ctx, sourceName, bundled); err != nil {
			for _, d := range deployed {
				deployer.Teardown(ctx, d)
			}
			return nil, fmt.Errorf("package.deploy: deploy service %q: %w", name, err)
		}
		deployed = append(deployed, sourceName)
	}

	return &Package{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Dir:      dir,
		Services: deployed,
	}, nil
}

// TeardownPackage removes all services from a deployed package.
func TeardownPackage(ctx context.Context, deployer Deployer, pkg *Package) error {
	var firstErr error
	for _, svc := range pkg.Services {
		if err := deployer.Teardown(ctx, svc); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
