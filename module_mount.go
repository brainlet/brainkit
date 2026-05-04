package brainkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	coretracing "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	plugincap "github.com/brainlet/brainkit/modulecap/plugin"
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
	desc := bkmodule.DescribeModule(mod)
	if k.moduleMounted(id) {
		return fmt.Errorf("brainkit: module %q already mounted", id)
	}
	if err := k.preflightModuleMount(desc); err != nil {
		return err
	}
	for _, dep := range desc.Requires {
		if dep == "" || dep == id {
			continue
		}
		if _, mounted := k.Module(dep); mounted {
			continue
		}
		depMod, err := buildRegisteredModule(dep, k.fsRoot)
		if err != nil {
			return err
		}
		if err := k.Mount(ctx, depMod); err != nil {
			if _, mounted := k.Module(dep); !mounted {
				return err
			}
		}
	}
	if id != "jsruntime" && descriptorDependsOn(desc, "jsruntime") {
		if _, mounted := k.Module("jsruntime"); !mounted {
			jsmod, err := buildRegisteredModule("jsruntime", k.fsRoot)
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

	if k.moduleMounted(id) {
		return fmt.Errorf("brainkit: module %q already mounted", id)
	}
	if err := k.preflightModuleMount(desc); err != nil {
		return err
	}

	k.mountMu.Lock()
	if k.mounted == nil {
		k.mounted = map[string]bkmodule.Scope{}
	}
	if k.modules == nil {
		k.modules = map[string]bkmodule.Module{}
	}
	if k.descs == nil {
		k.descs = map[string]bkmodule.Descriptor{}
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
	desc = mountedModuleDescriptor(id, desc, host.commands, host.subscriptions, host.capabilities, host.resources, scope.Resources())
	k.mountMu.Lock()
	k.modules[id] = mod
	k.descs[id] = desc
	k.mountOrder = append(k.mountOrder, id)
	k.mountMu.Unlock()
	return nil
}

func (k *Kit) moduleMounted(id string) bool {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	_, mounted := k.mounted[id]
	return mounted
}

func formatRequiredCapability(cap bkmodule.CapabilityDescriptor) string {
	if cap.Type == "" {
		return cap.Name
	}
	return fmt.Sprintf("%s (%s)", cap.Name, cap.Type)
}

func (k *Kit) hasCapability(name string) bool {
	_, _, ok := k.capabilitySource(name)
	return ok
}

func (k *Kit) capabilitySource(name string) (source, provider string, available bool) {
	if k == nil || name == "" {
		return "", "", false
	}
	if _, ok := k.caps.Get(name); ok {
		provider := k.mountedCapabilityProvider(name)
		if provider == "" {
			provider = "mounted"
		}
		return "mounted", provider, true
	}
	if _, ok := (&kitCapabilityHost{k: k}).coreCapability(name); ok {
		return "core", "brainkit.core", true
	}
	return "", "", false
}

func (k *Kit) mountedCapabilityProvider(name string) string {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	for _, id := range k.mountOrder {
		desc, ok := k.descs[id]
		if ok && descriptorProvidesCapability(desc, name) {
			return id
		}
	}
	var ids []string
	for id := range k.descs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		desc := k.descs[id]
		if descriptorProvidesCapability(desc, name) {
			return id
		}
	}
	return ""
}

func descriptorProvidesCapability(desc bkmodule.Descriptor, name string) bool {
	if desc.CapabilityGroups != nil {
		for _, cap := range desc.CapabilityGroups.Provided {
			if cap.Name == name {
				return true
			}
		}
	}
	for _, cap := range desc.Capabilities {
		if cap.Name == name && cap.Direction == bkmodule.CapabilityProvided {
			return true
		}
	}
	return false
}

// Unmount closes all resources owned by a mounted module.
func (k *Kit) Unmount(ctx context.Context, id string) error {
	if dependents := k.mountedDependents(id); len(dependents) > 0 {
		return fmt.Errorf("brainkit: module %q is required by mounted module(s): %s", id, strings.Join(dependents, ", "))
	}
	k.mountMu.Lock()
	scope, ok := k.mounted[id]
	k.mountMu.Unlock()
	if !ok {
		return fmt.Errorf("brainkit: module %q is not mounted", id)
	}
	if err := scope.Close(ctx); err != nil {
		return err
	}
	k.mountMu.Lock()
	if k.mounted[id] == scope {
		delete(k.mounted, id)
		delete(k.modules, id)
		delete(k.descs, id)
		k.removeMountOrderLocked(id)
	}
	k.mountMu.Unlock()
	return nil
}

// MountedModules returns descriptors for every module currently mounted in
// this Kit. The result is a snapshot sorted by module name.
func (k *Kit) MountedModules() []bkmodule.Descriptor {
	k.mountMu.Lock()
	defer k.mountMu.Unlock()
	out := make([]bkmodule.Descriptor, 0, len(k.descs))
	for _, desc := range k.descs {
		out = append(out, desc)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func mountedModuleDescriptor(
	id string,
	desc bkmodule.Descriptor,
	mountedCommands []bkmodule.MessageDescriptor,
	mountedSubscriptions []bkmodule.MessageDescriptor,
	mountedCapabilities []bkmodule.CapabilityDescriptor,
	mountedResources []bkmodule.ResourceDescriptor,
	scopeResources []bkmodule.ResourceDescriptor,
) bkmodule.Descriptor {
	if len(mountedCommands) > 0 {
		known := map[string]struct{}{}
		for _, cmd := range desc.Commands {
			if cmd.Topic != "" {
				known[cmd.Topic] = struct{}{}
			}
		}
		for _, cmd := range mountedCommands {
			if cmd.Topic == "" {
				continue
			}
			if _, ok := known[cmd.Topic]; ok {
				continue
			}
			desc.Commands = append(desc.Commands, cmd)
			known[cmd.Topic] = struct{}{}
		}
	}
	if len(mountedSubscriptions) > 0 {
		known := map[string]struct{}{}
		for _, sub := range desc.Subscriptions {
			if sub.Topic != "" {
				known[sub.Topic] = struct{}{}
			}
		}
		for _, sub := range mountedSubscriptions {
			if sub.Topic == "" {
				continue
			}
			if _, ok := known[sub.Topic]; ok {
				continue
			}
			desc.Subscriptions = append(desc.Subscriptions, sub)
			known[sub.Topic] = struct{}{}
		}
	}
	if len(mountedCapabilities) > 0 {
		known := map[string]struct{}{}
		for _, cap := range desc.Capabilities {
			if cap.Name != "" {
				known[string(cap.Direction)+"\x00"+cap.Name] = struct{}{}
			}
		}
		for _, cap := range mountedCapabilities {
			if cap.Name == "" {
				continue
			}
			key := string(cap.Direction) + "\x00" + cap.Name
			if _, ok := known[key]; ok {
				continue
			}
			desc.Capabilities = append(desc.Capabilities, cap)
			known[key] = struct{}{}
		}
	}
	if len(mountedResources) > 0 || len(scopeResources) > 0 {
		known := map[string]struct{}{}
		for _, res := range desc.Resources {
			if res.Kind != "" && res.Name != "" {
				known[string(res.Kind)+"\x00"+res.Name] = struct{}{}
			}
		}
		for _, res := range append(append([]bkmodule.ResourceDescriptor(nil), mountedResources...), scopeResources...) {
			if res.Kind == "" || res.Name == "" {
				continue
			}
			key := string(res.Kind) + "\x00" + res.Name
			if _, ok := known[key]; ok {
				continue
			}
			desc.Resources = append(desc.Resources, res)
			known[key] = struct{}{}
		}
	}
	return bkmodule.NormalizeDescriptor(id, desc)
}

func (k *Kit) closeMounted(ctx context.Context) error {
	type mountedScope struct {
		id    string
		scope bkmodule.Scope
	}

	k.mountMu.Lock()
	order := append([]string(nil), k.mountOrder...)
	scopes := make([]mountedScope, 0, len(k.mounted))
	seen := make(map[string]struct{}, len(k.mounted))
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		scope, ok := k.mounted[id]
		if !ok {
			continue
		}
		scopes = append(scopes, mountedScope{id: id, scope: scope})
		seen[id] = struct{}{}
	}
	remaining := make([]string, 0, len(k.mounted))
	for id := range k.mounted {
		if _, ok := seen[id]; !ok {
			remaining = append(remaining, id)
		}
	}
	sort.Strings(remaining)
	for i := len(remaining) - 1; i >= 0; i-- {
		id := remaining[i]
		scopes = append(scopes, mountedScope{id: id, scope: k.mounted[id]})
	}
	k.mountMu.Unlock()

	var err error
	var closed []mountedScope
	for _, mounted := range scopes {
		if closeErr := mounted.scope.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
			continue
		}
		closed = append(closed, mounted)
	}
	if len(closed) > 0 {
		k.mountMu.Lock()
		for _, mounted := range closed {
			if k.mounted[mounted.id] != mounted.scope {
				continue
			}
			delete(k.mounted, mounted.id)
			delete(k.modules, mounted.id)
			delete(k.descs, mounted.id)
			k.removeMountOrderLocked(mounted.id)
		}
		k.mountMu.Unlock()
	}
	return err
}

func (k *Kit) removeMountOrderLocked(id string) {
	for i, mountedID := range k.mountOrder {
		if mountedID != id {
			continue
		}
		k.mountOrder = append(k.mountOrder[:i], k.mountOrder[i+1:]...)
		return
	}
}

type moduleHost struct {
	k             *Kit
	scope         bkmodule.Scope
	commands      []bkmodule.MessageDescriptor
	subscriptions []bkmodule.MessageDescriptor
	capabilities  []bkmodule.CapabilityDescriptor
	resources     []bkmodule.ResourceDescriptor
}

func (h *moduleHost) Scope() bkmodule.Scope { return h.scope }
func (h *moduleHost) Messages() bkmodule.MessageHost {
	return &kitMessageHost{k: h.k, scope: h.scope, record: h.recordSubscription}
}
func (h *moduleHost) Commands() bkmodule.CommandHost {
	return &kitCommandHost{k: h.k, scope: h.scope, record: h.recordCommand}
}
func (h *moduleHost) Tools() bkmodule.ToolHost {
	return &kitToolHost{k: h.k, scope: h.scope, record: h.recordResource}
}
func (h *moduleHost) Logger() *slog.Logger { return h.k.kernel.Logger() }
func (h *moduleHost) Capabilities() bkmodule.CapabilityHost {
	return &kitCapabilityHost{k: h.k, scope: h.scope, record: h.recordCapability}
}

func (h *moduleHost) recordCommand(spec bkmodule.CommandSpec) {
	h.commands = append(h.commands, spec.Descriptor())
}

func (h *moduleHost) recordSubscription(topic string) {
	h.subscriptions = append(h.subscriptions, bkmodule.MessageDescriptor{
		Topic: topic,
		Kind:  bkmodule.MessageKindSubscription,
	})
}

func (h *moduleHost) recordCapability(desc bkmodule.CapabilityDescriptor) {
	h.capabilities = append(h.capabilities, desc)
}

func (h *moduleHost) recordResource(desc bkmodule.ResourceDescriptor) {
	h.resources = append(h.resources, desc)
}

type kitMessageHost struct {
	k      *Kit
	scope  bkmodule.Scope
	record func(string)
}

type rawSubscriptionRuntime interface {
	SubscribeRawHandle(context.Context, string, func(sdk.Message)) (bkmodule.Handle, error)
}

func (h *kitMessageHost) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return h.k.runtime().PublishRaw(ctx, topic, payload)
}

