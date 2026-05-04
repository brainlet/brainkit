package plugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/tools"
	coretracing "github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	plugincap "github.com/brainlet/brainkit/modulecap/plugin"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"

	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	_ "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/google/uuid"
)

// Module is the bkmodule.Module form of subprocess plugins. Mount launches
// the plugin WebSocket endpoint lazily on first plugin start, kicks off
// any statically-configured plugins, restores dynamically-started plugins
// from the configured Store, and registers the plugin.* bus commands.
//
// See the package doc for the full feature set.
type Module struct {
	mu      sync.RWMutex
	closeMu sync.Mutex

	cfg       Config
	kit       pluginHost
	manager   *pluginManager
	lifecycle *LifecycleDomain

	regMu         sync.Mutex
	registrations map[string]pluginmsg.PluginRegisteredEvent

	toolMu         sync.Mutex
	toolRefs       map[string]int
	pluginToolRefs map[string]map[string]int

	timerMu      sync.Mutex
	replayTimers []*time.Timer

	pluginCheckerLease   bkmodule.Handle
	pluginRestarterLease bkmodule.Handle

	pluginCheckerLeaseActive   atomic.Bool
	pluginRestarterLeaseActive atomic.Bool
	closing                    atomic.Bool
}

// NewModule builds the plugins module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) ID() string { return "plugins" }

