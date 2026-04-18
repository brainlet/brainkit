package brainkit

import (
	"context"
	"encoding/json"
	"log/slog"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/deploy"
	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
)

// Module is an opt-in kernel extension. Modules register bus commands and
// manage their own lifecycle; they run only when included in Config.Modules.
//
// This is the public contract — modules live outside internal/engine and
// satisfy it without importing internal packages. Legacy internal modules
// keep satisfying engine.Module; the module init loop dispatches to whichever
// interface is present.
type Module interface {
	Name() string
	Init(k *Kit) error
	Close() error
}

// ModuleStatus reports a module's maturity. Modules can optionally report
// their status for CLI listing / docs.
type ModuleStatus = string

const (
	ModuleStatusStable ModuleStatus = "stable"
	ModuleStatusBeta   ModuleStatus = "beta"
	ModuleStatusWIP    ModuleStatus = "wip"
)

// StatusReporter is implemented by modules that expose a maturity tag.
type StatusReporter interface {
	Status() ModuleStatus
}

// CommandSpec is the opaque handle produced by Command. Pass it to
// Kit.RegisterCommand to add the command to the kit's bus catalog.
type CommandSpec = engine.CommandSpec

// Command builds a CommandSpec from a typed handler. The handler only sees
// the context and decoded request; capture any Kit / Module state via
// closure.
//
//	k.RegisterCommand(brainkit.Command(func(ctx context.Context, req sdk.McpListToolsMsg) (*sdk.McpListToolsResp, error) {
//	    return m.domain.ListTools(ctx, req)
//	}))
func Command[Req sdk.BrainkitMessage, Resp any](handler func(context.Context, Req) (*Resp, error)) CommandSpec {
	return engine.MakeCommand(handler)
}

// RegisterCommand adds a bus command to the Kit's per-instance catalog.
// Intended for Module.Init; panics on duplicate topic.
func (k *Kit) RegisterCommand(spec CommandSpec) {
	k.kernel.RegisterCommand(spec)
}

