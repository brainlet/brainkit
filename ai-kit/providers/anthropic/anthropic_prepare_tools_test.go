// Ported from: packages/anthropic/src/anthropic-prepare-tools.test.ts
package anthropic

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestPrepareTools_NilTools(t *testing.T) {
	t.Run("should return nil tools and tool_choice when tools are nil", func(t *testing.T) {
		result := prepareTools(nil, nil, nil, nil, true)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected no tool warnings, got %v", result.ToolWarnings)
		}
		if len(result.Betas) != 0 {
			t.Errorf("expected no betas, got %v", result.Betas)
		}
	})
}

func TestPrepareTools_EmptyTools(t *testing.T) {
	t.Run("should return nil tools and tool_choice when tools are empty", func(t *testing.T) {
		result := prepareTools([]languagemodel.Tool{}, nil, nil, nil, true)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected no tool warnings, got %v", result.ToolWarnings)
		}
		if len(result.Betas) != 0 {
			t.Errorf("expected no betas, got %v", result.Betas)
		}
	})
}

func TestPrepareTools_FunctionTools(t *testing.T) {
	t.Run("should correctly prepare function tools", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{"eagerInputStreaming": true},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Name != "testFunction" {
			t.Errorf("expected tool name 'testFunction', got %q", result.Tools[0].Name)
		}
		if *result.Tools[0].Description != "A test function" {
			t.Errorf("expected tool description 'A test function', got %q", *result.Tools[0].Description)
		}
		if result.Tools[0].EagerInputStreaming == nil || !*result.Tools[0].EagerInputStreaming {
			t.Error("expected eager_input_streaming to be true")
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected no tool warnings, got %v", result.ToolWarnings)
		}
	})

	t.Run("should correctly preserve tool input examples", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "tool_with_examples",
				Description: strPtr("tool with examples"),
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"a": map[string]any{"type": "number"}},
				},
				InputExamples: []languagemodel.FunctionToolInputExample{
					{Input: map[string]any{"a": 1}},
					{Input: map[string]any{"a": 2}},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if len(result.Tools[0].InputExamples) != 2 {
			t.Fatalf("expected 2 input examples, got %d", len(result.Tools[0].InputExamples))
		}
		if result.Betas["structured-outputs-2025-11-13"] != true {
			t.Error("expected structured-outputs-2025-11-13 beta")
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})
}

func TestPrepareTools_StrictMode(t *testing.T) {
	t.Run("should include strict when supportsStructuredOutput is true and strict is true", func(t *testing.T) {
		strict := true
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				Strict:      &strict,
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Strict == nil || !*result.Tools[0].Strict {
			t.Error("expected strict to be true")
		}
		if result.Betas["structured-outputs-2025-11-13"] != true {
			t.Error("expected structured-outputs-2025-11-13 beta")
		}
	})

	t.Run("should not include strict when supportsStructuredOutput is false", func(t *testing.T) {
		strict := true
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				Strict:      &strict,
			},
		}

		result := prepareTools(tools, nil, nil, nil, false)

		if result.Tools[0].Strict != nil {
			t.Errorf("expected strict to be nil, got %v", *result.Tools[0].Strict)
		}
		if result.Betas["structured-outputs-2025-11-13"] {
			t.Error("should not have structured-outputs-2025-11-13 beta")
		}
	})

	t.Run("should include beta when strict is false and supportsStructuredOutput is true", func(t *testing.T) {
		strict := false
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				Strict:      &strict,
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Strict == nil || *result.Tools[0].Strict != false {
			t.Error("expected strict to be false")
		}
		if result.Betas["structured-outputs-2025-11-13"] != true {
			t.Error("expected structured-outputs-2025-11-13 beta")
		}
	})
}

