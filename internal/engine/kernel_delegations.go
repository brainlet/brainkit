package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/secrets"
	tracingpkg "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	provreg "github.com/brainlet/brainkit/modulehost/providerhost/providerreg"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// --- sdk.Runtime implementation ---

// Namespace returns the runtime namespace.
func (k *Kernel) Namespace() string { return k.namespace }

// CallerID returns the runtime identity.
func (k *Kernel) CallerID() string { return k.callerID }

// Remote returns the transport-level client. Used by (*Kit).PresenceTransport
// to expose cluster-wide publish/subscribe to brainkit.Modules (e.g. discovery)
// without leaking the full transport surface.
func (k *Kernel) Remote() *transport.RemoteClient {
	if k.transportHost == nil {
		return nil
	}
	return k.transportHost.Remote()
}

// SetScheduleHandler attaches the scheduler. The schedules module calls this
// during its Init; bridges_scheduling.go and the schedule.* bus commands
// dispatch through this handler. Passing nil (e.g. on module Close) detaches
// it and future bridge calls throw NOT_CONFIGURED.
func (k *Kernel) SetScheduleHandler(h types.ScheduleHandler) { k.scheduleHandler = h }

// HasCommand reports whether the given topic is a registered bus command
// (and therefore reserved for request/response routing). Schedules reject
// command topics — scheduling a command would bypass reply plumbing.
func (k *Kernel) HasCommand(topic string) bool { return k.catalog.HasCommand(topic) }

// MountCommand live-mounts a module command after the router exists.
func (k *Kernel) MountCommand(ctx context.Context, spec bkmodule.CommandSpec) (bkmodule.Handle, error) {
	cmd := moduleCommand(spec)
	if cmd.topic == "" {
		return nil, fmt.Errorf("command topic is required")
	}
	k.mu.Lock()
	if err := k.catalog.Add(cmd); err != nil {
		k.mu.Unlock()
		return nil, err
	}
	k.mu.Unlock()

	handle, err := k.transportHost.RegisterCommand(ctx, transport.RawCommandBinding{
		Name:  cmd.topic,
		Topic: cmd.topic,
		Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
			return cmd.invokeKernel(ctx, k, payload)
		},
	})
	if err != nil {
		k.mu.Lock()
		k.catalog.Remove(cmd.topic)
		k.mu.Unlock()
		return nil, err
	}

	return bkmodule.HandleFunc(func(ctx context.Context) error {
		k.mu.Lock()
		k.catalog.Remove(cmd.topic)
		k.mu.Unlock()
		return handle.Stop(ctx)
	}), nil
}

// SetAuditStore attaches (or detaches) the Recorder's underlying store.
// The audit module calls this during mount; without a store the Recorder
// is a no-op.
func (k *Kernel) SetAuditStore(s auditpkg.Store) { k.audit.SetStore(s) }

// SetAuditVerbosity flips the Recorder between normal and verbose tiers.
func (k *Kernel) SetAuditVerbosity(v auditpkg.Verbosity) { k.audit.SetVerbosity(v) }

// Audit returns the central Recorder for modules that need to record
// events directly (e.g. the plugins module's WS server recording
// plugin.registered / health.changed).
func (k *Kernel) Audit() *auditpkg.Recorder { return k.audit }

// SecretStore exposes the encrypted secret store for modules that need
// to resolve $secret: references at startup.
func (k *Kernel) SecretStore() secrets.SecretStore { return k.secretStore }

// Store exposes the kit's configured KitStore (nil if none).
func (k *Kernel) Store() types.KitStore { return k.config.Store }

// Tracer exposes the runtime tracer. Modules use this to mark plugin
// tool invocations and other cross-cutting spans.
func (k *Kernel) Tracer() *tracingpkg.Tracer { return k.tracer }

// ShutdownSignal returns a channel that closes when the kernel is
// tearing down. Modules with long-running goroutines (plugin restart
// backoff) select on this to exit promptly.
func (k *Kernel) ShutdownSignal() <-chan struct{} { return k.shutdownCtx.Done() }

// TransportKind returns the normalized transport type ("memory",
// "embedded", "nats", "amqp", "redis"). Modules use this to refuse
// configurations that the transport can't support — e.g. the plugins
// module requires real networking and refuses "memory".
func (k *Kernel) TransportKind() string {
	if k.transportHost == nil {
		return ""
	}
	return k.transportHost.TransportKind()
}

// SetPluginChecker installs the module-side PluginChecker used by
// package-deploy's `Requires.plugins` gate. Pass nil to detach.
func (k *Kernel) SetPluginChecker(pc bkmodule.PluginChecker) { k.pluginChecker = pc }

// PluginChecker returns the active plugin-presence gate for modules that
// validate plugin dependencies. Nil means no plugins module is mounted.
func (k *Kernel) PluginChecker() bkmodule.PluginChecker { return k.pluginChecker }

// SetPluginRestarter installs the module-side PluginRestarter used by the
// secrets module for rotation-driven plugin restart. Pass nil to detach.
func (k *Kernel) SetPluginRestarter(r PluginRestarter) {
	k.pluginRestarter = r
}

