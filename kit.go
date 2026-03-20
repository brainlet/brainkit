package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/jsbridge"
	"github.com/brainlet/brainkit/libsql"
	mcppkg "github.com/brainlet/brainkit/mcp"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/brainlet/brainkit/registry"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Kit is the brainkit execution engine.
// One Kit = one QuickJS runtime = one isolation boundary.
// All agents, AI calls, workflows share the same JS context.
type Kit struct {
	Bus       *bus.Bus
	Tools     *registry.ToolRegistry
	MCP       *mcppkg.MCPManager
	config    Config
	namespace string
	callerID  string
	bridge   *jsbridge.Bridge
	agents   *agentembed.Sandbox
	wasm     *WASMService
	plugins  *pluginManager
	storages  map[string]*libsql.Server // named embedded SQLite bridges
	agentReg  *agentRegistry
	network   *hostServer     // gRPC server for incoming peer connections
	transport *GRPCTransport  // transport with peer routing (nil if no network)
	discovery Discovery       // optional peer discovery

	mu     sync.Mutex
	closed bool
}

// New creates a Kit with one QuickJS runtime.
func New(cfg Config) (*Kit, error) {
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	var grpcTransport *GRPCTransport
	sharedBus := cfg.SharedBus
	if sharedBus == nil {
		switch cfg.Transport {
		case "nats":
			if cfg.NATS.URL == "" {
				return nil, fmt.Errorf("brainkit: NATS transport requires NATS.URL")
			}
			natsName := cfg.NATS.Name
			if natsName == "" {
				natsName = cfg.Name
			}
			nt, err := NewNATSTransport(cfg.NATS.URL, nats.Name(natsName))
			if err != nil {
				return nil, fmt.Errorf("brainkit: %w", err)
			}
			sharedBus = bus.NewBus(nt)
		default:
			if cfg.Network.Listen != "" || len(cfg.Network.Peers) > 0 || cfg.Network.Discovery.Type != "" {
				grpcTransport = NewGRPCTransport()
				sharedBus = bus.NewBus(grpcTransport)
			} else {
				sharedBus = bus.NewBus(bus.NewInProcessTransport())
			}
		}
	}
	if cfg.Name != "" {
		if err := sharedBus.RegisterName(cfg.Name); err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
	}
	sharedTools := cfg.SharedTools
	if sharedTools == nil {
		sharedTools = registry.New()
	}

	k := &Kit{
		Bus:       sharedBus,
		Tools:     sharedTools,
		config:    cfg,
		namespace: cfg.Namespace,
		callerID:  cfg.CallerID,
		storages:  make(map[string]*libsql.Server),
		agentReg:  newAgentRegistry(),
		transport: grpcTransport,
	}

	// Start network listener if configured
	if cfg.Network.Listen != "" {
		k.network = newHostServer(k)
		if err := k.network.Start(cfg.Network.Listen); err != nil {
			return nil, fmt.Errorf("brainkit: %w", err)
		}
	}

	// Connect to known peers
	for name, addr := range cfg.Network.Peers {
		if err := k.connectPeer(name, addr); err != nil {
			log.Printf("[kit] failed to connect to peer %q at %s: %v", name, addr, err)
		}
	}

	// Create discovery if configured
	switch cfg.Network.Discovery.Type {
	case "multicast":
		disc, err := NewMulticastDiscovery(cfg.Network.Discovery.ServiceName)
		if err != nil {
			log.Printf("[kit] multicast discovery failed: %v", err)
		} else {
			k.discovery = disc
		}
	case "static":
		if len(cfg.Network.Peers) > 0 {
			k.discovery = NewStaticDiscovery(cfg.Network.Peers)
		}
	}

	if k.discovery != nil && k.transport != nil {
		k.transport.discovery = k.discovery
		k.transport.ConnectFunc = k.connectPeer
	}

	// Register self with discovery
	if k.discovery != nil && cfg.Network.Listen != "" && cfg.Name != "" {
		k.discovery.Register(Peer{
			Name:    cfg.Name,
			Address: k.network.Addr(),
		})
	}

	// Build provider config for agent-embed
	providers := make(map[string]agentembed.ProviderConfig)
	for name, pc := range cfg.Providers {
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	// Create THE single agent-embed sandbox (one QuickJS runtime)
	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		Providers:    providers,
		EnvVars:      cfg.EnvVars,
		MaxStackSize: cfg.MaxStackSize,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create runtime: %w", err)
	}
	k.agents = agentSandbox
	k.bridge = agentSandbox.Bridge()

	// Register Go bridges for PLATFORM operations
	k.registerBridges()

	// Start embedded LibSQL bridges for configured storages
	for name, scfg := range cfg.Storages {
		if err := k.addStorageInternal(name, scfg); err != nil {
			// Clean up already-started storages
			for _, srv := range k.storages {
				srv.Close()
			}
			agentSandbox.Close()
			return nil, fmt.Errorf("brainkit: start storage %q: %w", name, err)
		}
	}

	// Inject observability config for brainlet-runtime.js to read
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
		`globalThis.__brainkit_obs_config = { enabled: %v, strategy: %q, serviceName: %q }`,
		obsEnabled, obsStrategy, obsServiceName,
	))

	// Load kit_runtime.js + register "kit" ES module
	if err := k.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Create WASM service (compiler is lazy — only created on first wasm.compile)
	k.wasm = newWASMService(k)

	// Register bus handlers
	k.registerHandlers()

	// Load persisted modules + shards from store (if configured)
	if cfg.Store != nil {
		if err := k.wasm.loadFromStore(cfg.Store); err != nil {
			log.Printf("[brainkit] warning: failed to load persisted data: %v", err)
		}
	}

	// Connect to MCP servers and auto-register their tools
	if len(cfg.MCPServers) > 0 {
		k.MCP = mcppkg.New()
		for name, serverCfg := range cfg.MCPServers {
			if err := k.MCP.Connect(context.Background(), name, serverCfg); err != nil {
				// Log but don't fail — MCP servers may be unavailable
				continue
			}
			// Register each MCP tool in the ToolRegistry
			for _, tool := range k.MCP.ListToolsForServer(name) {
				toolCopy := tool // capture loop variable
				fullName := registry.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
				k.Tools.Register(registry.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Owner:       "mcp",
					Package:     toolCopy.ServerName,
					Version:     "1.0.0",
					Description: toolCopy.Description,
					InputSchema: toolCopy.InputSchema,
					Executor: &registry.GoFuncExecutor{
						Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
							return k.MCP.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
						},
					},
				})
			}
		}
	}

	// Start plugin manager
	if len(cfg.Plugins) > 0 {
		k.plugins = newPluginManager(k)
		k.plugins.startAll(cfg.Plugins)
	}

	return k, nil
}

