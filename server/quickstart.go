package server

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/brainlet/brainkit/transports"
)

// QuickStart creates a Server with sensible defaults — EmbeddedNATS,
// SQLite store under fsRoot, HTTP gateway on :8080. Intended for
// "just run it" scenarios (demos, dev shells); library-embedded use
// should call New with an explicit Config.
//
// Additional modules (audit, tracing, probes, …) can be wired via
// WithExtraModules or by switching to the YAML-driven path.
func QuickStart(namespace, fsRoot string, opts ...QuickStartOption) (*Server, error) {
	mods := standard.CommandSet()
	mods = append(mods, gateway.New(gateway.Config{Listen: ":8080"}))
	cfg := Config{
		Namespace: namespace,
		FSRoot:    fsRoot,
		Transport: transports.EmbeddedNATS(),
		Modules:   mods,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return New(cfg)
}

// QuickStartOption configures QuickStart overrides.
type QuickStartOption func(*Config)

// WithListen overrides the HTTP gateway listen address. The default
// QuickStart wiring installs a gateway module first; this option
// replaces it with one bound to the requested address.
func WithListen(addr string) QuickStartOption {
	return func(c *Config) {
		for i, m := range c.Modules {
			if m != nil && m.ID() == "gateway" {
				c.Modules[i] = gateway.New(gateway.Config{Listen: addr})
				return
			}
		}
		c.Modules = append(c.Modules, gateway.New(gateway.Config{Listen: addr}))
	}
}

// WithSecretKey sets the encrypted secret store key.
func WithSecretKey(key string) QuickStartOption {
	return func(c *Config) { c.SecretKey = key }
}

// WithPackages auto-deploys packages on Start.
func WithPackages(pkgs ...packages.Package) QuickStartOption {
	return func(c *Config) { c.Packages = append(c.Packages, pkgs...) }
}

// WithExtraModules appends additional Modules to the composed set.
func WithExtraModules(mods ...brainkit.Module) QuickStartOption {
	return func(c *Config) { c.Modules = append(c.Modules, mods...) }
}