func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	ph, err := newMountedPluginHost(host)
	if err != nil {
		return err
	}
	host.Scope().Defer(func(closeCtx context.Context) error { return m.CloseContext(closeCtx) })
	if err := m.start(ctx, ph); err != nil {
		return err
	}
	lifecycleDebug, _ := bkmodule.Capability[bkmodule.LifecycleDebugRegistry](host, bkmodule.CapabilityLifecycleDebugRegistry)
	if lifecycleDebug != nil {
		handle, err := lifecycleDebug.RegisterLifecycleDebug(ctx, "plugins", func() any {
			return m.DebugSnapshot()
		})
		if err != nil {
			return fmt.Errorf("plugins: lifecycle debug: %w", err)
		}
		host.Scope().Defer(handle.Close)
	}
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindProcess, "plugins.manager", "Subprocess plugin manager."))
	host.Scope().Resource(bkmodule.Resource(bkmodule.ResourceKindHTTP, "plugins.websocket", "Plugin WebSocket control plane."))
	for _, spec := range []bkmodule.CommandSpec{
		bkmodule.Command(m.lifecycle.Start),
		bkmodule.Command(m.lifecycle.Stop),
		bkmodule.Command(m.lifecycle.Restart),
		bkmodule.Command(m.lifecycle.List),
		bkmodule.Command(m.lifecycle.Status),
		bkmodule.Command(m.processPluginManifest),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

func (m *Module) start(ctx context.Context, host pluginHost) error {
	m.mu.Lock()
	m.kit = host
	m.mu.Unlock()
	m.regMu.Lock()
	m.registrations = map[string]pluginmsg.PluginRegisteredEvent{}
	m.regMu.Unlock()

	// Plugins require a real transport: the WS control plane binds a
	// TCP socket and plugin→Kit bus traffic flows over the external
	// bus. Reject "memory" (in-process GoChannel) unconditionally — if
	// the module is wired at all, users intend to run plugins, and a
	// later restoreRunningPlugins() would silently start broken
	// subprocesses otherwise.
	if kind := m.kit.TransportKind(); kind == "" || kind == "memory" {
		return &sdkerrors.ValidationError{Field: "transport", Message: "plugins module requires a non-memory transport"}
	}

	// Registry-path factories build without a Store (the KitStore is
	// owned by the Kit, which doesn't exist at Build time). Fall back
	// to k.Store() here — types.KitStore satisfies the narrow Store
	// interface structurally. nil is still allowed for callers that
	// explicitly want ephemeral plugins.
	if m.cfg.Store == nil {
		if ks := m.kit.Store(); ks != nil {
			m.cfg.Store = ks
		}
	}

	manager := newPluginManager(m)
	lifecycle := newLifecycleDomain(m)
	m.mu.Lock()
	m.manager = manager
	m.lifecycle = lifecycle
	m.mu.Unlock()

	// Restore dynamically-started plugins from previous session.
	m.restoreRunningPlugins()

	// Launch statically-configured plugins.
	if len(m.cfg.Plugins) > 0 {
		manager.startAll(m.cfg.Plugins)
		m.replayRegistrationsAfter(250 * time.Millisecond)
	}

	// Attach scoped checker/restarter leases so packages/secrets can observe
	// plugin state while this module is mounted.
	checkerLease, err := m.kit.LeasePluginChecker(ctx, m)
	if err != nil {
		return err
	}
	restarterLease, err := m.kit.LeasePluginRestarter(ctx, m)
	if err != nil {
		_ = checkerLease.Close(ctx)
		return err
	}
	m.mu.Lock()
	m.pluginCheckerLease = checkerLease
	m.pluginRestarterLease = restarterLease
	m.pluginCheckerLeaseActive.Store(true)
	m.pluginRestarterLeaseActive.Store(true)
	m.mu.Unlock()
	return nil
}

func (m *Module) announceRegistered(ctx context.Context, evt pluginmsg.PluginRegisteredEvent) {
	m.regMu.Lock()
	if m.registrations == nil {
		m.registrations = map[string]pluginmsg.PluginRegisteredEvent{}
	}
	m.registrations[evt.Name] = evt
	m.regMu.Unlock()

	kit := m.currentKit()
	if kit == nil {
		return
	}
	_, _ = kit.PublishRaw(ctx, evt.BusTopic(), mustMarshalJSON(evt))
	kit.Audit().PluginRegistered(evt.Name, evt.Owner, evt.Version, evt.Tools)
}

func (m *Module) registerPluginTool(pluginName string, tool tools.RegisteredTool) error {
	kit := m.currentKit()
	if kit == nil {
		return &sdkerrors.NotConfiguredError{Feature: "plugins"}
	}
	if err := kit.Tools().Register(tool); err != nil {
		return err
	}
	m.toolMu.Lock()
	defer m.toolMu.Unlock()
	if m.toolRefs == nil {
		m.toolRefs = map[string]int{}
	}
	if m.pluginToolRefs == nil {
		m.pluginToolRefs = map[string]map[string]int{}
	}
	m.toolRefs[tool.Name]++
	if m.pluginToolRefs[pluginName] == nil {
		m.pluginToolRefs[pluginName] = map[string]int{}
	}
	m.pluginToolRefs[pluginName][tool.Name]++
	return nil
}

func (m *Module) unregisterPluginToolNames(pluginName string, names []string) {
	if len(names) == 0 {
		return
	}
	toRemove := m.decrementPluginToolRefs(pluginName, names)
	kit := m.currentKit()
	if kit == nil {
		return
	}
	for _, name := range toRemove {
		kit.Tools().Unregister(name)
	}
}

func (m *Module) unregisterPluginToolsForPlugin(pluginName string) {
	m.toolMu.Lock()
	pluginRefs := m.pluginToolRefs[pluginName]
	names := make([]string, 0, len(pluginRefs))
	for name, count := range pluginRefs {
		for i := 0; i < count; i++ {
			names = append(names, name)
		}
	}
	m.toolMu.Unlock()
	m.unregisterPluginToolNames(pluginName, names)
}

func (m *Module) unregisterAllPluginTools() {
	m.toolMu.Lock()
	names := make([]string, 0, len(m.toolRefs))
	for name := range m.toolRefs {
		names = append(names, name)
	}
	m.toolRefs = nil
	m.pluginToolRefs = nil
	m.toolMu.Unlock()
	kit := m.currentKit()
	if kit == nil {
		return
	}
	for _, name := range names {
		kit.Tools().Unregister(name)
	}
}

func (m *Module) decrementPluginToolRefs(pluginName string, names []string) []string {
	m.toolMu.Lock()
	defer m.toolMu.Unlock()
	if len(m.toolRefs) == 0 {
		return nil
	}
	var toRemove []string
	for _, name := range names {
		if pluginRefs := m.pluginToolRefs[pluginName]; len(pluginRefs) > 0 {
			if pluginRefs[name] <= 1 {
				delete(pluginRefs, name)
			} else {
				pluginRefs[name]--
			}
			if len(pluginRefs) == 0 {
				delete(m.pluginToolRefs, pluginName)
			}
		}
		if m.toolRefs[name] <= 1 {
			delete(m.toolRefs, name)
			toRemove = append(toRemove, name)
			continue
		}
		m.toolRefs[name]--
	}
	return toRemove
}

func (m *Module) replayRegistrationsAfter(delay time.Duration) {
	timer := time.AfterFunc(delay, func() {
		kit := m.currentKit()
		if kit == nil {
			return
		}
		m.regMu.Lock()
		events := make([]pluginmsg.PluginRegisteredEvent, 0, len(m.registrations))
		for _, evt := range m.registrations {
			events = append(events, evt)
		}
		m.regMu.Unlock()
		for _, evt := range events {
			_, _ = kit.PublishRaw(context.Background(), evt.BusTopic(), mustMarshalJSON(evt))
		}
	})
	m.timerMu.Lock()
	m.replayTimers = append(m.replayTimers, timer)
	m.timerMu.Unlock()
}

func (m *Module) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return m.CloseContext(ctx)
}

