package engine

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/internal/types"
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
	ListRunningPlugins() []types.RunningPlugin
	RestartPlugin(ctx context.Context, name string) error
}

// Deployer handles lifecycle of .ts/.js file deployments.
// Implemented by *Kernel.
type Deployer interface {
	Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error)
	Teardown(ctx context.Context, source string) (int, error)
	ListDeployments() []DeploymentInfo
}

// TSRunner evaluates JS/TS code in the QuickJS runtime via EvalTS.
type TSRunner interface {
	EvalTS(ctx context.Context, source, code string) (string, error)
}

// JSRuntimeAttachment is the engine-facing surface implemented by the
// optional JS/TS runtime package. Kernel delegates JS-dependent operations
// through this interface so the control plane can stay light when no runtime
// module is mounted.
type JSRuntimeAttachment interface {
	Deployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
	ListResources(resourceType ...string) ([]types.ResourceInfo, error)
	ResourcesFrom(filename string) ([]types.ResourceInfo, error)
	TeardownFile(filename string) (int, error)
	RemoveResource(resourceType, id string) error
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
	CallJSSync(fn string, args any)
	HarnessRuntime() any
	Interrupt()
	Close() error
	CurrentSource() string
	SetCurrentSource(source string)
	NextDeployOrder() int
	SetDeployOrderSeed(seed int32)
}
