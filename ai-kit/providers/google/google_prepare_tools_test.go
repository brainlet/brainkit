// Ported from: packages/google/src/google-prepare-tools.test.ts
package google

import (
	"reflect"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestPrepareTools(t *testing.T) {
	t.Run("should return nil tools and toolConfig when tools are nil", func(t *testing.T) {
		result, err := PrepareTools(nil, nil, "gemini-2.5-flash")
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolConfig != nil {
			t.Errorf("expected nil toolConfig, got %v", result.ToolConfig)
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should return nil tools and toolConfig when tools are empty", func(t *testing.T) {
		result, err := PrepareTools([]languagemodel.Tool{}, nil, "gemini-2.5-flash")
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if result.ToolConfig != nil {
			t.Errorf("expected nil toolConfig, got %v", result.ToolConfig)
		}
	})

	t.Run("should correctly prepare function tools", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool entry, got %d", len(result.Tools))
		}
		decls, ok := result.Tools[0]["functionDeclarations"]
		if !ok {
			t.Fatal("expected functionDeclarations key")
		}
		declSlice, ok := decls.([]map[string]any)
		if !ok {
			t.Fatalf("unexpected type for functionDeclarations: %T", decls)
		}
		if len(declSlice) != 1 {
			t.Fatalf("expected 1 declaration, got %d", len(declSlice))
		}
		if declSlice[0]["name"] != "testFunction" {
			t.Errorf("expected name 'testFunction', got %v", declSlice[0]["name"])
		}
		if declSlice[0]["description"] != "A test function" {
			t.Errorf("expected description 'A test function', got %v", declSlice[0]["description"])
		}
		// Empty object schema at root is converted to nil parameters
		if declSlice[0]["parameters"] != nil {
			t.Errorf("expected nil parameters for empty object schema, got %v", declSlice[0]["parameters"])
		}
		if result.ToolConfig != nil {
			t.Error("expected nil toolConfig")
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should correctly prepare provider-defined tools as array", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
				languagemodel.ProviderTool{ID: "google.url_context", Name: "url_context", Args: map[string]any{}},
				languagemodel.ProviderTool{
					ID:   "google.file_search",
					Name: "file_search",
					Args: map[string]any{"fileSearchStoreNames": []any{"projects/foo/fileSearchStores/bar"}},
				},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 3 {
			t.Fatalf("expected 3 tools, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleSearch"]; !ok {
			t.Error("expected googleSearch")
		}
		if _, ok := result.Tools[1]["urlContext"]; !ok {
			t.Error("expected urlContext")
		}
		if _, ok := result.Tools[2]["fileSearch"]; !ok {
			t.Error("expected fileSearch")
		}
		if len(result.ToolWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.ToolWarnings))
		}
	})

	t.Run("should correctly prepare single provider-defined tool", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleSearch"]; !ok {
			t.Error("expected googleSearch")
		}
	})

	t.Run("should add warnings for unsupported tools", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "unsupported.tool", Name: "unsupported_tool", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Errorf("expected nil tools, got %v", result.Tools)
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if w.Feature != "provider-defined tool unsupported.tool" {
			t.Errorf("unexpected feature: %q", w.Feature)
		}
	})

	t.Run("should add warnings for file search on unsupported models", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.file_search",
					Name: "file_search",
					Args: map[string]any{"fileSearchStoreNames": []any{"projects/foo/fileSearchStores/bar"}},
				},
			},
			nil,
			"gemini-1.5-flash-8b",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Error("expected nil tools")
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if w.Details == nil || *w.Details != "The file search tool is only supported with Gemini 2.5 models and Gemini 3 models." {
			t.Errorf("unexpected details: %v", w.Details)
		}
	})

	t.Run("should correctly prepare file search tool for gemini-2.5 models", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.file_search",
					Name: "file_search",
					Args: map[string]any{
						"fileSearchStoreNames": []any{"projects/foo/fileSearchStores/bar"},
						"metadataFilter":       "author=Robert Graves",
						"topK":                 5,
					},
				},
			},
			nil,
			"gemini-2.5-pro",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		fs, ok := result.Tools[0]["fileSearch"]
		if !ok {
			t.Fatal("expected fileSearch")
		}
		fsMap, ok := fs.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any for fileSearch, got %T", fs)
		}
		if fsMap["metadataFilter"] != "author=Robert Graves" {
			t.Error("expected metadataFilter")
		}
	})

	t.Run("should correctly prepare file search tool for gemini-3 models", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.file_search",
					Name: "file_search",
					Args: map[string]any{
						"fileSearchStoreNames": []any{"projects/foo/fileSearchStores/bar"},
						"metadataFilter":       "author=Robert Graves",
						"topK":                 5,
					},
				},
			},
			nil,
			"gemini-3.1-pro-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["fileSearch"]; !ok {
			t.Fatal("expected fileSearch")
		}
	})

	t.Run("should handle tool choice auto", func(t *testing.T) {
		desc := "Test"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{Name: "testFunction", Description: &desc, InputSchema: map[string]any{}},
			},
			languagemodel.ToolChoiceAuto{},
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig == nil || result.ToolConfig.FunctionCallingConfig == nil {
			t.Fatal("expected toolConfig")
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "AUTO" {
			t.Errorf("expected AUTO, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should handle tool choice required", func(t *testing.T) {
		desc := "Test"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{Name: "testFunction", Description: &desc, InputSchema: map[string]any{}},
			},
			languagemodel.ToolChoiceRequired{},
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
			t.Errorf("expected ANY, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should handle tool choice none", func(t *testing.T) {
		desc := "Test"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{Name: "testFunction", Description: &desc, InputSchema: map[string]any{}},
			},
			languagemodel.ToolChoiceNone{},
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "NONE" {
			t.Errorf("expected NONE, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should handle tool choice tool", func(t *testing.T) {
		desc := "Test"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{Name: "testFunction", Description: &desc, InputSchema: map[string]any{}},
			},
			languagemodel.ToolChoiceTool{ToolName: "testFunction"},
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "ANY" {
			t.Errorf("expected ANY, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
		if !reflect.DeepEqual(result.ToolConfig.FunctionCallingConfig.AllowedFunctionNames, []string{"testFunction"}) {
			t.Errorf("expected allowedFunctionNames ['testFunction']")
		}
	})

	t.Run("should warn when mixing function and provider-defined tools", func(t *testing.T) {
		desc := "A test function"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "testFunction",
					Description: &desc,
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
				},
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		// Provider-defined tools take precedence
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleSearch"]; !ok {
			t.Error("expected googleSearch tool")
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w, ok := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.ToolWarnings[0])
		}
		if w.Feature != "combination of function and provider-defined tools" {
			t.Errorf("unexpected feature: %q", w.Feature)
		}
	})

	t.Run("should handle latest modelId for provider-defined tools correctly", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
			},
			nil,
			"gemini-flash-latest",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleSearch"]; !ok {
			t.Error("expected googleSearch")
		}
	})

	t.Run("should handle gemini-3 modelId for provider-defined tools correctly", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
			},
			nil,
			"gemini-3.1-pro-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleSearch"]; !ok {
			t.Error("expected googleSearch")
		}
	})

	t.Run("should handle code execution tool", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.code_execution", Name: "code_execution", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["codeExecution"]; !ok {
			t.Error("expected codeExecution")
		}
	})

	t.Run("should handle url context tool alone", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.url_context", Name: "url_context", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["urlContext"]; !ok {
			t.Error("expected urlContext")
		}
	})

	t.Run("should handle google maps tool", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_maps", Name: "google_maps", Args: map[string]any{}},
			},
			nil,
			"gemini-2.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(result.Tools))
		}
		if _, ok := result.Tools[0]["googleMaps"]; !ok {
			t.Error("expected googleMaps")
		}
	})

	t.Run("should pass searchTypes args through for google search", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "google.google_search",
					Name: "google_search",
					Args: map[string]any{
						"searchTypes": map[string]any{"webSearch": map[string]any{}, "imageSearch": map[string]any{}},
					},
				},
			},
			nil,
			"gemini-3.1-flash-image-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		gs, ok := result.Tools[0]["googleSearch"]
		if !ok {
			t.Fatal("expected googleSearch")
		}
		gsMap, ok := gs.(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", gs)
		}
		if gsMap["searchTypes"] == nil {
			t.Error("expected searchTypes to be passed through")
		}
	})

	t.Run("should add warnings for google search on unsupported models", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_search", Name: "google_search", Args: map[string]any{}},
			},
			nil,
			"gemini-1.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Error("expected nil tools")
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if w.Details == nil || *w.Details != "Google Search requires Gemini 2.0 or newer." {
			t.Errorf("unexpected details: %v", w.Details)
		}
	})

	t.Run("should add warnings for google maps on unsupported models", func(t *testing.T) {
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.ProviderTool{ID: "google.google_maps", Name: "google_maps", Args: map[string]any{}},
			},
			nil,
			"gemini-1.5-flash",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.Tools != nil {
			t.Error("expected nil tools")
		}
		if len(result.ToolWarnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.ToolWarnings))
		}
		w := result.ToolWarnings[0].(shared.UnsupportedWarning)
		if w.Details == nil || *w.Details != "The Google Maps grounding tool is not supported with Gemini models other than Gemini 2 or newer." {
			t.Errorf("unexpected details: %v", w.Details)
		}
	})

	t.Run("should use VALIDATED mode when any function tool has strict true", func(t *testing.T) {
		desc := "Create a meeting"
		strict := true
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "createMeeting",
					Description: &desc,
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"title": map[string]any{"type": "string"}},
						"required":             []any{"title"},
						"additionalProperties": false,
					},
					Strict: &strict,
				},
			},
			nil,
			"gemini-3-flash-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig == nil || result.ToolConfig.FunctionCallingConfig == nil {
			t.Fatal("expected toolConfig")
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "VALIDATED" {
			t.Errorf("expected VALIDATED, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should use VALIDATED mode with toolChoice auto when strict true", func(t *testing.T) {
		desc := "Get weather"
		strict := true
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "getWeather",
					Description: &desc,
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"city": map[string]any{"type": "string"}},
						"required":             []any{"city"},
						"additionalProperties": false,
					},
					Strict: &strict,
				},
			},
			languagemodel.ToolChoiceAuto{},
			"gemini-3-flash-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "VALIDATED" {
			t.Errorf("expected VALIDATED, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should use VALIDATED mode with toolChoice required when strict true", func(t *testing.T) {
		desc := "Get weather"
		strict := true
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "getWeather",
					Description: &desc,
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"city": map[string]any{"type": "string"}},
						"required":             []any{"city"},
						"additionalProperties": false,
					},
					Strict: &strict,
				},
			},
			languagemodel.ToolChoiceRequired{},
			"gemini-3-flash-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "VALIDATED" {
			t.Errorf("expected VALIDATED, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})

	t.Run("should use AUTO mode when no tools have strict true", func(t *testing.T) {
		desc := "Get weather"
		result, err := PrepareTools(
			[]languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "getWeather",
					Description: &desc,
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"city": map[string]any{"type": "string"}},
						"required":             []any{"city"},
						"additionalProperties": false,
					},
				},
			},
			languagemodel.ToolChoiceAuto{},
			"gemini-3-flash-preview",
		)
		if err != nil {
			t.Fatal(err)
		}
		if result.ToolConfig.FunctionCallingConfig.Mode != "AUTO" {
			t.Errorf("expected AUTO, got %q", result.ToolConfig.FunctionCallingConfig.Mode)
		}
	})
}
