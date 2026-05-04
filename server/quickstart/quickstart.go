// Package quickstart composes the batteries-included server preset.
package quickstart

import (
	"fmt"
	"path/filepath"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/modules/packages/client"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/brainlet/brainkit/server"
	"github.com/brainlet/brainkit/server/packageboot"
	storesqlite "github.com/brainlet/brainkit/stores/sqlite"
	"github.com/brainlet/brainkit/transports/embeddednats"
)

// New creates a Server with sensible defaults: EmbeddedNATS, SQLite store
// under fsRoot, HTTP gateway on :8080, and the standard command module set.
//
// Library-embedded use should call server.New with an explicit Config.
func New(namespace, fsRoot string, opts ...Option) (*server.Server, error) {
	if fsRoot == "" {
		return nil, fmt.Errorf("server/quickstart: fsRoot is required")
	}
	store, err := storesqlite.New(filepath.Join(fsRoot, "kit.db"))
	if err != nil {
		return nil, err
	}
	mods := standard.CommandSet()
	mods = append(mods, gateway.New(gateway.Config{Listen: ":8080"}))
	cfg := server.Config{
		Namespace: namespace,
		FSRoot:    fsRoot,
		Transport: embeddednats.New(),
		Store:     store,
		Modules:   mods,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return server.New(cfg)
}

// Option configures QuickStart overrides.
type Option func(*server.Config)

// WithListen overrides the HTTP gateway listen address. The default QuickStart
// wiring installs a gateway module first; this option replaces it with one
// bound to the requested address.
func WithListen(addr string) Option {
	return func(c *server.Config) {
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
func WithSecretKey(key string) Option {
	return func(c *server.Config) { c.SecretKey = key }
}

// WithPackages auto-deploys packages on Start.
func WithPackages(pkgs ...packageclient.Package) Option {
	return func(c *server.Config) { packageboot.Add(c, pkgs...) }
}

// WithExtraModules appends additional Modules to the composed set.
func WithExtraModules(mods ...bkmodule.Module) Option {
	return func(c *server.Config) { c.Modules = append(c.Modules, mods...) }
}
