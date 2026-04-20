// Package server composes a brainkit.Kit with the standard
// service-mode module set — gateway, probes, tracing, audit, and
// optional plugins / discovery / topology — behind a single
// lifecycle. Callers embed server in their binary or run it under
// cmd/brainkit.
package server

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/audit"
	auditstores "github.com/brainlet/brainkit/modules/audit/stores"
	"github.com/brainlet/brainkit/modules/discovery"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/modules/mcp"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/modules/probes"
	"github.com/brainlet/brainkit/modules/schedules"
	"github.com/brainlet/brainkit/modules/topology"
	"github.com/brainlet/brainkit/modules/tracing"
	"github.com/brainlet/brainkit/modules/workflow"
	_ "modernc.org/sqlite" // required for tracing sql.DB
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

	// Audit — non-nil enables the audit module. Empty Path falls
	// back to `<FSRoot>/audit.db`.
	Audit *AuditConfig

	// Tracing — non-nil enables the tracing module. Empty Path falls
	// back to `<FSRoot>/tracing.db`.
	Tracing *TracingConfig

	// Probes — non-nil enables the health-probes module.
	Probes *ProbesConfig

	// Schedules — non-nil enables persisted cron-style scheduling.
	// Empty Path runs the scheduler in memory (no restart survival).
	Schedules *SchedulesConfig

	// MCP — non-nil enables the MCP client module. Requires at
	// least one entry in Servers.
	MCP *MCPConfig

	// Discovery — non-nil enables peer discovery.
	Discovery *DiscoveryConfig

	// Topology — non-nil enables cross-kit routing. When UseDiscovery
	// is true and Discovery is also set, the discovery module feeds
	// topology as a ProviderSource.
	Topology *TopologyConfig

	// Workflow — non-nil enables the workflow module.
	Workflow *WorkflowConfig

	// Packages are deployed after the Kit boots.
	Packages []brainkit.Package

	// Extra lets callers append additional Modules that the server
	// composition doesn't otherwise know about.
	Extra []brainkit.Module
}

// AuditConfig selects the audit store backing. Empty Path falls
// back to `<FSRoot>/audit.db`.
type AuditConfig struct {
	Path    string
	Verbose bool
}

// TracingConfig configures the tracing module. Empty Path falls
// back to `<FSRoot>/tracing.db`. Zero Retention disables cleanup.
type TracingConfig struct {
	Path      string
	Retention time.Duration
}

// ProbesConfig configures periodic health probing. Zero Interval
// disables the periodic sweep; the one-shot probe on register still
// runs unless ProbeOnRegister is explicitly false.
type ProbesConfig struct {
	Interval        time.Duration
	ProbeOnRegister bool
}

// SchedulesConfig configures the scheduling module. Empty Path
// yields an in-memory scheduler that doesn't survive restart (the
// module logs a warning at boot).
type SchedulesConfig struct {
	Path string
}

// MCPConfig configures the MCP-client module. Servers is a map of
// server name → connection config.
type MCPConfig struct {
	Servers map[string]brainkit.MCPServerConfig
}

// DiscoveryConfig configures peer discovery.
//
//	Type: "static" (peers only) | "bus" (broadcast presence) | ""
//
// Heartbeat and TTL are bus-mode tunables (both default when zero).
type DiscoveryConfig struct {
	Type      string
	Name      string
	Heartbeat time.Duration
	TTL       time.Duration
	Peers     []DiscoveryPeer
}

// DiscoveryPeer names a peer Kit reachable on the transport.
type DiscoveryPeer struct {
	Name      string
	Namespace string
	Address   string
	Meta      map[string]string
}

// TopologyConfig configures cross-kit routing. UseDiscovery, when
// true, wires the Discovery module as a dynamic ProviderSource.
type TopologyConfig struct {
	Peers        []DiscoveryPeer
	UseDiscovery bool
}

