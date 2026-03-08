// Ported from: packages/groq/src/groq-prepare-tools.ts
package groq

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// GroqToolFunction represents the function definition in a Groq tool.
type GroqToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      *bool  `json:"strict,omitempty"`
}

// GroqFunctionTool represents a function tool in Groq format.
type GroqFunctionTool struct {
	Type     string           `json:"type"`
	Function GroqToolFunction `json:"function"`
}

// GroqBrowserSearchTool represents a browser search tool in Groq format.
type GroqBrowserSearchTool struct {
	Type string `json:"type"`
}

// GroqToolChoiceFunction is a specific tool selection by function name.
type GroqToolChoiceFunction struct {
	Type     string                           `json:"type"`
	Function GroqToolChoiceFunctionReference `json:"function"`
}

// GroqToolChoiceFunctionReference holds the name of a specific function tool.
type GroqToolChoiceFunctionReference struct {
	Name string `json:"name"`
}

// PrepareToolsResult contains the result of preparing tools for a Groq API call.
type PrepareToolsResult struct {
	// Tools is the list of tools in Groq format, or nil if none.
	Tools []any
	// ToolChoice is the tool choice in Groq format.
	// Can be nil, a string ("auto", "none", "required"), or a GroqToolChoiceFunction.
	ToolChoice any
	// ToolWarnings contains any warnings about unsupported tools.
	ToolWarnings []shared.Warning
}

// PrepareTools converts language model tools and tool choice into Groq format.
func PrepareTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice, modelId GroqChatModelId) PrepareToolsResult {
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

	groqTools := []any{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			if t.ID == "groq.browser_search" {
				if !IsBrowserSearchSupportedModel(modelId) {
					detail := fmt.Sprintf("Browser search is only supported on the following models: %s. Current model: %s", GetSupportedModelsString(), modelId)
					toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
						Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
						Details: &detail,
					})
				} else {
					groqTools = append(groqTools, GroqBrowserSearchTool{
						Type: "browser_search",
					})
				}
			} else {
				toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
					Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
				})
			}
		case languagemodel.FunctionTool:
			ft := GroqFunctionTool{
				Type: "function",
				Function: GroqToolFunction{
					Name:       t.Name,
					Parameters: t.InputSchema,
				},
			}
			if t.Description != nil {
				ft.Function.Description = *t.Description
			}
			if t.Strict != nil {
				ft.Function.Strict = t.Strict
			}
			groqTools = append(groqTools, ft)
		}
	}

	if toolChoice == nil {
		return PrepareToolsResult{
			Tools:        groqTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareToolsResult{
			Tools:        groqTools,
			ToolChoice:   "auto",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return PrepareToolsResult{
			Tools:        groqTools,
			ToolChoice:   "none",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return PrepareToolsResult{
			Tools:        groqTools,
			ToolChoice:   "required",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		return PrepareToolsResult{
			Tools: groqTools,
			ToolChoice: GroqToolChoiceFunction{
				Type: "function",
				Function: GroqToolChoiceFunctionReference{
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
