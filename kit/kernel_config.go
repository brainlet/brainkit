package kit

import (
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/messaging"
	toolreg "github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit/registry"
)

// KernelConfig configures the local runtime.
type KernelConfig struct {
	Name         string
	Namespace    string
	CallerID     string
	EnvVars      map[string]string
	MaxStackSize int
	SharedTools  *toolreg.ToolRegistry
	MCPServers   map[string]mcppkg.ServerConfig
	Observability ObservabilityConfig
	Store        KitStore
	WorkspaceDir string

	// AIProviders registers typed AI provider configurations.
	// Each entry maps a name to a typed provider config.
	AIProviders map[string]registry.AIProviderRegistration

	// VectorStores registers typed vector store configurations.
	VectorStores map[string]registry.VectorStoreRegistration

	// MastraStorages registers typed Mastra storage adapter configurations.
	MastraStorages map[string]registry.StorageRegistration

	// EmbeddedStorages configures the embedded libsql HTTP bridge instances.
	// These provide local SQLite-backed storage accessible via HTTP (for LibSQLStore/LibSQLVector).
	EmbeddedStorages map[string]EmbeddedStorageConfig

	// Probe configures health probing behavior for registered providers.
	Probe registry.ProbeConfig

	// Transport is an optional external transport. If set, Kernel uses it instead of
	// creating its own internal GoChannel transport. Used by Node to inject NATS.
	Transport *messaging.Transport

	// DeferRouterStart skips starting the router during NewKernel.
	// Used by Node to register node-specific command bindings before starting.
	DeferRouterStart bool

	// LogHandler receives tagged log entries from .ts Compartments, WASM modules,
	// and the Kernel. Called concurrently from multiple goroutines — must be safe.
	// nil = default (print to stdout via log.Printf).
	LogHandler func(LogEntry)
}

// EmbeddedStorageConfig configures an embedded SQLite storage backend (libsql HTTP bridge).
type EmbeddedStorageConfig struct {
	Path string
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	Enabled     *bool
	Strategy    string
	ServiceName string
}
