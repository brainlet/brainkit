// Ported from: packages/openai/src/responses/openai-responses-prepare-tools.ts
package openai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// PrepareResponsesToolsOptions are the options for PrepareResponsesTools.
type PrepareResponsesToolsOptions struct {
	Tools                   []languagemodel.Tool
	ToolChoice              languagemodel.ToolChoice
	ToolNameMapping         *providerutils.ToolNameMapping
	CustomProviderToolNames map[string]struct{}
}

// PrepareResponsesToolsResult is the result of PrepareResponsesTools.
type PrepareResponsesToolsResult struct {
	Tools        []OpenAIResponsesTool
	ToolChoice   any // string ("auto"|"none"|"required") or map[string]any
	ToolWarnings []shared.Warning
}

// PrepareResponsesTools converts language model tools and tool choice into
// the format expected by the OpenAI Responses API.
func PrepareResponsesTools(opts PrepareResponsesToolsOptions) (*PrepareResponsesToolsResult, error) {
	tools := opts.Tools
	toolChoice := opts.ToolChoice

	// When the tools array is empty, change it to nil to prevent errors.
	if len(tools) == 0 {
		tools = nil
	}

	toolWarnings := []shared.Warning{}

	if tools == nil {
		return &PrepareResponsesToolsResult{
			Tools:        nil,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}, nil
	}

	var openaiTools []OpenAIResponsesTool
	resolvedCustomProviderToolNames := opts.CustomProviderToolNames
	if resolvedCustomProviderToolNames == nil {
		resolvedCustomProviderToolNames = make(map[string]struct{})
	}

	for _, tool := range tools {
		switch t := tool.(type) {
		case languagemodel.FunctionTool:
			entry := OpenAIResponsesTool{
				"type":        "function",
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			}
			if t.Strict != nil {
				entry["strict"] = *t.Strict
			}
			openaiTools = append(openaiTools, entry)

		case languagemodel.ProviderTool:
			providerTool, err := convertProviderTool(t, resolvedCustomProviderToolNames)
			if err != nil {
				return nil, err
			}
			if providerTool != nil {
				openaiTools = append(openaiTools, providerTool)
			}

		default:
			toolWarnings = append(toolWarnings, shared.UnsupportedWarning{
				Feature: "function tool",
			})
		}
	}

	if toolChoice == nil {
		return &PrepareResponsesToolsResult{
			Tools:        openaiTools,
			ToolChoice:   nil,
			ToolWarnings: toolWarnings,
		}, nil
	}

	resolvedToolChoice, err := convertToolChoice(toolChoice, opts.ToolNameMapping, resolvedCustomProviderToolNames)
	if err != nil {
		return nil, err
	}

	return &PrepareResponsesToolsResult{
		Tools:        openaiTools,
		ToolChoice:   resolvedToolChoice,
		ToolWarnings: toolWarnings,
	}, nil
}

