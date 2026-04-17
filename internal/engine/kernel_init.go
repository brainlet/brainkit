package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	auditpkg "github.com/brainlet/brainkit/internal/audit"
	"github.com/brainlet/brainkit/internal/bus/caller"
	js "github.com/brainlet/brainkit/internal/contract"
	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
)

func (k *Kernel) initProviders(cfg types.KernelConfig, bridgeURLs map[string]string) error {
	// Initialize the provider registry BEFORE loadRuntime so that JS code
	// evaluated during runtime init (patches.js, resolve.js, kit_runtime.js)
	// can access the registry via __go_registry_has / __go_registry_resolve.
	k.providers = provreg.New(cfg.Probe)
	for name, reg := range cfg.AIProviders {
		k.providers.RegisterAIProvider(name, reg)
	}
	// Register all storages and vectors in the provider registry
	k.registerStorages(cfg, bridgeURLs)
	if err := k.registerVectors(cfg, bridgeURLs); err != nil {
		return fmt.Errorf("brainkit: register vectors: %w", err)
	}

	obsEnabled := cfg.Observability.Enabled == nil || *cfg.Observability.Enabled
	obsStrategy := cfg.Observability.Strategy
	if obsStrategy == "" {
		obsStrategy = "realtime"
	}
	obsServiceName := cfg.Observability.ServiceName
	if obsServiceName == "" {
		obsServiceName = "brainkit"
	}
	k.bridge.Eval("__obs_config.js", fmt.Sprintf(
		`globalThis.`+js.JSObsConfig+` = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	// Inject provider configs into the JS runtime for ai.generate/embed model resolution.
	// The JS runtime's resolveModel() reads from globalThis.__kit_providers.
	if len(cfg.AIProviders) > 0 {
		provMap := make(map[string]map[string]string)
		for name, reg := range cfg.AIProviders {
			creds := extractProviderCredentials(reg)
			entry := map[string]string{"APIKey": creds.APIKey}
			if creds.BaseURL != "" {
				entry["BaseURL"] = creds.BaseURL
			}
			provMap[name] = entry
		}
		provJSON, _ := json.Marshal(provMap)
		k.bridge.Eval("__providers.js", fmt.Sprintf(
			`globalThis.`+js.JSProviders+` = %s;`, string(provJSON),
		))
	}

	if err := k.loadRuntime(); err != nil {
		return err
	}

	return nil
}

func (k *Kernel) initTransport(cfg types.KernelConfig) error {
	// Set up internal Watermill transport + router
	createdTransport := false
	if cfg.Transport != nil {
		k.transport = cfg.Transport.(*transport.Transport)
		k.ownsTransport = false
	} else {
		transport, err := transport.NewTransportSet(transport.TransportConfig{Type: "memory", Namespace: cfg.Namespace})
		if err != nil {
			return fmt.Errorf("brainkit: internal transport: %w", err)
		}
		k.transport = transport
		k.ownsTransport = true
		createdTransport = true
	}

	k.remote = transport.NewRemoteClientWithTransport(cfg.Namespace, cfg.CallerID, k.transport)
	k.remote.SetIdentity(cfg.ClusterID, cfg.RuntimeID)

	// SecretsDomain — needs kernel.remote for bus event publishing
	k.secretsDomain = newSecretsDomain(k.secretStore, k.remote, k.audit, cfg.CallerID, nil, k.refreshProviderIfSecret)

	wmLogger := watermill.NopLogger{}
	router, err := message.NewRouter(message.RouterConfig{}, wmLogger)
	if err != nil {
		if createdTransport {
			_ = k.transport.Close()
			k.transport = nil
			k.ownsTransport = false
		}
		return fmt.Errorf("brainkit: router: %w", err)
	}

	busMetrics := transport.NewMetrics()
	k.busMetrics = busMetrics
	router.AddMiddleware(
		transport.DepthMiddleware,
		transport.CallerIDMiddleware(cfg.CallerID),
		transport.MetricsMiddleware(busMetrics),
	)
	if cfg.MaxConcurrency > 0 {
		router.AddMiddleware(transport.MaxConcurrencyMiddleware(cfg.MaxConcurrency))
	}

	k.router = router
	k.host = transport.NewHostWithTransport(cfg.Namespace, router, k.transport)

	// Construct the shared-inbox reply router. Uses k (Kernel implements
	// sdk.Runtime) — its SubscribeRaw resolves the inbox topic into the
	// current namespace, so replies land here even for cross-namespace calls.
	runtimeID := cfg.RuntimeID
	if runtimeID == "" {
		runtimeID = watermill.NewUUID()
	}
	c, err := caller.NewCaller(k, runtimeID, k.logger)
	if err != nil {
		if createdTransport {
			_ = k.transport.Close()
			k.transport = nil
			k.ownsTransport = false
		}
		return fmt.Errorf("brainkit: caller: %w", err)
	}
	k.caller = c

	if !cfg.DeferRouterStart {
		// Standalone Kernel: register kernel-only bindings and start router now
		k.host.RegisterCommands(commandBindingsForKernel(k))
		go func() {
			_ = router.Run(context.Background())
		}()
		<-router.Running()
	}

	return nil
}

func (k *Kernel) initPersistence(cfg types.KernelConfig) {
	// Auto-redeploy persisted .ts deployments
	if cfg.Store != nil {
		k.redeployPersistedDeployments()
	}

	// Subscribe to deployment propagation events (for multi-replica sync).
	// Uses fan-out subscriber so ALL replicas receive deploy/teardown events.
	if cfg.Store != nil {
		k.subscribeToDeploymentPropagation()
	}

	// Restore persisted schedules
	if cfg.Store != nil {
		k.restoreSchedules()
	}

	// Restart workflows that were active before the previous Kernel shutdown.
	// Requires both deployment persistence (Store) and Mastra storage (Storages).
	if cfg.Store != nil && len(cfg.Storages) > 0 {
		k.restartActiveWorkflows()
	}
}

func (k *Kernel) initAudit(cfg types.KernelConfig) func() {
	// Initialize audit event log — auto-create SQLite store alongside KitStore
	verbosity := auditpkg.VerbosityNormal
	if cfg.AuditVerbose {
		verbosity = auditpkg.VerbosityVerbose
	}
	makeRecorder := func(store auditpkg.Store) *auditpkg.Recorder {
		return auditpkg.NewRecorderWithConfig(auditpkg.RecorderConfig{
			Store: store, RuntimeID: cfg.RuntimeID, Namespace: cfg.Namespace, Verbosity: verbosity,
		})
	}
	if cfg.AuditStore != nil {
		k.auditStore = cfg.AuditStore
		k.audit = makeRecorder(cfg.AuditStore)
		return nil
	}
	if cfg.FSRoot != "" {
		auditStore, auditErr := auditpkg.NewSQLiteStore(filepath.Join(cfg.FSRoot, "brainkit-audit.db"))
		if auditErr == nil {
			k.auditStore = auditStore
			k.audit = makeRecorder(auditStore)
			return func() { auditStore.Close() }
		}
	}

	return nil
}
