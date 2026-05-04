package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// PluginChecker checks whether a package-required plugin is available.
type PluginChecker interface {
	IsPluginRunning(name string) bool
}

// SecretChecker checks whether a package-required secret is available.
type SecretChecker interface {
	HasSecret(name string) bool
}

// BuildRequest is the package source payload carried by package.deploy.
type BuildRequest struct {
	Path     string
	Manifest json.RawMessage
	Files    map[string]string
}

// BuiltPackage is the normalized JavaScript artifact produced before runtime
// handoff.
type BuiltPackage struct {
	Name    string
	Version string
	Source  string
	Code    string
}

// PackageBuilder prepares package.deploy inputs for the runtime. Implementations
// own manifest parsing, dependency validation, TypeScript/file graph bundling,
// and artifact normalization.
type PackageBuilder interface {
	BuildPackage(ctx context.Context, req BuildRequest, plugins PluginChecker, secrets SecretChecker) (BuiltPackage, error)
}

// Option configures the packages module.
type Option func(*Module)

// WithPackageBuilder sets the package builder used by this module instance.
func WithPackageBuilder(builder PackageBuilder) Option {
	return func(m *Module) {
		m.builder = builder
	}
}

var packageBuilders = struct {
	sync.RWMutex
	builder PackageBuilder
}{}

// RegisterPackageBuilder registers the process-default package builder. Passing
// nil removes the registration.
func RegisterPackageBuilder(builder PackageBuilder) {
	packageBuilders.Lock()
	defer packageBuilders.Unlock()
	packageBuilders.builder = builder
}

func currentPackageBuilder() PackageBuilder {
	packageBuilders.RLock()
	defer packageBuilders.RUnlock()
	return packageBuilders.builder
}

func missingPackageBuilderError() error {
	return fmt.Errorf("packages: package builder is not registered (import github.com/brainlet/brainkit/modules/packages/bundlers/esbuild or pass packages.WithPackageBuilder(...))")
}
