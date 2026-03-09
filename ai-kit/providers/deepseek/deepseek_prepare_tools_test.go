// Ported from: packages/deepseek/src/chat/deepseek-prepare-tools.test.ts
package deepseek

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestPrepareTools(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		strictTrue := true
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictTrue,
				},
			},
			nil, // toolChoice
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
		if tool.Function.Description != "A test function" {
			t.Errorf("expected description 'A test function', got %q", tool.Function.Description)
		}
		if tool.Function.Strict == nil || *tool.Function.Strict != true {
			t.Errorf("expected strict true, got %v", tool.Function.Strict)
		}

		// Check parameters
		params, ok := tool.Function.Parameters.(map[string]any)
		if !ok {
			t.Fatalf("expected parameters to be map[string]any, got %T", tool.Function.Parameters)
		}
		if params["type"] != "object" {
			t.Errorf("expected parameters type 'object', got %v", params["type"])
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		strictFalse := false
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictFalse,
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
		if tool.Function.Strict == nil || *tool.Function.Strict != false {
			t.Errorf("expected strict false, got %v", tool.Function.Strict)
		}
	})

	t.Run("should not include strict mode when strict is undefined", func(t *testing.T) {
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					// Strict is nil (undefined)
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
		if tool.Function.Strict != nil {
			t.Errorf("expected strict to be nil (omitted), got %v", *tool.Function.Strict)
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		strictTrue := true
		strictFalse := false
		descStrict := "A strict tool"
		descNonStrict := "A non-strict tool"
		descDefault := "A tool without strict setting"

		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "strictTool",
					Description: &descStrict,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictTrue,
				},
				languagemodel.FunctionTool{
					Name:        "nonStrictTool",
					Description: &descNonStrict,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictFalse,
				},
				languagemodel.FunctionTool{
					Name:        "defaultTool",
					Description: &descDefault,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					// Strict is nil
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

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}

		// First tool: strict = true
		if result.Tools[0].Function.Name != "strictTool" {
			t.Errorf("expected name 'strictTool', got %q", result.Tools[0].Function.Name)
		}
		if result.Tools[0].Function.Description != "A strict tool" {
			t.Errorf("expected description 'A strict tool', got %q", result.Tools[0].Function.Description)
		}
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != true {
			t.Errorf("expected strict true for strictTool")
		}

		// Second tool: strict = false
		if result.Tools[1].Function.Name != "nonStrictTool" {
			t.Errorf("expected name 'nonStrictTool', got %q", result.Tools[1].Function.Name)
		}
		if result.Tools[1].Function.Description != "A non-strict tool" {
			t.Errorf("expected description 'A non-strict tool', got %q", result.Tools[1].Function.Description)
		}
		if result.Tools[1].Function.Strict == nil || *result.Tools[1].Function.Strict != false {
			t.Errorf("expected strict false for nonStrictTool")
		}

		// Third tool: strict = nil (omitted)
		if result.Tools[2].Function.Name != "defaultTool" {
			t.Errorf("expected name 'defaultTool', got %q", result.Tools[2].Function.Name)
		}
		if result.Tools[2].Function.Description != "A tool without strict setting" {
			t.Errorf("expected description 'A tool without strict setting', got %q", result.Tools[2].Function.Description)
		}
		if result.Tools[2].Function.Strict != nil {
			t.Errorf("expected strict nil for defaultTool, got %v", *result.Tools[2].Function.Strict)
		}
	})
}
