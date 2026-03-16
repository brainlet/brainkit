package brainkit

import (
	"github.com/brainlet/brainkit/bus"
	mcppkg "github.com/brainlet/brainkit/mcp"
	"github.com/brainlet/brainkit/registry"
)

// Config configures a Kit.
type Config struct {
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
}

// ProviderConfig configures an AI provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
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
