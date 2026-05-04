// Package server composes a brainkit.Kit with an explicit module set behind a
// single lifecycle. Callers embed server in their binary or run it under
// cmd/brainkit.
//
// Module selection is declarative: server has no hard-coded knowledge
// of individual modules — it walks the brainkit module registry,
// calling each factory registered through the module package. Binaries
// that want the standard registry should import
// github.com/brainlet/brainkit/server/standard for side effects. Custom
// binaries can blank-import only the modules they want.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
)

// Config configures a Server. Required: Namespace, Transport, FSRoot.
// At least one module named "gateway" must appear in Modules — server
// mode exists to serve HTTP traffic.
//
// The YAML-driven path in package server/configfile populates Modules via the registry.
// Programmatic callers can also append modules directly — both sources
// are merged in the order Modules is written.
type Config struct {
	// Namespace is the Kit's bus namespace. Required.
	Namespace string

	// Transport is the bus backend. Required — server mode rejects
	// Memory(): nothing plugin- or cross-kit would work on an
	// in-process channel.
	Transport brainkit.TransportConfig

	// FSRoot is the sandbox root for deployed .ts code. Required.
	FSRoot string

	// Store is the Kit persistence backend. Optional for lightweight server
	// embeddings; configfile and quickstart install SQLite stores for the
	// batteries-included paths.
	Store brainkit.KitStore

	// SecretKey seeds the encrypted secret store. Required in
	// production; empty logs a warning and stores secrets in cleartext
	// on top of the KitStore.
	SecretKey string

	// Providers, Storages, Vectors pass through to brainkit.Config
	// verbatim so callers don't have to reach around Server.
	Providers []brainkit.ProviderConfig
	Storages  map[string]brainkit.StorageConfig
	Vectors   map[string]brainkit.VectorConfig

	// Modules is the final list of modules to install. Package
	// server/configfile populates it from `modules:` YAML via the registry;
	// programmatic callers can append.
	Modules []bkmodule.Module

	// OnStart hooks run before the supervisor blocks on ctx/signal
	// cancellation. Optional packages such as server/packageboot can use this
	// without making core server import their implementation dependencies.
	OnStart []StartHook
}

// StartHook runs after the Kit has booted and before Server.Start begins
// waiting for ctx cancellation or SIGINT/SIGTERM.
type StartHook func(context.Context, *brainkit.Kit) error

// Server is a composed runtime — Kit plus explicit modules managed as a single
// lifecycle.
type Server struct {
	cfg Config
	kit *brainkit.Kit
}

// New composes a Kit with the Modules slice.
func New(cfg Config) (*Server, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: cfg.Namespace,
		CallerID:  cfg.Namespace,
		Transport: cfg.Transport,
		FSRoot:    cfg.FSRoot,
		Store:     cfg.Store,
		SecretKey: cfg.SecretKey,
		Providers: cfg.Providers,
		Storages:  cfg.Storages,
		Vectors:   cfg.Vectors,
		Modules:   cfg.Modules,
	})
	if err != nil {
		if cfg.Store != nil {
			_ = cfg.Store.Close()
		}
		return nil, fmt.Errorf("server: build kit: %w", err)
	}

	return &Server{cfg: cfg, kit: kit}, nil
}

// Start auto-deploys packages then blocks until ctx cancels or the
// process receives SIGINT/SIGTERM. The HTTP gateway is already
// listening at this point (gateway module's Init starts it); this
// method is the long-running supervisor loop.
func (s *Server) Start(ctx context.Context) error {
	for _, hook := range s.cfg.OnStart {
		if hook == nil {
			continue
		}
		if err := hook(ctx, s.kit); err != nil {
			return err
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
	case <-sigCh:
	}
	return nil
}

// Stop gracefully drains the Kit. Use Close for an immediate shutdown.
func (s *Server) Stop(ctx context.Context) error { return s.kit.Shutdown(ctx) }

// Close immediately releases Kit resources.
func (s *Server) Close() error { return s.kit.Close() }

// Kit exposes the underlying runtime for callers that need the full
// brainkit surface (accessors, bus.Call, etc.).
func (s *Server) Kit() *brainkit.Kit { return s.kit }

func validate(cfg Config) error {
	if cfg.Namespace == "" {
		return fmt.Errorf("server: Namespace is required")
	}
	if cfg.Transport == (brainkit.TransportConfig{}) {
		return fmt.Errorf("server: Transport is required")
	}
	if cfg.FSRoot == "" {
		return fmt.Errorf("server: FSRoot is required")
	}
	gatewayPresent := false
	for _, m := range cfg.Modules {
		if m != nil && m.ID() == "gateway" {
			gatewayPresent = true
			break
		}
	}
	if !gatewayPresent {
		return fmt.Errorf("server: a gateway module is required (add `modules.gateway:` to the YAML or append `gateway.New(...)` to Config.Modules)")
	}
	return nil
}

func hasModule(mods []bkmodule.Module, id string) bool {
	for _, m := range mods {
		if m != nil && m.ID() == id {
			return true
		}
	}
	return false
}
