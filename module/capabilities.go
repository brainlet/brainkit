package module

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/sdk"
)

const (
	CapabilityRuntimeID              = "brainkit.core.runtime_id"
	CapabilityNamespace              = "brainkit.core.namespace"
	CapabilityCallerID               = "brainkit.core.caller_id"
	CapabilityPresenceTransport      = "brainkit.core.presence_transport"
	CapabilityArtifactDeployer       = "brainkit.core.artifact_deployer"
	CapabilityEvalRuntime            = "brainkit.core.eval_runtime"
	CapabilityTestRuntime            = "brainkit.core.test_runtime"
	CapabilityPluginChecker          = "brainkit.core.plugin_checker"
	CapabilityScheduleHandlerLease   = "brainkit.core.schedule_handler_lease"
	CapabilityAuditStoreLease        = "brainkit.core.audit_store_lease"
	CapabilityAuditVerbosityLease    = "brainkit.core.audit_verbosity_lease"
	CapabilityTraceStoreLease        = "brainkit.core.trace_store_lease"
	CapabilityProbeAll               = "brainkit.core.probe_all"
	CapabilityCallJS                 = "brainkit.core.call_js"
	CapabilityHarnessRuntime         = "brainkit.core.harness_runtime"
	CapabilityTransportKind          = "brainkit.core.transport_kind"
	CapabilitySecretStore            = "brainkit.core.secret_store"
	CapabilityPluginRestarter        = "brainkit.core.plugin_restarter"
	CapabilityRefreshProviderSecret  = "brainkit.core.refresh_provider_secret"
	CapabilityShutdownSignal         = "brainkit.core.shutdown_signal"
	CapabilityRemoteClient           = "brainkit.core.remote_client"
	CapabilityReferenceCatalog       = "brainkit.core.reference_catalog"
	CapabilityToolRegistry           = "brainkit.core.tool_registry"
	CapabilityProviderRegistry       = "brainkit.core.provider_registry"
	CapabilityRegistryMutation       = "brainkit.core.registry_mutation"
	CapabilityKitStore               = "brainkit.core.kit_store"
	CapabilityMetricsSnapshot        = "brainkit.core.metrics_snapshot"
	CapabilityHealthSnapshot         = "brainkit.core.health_snapshot"
	CapabilityLifecycleDebugRegistry = "brainkit.core.lifecycle_debug_registry"
	CapabilityLifecycleDebugSnapshot = "brainkit.core.lifecycle_debug_snapshot"
	CapabilityHealthProbes           = "brainkit.core.health_probes"
	CapabilityRequestCaller          = "brainkit.core.request_caller"
	CapabilityAgentRegistry          = "brainkit.core.agent_registry"
	CapabilityMountedModules         = "brainkit.core.mounted_modules"
	CapabilityModuleLifecycle        = "brainkit.core.module_lifecycle"
	CapabilityToolCommands           = "brainkit.core.tool_commands"
	CapabilityRuntimeControl         = "brainkit.core.runtime_control"
	CapabilityTracer                 = "brainkit.core.tracer"
	CapabilityAuditRecorder          = "brainkit.core.audit_recorder"
	CapabilityReportError            = "brainkit.core.report_error"
	CapabilityPluginCheckerLease     = "brainkit.core.plugin_checker_lease"
	CapabilityPluginRestarterLease   = "brainkit.core.plugin_restarter_lease"
	CapabilityJSRuntimeHost          = "brainkit.core.jsruntime_host"
	CapabilityEnableJSRuntime        = "brainkit.core.enable_js_runtime"
	CapabilityHasJSRuntime           = "brainkit.core.has_js_runtime"
)

// PluginChecker is the narrow capability a plugins module exposes so package
// deployment can validate `requires.plugins` without depending on the plugins
// implementation.
type PluginChecker interface {
	IsPluginRunning(name string) bool
}

// ReferenceEntry describes one embedded reference item exposed through the
// reference catalog capability.
type ReferenceEntry struct {
	Name        string
	Kind        string
	Description string
	Size        int
	Parts       []string
}

// ReferenceCatalog is the core reference corpus capability consumed by
// modules/reference.
type ReferenceCatalog interface {
	GetReference(name string) (string, error)
	ListReferences() []ReferenceEntry
}

