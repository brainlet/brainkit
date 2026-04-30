package brainkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	coretracing "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk"
)

// Mount hot-mounts a module into a running Kit.
func (k *Kit) Mount(ctx context.Context, mod bkmodule.Module) error {
	if mod == nil {
		return fmt.Errorf("brainkit: module is nil")
	}
	id := mod.ID()
	if id == "" {
		return fmt.Errorf("brainkit: module ID is required")
	}
	for _, dep := range moduleDependencies(mod) {
		if dep == "" || dep == id {
			continue
		}
		if _, mounted := k.Module(dep); mounted {
			continue
		}
		depMod, err := buildRegisteredModule(dep, "")
		if err != nil {
			return err
		}
		if err := k.Mount(ctx, depMod); err != nil {
			if _, mounted := k.Module(dep); !mounted {
				return err
			}
		}
	}
	if id != "jsruntime" && moduleNeedsJSRuntime(mod) {
		if _, mounted := k.Module("jsruntime"); !mounted {
			jsmod, err := buildRegisteredModule("jsruntime", "")
			if err != nil {
				return err
			}
			if err := k.Mount(ctx, jsmod); err != nil {
				if _, mounted := k.Module("jsruntime"); !mounted {
					return err
				}
			}
		}
	}

	k.mountMu.Lock()
	if k.mounted == nil {
		k.mounted = map[string]bkmodule.Scope{}
	}
	if k.modules == nil {
		k.modules = map[string]bkmodule.Module{}
	}
	if _, exists := k.mounted[id]; exists {
		k.mountMu.Unlock()
		return fmt.Errorf("brainkit: module %q already mounted", id)
	}
	scope := bkmodule.NewScope(id)
	k.mounted[id] = scope
	k.mountMu.Unlock()

	host := &moduleHost{k: k, scope: scope}
	if err := mod.Mount(ctx, host); err != nil {
		_ = scope.Close(ctx)
		k.mountMu.Lock()
		delete(k.mounted, id)
		k.mountMu.Unlock()
		return fmt.Errorf("brainkit: mount %q: %w", id, err)
	}
	k.mountMu.Lock()
	k.modules[id] = mod
	k.mountMu.Unlock()
	return nil
}

// Unmount closes all resources owned by a mounted module.
func (k *Kit) Unmount(ctx context.Context, id string) error {
	k.mountMu.Lock()
	scope, ok := k.mounted[id]
	if ok {
		delete(k.mounted, id)
		delete(k.modules, id)
	}
	k.mountMu.Unlock()
	if !ok {
		return fmt.Errorf("brainkit: module %q is not mounted", id)
	}
	return scope.Close(ctx)
}

func (k *Kit) closeMounted(ctx context.Context) error {
	k.mountMu.Lock()
	ids := make([]string, 0, len(k.mounted))
	for id := range k.mounted {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	scopes := make([]bkmodule.Scope, 0, len(ids))
	for i := len(ids) - 1; i >= 0; i-- {
		id := ids[i]
		scopes = append(scopes, k.mounted[id])
		delete(k.mounted, id)
		delete(k.modules, id)
	}
	k.mountMu.Unlock()

	var err error
	for _, scope := range scopes {
		err = errors.Join(err, scope.Close(ctx))
	}
	return err
}

type moduleHost struct {
	k     *Kit
	scope bkmodule.Scope
}

func (h *moduleHost) Scope() bkmodule.Scope          { return h.scope }
func (h *moduleHost) Runtime() sdk.Runtime           { return h.k.runtime() }
func (h *moduleHost) Caller() *sdk.Caller            { return h.k.kernel.Caller() }
func (h *moduleHost) Messages() bkmodule.MessageHost { return &kitMessageHost{k: h.k, scope: h.scope} }
func (h *moduleHost) Commands() bkmodule.CommandHost { return &kitCommandHost{k: h.k, scope: h.scope} }
func (h *moduleHost) Tools() bkmodule.ToolHost       { return &kitToolHost{k: h.k, scope: h.scope} }
func (h *moduleHost) Logger() *slog.Logger           { return h.k.kernel.Logger() }
func (h *moduleHost) Store() any                     { return h.k.kernel.Store() }
func (h *moduleHost) Capabilities() bkmodule.CapabilityHost {
	return &kitCapabilityHost{k: h.k, scope: h.scope}
}

type kitMessageHost struct {
	k     *Kit
	scope bkmodule.Scope
}

func (h *kitMessageHost) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return h.k.runtime().PublishRaw(ctx, topic, payload)
}

