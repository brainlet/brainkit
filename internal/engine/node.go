package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"github.com/brainlet/brainkit/internal/syncx"
	"time"

	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// Node is a transport-connected runtime host. Implements sdk.Runtime by delegating to Kernel.
type Node struct {
	Kernel *Kernel

	config          types.NodeConfig
	nodeID          string
	plugins         *pluginManager
	pluginLifecycle *PluginLifecycleDomain
	discovery       discovery.Provider // nil if not configured

	mu      syncx.Mutex
	started bool
}

// NewNode creates a transport host around a local Kernel.
// The external transport is injected into KernelConfig so Kernel uses it directly.
func NewNode(cfg types.NodeConfig) (*Node, error) {
	if cfg.NodeID == "" {
		cfg.NodeID = uuid.NewString()
	}

	kernelCfg := cfg.Kernel
	if kernelCfg.Namespace == "" {
		if cfg.Namespace != "" {
			kernelCfg.Namespace = cfg.Namespace
		} else {
			kernelCfg.Namespace = "user"
		}
	}
	if kernelCfg.CallerID == "" {
		kernelCfg.CallerID = cfg.NodeID
	}

	// Create external transport — consumer group = namespace for competing consumers
	tcfg := messagingToTransportConfig(cfg.Messaging)
	tcfg.Namespace = kernelCfg.Namespace
	transport, err := transport.NewTransportSet(tcfg)
	if err != nil {
		return nil, fmt.Errorf("brainkit: transport: %w", err)
	}

	// Inject transport into KernelConfig — Kernel uses it instead of creating its own.
	// DeferRouterStart: we need to add node-specific bindings before the router starts.
	kernelCfg.Transport = transport
	kernelCfg.DeferRouterStart = true

	kernel, err := NewKernel(kernelCfg)
	if err != nil {
		_ = transport.Close()
		return nil, err
	}

	node := &Node{
		Kernel: kernel,
		config: cfg,
		nodeID: cfg.NodeID,
	}

	node.plugins = newPluginManager(node)
	kernel.node = node // back-reference for secrets rotation
	// Wire PluginRestarter for SecretsDomain's rotation restart
	if kernel.secretsDomain != nil {
		kernel.secretsDomain.pluginRestarter = node
	}

	// Wire discovery if configured (Register is deferred to Start — bus provider publishes on register)
	if cfg.Discovery.Type != "" {
		switch cfg.Discovery.Type {
		case "static":
			node.discovery = discovery.NewStaticFromConfig(cfg.Discovery.StaticPeers)
		case "bus":
			node.discovery = discovery.NewBus(discovery.BusConfig{
				Transport: kernel.remote,
				Heartbeat: cfg.Discovery.Heartbeat,
				TTL:       cfg.Discovery.TTL,
			})
		}
	}

	// Register ALL command bindings (kernel + node-specific) on the Kernel's router.
	// Bindings must be registered before router.Run() — the router subscribes at start.
	kernel.host.RegisterCommands(commandBindingsForNode(node))

	return node, nil
}

// Start starts the message router and launches plugins.
// The caller's context controls the startup timeout — for NATS with JetStream
// auto-provisioning, stream creation may take time. If ctx has no deadline,
// a 2-minute safety timeout is applied.
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.started {
		n.mu.Unlock()
		return nil
	}
	n.started = true
	n.mu.Unlock()

	// Start the router in a background goroutine
	go func() {
		_ = n.Kernel.router.Run(context.Background())
	}()

	// Wait for the router to be ready (all handlers subscribed).
	// Apply a default 2-minute timeout if the caller didn't set one.
	waitCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		waitCtx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}
	select {
	case <-n.Kernel.router.Running():
		// All handlers subscribed
	case <-waitCtx.Done():
		return &sdk.TimeoutError{Operation: "router start (NATS JetStream provisioning)"}
	}

	// Register with discovery after router is ready (bus provider publishes on register)
	if n.discovery != nil {
		if err := n.discovery.Register(discovery.Peer{
			Name:      n.nodeID,
			Namespace: n.Kernel.Namespace(),
		}); err != nil {
			// Non-fatal — discovery failure shouldn't block startup
			types.InvokeErrorHandler(n.Kernel.config.ErrorHandler, &sdkerrors.TransportError{
				Operation: "DiscoveryRegister", Cause: err,
			}, types.ErrorContext{Operation: "DiscoveryRegister", Component: "node"})
		}
	}

	// Restore dynamically-started plugins from previous session
	n.restoreRunningPlugins()

	if len(n.config.Plugins) > 0 {
		if n.config.Messaging.Transport == "" || n.config.Messaging.Transport == "memory" {
			return &sdk.ValidationError{Field: "transport", Message: "plugins require non-memory transport"}
		}
		n.plugins.startAll(n.config.Plugins)
	}
	return nil
}

