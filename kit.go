package brainkit

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/engine"
	"github.com/brainlet/brainkit/sdk"
)

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
// If Config.Transport is empty, creates a standalone in-memory runtime.
// If Config.Transport is set (e.g., "nats", "sql-sqlite"), creates a transport-connected
// runtime with plugin support and cross-Kit networking.
//
// Auto-behaviors:
//   - Providers nil → auto-detect from os.Getenv (OPENAI_API_KEY → openai, etc.)
//   - Store nil + FSRoot set → auto-create SQLiteStore
//   - SecretKey set → auto-create EncryptedKVStore
//   - Tracing true → auto-create MemoryTraceStore
func New(cfg Config) (*Kit, error) {
	kit := &Kit{}

	kernelCfg := cfg.toKernelConfig()

	if cfg.Transport == "" {
		// Standalone Kernel — in-memory transport
		kernel, err := engine.NewKernel(kernelCfg)
		if err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
		kit.kernel = kernel
	} else {
		// Transport-connected Node
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

// --- Lifecycle ---

// Shutdown drains in-flight handlers then closes. Use Close() for quick shutdown.
func (k *Kit) Shutdown(ctx context.Context) error {
	if k.node != nil {
		return k.node.Shutdown(ctx)
	}
	return k.kernel.Shutdown(ctx)
}
