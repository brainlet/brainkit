// Ported from: packages/ai/src/middleware/add-tool-input-examples-middleware.test.ts
package middleware

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

var toolTestBaseParams = lm.CallOptions{
	Prompt: lm.Prompt{
		lm.UserMessage{Content: []lm.UserMessagePart{lm.TextPart{Text: "Hello, world!"}}},
	},
}

var toolTestMockModel = &mockLanguageModel{providerVal: "mock-provider", modelIDVal: "mock-model"}

func TestAddToolInputExamples_AppendExamplesToDescription(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	desc := "Get the weather in a location"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "San Francisco"}},
				{Input: jsonvalue.JSONObject{"location": "London"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if ft.Description == nil {
		t.Fatal("expected description to be non-nil")
	}
	if !strings.Contains(*ft.Description, "Input Examples:") {
		t.Error("expected description to contain 'Input Examples:'")
	}
	if !strings.Contains(*ft.Description, `{"location":"San Francisco"}`) {
		t.Error("expected description to contain San Francisco example")
	}
	if !strings.Contains(*ft.Description, `{"location":"London"}`) {
		t.Error("expected description to contain London example")
	}
	// inputExamples should be removed by default
	if len(ft.InputExamples) != 0 {
		t.Error("expected inputExamples to be removed")
	}
}

func TestAddToolInputExamples_HandleToolWithoutDescription(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name: "weather",
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "Berlin"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if ft.Description == nil {
		t.Fatal("expected description to be set")
	}
	if !strings.HasPrefix(*ft.Description, "Input Examples:") {
		t.Errorf("expected description to start with 'Input Examples:', got %s", *ft.Description)
	}
}

func TestAddToolInputExamples_CustomPrefix(t *testing.T) {
	prefix := "Here are some example inputs:"
	m := AddToolInputExamplesMiddleware(&AddToolInputExamplesMiddlewareOptions{
		Prefix: &prefix,
	})

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "Paris"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if !strings.Contains(*ft.Description, "Here are some example inputs:") {
		t.Error("expected custom prefix")
	}
}

func TestAddToolInputExamples_DefaultJSONFormat(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	desc := "Search for items"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "search",
			Description: &desc,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string"},
					"limit": map[string]any{"type": "number"},
				},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"query": "test", "limit": json.Number("10")}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if !strings.Contains(*ft.Description, `"query":"test"`) {
		t.Error("expected query in description")
	}
}

func TestAddToolInputExamples_CustomFormatFunction(t *testing.T) {
	m := AddToolInputExamplesMiddleware(&AddToolInputExamplesMiddlewareOptions{
		Format: func(example lm.FunctionToolInputExample, index int) string {
			b, _ := json.Marshal(example.Input)
			return fmt.Sprintf("%d. %s", index+1, string(b))
		},
	})

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "Paris"}},
				{Input: jsonvalue.JSONObject{"location": "Tokyo"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if !strings.Contains(*ft.Description, "1. ") {
		t.Error("expected numbered format")
	}
	if !strings.Contains(*ft.Description, "2. ") {
		t.Error("expected second numbered format")
	}
}

func TestAddToolInputExamples_RemoveByDefault(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "NYC"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if len(ft.InputExamples) != 0 {
		t.Error("expected inputExamples to be removed")
	}
}

func TestAddToolInputExamples_KeepWhenRemoveFalse(t *testing.T) {
	removeFalse := false
	m := AddToolInputExamplesMiddleware(&AddToolInputExamplesMiddlewareOptions{
		Remove: &removeFalse,
	})

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "NYC"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if len(ft.InputExamples) != 1 {
		t.Error("expected inputExamples to be kept")
	}
}

func TestAddToolInputExamples_PassThroughWithoutExamples(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if *ft.Description != "Get the weather" {
		t.Errorf("expected original description, got %s", *ft.Description)
	}
}

func TestAddToolInputExamples_EmptyExamplesArray(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	desc := "Get the weather"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &desc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	ft := result.Tools[0].(lm.FunctionTool)
	if *ft.Description != "Get the weather" {
		t.Errorf("expected original description, got %s", *ft.Description)
	}
}

func TestAddToolInputExamples_ProviderToolPassThrough(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.ProviderTool{
			ID:   "anthropic.web_search_20250305",
			Name: "web_search",
			Args: map[string]any{"maxUses": 5},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	pt, ok := result.Tools[0].(lm.ProviderTool)
	if !ok {
		t.Fatal("expected ProviderTool")
	}
	if pt.Name != "web_search" {
		t.Errorf("expected web_search, got %s", pt.Name)
	}
}

func TestAddToolInputExamples_MixedTools(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	weatherDesc := "Get the weather"
	timeDesc := "Get the current time"
	params := toolTestBaseParams
	params.Tools = []lm.Tool{
		lm.FunctionTool{
			Name:        "weather",
			Description: &weatherDesc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"location": map[string]any{"type": "string"}},
			},
			InputExamples: []lm.FunctionToolInputExample{
				{Input: jsonvalue.JSONObject{"location": "NYC"}},
			},
		},
		lm.FunctionTool{
			Name:        "time",
			Description: &timeDesc,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"timezone": map[string]any{"type": "string"}},
			},
		},
	}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result.Tools))
	}

	// First tool should have examples appended
	ft1 := result.Tools[0].(lm.FunctionTool)
	if !strings.Contains(*ft1.Description, "Input Examples:") {
		t.Error("expected first tool to have examples")
	}

	// Second tool should be unchanged
	ft2 := result.Tools[1].(lm.FunctionTool)
	if *ft2.Description != "Get the current time" {
		t.Errorf("expected second tool unchanged, got %s", *ft2.Description)
	}
}

func TestAddToolInputExamples_EmptyToolsArray(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	params := toolTestBaseParams
	params.Tools = []lm.Tool{}

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Tools) != 0 {
		t.Errorf("expected empty tools, got %d", len(result.Tools))
	}
}

func TestAddToolInputExamples_NilTools(t *testing.T) {
	m := AddToolInputExamplesMiddleware(nil)

	params := toolTestBaseParams
	params.Tools = nil

	result, err := m.TransformParams(mw.TransformParamsOptions{
		Type:   "generate",
		Params: params,
		Model:  toolTestMockModel,
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Tools != nil {
		t.Errorf("expected nil tools, got %v", result.Tools)
	}
}
