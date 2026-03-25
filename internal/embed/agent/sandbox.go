package agentembed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/internal/jsbridge"
)

// SandboxConfig configures an isolated sandbox.
type SandboxConfig struct {
	// ID — unique identifier. Auto-generated if empty.
	ID string

	// HTTPClient used for all fetch calls within this sandbox.
	HTTPClient *http.Client

	// Providers maps provider names to configs.
	// e.g., {"openai": {APIKey: "sk-..."}}
	Providers map[string]ProviderConfig

	// EnvVars injected into process.env within this sandbox.
	// Fallback for provider resolution when Providers map doesn't have the key.
	EnvVars map[string]string

	// MaxStackSize for the QuickJS runtime in bytes. 0 = use jsbridge default.
	MaxStackSize int
}

// Sandbox is an isolated execution environment with its own QuickJS runtime.
// Multiple agents can live in the same sandbox and communicate directly via JS.
// Cross-sandbox communication goes through Go.
// Safe for concurrent use — Eval calls are serialized by the bridge mutex,
// and async I/O runs in goroutines.
type Sandbox struct {
	id        string
	bridge    *jsbridge.Bridge
	providers map[string]ProviderConfig
	envVars   map[string]string

	mu     sync.Mutex
	agents map[string]*Agent
	closed bool
}

// NewSandbox creates a sandbox with all polyfills and the Mastra bundle loaded.
func NewSandbox(cfg SandboxConfig) (*Sandbox, error) {
	fetchOpts := []jsbridge.FetchOption{}
	if cfg.HTTPClient != nil {
		fetchOpts = append(fetchOpts, jsbridge.FetchClient(cfg.HTTPClient))
	}

	bridgeCfg := jsbridge.Config{}
	if cfg.MaxStackSize > 0 {
		bridgeCfg.MaxStackSize = cfg.MaxStackSize
	}

	b, err := jsbridge.New(bridgeCfg,
		jsbridge.Console(),
		jsbridge.Process(),
		jsbridge.Encoding(),
		jsbridge.Streams(),
		jsbridge.Crypto(),
		jsbridge.URL(),
		jsbridge.Timers(),
		jsbridge.Abort(),
		jsbridge.Events(),
		jsbridge.StructuredClone(),
		jsbridge.Net(),
		jsbridge.WebAssembly(),
		jsbridge.FS(),
		jsbridge.Exec(),
		jsbridge.Fetch(fetchOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("agent-embed: create bridge: %w", err)
	}

	// Load Mastra bundle (runtime globals + bundle JS)
	if err := LoadBundle(b); err != nil {
		b.Close()
		return nil, err
	}

	id := cfg.ID
	if id == "" {
		// Generate a simple ID
		val, err := b.Eval("id.js", `crypto.randomUUID()`)
		if err != nil {
			b.Close()
			return nil, fmt.Errorf("agent-embed: generate sandbox ID: %w", err)
		}
		id = val.String()
		val.Free()
	}

	s := &Sandbox{
		id:        id,
		bridge:    b,
		providers: cfg.Providers,
		envVars:   cfg.EnvVars,
		agents:    make(map[string]*Agent),
	}

	// Inject provider configs and env vars into JS runtime
	if err := s.injectConfig(); err != nil {
		b.Close()
		return nil, err
	}

	// Initialize agent registry
	_, err = b.Eval("registry.js", `globalThis.__agents = {}`)
	if err != nil {
		b.Close()
		return nil, fmt.Errorf("agent-embed: init registry: %w", err)
	}

	return s, nil
}

// injectConfig sets up provider configs and env vars in the JS runtime.
func (s *Sandbox) injectConfig() error {
	// Inject providers map for model resolution
	if len(s.providers) > 0 {
		providersJSON, _ := json.Marshal(s.providers)
		_, err := s.bridge.Eval("providers.js", fmt.Sprintf(
			`globalThis.__kit_providers = %s`, string(providersJSON),
		))
		if err != nil {
			return fmt.Errorf("agent-embed: inject providers: %w", err)
		}
	}

	// Inject env vars into process.env
	for k, v := range s.envVars {
		_, err := s.bridge.Eval("env.js", fmt.Sprintf(
			`globalThis.process.env[%q] = %q`, k, v,
		))
		if err != nil {
			return fmt.Errorf("agent-embed: inject env var %s: %w", k, err)
		}
	}

	return nil
}

// ID returns the sandbox's unique identifier.
func (s *Sandbox) ID() string { return s.id }

// Bridge returns the underlying jsbridge.Bridge.
func (s *Sandbox) Bridge() *jsbridge.Bridge { return s.bridge }

// Close destroys the sandbox, closing all agents and freeing the JS runtime.
func (s *Sandbox) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	// Copy agents to close outside the lock
	agents := make([]*Agent, 0, len(s.agents))
	for _, a := range s.agents {
		agents = append(agents, a)
	}
	s.agents = nil
	s.mu.Unlock()

	for _, a := range agents {
		a.close()
	}

	if s.bridge != nil {
		s.bridge.Close()
	}
}

// registerAgent adds an agent to the sandbox's tracking map.
func (s *Sandbox) registerAgent(a *Agent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agents != nil {
		s.agents[a.id] = a
	}
}

// unregisterAgent removes an agent from the sandbox's tracking map.
func (s *Sandbox) unregisterAgent(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agents != nil {
		delete(s.agents, id)
	}
}

// registerToolCallbacks registers Go tool execute functions for an agent.
func (s *Sandbox) registerToolCallbacks(agentID string, tools map[string]Tool) {
	if len(tools) == 0 {
		return
	}

	ctx := s.bridge.Context()
	fnName := "__go_tool_" + agentID

	ctx.Globals().Set(fnName, ctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) < 2 {
			return qctx.ThrowError(fmt.Errorf("tool callback: expected 2 args (toolName, argsJSON)"))
		}
		toolName := args[0].String()
		argsJSON := args[1].String()

		tool, ok := tools[toolName]
		if !ok || tool.Execute == nil {
			return qctx.ThrowError(fmt.Errorf("tool %q not found or has no Execute function", toolName))
		}

		result, err := tool.Execute(ToolContext{}, json.RawMessage(argsJSON))
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("tool %q execution error: %w", toolName, err))
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			return qctx.ThrowError(fmt.Errorf("tool %q result marshal error: %w", toolName, err))
		}
		return qctx.NewString(string(resultJSON))
	}))
}

// Eval runs JS in the sandbox with async support.
func (s *Sandbox) Eval(ctx context.Context, filename, code string) (string, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return "", fmt.Errorf("agent-embed: sandbox is closed")
	}
	s.mu.Unlock()

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	val, err := s.bridge.EvalAsync(filename, code)
	if err != nil {
		return "", err
	}
	defer val.Free()
	return val.String(), nil
}
