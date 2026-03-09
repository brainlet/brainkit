// Ported from: packages/openai-compatible/src/chat/openai-compatible-prepare-tools.test.ts
package openaicompatible

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func boolPtr(v bool) *bool { return &v }

func TestPrepareTools(t *testing.T) {
	t.Run("should return nil tools and nil tool choice when tools is nil", func(t *testing.T) {
		result := PrepareTools(nil, nil)
		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should return nil tools and nil tool choice when tools is empty", func(t *testing.T) {
		result := PrepareTools([]languagemodel.Tool{}, nil)
		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
	})

	t.Run("should convert function tools to OpenAI-compatible format", func(t *testing.T) {
		desc := "A test function"
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "test-tool",
				Description: &desc,
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"value": map[string]any{"type": "string"},
					},
				},
			},
		}

		result := PrepareTools(tools, nil)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		tool := result.Tools[0]
		if tool.Type != "function" {
			t.Errorf("expected type 'function', got %q", tool.Type)
		}
		if tool.Function.Name != "test-tool" {
			t.Errorf("expected name 'test-tool', got %q", tool.Function.Name)
		}
		if tool.Function.Description != "A test function" {
			t.Errorf("expected description 'A test function', got %q", tool.Function.Description)
		}
		if tool.Function.Parameters == nil {
			t.Error("expected non-nil parameters")
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
	})

	t.Run("should add unsupported warning for provider-defined tools", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "provider.tool-1",
				Name: "tool-1",
			},
		}

		result := PrepareTools(tools, nil)

		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		warning, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if warning.Feature != "provider-defined tool provider.tool-1" {
			t.Errorf("expected warning about provider-defined tool, got %q", warning.Feature)
		}
	})

	t.Run("tool choice auto should return 'auto'", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{Name: "test"},
		}

		result := PrepareTools(tools, languagemodel.ToolChoiceAuto{})

		if result.ToolChoice != "auto" {
			t.Errorf("expected tool choice 'auto', got %v", result.ToolChoice)
		}
	})

	t.Run("tool choice required should return 'required'", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{Name: "test"},
		}

		result := PrepareTools(tools, languagemodel.ToolChoiceRequired{})

		if result.ToolChoice != "required" {
			t.Errorf("expected tool choice 'required', got %v", result.ToolChoice)
		}
	})

	t.Run("tool choice none should return 'none'", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{Name: "test"},
		}

		result := PrepareTools(tools, languagemodel.ToolChoiceNone{})

		if result.ToolChoice != "none" {
			t.Errorf("expected tool choice 'none', got %v", result.ToolChoice)
		}
	})

	t.Run("tool choice tool should return function reference", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{Name: "specific-tool"},
		}

		result := PrepareTools(tools, languagemodel.ToolChoiceTool{ToolName: "specific-tool"})

		tc, ok := result.ToolChoice.(OpenAICompatibleToolChoiceFunction)
		if !ok {
			t.Fatalf("expected OpenAICompatibleToolChoiceFunction, got %T", result.ToolChoice)
		}
		if tc.Type != "function" {
			t.Errorf("expected type 'function', got %q", tc.Type)
		}
		if tc.Function.Name != "specific-tool" {
			t.Errorf("expected function name 'specific-tool', got %q", tc.Function.Name)
		}
	})

	t.Run("strict mode true should be set on tool", func(t *testing.T) {
		strict := true
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:   "strict-tool",
				Strict: &strict,
			},
		}

		result := PrepareTools(tools, nil)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Function.Strict == nil {
			t.Fatal("expected strict to be set")
		}
		if *result.Tools[0].Function.Strict != true {
			t.Errorf("expected strict true, got %v", *result.Tools[0].Function.Strict)
		}
	})

	t.Run("strict mode false should be set on tool", func(t *testing.T) {
		strict := false
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:   "non-strict-tool",
				Strict: &strict,
			},
		}

		result := PrepareTools(tools, nil)

		if result.Tools[0].Function.Strict == nil {
			t.Fatal("expected strict to be set")
		}
		if *result.Tools[0].Function.Strict != false {
			t.Errorf("expected strict false, got %v", *result.Tools[0].Function.Strict)
		}
	})

	t.Run("strict mode nil should not set strict on tool", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name: "default-tool",
			},
		}

		result := PrepareTools(tools, nil)

		if result.Tools[0].Function.Strict != nil {
			t.Errorf("expected strict to be nil, got %v", *result.Tools[0].Function.Strict)
		}
	})

	t.Run("should handle multiple tools with different strict settings", func(t *testing.T) {
		strictTrue := true
		strictFalse := false
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:   "tool-strict-true",
				Strict: &strictTrue,
			},
			languagemodel.FunctionTool{
				Name:   "tool-strict-false",
				Strict: &strictFalse,
			},
			languagemodel.FunctionTool{
				Name: "tool-strict-nil",
			},
		}

		result := PrepareTools(tools, nil)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}

		// First tool: strict = true
		if result.Tools[0].Function.Strict == nil || *result.Tools[0].Function.Strict != true {
			t.Errorf("expected first tool strict=true")
		}
		// Second tool: strict = false
		if result.Tools[1].Function.Strict == nil || *result.Tools[1].Function.Strict != false {
			t.Errorf("expected second tool strict=false")
		}
		// Third tool: strict = nil
		if result.Tools[2].Function.Strict != nil {
			t.Errorf("expected third tool strict=nil")
		}
	})
}