// convertProviderTool converts a provider tool to the OpenAI Responses API format.
func convertProviderTool(tool languagemodel.ProviderTool, customNames map[string]struct{}) (OpenAIResponsesTool, error) {
	switch tool.ID {
	case "openai.file_search":
		entry := OpenAIResponsesTool{"type": "file_search"}
		args := tool.Args
		if args != nil {
			if vectorStoreIDs, ok := args["vectorStoreIds"]; ok {
				entry["vector_store_ids"] = vectorStoreIDs
			}
			if maxNumResults, ok := args["maxNumResults"]; ok {
				entry["max_num_results"] = maxNumResults
			}
			if ranking, ok := args["ranking"].(map[string]any); ok {
				rankOpts := map[string]any{}
				if ranker, ok := ranking["ranker"]; ok {
					rankOpts["ranker"] = ranker
				}
				if scoreThreshold, ok := ranking["scoreThreshold"]; ok {
					rankOpts["score_threshold"] = scoreThreshold
				}
				entry["ranking_options"] = rankOpts
			}
			if filters, ok := args["filters"]; ok {
				entry["filters"] = filters
			}
		}
		return entry, nil

	case "openai.local_shell":
		return OpenAIResponsesTool{"type": "local_shell"}, nil

	case "openai.shell":
		entry := OpenAIResponsesTool{"type": "shell"}
		args := tool.Args
		if args != nil {
			if env, ok := args["environment"]; ok {
				if envMap, ok := env.(map[string]any); ok {
					entry["environment"] = mapShellEnvironment(envMap)
				}
			}
		}
		return entry, nil

	case "openai.apply_patch":
		return OpenAIResponsesTool{"type": "apply_patch"}, nil

	case "openai.web_search_preview":
		entry := OpenAIResponsesTool{"type": "web_search_preview"}
		args := tool.Args
		if args != nil {
			if v, ok := args["searchContextSize"]; ok {
				entry["search_context_size"] = v
			}
			if v, ok := args["userLocation"]; ok {
				entry["user_location"] = v
			}
		}
		return entry, nil

	case "openai.web_search":
		entry := OpenAIResponsesTool{"type": "web_search"}
		args := tool.Args
		if args != nil {
			if filters, ok := args["filters"].(map[string]any); ok {
				entry["filters"] = map[string]any{
					"allowed_domains": filters["allowedDomains"],
				}
			}
			if v, ok := args["externalWebAccess"]; ok {
				entry["external_web_access"] = v
			}
			if v, ok := args["searchContextSize"]; ok {
				entry["search_context_size"] = v
			}
			if v, ok := args["userLocation"]; ok {
				entry["user_location"] = v
			}
		}
		return entry, nil

	case "openai.code_interpreter":
		entry := OpenAIResponsesTool{"type": "code_interpreter"}
		args := tool.Args
		if args != nil {
			container := args["container"]
			switch c := container.(type) {
			case nil:
				entry["container"] = map[string]any{"type": "auto"}
			case string:
				entry["container"] = c
			case map[string]any:
				containerEntry := map[string]any{"type": "auto"}
				if fileIDs, ok := c["fileIds"]; ok {
					containerEntry["file_ids"] = fileIDs
				}
				entry["container"] = containerEntry
			}
		} else {
			entry["container"] = map[string]any{"type": "auto"}
		}
		return entry, nil

	case "openai.image_generation":
		entry := OpenAIResponsesTool{"type": "image_generation"}
		args := tool.Args
		if args != nil {
			for _, key := range []string{"background", "inputFidelity", "model", "moderation", "quality", "size"} {
				if v, ok := args[key]; ok {
					// Convert camelCase to snake_case for the API
					apiKey := camelToSnake(key)
					entry[apiKey] = v
				}
			}
			if v, ok := args["outputCompression"]; ok {
				entry["output_compression"] = v
			}
			if v, ok := args["outputFormat"]; ok {
				entry["output_format"] = v
			}
			if v, ok := args["partialImages"]; ok {
				entry["partial_images"] = v
			}
			if mask, ok := args["inputImageMask"].(map[string]any); ok {
				entry["input_image_mask"] = map[string]any{
					"file_id":   mask["fileId"],
					"image_url": mask["imageUrl"],
				}
			}
		}
		return entry, nil

	case "openai.mcp":
		entry := OpenAIResponsesTool{"type": "mcp"}
		args := tool.Args
		if args != nil {
			if v, ok := args["serverLabel"]; ok {
				entry["server_label"] = v
			}
			if allowedTools, ok := args["allowedTools"]; ok {
				switch at := allowedTools.(type) {
				case []any:
					entry["allowed_tools"] = at
				case map[string]any:
					entry["allowed_tools"] = map[string]any{
						"read_only":  at["readOnly"],
						"tool_names": at["toolNames"],
					}
				}
			}
			if v, ok := args["authorization"]; ok {
				entry["authorization"] = v
			}
			if v, ok := args["connectorId"]; ok {
				entry["connector_id"] = v
			}
			if v, ok := args["headers"]; ok {
				entry["headers"] = v
			}
			if v, ok := args["serverDescription"]; ok {
				entry["server_description"] = v
			}
			if v, ok := args["serverUrl"]; ok {
				entry["server_url"] = v
			}

			requireApproval := args["requireApproval"]
			switch ra := requireApproval.(type) {
			case nil:
				entry["require_approval"] = "never"
			case string:
				entry["require_approval"] = ra
			case map[string]any:
				if never, ok := ra["never"].(map[string]any); ok {
					entry["require_approval"] = map[string]any{
						"never": map[string]any{
							"tool_names": never["toolNames"],
						},
					}
				} else {
					entry["require_approval"] = "never"
				}
			default:
				entry["require_approval"] = "never"
			}
		}
		return entry, nil

	case "openai.custom":
		args := tool.Args
		if args != nil {
			entry := OpenAIResponsesTool{
				"type": "custom",
				"name": args["name"],
			}
			if v, ok := args["description"]; ok {
				entry["description"] = v
			}
			if v, ok := args["format"]; ok {
				entry["format"] = v
			}
			if name, ok := args["name"].(string); ok {
				customNames[name] = struct{}{}
			}
			return entry, nil
		}
		return nil, nil
	}

	return nil, nil
}

