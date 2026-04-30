package module

import (
	"context"
	"fmt"
	"sync"
)

const (
	CapabilityRuntimeID             = "brainkit.core.runtime_id"
	CapabilityNamespace             = "brainkit.core.namespace"
	CapabilityCallerID              = "brainkit.core.caller_id"
	CapabilityPresenceTransport     = "brainkit.core.presence_transport"
	CapabilityDeployer              = "brainkit.core.deployer"
	CapabilityTSRunner              = "brainkit.core.ts_runner"
	CapabilityEvalRuntime           = "brainkit.core.eval_runtime"
	CapabilityPluginChecker         = "brainkit.core.plugin_checker"
	CapabilitySetScheduleHandler    = "brainkit.core.set_schedule_handler"
	CapabilitySetAuditStore         = "brainkit.core.set_audit_store"
	CapabilitySetAuditVerbosity     = "brainkit.core.set_audit_verbosity"
	CapabilitySetTraceStore         = "brainkit.core.set_trace_store"
	CapabilityProbeAll              = "brainkit.core.probe_all"
	CapabilityCallJS                = "brainkit.core.call_js"
	CapabilityHarnessRuntime        = "brainkit.core.harness_runtime"
	CapabilityTransportKind         = "brainkit.core.transport_kind"
	CapabilitySecretStore           = "brainkit.core.secret_store"
	CapabilityPluginRestarter       = "brainkit.core.plugin_restarter"
	CapabilityRefreshProviderSecret = "brainkit.core.refresh_provider_secret"
	CapabilityShutdownSignal        = "brainkit.core.shutdown_signal"
	CapabilityRemoteClient          = "brainkit.core.remote_client"
	CapabilityReferenceCatalog      = "brainkit.core.reference_catalog"
	CapabilityToolRegistry          = "brainkit.core.tool_registry"
	CapabilityProviderRegistry      = "brainkit.core.provider_registry"
	CapabilityStorageManager        = "brainkit.core.storage_manager"
	CapabilityMetricsSnapshot       = "brainkit.core.metrics_snapshot"
	CapabilityHealthSnapshot        = "brainkit.core.health_snapshot"
	CapabilityAgentRegistry         = "brainkit.core.agent_registry"
	CapabilityToolCommands          = "brainkit.core.tool_commands"
	CapabilityRuntimeControl        = "brainkit.core.runtime_control"
	CapabilityTracer                = "brainkit.core.tracer"
	CapabilityAuditRecorder         = "brainkit.core.audit_recorder"
	CapabilityReportError           = "brainkit.core.report_error"
	CapabilitySetPluginChecker      = "brainkit.core.set_plugin_checker"
	CapabilitySetPluginRestarter    = "brainkit.core.set_plugin_restarter"
	CapabilityJSRuntimeHost         = "brainkit.core.jsruntime_host"
	CapabilityEnableJSRuntime       = "brainkit.core.enable_js_runtime"
	CapabilityHasJSRuntime          = "brainkit.core.has_js_runtime"
)

// PluginChecker is the narrow capability a plugins module exposes so package
// deployment can validate `requires.plugins` without depending on the plugins
// implementation.
type PluginChecker interface {
	IsPluginRunning(name string) bool
}

// CapabilityHost manages named runtime capabilities exposed by modules.
type CapabilityHost interface {
	Provide(context.Context, string, any) (Handle, error)
	Get(string) (any, bool)
	Require(string) (any, error)
}

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
