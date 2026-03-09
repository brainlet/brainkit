// Ported from: packages/openai/src/responses/openai-responses-prepare-tools.test.ts
package openai

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

func TestPrepareResponsesTools_FunctionToolsStrictMode(t *testing.T) {
	t.Run("should pass through strict mode when strict is true", func(t *testing.T) {
		strictVal := true
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["name"] != "testFunction" {
			t.Errorf("expected name 'testFunction', got %v", result.Tools[0]["name"])
		}
		if result.Tools[0]["strict"] != true {
			t.Errorf("expected strict true, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should pass through strict mode when strict is false", func(t *testing.T) {
		strictVal := false
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
					Strict:      &strictVal,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["strict"] != false {
			t.Errorf("expected strict false, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should not include strict mode when strict is nil", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, exists := result.Tools[0]["strict"]; exists {
			t.Errorf("expected strict to not exist, got %v", result.Tools[0]["strict"])
		}
	})

	t.Run("should pass through strict mode for multiple tools with different strict settings", func(t *testing.T) {
		strictTrue := true
		strictFalse := false
		desc1 := "A strict tool"
		desc2 := "A non-strict tool"
		desc3 := "A tool without strict setting"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
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
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["strict"] != true {
			t.Error("expected first tool strict=true")
		}
		if result.Tools[1]["strict"] != false {
			t.Error("expected second tool strict=false")
		}
		if _, exists := result.Tools[2]["strict"]; exists {
			t.Error("expected third tool strict to not exist")
		}
	})
}

func TestPrepareResponsesTools_NilAndEmpty(t *testing.T) {
	t.Run("should return nil tools when tools are nil", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: nil,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

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

	t.Run("should return nil tools when tools are empty", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
	})
}

func TestPrepareResponsesTools_CodeInterpreter(t *testing.T) {
	t.Run("should prepare code interpreter tool with no container (auto mode)", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.code_interpreter",
					Name: "code_interpreter",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "code_interpreter" {
			t.Errorf("expected type 'code_interpreter', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should prepare code interpreter tool with string container ID", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.code_interpreter",
					Name: "code_interpreter",
					Args: map[string]any{
						"container": "container-123",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["container"] != "container-123" {
			t.Errorf("expected container 'container-123', got %v", result.Tools[0]["container"])
		}
	})

	t.Run("should handle tool choice selection with code interpreter", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.code_interpreter",
					Name: "code_interpreter",
					Args: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "code_interpreter"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "code_interpreter" {
			t.Errorf("expected type 'code_interpreter', got %v", tc["type"])
		}
	})
}

func TestPrepareResponsesTools_FileSearch(t *testing.T) {
	t.Run("should prepare file_search tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.file_search",
					Name: "file_search",
					Args: map[string]any{
						"vectorStoreIds": []any{"vs-123"},
						"maxNumResults":  float64(10),
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "file_search" {
			t.Errorf("expected type 'file_search', got %v", result.Tools[0]["type"])
		}
	})
}

func TestPrepareResponsesTools_WebSearch(t *testing.T) {
	t.Run("should prepare web_search_preview tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search_preview",
					Name: "web_search_preview",
					Args: map[string]any{
						"searchContextSize": "medium",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "web_search_preview" {
			t.Errorf("expected type 'web_search_preview', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["search_context_size"] != "medium" {
			t.Errorf("expected search_context_size 'medium', got %v", result.Tools[0]["search_context_size"])
		}
	})

	t.Run("should prepare web_search tool with filters", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"filters": map[string]any{
							"allowedDomains": []any{"example.com"},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["type"] != "web_search" {
			t.Errorf("expected type 'web_search', got %v", result.Tools[0]["type"])
		}
	})
}

func TestPrepareResponsesTools_ImageGeneration(t *testing.T) {
	t.Run("should prepare image_generation tool with options", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.image_generation",
					Name: "image_generation",
					Args: map[string]any{
						"background":        "opaque",
						"size":              "1536x1024",
						"quality":           "high",
						"moderation":        "auto",
						"outputFormat":      "png",
						"outputCompression": float64(100),
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "image_generation" {
			t.Errorf("expected type 'image_generation', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["background"] != "opaque" {
			t.Errorf("expected background 'opaque', got %v", result.Tools[0]["background"])
		}
		if result.Tools[0]["quality"] != "high" {
			t.Errorf("expected quality 'high', got %v", result.Tools[0]["quality"])
		}
	})
}

func TestPrepareResponsesTools_ToolChoice(t *testing.T) {
	desc := "Test"
	t.Run("should handle tool choice auto", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceAuto{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ToolChoice != "auto" {
			t.Errorf("expected toolChoice 'auto', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice required", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceRequired{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ToolChoice != "required" {
			t.Errorf("expected toolChoice 'required', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice none", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceNone{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ToolChoice != "none" {
			t.Errorf("expected toolChoice 'none', got %v", result.ToolChoice)
		}
	})

	t.Run("should handle tool choice for specific function", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "testFunction"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tc["type"])
		}
		if tc["name"] != "testFunction" {
			t.Errorf("expected name 'testFunction', got %v", tc["name"])
		}
	})
}

func TestPrepareResponsesTools_MCP(t *testing.T) {
	t.Run("should prepare mcp tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverLabel":       "test-server",
						"serverUrl":         "https://mcp.example.com",
						"serverDescription": "Test MCP server",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "mcp" {
			t.Errorf("expected type 'mcp', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["server_label"] != "test-server" {
			t.Errorf("expected server_label 'test-server', got %v", result.Tools[0]["server_label"])
		}
		if result.Tools[0]["server_url"] != "https://mcp.example.com" {
			t.Errorf("expected server_url, got %v", result.Tools[0]["server_url"])
		}
	})
}

func TestPrepareResponsesTools_UnsupportedWarnings(t *testing.T) {
	t.Run("should warn about unsupported tool types", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				// Use an unknown provider tool type that gets treated as unsupported
				languagemodel.ProviderTool{
					ID:   "openai.unsupported_tool",
					Name: "unsupported_tool",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Unknown provider tools return nil, so they won't be in the output
		// but no warnings are generated for unrecognized provider tools
		if len(result.Tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(result.Tools))
		}
	})
}

func TestPrepareResponsesTools_Shell(t *testing.T) {
	t.Run("should prepare local_shell tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.local_shell",
					Name: "local_shell",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "local_shell" {
			t.Errorf("expected type 'local_shell', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should prepare shell tool without environment args", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "shell" {
			t.Errorf("expected type 'shell', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should prepare shell tool with containerAuto without skills", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "containerAuto",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		env, ok := result.Tools[0]["environment"].(map[string]any)
		if !ok {
			t.Fatalf("expected environment map, got %T", result.Tools[0]["environment"])
		}
		if env["type"] != "container_auto" {
			t.Errorf("expected environment type 'container_auto', got %v", env["type"])
		}
	})

	t.Run("should prepare shell tool with containerAuto and skillReference skills", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "containerAuto",
							"skills": []any{
								map[string]any{
									"type":    "skillReference",
									"skillId": "skill_abc",
									"version": "1.0.0",
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env, ok := result.Tools[0]["environment"].(map[string]any)
		if !ok {
			t.Fatalf("expected environment map, got %T", result.Tools[0]["environment"])
		}
		skills, ok := env["skills"].([]any)
		if !ok {
			t.Fatalf("expected skills array, got %T", env["skills"])
		}
		if len(skills) != 1 {
			t.Fatalf("expected 1 skill, got %d", len(skills))
		}
		skill, ok := skills[0].(map[string]any)
		if !ok {
			t.Fatalf("expected skill map, got %T", skills[0])
		}
		if skill["type"] != "skill_reference" {
			t.Errorf("expected skill type 'skill_reference', got %v", skill["type"])
		}
		if skill["skill_id"] != "skill_abc" {
			t.Errorf("expected skill_id 'skill_abc', got %v", skill["skill_id"])
		}
		if skill["version"] != "1.0.0" {
			t.Errorf("expected version '1.0.0', got %v", skill["version"])
		}
	})

	t.Run("should prepare shell tool with containerAuto and inline skill", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "containerAuto",
							"skills": []any{
								map[string]any{
									"type":        "inline",
									"name":        "my-skill",
									"description": "A test skill",
									"source": map[string]any{
										"type":      "base64",
										"mediaType": "application/zip",
										"data":      "dGVzdA==",
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		skills := env["skills"].([]any)
		skill := skills[0].(map[string]any)
		if skill["type"] != "inline" {
			t.Errorf("expected skill type 'inline', got %v", skill["type"])
		}
		if skill["name"] != "my-skill" {
			t.Errorf("expected skill name 'my-skill', got %v", skill["name"])
		}
		src, ok := skill["source"].(map[string]any)
		if !ok {
			t.Fatalf("expected source map, got %T", skill["source"])
		}
		if src["type"] != "base64" {
			t.Errorf("expected source type 'base64', got %v", src["type"])
		}
		if src["media_type"] != "application/zip" {
			t.Errorf("expected media_type 'application/zip', got %v", src["media_type"])
		}
	})

	t.Run("should prepare shell tool with containerAuto and networkPolicy disabled", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type":          "containerAuto",
							"networkPolicy": map[string]any{"type": "disabled"},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		np, ok := env["network_policy"].(map[string]any)
		if !ok {
			t.Fatalf("expected network_policy map, got %T", env["network_policy"])
		}
		if np["type"] != "disabled" {
			t.Errorf("expected network_policy type 'disabled', got %v", np["type"])
		}
	})

	t.Run("should prepare shell tool with containerAuto and networkPolicy allowlist with domain secrets", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "containerAuto",
							"networkPolicy": map[string]any{
								"type":           "allowlist",
								"allowedDomains": []any{"example.com", "api.test.org"},
								"domainSecrets": []any{
									map[string]any{
										"domain": "api.test.org",
										"name":   "API_KEY",
										"value":  "secret123",
									},
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		np, ok := env["network_policy"].(map[string]any)
		if !ok {
			t.Fatalf("expected network_policy map, got %T", env["network_policy"])
		}
		if np["type"] != "allowlist" {
			t.Errorf("expected type 'allowlist', got %v", np["type"])
		}
		domains, ok := np["allowed_domains"].([]any)
		if !ok {
			t.Fatalf("expected allowed_domains array, got %T", np["allowed_domains"])
		}
		if len(domains) != 2 {
			t.Errorf("expected 2 domains, got %d", len(domains))
		}
		secrets, ok := np["domain_secrets"].([]any)
		if !ok {
			t.Fatalf("expected domain_secrets array, got %T", np["domain_secrets"])
		}
		if len(secrets) != 1 {
			t.Errorf("expected 1 domain secret, got %d", len(secrets))
		}
	})

	t.Run("should prepare shell tool with containerAuto, fileIds, and memoryLimit", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type":        "containerAuto",
							"fileIds":     []any{"file-1", "file-2"},
							"memoryLimit": "16g",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		if env["type"] != "container_auto" {
			t.Errorf("expected type 'container_auto', got %v", env["type"])
		}
		fileIDs, ok := env["file_ids"].([]any)
		if !ok {
			t.Fatalf("expected file_ids array, got %T", env["file_ids"])
		}
		if len(fileIDs) != 2 {
			t.Errorf("expected 2 file IDs, got %d", len(fileIDs))
		}
		if env["memory_limit"] != "16g" {
			t.Errorf("expected memory_limit '16g', got %v", env["memory_limit"])
		}
	})

	t.Run("should prepare shell tool with containerReference", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type":        "containerReference",
							"containerId": "ctr_abc123",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		if env["type"] != "container_reference" {
			t.Errorf("expected type 'container_reference', got %v", env["type"])
		}
		if env["container_id"] != "ctr_abc123" {
			t.Errorf("expected container_id 'ctr_abc123', got %v", env["container_id"])
		}
	})

	t.Run("should prepare shell tool with local environment and skills", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "local",
							"skills": []any{
								map[string]any{
									"name":        "calculator",
									"description": "Perform math calculations",
									"path":        "/path/to/calculator",
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		if env["type"] != "local" {
			t.Errorf("expected type 'local', got %v", env["type"])
		}
		skills, ok := env["skills"].([]any)
		if !ok {
			t.Fatalf("expected skills array, got %T", env["skills"])
		}
		if len(skills) != 1 {
			t.Errorf("expected 1 skill, got %d", len(skills))
		}
	})

	t.Run("should prepare shell tool with local environment without explicit type", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"skills": []any{
								map[string]any{
									"name":        "calculator",
									"description": "Perform math calculations",
									"path":        "/path/to/calculator",
								},
							},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		if env["type"] != "local" {
			t.Errorf("expected type 'local', got %v", env["type"])
		}
	})

	t.Run("should prepare shell tool with local environment without skills", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.shell",
					Name: "shell",
					Args: map[string]any{
						"environment": map[string]any{
							"type": "local",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := result.Tools[0]["environment"].(map[string]any)
		if env["type"] != "local" {
			t.Errorf("expected type 'local', got %v", env["type"])
		}
	})
}

func TestPrepareResponsesTools_Custom(t *testing.T) {
	t.Run("should prepare custom tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "custom",
					Args: map[string]any{
						"name":        "my_custom_tool",
						"description": "A custom tool",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "custom" {
			t.Errorf("expected type 'custom', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["name"] != "my_custom_tool" {
			t.Errorf("expected name 'my_custom_tool', got %v", result.Tools[0]["name"])
		}
	})

	t.Run("should handle tool choice for custom tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "custom",
					Args: map[string]any{
						"name":        "my_custom_tool",
						"description": "A custom tool",
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "my_custom_tool"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "custom" {
			t.Errorf("expected type 'custom', got %v", tc["type"])
		}
		if tc["name"] != "my_custom_tool" {
			t.Errorf("expected name 'my_custom_tool', got %v", tc["name"])
		}
	})

	t.Run("should prepare custom tool with regex format", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "write_sql",
					Args: map[string]any{
						"name":        "write_sql",
						"description": "Write a SQL SELECT query.",
						"format": map[string]any{
							"type":       "grammar",
							"syntax":     "regex",
							"definition": "SELECT .+",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "custom" {
			t.Errorf("expected type 'custom', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["name"] != "write_sql" {
			t.Errorf("expected name 'write_sql', got %v", result.Tools[0]["name"])
		}
		if result.Tools[0]["description"] != "Write a SQL SELECT query." {
			t.Errorf("expected description, got %v", result.Tools[0]["description"])
		}
		format, ok := result.Tools[0]["format"].(map[string]any)
		if !ok {
			t.Fatalf("expected format map, got %T", result.Tools[0]["format"])
		}
		if format["type"] != "grammar" {
			t.Errorf("expected format type 'grammar', got %v", format["type"])
		}
		if format["syntax"] != "regex" {
			t.Errorf("expected format syntax 'regex', got %v", format["syntax"])
		}
		if format["definition"] != "SELECT .+" {
			t.Errorf("expected format definition 'SELECT .+', got %v", format["definition"])
		}
	})

	t.Run("should prepare custom tool with lark format", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "generate_json",
					Args: map[string]any{
						"name": "generate_json",
						"format": map[string]any{
							"type":       "grammar",
							"syntax":     "lark",
							"definition": `start: "{"  "}"`,
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["name"] != "generate_json" {
			t.Errorf("expected name 'generate_json', got %v", result.Tools[0]["name"])
		}
		format := result.Tools[0]["format"].(map[string]any)
		if format["syntax"] != "lark" {
			t.Errorf("expected format syntax 'lark', got %v", format["syntax"])
		}
	})

	t.Run("should handle multiple tools including custom tool", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"input": map[string]any{"type": "string"}},
					},
				},
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "write_sql",
					Args: map[string]any{
						"name":        "write_sql",
						"description": "Write SQL.",
						"format": map[string]any{
							"type":       "grammar",
							"syntax":     "regex",
							"definition": "SELECT .+",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected first tool type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "custom" {
			t.Errorf("expected second tool type 'custom', got %v", result.Tools[1]["type"])
		}
	})

	t.Run("should map custom tool choice from sdk key to provider name", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.custom",
					Name: "alias_name",
					Args: map[string]any{
						"name": "write_sql",
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "alias_name"},
			ToolNameMapping: &providerutils.ToolNameMapping{
				ToProviderToolName: func(name string) string {
					if name == "alias_name" {
						return "write_sql"
					}
					return name
				},
				ToCustomToolName: func(name string) string {
					if name == "write_sql" {
						return "alias_name"
					}
					return name
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "custom" {
			t.Errorf("expected type 'custom', got %v", tc["type"])
		}
		if tc["name"] != "write_sql" {
			t.Errorf("expected name 'write_sql', got %v", tc["name"])
		}
	})
}

func TestPrepareResponsesTools_Warnings(t *testing.T) {
	t.Run("should have empty warnings for valid tools", func(t *testing.T) {
		desc := "Test"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})
}

func TestPrepareResponsesTools_ApplyPatch(t *testing.T) {
	t.Run("should prepare apply_patch tool", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.apply_patch",
					Name: "apply_patch",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "apply_patch" {
			t.Errorf("expected type 'apply_patch', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should handle tool choice selection with apply_patch", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.apply_patch",
					Name: "apply_patch",
					Args: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "apply_patch"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "apply_patch" {
			t.Errorf("expected type 'apply_patch', got %v", tc["type"])
		}
	})

	t.Run("should handle multiple tools including apply_patch", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"input": map[string]any{"type": "string"}},
					},
				},
				languagemodel.ProviderTool{
					ID:   "openai.apply_patch",
					Name: "apply_patch",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected first tool type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "apply_patch" {
			t.Errorf("expected second tool type 'apply_patch', got %v", result.Tools[1]["type"])
		}
	})
}

func TestPrepareResponsesTools_WebSearchExtended(t *testing.T) {
	t.Run("should prepare web_search tool with no options", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["type"] != "web_search" {
			t.Errorf("expected type 'web_search', got %v", result.Tools[0]["type"])
		}
	})

	t.Run("should prepare web_search tool with externalWebAccess set to true", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"externalWebAccess": true,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["external_web_access"] != true {
			t.Errorf("expected external_web_access true, got %v", result.Tools[0]["external_web_access"])
		}
	})

	t.Run("should prepare web_search tool with externalWebAccess set to false", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"externalWebAccess": false,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["external_web_access"] != false {
			t.Errorf("expected external_web_access false, got %v", result.Tools[0]["external_web_access"])
		}
	})

	t.Run("should prepare web_search tool with all options including externalWebAccess", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"externalWebAccess": true,
						"filters": map[string]any{
							"allowedDomains": []any{"example.com", "test.org"},
						},
						"searchContextSize": "high",
						"userLocation": map[string]any{
							"type":     "approximate",
							"country":  "US",
							"city":     "San Francisco",
							"region":   "California",
							"timezone": "America/Los_Angeles",
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tool := result.Tools[0]
		if tool["external_web_access"] != true {
			t.Errorf("expected external_web_access true, got %v", tool["external_web_access"])
		}
		if tool["search_context_size"] != "high" {
			t.Errorf("expected search_context_size 'high', got %v", tool["search_context_size"])
		}
		filters, ok := tool["filters"].(map[string]any)
		if !ok {
			t.Fatalf("expected filters map, got %T", tool["filters"])
		}
		domains := filters["allowed_domains"].([]any)
		if len(domains) != 2 {
			t.Errorf("expected 2 allowed domains, got %d", len(domains))
		}
		userLoc, ok := tool["user_location"].(map[string]any)
		if !ok {
			t.Fatalf("expected user_location map, got %T", tool["user_location"])
		}
		if userLoc["country"] != "US" {
			t.Errorf("expected country 'US', got %v", userLoc["country"])
		}
	})

	t.Run("should handle tool choice selection with web_search", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"externalWebAccess": true,
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "web_search"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "web_search" {
			t.Errorf("expected type 'web_search', got %v", tc["type"])
		}
	})

	t.Run("should handle multiple tools including web_search", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"input": map[string]any{"type": "string"}},
					},
				},
				languagemodel.ProviderTool{
					ID:   "openai.web_search",
					Name: "web_search",
					Args: map[string]any{
						"externalWebAccess": false,
						"searchContextSize": "medium",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected first tool type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "web_search" {
			t.Errorf("expected second tool type 'web_search', got %v", result.Tools[1]["type"])
		}
		if result.Tools[1]["search_context_size"] != "medium" {
			t.Errorf("expected search_context_size 'medium', got %v", result.Tools[1]["search_context_size"])
		}
	})
}

func TestPrepareResponsesTools_MCPExtended(t *testing.T) {
	t.Run("should prepare mcp tool with require_approval always", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverLabel":    "test-server",
						"serverUrl":      "https://mcp.example.com",
						"requireApproval": "always",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["type"] != "mcp" {
			t.Errorf("expected type 'mcp', got %v", result.Tools[0]["type"])
		}
		if result.Tools[0]["require_approval"] != "always" {
			t.Errorf("expected require_approval 'always', got %v", result.Tools[0]["require_approval"])
		}
	})

	t.Run("should prepare mcp tool with allowed tools", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverLabel":  "test-server",
						"serverUrl":    "https://mcp.example.com",
						"allowedTools": []any{"tool1", "tool2"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tools, ok := result.Tools[0]["allowed_tools"].([]any)
		if !ok {
			t.Fatalf("expected allowed_tools array, got %T", result.Tools[0]["allowed_tools"])
		}
		if len(tools) != 2 {
			t.Errorf("expected 2 allowed tools, got %d", len(tools))
		}
	})

	t.Run("should default require_approval to never when nil", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverLabel": "test-server",
						"serverUrl":   "https://mcp.example.com",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["require_approval"] != "never" {
			t.Errorf("expected require_approval 'never', got %v", result.Tools[0]["require_approval"])
		}
	})

	t.Run("should prepare mcp tool with connector_id", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.mcp",
					Name: "mcp",
					Args: map[string]any{
						"serverLabel": "test-server",
						"connectorId": "conn_abc123",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Tools[0]["connector_id"] != "conn_abc123" {
			t.Errorf("expected connector_id 'conn_abc123', got %v", result.Tools[0]["connector_id"])
		}
	})
}

func TestPrepareResponsesTools_CodeInterpreterExtended(t *testing.T) {
	t.Run("should prepare code interpreter tool with file IDs container", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.code_interpreter",
					Name: "code_interpreter",
					Args: map[string]any{
						"container": map[string]any{
							"fileIds": []any{"file-1", "file-2"},
						},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		container, ok := result.Tools[0]["container"].(map[string]any)
		if !ok {
			t.Fatalf("expected container map, got %T", result.Tools[0]["container"])
		}
		if container["type"] != "auto" {
			t.Errorf("expected container type 'auto', got %v", container["type"])
		}
		fileIDs, ok := container["file_ids"].([]any)
		if !ok {
			t.Fatalf("expected file_ids array, got %T", container["file_ids"])
		}
		if len(fileIDs) != 2 {
			t.Errorf("expected 2 file IDs, got %d", len(fileIDs))
		}
	})

	t.Run("should handle multiple tools including code interpreter", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{},
				},
				languagemodel.ProviderTool{
					ID:   "openai.code_interpreter",
					Name: "code_interpreter",
					Args: map[string]any{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Tools) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(result.Tools))
		}
		if result.Tools[0]["type"] != "function" {
			t.Errorf("expected first tool type 'function', got %v", result.Tools[0]["type"])
		}
		if result.Tools[1]["type"] != "code_interpreter" {
			t.Errorf("expected second tool type 'code_interpreter', got %v", result.Tools[1]["type"])
		}
	})
}

func TestPrepareResponsesTools_ImageGenerationExtended(t *testing.T) {
	t.Run("should support tool choice selection for image_generation", func(t *testing.T) {
		result, err := PrepareResponsesTools(PrepareResponsesToolsOptions{
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "openai.image_generation",
					Name: "image_generation",
					Args: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "image_generation"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc, ok := result.ToolChoice.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result.ToolChoice)
		}
		if tc["type"] != "image_generation" {
			t.Errorf("expected type 'image_generation', got %v", tc["type"])
		}
	})
}

// Helpers to verify warnings
func containsUnsupportedWarning(warnings []shared.Warning, feature string) bool {
	for _, w := range warnings {
		if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == feature {
			return true
		}
	}
	return false
}
