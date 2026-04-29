package engine

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/secrets"
	"github.com/brainlet/brainkit/internal/syncx"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	bkmodule "github.com/brainlet/brainkit/module"
	agentsmod "github.com/brainlet/brainkit/modules/agents"
	provreg "github.com/brainlet/brainkit/modules/registry/providerreg"
	"github.com/brainlet/brainkit/modules/registry/storagehost"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/sdk"
)

// Kernel is the local brainkit runtime. Implements sdk.Runtime.
// It owns the light control plane and delegates optional JS runtime work.
type Kernel struct {
	// Domain handlers — all take narrow interfaces, not *Kernel.
	toolsDomain  *toolsmod.Domain
	agentsDomain *agentsmod.Domain

	Tools         *toolreg.ToolRegistry
	providers     *provreg.ProviderRegistry
	tracer        *tracing.Tracer
	streamTracker *streamTracker // heartbeat goroutine manager for active streams

	// Internal Watermill transport — always present
	transport     *transport.Transport
	router        *message.Router
	remote        *transport.RemoteClient
	host          *transport.Host
	ownsTransport bool // true if Kernel created the transport (false if injected by Node)

	// Shared-inbox reply router. Created after transport init.
	caller *sdk.Caller

	config      types.KernelConfig
	logger      *slog.Logger
	namespace   string
	callerID    string
	jsRuntime   JSRuntimeAttachment
	storageHost *storagehost.Manager

	secretStore secrets.SecretStore
	audit       *auditpkg.Recorder // centralized event log — nil-safe
	node        *Node              // optional back-reference, set by Node after creation

	mu     syncx.Mutex
	closed bool

	// Graceful shutdown
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	activeHandlers atomic.Int64
	draining       atomic.Bool

	// Metrics
	pumpCycles atomic.Int64
	busMetrics *transport.Metrics // per-topic bus message counts

	// Scheduling handler — set by modules/schedules.Module at mount time.
	// The QuickJS bridges (bus.schedule / bus.unschedule) and the schedule.*
	// bus commands dispatch through this. Nil when the module isn't active.
	scheduleHandler types.ScheduleHandler

	// Plugin checker — set by modules/plugins.Module at mount time for the
	// package-deploy `Requires.plugins` gate. Nil when the module isn't
	// active; modules/packages substitutes a deny-all stub.
	pluginChecker bkmodule.PluginChecker

	// Plugin restarter — set by modules/plugins.Module for the secrets module's
	// rotation-driven plugin restart. Nil when the module isn't active.
	pluginRestarter PluginRestarter

	// Health
	startedAt time.Time

	// Per-instance catalogs (built in NewKernel, before initTransport)
	catalog *commandRegistry
	events  *knownEventRegistry
}

// enterHandler marks a bus handler as active.
// Returns false if draining — caller should drop the message.
func (k *Kernel) enterHandler() bool {
	if k.draining.Load() {
		return false
	}
	k.activeHandlers.Add(1)
	return true
}

// exitHandler marks a bus handler as complete.
func (k *Kernel) exitHandler() {
	k.activeHandlers.Add(-1)
}

// IsDraining returns true during the drain phase of Shutdown.
func (k *Kernel) IsDraining() bool {
	return k.draining.Load()
}

// Caller returns the Kernel's shared-inbox reply router. Nil until
// transport init completes.
func (k *Kernel) Caller() *sdk.Caller { return k.caller }

// SetDraining sets the draining state. Used for testing.
func (k *Kernel) SetDraining(v bool) {
	k.draining.Store(v)
}

// waitForDrain polls until all active handlers finish or ctx expires.
func (k *Kernel) waitForDrain(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			active := k.activeHandlers.Load()
			if active > 0 {
				k.logger.Warn("drain timeout, forcing shutdown", slog.Int64("active_handlers", active))
			}
			return
		case <-ticker.C:
			if k.activeHandlers.Load() == 0 {
				k.logger.Info("drain complete")
				return
			}
		}
	}
}

func (k *Kernel) nextDeployOrder() int {
	if k.jsRuntime == nil {
		return 0
	}
	return k.jsRuntime.NextDeployOrder()
}

// HasJSRuntime reports whether this kernel owns the embedded JS/TS runtime.
func (k *Kernel) HasJSRuntime() bool {
	return k != nil && k.jsRuntime != nil
}

// Scheduling is in kernel_scheduling.go

// Failure handling (retry, dead letter, error events) is in kernel_failure.go

// NewKernel creates a local runtime with no attached transport.
func NewKernel(cfg types.KernelConfig) (*Kernel, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	// Auto-detect AI providers from OS env + EnvVars before sandbox creation
	autoDetectProviders(&cfg)

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	sharedTools := cfg.SharedTools
	if sharedTools == nil {
		sharedTools = toolreg.New()
	}

	kernel := &Kernel{
		Tools:        sharedTools,
		config:       cfg,
		logger:       logger,
		namespace:    cfg.Namespace,
		callerID:     cfg.CallerID,
		agentsDomain: agentsmod.NewDomain(),
	}
	kernel.shutdownCtx, kernel.shutdownCancel = context.WithCancel(context.Background())

	// Cleanup stack: each resource allocation pushes its cleanup function.
	// On failure, all cleanups execute in reverse order. On success, the
	// slice is nilled — Kernel.Close() owns resource lifecycle from then on.
	cleanups := []func(){kernel.shutdownCancel}
	fail := func(err error) (*Kernel, error) {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		return nil, err
	}

	if err := kernel.initProviders(cfg, nil); err != nil {
		return fail(err)
	}

	// Initialize secret store
	kernel.secretStore = resolveSecretStore(cfg, logger)

	// Initialize tracer
	sampleRate := cfg.TraceSampleRate
	if sampleRate == 0 {
		sampleRate = 1.0
	}
	kernel.tracer = tracing.NewTracer(cfg.TraceStore, sampleRate)

	// ToolsDomain needs tracer — constructed here after tracer init.
	kernel.toolsDomain = toolsmod.NewDomain(sharedTools, nil, kernel.tracer, kernel.audit, cfg.CallerID, cfg.RuntimeID)

	kernel.streamTracker = newStreamTracker(kernel, 10*time.Second, 10*time.Minute)

	// Build per-instance catalogs
	kernel.catalog = buildCommandCatalog()
	kernel.events = buildEventCatalog(kernel.catalog)

	// Initial probe — probes module (session 05) owns periodic probing.
	go kernel.ProbeAll()

	if err := kernel.initTransport(cfg); err != nil {
		return fail(err)
	}
	// If DeferRouterStart: caller (Node) registers all bindings and starts the router

	kernel.initPersistence(cfg)

	if cleanup := kernel.initAudit(cfg); cleanup != nil {
		cleanups = append(cleanups, cleanup)
	}

	kernel.startedAt = time.Now()

	// Success — Kernel.Close() now owns all resources.
	// Nil out cleanups so fail() is harmless if called accidentally.
	cleanups = nil
	_ = cleanups

	return kernel, nil
}
