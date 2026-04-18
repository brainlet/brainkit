package server

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
)

// QuickStart creates a Server with sensible defaults — EmbeddedNATS,
// SQLite store under fsRoot, HTTP gateway on :8080, tracing + probes
// + audit enabled. Intended for "just run it" scenarios (demos, dev
// shells); library-embedded use should call New with an explicit
// Config.
func QuickStart(namespace, fsRoot string, opts ...QuickStartOption) (*Server, error) {
	cfg := Config{
		Namespace: namespace,
		FSRoot:    fsRoot,
		Transport: brainkit.EmbeddedNATS(),
		Gateway:   gateway.Config{Listen: ":8080"},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return New(cfg)
}

// QuickStartOption configures QuickStart overrides.
type QuickStartOption func(*Config)

// WithListen overrides the HTTP gateway listen address.
func WithListen(addr string) QuickStartOption {
	return func(c *Config) { c.Gateway.Listen = addr }
}

// WithSecretKey sets the encrypted secret store key.
func WithSecretKey(key string) QuickStartOption {
	return func(c *Config) { c.SecretKey = key }
}

// WithPackages auto-deploys packages on Start.
func WithPackages(pkgs ...brainkit.Package) QuickStartOption {
	return func(c *Config) { c.Packages = append(c.Packages, pkgs...) }
}

// WithExtraModules appends additional Modules to the composed set.
func WithExtraModules(mods ...brainkit.Module) QuickStartOption {
	return func(c *Config) { c.Extra = append(c.Extra, mods...) }
}