// Close shuts down the runtime and the bus.
func (k *Kit) Close() {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return
	}
	k.closed = true
	k.mu.Unlock()

	// Shutdown order: network/transport FIRST (kills gRPC streams so goroutines can exit),
	// then plugins, then bridge (waits for goroutines — now they can exit cleanly).
	if k.network != nil {
		k.network.Stop()
	}
	if k.discovery != nil {
		k.discovery.Close()
	}
	if k.agentReg != nil && k.agents != nil {
		k.agentReg.unregisterAllForKit(k.agents.ID())
	}
	if k.plugins != nil {
		k.plugins.stopAll()
	}
	if k.MCP != nil {
		k.MCP.Close()
	}
	if k.wasm != nil {
		k.wasm.close()
	}
	if k.config.Store != nil {
		k.config.Store.Close()
	}
	if k.agents != nil {
		k.agents.Close()
	}
	for _, srv := range k.storages {
		srv.Close()
	}
	if k.config.Name != "" {
		k.Bus.UnregisterName(k.config.Name)
	}
	if k.config.SharedBus == nil {
		k.Bus.Close()
	}
}

// connectPeer establishes a gRPC connection to a remote Kit.
func (k *Kit) connectPeer(name, addr string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("connect to peer %s at %s: %w", name, addr, err)
	}

	client := pluginv1.NewBrainkitHostServiceClient(conn)

	resp, err := client.Handshake(context.Background(), &pluginv1.HandshakeRequest{
		Name:    k.config.Name,
		Version: "v1",
		Type:    "kit",
	})
	if err != nil {
		conn.Close()
		return fmt.Errorf("handshake with %s: %w", name, err)
	}
	if !resp.Accepted {
		conn.Close()
		return fmt.Errorf("handshake with %s rejected: %s", name, resp.RejectionReason)
	}

	stream, err := client.MessageStream(context.Background())
	if err != nil {
		conn.Close()
		return fmt.Errorf("open stream to %s: %w", name, err)
	}

	// Send identity message
	stream.Send(&pluginv1.PluginMessage{
		Id:       uuid.NewString(),
		Type:     "identity",
		CallerId: k.config.Name,
	})

	pc := &peerConn{
		name:   name,
		addr:   addr,
		conn:   conn,
		stream: stream,
		done:   make(chan struct{}),
	}

	if k.transport != nil {
		k.transport.addPeer(pc)
	}

	go k.readPeerStream(name, pc)

	log.Printf("[kit] connected to peer %q at %s", name, addr)
	return nil
}