func (m *Module) CloseContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.closeMu.Lock()
	defer m.closeMu.Unlock()
	m.closing.Store(true)
	defer m.closing.Store(false)

	var err error
	m.stopReplayTimers()
	manager := m.currentManager()
	if manager != nil {
		err = errors.Join(err, manager.stopAll(ctx))
		err = errors.Join(err, manager.closeWSServer(ctx))
	}
	if m.currentKit() != nil {
		m.unregisterAllPluginTools()
	}

	m.mu.RLock()
	restarterLease := m.pluginRestarterLease
	checkerLease := m.pluginCheckerLease
	m.mu.RUnlock()

	if restarterLease != nil {
		if closeErr := restarterLease.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
		} else {
			m.mu.Lock()
			m.pluginRestarterLease = nil
			m.pluginRestarterLeaseActive.Store(false)
			m.mu.Unlock()
		}
	} else {
		m.pluginRestarterLeaseActive.Store(false)
	}
	if checkerLease != nil {
		if closeErr := checkerLease.Close(ctx); closeErr != nil {
			err = errors.Join(err, closeErr)
		} else {
			m.mu.Lock()
			m.pluginCheckerLease = nil
			m.pluginCheckerLeaseActive.Store(false)
			m.mu.Unlock()
		}
	} else {
		m.pluginCheckerLeaseActive.Store(false)
	}
	if err == nil {
		m.mu.Lock()
		m.kit = nil
		m.mu.Unlock()
		m.regMu.Lock()
		m.registrations = nil
		m.regMu.Unlock()
	}
	return err
}

func (m *Module) currentKit() pluginHost {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.kit
}

func (m *Module) currentManager() *pluginManager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.manager
}

func (m *Module) stopReplayTimers() {
	m.timerMu.Lock()
	timers := append([]*time.Timer(nil), m.replayTimers...)
	m.replayTimers = nil
	m.timerMu.Unlock()
	for _, timer := range timers {
		timer.Stop()
	}
}

// IsPluginRunning satisfies module.PluginChecker. It reports whether a
// plugin by that name is currently tracked by the manager.
func (m *Module) IsPluginRunning(name string) bool {
	if m.manager == nil {
		return false
	}
	for _, p := range m.manager.listPlugins() {
		if p.Name == name {
			return true
		}
	}
	return false
}

// StartPlugin starts a plugin dynamically at runtime. Equivalent to the
// pre-module Node.StartPlugin.
func (m *Module) StartPlugin(ctx context.Context, cfg PluginConfig) error {
	if kind := m.kit.TransportKind(); kind == "" || kind == "memory" {
		return &sdkerrors.ValidationError{Field: "transport", Message: "plugins require non-memory transport"}
	}
	pluginDefaults(&cfg)
	if err := m.manager.startPlugin(cfg, 0); err != nil {
		return err
	}
	// Persist running state
	if m.cfg.Store != nil {
		record := types.RunningPluginRecord{
			Name:       cfg.Name,
			BinaryPath: cfg.Binary,
			Env:        cfg.Env,
			Config:     cfg.Config,
			StartOrder: m.manager.nextStartOrder(),
			StartedAt:  time.Now(),
		}
		m.cfg.Store.SaveRunningPlugin(record)
	}
	// Emit event
	pid := 0
	for _, p := range m.manager.listPlugins() {
		if p.Name == cfg.Name {
			pid = p.PID
			break
		}
	}
	_, _ = m.kit.PublishRaw(ctx, "plugin.started", mustMarshalJSON(pluginmsg.PluginStartedEvent{
		Name: cfg.Name, PID: pid,
	}))
	m.kit.Audit().PluginStarted(cfg.Name, pid)
	return nil
}

