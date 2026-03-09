// Ported from: packages/xai/src/xai-prepare-tools.test.ts
package xai

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func boolPtr(v bool) *bool {
	return &v
}

func testStrPtr(v string) *string {
	return &v
}

func TestPrepareTools_NilTools(t *testing.T) {
	t.Run("should return nil tools and toolChoice when tools are nil", func(t *testing.T) {
		result := prepareTools(nil, nil)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestPrepareTools_EmptyTools(t *testing.T) {
	t.Run("should return nil tools and toolChoice when tools are empty", func(t *testing.T) {
		result := prepareTools([]languagemodel.Tool{}, nil)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestPrepareTools_FunctionTools(t *testing.T) {
	t.Run("should correctly prepare function tools", func(t *testing.T) {
		desc := "A test function"
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		ft := result.Tools[0].(map[string]interface{})
		if ft["type"] != "function" {
			t.Errorf("expected type 'function', got %v", ft["type"])
		}
		fn := ft["function"].(map[string]interface{})
		if fn["name"] != "testFunction" {
			t.Errorf("expected name 'testFunction', got %v", fn["name"])
		}
		if fn["description"] != "A test function" {
			t.Errorf("expected description 'A test function', got %v", fn["description"])
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestPrepareTools_UnsupportedProviderTool(t *testing.T) {
	t.Run("should add warnings for unsupported provider-defined tools", func(t *testing.T) {
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "some.unsupported_tool",
					Name: "unsupported_tool",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result.Tools))
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		w, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if w.Feature != "provider-defined tool unsupported_tool" {
			t.Errorf("expected feature 'provider-defined tool unsupported_tool', got %q", w.Feature)
		}
	})
}

func TestPrepareTools_ToolChoiceAuto(t *testing.T) {
	t.Run("should handle tool choice auto", func(t *testing.T) {
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceAuto{},
		)
		if result.ToolChoice != "auto" {
			t.Errorf("expected toolChoice 'auto', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceRequired(t *testing.T) {
	t.Run("should handle tool choice required", func(t *testing.T) {
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceRequired{},
		)
		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceNone(t *testing.T) {
	t.Run("should handle tool choice none", func(t *testing.T) {
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceNone{},
		)
		if result.ToolChoice != "none" {
			t.Errorf("expected toolChoice 'none', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceTool(t *testing.T) {
	t.Run("should handle tool choice tool", func(t *testing.T) {
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "testFunction"},
		)
		tc, ok := result.ToolChoice.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", result.ToolChoice)
		}
		if tc["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tc["type"])
		}
		fn := tc["function"].(map[string]interface{})
		if fn["name"] != "testFunction" {
			t.Errorf("expected function name 'testFunction', got %v", fn["name"])
		}
	})
}

func TestPrepareTools_StrictMode(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		desc := "A test function"
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(true),
				},
			},
			nil,
		)

		ft := result.Tools[0].(map[string]interface{})
		fn := ft["function"].(map[string]interface{})
		strict, ok := fn["strict"].(bool)
		if !ok || !strict {
			t.Errorf("expected strict true, got %v", fn["strict"])
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		desc := "A test function"
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(false),
				},
			},
			nil,
		)

		ft := result.Tools[0].(map[string]interface{})
		fn := ft["function"].(map[string]interface{})
		strict, ok := fn["strict"].(bool)
		if !ok || strict {
			t.Errorf("expected strict false, got %v", fn["strict"])
		}
	})

	t.Run("should not include strict when strict is undefined", func(t *testing.T) {
		desc := "A test function"
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		ft := result.Tools[0].(map[string]interface{})
		fn := ft["function"].(map[string]interface{})
		if _, exists := fn["strict"]; exists {
			t.Errorf("expected strict to not exist, got %v", fn["strict"])
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		desc1 := "A strict tool"
		desc2 := "A non-strict tool"
		desc3 := "A tool without strict setting"
		result := prepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "strictTool",
					Description: &desc1,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(true),
				},
				languagemodel.FunctionTool{
					Name:        "nonStrictTool",
					Description: &desc2,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(false),
				},
				languagemodel.FunctionTool{
					Name:        "defaultTool",
					Description: &desc3,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}

		ft1 := result.Tools[0].(map[string]interface{})
		fn1 := ft1["function"].(map[string]interface{})
		if s, ok := fn1["strict"].(bool); !ok || !s {
			t.Errorf("expected strict true for first tool, got %v", fn1["strict"])
		}

		ft2 := result.Tools[1].(map[string]interface{})
		fn2 := ft2["function"].(map[string]interface{})
		if s, ok := fn2["strict"].(bool); !ok || s {
			t.Errorf("expected strict false for second tool, got %v", fn2["strict"])
		}

		ft3 := result.Tools[2].(map[string]interface{})
		fn3 := ft3["function"].(map[string]interface{})
		if _, exists := fn3["strict"]; exists {
			t.Errorf("expected strict nil for third tool, got %v", fn3["strict"])
		}
	})
}
