// Ported from: packages/xai/src/responses/xai-responses-prepare-tools.test.ts
package xai

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestPrepareResponsesTools_WebSearch(t *testing.T) {
	t.Run("should prepare web_search tool with no args", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "web_search" {
			t.Errorf("expected type 'web_search', got %v", result.Tools[0]["type"])
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should prepare web_search tool with allowed domains", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"allowedDomains": []interface{}{"wikipedia.org", "example.com"},
					},
				},
			},
			nil,
		)

		domains, ok := result.Tools[0]["allowed_domains"].([]interface{})
		if !ok {
			t.Fatal("expected allowed_domains")
		}
		if len(domains) != 2 {
			t.Errorf("expected 2 domains, got %d", len(domains))
		}
	})

	t.Run("should prepare web_search tool with excluded domains", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"excludedDomains": []interface{}{"spam.com"},
					},
				},
			},
			nil,
		)

		domains, ok := result.Tools[0]["excluded_domains"].([]interface{})
		if !ok {
			t.Fatal("expected excluded_domains")
		}
		if len(domains) != 1 {
			t.Errorf("expected 1 domain, got %d", len(domains))
		}
	})

	t.Run("should prepare web_search tool with image understanding", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"enableImageUnderstanding": true,
					},
				},
			},
			nil,
		)

		if result.Tools[0]["enable_image_understanding"] != true {
			t.Errorf("expected enable_image_understanding true, got %v", result.Tools[0]["enable_image_understanding"])
		}
	})
}

func TestPrepareResponsesTools_XSearch(t *testing.T) {
	t.Run("should prepare x_search tool with no args", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.x_search",
					Name: "x_search",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "x_search" {
			t.Errorf("expected type 'x_search', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should prepare x_search tool with allowed handles", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.x_search",
					Name: "x_search",
					Args: map[string]any{
						"allowedXHandles": []interface{}{"elonmusk", "xai"},
					},
				},
			},
			nil,
		)

		handles, ok := result.Tools[0]["allowed_x_handles"].([]interface{})
		if !ok {
			t.Fatal("expected allowed_x_handles")
		}
		if len(handles) != 2 {
			t.Errorf("expected 2 handles, got %d", len(handles))
		}
	})

	t.Run("should prepare x_search tool with date range", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.x_search",
					Name: "x_search",
					Args: map[string]any{
						"fromDate": "2025-01-01",
						"toDate":   "2025-12-31",
					},
				},
			},
			nil,
		)

		if result.Tools[0]["from_date"] != "2025-01-01" {
			t.Errorf("expected from_date '2025-01-01', got %v", result.Tools[0]["from_date"])
		}
		if result.Tools[0]["to_date"] != "2025-12-31" {
			t.Errorf("expected to_date '2025-12-31', got %v", result.Tools[0]["to_date"])
		}
	})

	t.Run("should prepare x_search tool with video understanding", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.x_search",
					Name: "x_search",
					Args: map[string]any{
						"enableVideoUnderstanding": true,
						"enableImageUnderstanding": true,
					},
				},
			},
			nil,
		)

		if result.Tools[0]["enable_video_understanding"] != true {
			t.Errorf("expected enable_video_understanding true, got %v", result.Tools[0]["enable_video_understanding"])
		}
		if result.Tools[0]["enable_image_understanding"] != true {
			t.Errorf("expected enable_image_understanding true, got %v", result.Tools[0]["enable_image_understanding"])
		}
	})
}

func TestPrepareResponsesTools_CodeExecution(t *testing.T) {
	t.Run("should prepare code_execution tool as code_interpreter", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.code_execution",
					Name: "code_execution",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "code_interpreter" {
			t.Errorf("expected type 'code_interpreter', got %v", result.Tools[0]["type"])
		}
	})
}

func TestPrepareResponsesTools_ViewImage(t *testing.T) {
	t.Run("should prepare view_image tool", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.view_image",
					Name: "view_image",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "view_image" {
			t.Errorf("expected type 'view_image', got %v", result.Tools[0]["type"])
		}
	})
}

func TestPrepareResponsesTools_ViewXVideo(t *testing.T) {
	t.Run("should prepare view_x_video tool", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.view_x_video",
					Name: "view_x_video",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "view_x_video" {
			t.Errorf("expected type 'view_x_video', got %v", result.Tools[0]["type"])
		}
	})
}

