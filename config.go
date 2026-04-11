package brainkit

import (
	"log/slog"
	"path/filepath"

	kitstore "github.com/brainlet/brainkit/internal/store"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/types"
)

// Config configures a brainkit runtime.
//
// All fields are optional with sensible defaults. The zero value creates a
// standalone in-memory runtime with no persistence and auto-detected AI providers.
type Config struct {
	// ClusterID identifies the logical group of runtimes. Default: "default".
	// All runtimes on the same transport with the same ClusterID discover each other.
	ClusterID string

	// Namespace identifies this runtime on the bus. Default: "user".
	Namespace string

	// CallerID identifies this runtime instance in message metadata. Default: Namespace.
	CallerID string

	// Transport configures the bus backend. One per Kit.
	// Zero value = EmbeddedNATS() (in-process NATS, zero config, plugins work).
	// Use Memory() for tests, NATS(url) / AMQP(url) / Redis(url) for external infra.
	Transport TransportConfig

	// FSRoot is the filesystem sandbox for deployed .ts code.
	FSRoot string

	// Storages configures named storage backends.
	// Deployments access them via storage("name") in .ts code.
	Storages map[string]StorageConfig

	// Vectors configures named vector store backends.
	// Deployments access them via vectorStore("name") in .ts code.
	Vectors map[string]VectorConfig

	// Providers configures AI providers. Nil = auto-detect from env
	// (OPENAI_API_KEY → openai, ANTHROPIC_API_KEY → anthropic, etc.)
	Providers []ProviderConfig

	// EnvVars overrides os.Getenv for specific keys within this runtime.
	EnvVars map[string]string

	// SecretKey is the master encryption key for the secret store.
	// Empty = env-only dev mode (secrets read from os.Getenv).
	SecretKey string

	// SecretStore overrides the auto-created secret store.
	// Most users leave this nil and set SecretKey instead.
	SecretStore SecretStore

	// Tracing enables distributed tracing with an auto-created MemoryTraceStore.
	Tracing bool

	// TraceStore overrides the auto-created trace store. Overrides the Tracing flag.
	TraceStore TraceStore

	// TraceSampleRate controls trace sampling (0.0–1.0). Default: 1.0.
	TraceSampleRate float64

	// Store provides persistence for deployments, schedules, and plugins.
	// Nil + FSRoot set = auto-create SQLiteStore at <FSRoot>/brainkit-store.db.
	// Nil + FSRoot empty = no persistence (ephemeral).
	Store KitStore

	// StoreBackend selects the persistence engine: "sqlite" (default) or "postgres".
	// When set with StoreURL, the factory creates the store automatically.
	// Overridden by explicit Store field.
	StoreBackend string

	// StoreURL is the connection string for the store backend.
	// For postgres: "postgres://user:pass@host:5432/db?sslmode=disable"
	// For sqlite: file path (defaults to <FSRoot>/brainkit-store.db)
	StoreURL string

	// Plugins to start automatically (requires Transport).
	Plugins []PluginConfig

	// MCPServers configures external MCP tool providers.
	MCPServers map[string]MCPServerConfig

	// Logger for structured logging. Nil = slog.Default().
	Logger *slog.Logger

	// LogHandler receives tagged log entries from .ts code and the runtime.
	LogHandler func(LogEntry)

	// ErrorHandler receives non-fatal errors (persistence failures, plugin errors).
	ErrorHandler func(error)

	// AuditVerbose enables high-volume audit logging: every bus command completion
	// and periodic metric snapshots. Useful with high-throughput audit stores (Postgres).
	// Default: false (only lifecycle events, failures, security events, tool calls).
	AuditVerbose bool

	// MaxConcurrency limits concurrent bus handler invocations. 0 = unlimited.
	MaxConcurrency int

	// MaxStackSize for the QuickJS runtime in bytes. Default: 1MB.
	MaxStackSize int

	// RetryPolicies maps topic glob patterns to retry configurations.
	RetryPolicies map[string]RetryPolicy

	// Discovery configures cross-Kit peer discovery.
	Discovery DiscoveryConfig

	// Modules are optional subsystems that extend the kernel with additional commands.
	// See brainkit.NewMCPModule() for an example.
	Modules []Module
}