// StopPlugin stops a running plugin gracefully. Equivalent to the
// pre-module Node.StopPlugin.
func (m *Module) StopPlugin(ctx context.Context, name string) error {
	m.manager.mu.Lock()
	pc, ok := m.manager.plugins[name]
	m.manager.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "plugin", Name: name}
	}
	err := m.manager.stopPlugin(ctx, name, pc)
	m.unregisterPluginToolsForPlugin(name)
	if m.cfg.Store != nil {
		m.cfg.Store.DeleteRunningPlugin(name)
	}
	_, _ = m.kit.PublishRaw(ctx, "plugin.stopped", mustMarshalJSON(pluginmsg.PluginStoppedEvent{
		Name: name, Reason: "stopped",
	}))
	m.kit.Audit().PluginStopped(name, "stopped")
	return err
}

// RestartPlugin stops and re-starts a plugin. Equivalent to the
// pre-module Node.RestartPlugin.
func (m *Module) RestartPlugin(ctx context.Context, name string) error {
	m.manager.mu.Lock()
	pc, ok := m.manager.plugins[name]
	m.manager.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "plugin", Name: name}
	}
	cfg := pc.config
	if err := m.manager.stopPlugin(ctx, name, pc); err != nil {
		return err
	}
	m.unregisterPluginToolsForPlugin(name)
	return m.manager.startPlugin(cfg, 0)
}

// ListRunningPlugins returns all running plugins.
func (m *Module) ListRunningPlugins() []types.RunningPlugin {
	if m.manager == nil {
		return nil
	}
	return m.manager.listPlugins()
}

// restoreRunningPlugins restores plugins that were running before shutdown.
func (m *Module) restoreRunningPlugins() {
	if m.cfg.Store == nil {
		return
	}
	records, err := m.cfg.Store.LoadRunningPlugins()
	if err != nil {
		m.kit.ReportError(&sdkerrors.PersistenceError{
			Operation: "LoadRunningPlugins", Cause: err,
		}, types.ErrorContext{Operation: "LoadRunningPlugins", Component: "plugins"})
		return
	}
	if len(records) == 0 {
		return
	}
	restored := 0
	for _, r := range records {
		// Skip if already running (from cfg.Plugins static config)
		m.manager.mu.Lock()
		_, alreadyRunning := m.manager.plugins[r.Name]
		m.manager.mu.Unlock()
		if alreadyRunning {
			continue
		}

		cfg := PluginConfig{
			Name:   r.Name,
			Binary: r.BinaryPath,
			Env:    r.Env,
			Config: r.Config,
		}
		pluginDefaults(&cfg)
		if err := m.manager.startPlugin(cfg, 0); err != nil {
			m.kit.ReportError(&sdkerrors.PersistenceError{
				Operation: "RestorePlugin", Source: r.Name, Cause: err,
			}, types.ErrorContext{Operation: "RestorePlugin", Component: "plugins", Source: r.Name})
			continue
		}
		restored++
	}
	if restored > 0 {
		m.kit.Logger().Info("restored running plugins", slog.Int("count", restored))
	}
}

