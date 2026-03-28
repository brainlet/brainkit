package kit

import (
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/messaging"
	toolreg "github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit/registry"
)

// KernelConfig configures the local runtime.
// The Kernel is a resource pool — the Go developer fills it with providers,
// storages, and vectors. Deployed .ts code picks from the pool via
// storage("name"), vectorStore("name"), and model("provider", "id").
type KernelConfig struct {
	// Identity
	Namespace string
	CallerID  string

	// AI providers — explicit config for custom base URLs.
	// For simple API key usage, leave empty and set the env var
	// (e.g., OPENAI_API_KEY) — auto-detected from os.Getenv.
	AIProviders map[string]registry.AIProviderRegistration

	// EnvVars overrides os.Getenv for specific keys.
	// process.env already reads os.Getenv directly, so this is only needed
	// to override a key for THIS Kernel (e.g., different API key than OS default).
	EnvVars map[string]string

	// Storage pool — deployments pick via storage("name").
	// Multiple backends, multiple instances. SQLite backends auto-start
	// a libsql HTTP bridge transparently.
	Storages map[string]StorageConfig

	// Vector pool — deployments pick via vectorStore("name").
	Vectors map[string]VectorConfig

	// Filesystem sandbox root — deployments access via fs.read/write/list.
	FSRoot string

	// Infrastructure
	MaxStackSize  int
	SharedTools   *toolreg.ToolRegistry
	MCPServers    map[string]mcppkg.ServerConfig
	Observability ObservabilityConfig
	Store         KitStore
	Probe         registry.ProbeConfig

	// LogHandler receives tagged log entries from .ts Compartments, WASM modules,
	// and the Kernel. Called concurrently from multiple goroutines — must be safe.
	// nil = default (print to stdout via log.Printf).
	LogHandler func(LogEntry)

	// Transport is an optional external transport. If set, Kernel uses it instead of
	// creating its own internal GoChannel transport. Used by Node to inject NATS.
	Transport *messaging.Transport

	// DeferRouterStart skips starting the router during NewKernel.
	// Used by Node to register node-specific command bindings before starting.
	DeferRouterStart bool
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	Enabled     *bool
	Strategy    string
	ServiceName string
}
