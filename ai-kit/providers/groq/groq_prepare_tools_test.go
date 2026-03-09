// Ported from: packages/groq/src/groq-prepare-tools.test.ts
package groq

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestPrepareTools_NilTools(t *testing.T) {
	t.Run("should return nil tools and toolChoice when tools are nil", func(t *testing.T) {
		result := PrepareTools(nil, nil, "gemma2-9b-it")

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})
}

func TestPrepareTools_EmptyTools(t *testing.T) {
	t.Run("should return nil tools and toolChoice when tools are empty", func(t *testing.T) {
		result := PrepareTools([]languagemodel.Tool{}, nil, "gemma2-9b-it")

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})
}

func TestPrepareTools_FunctionTools(t *testing.T) {
	t.Run("should correctly prepare function tools", func(t *testing.T) {
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
			"gemma2-9b-it",
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		ft, ok := result.Tools[0].(GroqFunctionTool)
		if !ok {
			t.Fatalf("expected GroqFunctionTool, got %T", result.Tools[0])
		}
		if ft.Type != "function" {
			t.Errorf("expected type 'function', got %q", ft.Type)
		}
		if ft.Function.Name != "testFunction" {
			t.Errorf("expected name 'testFunction', got %q", ft.Function.Name)
		}
		if ft.Function.Description != "A test function" {
			t.Errorf("expected description 'A test function', got %q", ft.Function.Description)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})
}

func TestPrepareTools_UnsupportedProviderTool(t *testing.T) {
	t.Run("should add warnings for unsupported provider-defined tools", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "some.unsupported_tool",
					Name: "unsupported_tool",
					Args: map[string]any{},
				},
			},
			nil,
			"gemma2-9b-it",
		)

		if len(result.Tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result.Tools))
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if w.Feature != "provider-defined tool some.unsupported_tool" {
			t.Errorf("expected feature 'provider-defined tool some.unsupported_tool', got %q", w.Feature)
		}
	})
}

func TestPrepareTools_ToolChoiceAuto(t *testing.T) {
	t.Run("should handle tool choice auto", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceAuto{},
			"gemma2-9b-it",
		)
		if result.ToolChoice != "auto" {
			t.Errorf("expected toolChoice 'auto', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceRequired(t *testing.T) {
	t.Run("should handle tool choice required", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceRequired{},
			"gemma2-9b-it",
		)
		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceNone(t *testing.T) {
	t.Run("should handle tool choice none", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceNone{},
			"gemma2-9b-it",
		)
		if result.ToolChoice != "none" {
			t.Errorf("expected toolChoice 'none', got %v", result.ToolChoice)
		}
	})
}

func TestPrepareTools_ToolChoiceTool(t *testing.T) {
	t.Run("should handle tool choice tool", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "testFunction"},
			"gemma2-9b-it",
		)
		tc, ok := result.ToolChoice.(GroqToolChoiceFunction)
		if !ok {
			t.Fatalf("expected GroqToolChoiceFunction, got %T", result.ToolChoice)
		}
		if tc.Type != "function" {
			t.Errorf("expected type 'function', got %q", tc.Type)
		}
		if tc.Function.Name != "testFunction" {
			t.Errorf("expected function name 'testFunction', got %q", tc.Function.Name)
		}
	})
}

