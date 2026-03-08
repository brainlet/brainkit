// Ported from: packages/openai-compatible/src/chat/openai-compatible-prepare-tools.ts
package openaicompatible

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// OpenAICompatibleToolFunction represents the function definition in an OpenAI-compatible tool.
type OpenAICompatibleToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      *bool  `json:"strict,omitempty"`
}

// OpenAICompatibleTool represents a tool in OpenAI-compatible format.
type OpenAICompatibleTool struct {
	Type     string                       `json:"type"`
	Function OpenAICompatibleToolFunction `json:"function"`
}

// OpenAICompatibleToolChoiceFunction is a specific tool selection by function name.
type OpenAICompatibleToolChoiceFunction struct {
	Type     string                                     `json:"type"`
	Function OpenAICompatibleToolChoiceFunctionReference `json:"function"`
}

// OpenAICompatibleToolChoiceFunctionReference holds the name of a specific function tool.
type OpenAICompatibleToolChoiceFunctionReference struct {
	Name string `json:"name"`
}

// PrepareToolsResult contains the result of preparing tools for an OpenAI-compatible API call.
type PrepareToolsResult struct {
	// Tools is the list of tools in OpenAI-compatible format, or nil if none.
	Tools []OpenAICompatibleTool
	// ToolChoice is the tool choice in OpenAI-compatible format.
	// Can be nil, a string ("auto", "none", "required"), or an OpenAICompatibleToolChoiceFunction.
	ToolChoice any
	// ToolWarnings contains any warnings about unsupported tools.
	ToolWarnings []shared.Warning
}

// PrepareTools converts language model tools and tool choice into OpenAI-compatible format.
func PrepareTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice) PrepareToolsResult {
	// when the tools slice is empty, treat as nil to prevent errors
	if len(tools) == 0 {
		tools = nil
	}

	toolWarnings := []shared.Warning{}

	if tools == nil {
		return PrepareToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	openaiCompatTools := []OpenAICompatibleTool{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
			})
		case languagemodel.FunctionTool:
			oaiTool := OpenAICompatibleTool{
				Type: "function",
				Function: OpenAICompatibleToolFunction{
					Name:       t.Name,
					Parameters: t.InputSchema,
				},
			}
			if t.Description != nil {
				oaiTool.Function.Description = *t.Description
			}
			if t.Strict != nil {
				oaiTool.Function.Strict = t.Strict
			}
			openaiCompatTools = append(openaiCompatTools, oaiTool)
		}
	}

	if toolChoice == nil {
		return PrepareToolsResult{
			Tools:        openaiCompatTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareToolsResult{
			Tools:        openaiCompatTools,
			ToolChoice:   "auto",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return PrepareToolsResult{
			Tools:        openaiCompatTools,
			ToolChoice:   "none",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return PrepareToolsResult{
			Tools:        openaiCompatTools,
			ToolChoice:   "required",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		return PrepareToolsResult{
			Tools: openaiCompatTools,
			ToolChoice: OpenAICompatibleToolChoiceFunction{
				Type: "function",
				Function: OpenAICompatibleToolChoiceFunctionReference{
					Name: tc.ToolName,
				},
			},
			ToolWarnings: toolWarnings,
		}
	default:
		panic(errors.NewUnsupportedFunctionalityError(
			fmt.Sprintf("tool choice type: %T", tc), "",
		))
	}
}
