// Ported from: packages/core/src/stream/aisdk/v5/compat/prepare-tools.ts
package compat

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// ModelSpecVersion
// ---------------------------------------------------------------------------

// ModelSpecVersion represents the target model specification version.
type ModelSpecVersion string

const (
	// ModelSpecVersionV2 is AI SDK v5 (uses 'provider-defined' tool type).
	ModelSpecVersionV2 ModelSpecVersion = "v2"
	// ModelSpecVersionV3 is AI SDK v6 (uses 'provider' tool type).
	ModelSpecVersionV3 ModelSpecVersion = "v3"
)

// ---------------------------------------------------------------------------
// PreparedTool
// ---------------------------------------------------------------------------

// PreparedTool represents a tool prepared for the language model API.
// It can be either a function tool or a provider-defined tool.
type PreparedTool struct {
	Type            string         `json:"type"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	InputSchema     map[string]any `json:"inputSchema,omitempty"`
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
	// Provider tool fields
	ID   string         `json:"id,omitempty"`
	Args map[string]any `json:"args,omitempty"`
}

// ---------------------------------------------------------------------------
// PreparedToolChoice
// ---------------------------------------------------------------------------

// PreparedToolChoice represents the tool choice directive for the model.
type PreparedToolChoice struct {
	Type     string `json:"type"`
	ToolName string `json:"toolName,omitempty"`
}

// ---------------------------------------------------------------------------
// isProviderTool
// ---------------------------------------------------------------------------

// isProviderTool checks if a tool is a provider-defined tool from the AI SDK.
// Provider tools (like openai.tools.webSearch()) are created by the AI SDK with:
//   - type: "provider-defined" (AI SDK v5) or "provider" (AI SDK v6)
//   - id: in format '<provider>.<tool_name>' (e.g., 'openai.web_search')
func isProviderTool(tool any) (id string, args map[string]any, ok bool) {
	m, isMap := tool.(map[string]any)
	if !isMap {
		return "", nil, false
	}

	// Provider tools have type: "provider-defined" (v5) or "provider" (v6)
	t, _ := m["type"].(string)
	if t != "provider-defined" && t != "provider" {
		return "", nil, false
	}

	idStr, hasID := m["id"].(string)
	if !hasID {
		return "", nil, false
	}

	argsMap, _ := m["args"].(map[string]any)
	return idStr, argsMap, true
}

// getProviderToolName extracts the tool name from a provider tool id.
// e.g., 'openai.web_search' -> 'web_search'
func getProviderToolName(providerID string) string {
	parts := strings.SplitN(providerID, ".", 2)
	if len(parts) < 2 {
		return providerID
	}
	return strings.Join(parts[1:], ".")
}

// ---------------------------------------------------------------------------
// fixTypelessProperties
// ---------------------------------------------------------------------------

// fixTypelessProperties recursively fixes JSON Schema properties that lack a 'type' key.
// Zod v4's toJSONSchema serializes z.any() to just { description: "..." } with no 'type',
// which providers like OpenAI reject. This converts such schemas to a permissive type union.
func fixTypelessProperties(schema map[string]any) map[string]any {
	if schema == nil {
		return schema
	}

	result := make(map[string]any, len(schema))
	for k, v := range schema {
		result[k] = v
	}

	// Fix properties
	if props, ok := result["properties"].(map[string]any); ok {
		fixedProps := make(map[string]any, len(props))
		for key, value := range props {
			propSchema, isMap := value.(map[string]any)
			if !isMap {
				fixedProps[key] = value
				continue
			}

			_, hasType := propSchema["type"]
			_, hasRef := propSchema["$ref"]
			_, hasAnyOf := propSchema["anyOf"]
			_, hasOneOf := propSchema["oneOf"]
			_, hasAllOf := propSchema["allOf"]

			if !hasType && !hasRef && !hasAnyOf && !hasOneOf && !hasAllOf {
				// Create a permissive type union, excluding 'array'
				fixed := make(map[string]any, len(propSchema))
				for k, v := range propSchema {
					if k == "items" {
						continue // Exclude items (only valid when type is ARRAY)
					}
					fixed[k] = v
				}
				fixed["type"] = []string{"string", "number", "integer", "boolean", "object", "null"}
				fixedProps[key] = fixed
			} else {
				// Recurse into nested object schemas
				fixedProps[key] = fixTypelessProperties(propSchema)
			}
		}
		result["properties"] = fixedProps
	}

	// Fix items (arrays)
	if items, ok := result["items"]; ok {
		switch itemVal := items.(type) {
		case []any:
			fixedItems := make([]any, len(itemVal))
			for i, item := range itemVal {
				if itemMap, isMap := item.(map[string]any); isMap {
					fixedItems[i] = fixTypelessProperties(itemMap)
				} else {
					fixedItems[i] = item
				}
			}
			result["items"] = fixedItems
		case map[string]any:
			result["items"] = fixTypelessProperties(itemVal)
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// PrepareToolsAndToolChoice
// ---------------------------------------------------------------------------

// PrepareToolsAndToolChoiceParams configures tool preparation.
type PrepareToolsAndToolChoiceParams struct {
	// Tools is the tool set keyed by tool name.
	Tools map[string]any
	// ToolChoice is the tool choice directive (string or struct with ToolName).
	ToolChoice any
	// ActiveTools restricts which tools are active. nil means all.
	ActiveTools []string
	// TargetVersion is the target model version. Defaults to V2.
	TargetVersion ModelSpecVersion
}

// PrepareToolsAndToolChoiceResult holds the prepared tools and tool choice.
type PrepareToolsAndToolChoiceResult struct {
	Tools      []PreparedTool
	ToolChoice *PreparedToolChoice
}

// PrepareToolsAndToolChoice processes raw tool definitions and tool choice directives
// into the format expected by the language model API.
//
// It handles:
//   - Provider tools (type: "provider-defined" or "provider")
//   - Function tools (standard tools with inputSchema/parameters)
//   - Tool filtering via activeTools
//   - Tool choice normalization (string, struct, nil)
//   - Schema fixing for typeless properties (Zod v4 compatibility)
//   - Version-specific tool type conversion (v2 vs v3)
func PrepareToolsAndToolChoice(params PrepareToolsAndToolChoiceParams) PrepareToolsAndToolChoiceResult {
	targetVersion := params.TargetVersion
	if targetVersion == "" {
		targetVersion = ModelSpecVersionV2
	}

	if len(params.Tools) == 0 {
		// Preserve explicit 'none' toolChoice to tell the LLM not to attempt tool calls
		if tc, ok := params.ToolChoice.(string); ok && tc == "none" {
			return PrepareToolsAndToolChoiceResult{
				ToolChoice: &PreparedToolChoice{Type: "none"},
			}
		}
		return PrepareToolsAndToolChoiceResult{}
	}

	// Build active tools set for filtering
	var activeToolSet map[string]bool
	if params.ActiveTools != nil {
		activeToolSet = make(map[string]bool, len(params.ActiveTools))
		for _, name := range params.ActiveTools {
			activeToolSet[name] = true
		}
	}

	// Provider tool type differs between versions:
	// - V2 (AI SDK v5): 'provider-defined'
	// - V3 (AI SDK v6): 'provider'
	providerToolType := "provider-defined"
	if targetVersion == ModelSpecVersionV3 {
		providerToolType = "provider"
	}

	var preparedTools []PreparedTool
	for name, tool := range params.Tools {
		// Filter by activeTools if specified
		if activeToolSet != nil && !activeToolSet[name] {
			continue
		}

		prepared, err := prepareSingleTool(name, tool, providerToolType)
		if err != nil {
			fmt.Printf("Error preparing tool %s: %v\n", name, err)
			continue
		}
		if prepared != nil {
			preparedTools = append(preparedTools, *prepared)
		}
	}

	// Normalize tool choice
	var toolChoice *PreparedToolChoice
	switch tc := params.ToolChoice.(type) {
	case nil:
		toolChoice = &PreparedToolChoice{Type: "auto"}
	case string:
		toolChoice = &PreparedToolChoice{Type: tc}
	case map[string]any:
		if tn, ok := tc["toolName"].(string); ok {
			toolChoice = &PreparedToolChoice{Type: "tool", ToolName: tn}
		} else {
			toolChoice = &PreparedToolChoice{Type: "auto"}
		}
	default:
		toolChoice = &PreparedToolChoice{Type: "auto"}
	}

	return PrepareToolsAndToolChoiceResult{
		Tools:      preparedTools,
		ToolChoice: toolChoice,
	}
}

// prepareSingleTool processes a single tool definition into a PreparedTool.
func prepareSingleTool(name string, tool any, providerToolType string) (*PreparedTool, error) {
	// Check if this is a provider tool
	if id, args, ok := isProviderTool(tool); ok {
		if args == nil {
			args = make(map[string]any)
		}
		return &PreparedTool{
			Type: providerToolType,
			Name: getProviderToolName(id),
			ID:   id,
			Args: args,
		}, nil
	}

	// Handle as a function tool
	toolMap, isMap := tool.(map[string]any)
	if !isMap {
		return nil, fmt.Errorf("tool %s is not a map", name)
	}

	// Extract input schema (try inputSchema first, then parameters)
	var inputSchema map[string]any
	if is, ok := toolMap["inputSchema"].(map[string]any); ok {
		inputSchema = is
	} else if ps, ok := toolMap["parameters"].(map[string]any); ok {
		inputSchema = ps
	}

	// Fix typeless properties in schema
	if inputSchema != nil {
		inputSchema = fixTypelessProperties(inputSchema)
	}

	// Extract description
	description, _ := toolMap["description"].(string)

	// Extract provider options
	providerOptions, _ := toolMap["providerOptions"].(map[string]any)

	// Check tool type
	toolType, _ := toolMap["type"].(string)
	switch toolType {
	case "", "dynamic", "function":
		return &PreparedTool{
			Type:            "function",
			Name:            name,
			Description:     description,
			InputSchema:     inputSchema,
			ProviderOptions: providerOptions,
		}, nil
	case "provider-defined":
		// Fallback for tools that are provider-defined
		id, _ := toolMap["id"].(string)
		args, _ := toolMap["args"].(map[string]any)
		toolName := name
		if id != "" {
			toolName = getProviderToolName(id)
		}
		return &PreparedTool{
			Type: providerToolType,
			Name: toolName,
			ID:   id,
			Args: args,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported tool type: %s", toolType)
	}
}
