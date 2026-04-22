package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"

	"github.com/brainlet/brainkit"
	"github.com/google/uuid"
)

// Module is the brainkit.Module form of subprocess plugins. Init launches
// the plugin WebSocket endpoint lazily on first plugin start, kicks off
// any statically-configured plugins, restores dynamically-started plugins
// from the configured Store, and registers the plugin.* bus commands.
//
// See the package doc for the full feature set.
type Module struct {
	cfg       Config
	kit       *brainkit.Kit
	manager   *pluginManager
	lifecycle *LifecycleDomain
}

// NewModule builds the plugins module from config. Pass it to
// brainkit.Config.Modules.
func NewModule(cfg Config) *Module { return &Module{cfg: cfg} }

func (m *Module) Name() string { return "plugins" }

func (m *Module) Init(k *brainkit.Kit) error {
	m.kit = k

	// Plugins require a real transport: the WS control plane binds a
	// TCP socket and plugin→Kit bus traffic flows over the external
	// bus. Reject "memory" (in-process GoChannel) unconditionally — if
	// the module is wired at all, users intend to run plugins, and a
	// later restoreRunningPlugins() would silently start broken
	// subprocesses otherwise.
	if kind := k.TransportKind(); kind == "" || kind == "memory" {
		return &sdkerrors.ValidationError{Field: "transport", Message: "plugins module requires a non-memory transport"}
	}

	// Registry-path factories build without a Store (the KitStore is
	// owned by the Kit, which doesn't exist at Build time). Fall back
	// to k.Store() here — types.KitStore satisfies the narrow Store
	// interface structurally. nil is still allowed for callers that
	// explicitly want ephemeral plugins.
	if m.cfg.Store == nil {
		if ks := k.Store(); ks != nil {
			m.cfg.Store = ks
		}
	}

	m.manager = newPluginManager(m)
	m.lifecycle = newLifecycleDomain(m)

	// Restore dynamically-started plugins from previous session.
	m.restoreRunningPlugins()

	// Launch statically-configured plugins.
	if len(m.cfg.Plugins) > 0 {
		m.manager.startAll(m.cfg.Plugins)
	}

	// Register plugin.* bus commands.
	k.RegisterCommand(brainkit.Command(m.lifecycle.Start))
	k.RegisterCommand(brainkit.Command(m.lifecycle.Stop))
	k.RegisterCommand(brainkit.Command(m.lifecycle.Restart))
	k.RegisterCommand(brainkit.Command(m.lifecycle.List))
	k.RegisterCommand(brainkit.Command(m.lifecycle.Status))
	k.RegisterCommand(brainkit.Command(m.processPluginManifest))

	// Attach ourselves as the deploy.PluginChecker so the package
	// deploy dependency validator can see running plugins, and as the
	// engine.PluginRestarter so SecretsDomain can restart plugins on
	// secret rotation.
	k.SetPluginChecker(m)
	k.SetPluginRestarter(m)
	return nil
}

func (m *Module) Close() error {
	if m.manager != nil {
		m.manager.stopAll()
		if m.manager.wsServer != nil {
			m.manager.wsServer.Close()
		}
	}
	if m.kit != nil {
		m.kit.SetPluginChecker(nil)
		m.kit.SetPluginRestarter(nil)
	}
	return nil
}

// IsPluginRunning satisfies deploy.PluginChecker. It reports whether a
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
func (m *Module) StartPlugin(ctx context.Context, cfg types.PluginConfig) error {
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
	_, _ = m.kit.PublishRaw(ctx, "plugin.started", mustMarshalJSON(sdk.PluginStartedEvent{
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
	m.manager.stopPlugin(name, pc)
	if m.cfg.Store != nil {
		m.cfg.Store.DeleteRunningPlugin(name)
	}
	_, _ = m.kit.PublishRaw(ctx, "plugin.stopped", mustMarshalJSON(sdk.PluginStoppedEvent{
		Name: name, Reason: "stopped",
	}))
	m.kit.Audit().PluginStopped(name, "stopped")
	return nil
}

// RestartPlugin stops and re-starts a plugin. Equivalent to the
// pre-module Node.RestartPlugin.
func (m *Module) RestartPlugin(_ context.Context, name string) error {
	m.manager.mu.Lock()
	pc, ok := m.manager.plugins[name]
	m.manager.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "plugin", Name: name}
	}
	cfg := pc.config
	m.manager.stopPlugin(name, pc)
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

		cfg := types.PluginConfig{
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
func (m *Module) processPluginManifest(ctx context.Context, manifest sdk.PluginManifestMsg) (*sdk.PluginManifestResp, error) {
	for _, tool := range manifest.Tools {
		tool := tool
		fullName := tools.ComposeName(manifest.Owner, manifest.Name, manifest.Version, tool.Name)
		_ = m.kit.Tools().Register(tools.RegisteredTool{
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
						var result sdk.ToolCallResp
						if err := json.Unmarshal(payload, &result); err != nil {
							span.End(err)
							return nil, fmt.Errorf("brainkit: decode plugin tool result: %w", err)
						}
						span.End(nil)
						return result.Result, nil
					}
				},
			},
		})
	}

	_, _ = m.kit.PublishRaw(ctx, sdk.PluginRegisteredEvent{}.BusTopic(), mustMarshalJSON(sdk.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	}))
	m.kit.Audit().PluginRegistered(manifest.Name, manifest.Owner, manifest.Version, len(manifest.Tools))

	return &sdk.PluginManifestResp{Registered: true}, nil
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
// field is left nil — Init fills it from k.Store() at Init time.
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	cfg := Config{}
	for _, p := range y {
		cfg.Plugins = append(cfg.Plugins, types.PluginConfig{
			Name:   p.Name,
			Binary: p.Binary,
			Env:    p.Env,
		})
	}
	return NewModule(cfg), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "plugins",
		Status:  brainkit.ModuleStatusStable,
		Summary: "Subprocess plugin manager with WS control plane.",
	}
}

func init() { brainkit.RegisterModule("plugins", Factory{}) }

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
