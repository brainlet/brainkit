package kit

import (
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/registry"
)

// KernelConfig configures the local runtime.
type KernelConfig struct {
	Name          string
	Namespace     string
	CallerID      string
	Providers     map[string]ProviderConfig
	EnvVars       map[string]string
	MaxStackSize  int
	SharedTools   *registry.ToolRegistry
	MCPServers    map[string]mcppkg.ServerConfig
	Observability ObservabilityConfig
	Store         KitStore
	Storages      map[string]StorageConfig
	WorkspaceDir  string

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

// ProviderConfig configures an AI provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

// StorageConfig configures an embedded SQLite storage backend.
type StorageConfig struct {
	Path string
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	Enabled     *bool
	Strategy    string
	ServiceName string
}
