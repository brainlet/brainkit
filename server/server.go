// Package server composes a brainkit.Kit with the standard
// service-mode module set — gateway, probes, tracing, audit, and
// optional plugins / discovery / topology — behind a single
// lifecycle. Callers embed server in their binary or run it under
// cmd/brainkit.
package server

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/audit"
	auditstores "github.com/brainlet/brainkit/modules/audit/stores"
	"github.com/brainlet/brainkit/modules/gateway"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/modules/probes"
	"github.com/brainlet/brainkit/modules/tracing"
)

// Config configures a Server. All required fields must be set; New
// returns an error otherwise.
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

	// Gateway configures the HTTP gateway. Required — server mode
	// exists to serve HTTP traffic.
	Gateway gateway.Config

	// Providers, Storages, Vectors pass through to brainkit.Config
	// verbatim so callers don't have to reach around Server.
	Providers []brainkit.ProviderConfig
	Storages  map[string]brainkit.StorageConfig
	Vectors   map[string]brainkit.VectorConfig

	// Plugins, when non-empty, wires the plugins module.
	Plugins []brainkit.PluginConfig

	// Audit configures the audit store. Nil = SQLite at <FSRoot>/audit.db.
	// Set to an explicit AuditStore pointer to override.
	Audit *AuditConfig

	// Tracing toggles the tracing module. Defaults to on.
	Tracing *bool

	// Probes toggles the health probes module. Defaults to on.
	Probes *bool

	// Packages are deployed after the Kit boots.
	Packages []brainkit.Package

	// Extra lets callers append additional Modules that the server
	// composition doesn't otherwise know about.
	Extra []brainkit.Module
}

// AuditConfig selects the audit store backing. Callers typically
// leave Path empty to use the default <FSRoot>/audit.db.
type AuditConfig struct {
	Path    string
	Verbose bool
}

// Server is a composed runtime — Kit + standard modules + HTTP
// gateway, managed as a single lifecycle.
type Server struct {
	cfg Config
	kit *brainkit.Kit
	gw  *gateway.Gateway
}

// New composes a Kit with the standard module set.
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

	modules := make([]brainkit.Module, 0, 8)

	// Gateway — required.
	gw := gateway.New(cfg.Gateway)
	modules = append(modules, gw)

	// Tracing + Probes on by default.
	if cfg.Tracing == nil || *cfg.Tracing {
		modules = append(modules, tracing.New(tracing.Config{}))
	}
	if cfg.Probes == nil || *cfg.Probes {
		modules = append(modules, probes.New(probes.Config{}))
	}

	// Audit — SQLite by default at <FSRoot>/audit.db.
	auditStore, err := resolveAuditStore(cfg)
	if err != nil {
		return nil, err
	}
	if auditStore != nil {
		modules = append(modules, audit.NewModule(audit.Config{
			Store:   auditStore,
			Verbose: cfg.Audit != nil && cfg.Audit.Verbose,
		}))
	}

	// Plugins — only when configured.
	if len(cfg.Plugins) > 0 {
		modules = append(modules, pluginsmod.NewModule(pluginsmod.Config{
			Plugins: cfg.Plugins,
			Store:   store,
		}))
	}

	// Caller-supplied extras round out the set.
	modules = append(modules, cfg.Extra...)

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
		Modules:   modules,
	})
	if err != nil {
		return nil, fmt.Errorf("server: build kit: %w", err)
	}

	return &Server{cfg: cfg, kit: kit, gw: gw}, nil
}

// Start auto-deploys packages then blocks until ctx cancels or the
// process receives SIGINT/SIGTERM. The HTTP gateway is already
// listening at this point (gateway.Module.Init starts it); this
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
	if cfg.Gateway.Listen == "" {
		return fmt.Errorf("server: Gateway.Listen is required")
	}
	return nil
}

func resolveAuditStore(cfg Config) (audit.Store, error) {
	// Explicit Audit config → use its Path (SQLite).
	var path string
	if cfg.Audit != nil && cfg.Audit.Path != "" {
		path = cfg.Audit.Path
	} else {
		path = filepath.Join(cfg.FSRoot, "audit.db")
	}
	store, err := auditstores.NewSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("server: open audit store %q: %w", path, err)
	}
	return store, nil
}