// readPeerStream reads messages from a remote Kit.
func (k *Kit) readPeerStream(name string, pc *peerConn) {
	defer func() {
		if k.transport != nil {
			k.transport.removePeer(name)
		}
		pc.closeDone()
		log.Printf("[kit] peer %q stream closed", name)
	}()

	for {
		msg, err := pc.stream.Recv()
		if err != nil {
			log.Printf("[kit] peer %s recv: %v", name, err)
			return
		}

		switch msg.Type {
		case "ask.reply":
			if msg.ReplyTo != "" {
				k.Bus.Send(bus.Message{
					Topic:    msg.ReplyTo,
					CallerID: "peer/" + name,
					Payload:  msg.Payload,
					Metadata: msg.Metadata,
				})
			}
		case "event":
			k.Bus.Send(bus.Message{
				Topic:    msg.Topic,
				CallerID: "peer/" + name,
				TraceID:  msg.TraceId,
				Payload:  msg.Payload,
				Metadata: msg.Metadata,
			})
		default:
			log.Printf("[kit] peer %s: unknown message type %q", name, msg.Type)
		}
	}
}

// Namespace returns the Kit's namespace.
func (k *Kit) Namespace() string { return k.namespace }

// CallerID returns the Kit's identity for bus messages.
func (k *Kit) CallerID() string { return k.callerID }

// CreateAgent creates a persistent agent in the Kit's runtime.
func (k *Kit) CreateAgent(cfg agentembed.AgentConfig) (*agentembed.Agent, error) {
	return k.agents.CreateAgent(cfg)
}

// AddStorage starts a new named embedded SQLite storage and makes it available to JS.
// JS code can then use `new LibSQLStore({ id: "x", storage: "name" })`.
func (k *Kit) AddStorage(name string, cfg StorageConfig) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, exists := k.storages[name]; exists {
		return fmt.Errorf("brainkit: storage %q already exists", name)
	}
	return k.addStorageInternal(name, cfg)
}

// RemoveStorage stops and removes a named storage.
func (k *Kit) RemoveStorage(name string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	srv, ok := k.storages[name]
	if !ok {
		return fmt.Errorf("brainkit: storage %q not found", name)
	}
	srv.Close()
	delete(k.storages, name)
	// Update JS-side storage map
	k.bridge.Eval("__storage_remove.js", fmt.Sprintf(
		`delete globalThis.__brainkit_storages[%q]`, name,
	))
	return nil
}

// StorageURL returns the HTTP URL for a named storage bridge.
// Returns "" if the storage doesn't exist.
func (k *Kit) StorageURL(name string) string {
	k.mu.Lock()
	defer k.mu.Unlock()
	if srv, ok := k.storages[name]; ok {
		return srv.URL()
	}
	return ""
}

