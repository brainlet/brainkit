// Ported from: packages/anthropic/src/anthropic-prepare-tools.ts
package anthropic

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// AnthropicToolOptions contains Anthropic-specific provider options for tools.
type AnthropicToolOptions struct {
	DeferLoading       *bool    `json:"deferLoading,omitempty"`
	AllowedCallers     []string `json:"allowedCallers,omitempty"`
	EagerInputStreaming *bool   `json:"eagerInputStreaming,omitempty"`
}

// PrepareToolsResult contains the result of preparing tools for the Anthropic API.
type PrepareToolsResult struct {
	Tools        []AnthropicTool
	ToolChoice   *AnthropicToolChoice
	ToolWarnings []shared.Warning
	Betas        map[string]bool // Set<string> equivalent
}

// prepareTools converts SDK tools to Anthropic API tool format.
func prepareTools(
	tools []languagemodel.Tool,
	toolChoice languagemodel.ToolChoice,
	disableParallelToolUse *bool,
	cacheControlValidator *CacheControlValidator,
	supportsStructuredOutput bool,
) PrepareToolsResult {
	// When the tools array is empty, change it to nil to prevent errors
	if len(tools) == 0 {
		tools = nil
	}

	toolWarnings := []shared.Warning{}
	betas := map[string]bool{}
	validator := cacheControlValidator
	if validator == nil {
		validator = NewCacheControlValidator()
	}

	if tools == nil {
		return PrepareToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}
	}

	anthropicTools := []AnthropicTool{}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			cc := validator.GetCacheControl(t.ProviderOptions, CacheControlContext{
				Type:     "tool definition",
				CanCache: true,
			})

			// Read Anthropic-specific provider options
			var anthropicOpts *AnthropicToolOptions
			if t.ProviderOptions != nil {
				if aoMap, ok := t.ProviderOptions["anthropic"]; ok && aoMap != nil {
					anthropicOpts = &AnthropicToolOptions{}
					if v, ok := aoMap["eagerInputStreaming"].(bool); ok {
						anthropicOpts.EagerInputStreaming = &v
					}
					if v, ok := aoMap["deferLoading"].(bool); ok {
						anthropicOpts.DeferLoading = &v
					}
					if v, ok := aoMap["allowedCallers"]; ok {
						if callers, ok := v.([]any); ok {
							for _, c := range callers {
								if s, ok := c.(string); ok {
									anthropicOpts.AllowedCallers = append(anthropicOpts.AllowedCallers, s)
								}
							}
						}
					}
				}
			}

			at := AnthropicTool{
				Name:         t.Name,
				Description:  t.Description,
				InputSchema:  t.InputSchema,
				CacheControl: cc,
			}

			if anthropicOpts != nil {
				at.EagerInputStreaming = anthropicOpts.EagerInputStreaming
				at.DeferLoading = anthropicOpts.DeferLoading
				at.AllowedCallers = anthropicOpts.AllowedCallers
			}

			if supportsStructuredOutput && t.Strict != nil {
				at.Strict = t.Strict
			}

			if len(t.InputExamples) > 0 {
				examples := make([]any, len(t.InputExamples))
				for i, example := range t.InputExamples {
					examples[i] = example.Input
				}
				at.InputExamples = examples
			}

			if supportsStructuredOutput {
				betas["structured-outputs-2025-11-13"] = true
			}

			if len(t.InputExamples) > 0 || len(at.AllowedCallers) > 0 {
				betas["advanced-tool-use-2025-11-20"] = true
			}

			anthropicTools = append(anthropicTools, at)

		case languagemodel.ProviderTool:
			switch t.ID {
			case "anthropic.code_execution_20250522":
				betas["code-execution-2025-05-22"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type: "code_execution_20250522",
					Name: "code_execution",
				})

			case "anthropic.code_execution_20250825":
				betas["code-execution-2025-08-25"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type: "code_execution_20250825",
					Name: "code_execution",
				})

			case "anthropic.code_execution_20260120":
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type: "code_execution_20260120",
					Name: "code_execution",
				})

			case "anthropic.computer_20250124":
				betas["computer-use-2025-01-24"] = true
				dwpx := intFromArgs(t.Args, "displayWidthPx")
				dhpx := intFromArgs(t.Args, "displayHeightPx")
				dn := intFromArgs(t.Args, "displayNumber")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name:            "computer",
					Type:            "computer_20250124",
					DisplayWidthPx:  dwpx,
					DisplayHeightPx: dhpx,
					DisplayNumber:   dn,
				})

			case "anthropic.computer_20251124":
				betas["computer-use-2025-11-24"] = true
				dwpx := intFromArgs(t.Args, "displayWidthPx")
				dhpx := intFromArgs(t.Args, "displayHeightPx")
				dn := intFromArgs(t.Args, "displayNumber")
				ez := boolFromArgs(t.Args, "enableZoom")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name:            "computer",
					Type:            "computer_20251124",
					DisplayWidthPx:  dwpx,
					DisplayHeightPx: dhpx,
					DisplayNumber:   dn,
					EnableZoom:      ez,
				})

			case "anthropic.computer_20241022":
				betas["computer-use-2024-10-22"] = true
				dwpx := intFromArgs(t.Args, "displayWidthPx")
				dhpx := intFromArgs(t.Args, "displayHeightPx")
				dn := intFromArgs(t.Args, "displayNumber")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name:            "computer",
					Type:            "computer_20241022",
					DisplayWidthPx:  dwpx,
					DisplayHeightPx: dhpx,
					DisplayNumber:   dn,
				})

			case "anthropic.text_editor_20250124":
				betas["computer-use-2025-01-24"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "str_replace_editor",
					Type: "text_editor_20250124",
				})

			case "anthropic.text_editor_20241022":
				betas["computer-use-2024-10-22"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "str_replace_editor",
					Type: "text_editor_20241022",
				})

			case "anthropic.text_editor_20250429":
				betas["computer-use-2025-01-24"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "str_replace_based_edit_tool",
					Type: "text_editor_20250429",
				})

			case "anthropic.text_editor_20250728":
				mc := intFromArgs(t.Args, "maxCharacters")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name:          "str_replace_based_edit_tool",
					Type:          "text_editor_20250728",
					MaxCharacters: mc,
				})

			case "anthropic.bash_20250124":
				betas["computer-use-2025-01-24"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "bash",
					Type: "bash_20250124",
				})

			case "anthropic.bash_20241022":
				betas["computer-use-2024-10-22"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "bash",
					Type: "bash_20241022",
				})

			case "anthropic.memory_20250818":
				betas["context-management-2025-06-27"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Name: "memory",
					Type: "memory_20250818",
				})

			case "anthropic.web_fetch_20250910":
				betas["web-fetch-2025-09-10"] = true
				mu := intFromArgs(t.Args, "maxUses")
				mct := intFromArgs(t.Args, "maxContentTokens")
				ad := stringSliceFromArgs(t.Args, "allowedDomains")
				bd := stringSliceFromArgs(t.Args, "blockedDomains")
				cit := mapFromArgs(t.Args, "citations")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type:             "web_fetch_20250910",
					Name:             "web_fetch",
					MaxUses:          mu,
					AllowedDomains:   ad,
					BlockedDomains:   bd,
					Citations:        cit,
					MaxContentTokens: mct,
				})

			case "anthropic.web_fetch_20260209":
				betas["code-execution-web-tools-2026-02-09"] = true
				mu := intFromArgs(t.Args, "maxUses")
				mct := intFromArgs(t.Args, "maxContentTokens")
				ad := stringSliceFromArgs(t.Args, "allowedDomains")
				bd := stringSliceFromArgs(t.Args, "blockedDomains")
				cit := mapFromArgs(t.Args, "citations")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type:             "web_fetch_20260209",
					Name:             "web_fetch",
					MaxUses:          mu,
					AllowedDomains:   ad,
					BlockedDomains:   bd,
					Citations:        cit,
					MaxContentTokens: mct,
				})

			case "anthropic.web_search_20250305":
				mu := intFromArgs(t.Args, "maxUses")
				ad := stringSliceFromArgs(t.Args, "allowedDomains")
				bd := stringSliceFromArgs(t.Args, "blockedDomains")
				ul := mapFromArgs(t.Args, "userLocation")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type:           "web_search_20250305",
					Name:           "web_search",
					MaxUses:        mu,
					AllowedDomains: ad,
					BlockedDomains: bd,
					UserLocation:   ul,
				})

			case "anthropic.web_search_20260209":
				betas["code-execution-web-tools-2026-02-09"] = true
				mu := intFromArgs(t.Args, "maxUses")
				ad := stringSliceFromArgs(t.Args, "allowedDomains")
				bd := stringSliceFromArgs(t.Args, "blockedDomains")
				ul := mapFromArgs(t.Args, "userLocation")
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type:           "web_search_20260209",
					Name:           "web_search",
					MaxUses:        mu,
					AllowedDomains: ad,
					BlockedDomains: bd,
					UserLocation:   ul,
				})

			case "anthropic.tool_search_regex_20251119":
				betas["advanced-tool-use-2025-11-20"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type: "tool_search_tool_regex_20251119",
					Name: "tool_search_tool_regex",
				})

			case "anthropic.tool_search_bm25_20251119":
				betas["advanced-tool-use-2025-11-20"] = true
				anthropicTools = append(anthropicTools, AnthropicTool{
					Type: "tool_search_tool_bm25_20251119",
					Name: "tool_search_tool_bm25",
				})

			default:
				toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
					Feature: fmt.Sprintf("provider-defined tool %s", t.ID),
				})
			}

		default:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("tool type %T", tool),
			})
		}
	}

	// Handle tool choice
	if toolChoice == nil {
		var tc *AnthropicToolChoice
		if disableParallelToolUse != nil && *disableParallelToolUse {
			tc = &AnthropicToolChoice{
				Type:                   "auto",
				DisableParallelToolUse: disableParallelToolUse,
			}
		}
		return PrepareToolsResult{
			Tools:        anthropicTools,
			ToolChoice:   tc,
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}
	}

	switch tc := toolChoice.(type) {
	case languagemodel.ToolChoiceAuto:
		return PrepareToolsResult{
			Tools: anthropicTools,
			ToolChoice: &AnthropicToolChoice{
				Type:                   "auto",
				DisableParallelToolUse: disableParallelToolUse,
			},
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}

	case languagemodel.ToolChoiceRequired:
		return PrepareToolsResult{
			Tools: anthropicTools,
			ToolChoice: &AnthropicToolChoice{
				Type:                   "any",
				DisableParallelToolUse: disableParallelToolUse,
			},
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}

	case languagemodel.ToolChoiceNone:
		// Anthropic does not support 'none' tool choice, so we remove the tools
		return PrepareToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}

	case languagemodel.ToolChoiceTool:
		return PrepareToolsResult{
			Tools: anthropicTools,
			ToolChoice: &AnthropicToolChoice{
				Type:                   "tool",
				Name:                   tc.ToolName,
				DisableParallelToolUse: disableParallelToolUse,
			},
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}

	default:
		toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
			Feature: fmt.Sprintf("tool choice type %T", toolChoice),
		})
		return PrepareToolsResult{
			Tools:        anthropicTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
			Betas:        betas,
		}
	}
}

// Helper functions for extracting typed values from map[string]any args.

func intFromArgs(args map[string]any, key string) *int {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case int:
		return &val
	case float64:
		i := int(val)
		return &i
	case int64:
		i := int(val)
		return &i
	default:
		return nil
	}
}

func boolFromArgs(args map[string]any, key string) *bool {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	if val, ok := v.(bool); ok {
		return &val
	}
	return nil
}

func stringSliceFromArgs(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	if arr, ok := v.([]any); ok {
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if arr, ok := v.([]string); ok {
		return arr
	}
	return nil
}

func mapFromArgs(args map[string]any, key string) *map[string]any {
	if args == nil {
		return nil
	}
	v, ok := args[key]
	if !ok || v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return &m
	}
	return nil
}
