package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// Node is a transport-connected runtime host. Implements sdk.Runtime by delegating to Kernel.
type Node struct {
	Kernel *Kernel

	config      NodeConfig
	nodeID      string
	plugins     *pluginManager
	pluginState PluginStateStore

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

	if len(cfg.Plugins) > 0 {
		node.plugins = newPluginManager(node)
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
		return nil, fmt.Errorf("brainkit: router start timeout (NATS JetStream provisioning may be slow)")
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

	if err := n.Kernel.wasm.restoreTransportSubscriptions(); err != nil {
		return err
	}

	if n.plugins != nil {
		if n.config.Messaging.Transport == "" || n.config.Messaging.Transport == "memory" {
			return fmt.Errorf("brainkit: plugins require nats transport")
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

// Close shuts down plugins, plugin state, then the wrapped Kernel.
func (n *Node) Close() error {
	n.mu.Lock()
	started := n.started
	n.started = false
	n.mu.Unlock()

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if started && n.plugins != nil {
		n.plugins.stopAll()
	}
	if n.pluginState != nil {
		collect(n.pluginState.Close())
	}
	// Kernel.Close() handles router + transport shutdown
	if n.Kernel != nil {
		collect(n.Kernel.Close())
	}
	return firstErr
}

// --- Node-specific command handlers ---

func (n *Node) processPluginManifest(ctx context.Context, manifest messages.PluginManifestMsg) (*messages.PluginManifestResp, error) {
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
						return nil, err
					}
					defer stop()

					if _, err := n.Kernel.remote.PublishRaw(messaging.ContextWithCorrelationID(callCtx, correlationID), topic, input); err != nil {
						return nil, fmt.Errorf("publish plugin tool %s: %w", topic, err)
					}

					select {
					case <-callCtx.Done():
						return nil, callCtx.Err()
					case msg := <-resultCh:
						var result messages.ToolCallResp
						if err := json.Unmarshal(msg.Payload, &result); err != nil {
							return nil, fmt.Errorf("brainkit: decode plugin tool result: %w", err)
						}
						if resultErr := messages.ResultErrorOf(result); resultErr != "" {
							return nil, fmt.Errorf("%s", resultErr)
						}
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
