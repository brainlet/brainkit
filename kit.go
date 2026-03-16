package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/jsbridge"
	mcppkg "github.com/brainlet/brainkit/mcp"
	"github.com/brainlet/brainkit/registry"
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
	bridge    *jsbridge.Bridge
	agents    *agentembed.Sandbox
	wasm      *WASMService

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

	sharedBus := cfg.SharedBus
	if sharedBus == nil {
		sharedBus = bus.New()
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

	// Load brainlet-runtime.js + register "brainlet" ES module
	if err := k.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Create WASM service (compiler is lazy — only created on first wasm.compile)
	k.wasm = newWASMService(k)

	// Register bus handlers
	k.registerHandlers()

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
				fullName := "mcp." + toolCopy.ServerName + "." + toolCopy.Name
				k.Tools.Register(registry.RegisteredTool{
					Name:        fullName,
					ShortName:   toolCopy.Name,
					Namespace:   "mcp." + toolCopy.ServerName,
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

	if k.MCP != nil {
		k.MCP.Close()
	}
	if k.wasm != nil {
		k.wasm.close()
	}
	if k.agents != nil {
		k.agents.Close()
	}
	// Only close the bus if we own it (not shared)
	if k.config.SharedBus == nil {
		k.Bus.Close()
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

// EvalTS runs .ts-style code with brainlet imports destructured.
func (k *Kit) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		const { agent, createTool, createWorkflow, createStep, createMemory, z, ai, wasm, tools, tool, bus, mcp, sandbox, output, Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore, LibSQLVector, PgVector, MongoDBVector, generateText, streamText, generateObject, streamObject, createWorkflowRun, resumeWorkflow, createScorer, runEvals, scorers, RequestContext, MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool, rerank, rerankWithScorer } = globalThis.__brainlet;
		%s
	})()`, code)

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

// EvalModule runs code as an ES module with import { ... } from "brainlet".
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
//	kit.RegisterTool("platform.math.add", registry.TypedTool[AddInput]{
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
		var result = await globalThis.__brainlet.resumeWorkflow(%q, %s, %s);
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
	k.Bus.Handle("wasm.*", k.wasm.handleBusMessage)

	k.Bus.Handle("tools.*", func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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
	})

	k.Bus.Handle("mcp.*", func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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
	})
}

func (k *Kit) handleToolsCall(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("tools.call: invalid request: %w", err)
	}

	tool, err := k.Tools.Resolve(req.Name, msg.CallerID)
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

	ns := msg.CallerID
	shortName := req.Name
	fullName := ns + "." + req.Name

	k.Tools.Register(registry.RegisteredTool{
		Name:        fullName,
		ShortName:   shortName,
		Namespace:   ns,
		Description: req.Description,
		InputSchema: req.InputSchema,
	})

	result, _ := json.Marshal(map[string]string{"registered": fullName})
	return &bus.Message{Payload: result}, nil
}

func (k *Kit) handleToolsList(_ context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Namespace string `json:"namespace"`
	}
	json.Unmarshal(msg.Payload, &req)

	toolList := k.Tools.List(req.Namespace)
	var infos []map[string]any
	for _, t := range toolList {
		infos = append(infos, map[string]any{
			"name":        t.Name,
			"shortName":   t.ShortName,
			"namespace":   t.Namespace,
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

	tool, err := k.Tools.Resolve(req.Name, msg.CallerID)
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
