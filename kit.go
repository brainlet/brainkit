package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/jsbridge"
	"github.com/brainlet/brainkit/registry"
)

// Kit is the brainkit execution engine.
// One Kit = one QuickJS runtime = one isolation boundary.
// All agents, AI calls, workflows share the same JS context.
type Kit struct {
	Bus       *bus.Bus
	Tools     *registry.ToolRegistry
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

	// Load brainlet-runtime.js + register "brainlet" ES module
	if err := k.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	// Create WASM service (compiler is lazy — only created on first wasm.compile)
	k.wasm = newWASMService(k)

	// Register bus handlers
	k.registerHandlers()

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
		const { agent, createTool, createWorkflow, createStep, createMemory, z, ai, wasm, tools, tool, bus, sandbox, output, Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore } = globalThis.__brainlet;
		%s
	})()`, code)
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

func (k *Kit) registerHandlers() {
	k.Bus.Handle("wasm.*", k.wasm.handleBusMessage)

	k.Bus.Handle("tools.*", func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		switch msg.Topic {
		case "tools.resolve":
			return k.handleToolsResolve(ctx, msg)
		case "tools.call":
			return k.handleToolsCall(ctx, msg)
		default:
			return nil, fmt.Errorf("tools: unknown topic %q", msg.Topic)
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
