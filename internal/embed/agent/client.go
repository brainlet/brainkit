package agentembed

import (
	"context"
)

// ClientConfig configures an agent-embed client.
type ClientConfig struct {
	// Providers maps provider names to configs.
	Providers map[string]ProviderConfig

	// EnvVars injected into process.env.
	// Fallback for provider resolution when Providers map doesn't have the key.
	EnvVars map[string]string
}

// Client is the top-level entry point for agent-embed.
// It manages sandboxes and provides convenience methods for single-agent use.
type Client struct {
	providers map[string]ProviderConfig
	envVars   map[string]string
}

// NewClient creates a new agent-embed client.
func NewClient(cfg ClientConfig) *Client {
	return &Client{
		providers: cfg.Providers,
		envVars:   cfg.EnvVars,
	}
}

// CreateSandbox creates a new isolated sandbox with its own QuickJS runtime.
func (c *Client) CreateSandbox(cfg SandboxConfig) (*Sandbox, error) {
	// Merge client-level config with sandbox-level config
	if cfg.Providers == nil && c.providers != nil {
		cfg.Providers = c.providers
	}
	if cfg.EnvVars == nil && c.envVars != nil {
		cfg.EnvVars = c.envVars
	}
	return NewSandbox(cfg)
}

// CreateAgent is a convenience that creates a sandbox with one agent.
// Closing the agent also closes its sandbox.
func (c *Client) CreateAgent(cfg AgentConfig) (*Agent, error) {
	sandbox, err := c.CreateSandbox(SandboxConfig{})
	if err != nil {
		return nil, err
	}
	agent, err := sandbox.CreateAgent(cfg)
	if err != nil {
		sandbox.Close()
		return nil, err
	}
	return agent, nil
}

// QuickGenerateParams combines agent config + generate params for one-shot use.
type QuickGenerateParams struct {
	// Agent config
	Name         string
	Model        string
	Instructions string
	Tools        map[string]Tool

	// Generate params
	Prompt   string
	Messages []Message
	MaxSteps int
}

// Generate creates an ephemeral sandbox+agent, generates, and destroys both.
func (c *Client) Generate(ctx context.Context, params QuickGenerateParams) (*GenerateResult, error) {
	agent, err := c.CreateAgent(AgentConfig{
		Name:         params.Name,
		Model:        params.Model,
		Instructions: params.Instructions,
		Tools:        params.Tools,
		MaxSteps:     params.MaxSteps,
	})
	if err != nil {
		return nil, err
	}
	defer agent.Sandbox().Close()

	return agent.Generate(ctx, GenerateParams{
		Prompt:   params.Prompt,
		Messages: params.Messages,
		MaxSteps: params.MaxSteps,
	})
}

// QuickStreamParams combines agent config + stream params for one-shot use.
type QuickStreamParams struct {
	Name         string
	Model        string
	Instructions string
	Tools        map[string]Tool
	Prompt       string
	Messages     []Message
	MaxSteps     int
	OnToken      func(token string)
}

// Stream creates an ephemeral sandbox+agent, streams, and destroys both.
func (c *Client) Stream(ctx context.Context, params QuickStreamParams) (*StreamResult, error) {
	agent, err := c.CreateAgent(AgentConfig{
		Name:         params.Name,
		Model:        params.Model,
		Instructions: params.Instructions,
		Tools:        params.Tools,
		MaxSteps:     params.MaxSteps,
	})
	if err != nil {
		return nil, err
	}
	defer agent.Sandbox().Close()

	return agent.Stream(ctx, StreamParams{
		Prompt:   params.Prompt,
		Messages: params.Messages,
		MaxSteps: params.MaxSteps,
		OnToken:  params.OnToken,
	})
}