func (h *kitMessageHost) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (bkmodule.Handle, error) {
	cancel, err := h.k.runtime().SubscribeRaw(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	handle := bkmodule.HandleFunc(func(context.Context) error {
		cancel()
		return nil
	})
	h.scope.Defer(handle.Close)
	return handle, nil
}

type kitCommandHost struct {
	k     *Kit
	scope bkmodule.Scope
}

func (h *kitCommandHost) Handle(spec bkmodule.CommandSpec) (bkmodule.Handle, error) {
	handle, err := h.k.kernel.MountCommand(context.Background(), spec)
	if err != nil {
		return nil, err
	}
	h.scope.Defer(handle.Close)
	return handle, nil
}

func (h *kitCommandHost) Has(topic string) bool { return h.k.kernel.HasCommand(topic) }

type kitToolHost struct {
	k     *Kit
	scope bkmodule.Scope
}

func (h *kitToolHost) Register(_ context.Context, spec bkmodule.ToolSpec) (bkmodule.Handle, error) {
	if spec.Executor == nil {
		return nil, fmt.Errorf("brainkit: tool %q executor is required", spec.Name)
	}
	tool := toolreg.RegisteredTool{
		Name:        spec.Name,
		ShortName:   spec.ShortName,
		Owner:       spec.Owner,
		Package:     spec.Package,
		Version:     spec.Version,
		Description: spec.Description,
		InputSchema: spec.InputSchema,
		Local:       spec.Local,
		Executor: &toolreg.GoFuncExecutor{
			Fn: spec.Executor.Call,
		},
	}
	if err := h.k.kernel.Tools.Register(tool); err != nil {
		return nil, err
	}
	handle := bkmodule.HandleFunc(func(context.Context) error {
		h.k.kernel.Tools.Unregister(spec.Name)
		return nil
	})
	h.scope.Defer(handle.Close)
	return handle, nil
}

type kitCapabilityHost struct {
	k     *Kit
	scope bkmodule.Scope
}

func (h *kitCapabilityHost) Provide(ctx context.Context, name string, value any) (bkmodule.Handle, error) {
	handle, err := h.k.caps.Provide(ctx, name, value)
	if err != nil {
		return nil, err
	}
	h.scope.Defer(handle.Close)
	return handle, nil
}

func (h *kitCapabilityHost) Get(name string) (any, bool) {
	if value, ok := h.k.caps.Get(name); ok {
		return value, true
	}
	return h.coreCapability(name)
}

func (h *kitCapabilityHost) Require(name string) (any, error) {
	if value, ok := h.Get(name); ok {
		return value, nil
	}
	return nil, fmt.Errorf("capability %q is required", name)
}

func (h *kitCapabilityHost) coreCapability(name string) (any, bool) {
	switch name {
	case bkmodule.CapabilityRuntimeID:
		return RuntimeID(), true
	case bkmodule.CapabilityNamespace:
		return h.k.kernel.Namespace(), true
	case bkmodule.CapabilityCallerID:
		return h.k.kernel.CallerID(), true
	case bkmodule.CapabilityPresenceTransport:
		return transport.Presence(h.k.kernel.Remote()), true
	case bkmodule.CapabilityPluginChecker:
		return func() bkmodule.PluginChecker { return h.k.kernel.PluginChecker() }, true
	case bkmodule.CapabilitySetScheduleHandler:
		return func(handler types.ScheduleHandler) { h.k.kernel.SetScheduleHandler(handler) }, true
	case bkmodule.CapabilitySetAuditStore:
		return func(store auditpkg.Store) { h.k.kernel.SetAuditStore(store) }, true
	case bkmodule.CapabilitySetAuditVerbosity:
		return func(verbosity auditpkg.Verbosity) { h.k.kernel.SetAuditVerbosity(verbosity) }, true
	case bkmodule.CapabilitySetTraceStore:
		return func(store coretracing.TraceStore) { h.k.kernel.SetTraceStore(store) }, true
	case bkmodule.CapabilityProbeAll:
		return func() { h.k.kernel.ProbeAll() }, true
	case bkmodule.CapabilityTransportKind:
		return h.k.kernel.TransportKind(), true
	case bkmodule.CapabilitySecretStore:
		if store := h.k.kernel.SecretStore(); store != nil {
			return store, true
		}
		return nil, false
	case bkmodule.CapabilityPluginRestarter:
		return func() any { return h.k.kernel.PluginRestarter() }, true
	case bkmodule.CapabilityRefreshProviderSecret:
		return func(name, value string) { h.k.kernel.RefreshProviderIfSecret(name, value) }, true
	case bkmodule.CapabilityShutdownSignal:
		return h.k.kernel.ShutdownSignal(), true
	case bkmodule.CapabilityRemoteClient:
		return h.k.kernel.Remote(), true
	case bkmodule.CapabilityReferenceCatalog:
		return referenceCatalogCapability{}, true
	case bkmodule.CapabilityToolRegistry:
		return h.k.kernel.Tools, true
	case bkmodule.CapabilityProviderRegistry:
		return h.k.kernel.ProviderRegistry(), true
	case bkmodule.CapabilityStorageManager:
		return h.k.kernel.StorageManager(), true
	case bkmodule.CapabilityMetricsSnapshot:
		return func() any { return h.k.kernel.Metrics() }, true
	case bkmodule.CapabilityHealthSnapshot:
		return func(ctx context.Context) any { return h.k.kernel.Health(ctx) }, true
	case bkmodule.CapabilityAgentRegistry:
		return h.k.kernel.AgentsDomain(), true
	case bkmodule.CapabilityToolCommands:
		return h.k.kernel.ToolsDomain(), true
	case bkmodule.CapabilityRuntimeControl:
		return h.k.kernel, true
	case bkmodule.CapabilityTracer:
		return h.k.kernel.Tracer(), true
	case bkmodule.CapabilityAuditRecorder:
		return h.k.kernel.Audit(), true
	case bkmodule.CapabilityReportError:
		return func(err error, errorCtx types.ErrorContext) {
			h.k.kernel.ReportError(err, errorCtx)
		}, true
	case bkmodule.CapabilitySetPluginChecker:
		return func(checker bkmodule.PluginChecker) {
			h.k.kernel.SetPluginChecker(checker)
		}, true
	case bkmodule.CapabilitySetPluginRestarter:
		return func(restarter PluginRestarter) {
			h.k.kernel.SetPluginRestarter(restarter)
		}, true
	case bkmodule.CapabilityJSRuntimeHost:
		return h.k.kernel, true
	default:
		return nil, false
	}
}
