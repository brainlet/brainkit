package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/sdkerrors"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// Node is a transport-connected runtime host. Implements sdk.Runtime by delegating to Kernel.
type Node struct {
	Kernel *Kernel

	config          NodeConfig
	nodeID          string
	plugins         *pluginManager
	pluginState     PluginStateStore
	pluginLifecycle *PluginLifecycleDomain
	discovery       discovery.Provider // nil if not configured

	mu      sync.Mutex
	started bool
}

// NewNode creates a transport host around a local Kernel.
// The external transport is injected into KernelConfig so Kernel uses it directly.
func NewNode(cfg NodeConfig) (*Node, error) {
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

	// Create external transport
	transport, err := messaging.NewTransportSet(cfg.Messaging.transportConfig())
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

	stateStore, err := newPluginStateStore(cfg)
	if err != nil {
		_ = kernel.Close()
		return nil, err
	}

	node := &Node{
		Kernel:      kernel,
		config:      cfg,
		nodeID:      cfg.NodeID,
		pluginState: stateStore,
	}

	node.plugins = newPluginManager(node)
	kernel.node = node // back-reference for secrets rotation
	// Wire PluginRestarter for SecretsDomain's rotation restart
	if kernel.secretsDomain != nil {
		kernel.secretsDomain.pluginRestarter = node
	}

	// Wire discovery if configured
	if cfg.Discovery.Type != "" {
		var disc discovery.Provider
		var discErr error
		switch cfg.Discovery.Type {
		case "static":
			disc = discovery.NewStaticFromConfig(cfg.Discovery.StaticPeers)
		case "multicast":
			disc, discErr = discovery.NewMulticast(cfg.Discovery.ServiceName)
			if discErr != nil {
				InvokeErrorHandler(cfg.Kernel.ErrorHandler, &sdkerrors.TransportError{
					Operation: "MulticastDiscovery", Cause: discErr,
				}, ErrorContext{Operation: "MulticastDiscovery", Component: "node"})
			}
		}
		if disc != nil {
			node.discovery = disc
			_ = disc.Register(discovery.Peer{
				Name:      cfg.NodeID,
				Namespace: kernelCfg.Namespace,
			})
		}
	}

	// Register ALL command bindings (kernel + node-specific) on the Kernel's router
	kernel.host.RegisterCommands(commandBindingsForNode(node))

	// Start the router. For NATS with JetStream auto-provisioning, the router's
	// Running() signal may take time (one JetStream stream per command handler).
	go func() {
		_ = kernel.router.Run(context.Background())
	}()

	select {
	case <-kernel.router.Running():
		// All handlers subscribed
	case <-time.After(2 * time.Minute):
		return nil, &sdk.TimeoutError{Operation: "router start (NATS JetStream provisioning)"}
	}

	return node, nil
}

// Start restores WASM shard subscriptions and launches plugins.
// The router is already running from NewNode.
func (n *Node) Start(ctx context.Context) error {
	n.mu.Lock()
	if n.started {
		n.mu.Unlock()
		return nil
	}
	n.started = true
	n.mu.Unlock()

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

func (n *Node) SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (func(), error) {
	return n.Kernel.SubscribeRaw(ctx, topic, handler)
}

// --- sdk.CrossNamespaceRuntime implementation ---

func (n *Node) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return n.Kernel.PublishRawTo(ctx, targetNamespace, topic, payload)
}