func TestPrepareTools_ProviderDefinedTools(t *testing.T) {
	t.Run("computer_20241022", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.computer_20241022",
				Name: "computer",
				Args: map[string]any{
					"displayWidthPx":  800,
					"displayHeightPx": 600,
					"displayNumber":   1,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0].Type != "computer_20241022" {
			t.Errorf("expected type 'computer_20241022', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "computer" {
			t.Errorf("expected name 'computer', got %q", result.Tools[0].Name)
		}
		if result.Tools[0].DisplayWidthPx == nil || *result.Tools[0].DisplayWidthPx != 800 {
			t.Errorf("expected displayWidthPx 800")
		}
		if result.Tools[0].DisplayHeightPx == nil || *result.Tools[0].DisplayHeightPx != 600 {
			t.Errorf("expected displayHeightPx 600")
		}
		if result.Tools[0].DisplayNumber == nil || *result.Tools[0].DisplayNumber != 1 {
			t.Errorf("expected displayNumber 1")
		}
		if result.Betas["computer-use-2024-10-22"] != true {
			t.Error("expected computer-use-2024-10-22 beta")
		}
	})

	t.Run("computer_20250124", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.computer_20250124",
				Name: "computer",
				Args: map[string]any{
					"displayWidthPx":  1024,
					"displayHeightPx": 768,
					"displayNumber":   1,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "computer_20250124" {
			t.Errorf("expected type 'computer_20250124', got %q", result.Tools[0].Type)
		}
		if result.Betas["computer-use-2025-01-24"] != true {
			t.Error("expected computer-use-2025-01-24 beta")
		}
	})

	t.Run("computer_20251124", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.computer_20251124",
				Name: "computer",
				Args: map[string]any{
					"displayWidthPx":  1024,
					"displayHeightPx": 768,
					"displayNumber":   1,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "computer_20251124" {
			t.Errorf("expected type 'computer_20251124', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].EnableZoom != nil {
			t.Errorf("expected enableZoom to be nil, got %v", result.Tools[0].EnableZoom)
		}
		if result.Betas["computer-use-2025-11-24"] != true {
			t.Error("expected computer-use-2025-11-24 beta")
		}
	})

	t.Run("computer_20251124 with enableZoom true", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.computer_20251124",
				Name: "computer",
				Args: map[string]any{
					"displayWidthPx":  1024,
					"displayHeightPx": 768,
					"displayNumber":   1,
					"enableZoom":      true,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].EnableZoom == nil || !*result.Tools[0].EnableZoom {
			t.Error("expected enableZoom to be true")
		}
	})

	t.Run("computer_20251124 with enableZoom false", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.computer_20251124",
				Name: "computer",
				Args: map[string]any{
					"displayWidthPx":  1024,
					"displayHeightPx": 768,
					"displayNumber":   1,
					"enableZoom":      false,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].EnableZoom == nil || *result.Tools[0].EnableZoom {
			t.Error("expected enableZoom to be false")
		}
	})

	t.Run("text_editor_20241022", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.text_editor_20241022",
				Name: "text_editor",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "text_editor_20241022" {
			t.Errorf("expected type 'text_editor_20241022', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "str_replace_editor" {
			t.Errorf("expected name 'str_replace_editor', got %q", result.Tools[0].Name)
		}
		if result.Betas["computer-use-2024-10-22"] != true {
			t.Error("expected computer-use-2024-10-22 beta")
		}
	})

	t.Run("bash_20241022", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.bash_20241022",
				Name: "bash",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "bash_20241022" {
			t.Errorf("expected type 'bash_20241022', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "bash" {
			t.Errorf("expected name 'bash', got %q", result.Tools[0].Name)
		}
		if result.Betas["computer-use-2024-10-22"] != true {
			t.Error("expected computer-use-2024-10-22 beta")
		}
	})

	t.Run("text_editor_20250728 with max_characters", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.text_editor_20250728",
				Name: "str_replace_based_edit_tool",
				Args: map[string]any{"maxCharacters": 10000},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "text_editor_20250728" {
			t.Errorf("expected type 'text_editor_20250728', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].MaxCharacters == nil || *result.Tools[0].MaxCharacters != 10000 {
			t.Error("expected maxCharacters to be 10000")
		}
	})

	t.Run("text_editor_20250728 without max_characters", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.text_editor_20250728",
				Name: "str_replace_based_edit_tool",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].MaxCharacters != nil {
			t.Errorf("expected maxCharacters to be nil, got %v", *result.Tools[0].MaxCharacters)
		}
	})

	t.Run("web_search_20250305", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.web_search_20250305",
				Name: "web_search",
				Args: map[string]any{
					"maxUses":        10,
					"allowedDomains": []any{"https://www.google.com"},
					"userLocation":   map[string]any{"type": "approximate", "city": "New York"},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "web_search_20250305" {
			t.Errorf("expected type 'web_search_20250305', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].MaxUses == nil || *result.Tools[0].MaxUses != 10 {
			t.Error("expected maxUses to be 10")
		}
		if len(result.Tools[0].AllowedDomains) != 1 || result.Tools[0].AllowedDomains[0] != "https://www.google.com" {
			t.Errorf("expected allowedDomains [https://www.google.com], got %v", result.Tools[0].AllowedDomains)
		}
		if result.Tools[0].UserLocation == nil {
			t.Error("expected userLocation to be non-nil")
		}
	})

	t.Run("web_search_20260209", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.web_search_20260209",
				Name: "web_search",
				Args: map[string]any{
					"maxUses":        10,
					"allowedDomains": []any{"https://www.google.com"},
					"userLocation":   map[string]any{"type": "approximate", "city": "New York"},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "web_search_20260209" {
			t.Errorf("expected type 'web_search_20260209', got %q", result.Tools[0].Type)
		}
		if result.Betas["code-execution-web-tools-2026-02-09"] != true {
			t.Error("expected code-execution-web-tools-2026-02-09 beta")
		}
	})

	t.Run("web_fetch_20250910", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.web_fetch_20250910",
				Name: "web_fetch",
				Args: map[string]any{
					"maxUses":          10,
					"allowedDomains":   []any{"https://www.google.com"},
					"citations":        map[string]any{"enabled": true},
					"maxContentTokens": 1000,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "web_fetch_20250910" {
			t.Errorf("expected type 'web_fetch_20250910', got %q", result.Tools[0].Type)
		}
		if result.Betas["web-fetch-2025-09-10"] != true {
			t.Error("expected web-fetch-2025-09-10 beta")
		}
		if result.Tools[0].MaxContentTokens == nil || *result.Tools[0].MaxContentTokens != 1000 {
			t.Error("expected maxContentTokens to be 1000")
		}
	})

	t.Run("web_fetch_20260209", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.web_fetch_20260209",
				Name: "web_fetch",
				Args: map[string]any{
					"maxUses":          10,
					"allowedDomains":   []any{"https://www.google.com"},
					"citations":        map[string]any{"enabled": true},
					"maxContentTokens": 1000,
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "web_fetch_20260209" {
			t.Errorf("expected type 'web_fetch_20260209', got %q", result.Tools[0].Type)
		}
		if result.Betas["code-execution-web-tools-2026-02-09"] != true {
			t.Error("expected code-execution-web-tools-2026-02-09 beta")
		}
	})

	t.Run("tool_search_regex_20251119", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.tool_search_regex_20251119",
				Name: "tool_search",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "tool_search_tool_regex_20251119" {
			t.Errorf("expected type 'tool_search_tool_regex_20251119', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "tool_search_tool_regex" {
			t.Errorf("expected name 'tool_search_tool_regex', got %q", result.Tools[0].Name)
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})

	t.Run("code_execution_20260120 without beta header", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.code_execution_20260120",
				Name: "code_execution",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "code_execution_20260120" {
			t.Errorf("expected type 'code_execution_20260120', got %q", result.Tools[0].Type)
		}
		if len(result.Betas) != 0 {
			t.Errorf("expected no betas, got %v", result.Betas)
		}
	})

	t.Run("tool_search_bm25_20251119", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "anthropic.tool_search_bm25_20251119",
				Name: "tool_search",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].Type != "tool_search_tool_bm25_20251119" {
			t.Errorf("expected type 'tool_search_tool_bm25_20251119', got %q", result.Tools[0].Type)
		}
		if result.Tools[0].Name != "tool_search_tool_bm25" {
			t.Errorf("expected name 'tool_search_tool_bm25', got %q", result.Tools[0].Name)
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})
}