// WorkflowConfig — currently a presence toggle; type exists so the
// shape can grow without breaking opt-in semantics.
type WorkflowConfig struct{}

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

	modules := make([]brainkit.Module, 0, 12)

	// Gateway — required.
	gw := gateway.New(cfg.Gateway)
	modules = append(modules, gw)

	// Audit — opt-in. Empty path falls back to <FSRoot>/audit.db.
	if cfg.Audit != nil {
		auditStore, err := resolveAuditStore(cfg)
		if err != nil {
			return nil, err
		}
		modules = append(modules, audit.NewModule(audit.Config{
			Store:    auditStore,
			Verbose:  cfg.Audit.Verbose,
			OwnStore: true,
		}))
	}

	// Tracing — opt-in. Empty path falls back to <FSRoot>/tracing.db.
	if cfg.Tracing != nil {
		tstore, err := resolveTraceStore(cfg)
		if err != nil {
			return nil, err
		}
		modules = append(modules, tracing.New(tracing.Config{Store: tstore}))
	}

	// Probes — opt-in.
	if cfg.Probes != nil {
		modules = append(modules, probes.New(probes.Config{
			Interval:        cfg.Probes.Interval,
			ProbeOnRegister: cfg.Probes.ProbeOnRegister,
		}))
	}

	// Schedules — opt-in. Empty path = in-memory (warn).
	if cfg.Schedules != nil {
		schedStore, err := resolveScheduleStore(cfg)
		if err != nil {
			return nil, err
		}
		modules = append(modules, schedules.NewModule(schedules.Config{Store: schedStore}))
	}

	// Plugins — opt-in (legacy: Plugins slice, not a Config pointer).
	if len(cfg.Plugins) > 0 {
		modules = append(modules, pluginsmod.NewModule(pluginsmod.Config{
			Plugins: cfg.Plugins,
			Store:   store,
		}))
	}

	// MCP — opt-in.
	if cfg.MCP != nil {
		modules = append(modules, mcp.New(cfg.MCP.Servers))
	}

	// Discovery + Topology. Discovery is built first so topology
	// can optionally wire it as a ProviderSource without caring
	// about init order.
	var discoveryModule *discovery.Module
	if cfg.Discovery != nil {
		dpeers := make([]discovery.PeerConfig, 0, len(cfg.Discovery.Peers))
		for _, p := range cfg.Discovery.Peers {
			dpeers = append(dpeers, discovery.PeerConfig{
				Name:      p.Name,
				Namespace: p.Namespace,
				Address:   p.Address,
				Meta:      p.Meta,
			})
		}
		discoveryModule = discovery.NewModule(discovery.ModuleConfig{
			Type:        cfg.Discovery.Type,
			Name:        cfg.Discovery.Name,
			Heartbeat:   cfg.Discovery.Heartbeat,
			TTL:         cfg.Discovery.TTL,
			StaticPeers: dpeers,
		})
		modules = append(modules, discoveryModule)
	}
	if cfg.Topology != nil {
		tpeers := make([]topology.Peer, 0, len(cfg.Topology.Peers))
		for _, p := range cfg.Topology.Peers {
			tpeers = append(tpeers, topology.Peer{
				Name:      p.Name,
				Namespace: p.Namespace,
				Address:   p.Address,
				Meta:      p.Meta,
			})
		}
		topoCfg := topology.Config{Peers: tpeers}
		if cfg.Topology.UseDiscovery {
			if discoveryModule == nil {
				return nil, fmt.Errorf("server: topology: use_discovery is true but no discovery module is configured")
			}
			topoCfg.Discovery = discoveryModule
		}
		modules = append(modules, topology.NewModule(topoCfg))
	}

	// Workflow — opt-in toggle.
	if cfg.Workflow != nil {
		modules = append(modules, workflow.New())
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
	path := cfg.Audit.Path
	if path == "" {
		path = filepath.Join(cfg.FSRoot, "audit.db")
	}
	store, err := auditstores.NewSQLite(path)
	if err != nil {
		return nil, fmt.Errorf("server: open audit store %q: %w", path, err)
	}
	return store, nil
}

// resolveTraceStore opens the SQLite database backing the tracing
// module. Empty Path falls back to `<FSRoot>/tracing.db`. Zero
// Retention disables cleanup.
func resolveTraceStore(cfg Config) (tracing.TraceStore, error) {
	path := cfg.Tracing.Path
	if path == "" {
		path = filepath.Join(cfg.FSRoot, "tracing.db")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("server: open trace db %q: %w", path, err)
	}
	opts := make([]tracing.SQLiteTraceStoreOption, 0, 1)
	if cfg.Tracing.Retention > 0 {
		opts = append(opts, tracing.WithRetention(cfg.Tracing.Retention))
	}
	store, err := tracing.NewSQLiteTraceStore(db, opts...)
	if err != nil {
		return nil, fmt.Errorf("server: init trace store %q: %w", path, err)
	}
	return store, nil
}

// resolveScheduleStore returns the persistence backend for the
// scheduling module. Empty Path means in-memory — the module logs a
// warning at boot so operators know schedules won't survive restart.
func resolveScheduleStore(cfg Config) (schedules.Store, error) {
	if cfg.Schedules.Path == "" {
		slog.Warn("schedules: no path configured; using in-memory store (schedules will not survive restart)")
		return nil, nil
	}
	store, err := brainkit.NewSQLiteStore(cfg.Schedules.Path)
	if err != nil {
		return nil, fmt.Errorf("server: open schedules store %q: %w", cfg.Schedules.Path, err)
	}
	return store, nil
}
