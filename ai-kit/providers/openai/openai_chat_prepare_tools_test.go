// Ported from: packages/openai/src/chat/openai-chat-prepare-tools.test.ts
package openai

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestPrepareChatTools(t *testing.T) {
	t.Run("should return nil tools and toolChoice when tools are nil", func(t *testing.T) {
		result := PrepareChatTools(nil, nil)

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

	t.Run("should return nil tools and toolChoice when tools are empty", func(t *testing.T) {
		result := PrepareChatTools([]languagemodel.Tool{}, nil)

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

	t.Run("should correctly prepare function tools", func(t *testing.T) {
		desc := "A test function"
		result := PrepareChatTools(
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
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should add warnings for unsupported tools", func(t *testing.T) {
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.unsupported_tool",
					Name: "unsupported_tool",
					Args: map[string]any{},
				},
			},
			nil,
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
		if w.Feature == "" {
			t.Error("expected non-empty feature in warning")
		}
	})

	t.Run("should handle tool choice auto", func(t *testing.T) {
		desc := "Test"
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceAuto{},
		)

		if result.ToolChoice != "auto" {
			t.Errorf("expected toolChoice 'auto', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice required", func(t *testing.T) {
		desc := "Test"
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceRequired{},
		)

		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice none", func(t *testing.T) {
		desc := "Test"
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceNone{},
		)

		if result.ToolChoice != "none" {
			t.Errorf("expected toolChoice 'none', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice tool", func(t *testing.T) {
		desc := "Test"
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "testFunction"},
		)

		tc, ok := result.ToolChoice.(OpenAIChatToolChoiceFunction)
		if !ok {
			t.Fatalf("expected OpenAIChatToolChoiceFunction, got %T", result.ToolChoice)
		}
		if tc.Type != "function" {
			t.Errorf("expected type 'function', got %q", tc.Type)
		}
		if tc.Function.Name != "testFunction" {
			t.Errorf("expected function name 'testFunction', got %q", tc.Function.Name)
		}
	})

	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		desc := "A test function"
		strictVal := true
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != true {
			t.Error("expected strict to be true")
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		desc := "A test function"
		strictVal := false
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != false {
			t.Error("expected strict to be false")
		}
	})

	t.Run("should not include strict mode when strict is nil", func(t *testing.T) {
		desc := "A test function"
		result := PrepareChatTools(
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
		if result.Tools[0].Function.Strict != nil {
			t.Errorf("expected strict to be nil, got %v", *result.Tools[0].Function.Strict)
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		strictTrue := true
		strictFalse := false
		desc1 := "A strict tool"
		desc2 := "A non-strict tool"
		desc3 := "A tool without strict setting"
		result := PrepareChatTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "strictTool",
					Description: &desc1,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictTrue,
				},
				languagemodel.FunctionTool{
					Name:        "nonStrictTool",
					Description: &desc2,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictFalse,
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
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != true {
			t.Error("expected first tool strict=true")
		}
		if result.Tools[1].Function.Strict == nil || *result.Tools[1].Function.Strict != false {
			t.Error("expected second tool strict=false")
		}
		if result.Tools[2].Function.Strict != nil {
			t.Error("expected third tool strict=nil")
		}
	})
}
