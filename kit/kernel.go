package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/jsbridge"
	"github.com/brainlet/brainkit/internal/libsql"
	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/sdk/messages"
)

// Kernel is the local brainkit runtime. Implements sdk.Runtime.
// It owns JS/WASM/runtime state and an internal Watermill transport.
type Kernel struct {
	// Domain handlers (internal — accessed via command catalog, not directly)
	ai              *AIDomain
	toolsDomain     *ToolsDomain
	agentsDomain    *AgentsDomain
	fsDomain        *FSDomain
	memoryDomain    *MemoryDomain
	workflowsDomain *WorkflowsDomain
	vectorsDomain   *VectorsDomain
	wasmDomainInst  *WASMDomain
	lifecycle       *LifecycleDomain
	mcpDomainInst   *MCPDomain

	Tools *registry.ToolRegistry
	mcp   *mcppkg.MCPManager

	// Internal Watermill transport — always present
	transport      *messaging.Transport
	router         *message.Router
	remote         *messaging.RemoteClient
	host           *messaging.Host
	ownsTransport  bool // true if Kernel created the transport (false if injected by Node)

	config    KernelConfig
	namespace string
	callerID  string
	bridge    *jsbridge.Bridge
	agents    *agentembed.Sandbox
	wasm      *WASMService
	storages  map[string]*libsql.Server

	deployments map[string]*deploymentInfo
	bridgeSubs  map[string]func()

	mu     sync.Mutex
	closed bool
}

