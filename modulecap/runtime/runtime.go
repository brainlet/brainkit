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
	agentsmod "github.com/brainlet/brainkit/modules/agents"
	provreg "github.com/brainlet/brainkit/modules/registry/providerreg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
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

// Host is the narrow kernel surface required to activate and run the optional
// JS/TS runtime.
type Host interface {
	Deployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
	HarnessRuntime() any

	RuntimeConfig() types.KernelConfig
	SetRuntimeConfigJSRuntime(active bool)
	HasJSRuntime() bool
	AttachJSRuntime(Attachment) error
	DetachJSRuntime(Attachment)
	SetDeployOrderSeed(seed int32)

	Logger() *slog.Logger
	Tracer() *tracing.Tracer
	Namespace() string
	CallerID() string
	SecretStore() types.SecretStore
	ProviderRegistry() *provreg.ProviderRegistry
	ToolsDomain() *toolsmod.Domain
	AgentsDomain() *agentsmod.Domain
	ScheduleHandler() types.ScheduleHandler

	SetToolEvaluator(JSEvaluator)
	ExistingStorageBridgeNames() map[string]bool
	CloseStorageBridgesExcept(keep map[string]bool)
	InitStorageBridges(cfg types.KernelConfig) (map[string]string, error)
	RegisterConfiguredStorages(cfg types.KernelConfig, bridgeURLs map[string]string)
	RegisterConfiguredVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error
	RestoreJSRuntimeState(cfg types.KernelConfig)
	ProbeAll()

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
	EnterHandler() bool
	ExitHandler()
	HandleHandlerFailure(msg sdk.Message, topic string, err error)

	IsClosed() bool
	IncrementPumpCycles()
	EmitLog(source, level, message string)
}
