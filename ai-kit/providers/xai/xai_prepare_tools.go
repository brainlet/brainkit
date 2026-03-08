// Ported from: packages/xai/src/xai-prepare-tools.ts
package xai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// prepareToolsResult is the result of preparing tools for xAI.
type prepareToolsResult struct {
	Tools      []interface{}
	ToolChoice interface{} // string or map
	Warnings   []shared.Warning
}

// prepareTools converts AI SDK tools and tool choice to xAI format.
func prepareTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice) prepareToolsResult {
	var toolWarnings []shared.Warning

	// When the tools array is empty, treat as nil
	if len(tools) == 0 {
		tools = nil
	}

	if tools == nil {
		return prepareToolsResult{
			Tools:      nil,
			ToolChoice: nil,
			Warnings:   toolWarnings,
		}
	}

	var xaiTools []interface{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: "provider-defined tool " + t.Name,
			})
		case languagemodel.FunctionTool:
			fn := map[string]interface{}{
				"name":       t.Name,
				"parameters": t.InputSchema,
			}
			if t.Description != nil {
				fn["description"] = *t.Description
			}
			if t.Strict != nil {
				fn["strict"] = *t.Strict
			}
			xaiTools = append(xaiTools, map[string]interface{}{
				"type":     "function",
				"function": fn,
			})
		}
	}

	if toolChoice == nil {
		return prepareToolsResult{
			Tools:      xaiTools,
			ToolChoice: nil,
			Warnings:   toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return prepareToolsResult{
			Tools:      xaiTools,
			ToolChoice: "auto",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return prepareToolsResult{
			Tools:      xaiTools,
			ToolChoice: "none",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return prepareToolsResult{
			Tools:      xaiTools,
			ToolChoice: "required",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		return prepareToolsResult{
			Tools: xaiTools,
			ToolChoice: map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name": tc.ToolName,
				},
			},
			Warnings: toolWarnings,
		}
	default:
		panic(errors.NewUnsupportedFunctionalityError("tool choice type", ""))
	}
}
