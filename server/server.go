// Package server composes a brainkit.Kit with a YAML-driven,
// registry-based module set behind a single lifecycle. Callers embed
// server in their binary or run it under cmd/brainkit.
//
// Module selection is declarative: server has no hard-coded knowledge
// of individual modules — it walks the brainkit module registry,
// calling each factory registered via `brainkit.RegisterModule`. The
// blank imports below wire the standard set into every binary that
// imports this package; custom binaries can blank-import additional
// third-party modules to extend the catalog.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/brainlet/brainkit"

	// Blank-import the standard module set. Each package's init()
	// registers a factory into the global brainkit module registry,
	// making `modules.<name>:` in YAML work out of the box.
	_ "github.com/brainlet/brainkit/modules/audit"
	_ "github.com/brainlet/brainkit/modules/discovery"
	_ "github.com/brainlet/brainkit/modules/gateway"
	_ "github.com/brainlet/brainkit/modules/harness"
	_ "github.com/brainlet/brainkit/modules/mcp"
	_ "github.com/brainlet/brainkit/modules/plugins"
	_ "github.com/brainlet/brainkit/modules/probes"
	_ "github.com/brainlet/brainkit/modules/schedules"
	_ "github.com/brainlet/brainkit/modules/topology"
	_ "github.com/brainlet/brainkit/modules/tracing"
	_ "github.com/brainlet/brainkit/modules/workflow"
)

// Config configures a Server. Required: Namespace, Transport, FSRoot.
// At least one module named "gateway" must appear in Modules — server
// mode exists to serve HTTP traffic.
//
// The YAML-driven path (LoadConfig) populates Modules via the registry.
// Programmatic callers can also append modules directly — both sources
// are merged in the order Modules is written.
type Config struct {
	// Namespace is the Kit's bus namespace. Required.
	Namespace string

	// Transport is the bus backend. Required — server mode rejects
	// Memory(): nothing plugin- or cross-kit would work on an
	// in-process channel.
	Transport brainkit.TransportConfig

	// FSRoot is the sandbox root for deployed .ts code + the default
	// store location when KitStorePath is empty. Required.
	FSRoot string

	// KitStorePath is the SQLite file backing deployments, schedules,
	// installed plugins, and secrets. Empty = <FSRoot>/kit.db.
	KitStorePath string

	// SecretKey seeds the encrypted secret store. Required in
	// production; empty logs a warning and stores secrets in cleartext
	// on top of the KitStore.
	SecretKey string

	// Providers, Storages, Vectors pass through to brainkit.Config
	// verbatim so callers don't have to reach around Server.
	Providers []brainkit.ProviderConfig
	Storages  map[string]brainkit.StorageConfig
	Vectors   map[string]brainkit.VectorConfig

	// Modules is the final list of modules to install. LoadConfig
	// populates it from `modules:` YAML via the registry;
	// programmatic callers can append.
	Modules []brainkit.Module

	// Packages are deployed after the Kit boots.
	Packages []brainkit.Package
}

// Server is a composed runtime — Kit + YAML-driven module set,
// managed as a single lifecycle.
type Server struct {
	cfg Config
	kit *brainkit.Kit
}

// New composes a Kit with the Modules slice.
func New(cfg Config) (*Server, error) {
	if err := validate(cfg); err != nil {
		return nil, err
	}

	storePath := cfg.KitStorePath
	if storePath == "" {
		storePath = filepath.Join(cfg.FSRoot, "kit.db")
	}
	store, err := brainkit.NewSQLiteStore(storePath)
	if err != nil {
		return nil, fmt.Errorf("server: open kit store %q: %w", storePath, err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: cfg.Namespace,
		CallerID:  cfg.Namespace,
		Transport: cfg.Transport,
		FSRoot:    cfg.FSRoot,
		Store:     store,
		SecretKey: cfg.SecretKey,
		Providers: cfg.Providers,
		Storages:  cfg.Storages,
		Vectors:   cfg.Vectors,
		Modules:   cfg.Modules,
	})
	if err != nil {
		return nil, fmt.Errorf("server: build kit: %w", err)
	}

	return &Server{cfg: cfg, kit: kit}, nil
}

// Start auto-deploys packages then blocks until ctx cancels or the
// process receives SIGINT/SIGTERM. The HTTP gateway is already
// listening at this point (gateway module's Init starts it); this
// method is the long-running supervisor loop.
func (s *Server) Start(ctx context.Context) error {
	for _, pkg := range s.cfg.Packages {
		if _, err := s.kit.Deploy(ctx, pkg); err != nil {
			return fmt.Errorf("server: deploy package: %w", err)
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
		if m != nil && m.Name() == "gateway" {
			gatewayPresent = true
			break
		}
	}
	if !gatewayPresent {
		return fmt.Errorf("server: a gateway module is required (add `modules.gateway:` to the YAML or append `gateway.New(...)` to Config.Modules)")
	}
	return nil
}
