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
	kernel  *engine.Kernel
	node    *engine.Node
	modules []Module // Kit-scoped modules initialized from cfg.Modules
}

// New creates a brainkit runtime from config.
//
// Default Transport is Memory() — in-process GoChannel, side-effect-free on
// disk, no plugins, fast for tests and library-embedded use. Use
// brainkit.QuickStart() for the batteries-included path (embedded NATS +
// SQLite stores), or set Transport to EmbeddedNATS() / NATS(url) / AMQP(url) /
// Redis(url) explicitly.
//
// Auto-behaviors:
//   - Providers nil → auto-detect from os.Getenv (OPENAI_API_KEY → openai, etc.)
//   - SecretKey set → auto-create EncryptedKVStore
func New(cfg Config) (*Kit, error) {
	kit := &Kit{}

	// Zero-value transport defaults to Memory — no disk side-effects, no
	// background goroutines beyond the QuickJS runtime itself.
	if cfg.Transport.typ == "" {
		cfg.Transport = Memory()
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

	// Initialize Kit-scoped modules (Init(*Kit)) after the kernel is built.
	// Legacy engine.Module instances were already initialized inside engine.NewKernel.
	for _, m := range cfg.Modules {
		pkgMod, ok := m.(Module)
		if !ok {
			continue
		}
		if err := pkgMod.Init(kit); err != nil {
			kit.Close()
			return nil, fmt.Errorf("brainkit: module %q init: %w", pkgMod.Name(), err)
		}
		kit.modules = append(kit.modules, pkgMod)
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
	for i := len(k.modules) - 1; i >= 0; i-- {
		_ = k.modules[i].Close()
	}
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