// --- sdk.Runtime implementation (delegates to Kernel) ---

func (n *Node) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return n.Kernel.PublishRaw(ctx, topic, payload)
}

func (n *Node) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	return n.Kernel.SubscribeRaw(ctx, topic, handler)
}

// --- sdk.CrossNamespaceRuntime implementation ---

func (n *Node) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return n.Kernel.PublishRawTo(ctx, targetNamespace, topic, payload)
}

func (n *Node) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(sdk.Message)) (func(), error) {
	return n.Kernel.SubscribeRawTo(ctx, targetNamespace, topic, handler)
}

// ReplyRaw delegates to Kernel's ReplyRaw.
func (n *Node) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	return n.Kernel.ReplyRaw(ctx, replyTo, correlationID, payload, done)
}

// Shutdown drains plugins and handlers, then closes everything.
func (n *Node) Shutdown(ctx context.Context) error {
	n.Kernel.draining.Store(true)

	n.mu.Lock()
	started := n.started
	n.started = false
	n.mu.Unlock()

	if started && n.plugins != nil {
		n.plugins.stopAll()
	}

	n.Kernel.waitForDrain(ctx)

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if n.discovery != nil {
		collect(n.discovery.Close())
	}
	if n.Kernel != nil {
		collect(n.Kernel.close())
	}
	return firstErr
}

// Close shuts down with a short drain timeout (5s).
func (n *Node) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return n.Shutdown(ctx)
}

// IsDraining returns true during the drain phase.
func (n *Node) IsDraining() bool {
	return n.Kernel.IsDraining()
}

// --- Dynamic plugin lifecycle ---

// StartPlugin starts a plugin dynamically at runtime.
func (n *Node) StartPlugin(ctx context.Context, cfg types.PluginConfig) error {
	if n.config.Messaging.Transport == "" || n.config.Messaging.Transport == "memory" {
		return &sdk.ValidationError{Field: "transport", Message: "plugins require non-memory transport"}
	}
	pluginDefaults(&cfg)
	if err := n.plugins.startPlugin(cfg, 0); err != nil {
		return err
	}
	// Persist running state
	if n.Kernel.config.Store != nil {
		record := types.RunningPluginRecord{
			Name:       cfg.Name,
			BinaryPath: cfg.Binary,
			Env:        cfg.Env,
			Config:     cfg.Config,
			StartOrder: n.plugins.nextStartOrder(),
			StartedAt:  time.Now(),
		}
		n.Kernel.config.Store.SaveRunningPlugin(record)
	}
	// Emit event
	pid := 0
	for _, p := range n.plugins.listPlugins() {
		if p.Name == cfg.Name {
			pid = p.PID
			break
		}
	}
	n.Kernel.publish(ctx, "plugin.started", mustMarshalJSON(sdk.PluginStartedEvent{
		Name: cfg.Name, PID: pid,
	}))
	n.Kernel.audit.PluginStarted(cfg.Name, pid)
	return nil
}

// StopPlugin stops a running plugin gracefully.
func (n *Node) StopPlugin(ctx context.Context, name string) error {
	n.plugins.mu.Lock()
	pc, ok := n.plugins.plugins[name]
	n.plugins.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "plugin", Name: name}
	}
	n.plugins.stopPlugin(name, pc)
	if n.Kernel.config.Store != nil {
		n.Kernel.config.Store.DeleteRunningPlugin(name)
	}
	n.Kernel.publish(ctx, "plugin.stopped", mustMarshalJSON(sdk.PluginStoppedEvent{
		Name: name, Reason: "stopped",
	}))
	n.Kernel.audit.PluginStopped(name, "stopped")
	return nil
}

