package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/brainlet/brainkit/internal/syncx"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	auditpkg "github.com/brainlet/brainkit/internal/audit"
	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/internal/libsql"
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/deploy"
	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/internal/secrets"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Kernel is the local brainkit runtime. Implements sdk.Runtime.
// It owns JS runtime state and an internal Watermill transport.
type Kernel struct {
	// Domain handlers — all take narrow interfaces, not *Kernel.
	// MetricsDomain is the exception (cross-cutting, reads from multiple subsystems).
	toolsDomain         *ToolsDomain
	agentsDomain        *AgentsDomain
	secretsDomain       *SecretsDomain
	mcpDomain           *MCPDomain
	registryDomain      *RegistryDomain
	tracingDomain       *TracingDomain
	metricsDomain       *MetricsDomain
	lifecycle           *LifecycleDomain
	packageDeployDomain *PackageDeployDomain
	testingDomain       *TestingDomain

	Tools           *toolreg.ToolRegistry
	mcp             *mcppkg.MCPManager
	providers       *provreg.ProviderRegistry
	tracer        *tracing.Tracer
	streamTracker *streamTracker // heartbeat goroutine manager for active streams

	// Internal Watermill transport — always present
	transport     *transport.Transport
	router        *message.Router
	remote        *transport.RemoteClient
	host          *transport.Host
	ownsTransport bool // true if Kernel created the transport (false if injected by Node)

	config    types.KernelConfig
	logger    *slog.Logger
	namespace string
	callerID  string
	bridge    *jsbridge.Bridge
	agents    *agentembed.Sandbox
	storages  map[string]*libsql.Server

	secretStore   secrets.SecretStore
	audit         *auditpkg.Recorder // centralized event log — nil-safe
	auditStore    auditpkg.Store     // underlying store for query access
	node          *Node              // optional back-reference, set by Node after creation
	deploymentMgr *DeploymentManager // owns deploy/teardown/eval lifecycle

	bridgeSubs map[string]func()

	mu     syncx.Mutex
	closed bool

	// Graceful shutdown
	activeHandlers atomic.Int64
	draining       atomic.Bool

	// Metrics
	pumpCycles atomic.Int64
	busMetrics *transport.Metrics // per-topic bus message counts

	// Schedules
	schedules map[string]*scheduleEntry

	// Health
	startedAt time.Time
}

