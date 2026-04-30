// Package runtimecap defines module-facing JS runtime capability contracts.
package runtimecap

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	agenthost "github.com/brainlet/brainkit/modulehost/agenthost"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	toolhost "github.com/brainlet/brainkit/modulehost/toolhost"
	"github.com/brainlet/brainkit/sdk"
)

// DeploymentInfo describes a deployed JS/TS source and its tracked resources.
type DeploymentInfo struct {
	Source    string               `json:"source"`
	CreatedAt time.Time            `json:"createdAt"`
	Resources []types.ResourceInfo `json:"resources,omitempty"`
	Order     int                  `json:"order"`
}

// Deployer handles lifecycle of .ts/.js file deployments.
type Deployer interface {
	Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error)
	Teardown(ctx context.Context, source string) (int, error)
	ListDeployments() []DeploymentInfo
}

// TSRunner evaluates JS/TS code in the active runtime.
type TSRunner interface {
	EvalTS(ctx context.Context, source, code string) (string, error)
}

// EvalRuntime is the module-facing JS/TS eval surface. It is narrower than
// Host so eval consumers cannot reach the runtime activation/kernel adapter.
type EvalRuntime interface {
	Deployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
}

// JSEvaluator runs JavaScript on the runtime bridge's JS thread.
type JSEvaluator interface {
	EvalOnJSThread(filename, code string) (string, error)
}

// Attachment is the engine-facing surface implemented by the optional JS/TS
// runtime package.
type Attachment interface {
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

// Access is the runtime-facing surface exposed after the JS/TS runtime is
// attached.
type Access interface {
	Deployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
	HarnessRuntime() any
}

// ActivationHost owns runtime attachment lifecycle and persistence restore.
type ActivationHost interface {
	RuntimeConfig() types.KernelConfig
	SetRuntimeConfigJSRuntime(active bool)
	HasJSRuntime() bool
	AttachJSRuntime(Attachment) error
	DetachJSRuntime(Attachment)
	SetDeployOrderSeed(seed int32)
	RestoreJSRuntimeState(cfg types.KernelConfig)
	ProbeAll()
}

// CoreHost exposes identity, logging, tracing, and secret lookup.
type CoreHost interface {
	Logger() *slog.Logger
	Tracer() *tracing.Tracer
	Namespace() string
	CallerID() string
	SecretStore() types.SecretStore
}

// RegistryHost exposes provider/storage/vector registry state.
type RegistryHost interface {
	ProviderRegistry() *provreg.ProviderRegistry
}

// ToolAgentHost exposes local tool and agent registries used by JS code.
type ToolAgentHost interface {
	ToolsDomain() *toolhost.Domain
	AgentsDomain() *agenthost.Domain

	SetToolEvaluator(JSEvaluator)
}

// StorageHost owns storage bridge lifecycle and registry refresh.
type StorageHost interface {
	ExistingStorageBridgeNames() map[string]bool
	CloseStorageBridgesExcept(keep map[string]bool)
	InitStorageBridges(cfg types.KernelConfig) (map[string]string, error)
	RegisterConfiguredStorages(cfg types.KernelConfig, bridgeURLs map[string]string)
	RegisterConfiguredVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error
}

// BusHost exposes bus operations used by JS bus and command bridges.
type BusHost interface {
	Remote() *transport.RemoteClient
	Caller() *sdk.Caller
	InvokeCommand(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error)
	HasCommand(topic string) bool
	ValidateEvent(topic string, payload json.RawMessage) error
	PublishEvent(ctx context.Context, topic string, payload json.RawMessage) error
	SubscribeEvent(topic string, handler func(sdk.Message)) (func(), error)
	ReplyRawWithEnvelope(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool, envelope bool) error
	StartStreamHeartbeat(replyTo, correlationID string)
	StopStreamHeartbeat(replyTo string)
}

// HandlerHost tracks JS-dispatched bus handler lifecycle and failures.
type HandlerHost interface {
	EnterHandler() bool
	ExitHandler()
	HandleHandlerFailure(msg sdk.Message, topic string, err error)

	IsClosed() bool
	IncrementPumpCycles()
	EmitLog(source, level, message string)
}

// ScheduleHost exposes the active schedule handler, if modules/schedules is mounted.
type ScheduleHost interface {
	ScheduleHandler() types.ScheduleHandler
}

// Host is the complete kernel surface required to activate and run the optional
// JS/TS runtime. Keep it as a composition of smaller capability contracts so
// new dependencies are added to the narrow group that actually needs them.
type Host interface {
	Access
	ActivationHost
	CoreHost
	RegistryHost
	ToolAgentHost
	StorageHost
	BusHost
	HandlerHost
	ScheduleHost
}