// processPluginManifest is the plugin.manifest bus command handler. It
// registers the plugin's tool set against the Kit's tool registry and
// emits a plugin.registered event. Moved from Node.processPluginManifest.
func (m *Module) processPluginManifest(ctx context.Context, manifest pluginmsg.PluginManifestMsg) (*pluginmsg.PluginManifestResp, error) {
	for _, tool := range manifest.Tools {
		tool := tool
		fullName := tools.ComposeName(manifest.Owner, manifest.Name, manifest.Version, tool.Name)
		if err := m.registerPluginTool(manifest.Name, tools.RegisteredTool{
			Name:        fullName,
			ShortName:   tool.Name,
			Owner:       manifest.Owner,
			Package:     manifest.Name,
			Version:     manifest.Version,
			Description: tool.Description,
			InputSchema: json.RawMessage(tool.InputSchema),
			Executor: &tools.GoFuncExecutor{
				// Two execution paths:
				//
				// Path 1 (pass-through): When called via the bus command router, the context
				// carries the caller's replyTo. The tool call is forwarded to the plugin with
				// that replyTo — the plugin responds directly to the original caller. Returns
				// (nil, nil) because the response bypasses this executor.
				//
				// Path 2 (direct Go call): No replyTo in context. Creates a temporary
				// subscription on .result, sends the call, and waits for the plugin's response.
				// Returns the actual (result, error).
				Fn: func(callCtx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
					topic := pluginToolTopic(manifest.Owner, manifest.Name, manifest.Version, tool.Name)

					span := m.kit.Tracer().StartSpan("plugin.tool:"+tool.Name, callCtx)
					span.SetAttribute("plugin", manifest.Name)
					span.SetAttribute("topic", topic)

					callerReplyTo := transport.ReplyToFromContext(callCtx)
					if callerReplyTo != "" {
						_, err := m.kit.Remote().PublishRawWithMeta(callCtx, topic, input, map[string]string{
							"replyTo": callerReplyTo,
						})
						span.End(err)
						if err != nil {
							return nil, fmt.Errorf("publish plugin tool %s: %w", topic, err)
						}
						return nil, nil
					}

					// Fallback path: direct Go call (no bus command router, no replyTo).
					// Subscribe to .result and wait — safe because this path doesn't
					// nest inside a command handler.
					resultTopic := topic + ".result"
					correlationID := uuid.NewString()
					waitCtx, cancel := context.WithCancel(callCtx)
					defer cancel()

					resultCh := make(chan sdk.Message, 1)
					stop, err := m.kit.Remote().SubscribeRaw(waitCtx, resultTopic, func(msg sdk.Message) {
						if msg.Metadata["correlationId"] == correlationID {
							select {
							case resultCh <- msg:
							default:
							}
							cancel()
						}
					})
					if err != nil {
						span.End(err)
						return nil, err
					}
					defer stop()

					if _, err := m.kit.Remote().PublishRaw(transport.ContextWithCorrelationID(callCtx, correlationID), topic, input); err != nil {
						span.End(err)
						return nil, fmt.Errorf("publish plugin tool %s: %w", topic, err)
					}

					select {
					case <-callCtx.Done():
						span.End(callCtx.Err())
						return nil, callCtx.Err()
					case msg := <-resultCh:
						payload := msg.Payload
						if msg.Metadata["envelope"] == "true" {
							if wire, err := sdk.DecodeEnvelope(payload); err == nil {
								if !wire.Ok && wire.Error != nil {
									retErr := sdk.FromEnvelope(wire)
									span.End(retErr)
									return nil, retErr
								}
								if wire.Ok {
									payload = wire.Data
								}
							}
						}
						var result toolmsg.ToolCallResp
						if err := json.Unmarshal(payload, &result); err != nil {
							span.End(err)
							return nil, fmt.Errorf("brainkit: decode plugin tool result: %w", err)
						}
						span.End(nil)
						return result.Result, nil
					}
				},
			},
		}); err != nil {
			return nil, err
		}
	}

	m.announceRegistered(ctx, pluginmsg.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	})

	return &pluginmsg.PluginManifestResp{Registered: true}, nil
}

// PluginYAML is one entry in the plugins list.
type PluginYAML struct {
	Name   string            `yaml:"name"`
	Binary string            `yaml:"binary"`
	Env    map[string]string `yaml:"env"`
}

// YAML is the config shape decoded by the registry factory.
//
//	modules:
//	  plugins:
//	    - name: foo
//	      binary: ./bin/foo
//	      env: { LOG_LEVEL: debug }
//
// A sequence (not a map) because plugin order can matter for
// deterministic start-up and the set is inherently a list.
type YAML []PluginYAML

// Factory is the registered ModuleFactory for plugins.
type Factory struct{}