// PluginRestarter returns the active plugin restarter, if the plugins module
// is mounted. Nil means there is no plugin module to restart.
func (k *Kernel) PluginRestarter() PluginRestarter { return k.pluginRestarter }

// Logger returns the structured logger.
func (k *Kernel) Logger() *slog.Logger { return k.logger }

// ProviderRegistry exposes the shared provider/storage/vector registry to
// module hosts and JS runtime bridges. Public runtime administration goes
// through modules/registry bus messages.
func (k *Kernel) ProviderRegistry() *provreg.ProviderRegistry { return k.providerHost.Registry() }

// CallTool invokes a registered tool through the core registry.
func (k *Kernel) CallTool(ctx context.Context, req bkmodule.ToolCallRequest) (*bkmodule.ToolCallResponse, error) {
	return k.toolsDomain.Call(ctx, req)
}

// ResolveTool returns registration metadata for a tool.
func (k *Kernel) ResolveTool(ctx context.Context, req bkmodule.ToolResolveRequest) (*bkmodule.ToolResolveResponse, error) {
	return k.toolsDomain.Resolve(ctx, req)
}

// ListTools returns registered tools matching the optional SDK filter.
func (k *Kernel) ListTools(ctx context.Context, req bkmodule.ToolListRequest) (*bkmodule.ToolListResponse, error) {
	return k.toolsDomain.List(ctx, req)
}

// ClusterPeers returns this runtime's cluster identity. Multi-peer discovery is
// module-owned; this core snapshot is the local control-plane identity data.
func (k *Kernel) ClusterPeers(_ context.Context) ([]bkmodule.ClusterPeerInfo, error) {
	return []bkmodule.ClusterPeerInfo{{
		ClusterID: k.config.ClusterID,
		RuntimeID: k.config.RuntimeID,
		Namespace: k.config.Namespace,
		CallerID:  k.config.CallerID,
		StartedAt: k.startedAt.Format("2006-01-02T15:04:05Z07:00"),
	}}, nil
}

// --- Deployment delegation ---

// ListResources returns all tracked resources, optionally filtered by type.
func (k *Kernel) ListResources(resourceType ...string) ([]types.ResourceInfo, error) {
	if k.jsRuntime == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.ListResources(resourceType...)
}

// ResourcesFrom returns all resources created by a specific .ts file.
func (k *Kernel) ResourcesFrom(filename string) ([]types.ResourceInfo, error) {
	if k.jsRuntime == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.ResourcesFrom(filename)
}

// TeardownFile removes all resources created by a specific .ts file.
func (k *Kernel) TeardownFile(filename string) (int, error) {
	if k.jsRuntime == nil {
		return 0, &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.TeardownFile(filename)
}

// RemoveResource removes a specific resource by type and ID.
func (k *Kernel) RemoveResource(resourceType, id string) error {
	if k.jsRuntime == nil {
		return &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.RemoveResource(resourceType, id)
}

// --- Eval delegation ---

// evalDomain marshals a request into JS globals and evaluates code atomically.
// Replaces per-domain evalAI/evalMemory/evalVector/evalWorkflow methods.
func (k *Kernel) evalDomain(ctx context.Context, req any, filename, code string) (json.RawMessage, error) {
	reqJSON, _ := json.Marshal(req)
	wrappedCode := fmt.Sprintf(`
		globalThis.__pending_req = %s;
		%s
	`, string(reqJSON), code)
	resultJSON, err := k.EvalTS(ctx, filename, wrappedCode)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resultJSON), nil
}

// EvalTS runs .ts-style code with brainkit infrastructure imports destructured.
func (k *Kernel) EvalTS(ctx context.Context, filename, code string) (string, error) {
	if k.jsRuntime == nil {
		return "", &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.EvalTS(ctx, filename, code)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kernel) EvalModule(ctx context.Context, filename, code string) (string, error) {
	if k.jsRuntime == nil {
		return "", &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	return k.jsRuntime.EvalModule(ctx, filename, code)
}

// ReportError forwards a non-fatal error through the Kernel's configured
// ErrorHandler (no-op if none is configured). Used by modules.
func (k *Kernel) ReportError(err error, ctx types.ErrorContext) {
	types.InvokeErrorHandler(k.config.ErrorHandler, err, ctx)
}

// SetTraceStore attaches a trace store to the Kernel's tracer. Used by
// modules (e.g. tracing) to install durable storage at mount time.
func (k *Kernel) SetTraceStore(store types.TraceStore) {
	k.tracer.SetStore(store)
}

// currentDeploymentSource returns the deployment source currently executing on the JS thread.
// Used for tracing span attribution and audit source tracking.
func (k *Kernel) currentDeploymentSource() string {
	if k.jsRuntime == nil {
		return ""
	}
	return k.jsRuntime.CurrentSource()
}

func (k *Kernel) setCurrentSource(source string) {
	if k.jsRuntime == nil {
		return
	}
	k.jsRuntime.SetCurrentSource(source)
}