func (n *Node) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(messages.Message)) (func(), error) {
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
	if n.pluginState != nil {
		collect(n.pluginState.Close())
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
func (n *Node) StartPlugin(ctx context.Context, cfg PluginConfig) error {
	if n.config.Messaging.Transport == "" || n.config.Messaging.Transport == "memory" {
		return &sdk.ValidationError{Field: "transport", Message: "plugins require non-memory transport"}
	}
	pluginDefaults(&cfg)
	if err := n.plugins.startPlugin(cfg, 0); err != nil {
		return err
	}
	// Persist running state
	if n.Kernel.config.Store != nil {
		record := RunningPluginRecord{
			Name:       cfg.Name,
			BinaryPath: cfg.Binary,
			Env:        cfg.Env,
			Config:     cfg.Config,
			StartOrder: n.plugins.nextStartOrder(),
			StartedAt:  time.Now(),
			Role:       cfg.Role,
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
	n.Kernel.publish(ctx, "plugin.started", mustMarshalJSON(messages.PluginStartedEvent{
		Name: cfg.Name, PID: pid,
	}))
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
	n.Kernel.publish(ctx, "plugin.stopped", mustMarshalJSON(messages.PluginStoppedEvent{
		Name: name, Reason: "stopped",
	}))
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
func (n *Node) ListRunningPlugins() []RunningPlugin {
	return n.plugins.listPlugins()
}

// restoreRunningPlugins restores plugins that were running before shutdown.
func (n *Node) restoreRunningPlugins() {
	if n.Kernel.config.Store == nil {
		return
	}
	records, err := n.Kernel.config.Store.LoadRunningPlugins()
	if err != nil {
		InvokeErrorHandler(n.Kernel.config.ErrorHandler, &sdkerrors.PersistenceError{
			Operation: "LoadRunningPlugins", Cause: err,
		}, ErrorContext{Operation: "LoadRunningPlugins", Component: "node"})
		return
	}
	if len(records) == 0 {
		return
	}
	restored := 0
	for _, r := range records {
		// Skip if already running (from NodeConfig.Plugins static config)
		n.plugins.mu.Lock()
		_, alreadyRunning := n.plugins.plugins[r.Name]
		n.plugins.mu.Unlock()
		if alreadyRunning {
			continue
		}

		cfg := PluginConfig{
			Name:   r.Name,
			Binary: r.BinaryPath,
			Env:    r.Env,
			Config: r.Config,
			Role:   r.Role,
		}
		pluginDefaults(&cfg)
		if err := n.plugins.startPlugin(cfg, 0); err != nil {
			InvokeErrorHandler(n.Kernel.config.ErrorHandler, &sdkerrors.PersistenceError{
				Operation: "RestorePlugin", Source: r.Name, Cause: err,
			}, ErrorContext{Operation: "RestorePlugin", Component: "node", Source: r.Name})
			continue
		}
		if r.Role != "" && n.Kernel.rbac != nil {
			n.Kernel.rbac.Assign(r.Name, r.Role)
		}
		restored++
	}
	if restored > 0 {
		log.Printf("[brainkit] restored %d running plugins", restored)
	}
}

// --- Node-specific command handlers ---

func (n *Node) processPluginManifest(ctx context.Context, manifest messages.PluginManifestMsg) (*messages.PluginManifestResp, error) {
	// RBAC: validate manifest subscriptions against plugin's assigned role
	if n.Kernel.rbac != nil {
		role := n.Kernel.rbac.RoleForPlugin(manifest.Name)
		for _, sub := range manifest.Subscriptions {
			if !role.Bus.Subscribe.Allows(sub) {
				return nil, fmt.Errorf("plugin %s: subscription to %q denied by role %s", manifest.Name, sub, role.Name)
			}
		}
	}

	for _, tool := range manifest.Tools {
		tool := tool
		fullName := registry.ComposeName(manifest.Owner, manifest.Name, manifest.Version, tool.Name)
		_ = n.Kernel.Tools.Register(registry.RegisteredTool{
			Name:        fullName,
			ShortName:   tool.Name,
			Owner:       manifest.Owner,
			Package:     manifest.Name,
			Version:     manifest.Version,
			Description: tool.Description,
			InputSchema: json.RawMessage(tool.InputSchema),
			Executor: &registry.GoFuncExecutor{
				Fn: func(callCtx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
					topic := pluginToolTopic(manifest.Owner, manifest.Name, manifest.Version, tool.Name)
					resultTopic := topic + ".result"

					// Tracing: span around plugin tool call
					span := n.Kernel.tracer.StartSpan("plugin.tool:"+tool.Name, callCtx)
					span.SetAttribute("plugin", manifest.Name)
					span.SetAttribute("topic", topic)

					correlationID := uuid.NewString()
					waitCtx, cancel := context.WithCancel(callCtx)
					defer cancel()

					resultCh := make(chan messages.Message, 1)
					stop, err := n.Kernel.remote.SubscribeRaw(waitCtx, resultTopic, func(msg messages.Message) {
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

					if _, err := n.Kernel.remote.PublishRaw(messaging.ContextWithCorrelationID(callCtx, correlationID), topic, input); err != nil {
						span.End(err)
						return nil, fmt.Errorf("publish plugin tool %s: %w", topic, err)
					}

					select {
					case <-callCtx.Done():
						span.End(callCtx.Err())
						return nil, callCtx.Err()
					case msg := <-resultCh:
						var result messages.ToolCallResp
						if err := json.Unmarshal(msg.Payload, &result); err != nil {
							span.End(err)
							return nil, fmt.Errorf("brainkit: decode plugin tool result: %w", err)
						}
						if resultErr := messages.ResultErrorOf(result); resultErr != "" {
							retErr := fmt.Errorf("%s", resultErr)
							span.End(retErr)
							return nil, retErr
						}
						span.End(nil)
						return result.Result, nil
					}
				},
			},
		})
	}

	_ = n.Kernel.publish(ctx, messages.PluginRegisteredEvent{}.BusTopic(), mustMarshalJSON(messages.PluginRegisteredEvent{
		Owner:   manifest.Owner,
		Name:    manifest.Name,
		Version: manifest.Version,
		Tools:   len(manifest.Tools),
	}))

	return &messages.PluginManifestResp{Registered: true}, nil
}

func pluginToolTopic(owner, name, version, tool string) string {
	return fmt.Sprintf("plugin.tool.%s/%s@%s/%s", owner, name, version, tool)
}

func (n *Node) getPluginState(ctx context.Context, req messages.PluginStateGetMsg) (*messages.PluginStateGetResp, error) {
	pluginID := messaging.CallerIDFromContext(ctx)
	if pluginID == "" {
		return nil, fmt.Errorf("brainkit: plugin request missing caller identity")
	}
	value, err := n.pluginState.Get(ctx, pluginID, req.Key)
	if err != nil {
		return nil, err
	}
	return &messages.PluginStateGetResp{Value: value}, nil
}

func (n *Node) setPluginState(ctx context.Context, req messages.PluginStateSetMsg) (*messages.PluginStateSetResp, error) {
	pluginID := messaging.CallerIDFromContext(ctx)
	if pluginID == "" {
		return nil, fmt.Errorf("brainkit: plugin request missing caller identity")
	}
	if err := n.pluginState.Set(ctx, pluginID, req.Key, req.Value); err != nil {
		return nil, err
	}
	return &messages.PluginStateSetResp{OK: true}, nil
}

func mustMarshalJSON(v any) json.RawMessage {
	payload, _ := json.Marshal(v)
	return payload
}