func TestPrepareTools_DeferLoading(t *testing.T) {
	t.Run("should include defer_loading when set to true", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{"deferLoading": true},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].DeferLoading == nil || !*result.Tools[0].DeferLoading {
			t.Error("expected defer_loading to be true")
		}
	})

	t.Run("should include defer_loading when set to false", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{"deferLoading": false},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].DeferLoading == nil || *result.Tools[0].DeferLoading {
			t.Error("expected defer_loading to be false")
		}
	})

	t.Run("should not include defer_loading when not specified", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].DeferLoading != nil {
			t.Errorf("expected defer_loading to be nil, got %v", *result.Tools[0].DeferLoading)
		}
	})
}

func TestPrepareTools_AllowedCallers(t *testing.T) {
	t.Run("should include allowed_callers and advanced-tool-use beta when allowedCallers is set", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "query_database",
				Description: strPtr("Query a database"),
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"sql": map[string]any{"type": "string"}},
				},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{
						"allowedCallers": []any{"code_execution_20250825"},
					},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools[0].AllowedCallers) != 1 || result.Tools[0].AllowedCallers[0] != "code_execution_20250825" {
			t.Errorf("expected allowedCallers ['code_execution_20250825'], got %v", result.Tools[0].AllowedCallers)
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})

	t.Run("should not include allowed_callers when not specified", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("A test function"),
				InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools[0].AllowedCallers) != 0 {
			t.Errorf("expected no allowed_callers, got %v", result.Tools[0].AllowedCallers)
		}
	})

	t.Run("should include both deferLoading and allowedCallers when both are set", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "query_database",
				Description: strPtr("Query a database"),
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"sql": map[string]any{"type": "string"}},
				},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{
						"deferLoading":   true,
						"allowedCallers": []any{"code_execution_20250825"},
					},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].DeferLoading == nil || !*result.Tools[0].DeferLoading {
			t.Error("expected defer_loading to be true")
		}
		if len(result.Tools[0].AllowedCallers) != 1 || result.Tools[0].AllowedCallers[0] != "code_execution_20250825" {
			t.Errorf("expected allowedCallers ['code_execution_20250825'], got %v", result.Tools[0].AllowedCallers)
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})

	t.Run("should include allowed_callers with code_execution_20260120", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "query_database",
				Description: strPtr("Query a database"),
				InputSchema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"sql": map[string]any{"type": "string"}},
				},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{
						"allowedCallers": []any{"code_execution_20260120"},
					},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools[0].AllowedCallers) != 1 || result.Tools[0].AllowedCallers[0] != "code_execution_20260120" {
			t.Errorf("expected allowedCallers ['code_execution_20260120'], got %v", result.Tools[0].AllowedCallers)
		}
		if result.Betas["structured-outputs-2025-11-13"] != true {
			t.Error("expected structured-outputs-2025-11-13 beta")
		}
		if result.Betas["advanced-tool-use-2025-11-20"] != true {
			t.Error("expected advanced-tool-use-2025-11-20 beta")
		}
	})
}

