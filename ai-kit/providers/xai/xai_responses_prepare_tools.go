// Ported from: packages/xai/src/responses/xai-responses-prepare-tools.ts
package xai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// xaiResponsesToolChoice represents the tool choice for responses API.
// Can be "auto", "none", "required", or map[string]interface{}{"type": "function", "name": "..."}
type xaiResponsesToolChoice = interface{}

// prepareResponsesToolsResult is the result of preparing tools for the responses API.
type prepareResponsesToolsResult struct {
	Tools      []XaiResponsesTool
	ToolChoice xaiResponsesToolChoice
	Warnings   []shared.Warning
}

// prepareResponsesTools converts AI SDK tools and tool choice to xAI responses API format.
func prepareResponsesTools(tools []languagemodel.Tool, toolChoice languagemodel.ToolChoice) prepareResponsesToolsResult {
	var toolWarnings []shared.Warning

	// Normalize empty tools to nil
	if len(tools) == 0 {
		tools = nil
	}

	if tools == nil {
		return prepareResponsesToolsResult{
			Tools:      nil,
			ToolChoice: nil,
			Warnings:   toolWarnings,
		}
	}

	var xaiTools []XaiResponsesTool
	toolByName := make(map[string]languagemodel.Tool)

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.ProviderTool:
			toolByName[t.Name] = t

			switch t.ID {
			case "xai.web_search":
				toolEntry := XaiResponsesTool{
					"type": "web_search",
				}
				if t.Args != nil {
					if v, ok := t.Args["allowedDomains"]; ok {
						toolEntry["allowed_domains"] = v
					}
					if v, ok := t.Args["excludedDomains"]; ok {
						toolEntry["excluded_domains"] = v
					}
					if v, ok := t.Args["enableImageUnderstanding"]; ok {
						toolEntry["enable_image_understanding"] = v
					}
				}
				xaiTools = append(xaiTools, toolEntry)

			case "xai.x_search":
				toolEntry := XaiResponsesTool{
					"type": "x_search",
				}
				if t.Args != nil {
					if v, ok := t.Args["allowedXHandles"]; ok {
						toolEntry["allowed_x_handles"] = v
					}
					if v, ok := t.Args["excludedXHandles"]; ok {
						toolEntry["excluded_x_handles"] = v
					}
					if v, ok := t.Args["fromDate"]; ok {
						toolEntry["from_date"] = v
					}
					if v, ok := t.Args["toDate"]; ok {
						toolEntry["to_date"] = v
					}
					if v, ok := t.Args["enableImageUnderstanding"]; ok {
						toolEntry["enable_image_understanding"] = v
					}
					if v, ok := t.Args["enableVideoUnderstanding"]; ok {
						toolEntry["enable_video_understanding"] = v
					}
				}
				xaiTools = append(xaiTools, toolEntry)

			case "xai.code_execution":
				xaiTools = append(xaiTools, XaiResponsesTool{
					"type": "code_interpreter",
				})

			case "xai.view_image":
				xaiTools = append(xaiTools, XaiResponsesTool{
					"type": "view_image",
				})

			case "xai.view_x_video":
				xaiTools = append(xaiTools, XaiResponsesTool{
					"type": "view_x_video",
				})

			case "xai.file_search":
				toolEntry := XaiResponsesTool{
					"type": "file_search",
				}
				if t.Args != nil {
					if v, ok := t.Args["vectorStoreIds"]; ok {
						toolEntry["vector_store_ids"] = v
					}
					if v, ok := t.Args["maxNumResults"]; ok {
						toolEntry["max_num_results"] = v
					}
				}
				xaiTools = append(xaiTools, toolEntry)

			case "xai.mcp":
				toolEntry := XaiResponsesTool{
					"type": "mcp",
				}
				if t.Args != nil {
					if v, ok := t.Args["serverUrl"]; ok {
						toolEntry["server_url"] = v
					}
					if v, ok := t.Args["serverLabel"]; ok {
						toolEntry["server_label"] = v
					}
					if v, ok := t.Args["serverDescription"]; ok {
						toolEntry["server_description"] = v
					}
					if v, ok := t.Args["allowedTools"]; ok {
						toolEntry["allowed_tools"] = v
					}
					if v, ok := t.Args["headers"]; ok {
						toolEntry["headers"] = v
					}
					if v, ok := t.Args["authorization"]; ok {
						toolEntry["authorization"] = v
					}
				}
				xaiTools = append(xaiTools, toolEntry)

			default:
				toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
					Feature: "provider-defined tool " + t.Name,
				})
			}

		case languagemodel.FunctionTool:
			toolByName[t.Name] = t

			toolEntry := XaiResponsesTool{
				"type":       "function",
				"name":       t.Name,
				"parameters": t.InputSchema,
			}
			if t.Description != nil {
				toolEntry["description"] = *t.Description
			}
			if t.Strict != nil {
				toolEntry["strict"] = *t.Strict
			}
			xaiTools = append(xaiTools, toolEntry)
		}
	}

	if toolChoice == nil {
		return prepareResponsesToolsResult{
			Tools:      xaiTools,
			ToolChoice: nil,
			Warnings:   toolWarnings,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return prepareResponsesToolsResult{
			Tools:      xaiTools,
			ToolChoice: "auto",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceNone:
		return prepareResponsesToolsResult{
			Tools:      xaiTools,
			ToolChoice: "none",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceRequired:
		return prepareResponsesToolsResult{
			Tools:      xaiTools,
			ToolChoice: "required",
			Warnings:   toolWarnings,
		}
	case languagemodel.ToolChoiceTool:
		selectedTool, exists := toolByName[tc.ToolName]
		if !exists {
			return prepareResponsesToolsResult{
				Tools:      xaiTools,
				ToolChoice: nil,
				Warnings:   toolWarnings,
			}
		}

		if _, isProvider := selectedTool.(languagemodel.ProviderTool); isProvider {
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: "toolChoice for server-side tool \"" + tc.ToolName + "\"",
			})
			return prepareResponsesToolsResult{
				Tools:      xaiTools,
				ToolChoice: nil,
				Warnings:   toolWarnings,
			}
		}

		return prepareResponsesToolsResult{
			Tools: xaiTools,
			ToolChoice: map[string]interface{}{
				"type": "function",
				"name": tc.ToolName,
			},
			Warnings: toolWarnings,
		}
	default:
		panic(errors.NewUnsupportedFunctionalityError("tool choice type", ""))
	}
}