// NewKernel creates a local runtime with no attached transport.
func NewKernel(cfg KernelConfig) (*Kernel, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	sharedTools := cfg.SharedTools
	if sharedTools == nil {
		sharedTools = registry.New()
	}

	kernel := &Kernel{
		Tools:       sharedTools,
		config:      cfg,
		namespace:   cfg.Namespace,
		callerID:    cfg.CallerID,
		storages:    make(map[string]*libsql.Server),
		deployments: make(map[string]*deploymentInfo),
		bridgeSubs:  make(map[string]func()),
	}
	providers := make(map[string]agentembed.ProviderConfig)
	for name, pc := range cfg.Providers {
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create runtime: %w", err)
	}
	kernel.agents = agentSandbox
	kernel.bridge = agentSandbox.Bridge()

	kernel.ai = newAIDomain(kernel)
	kernel.toolsDomain = newToolsDomain(kernel)
	kernel.agentsDomain = newAgentsDomain(kernel)
	kernel.fsDomain = newFSDomain(kernel)
	kernel.memoryDomain = newMemoryDomain(kernel)
	kernel.workflowsDomain = newWorkflowsDomain(kernel)
	kernel.vectorsDomain = newVectorsDomain(kernel)

	kernel.registerBridges()

	for name, scfg := range cfg.Storages {
		if err := kernel.addStorageInternal(name, scfg); err != nil {
			for _, srv := range kernel.storages {
				_ = srv.Close()
			}
			agentSandbox.Close()
			return nil, fmt.Errorf("brainkit: start storage %q: %w", name, err)
		}
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
	kernel.bridge.Eval("__obs_config.js", fmt.Sprintf(
		`globalThis.__brainkit_obs_config = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	if err := kernel.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Inject provider configs into the JS runtime for ai.generate/embed model resolution.
	// The JS runtime's resolveModel() reads from globalThis.__kit_providers.
	if len(cfg.Providers) > 0 {
		provJSON, _ := json.Marshal(cfg.Providers)
		kernel.bridge.Eval("__providers.js", fmt.Sprintf(
			`globalThis.__kit_providers = %s;`, string(provJSON),
		))
	}

	kernel.wasm = newWASMService(kernel)
	kernel.wasmDomainInst = newWASMDomain(kernel, kernel.wasm)
	kernel.lifecycle = newLifecycleDomain(kernel)

	if cfg.Store != nil {
		if err := kernel.wasm.loadFromStore(cfg.Store); err != nil {
			log.Printf("[brainkit] warning: failed to load persisted data: %v", err)
		}
	}

	if len(cfg.MCPServers) > 0 {
		kernel.mcp = mcppkg.New()
		kernel.mcpDomainInst = newMCPDomain(kernel, kernel.mcp)
		for name, serverCfg := range cfg.MCPServers {
			if err := kernel.mcp.Connect(context.Background(), name, serverCfg); err != nil {
				continue
			}
			for _, tool := range kernel.mcp.ListToolsForServer(name) {
				toolCopy := tool
				fullName := registry.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
				_ = kernel.Tools.Register(registry.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Owner:       "mcp",
					Package:     toolCopy.ServerName,
					Version:     "1.0.0",
					Description: toolCopy.Description,
					InputSchema: toolCopy.InputSchema,
					Executor: &registry.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return kernel.mcp.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
						},
					},
				})
			}
		}
	} else {
		kernel.mcpDomainInst = newMCPDomain(kernel, nil)
	}

	// Set up internal Watermill transport + router
	if cfg.Transport != nil {
		kernel.transport = cfg.Transport
		kernel.ownsTransport = false
	} else {
		transport, err := messaging.NewTransportSet(messaging.TransportConfig{Type: "memory"})
		if err != nil {
			agentSandbox.Close()
			return nil, fmt.Errorf("brainkit: internal transport: %w", err)
		}
		kernel.transport = transport
		kernel.ownsTransport = true
	}

	kernel.remote = messaging.NewRemoteClientWithTransport(cfg.Namespace, cfg.CallerID, kernel.transport)

	logger := watermill.NopLogger{}
	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		if kernel.ownsTransport {
			kernel.transport.Close()
		}
		agentSandbox.Close()
		return nil, fmt.Errorf("brainkit: router: %w", err)
	}

	metrics := messaging.NewMetrics()
	router.AddMiddleware(
		messaging.DepthMiddleware,
		messaging.CallerIDMiddleware(cfg.CallerID),
		messaging.MetricsMiddleware(metrics),
	)

	kernel.router = router
	kernel.host = messaging.NewHostWithTransport(cfg.Namespace, router, kernel.transport)

	if !cfg.DeferRouterStart {
		// Standalone Kernel: register kernel-only bindings and start router now
		kernel.host.RegisterCommands(commandBindingsForKernel(kernel))
		go func() {
			_ = router.Run(context.Background())
		}()
		<-router.Running()
	}
	// If DeferRouterStart: caller (Node) registers all bindings and starts the router

	return kernel, nil
}

// --- sdk.Runtime implementation ---

// PublishRaw sends a message to a topic. Returns correlationID.
func (k *Kernel) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic. Subscription is active before this returns.
func (k *Kernel) SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (func(), error) {
	return k.remote.SubscribeRaw(ctx, topic, handler)
}

// --- sdk.CrossNamespaceRuntime implementation ---

// PublishRawTo publishes to a specific Kit's namespace.
func (k *Kernel) PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (string, error) {
	return k.remote.PublishRawToNamespace(ctx, targetNamespace, topic, payload)
}

// SubscribeRawTo subscribes to a topic in a specific Kit's namespace.
func (k *Kernel) SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(messages.Message)) (func(), error) {
	return k.remote.SubscribeRawToNamespace(ctx, targetNamespace, topic, handler)
}

// publish is an internal convenience for fire-and-forget event publishing.
func (k *Kernel) publish(ctx context.Context, topic string, payload json.RawMessage) error {
	_, err := k.remote.PublishRaw(ctx, topic, payload)
	return err
}

// subscribe is an internal convenience for subscribing with raw byte handler.
func (k *Kernel) subscribe(topic string, handler func([]byte)) (func(), error) {
	return k.remote.SubscribeRaw(context.Background(), topic, func(msg messages.Message) {
		handler(msg.Payload)
	})
}

// Close shuts down the runtime and all local services.
func (k *Kernel) Close() error {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return nil
	}
	k.closed = true
	subs := make([]func(), 0, len(k.bridgeSubs))
	for _, cancel := range k.bridgeSubs {
		subs = append(subs, cancel)
	}
	k.bridgeSubs = map[string]func(){}
	k.mu.Unlock()

	for _, cancel := range subs {
		cancel()
	}

	var firstErr error
	collect := func(err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	// Shut down router first (stops processing messages)
	if k.router != nil {
		collect(k.router.Close())
	}

	if k.agentsDomain != nil && k.agents != nil {
		k.agentsDomain.UnregisterAllForKit(k.agents.ID())
	}
	if k.mcp != nil {
		collect(k.mcp.Close())
	}
	if k.wasm != nil {
		k.wasm.close()
	}
	if k.config.Store != nil {
		collect(k.config.Store.Close())
	}
	if k.agents != nil {
		k.agents.Close()
	}
	for name, srv := range k.storages {
		if err := srv.Close(); err != nil {
			collect(fmt.Errorf("storage %q: %w", name, err))
		}
	}

	// Shut down transport last (only if we own it — Node owns its own)
	if k.ownsTransport && k.transport != nil {
		collect(k.transport.Close())
	}

	return firstErr
}

// Namespace returns the runtime namespace.
func (k *Kernel) Namespace() string { return k.namespace }

// CallerID returns the runtime identity.
func (k *Kernel) CallerID() string { return k.callerID }

// CreateAgent creates a persistent agent in the runtime.
func (k *Kernel) CreateAgent(cfg agentembed.AgentConfig) (*agentembed.Agent, error) {
	return k.agents.CreateAgent(cfg)
}

// AddStorage starts a new named embedded SQLite storage and makes it available to JS.
func (k *Kernel) AddStorage(name string, cfg StorageConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, exists := k.storages[name]; exists {
		return fmt.Errorf("brainkit: storage %q already exists", name)
	}
	return k.addStorageInternal(name, cfg)
}

// RemoveStorage stops and removes a named storage.
func (k *Kernel) RemoveStorage(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	srv, ok := k.storages[name]
	if !ok {
		return fmt.Errorf("brainkit: storage %q not found", name)
	}
	_ = srv.Close()
	delete(k.storages, name)
	k.bridge.Eval("__storage_remove.js", fmt.Sprintf(
		`delete globalThis.__brainkit_storages[%q]`, name,
	))
	return nil
}

// StorageURL returns the HTTP URL for a named storage bridge.
func (k *Kernel) StorageURL(name string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		return srv.URL()
	}
	return ""
}

// ListResources returns all tracked resources, optionally filtered by type.
func (k *Kernel) ListResources(resourceType ...string) ([]ResourceInfo, error) {
	filter := ""
	if len(resourceType) > 0 {
		filter = resourceType[0]
	}
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.list(%q))`, filter)
	result, err := k.EvalTS(context.Background(), "__list_resources.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}
	return resources, nil
}

// ResourcesFrom returns all resources created by a specific .ts file.
func (k *Kernel) ResourcesFrom(filename string) ([]ResourceInfo, error) {
	code := fmt.Sprintf(`return JSON.stringify(globalThis.__kit_registry.listBySource(%q))`, filename)
	result, err := k.EvalTS(context.Background(), "__resources_from.ts", code)
	if err != nil {
		return nil, err
	}
	var resources []ResourceInfo
	if err := json.Unmarshal([]byte(result), &resources); err != nil {
		return nil, fmt.Errorf("resources from: %w", err)
	}
	return resources, nil
}

// TeardownFile removes all resources created by a specific .ts file.
func (k *Kernel) TeardownFile(filename string) (int, error) {
	code := fmt.Sprintf(`
		var resources = globalThis.__kit_registry.listBySource(%q);
		var count = 0;
		for (var i = resources.length - 1; i >= 0; i--) {
			globalThis.__kit_registry.unregister(resources[i].type, resources[i].id);
			count++;
		}
		return JSON.stringify(count);
	`, filename)
	result, err := k.EvalTS(context.Background(), "__teardown_file.ts", code)
	if err != nil {
		return 0, err
	}
	var count int
	if err := json.Unmarshal([]byte(result), &count); err != nil {
		return 0, nil
	}
	return count, nil
}

// RemoveResource removes a specific resource by type and ID.
func (k *Kernel) RemoveResource(resourceType, id string) error {
	code := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.unregister(%q, %q);
		return JSON.stringify(entry !== null);
	`, resourceType, id)
	_, err := k.EvalTS(context.Background(), "__remove_resource.ts", code)
	return err
}

// ListWASMModules returns metadata for all compiled WASM modules.
func (k *Kernel) ListWASMModules() ([]WASMModuleInfo, error) {
	k.wasm.mu.Lock()
	defer k.wasm.mu.Unlock()
	infos := make([]WASMModuleInfo, 0, len(k.wasm.modules))
	for _, mod := range k.wasm.modules {
		infos = append(infos, WASMModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}
	return infos, nil
}

// GetWASMModule returns metadata for a specific module by name.
func (k *Kernel) GetWASMModule(name string) (*WASMModuleInfo, error) {
	k.wasm.mu.Lock()
	mod, ok := k.wasm.modules[name]
	k.wasm.mu.Unlock()
	if !ok {
		return nil, nil
	}
	return &WASMModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}, nil
}

// RemoveWASMModule unloads a compiled module by name via the typed domain.
func (k *Kernel) RemoveWASMModule(name string) error {
	_, err := k.wasmDomainInst.Remove(context.Background(), messages.WasmRemoveMsg{Name: name})
	return err
}

// DeployWASM activates a compiled shard via the typed domain.
func (k *Kernel) DeployWASM(name string) (*ShardDescriptor, error) {
	resp, err := k.wasmDomainInst.Deploy(context.Background(), messages.WasmDeployMsg{Name: name})
	if err != nil {
		return nil, err
	}
	return &ShardDescriptor{
		Module:     resp.Module,
		Mode:       resp.Mode,
		Handlers:   resp.Handlers,
		DeployedAt: time.Now(),
	}, nil
}

// UndeployWASM removes all event subscriptions for a deployed shard.
func (k *Kernel) UndeployWASM(name string) error {
	_, err := k.wasmDomainInst.Undeploy(context.Background(), messages.WasmUndeployMsg{Name: name})
	return err
}

// DescribeWASM returns the shard's registrations via the typed domain.
func (k *Kernel) DescribeWASM(name string) (*ShardDescriptor, error) {
	resp, err := k.wasmDomainInst.Describe(context.Background(), messages.WasmDescribeMsg{Name: name})
	if err != nil {
		return nil, err
	}
	return &ShardDescriptor{
		Module:     resp.Module,
		Mode:       resp.Mode,
		Handlers:   resp.Handlers,
		DeployedAt: time.Now(),
	}, nil
}

// ListDeployedWASM returns all active shard descriptors.
func (k *Kernel) ListDeployedWASM() []ShardDescriptor {
	return k.wasm.listDeployedShards()
}

// InjectWASMEvent manually triggers a shard handler.
func (k *Kernel) InjectWASMEvent(shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
	return k.wasm.invokeShardHandler(context.Background(), shardName, topic, payload)
}

func (k *Kernel) addStorageInternal(name string, cfg StorageConfig) error {
	srv, err := libsql.NewServer(cfg.Path)
	if err != nil {
		return err
	}
	k.storages[name] = srv
	k.bridge.Eval("__storage_add.js", fmt.Sprintf(
		`if (!globalThis.__brainkit_storages) globalThis.__brainkit_storages = {};
		 globalThis.__brainkit_storages[%q] = %q;`, name, srv.URL(),
	))
	return nil
}

// evalDomain marshals a request into JS globals and evaluates code atomically.
// Replaces per-domain evalAI/evalMemory/evalVector/evalWorkflow methods.
func (k *Kernel) evalDomain(ctx context.Context, req any, filename, code string) (json.RawMessage, error) {
	reqJSON, _ := json.Marshal(req)
	wrappedCode := fmt.Sprintf(`
		globalThis.__pending_req = %s;
		%s
	`, string(reqJSON), code)
	resultJSON, err := k.EvalTS(ctx, filename, wrappedCode)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(resultJSON), nil
}

// EvalTS runs .ts-style code with brainlet imports destructured.
func (k *Kernel) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		return await globalThis.__kitRunWithSource(%q, async () => {
			const { agent, createTool, createSubagent, createWorkflow, createStep, createMemory, z, ai, wasm, tools, tool, bus, agents, mcp, sandbox, output, Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore, LibSQLVector, PgVector, MongoDBVector, generateText, streamText, generateObject, streamObject, createWorkflowRun, resumeWorkflow, createScorer, runEvals, scorers, processors, RequestContext, MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool, rerank, rerankWithScorer, Workspace, LocalFilesystem, LocalSandbox, createHarness } = globalThis.__kit;
			%s
		});
	})()`, filename, code)

	if k.bridge.IsEvalBusy() {
		return k.bridge.EvalOnJSThread(filename, wrapped)
	}
	return k.agents.Eval(ctx, filename, wrapped)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kernel) EvalModule(ctx context.Context, filename, code string) (string, error) {
	k.bridge.Eval("__clear_result.js", `delete globalThis.__module_result`)
	val, err := k.bridge.EvalAsyncModule(filename, code)
	if err != nil {
		return "", fmt.Errorf("brainkit: eval module: %w", err)
	}
	if val != nil {
		val.Free()
	}
	result, err := k.bridge.Eval("__get_result.js",
		`typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}

// RegisterTool is a convenience method for registering typed Go tools.
func RegisterTool[T any](k *Kernel, name string, tool registry.TypedTool[T]) error {
	return registry.Register(k.Tools, name, tool)
}

// ResumeWorkflow resumes a suspended workflow run from the Go side.
func (k *Kernel) ResumeWorkflow(ctx context.Context, runId, stepId, resumeDataJSON string) (string, error) {
	stepArg := "undefined"
	if stepId != "" {
		stepArg = fmt.Sprintf("%q", stepId)
	}
	code := fmt.Sprintf(`(async () => {
		var result = await globalThis.__kit.resumeWorkflow(%q, %s, %s);
		globalThis.__module_result = JSON.stringify(result);
	})()`, runId, stepArg, resumeDataJSON)

	val, err := k.bridge.EvalAsync("__resume_workflow.js", code)
	if err != nil {
		return "", fmt.Errorf("resume workflow %s: %w", runId, err)
	}
	if val != nil {
		val.Free()
	}
	result, err := k.bridge.Eval("__get_resume_result.js", `typeof globalThis.__module_result !== 'undefined' ? String(globalThis.__module_result) : ""`)
	if err != nil {
		return "", err
	}
	defer result.Free()
	return result.String(), nil
}
