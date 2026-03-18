package sdk

import (
	"context"
	"encoding/json"
)

// Plugin is the interface plugin authors implement.
type Plugin interface {
	// Manifest returns the plugin's capability declaration.
	Manifest() PluginManifest

	// OnStart is called after handshake + manifest processing.
	OnStart(client BrainkitClient) error

	// OnStop is called before shutdown.
	OnStop() error

	// HandleToolCall processes a tool invocation.
	HandleToolCall(ctx context.Context, tool string, input json.RawMessage) (json.RawMessage, error)

	// HandleEvent processes a subscribed bus event.
	HandleEvent(ctx context.Context, event Event) error

	// HandleIntercept processes a message interception.
	HandleIntercept(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error)
}

// BrainkitClient is the Kit API available to plugins via gRPC.
type BrainkitClient interface {
	// Bus operations
	Send(ctx context.Context, topic string, payload json.RawMessage) error
	Ask(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error)

	// Convenience methods (route through bus internally)
	CallTool(ctx context.Context, name string, input json.RawMessage) (json.RawMessage, error)
	CallAgent(ctx context.Context, name string, prompt string) (string, error)
	CompileWASM(ctx context.Context, source string, opts WASMCompileOpts) (*WASMModule, error)
	DeployWASM(ctx context.Context, name string) (*ShardDescriptor, error)

	// State (per-plugin key-value, backed by Kit persistence)
	GetState(ctx context.Context, key string) (string, error)
	SetState(ctx context.Context, key string, value string) error
}
