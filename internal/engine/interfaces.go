package engine

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk"
)

// BusPublisher sends and receives bus sdk.
// Implemented by *transport.RemoteClient (via Kernel).
type BusPublisher interface {
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error)
	SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error)
}

// JSEvaluator runs JavaScript on the bridge.
// Implemented by *jsbridge.Bridge.
type JSEvaluator interface {
	EvalOnJSThread(filename, code string) (string, error)
}

// PluginRestarter abstracts plugin restart for secrets rotation.
// Implemented by *Node. Nil on standalone Kernel.
type PluginRestarter interface {
	ListRunningPlugins() []RunningPlugin
	RestartPlugin(ctx context.Context, name string) error
}