func TestPrepareTools_UnsupportedTools(t *testing.T) {
	t.Run("should add warnings for unsupported tools", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.ProviderTool{
				ID:   "unsupported.tool",
				Name: "unsupported_tool",
				Args: map[string]any{},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if len(result.Tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result.Tools))
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		warning, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if warning.Feature != "provider-defined tool unsupported.tool" {
			t.Errorf("expected feature 'provider-defined tool unsupported.tool', got %q", warning.Feature)
		}
	})
}

func TestPrepareTools_ToolChoice(t *testing.T) {
	t.Run("should handle tool choice auto", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("Test"),
				InputSchema: map[string]any{},
			},
		}

		result := prepareTools(tools, languagemodel.ToolChoiceAuto{}, nil, nil, true)

		if result.ToolChoice == nil {
			t.Fatal("expected non-nil tool choice")
		}
		if result.ToolChoice.Type != "auto" {
			t.Errorf("expected type 'auto', got %q", result.ToolChoice.Type)
		}
	})

	t.Run("should handle tool choice required", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("Test"),
				InputSchema: map[string]any{},
			},
		}

		result := prepareTools(tools, languagemodel.ToolChoiceRequired{}, nil, nil, true)

		if result.ToolChoice == nil {
			t.Fatal("expected non-nil tool choice")
		}
		if result.ToolChoice.Type != "any" {
			t.Errorf("expected type 'any', got %q", result.ToolChoice.Type)
		}
	})

	t.Run("should handle tool choice none", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("Test"),
				InputSchema: map[string]any{},
			},
		}

		result := prepareTools(tools, languagemodel.ToolChoiceNone{}, nil, nil, true)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil tool choice, got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice tool", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("Test"),
				InputSchema: map[string]any{},
			},
		}

		result := prepareTools(tools, languagemodel.ToolChoiceTool{ToolName: "testFunction"}, nil, nil, true)

		if result.ToolChoice == nil {
			t.Fatal("expected non-nil tool choice")
		}
		if result.ToolChoice.Type != "tool" {
			t.Errorf("expected type 'tool', got %q", result.ToolChoice.Type)
		}
		if result.ToolChoice.Name != "testFunction" {
			t.Errorf("expected name 'testFunction', got %q", result.ToolChoice.Name)
		}
	})
}

