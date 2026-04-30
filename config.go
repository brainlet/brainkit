package brainkit

import (
	"log/slog"
	"path/filepath"

	"github.com/brainlet/brainkit/audio"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
)

// Config configures a brainkit runtime.
//
// All fields are optional with sensible defaults. The zero value creates a
// standalone in-memory control-plane runtime with no persistence. The embedded
// JS/TS runtime starts only when JSRuntime is true or a mounted module requests
// it.
type Config struct {
	// ClusterID identifies the logical group of runtimes. Default: "default".
	// All runtimes on the same transport with the same ClusterID discover each other.
	ClusterID string

	// Namespace identifies this runtime on the bus. Default: "user".
	Namespace string

	// CallerID identifies this runtime instance in message metadata. Default: Namespace.
	CallerID string

	// Transport configures the bus backend. One per Kit.
	// Zero value = Memory() (in-process GoChannel, no external backend linked).
	// Import github.com/brainlet/brainkit/transports before using
	// EmbeddedNATS(), NATS(url), AMQP(url), or Redis(url).
	Transport TransportConfig

	// FSRoot is the filesystem sandbox for deployed .ts code.
	FSRoot string

	// Storages configures named storage backends.
	// Deployments access them via storage("name") in .ts code.
	Storages map[string]StorageConfig

	// Vectors configures named vector store backends.
	// Deployments access them via vectorStore("name") in .ts code.
	Vectors map[string]VectorConfig

	// Providers configures AI providers. Nil = auto-detect from env
	// (OPENAI_API_KEY → openai, ANTHROPIC_API_KEY → anthropic, etc.)
	Providers []ProviderConfig

	// EnvVars overrides os.Getenv for specific keys within this runtime.
	EnvVars map[string]string

	// SecretKey is the master encryption key for the secret store.
	// Empty = env-only dev mode (secrets read from os.Getenv).
	SecretKey string

	// SecretStore overrides the auto-created secret store.
	// Most users leave this nil and set SecretKey instead.
	SecretStore SecretStore

	// Tracing enables distributed tracing with an auto-created MemoryTraceStore.
	Tracing bool

	// TraceStore overrides the auto-created trace store. Overrides the Tracing flag.
	TraceStore TraceStore

	// TraceSampleRate controls trace sampling (0.0–1.0). Default: 1.0.
	TraceSampleRate float64

	// Store provides persistence for deployments, schedules, and plugins.
	// Nil = no persistence (ephemeral). Use package stores for concrete stores.
	Store KitStore

	// Logger for structured logging. Nil = slog.Default().
	Logger *slog.Logger

	// LogHandler receives tagged log entries from .ts code and the runtime.
	LogHandler func(LogEntry)

	// ErrorHandler receives non-fatal errors (persistence failures, plugin errors).
	ErrorHandler func(error)

	// MaxConcurrency limits concurrent bus handler invocations. 0 = unlimited.
	MaxConcurrency int

	// JSRuntime enables the embedded JS/TS runtime. It is required for Deploy,
	// EvalTS/EvalModule, package deployment, workflow commands, harnesses, and
	// JS-backed storage/vector probes. Zero-value Config keeps the core control
	// plane light. JS-dependent modules such as eval, packages, testing,
	// workflow, and harness request it automatically; binaries must import
	// github.com/brainlet/brainkit/modules/jsruntime or
	// github.com/brainlet/brainkit/presets/standard so that request can be
	// satisfied.
	JSRuntime bool

	// MaxStackSize for the QuickJS runtime in bytes. Default: 1MB.
	MaxStackSize int

	// RetryPolicies maps topic glob patterns to retry configurations.
	RetryPolicies map[string]RetryPolicy

	// Modules are optional hot-mountable subsystems that extend the kernel.
	Modules []bkmodule.Module

	// Audio plays audio bytes from `.ts` agent code that calls
	// `new Audio(stream).play()`. Nil = silent (the polyfill is
	// always installed so portable agent code runs unchanged on
	// headless / server kits). For desktop playback, import
	// brainkit/audio/local and pass `local.New()`. Compose
	// multiple sinks with `audio.Composite(...)`.
	Audio audio.Sink
}

