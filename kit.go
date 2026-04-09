package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// runtimeID is generated once per process. All Kit instances in the same
// Go process share this ID. Used to distinguish local vs remote messages.
var runtimeID = uuid.NewString()

// RuntimeID returns the process-level identity shared by all Kits.
// Two Kits with the same RuntimeID are in the same OS process.
func RuntimeID() string { return runtimeID }

// Kit is a brainkit runtime.
//
// Kit implements sdk.Runtime — interact with it through sdk.Publish and sdk.SubscribeTo.
// Every feature is a typed bus command: deploy packages, manage providers, schedule messages,
// manage secrets, control plugins — all through async message passing.
//
// Create with New(). Use sdk.Publish(kit, ctx, msg) to send commands.
// Use sdk.SubscribeTo[Resp](kit, ctx, replyTo, handler) to receive responses.
type Kit struct {
	kernel *engine.Kernel
	node   *engine.Node
}

// New creates a brainkit runtime from config.
//
// If Config.Transport is empty or "embedded", starts an in-process NATS server
// with JetStream — zero config, plugins work, real pub/sub.
// If Config.Transport is "memory", creates a standalone in-memory runtime
// (GoChannel transport, no plugins, fast for tests).
// Other transports ("nats", "amqp", "redis") connect to external servers.
//
// Auto-behaviors:
//   - Providers nil → auto-detect from os.Getenv (OPENAI_API_KEY → openai, etc.)
//   - Store nil + FSRoot set → auto-create SQLiteStore
//   - SecretKey set → auto-create EncryptedKVStore
//   - Tracing true → auto-create MemoryTraceStore
func New(cfg Config) (*Kit, error) {
	kit := &Kit{}

	// Zero-value transport defaults to embedded NATS — zero-config production mode.
	if cfg.Transport.typ == "" {
		cfg.Transport = EmbeddedNATS()
	}

	kernelCfg := cfg.toKernelConfig()

	if cfg.Transport.typ == "memory" {
		// Standalone Kernel — in-memory GoChannel, no plugins, fast for tests.
		kernel, err := engine.NewKernel(kernelCfg)
		if err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
		kit.kernel = kernel
	} else {
		// Transport-connected Node (embedded, nats, amqp, redis)
		nodeCfg := cfg.toNodeConfig(kernelCfg)
		node, err := engine.NewNode(nodeCfg)
		if err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
		if err := node.Start(context.Background()); err != nil {
			node.Close()
			return nil, fmt.Errorf("brainkit: start: %w", err)
		}
		kit.node = node
		kit.kernel = node.Kernel
	}

	return kit, nil
}

// runtime returns the underlying sdk.Runtime (Node if present, else Kernel).
func (k *Kit) runtime() sdk.Runtime {
	if k.node != nil {
		return k.node
	}
	return k.kernel
}

// --- sdk.Runtime implementation ---

// PublishRaw sends a message to a topic. Returns correlationID.
func (k *Kit) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return k.runtime().PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic. Returns cancel function.
func (k *Kit) SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (func(), error) {
	return k.runtime().SubscribeRaw(ctx, topic, handler)
}

// Close shuts down with a short drain timeout (5s).
func (k *Kit) Close() error {
	return k.runtime().Close()
}

// --- sdk.CrossNamespaceRuntime implementation ---

// PublishRawTo publishes to a specific Kit's namespace.
func (k *Kit) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return k.kernel.PublishRawTo(ctx, targetNamespace, topic, payload)
}

// SubscribeRawTo subscribes to a topic in a specific Kit's namespace.
func (k *Kit) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(sdk.Message)) (func(), error) {
	return k.kernel.SubscribeRawTo(ctx, targetNamespace, topic, handler)
}

// --- sdk.Replier implementation (gateway type-asserts for this) ---

// ReplyRaw publishes directly to a resolved replyTo topic.
func (k *Kit) ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error {
	return k.kernel.ReplyRaw(ctx, replyTo, correlationID, payload, done)
}

// --- Health probes (gateway type-asserts for these) ---

// Alive returns true if the QuickJS runtime can evaluate a trivial expression.
func (k *Kit) Alive(ctx context.Context) bool {
	return k.kernel.Alive(ctx)
}

// Ready returns true if the Kit can serve traffic (not draining, runtime alive).
func (k *Kit) Ready(ctx context.Context) bool {
	return k.kernel.Ready(ctx)
}

// HealthJSON returns the full health status as JSON.
func (k *Kit) HealthJSON(ctx context.Context) json.RawMessage {
	return k.kernel.HealthJSON(ctx)
}

// IsDraining returns true during the drain phase.
func (k *Kit) IsDraining() bool {
	return k.kernel.IsDraining()
}

// --- Lifecycle ---

// Shutdown drains in-flight handlers then closes. Use Close() for quick shutdown.
func (k *Kit) Shutdown(ctx context.Context) error {
	if k.node != nil {
		return k.node.Shutdown(ctx)
	}
	return k.kernel.Shutdown(ctx)
}
