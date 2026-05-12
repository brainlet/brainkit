package engine

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/modulecap/plugin"
	"github.com/brainlet/brainkit/modulecap/runtime"
	"github.com/brainlet/brainkit/sdk"
)

// BusPublisher sends and receives bus sdk.
// Implemented by *transport.RemoteClient (via Kernel).
type BusPublisher interface {
	PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error)
	SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error)
}

// JSEvaluator runs JavaScript on the bridge.
type JSEvaluator = runtimecap.JSEvaluator

// PluginRestarter abstracts plugin restart for secrets rotation.
type PluginRestarter = plugincap.Restarter

// JSRunner evaluates JavaScript code in the active runtime.
type JSRunner = runtimecap.JSRunner

// JSRuntimeAttachment is the engine-facing surface implemented by the
// optional JavaScript runtime package.
type JSRuntimeAttachment = runtimecap.Attachment

var _ runtimecap.Host = (*Kernel)(nil)
var _ runtimecap.EnableHost = (*Kernel)(nil)
