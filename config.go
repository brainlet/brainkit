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
}

// ProviderConfig configures an AI provider.
type ProviderConfig struct {
	APIKey  string
	BaseURL string
}