func TestPrepareTools_CacheControl(t *testing.T) {
	t.Run("should set cache control", func(t *testing.T) {
		tools := []languagemodel.Tool{
			languagemodel.FunctionTool{
				Name:        "testFunction",
				Description: strPtr("Test"),
				InputSchema: map[string]any{},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{
						"cacheControl": map[string]any{"type": "ephemeral"},
					},
				},
			},
		}

		result := prepareTools(tools, nil, nil, nil, true)

		if result.Tools[0].CacheControl == nil {
			t.Fatal("expected cache_control to be non-nil")
		}
		if result.Tools[0].CacheControl.Type != "ephemeral" {
			t.Errorf("expected cache_control type 'ephemeral', got %q", result.Tools[0].CacheControl.Type)
		}
	})

	t.Run("should limit cache breakpoints to 4", func(t *testing.T) {
		validator := NewCacheControlValidator()
		tools := make([]languagemodel.Tool, 5)
		for i := 0; i < 5; i++ {
			desc := "Test"
			if i == 4 {
				desc = "Test 5 (should be rejected)"
			}
			tools[i] = languagemodel.FunctionTool{
				Name:        "tool" + string(rune('1'+i)),
				Description: strPtr(desc),
				InputSchema: map[string]any{},
				ProviderOptions: shared.ProviderOptions{
					"anthropic": map[string]any{
						"cacheControl": map[string]any{"type": "ephemeral"},
					},
				},
			}
		}

		result := prepareTools(tools, nil, nil, validator, true)

		// First 4 should have cache_control
		for i := 0; i < 4; i++ {
			if result.Tools[i].CacheControl == nil {
				t.Errorf("expected tool %d to have cache_control", i)
			}
		}
		// 5th should be rejected
		if result.Tools[4].CacheControl != nil {
			t.Error("expected 5th tool to have nil cache_control (limit exceeded)")
		}

		warnings := validator.GetWarnings()
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		w, ok := warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", warnings[0])
		}
		if w.Feature != "cacheControl breakpoint limit" {
			t.Errorf("expected feature 'cacheControl breakpoint limit', got %q", w.Feature)
		}
	})
}
