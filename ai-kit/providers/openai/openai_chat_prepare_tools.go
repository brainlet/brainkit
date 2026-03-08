// Ported from: packages/openai/src/chat/openai-chat-prepare-tools.ts
package openai

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// PrepareChatToolsResult contains the result of preparing tools for an OpenAI chat API call.
type PrepareChatToolsResult struct {
	// Tools is the list of tools in OpenAI format, or nil if none.
	Tools []OpenAIChatFunctionTool
	// ToolChoice is the tool choice in OpenAI format.
	ToolChoice OpenAIChatToolChoice
	// ToolWarnings contains any warnings about unsupported tools.
	ToolWarnings []shared.Warning
}

// PrepareChatTools converts language model tools and tool choice into OpenAI chat format.
func PrepareChatTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice) PrepareChatToolsResult {
	// when the tools slice is empty, treat as nil to prevent errors
	if len(tools) == 0 {
		tools = nil
	}

	toolWarnings := []shared.Warning{}

	if tools == nil {
		return PrepareChatToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	openaiTools := []OpenAIChatFunctionTool{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			oaiTool := OpenAIChatFunctionTool{
				Type: "function",
				Function: OpenAIChatFunctionToolFunction{
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
			openaiTools = append(openaiTools, oaiTool)
		default:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("tool type: %T", t),
			})
		}
	}

	if toolChoice == nil {
		return PrepareChatToolsResult{
			Tools:        openaiTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareChatToolsResult{
			Tools:        openaiTools,
			ToolChoice:   "auto",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return PrepareChatToolsResult{
			Tools:        openaiTools,
			ToolChoice:   "none",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return PrepareChatToolsResult{
			Tools:        openaiTools,
			ToolChoice:   "required",
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		return PrepareChatToolsResult{
			Tools: openaiTools,
			ToolChoice: OpenAIChatToolChoiceFunction{
				Type: "function",
				Function: OpenAIChatToolChoiceFunctionReference{
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
