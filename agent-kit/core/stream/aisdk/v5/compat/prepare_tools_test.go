// Ported from: packages/core/src/stream/aisdk/v5/compat/prepare-tools.test.ts
package compat

import (
	"testing"
)

func TestIsProviderTool(t *testing.T) {
	t.Run("should detect provider-defined tool", func(t *testing.T) {
		tool := map[string]any{
			"type": "provider-defined",
			"id":   "openai.web_search",
			"args": map[string]any{"query": "test"},
		}
		id, args, ok := isProviderTool(tool)
		if !ok {
			t.Fatal("expected tool to be identified as provider tool")
		}
		if id != "openai.web_search" {
			t.Errorf("expected id 'openai.web_search', got %q", id)
		}
		if args["query"] != "test" {
			t.Errorf("expected args query 'test', got %v", args["query"])
		}
	})

	t.Run("should detect provider tool", func(t *testing.T) {
		tool := map[string]any{
			"type": "provider",
			"id":   "anthropic.thinking",
		}
		id, _, ok := isProviderTool(tool)
		if !ok {
			t.Fatal("expected tool to be identified as provider tool")
		}
		if id != "anthropic.thinking" {
			t.Errorf("expected id 'anthropic.thinking', got %q", id)
		}
	})

	t.Run("should return false for function tool", func(t *testing.T) {
		tool := map[string]any{
			"type":        "function",
			"description": "A function tool",
		}
		_, _, ok := isProviderTool(tool)
		if ok {
			t.Error("expected function tool not to be identified as provider tool")
		}
	})

	t.Run("should return false for non-map input", func(t *testing.T) {
		_, _, ok := isProviderTool("not a map")
		if ok {
			t.Error("expected non-map to return false")
		}
	})

	t.Run("should return false if no id field", func(t *testing.T) {
		tool := map[string]any{
			"type": "provider-defined",
		}
		_, _, ok := isProviderTool(tool)
		if ok {
			t.Error("expected tool without id to return false")
		}
	})
}

func TestGetProviderToolName(t *testing.T) {
	t.Run("should extract tool name from provider id", func(t *testing.T) {
		result := getProviderToolName("openai.web_search")
		if result != "web_search" {
			t.Errorf("expected 'web_search', got %q", result)
		}
	})

	t.Run("should handle id with multiple dots", func(t *testing.T) {
		result := getProviderToolName("provider.nested.tool")
		if result != "nested.tool" {
			t.Errorf("expected 'nested.tool', got %q", result)
		}
	})

	t.Run("should return full id if no dot", func(t *testing.T) {
		result := getProviderToolName("simple")
		if result != "simple" {
			t.Errorf("expected 'simple', got %q", result)
		}
	})
}

func TestFixTypelessProperties(t *testing.T) {
	t.Run("should return nil for nil schema", func(t *testing.T) {
		result := fixTypelessProperties(nil)
		if result != nil {
			t.Error("expected nil")
		}
	})

	t.Run("should add type union for typeless properties", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"description": "A name field",
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		nameProp := props["name"].(map[string]any)
		typeVal, ok := nameProp["type"]
		if !ok {
			t.Fatal("expected type field to be added")
		}
		types, ok := typeVal.([]string)
		if !ok {
			t.Fatal("expected type to be a string slice")
		}
		if len(types) != 6 {
			t.Errorf("expected 6 types in union, got %d", len(types))
		}
	})

	t.Run("should not modify properties that have type", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type": "string",
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		nameProp := props["name"].(map[string]any)
		if nameProp["type"] != "string" {
			t.Errorf("expected type 'string', got %v", nameProp["type"])
		}
	})

	t.Run("should not modify properties that have $ref", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"ref": map[string]any{
					"$ref": "#/definitions/Foo",
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		refProp := props["ref"].(map[string]any)
		if _, hasType := refProp["type"]; hasType {
			t.Error("should not have added type to $ref property")
		}
	})

	t.Run("should not modify properties that have anyOf", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"union": map[string]any{
					"anyOf": []any{
						map[string]any{"type": "string"},
						map[string]any{"type": "number"},
					},
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		unionProp := props["union"].(map[string]any)
		if _, hasType := unionProp["type"]; hasType {
			t.Error("should not have added type to anyOf property")
		}
	})

	t.Run("should remove items key from typeless properties", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"field": map[string]any{
					"description": "typeless with items",
					"items":       map[string]any{"type": "string"},
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		fieldProp := props["field"].(map[string]any)
		if _, hasItems := fieldProp["items"]; hasItems {
			t.Error("expected items to be removed from typeless property")
		}
		if _, hasType := fieldProp["type"]; !hasType {
			t.Error("expected type union to be added")
		}
	})

	t.Run("should recurse into nested object schemas", func(t *testing.T) {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"inner": map[string]any{
							"description": "typeless nested",
						},
					},
				},
			},
		}
		result := fixTypelessProperties(schema)
		props := result["properties"].(map[string]any)
		nested := props["nested"].(map[string]any)
		innerProps := nested["properties"].(map[string]any)
		inner := innerProps["inner"].(map[string]any)
		if _, hasType := inner["type"]; !hasType {
			t.Error("expected type to be added to nested typeless property")
		}
	})

	t.Run("should fix items in arrays", func(t *testing.T) {
		schema := map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"description": "typeless in array item",
					},
				},
			},
		}
		result := fixTypelessProperties(schema)
		items := result["items"].(map[string]any)
		itemProps := items["properties"].(map[string]any)
		field := itemProps["field"].(map[string]any)
		if _, hasType := field["type"]; !hasType {
			t.Error("expected type to be added to typeless property in array items")
		}
	})
}

