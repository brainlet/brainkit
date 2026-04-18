package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/internal/syncx"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// Node is a transport-connected runtime host. Implements sdk.Runtime by delegating to Kernel.
type Node struct {
	Kernel *Kernel

	config types.NodeConfig
	nodeID string

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

	kernel.node = node // back-reference for secrets rotation
	return node, nil
}

// StartRouter registers all command bindings (kernel + node-specific) on the
// host and starts the Watermill router. brainkit.New calls this after all
// brainkit.Modules have registered their commands so the router starts with
// the complete binding set.
func (n *Node) StartRouter(ctx context.Context) error {
	n.Kernel.host.RegisterCommands(commandBindingsForNode(n))
	return n.Start(ctx)
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

	_ = started
	n.Kernel.waitForDrain(ctx)

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
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

// Plugin lifecycle moved to modules/plugins. SecretsDomain's plugin
// restarter is attached via (*Kit).SetPluginRestarter; the
// package-deploy `Requires.plugins` gate reads kernel.pluginChecker.

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
