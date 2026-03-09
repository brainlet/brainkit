// Ported from: packages/core/src/tool-loop-agent/index.ts
package toolloopagent

import (
	"crypto/rand"
	"fmt"
)

// ---------------------------------------------------------------------------
// Agent — structurally compatible with agent.Agent
// ---------------------------------------------------------------------------
//
// The real agent.Agent embeds *agentkit.MastraBase, has private fields (sync.Mutex,
// DynamicArgument for instructions/tools/etc.), and requires NewAgent(config)
// constructor. We keep a local struct for backward compatibility since the
// ToolLoopAgentToMastraAgent function constructs it via struct literal.
// Fields match the real agent.Agent exported fields.
type Agent struct {
	// ID uniquely identifies the agent.
	ID string `json:"id"`
	// AgentName is the display name for the agent (field name matches real agent.Agent).
	Name string `json:"name"`
	// Source indicates whether the agent was created from code or storage.
	Source string `json:"source,omitempty"`

	// Model is the language model used by the agent.
	Model any `json:"model"`
	// MaxRetries for model calls in case of failure. Default: 0.
	MaxRetries *int `json:"maxRetries,omitempty"`

	// Instructions that guide the agent's behavior.
	Instructions AgentInstructions `json:"instructions,omitempty"`
	// Tools that the agent can access.
	Tools map[string]any `json:"tools,omitempty"`
	// DefaultOptions is the default options for stream()/generate().
	DefaultOptions *AgentExecutionOptions `json:"defaultOptions,omitempty"`

	// InputProcessors adapts ToolLoopAgent into the processor pipeline.
	InputProcessors []*ToolLoopAgentProcessor `json:"-"`
}

// ---------------------------------------------------------------------------
// generateID
// ---------------------------------------------------------------------------

// generateID produces a short random hex string, mirroring AI SDK's generateId().
// ai-kit only ported the @ai-sdk/provider layer; utility functions remain local.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ---------------------------------------------------------------------------
// ToolLoopAgentToMastraAgentOptions
// ---------------------------------------------------------------------------

// ToolLoopAgentToMastraAgentOptions holds optional parameters for the conversion.
type ToolLoopAgentToMastraAgentOptions struct {
	// FallbackName is used when the ToolLoopAgent has no ID.
	FallbackName string `json:"fallbackName,omitempty"`
}

// ---------------------------------------------------------------------------
// ToolLoopAgentToMastraAgent
// ---------------------------------------------------------------------------

// ToolLoopAgentToMastraAgent converts an AI SDK v6 ToolLoopAgent instance into
// a Mastra Agent.
//
// This enables users to create a ToolLoopAgent using the AI SDK's API while
// gaining access to Mastra features like memory, processors, scorers, and
// observability.
//
// Example:
//
//	toolLoopAgent := &toolloopagent.ToolLoopAgent{
//	    Version: "agent-v1",
//	    ID:      "weather-agent",
//	    Settings: &toolloopagent.ToolLoopAgentSettings{
//	        ID:           "weather-agent",
//	        Model:        openaiModel,
//	        Instructions: "You are a helpful weather assistant.",
//	        Tools:        map[string]any{"weather": weatherTool},
//	    },
//	}
//
//	mastraAgent, err := toolloopagent.ToolLoopAgentToMastraAgent(toolLoopAgent, nil)
func ToolLoopAgentToMastraAgent(agent ToolLoopAgentLike, options *ToolLoopAgentToMastraAgentOptions) (*Agent, error) {
	processor, err := NewToolLoopAgentProcessor(agent)
	if err != nil {
		return nil, fmt.Errorf("ToolLoopAgentToMastraAgent: %w", err)
	}

	agentConfig := processor.GetAgentConfig()

	id := agentConfig.ID
	if id == "" && options != nil && options.FallbackName != "" {
		id = options.FallbackName
	}
	if id == "" {
		id = "tool-loop-agent-" + generateID()
	}

	name := agentConfig.Name
	if name == "" {
		name = id
	}

	return &Agent{
		ID:              id,
		Name:            name,
		Instructions:    agentConfig.Instructions,
		Model:           agentConfig.Model,
		Tools:           agentConfig.Tools,
		MaxRetries:      agentConfig.MaxRetries,
		DefaultOptions:  agentConfig.DefaultOptions,
		InputProcessors: []*ToolLoopAgentProcessor{processor},
	}, nil
}