// Module looks up a Kit-scoped module by Name. Used for cross-module
// coordination (e.g. WithCallTo consults the topology module when
// present to resolve peer names). Returns (nil, false) when the
// module is absent.
func (k *Kit) Module(name string) (Module, bool) {
	for _, m := range k.modules {
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}

// RegisterRawTool registers a pre-built RegisteredTool with the Kit's tool
// registry. Modules use this to surface tools whose executor isn't a typed
// Go function (e.g. MCP tools that proxy to an external server).
func (k *Kit) RegisterRawTool(t RegisteredTool) error {
	return k.kernel.Tools.Register(t)
}

// ReportError forwards a non-fatal error through the Kit's ErrorHandler
// (no-op if one isn't configured).
func (k *Kit) ReportError(err error, ctx ErrorContext) {
	k.kernel.ReportError(err, ctx)
}

// CallJS invokes a named JS function on the Kit's runtime and decodes its
// JSON result. Modules use this to dispatch into runtime-side helpers
// registered on globalThis (e.g. __brainkit.workflow.start).
func (k *Kit) CallJS(ctx context.Context, fn string, args any) (json.RawMessage, error) {
	return k.kernel.CallJS(ctx, fn, args)
}

// ProbeAll probes every registered AI provider, vector store, and storage
// backend. Modules use this to trigger periodic or on-demand probing.
func (k *Kit) ProbeAll() {
	k.kernel.ProbeAll()
}

// SetTraceStore attaches a durable trace store to the Kit's tracer. The
// tracing module uses this during Init to promote the default in-memory
// ring buffer to persistent storage.
func (k *Kit) SetTraceStore(store TraceStore) {
	k.kernel.SetTraceStore(store)
}

// Namespace returns the Kit's bus namespace (message topic scoping).
func (k *Kit) Namespace() string { return k.kernel.Namespace() }

// CallerID returns the Kit's identity stamped onto outbound bus messages.
func (k *Kit) CallerID() string { return k.kernel.CallerID() }

// PresenceTransport exposes cluster-wide publish/subscribe on non-namespaced
// topics. Modules such as discovery use this for presence announcements.
// The concrete type is transport.Presence — a narrow, purpose-built interface
// that modules can import without pulling the full transport surface.
func (k *Kit) PresenceTransport() transport.Presence { return k.kernel.Remote() }

// Logger returns the Kit's structured logger (slog.Default() by default).
// Modules use this to emit runtime diagnostics (e.g. "restored N schedules").
func (k *Kit) Logger() *slog.Logger { return k.kernel.Logger() }

// SetScheduleHandler installs the scheduler. Owned by the schedules module:
// during its Init, the module calls this to route the QuickJS
// bus.schedule / bus.unschedule bridges and the schedule.* bus commands
// into its own Scheduler. Nil detaches the handler (module Close).
func (k *Kit) SetScheduleHandler(h types.ScheduleHandler) { k.kernel.SetScheduleHandler(h) }

// HasCommand reports whether a topic is a registered bus command (reserved
// for request/response routing). Modules that accept topic arguments
// (e.g. schedules) use this to reject command topics.
func (k *Kit) HasCommand(topic string) bool { return k.kernel.HasCommand(topic) }

// SetAuditStore attaches a store to the Kit's central audit Recorder. The
// Recorder always exists; without a store, Record calls no-op. The audit
// module calls this during Init so every subsystem's Record calls start
// persisting. Pass nil to detach.
func (k *Kit) SetAuditStore(s AuditStore) { k.kernel.SetAuditStore(s) }

// SetAuditVerbosity switches the Recorder between normal and verbose
// tiers. Owned by the audit module.
func (k *Kit) SetAuditVerbosity(v AuditVerbosity) { k.kernel.SetAuditVerbosity(v) }

// Audit returns the central event recorder. Modules that record events
// from non-bus code paths (e.g. the plugins module's WS server) call
// methods on this directly.
func (k *Kit) Audit() *auditpkg.Recorder { return k.kernel.Audit() }

// SecretStore exposes the encrypted secret store. The plugins module
// uses it to resolve $secret:NAME references in plugin env vars.
func (k *Kit) SecretStore() secrets.SecretStore { return k.kernel.SecretStore() }

// Store returns the configured KitStore (nil if unset). Modules that
// persist their own state (plugins, schedules) read it here.
func (k *Kit) Store() types.KitStore { return k.kernel.Store() }

// Tools exposes the shared tool registry. Modules register plugin or
// subsystem tools through this.
func (k *Kit) Tools() *tools.ToolRegistry { return k.kernel.Tools }

// Tracer returns the runtime tracer for modules that need to mark
// cross-cutting spans (plugin tool calls, workflow steps).
func (k *Kit) Tracer() *tracing.Tracer { return k.kernel.Tracer() }

// Remote returns the transport-level client for modules that need raw
// publish/subscribe with metadata (plugins WS server, presence).
func (k *Kit) Remote() *transport.RemoteClient { return k.kernel.Remote() }

// ShutdownSignal is a channel that closes when the Kit is tearing down.
// Long-running module goroutines select on it to exit promptly.
func (k *Kit) ShutdownSignal() <-chan struct{} { return k.kernel.ShutdownSignal() }

// TransportKind returns the normalized transport type ("memory",
// "embedded", "nats", "amqp", "redis"). Modules use this to refuse
// configurations that the transport can't support — e.g. plugins
// module requires real networking and refuses "memory".
func (k *Kit) TransportKind() string { return k.kernel.TransportKind() }

// SetPluginChecker installs the module-side plugin-presence gate used
// by package deploys with `requires.plugins`. Owned by the plugins
// module (nil when the module is absent → no plugin requirements can
// be satisfied).
func (k *Kit) SetPluginChecker(pc deploy.PluginChecker) { k.kernel.SetPluginChecker(pc) }

// SetPluginRestarter installs the module-side plugin restarter used by
// secrets rotation to restart plugins whose env refers to a rotated
// secret. Owned by the plugins module.
func (k *Kit) SetPluginRestarter(r PluginRestarter) { k.kernel.SetPluginRestarter(r) }

// PluginRestarter is the narrow surface the plugins module exposes to
// secret rotation — list + restart. Re-exported from engine so Kit
// callers can name it.
type PluginRestarter = engine.PluginRestarter
