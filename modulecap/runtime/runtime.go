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
	bkmodule "github.com/brainlet/brainkit/module"
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

// DebugSnapshot reports JS runtime ownership counters for lifecycle inspect
// surfaces. It intentionally contains state and counts only.
type DebugSnapshot struct {
	Phase                               string         `json:"phase"`
	Closed                              bool           `json:"closed"`
	ActiveDeployments                   int            `json:"activeDeployments"`
	ResourceCount                       int            `json:"resourceCount"`
	ResourcesByType                     map[string]int `json:"resourcesByType,omitempty"`
	BridgeSubscriptions                 int            `json:"bridgeSubscriptions"`
	BridgeClosing                       bool           `json:"bridgeClosing"`
	BridgeClosed                        bool           `json:"bridgeClosed"`
	BridgeGoroutines                    int            `json:"bridgeGoroutines"`
	RuntimeHostPropagationSubscriptions int            `json:"runtimeHostPropagationSubscriptions,omitempty"`
}

// DebugSnapshotter exposes read-only JS runtime lifecycle counters.
type DebugSnapshotter interface {
	JSRuntimeDebugSnapshot() DebugSnapshot
}

// SourceDeployer handles lifecycle of raw .ts/.js source deployments. Callers
// that already own bundling/normalization should consume ArtifactDeployer
// instead.
type SourceDeployer interface {
	Deploy(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error)
	Teardown(ctx context.Context, source string) (int, error)
	ListDeployments() []DeploymentInfo
}

// ArtifactDeployer handles already-normalized JavaScript artifacts. It exists
// so package/tooling modules do not need broad raw-source deployment access.
type ArtifactDeployer interface {
	DeployArtifact(ctx context.Context, source, code string, opts ...types.DeployOption) ([]types.ResourceInfo, error)
	Teardown(ctx context.Context, source string) (int, error)
	ListDeployments() []DeploymentInfo
}

// TSRunner evaluates direct JS/TS snippets in the active runtime. It is not a
// package/file-graph bundler; package normalization belongs to the package
// builder registered for modules/packages or to tooling before deploy handoff.
type TSRunner interface {
	EvalTS(ctx context.Context, source, code string) (string, error)
}

// DirectEvaluator evaluates direct snippets and modules without exposing raw
// deploy/teardown to command modules.
type DirectEvaluator interface {
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
}

// ScriptEvaluator runs a temporary raw-source script and returns its result.
// Implementations own the deploy/teardown details.
type ScriptEvaluator interface {
	EvalScript(ctx context.Context, source, code string) (string, error)
}

// EvalRuntime is the module-facing JS/TS eval surface. It is narrower than
// Host so eval consumers cannot reach the runtime activation/kernel adapter or
// raw deploy/teardown primitives.
type EvalRuntime interface {
	DirectEvaluator
	ScriptEvaluator
}

// TestRuntime is the explicit dev/test runtime surface consumed by
// modules/testing. It keeps raw source deploy and direct TS evaluation out of
// the general module capability list while still letting the test runner deploy
// fixture source and already-bundled test artifacts.
type TestRuntime interface {
	EvalTS(ctx context.Context, source, code string) (string, error)
	DeploySource(ctx context.Context, source, code string) ([]types.ResourceInfo, error)
	DeployArtifact(ctx context.Context, source, code string) ([]types.ResourceInfo, error)
	Teardown(ctx context.Context, source string) (int, error)
}

// JSEvaluator runs JavaScript on the runtime bridge's JS thread.
type JSEvaluator interface {
	EvalOnJSThread(filename, code string) (string, error)
}

// Attachment is the engine-facing surface implemented by the optional JS/TS
// runtime package.
type Attachment interface {
	SourceDeployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
	ListResources(resourceType ...string) ([]types.ResourceInfo, error)
	ResourcesFrom(filename string) ([]types.ResourceInfo, error)
	TeardownFile(filename string) (int, error)
	RemoveResource(resourceType, id string) error
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
	// HarnessRuntime returns the optional harness adapter as an opaque value.
	// Keep this untyped so the root runtime attachment does not import
	// QuickJS-shaped harness contracts; modules/jsruntime type-checks it before
	// providing the typed module-facing capability.
	HarnessRuntime() any
	Interrupt()
	Unmount(ctx context.Context) error
	Shutdown(ctx context.Context) error
	Close() error
	CurrentSource() string
	SetCurrentSource(source string)
	NextDeployOrder() int
	SetDeployOrderSeed(seed int32)
}

// Access is the runtime-facing surface exposed after the JS/TS runtime is
// attached.
type Access interface {
	SourceDeployer
	TSRunner
	EvalModule(ctx context.Context, source, code string) (string, error)
	CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error)
	// HarnessRuntime returns the optional harness adapter as an opaque value.
	// Module consumers receive the typed capability from modules/jsruntime, not
	// this root-light attachment boundary.
	HarnessRuntime() any
}

// ActivationHost owns runtime attachment lifecycle and persistence restore.
type ActivationHost interface {
	RuntimeConfig() types.KernelConfig
	SetRuntimeConfigJSRuntime(active bool)
	HasJSRuntime() bool
	AttachJSRuntime(Attachment) error
	DetachJSRuntime(Attachment)
	DisableJSRuntime(ctx context.Context) error
	SetDeployOrderSeed(seed int32)
	RestoreJSRuntimeState(cfg types.KernelConfig)
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

	LeaseToolEvaluator(context.Context, JSEvaluator) (bkmodule.Handle, error)
}

// StorageHost owns storage bridge lifecycle and registry refresh.
type StorageHost interface {
	AddStorage(name string, cfg types.StorageConfig) error
	RemoveStorage(name string) error
	RemoveStorageContext(context.Context, string) error
	AddVector(name string, cfg types.VectorConfig) error
	RemoveVector(name string) error
	RemoveVectorContext(context.Context, string) error
	ExistingStorageBridgeNames() map[string]bool
	CloseStorageBridgesExcept(keep map[string]bool) error
	CloseStorageBridgesExceptContext(context.Context, map[string]bool) error
	InitStorageBridges(cfg types.KernelConfig) (map[string]string, error)
	RegisterConfiguredStorages(cfg types.KernelConfig, bridgeURLs map[string]string) error
	RegisterConfiguredVectors(cfg types.KernelConfig, bridgeURLs map[string]string) error
	RestoreConfiguredStorageRegistry(cfg types.KernelConfig, bridgeURLs map[string]string) error
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

// EnableHost is the kernel surface required to activate and run the optional
// JS/TS runtime. It intentionally excludes Access: runtime access becomes
// available only after activation attaches a runtime to the kernel.
type EnableHost interface {
	ActivationHost
	CoreHost
	RegistryHost
	ToolAgentHost
	StorageHost
	BusHost
	HandlerHost
	ScheduleHost
}

// Host is the complete module-facing runtime capability. Keep it as a
// composition of smaller contracts so new dependencies are added to the narrow
// group that actually needs them.
type Host interface {
	Access
	EnableHost
}
