// Ported from: packages/core/src/tool-loop-agent/index.ts
package toolloopagent

import (
	"crypto/rand"
	"fmt"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Agent is a stub for ../agent.Agent.
// STUB REASON: The real agent.Agent is a large struct with sync.Mutex, Mastra reference,
// LLM, memory, processors, scorers, etc. This stub has only the subset of fields needed
// by ToolLoopAgentToMastraAgent. Importing agent would require refactoring all struct
// literal construction to use the real type's constructor pattern.
type Agent struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Instructions    AgentInstructions      `json:"instructions,omitempty"`
	Model           any                    `json:"model,omitempty"`
	Tools           map[string]any         `json:"tools,omitempty"`
	MaxRetries      *int                   `json:"maxRetries,omitempty"`
	DefaultOptions  *AgentExecutionOptions `json:"defaultOptions,omitempty"`
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
