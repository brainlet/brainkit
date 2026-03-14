package brainkit

import (
	"context"
	"fmt"
	"sync"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/google/uuid"
)

// SandboxConfig configures a sandbox.
type SandboxConfig struct {
	ID        string
	Namespace string // e.g., "user", "agent.team-1"
	CallerID  string // identity for bus messages from this sandbox
	Providers map[string]ProviderConfig
	EnvVars   map[string]string
}

// Sandbox is an isolated execution environment with its own QuickJS runtime.
type Sandbox struct {
	id        string
	namespace string
	callerID  string
	kit       *Kit
	agents    *agentembed.Sandbox
	mu        sync.Mutex
	closed    bool
}

// CreateSandbox creates a new isolated sandbox.
func (k *Kit) CreateSandbox(cfg SandboxConfig) (*Sandbox, error) {
	if cfg.ID == "" {
		cfg.ID = uuid.NewString()
	}
	if cfg.Namespace == "" {
		cfg.Namespace = "user"
	}
	if cfg.CallerID == "" {
		cfg.CallerID = cfg.Namespace
	}

	// Merge kit-level providers with sandbox-level
	providers := make(map[string]agentembed.ProviderConfig)
	for name, pc := range k.config.Providers {
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}
	for name, pc := range cfg.Providers {
		providers[name] = agentembed.ProviderConfig{APIKey: pc.APIKey, BaseURL: pc.BaseURL}
	}

	envVars := make(map[string]string)
	for key, val := range k.config.EnvVars {
		envVars[key] = val
	}
	for key, val := range cfg.EnvVars {
		envVars[key] = val
	}

	agentSandbox, err := agentembed.NewSandbox(agentembed.SandboxConfig{
		ID:        cfg.ID,
		Providers: providers,
		EnvVars:   envVars,
	})
	if err != nil {
		return nil, fmt.Errorf("brainkit: create sandbox: %w", err)
	}

	s := &Sandbox{
		id:        cfg.ID,
		namespace: cfg.Namespace,
		callerID:  cfg.CallerID,
		kit:       k,
		agents:    agentSandbox,
	}

	// Register Go bridge functions for PLATFORM operations
	s.registerBridges()

	// Load brainlet-runtime.js (the "brainlet" import surface)
	if err := s.loadRuntime(); err != nil {
		agentSandbox.Close()
		return nil, err
	}

	k.mu.Lock()
	k.sandboxes[cfg.ID] = s
	k.mu.Unlock()

	return s, nil
}

// ID returns the sandbox ID.
func (s *Sandbox) ID() string { return s.id }

// Namespace returns the sandbox namespace.
func (s *Sandbox) Namespace() string { return s.namespace }

// CallerID returns the sandbox's identity for bus messages.
func (s *Sandbox) CallerID() string { return s.callerID }

// AgentSandbox returns the underlying agent-embed sandbox for direct agent operations.
func (s *Sandbox) AgentSandbox() *agentembed.Sandbox { return s.agents }

// Eval runs raw JS code in this sandbox with async support.
func (s *Sandbox) Eval(ctx context.Context, filename, code string) (string, error) {
	return s.agents.Eval(ctx, filename, code)
}

// EvalTS runs .ts-style code with the brainlet import surface available.
// The code has access to: agent, createTool, z, ai, wasm, tools, tool, bus, sandbox.
// The code should return a value (the last expression is the result).
func (s *Sandbox) EvalTS(ctx context.Context, filename, code string) (string, error) {
	wrapped := fmt.Sprintf(`(async () => {
		const { agent, createTool, z, ai, wasm, tools, tool, bus, sandbox } = globalThis.__brainlet;
		%s
	})()`, code)
	return s.agents.Eval(ctx, filename, wrapped)
}

// Close destroys the sandbox and removes it from the kit.
func (s *Sandbox) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	s.mu.Unlock()

	if s.agents != nil {
		s.agents.Close()
	}

	s.kit.mu.Lock()
	delete(s.kit.sandboxes, s.id)
	s.kit.mu.Unlock()
}
