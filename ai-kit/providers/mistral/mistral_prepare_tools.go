// Ported from: packages/mistral/src/mistral-prepare-tools.ts
package mistral

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// MistralToolFunction represents a Mistral tool function definition.
type MistralToolFunction struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Parameters  any     `json:"parameters"`
	Strict      *bool   `json:"strict,omitempty"`
}

// MistralTool represents a tool in Mistral format.
type MistralTool struct {
	Type     string              `json:"type"`
	Function MistralToolFunction `json:"function"`
}

// PrepareToolsResult contains the result of preparing tools for a Mistral API call.
type PrepareToolsResult struct {
	Tools        []MistralTool
	ToolChoice   MistralToolChoice
	ToolWarnings []shared.Warning
}

// PrepareTools converts language model tools and tool choice to Mistral format.
func PrepareTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice) PrepareToolsResult {
	// When the tools array is empty, treat as nil to prevent errors.
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

	var mistralTools []MistralTool

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
			})
		case languagemodel.FunctionTool:
			mt := MistralTool{
				Type: "function",
				Function: MistralToolFunction{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.InputSchema,
				},
			}
			if t.Strict != nil {
				mt.Function.Strict = t.Strict
			}
			mistralTools = append(mistralTools, mt)
		}
	}

	if toolChoice == nil {
		return PrepareToolsResult{
			Tools:        mistralTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareToolsResult{
			Tools:        mistralTools,
			ToolChoice:   MistralToolChoiceAuto,
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return PrepareToolsResult{
			Tools:        mistralTools,
			ToolChoice:   MistralToolChoiceNone,
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return PrepareToolsResult{
			Tools:        mistralTools,
			ToolChoice:   MistralToolChoiceAny,
			ToolWarnings: toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		// Mistral does not support tool mode directly,
		// so we filter the tools and force the tool choice through 'any'.
		var filtered []MistralTool
		for _, tool := range mistralTools {
			if tool.Function.Name == tc.ToolName {
				filtered = append(filtered, tool)
			}
		}
		return PrepareToolsResult{
			Tools:        filtered,
			ToolChoice:   MistralToolChoiceAny,
			ToolWarnings: toolWarnings,
		}
	default:
		panic(errors.NewUnsupportedFunctionalityError(
			fmt.Sprintf("tool choice type: %T", tc), "",
		))
	}
}