// toKernelConfig converts the flat Config to the internal engine KernelConfig.
func (c Config) toKernelConfig() types.KernelConfig {
	cfg := types.KernelConfig{
		ClusterID:          c.ClusterID,
		RuntimeID:          runtimeID,
		Namespace:          c.Namespace,
		CallerID:           c.CallerID,
		FSRoot:             c.FSRoot,
		Storages:           c.Storages,
		Vectors:            c.Vectors,
		EnvVars:            c.EnvVars,
		SecretKey:          c.SecretKey,
		SecretStore:        c.SecretStore,
		TraceSampleRate:    c.TraceSampleRate,
		MaxStackSize:       c.MaxStackSize,
		MaxConcurrency:     c.MaxConcurrency,
		RetryPolicies:      c.RetryPolicies,
		Logger:             c.Logger,
		LogHandler:         c.LogHandler,
		AuditVerbose:       c.AuditVerbose,
	}

	// Convert []ProviderConfig → map[string]AIProviderRegistration
	if len(c.Providers) > 0 {
		cfg.AIProviders = make(map[string]types.AIProviderRegistration, len(c.Providers))
		for _, p := range c.Providers {
			cfg.AIProviders[p.name] = types.AIProviderRegistration{
				Type:   types.AIProviderType(p.typ),
				Config: p.toConfig(),
			}
		}
	}

	// TraceStore: explicit > Tracing flag > nil
	if c.TraceStore != nil {
		cfg.TraceStore = c.TraceStore
	} else if c.Tracing {
		cfg.TraceStore = tracing.NewMemoryTraceStore(10000)
	}

	// Store: explicit > StoreBackend factory > auto-create from FSRoot > nil
	if c.Store != nil {
		cfg.Store = c.Store
	} else if c.StoreBackend != "" {
		storeURL := c.StoreURL
		if storeURL == "" && c.FSRoot != "" {
			storeURL = filepath.Join(c.FSRoot, "brainkit-store.db")
		}
		if storeURL != "" {
			s, err := kitstore.NewKitStore(kitstore.Config{
				Backend: c.StoreBackend, SQLitePath: storeURL, PostgresURL: storeURL,
			})
			if err == nil {
				cfg.Store = s
			}
		}
	} else if c.FSRoot != "" {
		store, err := types.NewSQLiteStore(filepath.Join(c.FSRoot, "brainkit-store.db"))
		if err == nil {
			cfg.Store = store
		}
	}

	// ErrorHandler adaptation (Config takes func(error), engine takes func(error, ErrorContext))
	if c.ErrorHandler != nil {
		cfg.ErrorHandler = func(err error, ctx types.ErrorContext) {
			c.ErrorHandler(err)
		}
	}

	// Modules — pass through directly, engine.NewKernel asserts the interface
	if len(c.Modules) > 0 {
		cfg.Modules = make([]any, len(c.Modules))
		for i, m := range c.Modules {
			cfg.Modules[i] = m
		}
	}

	// MCPServers
	if len(c.MCPServers) > 0 {
		cfg.MCPServers = c.MCPServers
	}

	return cfg
}

// toNodeConfig builds a NodeConfig for transport-connected mode.
func (c Config) toNodeConfig(kernelCfg types.KernelConfig) types.NodeConfig {
	// DeferRouterStart is handled internally by engine.NewNode
	kernelCfg.DeferRouterStart = true

	// For embedded NATS, derive JetStream store from FSRoot.
	natsStoreDir := ""
	if c.Transport.typ == "embedded" && c.FSRoot != "" {
		natsStoreDir = filepath.Join(c.FSRoot, "nats-data")
	}

	nc := types.NodeConfig{
		Kernel: kernelCfg,
		Messaging: types.MessagingConfig{
			Transport:    c.Transport.typ,
			NATSURL:      c.Transport.natsURL,
			NATSName:     c.Transport.natsName,
			AMQPURL:      c.Transport.amqpURL,
			RedisURL:     c.Transport.redisURL,
			NATSStoreDir: natsStoreDir,
		},
		Discovery: c.Discovery, // type alias — no conversion needed
	}

	// Convert plugins
	if len(c.Plugins) > 0 {
		nc.Plugins = c.Plugins
	}

	return nc
}
