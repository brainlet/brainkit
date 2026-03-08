// Ported from: packages/deepseek/src/chat/deepseek-prepare-tools.ts
package deepseek

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// PrepareToolsResult contains the result of preparing tools for a DeepSeek API call.
type PrepareToolsResult struct {
	// Tools is the list of tools in DeepSeek format, or nil if none.
	Tools []DeepSeekFunctionTool
	// ToolChoice is the tool choice in DeepSeek format.
	ToolChoice DeepSeekToolChoice
	// ToolWarnings contains any warnings about unsupported tools.
	ToolWarnings []shared.Warning
}

// PrepareTools converts language model tools and tool choice into DeepSeek format.
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

	deepseekTools := []DeepSeekFunctionTool{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
			})
		case languagemodel.FunctionTool:
			dsTool := DeepSeekFunctionTool{
				Type: "function",
				Function: DeepSeekFunctionToolFunction{
					Name:       t.Name,
					Parameters: t.InputSchema,
				},
			}
			if t.Description != nil {
				dsTool.Function.Description = *t.Description
			}
			if t.Strict != nil {
				dsTool.Function.Strict = t.Strict
			}
			deepseekTools = append(deepseekTools, dsTool)
		}
	}

	if toolChoice == nil {
		return PrepareToolsResult{
			Tools:        deepseekTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareToolsResult{
			Tools:        deepseekTools,
			ToolChoice:   "auto",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return PrepareToolsResult{
			Tools:        deepseekTools,
			ToolChoice:   "none",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return PrepareToolsResult{
			Tools:        deepseekTools,
			ToolChoice:   "required",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		return PrepareToolsResult{
			Tools: deepseekTools,
			ToolChoice: DeepSeekToolChoiceFunction{
				Type: "function",
				Function: DeepSeekToolChoiceFunctionReference{
					Name: tc.ToolName,
				},
			},
			ToolWarnings: toolWarnings,
		}
	default:
		return PrepareToolsResult{
			Tools:      deepseekTools,
			ToolChoice: nil,
			ToolWarnings: append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("tool choice type: %T", tc),
			}),
		}
	}
}