func (h *kitMessageHost) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (bkmodule.Handle, error) {
	if rt, ok := h.k.runtime().(rawSubscriptionRuntime); ok {
		handle, err := rt.SubscribeRawHandle(ctx, topic, handler)
		if err != nil {
			return nil, err
		}
		if h.record != nil {
			h.record(topic)
		}
		h.scope.Defer(handle.Close)
		return handle, nil
	}
	cancel, err := h.k.runtime().SubscribeRaw(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	handle := bkmodule.HandleFunc(func(context.Context) error {
		cancel()
		return nil
	})
	if h.record != nil {
		h.record(topic)
	}
	h.scope.Defer(handle.Close)
	return handle, nil
}

func (h *kitMessageHost) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	return h.k.kernel.ReplyRaw(ctx, replyTo, correlationID, payload, done)
}

type kitCommandHost struct {
	k      *Kit
	scope  bkmodule.Scope
	record func(bkmodule.CommandSpec)
}

func (h *kitCommandHost) Handle(spec bkmodule.CommandSpec) (bkmodule.Handle, error) {
	handle, err := h.k.kernel.MountCommand(context.Background(), spec)
	if err != nil {
		return nil, err
	}
	if h.record != nil {
		h.record(spec)
	}
	h.scope.Defer(handle.Close)
	return handle, nil
}