// HealthProbes is the neutral capability for HTTP/lifecycle modules that need
// liveness and readiness without access to the concrete runtime.
type HealthProbes interface {
	Alive(context.Context) bool
	Ready(context.Context) bool
}

// ProbeRunner runs provider/storage/vector probe sweeps under caller-owned
// lifecycle cancellation. Modules should use this instead of raw callback
// functions so hot-unmount can cancel and wait for in-flight probe work.
type ProbeRunner interface {
	ProbeAll(context.Context)
}

// ProbeRunnerFunc adapts a function to ProbeRunner.
type ProbeRunnerFunc func(context.Context)

// ProbeAll satisfies ProbeRunner.
func (f ProbeRunnerFunc) ProbeAll(ctx context.Context) {
	if f != nil {
		f(ctx)
	}
}

// ProviderSecretRefresher refreshes provider runtime state after a secret has
// rotated. Implementations must update the provider source of truth and return
// errors from any active runtime cache refresh instead of silently dropping
// them.
type ProviderSecretRefresher interface {
	RefreshProviderSecret(context.Context, string, string) error
}

// RegistryMutationManager owns live provider/storage/vector mutations that can
// affect runtime resources. Command modules should use this instead of
// composing raw provider registries, storage managers, and runtime cache
// invalidators themselves.
type RegistryMutationManager interface {
	AddRegistryProvider(context.Context, string, string, json.RawMessage) error
	RemoveRegistryProvider(context.Context, string) error
	AddRegistryStorage(context.Context, string, string, json.RawMessage) error
	RemoveRegistryStorage(context.Context, string) error
	AddRegistryVector(context.Context, string, string, json.RawMessage) error
	RemoveRegistryVector(context.Context, string) error
}

// RequestCaller is the neutral request/reply capability for modules that need
// to call arbitrary bus commands. The default implementation is the SDK
// shared-inbox caller, so replies are routed asynchronously by correlation ID
// rather than by creating a subscription per call.
type RequestCaller = sdk.RequestCaller

// CapabilityHost manages named runtime capabilities exposed by modules.
type CapabilityHost interface {
	Provide(context.Context, string, any) (Handle, error)
	Get(string) (any, bool)
	Require(string) (any, error)
}

// LeaseFunc installs a module-owned runtime hook and returns a closeable lease
// that detaches that exact hook when the module scope closes.
type LeaseFunc[T any] func(context.Context, T) (Handle, error)

// Capability returns a capability value only when it is present and has the
// requested type.
func Capability[T any](host Host, name string) (T, bool) {
	var zero T
	if host == nil {
		return zero, false
	}
	value, ok := host.Capabilities().Get(name)
	if !ok {
		return zero, false
	}
	typed, ok := value.(T)
	if !ok {
		return zero, false
	}
	return typed, true
}

// RequireCapability returns a typed capability or a descriptive error.
func RequireCapability[T any](host Host, name string) (T, error) {
	var zero T
	if host == nil {
		return zero, fmt.Errorf("capability %q requires a host", name)
	}
	value, err := host.Capabilities().Require(name)
	if err != nil {
		return zero, err
	}
	typed, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("capability %q has type %T", name, value)
	}
	return typed, nil
}

// CapabilityRegistry is a simple in-process CapabilityHost.
type CapabilityRegistry struct {
	mu     sync.RWMutex
	values map[string]any
}

// NewCapabilityRegistry creates an empty capability registry.
func NewCapabilityRegistry() *CapabilityRegistry {
	return &CapabilityRegistry{values: map[string]any{}}
}

// Provide registers a capability until the returned handle is closed.
func (r *CapabilityRegistry) Provide(_ context.Context, name string, value any) (Handle, error) {
	if name == "" {
		return nil, fmt.Errorf("capability name is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.values[name]; exists {
		return nil, fmt.Errorf("capability %q already provided", name)
	}
	r.values[name] = value
	return HandleFunc(func(context.Context) error {
		r.mu.Lock()
		delete(r.values, name)
		r.mu.Unlock()
		return nil
	}), nil
}

// Get returns a capability by name.
func (r *CapabilityRegistry) Get(name string) (any, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	value, ok := r.values[name]
	return value, ok
}

// Require returns a capability or an error when it is absent.
func (r *CapabilityRegistry) Require(name string) (any, error) {
	value, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("capability %q is required", name)
	}
	return value, nil
}
