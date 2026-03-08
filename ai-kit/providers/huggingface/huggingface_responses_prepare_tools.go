// Ported from: packages/huggingface/src/responses/huggingface-responses-prepare-tools.ts
package huggingface

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// ResponsesTool represents a tool in the HuggingFace responses format.
type ResponsesTool struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
}

// ResponsesToolChoice represents a tool choice in the HuggingFace responses format.
// It can be a string ("auto", "required") or a structured object.
type ResponsesToolChoice struct {
	// StringValue holds the value when tool choice is a simple string.
	StringValue *string
	// FunctionValue holds the value when tool choice is a specific function.
	FunctionValue *ResponsesToolChoiceFunction
}

// ResponsesToolChoiceFunction represents a function tool choice.
type ResponsesToolChoiceFunction struct {
	Type     string                              `json:"type"`
	Function ResponsesToolChoiceFunctionDetails  `json:"function"`
}

// ResponsesToolChoiceFunctionDetails holds the function name for a specific tool choice.
type ResponsesToolChoiceFunctionDetails struct {
	Name string `json:"name"`
}

// MarshalToolChoice returns the JSON-serializable representation of the tool choice.
func (tc *ResponsesToolChoice) MarshalToolChoice() any {
	if tc == nil {
		return nil
	}
	if tc.StringValue != nil {
		return *tc.StringValue
	}
	if tc.FunctionValue != nil {
		return tc.FunctionValue
	}
	return nil
}

// PrepareResponsesToolsResult holds the result of prepareResponsesTools.
type PrepareResponsesToolsResult struct {
	Tools        []ResponsesTool
	ToolChoice   *ResponsesToolChoice
	ToolWarnings []shared.Warning
}

// prepareResponsesTools converts language model tools and tool choice to
// the HuggingFace responses format.
func prepareResponsesTools(
	tools []languagemodel.Tool,
	toolChoice languagemodel.ToolChoice,
) PrepareResponsesToolsResult {
	// When the tools array is empty, treat it as nil to prevent errors.
	if len(tools) == 0 {
		tools = nil
	}

	toolWarnings := []shared.Warning{}

	if tools == nil {
		return PrepareResponsesToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	huggingfaceTools := []ResponsesTool{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			desc := ""
			if t.Description != nil {
				desc = *t.Description
			}
			huggingfaceTools = append(huggingfaceTools, ResponsesTool{
				Type:        "function",
				Name:        t.Name,
				Description: desc,
				Parameters:  t.InputSchema,
			})
		case languagemodel.ProviderTool:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
			})
		default:
			panic(fmt.Sprintf("Unsupported tool type: %T", tool))
		}
	}

	// Prepare tool choice.
	var mappedToolChoice *ResponsesToolChoice
	if toolChoice != nil {
		switch toolChoice.(type) {
		case languagemodel.ToolChoiceAuto:
			auto := "auto"
			mappedToolChoice = &ResponsesToolChoice{StringValue: &auto}
		case languagemodel.ToolChoiceRequired:
			required := "required"
			mappedToolChoice = &ResponsesToolChoice{StringValue: &required}
		case languagemodel.ToolChoiceNone:
			// Not supported, ignore.
		case languagemodel.ToolChoiceTool:
			tc := toolChoice.(languagemodel.ToolChoiceTool)
			mappedToolChoice = &ResponsesToolChoice{
				FunctionValue: &ResponsesToolChoiceFunction{
					Type: "function",
					Function: ResponsesToolChoiceFunctionDetails{
						Name: tc.ToolName,
					},
				},
			}
		default:
			panic(fmt.Sprintf("Unsupported tool choice type: %T", toolChoice))
		}
	}

	return PrepareResponsesToolsResult{
		Tools:        huggingfaceTools,
		ToolChoice:   mappedToolChoice,
		ToolWarnings: toolWarnings,
	}
}