func TestPrepareToolsAndToolChoice(t *testing.T) {
	t.Run("should return empty result for no tools", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{})
		if len(result.Tools) != 0 {
			t.Errorf("expected no tools, got %d", len(result.Tools))
		}
		if result.ToolChoice != nil {
			t.Error("expected nil tool choice")
		}
	})

	t.Run("should preserve none toolChoice with no tools", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			ToolChoice: "none",
		})
		if result.ToolChoice == nil {
			t.Fatal("expected tool choice")
		}
		if result.ToolChoice.Type != "none" {
			t.Errorf("expected type 'none', got %q", result.ToolChoice.Type)
		}
	})

	t.Run("should prepare function tools", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"search": map[string]any{
					"description": "Search the web",
					"inputSchema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"query": map[string]any{"type": "string"},
						},
					},
				},
			},
		})
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Type != "function" {
			t.Errorf("expected type 'function', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "search" {
			t.Errorf("expected name 'search', got %q", result.Tools[0].Name)
		}
		if result.Tools[0].Description != "Search the web" {
			t.Errorf("expected description 'Search the web', got %q", result.Tools[0].Description)
		}
	})

	t.Run("should prepare provider-defined tools", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"web_search": map[string]any{
					"type": "provider-defined",
					"id":   "openai.web_search",
					"args": map[string]any{"search_context_size": "medium"},
				},
			},
		})
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Type != "provider-defined" {
			t.Errorf("expected type 'provider-defined', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "web_search" {
			t.Errorf("expected name 'web_search', got %q", result.Tools[0].Name)
		}
		if result.Tools[0].ID != "openai.web_search" {
			t.Errorf("expected id 'openai.web_search', got %q", result.Tools[0].ID)
		}
	})

	t.Run("should use provider type for v3 target version", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"web_search": map[string]any{
					"type": "provider-defined",
					"id":   "openai.web_search",
				},
			},
			TargetVersion: ModelSpecVersionV3,
		})
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Type != "provider" {
			t.Errorf("expected type 'provider', got %q", result.Tools[0].Type)
		}
	})

	t.Run("should filter tools by activeTools", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"search": map[string]any{"description": "Search"},
				"calc":   map[string]any{"description": "Calculate"},
			},
			ActiveTools: []string{"search"},
		})
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Name != "search" {
			t.Errorf("expected name 'search', got %q", result.Tools[0].Name)
		}
	})

	t.Run("should default tool choice to auto", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"search": map[string]any{"description": "Search"},
			},
		})
		if result.ToolChoice == nil {
			t.Fatal("expected tool choice")
		}
		if result.ToolChoice.Type != "auto" {
			t.Errorf("expected type 'auto', got %q", result.ToolChoice.Type)
		}
	})

	t.Run("should handle string tool choice", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"search": map[string]any{"description": "Search"},
			},
			ToolChoice: "required",
		})
		if result.ToolChoice.Type != "required" {
			t.Errorf("expected type 'required', got %q", result.ToolChoice.Type)
		}
	})

	t.Run("should handle tool choice with toolName", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"search": map[string]any{"description": "Search"},
			},
			ToolChoice: map[string]any{"toolName": "search"},
		})
		if result.ToolChoice.Type != "tool" {
			t.Errorf("expected type 'tool', got %q", result.ToolChoice.Type)
		}
		if result.ToolChoice.ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", result.ToolChoice.ToolName)
		}
	})

	t.Run("should use parameters as fallback for inputSchema", func(t *testing.T) {
		result := PrepareToolsAndToolChoice(PrepareToolsAndToolChoiceParams{
			Tools: map[string]any{
				"calc": map[string]any{
					"description": "Calculate",
					"parameters": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"x": map[string]any{"type": "number"},
						},
					},
				},
			},
		})
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].InputSchema == nil {
			t.Fatal("expected inputSchema to be populated from parameters")
		}
	})
}