func TestPrepareResponsesTools_FileSearch(t *testing.T) {
	t.Run("should prepare file_search tool with vector store IDs", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]any{
						"vectorStoreIds": []interface{}{"collection_1", "collection_2"},
					},
				},
			},
			nil,
		)

		if result.Tools[0]["type"] != "file_search" {
			t.Errorf("expected type 'file_search', got %v", result.Tools[0]["type"])
		}
		ids, ok := result.Tools[0]["vector_store_ids"].([]interface{})
		if !ok {
			t.Fatal("expected vector_store_ids")
		}
		if len(ids) != 2 {
			t.Errorf("expected 2 store IDs, got %d", len(ids))
		}
	})

	t.Run("should prepare file_search tool with max num results", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]any{
						"vectorStoreIds": []interface{}{"collection_1"},
						"maxNumResults":  float64(10),
					},
				},
			},
			nil,
		)

		if result.Tools[0]["max_num_results"] != float64(10) {
			t.Errorf("expected max_num_results 10, got %v", result.Tools[0]["max_num_results"])
		}
	})

	t.Run("should handle multiple tools including file_search", func(t *testing.T) {
		desc := "calculate numbers"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]any{
						"vectorStoreIds": []interface{}{"collection_1"},
					},
				},
				languagemodel.FunctionTool{
					Name:        "calculator",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "web_search" {
			t.Errorf("expected first tool 'web_search', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "file_search" {
			t.Errorf("expected second tool 'file_search', got %v", result.Tools[1]["type"])
		}
		if result.Tools[2]["type"] != "function" {
			t.Errorf("expected third tool 'function', got %v", result.Tools[2]["type"])
		}
	})
}

func TestPrepareResponsesTools_FunctionTools(t *testing.T) {
	t.Run("should prepare function tools", func(t *testing.T) {
		desc := "get weather information"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: &desc,
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
						"required": []interface{}{"location"},
					},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["name"] != "weather" {
			t.Errorf("expected name 'weather', got %v", result.Tools[0]["name"])
		}
		if result.Tools[0]["description"] != "get weather information" {
			t.Errorf("expected description, got %v", result.Tools[0]["description"])
		}
	})
}

func TestPrepareResponsesTools_StrictMode(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		desc := "A test function"
		result := prepareResponsesTools(
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

		if s, ok := result.Tools[0]["strict"].(bool); !ok || !s {
			t.Errorf("expected strict true, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		desc := "A test function"
		result := prepareResponsesTools(
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

		if s, ok := result.Tools[0]["strict"].(bool); !ok || s {
			t.Errorf("expected strict false, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should not include strict when strict is undefined", func(t *testing.T) {
		desc := "A test function"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if _, exists := result.Tools[0]["strict"]; exists {
			t.Errorf("expected strict to not exist, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should pass through strict mode for multiple tools with different settings", func(t *testing.T) {
		desc1 := "A strict tool"
		desc2 := "A non-strict tool"
		desc3 := "A tool without strict setting"
		result := prepareResponsesTools(
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

		if s, ok := result.Tools[0]["strict"].(bool); !ok || !s {
			t.Errorf("expected strict true for first tool, got %v", result.Tools[0]["strict"])
		}
		if s, ok := result.Tools[1]["strict"].(bool); !ok || s {
			t.Errorf("expected strict false for second tool, got %v", result.Tools[1]["strict"])
		}
		if _, exists := result.Tools[2]["strict"]; exists {
			t.Errorf("expected no strict for third tool, got %v", result.Tools[2]["strict"])
		}
	})
}

func TestPrepareResponsesTools_ToolChoice(t *testing.T) {
	t.Run("should handle tool choice auto", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
			languagemodel.ToolChoiceAuto{},
		)

		if result.ToolChoice != "auto" {
			t.Errorf("expected toolChoice 'auto', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice required", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
			languagemodel.ToolChoiceRequired{},
		)

		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice none", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
			languagemodel.ToolChoiceNone{},
		)

		if result.ToolChoice != "none" {
			t.Errorf("expected toolChoice 'none', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle specific tool choice", func(t *testing.T) {
		desc := "get weather"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "weather"},
		)

		tc, ok := result.ToolChoice.(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", result.ToolChoice)
		}
		if tc["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tc["type"])
		}
		if tc["name"] != "weather" {
			t.Errorf("expected name 'weather', got %v", tc["name"])
		}
	})

	t.Run("should warn when trying to force server-side tool via toolChoice", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "web_search"},
		)

		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}

		var found bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok {
				if uw.Feature == `toolChoice for server-side tool "web_search"` {
					found = true
				}
			}
		}
		if !found {
			t.Error("expected unsupported warning for server-side tool choice")
		}
	})
}

func TestPrepareResponsesTools_MultipleTools(t *testing.T) {
	t.Run("should handle multiple tools including provider-defined and functions", func(t *testing.T) {
		desc := "calculate numbers"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "calculator",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
				languagemodel.ProviderTool{
					ID:   "xai.x_search",
					Name: "x_search",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected first tool 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "web_search" {
			t.Errorf("expected second tool 'web_search', got %v", result.Tools[1]["type"])
		}
		if result.Tools[2]["type"] != "x_search" {
			t.Errorf("expected third tool 'x_search', got %v", result.Tools[2]["type"])
		}
	})
}

func TestPrepareResponsesTools_EmptyTools(t *testing.T) {
	t.Run("should return nil for empty tools array", func(t *testing.T) {
		result := prepareResponsesTools([]languagemodel.Tool{}, nil)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
	})

	t.Run("should return nil for nil tools", func(t *testing.T) {
		result := prepareResponsesTools(nil, nil)

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
	})
}

func TestPrepareResponsesTools_UnsupportedTools(t *testing.T) {
	t.Run("should warn about unsupported provider-defined tools", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "unsupported.tool",
					Name: "unsupported",
					Args: map[string]any{},
				},
			},
			nil,
		)

		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		w, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if w.Feature != "provider-defined tool unsupported" {
			t.Errorf("expected feature 'provider-defined tool unsupported', got %q", w.Feature)
		}
	})
}