// Build decodes the plugin list and returns a module whose Store
// field is left nil — Mount fills it from k.Store() at mount time.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := Config{}
	for _, p := range y {
		cfg.Plugins = append(cfg.Plugins, PluginConfig{
			Name:   p.Name,
			Binary: p.Binary,
			Env:    p.Env,
		})
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "plugins",
		Status:  bkmodule.StatusStable,
		Summary: "Subprocess plugin manager with WS control plane.",
		Requires: []string{
			"tools",
		},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[pluginmsg.PluginListRunningMsg, pluginmsg.PluginListRunningResp](),
			bkmodule.CommandMessage[pluginmsg.PluginManifestMsg, pluginmsg.PluginManifestResp](),
			bkmodule.CommandMessage[pluginmsg.PluginRestartMsg, pluginmsg.PluginRestartResp](),
			bkmodule.CommandMessage[pluginmsg.PluginStartMsg, pluginmsg.PluginStartResp](),
			bkmodule.CommandMessage[pluginmsg.PluginStatusMsg, pluginmsg.PluginStatusResp](),
			bkmodule.CommandMessage[pluginmsg.PluginStopMsg, pluginmsg.PluginStopResp](),
		},
		Events: []bkmodule.MessageDescriptor{
			bkmodule.EventMessage[pluginmsg.PluginRegisteredEvent](),
			bkmodule.EventMessage[pluginmsg.PluginStartedEvent](),
			bkmodule.EventMessage[pluginmsg.PluginStoppedEvent](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.OptionalCapabilityOf[types.SecretStore](bkmodule.CapabilitySecretStore),
			bkmodule.OptionalCapabilityOf[Store](bkmodule.CapabilityKitStore),
			bkmodule.OptionalCapabilityOf[bkmodule.LifecycleDebugRegistry](bkmodule.CapabilityLifecycleDebugRegistry),
			bkmodule.ProvidedCapabilityOf[func() bkmodule.PluginChecker](bkmodule.CapabilityPluginChecker),
			bkmodule.ProvidedCapabilityOf[func() plugincap.Restarter](bkmodule.CapabilityPluginRestarter),
			bkmodule.RequiredCapabilityOf[*auditpkg.Recorder](bkmodule.CapabilityAuditRecorder),
			bkmodule.RequiredCapabilityOf[string](bkmodule.CapabilityCallerID),
			bkmodule.RequiredCapabilityOf[string](bkmodule.CapabilityNamespace),
			bkmodule.RequiredCapabilityOf[*transport.RemoteClient](bkmodule.CapabilityRemoteClient),
			bkmodule.RequiredCapabilityOf[func(error, types.ErrorContext)](bkmodule.CapabilityReportError),
			bkmodule.RequiredCapabilityOf[<-chan struct{}](bkmodule.CapabilityShutdownSignal),
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[bkmodule.PluginChecker]](bkmodule.CapabilityPluginCheckerLease),
			bkmodule.RequiredCapabilityOf[bkmodule.LeaseFunc[plugincap.Restarter]](bkmodule.CapabilityPluginRestarterLease),
			bkmodule.RequiredCapabilityOf[*tools.ToolRegistry](bkmodule.CapabilityToolRegistry),
			bkmodule.RequiredCapabilityOf[string](bkmodule.CapabilityTransportKind),
			bkmodule.RequiredCapabilityOf[*coretracing.Tracer](bkmodule.CapabilityTracer),
		},
		Resources: []bkmodule.ResourceDescriptor{
			bkmodule.Resource(bkmodule.ResourceKindProcess, "plugins.manager", "Subprocess plugin manager."),
			bkmodule.Resource(bkmodule.ResourceKindHTTP, "plugins.websocket", "Plugin WebSocket control plane."),
		},
	}
}

func init() { bkmodule.Register("plugins", Factory{}) }

// pluginToolTopic is the wire topic for a plugin tool call. Moved from
// node.go as unexported.
func pluginToolTopic(owner, name, version, tool string) string {
	return fmt.Sprintf("plugin.tool.%s/%s@%s/%s", owner, name, version, tool)
}

// mustMarshalJSON marshals v or returns nil on error. Inlined from
// node.go so the module doesn't depend on internal/engine.
func mustMarshalJSON(v any) json.RawMessage {
	payload, err := json.Marshal(v)
	if err != nil {
		slog.Error("mustMarshalJSON: marshal failed", slog.String("error", err.Error()), slog.String("type", fmt.Sprintf("%T", v)))
		return nil
	}
	return payload
}