// convertToolChoice converts a ToolChoice to the OpenAI Responses API format.
func convertToolChoice(
	tc languagemodel.ToolChoice,
	toolNameMapping *providerutils.ToolNameMapping,
	customProviderToolNames map[string]struct{},
) (any, error) {
	switch choice := tc.(type) {
	case languagemodel.ToolChoiceAuto:
		return "auto", nil
	case languagemodel.ToolChoiceNone:
		return "none", nil
	case languagemodel.ToolChoiceRequired:
		return "required", nil
	case languagemodel.ToolChoiceTool:
		resolvedToolName := choice.ToolName
		if toolNameMapping != nil {
			resolvedToolName = toolNameMapping.ToProviderToolName(resolvedToolName)
		}

		// Check for built-in tool types
		builtinTypes := map[string]bool{
			"code_interpreter":  true,
			"file_search":      true,
			"image_generation":  true,
			"web_search_preview": true,
			"web_search":        true,
			"mcp":               true,
			"apply_patch":       true,
		}

		if builtinTypes[resolvedToolName] {
			return map[string]any{"type": resolvedToolName}, nil
		}

		if _, ok := customProviderToolNames[resolvedToolName]; ok {
			return map[string]any{"type": "custom", "name": resolvedToolName}, nil
		}

		return map[string]any{"type": "function", "name": resolvedToolName}, nil
	}

	return nil, nil
}

// camelToSnake converts a camelCase string to snake_case.
// This is a simple version that handles the common cases.
func camelToSnake(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+'a'-'A'))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}

// mapShellEnvironment converts shell environment arguments from camelCase to snake_case format.
func mapShellEnvironment(env map[string]any) map[string]any {
	envType, _ := env["type"].(string)

	switch envType {
	case "containerReference":
		return map[string]any{
			"type":         "container_reference",
			"container_id": env["containerId"],
		}

	case "containerAuto":
		result := map[string]any{
			"type": "container_auto",
		}
		if v, ok := env["fileIds"]; ok {
			result["file_ids"] = v
		}
		if v, ok := env["memoryLimit"]; ok {
			result["memory_limit"] = v
		}
		if np, ok := env["networkPolicy"].(map[string]any); ok {
			npType, _ := np["type"].(string)
			if npType == "disabled" {
				result["network_policy"] = map[string]any{"type": "disabled"}
			} else {
				policy := map[string]any{
					"type":            "allowlist",
					"allowed_domains": np["allowedDomains"],
				}
				if ds, ok := np["domainSecrets"]; ok {
					policy["domain_secrets"] = ds
				}
				result["network_policy"] = policy
			}
		}
		if skills, ok := env["skills"].([]any); ok {
			result["skills"] = mapShellSkills(skills)
		}
		return result

	default:
		// "local" or unspecified
		result := map[string]any{
			"type": "local",
		}
		if skills, ok := env["skills"]; ok {
			result["skills"] = skills
		}
		return result
	}
}

// mapShellSkills converts shell skills from camelCase to snake_case format.
func mapShellSkills(skills []any) []any {
	if skills == nil {
		return nil
	}
	var result []any
	for _, s := range skills {
		skill, ok := s.(map[string]any)
		if !ok {
			continue
		}
		skillType, _ := skill["type"].(string)
		if skillType == "skillReference" {
			entry := map[string]any{
				"type":     "skill_reference",
				"skill_id": skill["skillId"],
			}
			if v, ok := skill["version"]; ok {
				entry["version"] = v
			}
			result = append(result, entry)
		} else {
			entry := map[string]any{
				"type":        "inline",
				"name":        skill["name"],
				"description": skill["description"],
			}
			if src, ok := skill["source"].(map[string]any); ok {
				entry["source"] = map[string]any{
					"type":       "base64",
					"media_type": src["mediaType"],
					"data":       src["data"],
				}
			}
			result = append(result, entry)
		}
	}
	return result
}