type scheduleEntry struct {
	types.PersistedSchedule
	timer *time.Timer
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
	return k.deploymentMgr.nextDeployOrder()
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
		Tools:      sharedTools,
		config:     cfg,
		logger:     logger,
		namespace:  cfg.Namespace,
		callerID:   cfg.CallerID,
		storages:   make(map[string]*libsql.Server),
		bridgeSubs: make(map[string]func()),
		schedules:  make(map[string]*scheduleEntry),
	}
	providers := make(map[string]agentembed.ProviderConfig)
	for name, reg := range cfg.AIProviders {
		pc := extractProviderCredentials(reg)
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	// Cleanup stack: each resource allocation pushes its cleanup function.
	// On failure, all cleanups execute in reverse order. On success, the
	// slice is nilled — Kernel.Close() owns resource lifecycle from then on.
	var cleanups []func()
	fail := func(err error) (*Kernel, error) {
		for i := len(cleanups) - 1; i >= 0; i-- {
			cleanups[i]()
		}
		return nil, err
	}

	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
		CWD:          cfg.FSRoot,
		FetchSpanHook: func(method, url string) func(int, error) {
			// Lazy reference — tracer is initialized after sandbox creation
			if kernel.tracer == nil {
				return nil
			}
			span := kernel.tracer.StartSpan("fetch", context.Background())
			span.SetAttribute("method", method)
			span.SetAttribute("url", url)
			return func(statusCode int, err error) {
				if statusCode > 0 {
					span.SetAttribute("status", strconv.Itoa(statusCode))
				}
				span.End(err)
			}
		},
	})
	if err != nil {
		return fail(fmt.Errorf("brainkit: create runtime: %w", err))
	}
	cleanups = append(cleanups, func() { agentSandbox.Close() })
	kernel.agents = agentSandbox
	kernel.bridge = agentSandbox.Bridge()

	kernel.agentsDomain = newAgentsDomain()

	kernel.registerBridges()

	// Start sqlite storage bridges (must happen before loadRuntime)
	bridgeURLs, err := kernel.initStorages(cfg)
	if err != nil {
		return fail(fmt.Errorf("brainkit: start storage: %w", err))
	}
	cleanups = append(cleanups, func() {
		for _, srv := range kernel.storages {
			_ = srv.Close()
		}
	})

	if err := kernel.initProviders(cfg, bridgeURLs); err != nil {
		return fail(err)
	}

	// Initialize secret store
	kernel.secretStore = resolveSecretStore(cfg, logger)

	// pluginCheckerFactory closure captures kernel.node (set later by Node)
	kernel.packageDeployDomain = newPackageDeployDomain(
		kernel,
		kernel.secretStore,
		func() deploy.PluginChecker {
			return &pluginCheckerImpl{node: kernel.node}
		},
	)
	// SecretsDomain constructed later (needs kernel.remote for bus publishing)

	// Initialize tracer
	sampleRate := cfg.TraceSampleRate
	if sampleRate == 0 {
		sampleRate = 1.0
	}
	kernel.tracer = tracing.NewTracer(cfg.TraceStore, sampleRate)

	kernel.deploymentMgr = NewDeploymentManager(DeploymentManagerConfig{
		Bridge:       kernel.bridge,
		Agents:       kernel.agents,
		Tracer:       kernel.tracer,
		Store:        cfg.Store,
		ErrorHandler: cfg.ErrorHandler,
		Logger:       logger,
		ToolCleanup: func(id string) {
			kernel.toolsDomain.Unregister(context.Background(), id)
		},
		AgentCleanup: func(id string) {
			kernel.agentsDomain.Unregister(context.Background(), id)
		},
		SubCleanup: func(id string) {
			kernel.mu.Lock()
			cancel := kernel.bridgeSubs[id]
			delete(kernel.bridgeSubs, id)
			kernel.mu.Unlock()
			if cancel != nil {
				cancel()
			}
		},
		ScheduleCleanup: func(id string) {
			kernel.removeSchedule(id)
		},
	})

	// Upgrade Mastra storage from InMemoryStore to configured backend.
	// patches.js creates _storeHolder with InMemoryStore. If a storage backend is
	// configured, resolve it, call init() (creates mastra_workflow_snapshot table + others),
	// and replace the holder's store so all Mastra persistence goes to the real database.
	// Must run after deploymentMgr construction — upgradeMastraStorage calls EvalTS.
	if len(cfg.Storages) > 0 {
		kernel.upgradeMastraStorage()
	}

	// ToolsDomain needs tracer — constructed here after tracer init
	kernel.toolsDomain = newToolsDomain(sharedTools, kernel.bridge, kernel.tracer, kernel.audit, cfg.CallerID, cfg.RuntimeID)

	kernel.testingDomain = newTestingDomain(kernel, kernel)
	kernel.registryDomain = newRegistryDomain(kernel.providers)
	kernel.tracingDomain = newTracingDomain(cfg.TraceStore)
	kernel.metricsDomain = newMetricsDomain(kernel)
	kernel.streamTracker = newStreamTracker(kernel, 10*time.Second, 10*time.Minute)

	// Start periodic probing if configured
	kernel.startPeriodicProbing()

	// Initial probe — don't wait for first periodic tick
	go kernel.ProbeAll()

	if len(cfg.MCPServers) > 0 {
		kernel.mcp = mcppkg.New()
		kernel.mcpDomain = newMCPDomain(kernel.mcp)
		for name, serverCfg := range cfg.MCPServers {
			if err := kernel.mcp.Connect(context.Background(), name, serverCfg); err != nil {
				types.InvokeErrorHandler(cfg.ErrorHandler, &sdkerrors.TransportError{
					Operation: "MCP.Connect:" + name, Cause: err,
				}, types.ErrorContext{Operation: "ConnectMCP", Component: "mcp", Source: name})
				continue
			}
			for _, tool := range kernel.mcp.ListToolsForServer(name) {
				toolCopy := tool
				fullName := toolreg.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
				_ = kernel.Tools.Register(toolreg.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Owner:       "mcp",
					Package:     toolCopy.ServerName,
					Version:     "1.0.0",
					Description: toolCopy.Description,
					InputSchema: toolCopy.InputSchema,
					Executor: &toolreg.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return kernel.mcp.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
						},
					},
				})
			}
		}
	}
	if kernel.mcpDomain == nil {
		kernel.mcpDomain = newMCPDomain(nil) // nil-safe — returns types.ErrMCPNotConfigured
	}

	if err := kernel.initTransport(cfg); err != nil {
		return fail(err)
	}
	// If DeferRouterStart: caller (Node) registers all bindings and starts the router

	// Start background job pump — processes qctx.Schedule'd callbacks
	// even when no EvalTS is active. Enables deployed .ts services to
	// receive bus messages asynchronously.
	kernel.startJobPump()

	kernel.initPersistence(cfg)

	if cleanup := kernel.initAudit(cfg); cleanup != nil {
		cleanups = append(cleanups, cleanup)
	}

	kernel.lifecycle = newLifecycleDomain(kernel.deploymentMgr, kernel.remote, kernel.audit, cfg.RuntimeID)

	kernel.startedAt = time.Now()

	// Periodic metric snapshots — when verbose audit is enabled, snapshot metrics every 60s
	if kernel.audit != nil && kernel.audit.IsVerbose() {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if kernel.draining.Load() {
						return
					}
					kernel.audit.MetricsSnapshot(kernel.Metrics())
				}
			}
		}()
	}

	// Success — Kernel.Close() now owns all resources.
	// Nil out cleanups so fail() is harmless if called accidentally.
	cleanups = nil
	_ = cleanups

	return kernel, nil
}