// ListResources returns all tracked resources, optionally filtered by type.
// Types: "agent", "tool", "workflow", "wasm", "memory", "harness"
func (k *Kit) ListResources(resourceType ...string) ([]ResourceInfo, error) {
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
func (k *Kit) ResourcesFrom(filename string) ([]ResourceInfo, error) {
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
// Returns the number of resources removed.
func (k *Kit) TeardownFile(filename string) (int, error) {
	code := fmt.Sprintf(`
		var resources = globalThis.__kit_registry.listBySource(%q);
		var count = 0;
		// Teardown in reverse order (LIFO — last created, first destroyed)
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
func (k *Kit) RemoveResource(resourceType, id string) error {
	code := fmt.Sprintf(`
		var entry = globalThis.__kit_registry.unregister(%q, %q);
		return JSON.stringify(entry !== null);
	`, resourceType, id)
	_, err := k.EvalTS(context.Background(), "__remove_resource.ts", code)
	return err
}

// ListWASMModules returns metadata for all compiled WASM modules.
func (k *Kit) ListWASMModules() ([]WASMModuleInfo, error) {
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
func (k *Kit) GetWASMModule(name string) (*WASMModuleInfo, error) {
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

// RemoveWASMModule unloads a compiled module by name.
// Fails if a shard is deployed from this module (undeploy first).
func (k *Kit) RemoveWASMModule(name string) error {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.remove",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return err
	}
	// Check for error in response payload (bus wraps handler errors as {"error":"..."})
	var result struct {
		Error   string `json:"error"`
		Removed bool   `json:"removed"`
	}
	json.Unmarshal(resp.Payload, &result)
	if result.Error != "" {
		return fmt.Errorf("%s", result.Error)
	}
	if !result.Removed {
		return fmt.Errorf("wasm module %q not found", name)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shard API — deploy/undeploy event-reactive WASM modules
// ---------------------------------------------------------------------------

// DeployWASM activates a compiled shard — calls init(), registers event handlers.
func (k *Kit) DeployWASM(name string) (*ShardDescriptor, error) {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.deploy",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return nil, err
	}
	var desc ShardDescriptor
	json.Unmarshal(resp.Payload, &desc)
	return &desc, nil
}

// UndeployWASM removes all event subscriptions for a deployed shard.
func (k *Kit) UndeployWASM(name string) error {
	_, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.undeploy",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	return err
}

// DescribeWASM returns the shard's registrations (mode, handlers, state key).
func (k *Kit) DescribeWASM(name string) (*ShardDescriptor, error) {
	resp, err := bus.AskSync(k.Bus, context.Background(), bus.Message{
		Topic:    "wasm.describe",
		CallerID: k.callerID,
		Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q}`, name)),
	})
	if err != nil {
		return nil, err
	}
	var desc ShardDescriptor
	json.Unmarshal(resp.Payload, &desc)
	if desc.Module == "" {
		return nil, nil
	}
	return &desc, nil
}

// ListDeployedWASM returns all active shard descriptors.
func (k *Kit) ListDeployedWASM() []ShardDescriptor {
	return k.wasm.listDeployedShards()
}

// InjectWASMEvent manually triggers a shard handler (for testing and SDK use).
func (k *Kit) InjectWASMEvent(shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
	return k.wasm.invokeShardHandler(context.Background(), shardName, topic, payload)
}

func (k *Kit) addStorageInternal(name string, cfg StorageConfig) error {
	srv, err := libsql.NewServer(cfg.Path)
	if err != nil {
		return err
	}
	k.storages[name] = srv
	// Register in JS-side storage map
	k.bridge.Eval("__storage_add.js", fmt.Sprintf(
		`if (!globalThis.__brainkit_storages) globalThis.__brainkit_storages = {};
		 globalThis.__brainkit_storages[%q] = %q;`, name, srv.URL(),
	))
	return nil
}

