package brainkit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/internal/engine"
	bkmodule "github.com/brainlet/brainkit/module"
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
	modules map[string]bkmodule.Module
	mounted map[string]bkmodule.Scope
	descs   map[string]bkmodule.Descriptor
	mountMu sync.Mutex
	caps    *bkmodule.CapabilityRegistry

	// Accessor caches — populated lazily on first call to the
	// matching accessor method. They are stateless wrappers over the
	// kernel's registries / secret store, so a single instance per
	// Kit is enough.
	providers *Providers
	storages  *Storages
	vectors   *Vectors
	secrets   *Secrets
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
	kit := &Kit{
		modules: map[string]bkmodule.Module{},
		mounted: map[string]bkmodule.Scope{},
		descs:   map[string]bkmodule.Descriptor{},
		caps:    bkmodule.NewCapabilityRegistry(),
	}

	// Zero-value transport defaults to Memory — no disk side-effects, no
	// background goroutines beyond the QuickJS runtime itself.
	if cfg.Transport.typ == "" {
		cfg.Transport = Memory()
	}
	var err error
	if cfg.needsJSRuntime() {
		cfg.Modules, err = ensureJSRuntimeModule(cfg.Modules, cfg.FSRoot)
		if err != nil {
			return nil, err
		}
	}
	cfg.Modules, err = orderModulesForStartup(cfg.Modules, cfg.FSRoot)
	if err != nil {
		return nil, err
	}

	kernelCfg := cfg.toKernelConfig()
	// Root assembly does not instantiate JS directly. When requested, the
	// registered jsruntime module is mounted after the router starts and enables
	// the runtime through a core capability. Keeping this false is the next step
	// toward removing JS implementation imports from the core package.
	kernelCfg.JSRuntime = false

	if cfg.Transport.typ == "memory" {
		// Standalone Kernel — in-memory GoChannel, no plugins, fast for tests.
		// Defer router start so the root command catalog is bound consistently
		// with the transport-connected path.
		kernelCfg.DeferRouterStart = true
		kernel, err := engine.NewKernel(kernelCfg)
		if err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
		kit.kernel = kernel
	} else {
		// Transport-connected Node (embedded, nats, amqp, redis)
		transportNamespace := kernelCfg.Namespace
		if transportNamespace == "" {
			transportNamespace = "user"
		}
		builtTransport, err := buildConfiguredTransport(cfg.Transport, transportNamespace, cfg.FSRoot)
		if err != nil {
			return nil, err
		}
		kernelCfg.Transport = builtTransport
		nodeCfg := cfg.toNodeConfig(kernelCfg)
		node, err := engine.NewNode(nodeCfg)
		if err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
		kit.node = node
		kit.kernel = node.Kernel
	}

	// Finalize transport bindings + start the router.
	if kit.node != nil {
		// Node path: command bindings were registered in engine.NewNode.
		// Router starts here.
		if err := kit.node.StartRouter(context.Background()); err != nil {
			kit.Close()
			return nil, fmt.Errorf("brainkit: start router: %w", err)
		}
	} else {
		if err := kit.kernel.StartRouter(context.Background()); err != nil {
			kit.Close()
			return nil, fmt.Errorf("brainkit: start router: %w", err)
		}
	}

	// Modules mount after the router is live. Their command host can add
	// handlers dynamically, which is the same path used for hot-mounting
	// modules after New returns.
	for _, mod := range cfg.Modules {
		if mod == nil {
			continue
		}
		if err := kit.Mount(context.Background(), mod); err != nil {
			kit.Close()
			return nil, fmt.Errorf("brainkit: module %q mount: %w", mod.ID(), err)
		}
	}

	return kit, nil
}

func ensureJSRuntimeModule(mods []bkmodule.Module, fsRoot string) ([]bkmodule.Module, error) {
	for i, mod := range mods {
		if mod != nil && mod.ID() == "jsruntime" {
			if i == 0 {
				return mods, nil
			}
			out := make([]bkmodule.Module, 0, len(mods))
			out = append(out, mod)
			out = append(out, mods[:i]...)
			out = append(out, mods[i+1:]...)
			return out, nil
		}
	}
	mod, err := buildRegisteredModule("jsruntime", fsRoot)
	if err != nil {
		return nil, err
	}
	out := make([]bkmodule.Module, 0, len(mods)+1)
	out = append(out, mod)
	out = append(out, mods...)
	return out, nil
}

func orderModulesForStartup(mods []bkmodule.Module, fsRoot string) ([]bkmodule.Module, error) {
	explicit := make(map[string]bkmodule.Module, len(mods))
	order := make([]string, 0, len(mods))
	for _, mod := range mods {
		if mod == nil {
			continue
		}
		id := mod.ID()
		if id == "" {
			return nil, fmt.Errorf("brainkit: module ID is required")
		}
		if _, exists := explicit[id]; exists {
			continue
		}
		explicit[id] = mod
		order = append(order, id)
	}

	state := map[string]int{}
	out := make([]bkmodule.Module, 0, len(explicit))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("brainkit: module dependency cycle involving %q", id)
		}
		state[id] = 1

		mod := explicit[id]
		if mod == nil {
			var err error
			mod, err = buildRegisteredModule(id, fsRoot)
			if err != nil {
				return err
			}
		}
		for _, dep := range moduleDependencies(mod) {
			if dep == "" || dep == id {
				continue
			}
			if err := visit(dep); err != nil {
				return err
			}
		}

		out = append(out, mod)
		state[id] = 2
		return nil
	}

	for _, id := range order {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func buildRegisteredModule(id, fsRoot string) (bkmodule.Module, error) {
	factory, ok := bkmodule.Lookup(id)
	if !ok {
		if id == "jsruntime" {
			return nil, fmt.Errorf("brainkit: JS runtime requested but module %q is not registered; import github.com/brainlet/brainkit/modules/jsruntime or github.com/brainlet/brainkit/presets/standard", id)
		}
		return nil, fmt.Errorf("brainkit: module dependency %q is not registered", id)
	}
	mod, err := factory.Build(bkmodule.BuildContext{
		FSRoot: fsRoot,
		Decode: func(any) error {
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: build module %q: %w", id, err)
	}
	return mod, nil
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
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = errors.Join(err, k.closeMounted(ctx))
	err = errors.Join(err, k.runtime().Close())
	return err
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
	var err error
	err = errors.Join(err, k.closeMounted(ctx))
	if k.node != nil {
		err = errors.Join(err, k.node.Shutdown(ctx))
		return err
	}
	err = errors.Join(err, k.kernel.Shutdown(ctx))
	return err
}
