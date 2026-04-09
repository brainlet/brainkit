package brainkit

import (
	"log/slog"
	"path/filepath"

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

	// Transport backend. Empty or "embedded" = in-process NATS (zero config, plugins work).
	// "memory" = GoChannel (fast, no plugins — for tests).
	// "nats" = external NATS (requires NATSURL). "amqp", "redis" for existing infra.
	Transport string

	// Transport connection details (used when Transport is set).
	NATSURL  string
	NATSName string
	AMQPURL  string
	RedisURL string

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

	// Roles configures RBAC. Nil = no enforcement.
	Roles map[string]Role

	// DefaultRole is applied when a deployment has no explicit role. Default: "service".
	DefaultRole string

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

	// MaxConcurrency limits concurrent bus handler invocations. 0 = unlimited.
	MaxConcurrency int

	// MaxStackSize for the QuickJS runtime in bytes. Default: 1MB.
	MaxStackSize int

	// RetryPolicies maps topic glob patterns to retry configurations.
	RetryPolicies map[string]RetryPolicy

	// BusRateLimits maps RBAC role names to publish rate limits (req/s).
	BusRateLimits map[string]float64

	// Discovery configures cross-Kit peer discovery.
	Discovery DiscoveryConfig

	// PluginRegistries for packages.search/install. Default: official brainlet registry.
	PluginRegistries []RegistryConfig

	// PluginDir is the local cache for installed plugin binaries.
	PluginDir string
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
		DefaultRole:        c.DefaultRole,
		TraceSampleRate:    c.TraceSampleRate,
		MaxStackSize:       c.MaxStackSize,
		MaxConcurrency:     c.MaxConcurrency,
		RetryPolicies:      c.RetryPolicies,
		BusRateLimits:      c.BusRateLimits,
		PluginRegistries:   c.PluginRegistries,
		PluginDir:          c.PluginDir,
		Logger:             c.Logger,
		LogHandler:         c.LogHandler,
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

	// Convert Roles
	if len(c.Roles) > 0 {
		cfg.Roles = c.Roles
	}

	// TraceStore: explicit > Tracing flag > nil
	if c.TraceStore != nil {
		cfg.TraceStore = c.TraceStore
	} else if c.Tracing {
		cfg.TraceStore = tracing.NewMemoryTraceStore(10000)
	}

	// Store: explicit > auto from FSRoot > nil
	if c.Store != nil {
		cfg.Store = c.Store
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
	if c.Transport == "embedded" && c.FSRoot != "" {
		natsStoreDir = filepath.Join(c.FSRoot, "nats-data")
	}

	nc := types.NodeConfig{
		Kernel: kernelCfg,
		Messaging: types.MessagingConfig{
			Transport:    c.Transport,
			NATSURL:      c.NATSURL,
			NATSName:     c.NATSName,
			AMQPURL:      c.AMQPURL,
			RedisURL:     c.RedisURL,
			NATSStoreDir: natsStoreDir,
		},
		Discovery: types.DiscoveryConfig{
			Type:        c.Discovery.Type,
			ServiceName: c.Discovery.ServiceName,
		},
	}

	// Convert plugins
	if len(c.Plugins) > 0 {
		nc.Plugins = c.Plugins
	}

	// Convert discovery static peers
	if len(c.Discovery.StaticPeers) > 0 {
		nc.Discovery.StaticPeers = c.Discovery.StaticPeers
	}

	return nc
}
