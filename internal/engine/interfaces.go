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

// Deployer handles lifecycle of .ts/.js file deployments.
type Deployer = runtimecap.Deployer

// TSRunner evaluates JS/TS code in the active runtime.
type TSRunner = runtimecap.TSRunner

// JSRuntimeAttachment is the engine-facing surface implemented by the
// optional JS/TS runtime package.
type JSRuntimeAttachment = runtimecap.Attachment
