package brainkit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/internal/transport"
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