// EvalTS runs .ts-style code with brainlet imports destructured.
func (k *Kit) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		globalThis.__kit_current_source = %q;
		const { agent, createTool, createSubagent, createWorkflow, createStep, createMemory, z, ai, wasm, tools, tool, bus, agents, mcp, sandbox, output, Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore, LibSQLVector, PgVector, MongoDBVector, generateText, streamText, generateObject, streamObject, createWorkflowRun, resumeWorkflow, createScorer, runEvals, scorers, processors, RequestContext, MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool, rerank, rerankWithScorer, Workspace, LocalFilesystem, LocalSandbox, createHarness } = globalThis.__kit;
		%s
	})()`, filename, code)

	// If the Bridge is currently in an eval/await loop (e.g., we're being called
	// from a Go tool callback during agent.generate/stream), use EvalOnJSThread.
	// This handles two cases:
	//   1. Direct tool callback (same goroutine) → calls ctx.Eval directly
	//   2. Bus handler (different goroutine) → schedules via ctx.Schedule + channel
	// Both avoid the mutex deadlock.
	if k.bridge.IsEvalBusy() {
		return k.bridge.EvalOnJSThread(filename, wrapped)
	}

	return k.agents.Eval(ctx, filename, wrapped)
}

// EvalModule runs code as an ES module with import { ... } from "kit".
func (k *Kit) EvalModule(ctx context.Context, filename, code string) (string, error) {
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
// The JSON Schema is generated automatically from T's struct tags.
//
// Example:
//
//	type AddInput struct {
//	    A float64 `json:"a" desc:"First number"`
//	    B float64 `json:"b" desc:"Second number"`
//	}
//	kit.RegisterTool("brainlet/math@1.0.0/add", registry.TypedTool[AddInput]{
//	    Description: "Adds two numbers",
//	    Execute: func(ctx context.Context, input AddInput) (any, error) {
//	        return map[string]any{"result": input.A + input.B}, nil
//	    },
//	})
func RegisterTool[T any](k *Kit, name string, tool registry.TypedTool[T]) error {
	return registry.Register(k.Tools, name, tool)
}

// ResumeWorkflow resumes a suspended workflow run from the Go side.
// runId: the workflow run's ID
// stepId: which step to resume (empty string for auto-detect)
// resumeDataJSON: JSON-encoded resume data to pass to the step
func (k *Kit) ResumeWorkflow(ctx context.Context, runId, stepId, resumeDataJSON string) (string, error) {
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

func (k *Kit) registerHandlers() {
	// If this Kit is part of a pool, use AsWorker for competing consumers
	var subOpts []bus.SubscribeOption
	if k.config.WorkerGroup != "" {
		subOpts = append(subOpts, bus.AsWorker(k.config.WorkerGroup))
	}

	// wrapHandler adapts the old Handler signature to the new On/ReplyFunc pattern.
	wrapHandler := func(h func(ctx context.Context, msg bus.Message) (*bus.Message, error)) func(bus.Message, bus.ReplyFunc) {
		return func(msg bus.Message, reply bus.ReplyFunc) {
			resp, err := h(context.Background(), msg)
			if err != nil {
				errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
				reply(errPayload)
				return
			}
			if resp != nil {
				reply(resp.Payload)
			}
		}
	}

	k.Bus.On("wasm.*", wrapHandler(k.wasm.handleBusMessage), subOpts...)

	k.Bus.On("tools.*", wrapHandler(func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		switch msg.Topic {
		case "tools.resolve":
			return k.handleToolsResolve(ctx, msg)
		case "tools.call":
			return k.handleToolsCall(ctx, msg)
		case "tools.register":
			return k.handleToolsRegister(ctx, msg)
		case "tools.list":
			return k.handleToolsList(ctx, msg)
		default:
			return nil, fmt.Errorf("tools: unknown topic %q", msg.Topic)
		}
	}), subOpts...)

	k.Bus.On("mcp.*", wrapHandler(func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		if k.MCP == nil {
			return nil, fmt.Errorf("mcp: no MCP servers configured")
		}
		switch msg.Topic {
		case "mcp.listTools":
			tools := k.MCP.ListTools()
			data, _ := json.Marshal(tools)
			return &bus.Message{Payload: data}, nil
		case "mcp.callTool":
			var params struct {
				Server string          `json:"server"`
				Tool   string          `json:"tool"`
				Args   json.RawMessage `json:"args"`
			}
			if err := json.Unmarshal(msg.Payload, &params); err != nil {
				return nil, fmt.Errorf("mcp.callTool: %w", err)
			}
			result, err := k.MCP.CallTool(ctx, params.Server, params.Tool, params.Args)
			if err != nil {
				return nil, err
			}
			return &bus.Message{Payload: result}, nil
		default:
			return nil, fmt.Errorf("mcp: unknown topic %q", msg.Topic)
		}
	}), subOpts...)

	k.Bus.On("agents.*", wrapHandler(k.handleAgents), subOpts...)
	k.Bus.On("fs.*", wrapHandler(k.handleFs), subOpts...)
	k.Bus.On("ai.*", wrapHandler(k.handleAI), subOpts...)
	k.Bus.On("memory.*", wrapHandler(k.handleMemory), subOpts...)
	k.Bus.On("workflows.*", wrapHandler(k.handleWorkflows), subOpts...)
	k.Bus.On("vectors.*", wrapHandler(k.handleVectors), subOpts...)

	// Plugin state handlers — plugins call GetState/SetState via typed messages
	pluginState := make(map[string]string)
	var pluginStateMu sync.Mutex
	k.Bus.On("plugin.state.*", wrapHandler(func(_ context.Context, msg bus.Message) (*bus.Message, error) {
		switch msg.Topic {
		case "plugin.state.get":
			var req struct{ Key string `json:"key"` }
			json.Unmarshal(msg.Payload, &req)
			pluginStateMu.Lock()
			val := pluginState[req.Key]
			pluginStateMu.Unlock()
			result, _ := json.Marshal(map[string]string{"value": val})
			return &bus.Message{Payload: result}, nil
		case "plugin.state.set":
			var req struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			json.Unmarshal(msg.Payload, &req)
			pluginStateMu.Lock()
			pluginState[req.Key] = req.Value
			pluginStateMu.Unlock()
			result, _ := json.Marshal(map[string]bool{"ok": true})
			return &bus.Message{Payload: result}, nil
		default:
			return nil, fmt.Errorf("plugin.state: unknown topic %q", msg.Topic)
		}
	}), subOpts...)
}

func (k *Kit) handleToolsCall(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.call: invalid request: %w", err)
	}

	tool, err := k.Tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	result, err := tool.Executor.Call(ctx, msg.CallerID, req.Input)
	if err != nil {
		return nil, err
	}

	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsRegister(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		InputSchema json.RawMessage `json:"inputSchema"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.register: invalid request: %w", err)
	}

	// Compose new-format name from caller identity.
	// CallerID is the owner context; req.Name is the short tool name.
	callerID := msg.CallerID
	var fullName string
	if registry.IsNewFormat(req.Name) {
		fullName = req.Name
	} else {
		fullName = registry.ComposeName(callerID, callerID, "0.0.0", req.Name)
	}

	k.Tools.Register(registry.RegisteredTool{
		Name:        fullName,
		ShortName:   req.Name,
		Description: req.Description,
		InputSchema: req.InputSchema,
	})

	result, _ := json.Marshal(map[string]string{"registered": fullName})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsList(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Filter string `json:"filter"`
	}
	json.Unmarshal(msg.Payload, &req)

	toolList := k.Tools.List(req.Filter)
	var infos []map[string]any
	for _, t := range toolList {
		infos = append(infos, map[string]any{
			"name":        t.Name,
			"shortName":   t.ShortName,
			"description": t.Description,
		})
	}

	result, _ := json.Marshal(infos)
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsResolve(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.resolve: invalid request: %w", err)
	}

	tool, err := k.Tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	info := map[string]any{
		"name":        tool.Name,
		"shortName":   tool.ShortName,
		"description": tool.Description,
	}
	if tool.InputSchema != nil {
		info["inputSchema"] = string(tool.InputSchema)
	}

	result, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	return &bus.Message{Payload: result}, nil
}