func TestPrepareTools_StrictMode(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(true),
				},
			},
			nil,
			"gemma2-9b-it",
		)

		ft := result.Tools[0].(GroqFunctionTool)
		if ft.Function.Strict == nil || *ft.Function.Strict != true {
			t.Errorf("expected strict true, got %v", ft.Function.Strict)
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      boolPtr(false),
				},
			},
			nil,
			"gemma2-9b-it",
		)

		ft := result.Tools[0].(GroqFunctionTool)
		if ft.Function.Strict == nil || *ft.Function.Strict != false {
			t.Errorf("expected strict false, got %v", ft.Function.Strict)
		}
	})

	t.Run("should not include strict when strict is undefined", func(t *testing.T) {
		desc := "A test function"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
			"gemma2-9b-it",
		)

		ft := result.Tools[0].(GroqFunctionTool)
		if ft.Function.Strict != nil {
			t.Errorf("expected strict nil, got %v", ft.Function.Strict)
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		desc1 := "A strict tool"
		desc2 := "A non-strict tool"
		desc3 := "A tool without strict setting"
		result := PrepareTools(
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
			"gemma2-9b-it",
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}

		ft1 := result.Tools[0].(GroqFunctionTool)
		if ft1.Function.Strict == nil || *ft1.Function.Strict != true {
			t.Errorf("expected strict true for first tool, got %v", ft1.Function.Strict)
		}

		ft2 := result.Tools[1].(GroqFunctionTool)
		if ft2.Function.Strict == nil || *ft2.Function.Strict != false {
			t.Errorf("expected strict false for second tool, got %v", ft2.Function.Strict)
		}

		ft3 := result.Tools[2].(GroqFunctionTool)
		if ft3.Function.Strict != nil {
			t.Errorf("expected strict nil for third tool, got %v", ft3.Function.Strict)
		}
	})
}

func TestPrepareTools_BrowserSearch(t *testing.T) {
	t.Run("should handle browser search tool with supported model", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "groq.browser_search",
					Name: "browser_search",
					Args: map[string]any{},
				},
			},
			nil,
			"openai/gpt-oss-120b",
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		bt, ok := result.Tools[0].(GroqBrowserSearchTool)
		if !ok {
			t.Fatalf("expected GroqBrowserSearchTool, got %T", result.Tools[0])
		}
		if bt.Type != "browser_search" {
			t.Errorf("expected type 'browser_search', got %q", bt.Type)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should warn when browser search is used with unsupported model", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "groq.browser_search",
					Name: "browser_search",
					Args: map[string]any{},
				},
			},
			nil,
			"gemma2-9b-it",
		)

		if len(result.Tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result.Tools))
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if w.Details == nil {
			t.Fatal("expected warning details to be non-nil")
		}
	})

	t.Run("should handle mixed tools with model validation", func(t *testing.T) {
		desc := "A test tool"
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "test-tool",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
				languagemodel.ProviderTool{
					ID:   "groq.browser_search",
					Name: "browser_search",
					Args: map[string]any{},
				},
			},
			nil,
			"openai/gpt-oss-20b",
		)

		if len(result.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0].(GroqFunctionTool); !ok {
			t.Errorf("expected first tool to be GroqFunctionTool, got %T", result.Tools[0])
		}
		if _, ok := result.Tools[1].(GroqBrowserSearchTool); !ok {
			t.Errorf("expected second tool to be GroqBrowserSearchTool, got %T", result.Tools[1])
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should validate all browser search supported models", func(t *testing.T) {
		supportedModels := []string{"openai/gpt-oss-20b", "openai/gpt-oss-120b"}

		for _, modelID := range supportedModels {
			result := PrepareTools(
				[]languagemodel.Tool{
					languagemodel.ProviderTool{
						ID:   "groq.browser_search",
						Name: "browser_search",
						Args: map[string]any{},
					},
				},
				nil,
				modelID,
			)

			if len(result.Tools) != 1 {
				t.Errorf("expected 1 tool for model %q, got %d", modelID, len(result.Tools))
			}
			if len(result.ToolWarnings) != 0 {
				t.Errorf("expected 0 warnings for model %q, got %d", modelID, len(result.ToolWarnings))
			}
		}
	})

	t.Run("should handle browser search with tool choice", func(t *testing.T) {
		result := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "groq.browser_search",
					Name: "browser_search",
					Args: map[string]any{},
				},
			},
			languagemodel.ToolChoiceRequired{},
			"openai/gpt-oss-120b",
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})
}