func (h *kitCommandHost) Has(topic string) bool { return h.k.kernel.HasCommand(topic) }

type kitToolHost struct {
	k      *Kit
	scope  bkmodule.Scope
	record func(bkmodule.ResourceDescriptor)
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
	if h.record != nil {
		h.record(bkmodule.ToolResource(spec))
	}
	handle := bkmodule.HandleFunc(func(context.Context) error {
		h.k.kernel.Tools.Unregister(spec.Name)
		return nil
	})
	h.scope.Defer(handle.Close)
	return handle, nil
}

type kitCapabilityHost struct {
	k      *Kit
	scope  bkmodule.Scope
	record func(bkmodule.CapabilityDescriptor)
}

func (h *kitCapabilityHost) Provide(ctx context.Context, name string, value any) (bkmodule.Handle, error) {
	handle, err := h.k.caps.Provide(ctx, name, value)
	if err != nil {
		return nil, err
	}
	if h.record != nil {
		h.record(bkmodule.ProvidedCapability(name, value))
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
		return runtimeID, true
	case bkmodule.CapabilityNamespace:
		return h.k.kernel.Namespace(), true
	case bkmodule.CapabilityCallerID:
		return h.k.kernel.CallerID(), true
	case bkmodule.CapabilityPresenceTransport:
		return transport.Presence(h.k.kernel.Remote()), true
	case bkmodule.CapabilityPluginChecker:
		return func() bkmodule.PluginChecker { return h.k.kernel.PluginChecker() }, true
	case bkmodule.CapabilityScheduleHandlerLease:
		return bkmodule.LeaseFunc[types.ScheduleHandler](h.k.kernel.LeaseScheduleHandler), true
	case bkmodule.CapabilityAuditStoreLease:
		return bkmodule.LeaseFunc[auditpkg.Store](h.k.kernel.LeaseAuditStore), true
	case bkmodule.CapabilityAuditVerbosityLease:
		return bkmodule.LeaseFunc[auditpkg.Verbosity](h.k.kernel.LeaseAuditVerbosity), true
	case bkmodule.CapabilityTraceStoreLease:
		return bkmodule.LeaseFunc[coretracing.TraceStore](h.k.kernel.LeaseTraceStore), true
	case bkmodule.CapabilityProbeAll:
		return bkmodule.ProbeRunnerFunc(h.k.kernel.ProbeAllContext), true
	case bkmodule.CapabilityTransportKind:
		return h.k.kernel.TransportKind(), true
	case bkmodule.CapabilitySecretStore:
		if store := h.k.kernel.SecretStore(); store != nil {
			return store, true
		}
		return nil, false
	case bkmodule.CapabilityPluginRestarter:
		return func() plugincap.Restarter { return h.k.kernel.PluginRestarter() }, true
	case bkmodule.CapabilityRefreshProviderSecret:
		if refresher := h.k.kernel.ProviderSecretRefresher(); refresher != nil {
			return refresher, true
		}
		return nil, false
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
	case bkmodule.CapabilityRegistryMutation:
		if mutations := h.k.kernel.RegistryMutationManager(); mutations != nil {
			return mutations, true
		}
		return nil, false
	case bkmodule.CapabilityKitStore:
		if store := h.k.kernel.Store(); store != nil {
			return store, true
		}
		return nil, false
	case bkmodule.CapabilityMetricsSnapshot:
		return func() types.KernelMetrics { return h.k.kernel.Metrics() }, true
	case bkmodule.CapabilityHealthSnapshot:
		return func(ctx context.Context) any { return h.k.kernel.Health(ctx) }, true
	case bkmodule.CapabilityLifecycleDebugRegistry:
		return bkmodule.LifecycleDebugRegistry(h.k.lifecycleDebug), true
	case bkmodule.CapabilityLifecycleDebugSnapshot:
		return func() bkmodule.LifecycleDebugSnapshot { return h.k.lifecycleDebugSnapshot() }, true
	case bkmodule.CapabilityHealthProbes:
		return bkmodule.HealthProbes(h.k.kernel), true
	case bkmodule.CapabilityRequestCaller:
		if caller := h.k.kernel.Caller(); caller != nil {
			return bkmodule.RequestCaller(caller), true
		}
		return nil, false
	case bkmodule.CapabilityAgentRegistry:
		return h.k.kernel.AgentsDomain(), true
	case bkmodule.CapabilityMountedModules:
		return func() []bkmodule.Descriptor { return h.k.MountedModules() }, true
	case bkmodule.CapabilityModuleLifecycle:
		return kitModuleLifecycle{k: h.k}, true
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
	case bkmodule.CapabilityPluginCheckerLease:
		return bkmodule.LeaseFunc[bkmodule.PluginChecker](h.k.kernel.LeasePluginChecker), true
	case bkmodule.CapabilityPluginRestarterLease:
		return bkmodule.LeaseFunc[plugincap.Restarter](h.k.kernel.LeasePluginRestarter), true
	case bkmodule.CapabilityJSRuntimeHost:
		return h.k.kernel, true
	default:
		return nil, false
	}
}