// RestartPlugin stops and re-starts a plugin.
func (n *Node) RestartPlugin(ctx context.Context, name string) error {
	n.plugins.mu.Lock()
	pc, ok := n.plugins.plugins[name]
	n.plugins.mu.Unlock()
	if !ok {
		return &sdk.NotFoundError{Resource: "plugin", Name: name}
	}
	cfg := pc.config
	n.plugins.stopPlugin(name, pc)
	return n.plugins.startPlugin(cfg, 0)
}

// ListRunningPlugins returns all running plugins.
func (n *Node) ListRunningPlugins() []types.RunningPlugin {
	return n.plugins.listPlugins()
}

// restoreRunningPlugins restores plugins that were running before shutdown.
func (n *Node) restoreRunningPlugins() {
	if n.Kernel.config.Store == nil {
		return
	}
	records, err := n.Kernel.config.Store.LoadRunningPlugins()
	if err != nil {
		types.InvokeErrorHandler(n.Kernel.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadRunningPlugins", Cause: err,
		}, types.ErrorContext{Operation: "LoadRunningPlugins", Component: "node"})
		return
	}
	if len(records) == 0 {
		return
	}
	restored := 0
	for _, r := range records {
		// Skip if already running (from types.NodeConfig.Plugins static config)
		n.plugins.mu.Lock()
		_, alreadyRunning := n.plugins.plugins[r.Name]
		n.plugins.mu.Unlock()
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
		if err := n.plugins.startPlugin(cfg, 0); err != nil {
			types.InvokeErrorHandler(n.Kernel.config.ErrorHandler, &sdkerrors.PersistenceError{
				Operation: "RestorePlugin", Source: r.Name, Cause: err,
			}, types.ErrorContext{Operation: "RestorePlugin", Component: "node", Source: r.Name})
			continue
		}
		restored++
	}
	if restored > 0 {
		n.Kernel.logger.Info("restored running plugins", slog.Int("count", restored))
	}
}

// --- Node-specific command handlers ---

func (n *Node) processPluginManifest(ctx context.Context, manifest sdk.PluginManifestMsg) (*sdk.PluginManifestResp, error) {
	for _, tool := range manifest.Tools {
		tool := tool
		fullName := tools.ComposeName(manifest.Owner, manifest.Name, manifest.Version, tool.Name)
		_ = n.Kernel.Tools.Register(tools.RegisteredTool{
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

					span := n.Kernel.tracer.StartSpan("plugin.tool:"+tool.Name, callCtx)
					span.SetAttribute("plugin", manifest.Name)
					span.SetAttribute("topic", topic)

					callerReplyTo := transport.ReplyToFromContext(callCtx)
					if callerReplyTo != "" {
						_, err := n.Kernel.remote.PublishRawWithMeta(callCtx, topic, input, map[string]string{
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
					stop, err := n.Kernel.remote.SubscribeRaw(waitCtx, resultTopic, func(msg sdk.Message) {
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

					if _, err := n.Kernel.remote.PublishRaw(transport.ContextWithCorrelationID(callCtx, correlationID), topic, input); err != nil {
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

	_ = n.Kernel.publish(ctx, sdk.PluginRegisteredEvent{}.BusTopic(), mustMarshalJSON(sdk.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	}))
	n.Kernel.audit.PluginRegistered(manifest.Name, manifest.Owner, manifest.Version, len(manifest.Tools))

	return &sdk.PluginManifestResp{Registered: true}, nil
}

func pluginToolTopic(owner, name, version, tool string) string {
	return fmt.Sprintf("plugin.tool.%s/%s@%s/%s", owner, name, version, tool)
}

func mustMarshalJSON(v any) json.RawMessage {
	payload, err := json.Marshal(v)
	if err != nil {
		slog.Error("mustMarshalJSON: marshal failed", slog.String("error", err.Error()), slog.String("type", fmt.Sprintf("%T", v)))
		return nil
	}
	return payload
}

// messagingToTransportConfig converts types.MessagingConfig to transport.TransportConfig.
func messagingToTransportConfig(cfg types.MessagingConfig) transport.TransportConfig {
	return transport.TransportConfig{
		Type:         cfg.Transport,
		NATSURL:      cfg.NATSURL,
		NATSName:     cfg.NATSName,
		AMQPURL:      cfg.AMQPURL,
		RedisURL:     cfg.RedisURL,
		NATSStoreDir: cfg.NATSStoreDir,
	}
}
