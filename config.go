package brainkit

import (
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/internal/plugin"
	mcppkg "github.com/brainlet/brainkit/mcp"
	"github.com/brainlet/brainkit/registry"
)

// PluginConfig configures a single plugin.
// Implementation moved to internal/plugin.
type PluginConfig = plugin.Config

// Config configures a Kit.
type Config struct {
	// Name is an optional unique name for this Kit on the bus.
	// When set, the bus enforces uniqueness — no two Kits can share the same name.
	// Used for addressing: "kit:<name>/agent:X".
	Name string

	// Namespace for this Kit (e.g., "user", "agent.team-1").
	Namespace string

	// CallerID — identity for bus messages from this Kit.
	CallerID string

	// Providers maps provider names to AI model configs.
	Providers map[string]ProviderConfig

	// EnvVars injected into the runtime.
	EnvVars map[string]string

	// MaxStackSize for the QuickJS runtime in bytes. Default: 64MB.
	// Increase if you hit stack overflow with deeply recursive JS.
	MaxStackSize int

	// SharedBus — if set, this Kit uses the provided Bus instead of creating its own.
	// Multiple Kits sharing a Bus can communicate via pub/sub.
	// Each Kit still has its own CallerID for message identity.
	SharedBus *bus.Bus

	// SharedTools — if set, this Kit uses the provided ToolRegistry instead of creating its own.
	// Tools registered in one Kit are visible to all Kits sharing the registry.
	SharedTools *registry.ToolRegistry

	// MCPServers — external MCP servers to connect to on Kit creation.
	// Tools from these servers are auto-registered in the ToolRegistry.
	MCPServers map[string]mcppkg.ServerConfig

	// Observability configures tracing for agents, tools, workflows, and LLM calls.
	// Default: enabled with realtime strategy and InMemoryStore.
	Observability ObservabilityConfig

	// Store provides optional persistence for WASM modules, shard descriptors, and shard state.
	// When set, data survives Kit restarts. Use NewSQLiteStore(path) for the default implementation.
	// nil = no persistence (everything in-memory, current behavior).
	Store KitStore

	// Storages configures named storage backends started with the Kit.
	// Each entry starts an embedded SQLite-backed LibSQL bridge.
	// JS code uses `new LibSQLStore({ id: "x", storage: "name" })` to connect.
	// The first entry (or one named "default") is used when no storage name is given.
	//
	// Example:
	//   Storages: map[string]StorageConfig{
	//       "default": { Path: "./data.db" },
	//       "vectors": { Path: "./vectors.db" },
	//   }
	Storages map[string]StorageConfig

	// Plugins configures external plugin processes.
	// Each plugin is started as a subprocess communicating via gRPC.
	Plugins []PluginConfig

	// Network configures Kit-to-Kit networking.
	Network NetworkConfig

	// Transport selects the message transport: "" (default, in-process/grpc), "nats".
	Transport string

	// NATS configures the NATS transport. Only used when Transport is "nats".
	NATS NATSConfig

	// WorkerGroup, when set, causes registerHandlers to use AsWorker(WorkerGroup)
	// for all Bus.On() calls. Set automatically by InstanceManager.spawnInstance.
	WorkerGroup string

	// WorkspaceDir is the root directory for fs.* operations.
	// All file paths are sandboxed to this directory.
	WorkspaceDir string
}

// ProviderConfig configures an AI provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}

// StorageConfig configures an embedded SQLite storage backend.
// Backed by modernc.org/sqlite (pure Go) with a Hrana HTTP bridge.
type StorageConfig struct {
	// Path to the SQLite database file. Created if it doesn't exist.
	// Use ":memory:" for an in-memory database (lost on close).
	Path string
}

// ObservabilityConfig configures the tracing/observability system.
type ObservabilityConfig struct {
	// Enabled controls whether observability is active. Default: true.
	Enabled *bool

	// Strategy controls how spans are exported to storage.
	// "realtime" — writes immediately per span event (default, best for debugging)
	// "insert-only" — batches writes, flushes on timer (better for high-throughput production)
	// "batch-with-updates" — batches creates + updates (most efficient for SQL stores)
	Strategy string

	// ServiceName identifies this service in traces. Default: "brainkit".
	ServiceName string
}
