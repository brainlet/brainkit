// Ported from: packages/mistral/src/mistral-prepare-tools.test.ts
package mistral

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestPrepareTools(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		strictVal := true
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: strPtr("A test function"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
			nil,
		)

		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 tool warnings, got %d", len(result.ToolWarnings))
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}

		tool := result.Tools[0]
		if tool.Type != "function" {
			t.Errorf("expected type 'function', got %q", tool.Type)
		}
		if tool.Function.Name != "testFunction" {
			t.Errorf("expected name 'testFunction', got %q", tool.Function.Name)
		}
		if tool.Function.Description == nil || *tool.Function.Description != "A test function" {
			t.Errorf("expected description 'A test function', got %v", tool.Function.Description)
		}
		if tool.Function.Strict == nil || *tool.Function.Strict != true {
			t.Errorf("expected strict true, got %v", tool.Function.Strict)
		}

		// Verify parameters
		paramsJSON, _ := json.Marshal(tool.Function.Parameters)
		expectedParams := `{"properties":{},"type":"object"}`
		if string(paramsJSON) != expectedParams {
			t.Errorf("expected parameters %s, got %s", expectedParams, string(paramsJSON))
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		strictVal := false
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: strPtr("A test function"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}

		tool := result.Tools[0]
		if tool.Function.Strict == nil || *tool.Function.Strict != false {
			t.Errorf("expected strict false, got %v", tool.Function.Strict)
		}
	})

	t.Run("should not include strict mode when strict is undefined", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: strPtr("A test function"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}

		tool := result.Tools[0]
		if tool.Function.Strict != nil {
			t.Errorf("expected strict nil, got %v", tool.Function.Strict)
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		strictTrue := true
		strictFalse := false
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "strictTool",
					Description: strPtr("A strict tool"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictTrue,
				},
				languagemodel.FunctionTool{
					Name:        "nonStrictTool",
					Description: strPtr("A non-strict tool"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictFalse,
				},
				languagemodel.FunctionTool{
					Name:        "defaultTool",
					Description: strPtr("A tool without strict setting"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}

		// First tool: strict true
		if result.Tools[0].Function.Name != "strictTool" {
			t.Errorf("expected name 'strictTool', got %q", result.Tools[0].Function.Name)
		}
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != true {
			t.Errorf("expected strict true for strictTool, got %v", result.Tools[0].Function.Strict)
		}

		// Second tool: strict false
		if result.Tools[1].Function.Name != "nonStrictTool" {
			t.Errorf("expected name 'nonStrictTool', got %q", result.Tools[1].Function.Name)
		}
		if result.Tools[1].Function.Strict == nil || *result.Tools[1].Function.Strict != false {
			t.Errorf("expected strict false for nonStrictTool, got %v", result.Tools[1].Function.Strict)
		}

		// Third tool: strict nil
		if result.Tools[2].Function.Name != "defaultTool" {
			t.Errorf("expected name 'defaultTool', got %q", result.Tools[2].Function.Name)
		}
		if result.Tools[2].Function.Strict != nil {
			t.Errorf("expected strict nil for defaultTool, got %v", result.Tools[2].Function.Strict)
		}
	})
}