func TestPrepareResponsesTools_MCP(t *testing.T) {
	t.Run("should prepare mcp tool with required args only", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverUrl":   "https://example.com/mcp",
						"serverLabel": "test-server",
					},
				},
			},
			nil,
		)

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "mcp" {
			t.Errorf("expected type 'mcp', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["server_url"] != "https://example.com/mcp" {
			t.Errorf("expected server_url, got %v", result.Tools[0]["server_url"])
		}
		if result.Tools[0]["server_label"] != "test-server" {
			t.Errorf("expected server_label 'test-server', got %v", result.Tools[0]["server_label"])
		}
	})

	t.Run("should prepare mcp tool with all optional args", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverUrl":         "https://example.com/mcp",
						"serverLabel":       "test-server",
						"serverDescription": "A test MCP server",
						"allowedTools":      []interface{}{"tool1", "tool2"},
						"headers":           map[string]interface{}{"X-Custom": "value"},
						"authorization":     "Bearer token123",
					},
				},
			},
			nil,
		)

		tool := result.Tools[0]
		if tool["server_description"] != "A test MCP server" {
			t.Errorf("expected server_description, got %v", tool["server_description"])
		}
		allowedTools, ok := tool["allowed_tools"].([]interface{})
		if !ok || len(allowedTools) != 2 {
			t.Errorf("expected 2 allowed_tools, got %v", tool["allowed_tools"])
		}
		if tool["authorization"] != "Bearer token123" {
			t.Errorf("expected authorization, got %v", tool["authorization"])
		}
		headers, ok := tool["headers"].(map[string]interface{})
		if !ok {
			t.Fatal("expected headers map")
		}
		if headers["X-Custom"] != "value" {
			t.Errorf("expected X-Custom header, got %v", headers["X-Custom"])
		}
	})

	t.Run("should warn when trying to force mcp tool via toolChoice", func(t *testing.T) {
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverUrl":   "https://example.com/mcp",
						"serverLabel": "test-server",
					},
				},
			},
			languagemodel.ToolChoiceTool{ToolName: "mcp"},
		)

		if result.ToolChoice != nil {
			t.Errorf("expected nil toolChoice, got %v", result.ToolChoice)
		}
		var found bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok {
				if uw.Feature == `toolChoice for server-side tool "mcp"` {
					found = true
				}
			}
		}
		if !found {
			t.Error("expected unsupported warning for mcp tool choice")
		}
	})

	t.Run("should handle multiple tools including mcp", func(t *testing.T) {
		desc := "calculate numbers"
		result := prepareResponsesTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
				languagemodel.ProviderTool{
					ID:   "xai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverUrl":   "https://example.com/mcp",
						"serverLabel": "test-server",
					},
				},
				languagemodel.FunctionTool{
					Name:        "calculator",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
		)

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "web_search" {
			t.Errorf("expected first tool 'web_search', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "mcp" {
			t.Errorf("expected second tool 'mcp', got %v", result.Tools[1]["type"])
		}
		if result.Tools[2]["type"] != "function" {
			t.Errorf("expected third tool 'function', got %v", result.Tools[2]["type"])
		}
	})
}