// toKernelConfig converts the flat Config to the internal engine KernelConfig.
func (c Config) toKernelConfig() types.KernelConfig {
	jsRuntime := c.JSRuntime || c.needsJSRuntime()

	cfg := types.KernelConfig{
		ClusterID:      c.ClusterID,
		RuntimeID:      runtimeID,
		Namespace:      c.Namespace,
		CallerID:       c.CallerID,
		FSRoot:         c.FSRoot,
		Storages:       c.Storages,
		Vectors:        c.Vectors,
		EnvVars:        c.EnvVars,
		SecretKey:      c.SecretKey,
		SecretStore:    c.SecretStore,
		JSRuntime:      jsRuntime,
		MaxStackSize:   c.MaxStackSize,
		MaxConcurrency: c.MaxConcurrency,
		RetryPolicies:  c.RetryPolicies,
		Logger:         c.Logger,
		LogHandler:     c.LogHandler,
	}

	if c.Audio != nil {
		cfg.AudioSink = c.Audio
	}

	// Convert []ProviderConfig → map[string]AIProviderRegistration. A nil
	// slice means "auto-detect from env"; an explicitly empty slice disables
	// auto-detection.
	if c.Providers != nil {
		cfg.AIProviders = make(map[string]types.AIProviderRegistration, len(c.Providers))
		for _, p := range c.Providers {
			cfg.AIProviders[p.name] = types.AIProviderRegistration{
				Type:   types.AIProviderType(p.typ),
				Config: p.toConfig(),
			}
		}
	}

	// TraceStore: only if explicitly set. Tracing module (session 05) owns
	// the real store; Tracer defaults to a nil store = no-op.
	if c.TraceStore != nil {
		cfg.TraceStore = c.TraceStore
	}

	// Store: explicit > nil. No auto-create from FSRoot — use brainkit.QuickStart
	// or pass an explicit Store for persistence.
	if c.Store != nil {
		cfg.Store = c.Store
	}

	// ErrorHandler adaptation (Config takes func(error), engine takes func(error, ErrorContext))
	if c.ErrorHandler != nil {
		cfg.ErrorHandler = func(err error, ctx types.ErrorContext) {
			c.ErrorHandler(err)
		}
	}

	return cfg
}

func (c Config) needsJSRuntime() bool {
	if c.JSRuntime || c.Audio != nil || c.LogHandler != nil || c.MaxStackSize != 0 {
		return true
	}
	if len(c.Storages) > 0 || len(c.Vectors) > 0 {
		return true
	}
	for _, mod := range c.Modules {
		if mod == nil {
			continue
		}
		if moduleNeedsJSRuntime(mod) {
			return true
		}
	}
	return false
}

func moduleNeedsJSRuntime(mod bkmodule.Module) bool {
	if moduleDependsOn(mod, "jsruntime") {
		return true
	}
	switch mod.ID() {
	case "eval", "packages", "testing", "workflow", "harness":
		return true
	default:
		return false
	}
}

func moduleDependsOn(mod bkmodule.Module, dependency string) bool {
	for _, dep := range moduleDependencies(mod) {
		if dep == dependency {
			return true
		}
	}
	return false
}

func moduleDependencies(mod bkmodule.Module) []string {
	if mod == nil {
		return nil
	}
	reporter, ok := mod.(bkmodule.DependencyReporter)
	if !ok {
		return nil
	}
	return reporter.Dependencies()
}

// toNodeConfig builds a NodeConfig for transport-connected mode.
func (c Config) toNodeConfig(kernelCfg types.KernelConfig) types.NodeConfig {
	// DeferRouterStart is handled internally by engine.NewNode
	kernelCfg.DeferRouterStart = true

	// For embedded NATS, derive JetStream store from FSRoot.
	natsStoreDir := ""
	if c.Transport.typ == "embedded" && c.FSRoot != "" {
		natsStoreDir = filepath.Join(c.FSRoot, "nats-data")
	}

	nc := types.NodeConfig{
		Kernel: kernelCfg,
		Messaging: types.MessagingConfig{
			Transport:    c.Transport.typ,
			NATSURL:      c.Transport.natsURL,
			NATSName:     c.Transport.natsName,
			AMQPURL:      c.Transport.amqpURL,
			RedisURL:     c.Transport.redisURL,
			NATSStoreDir: natsStoreDir,
		},
	}

	return nc
}
