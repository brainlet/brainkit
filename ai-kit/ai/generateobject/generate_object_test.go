// Ported from: packages/ai/src/generate-object/generate-object.test.ts
package generateobject

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// mockLanguageModel implements the LanguageModel interface for testing.
type mockLanguageModel struct {
	provider   string
	modelID    string
	doGenerate func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error)
	calls      []DoGenerateObjectOptions
}

func (m *mockLanguageModel) Provider() string {
	if m.provider != "" {
		return m.provider
	}
	return "mock-provider"
}

func (m *mockLanguageModel) ModelID() string {
	if m.modelID != "" {
		return m.modelID
	}
	return "mock-model-id"
}

func (m *mockLanguageModel) DoGenerate(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
	m.calls = append(m.calls, opts)
	if m.doGenerate != nil {
		return m.doGenerate(ctx, opts)
	}
	return nil, fmt.Errorf("doGenerate not configured")
}

// dummyResponseResult creates a standard DoGenerateObjectResult for tests.
func dummyResponseResult(text string) *DoGenerateObjectResult {
	return &DoGenerateObjectResult{
		Text:         text,
		FinishReason: "stop",
		Usage: LanguageModelUsage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
		Response: LanguageModelResponseMetadata{
			ID:      "id-1",
			ModelID: "m-1",
		},
	}
}

func TestGenerateObject_Object_ShouldGenerateObject(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "Hello, world!" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_ShouldUseNameAndDescription(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			if opts.SchemaName != "test-name" {
				t.Errorf("expected SchemaName 'test-name', got %q", opts.SchemaName)
			}
			if opts.SchemaDescription != "test description" {
				t.Errorf("expected SchemaDescription 'test description', got %q", opts.SchemaDescription)
			}
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:             model,
		Schema:            schema,
		SchemaName:        "test-name",
		SchemaDescription: "test description",
		Prompt:            "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "Hello, world!" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_ShouldReturnWarnings(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			r := dummyResponseResult(`{ "content": "Hello, world!" }`)
			r.Warnings = []CallWarning{
				{Type: "other", Message: "Setting is not supported"},
			}
			return r, nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Type != "other" {
		t.Errorf("unexpected warning type: %s", result.Warnings[0].Type)
	}
	if result.Warnings[0].Message != "Setting is not supported" {
		t.Errorf("unexpected warning message: %s", result.Warnings[0].Message)
	}
}

func TestGenerateObject_Object_ResultRequest(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			r := dummyResponseResult(`{ "content": "Hello, world!" }`)
			r.Request = LanguageModelRequestMetadata{Body: "test body"}
			return r, nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Request.Body != "test body" {
		t.Errorf("unexpected request body: %v", result.Request.Body)
	}
}

func TestGenerateObject_Object_ResultResponse(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			r := dummyResponseResult(`{ "content": "Hello, world!" }`)
			r.Response = LanguageModelResponseMetadata{
				ID:      "test-id-from-model",
				ModelID: "test-response-model-id",
				Headers: map[string]string{
					"custom-response-header": "response-header-value",
				},
				Body: "test body",
			}
			return r, nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Response.ID != "test-id-from-model" {
		t.Errorf("unexpected response ID: %s", result.Response.ID)
	}
	if result.Response.ModelID != "test-response-model-id" {
		t.Errorf("unexpected response model ID: %s", result.Response.ModelID)
	}
	if result.Response.Headers["custom-response-header"] != "response-header-value" {
		t.Errorf("unexpected response header: %v", result.Response.Headers)
	}
	if result.Response.Body != "test body" {
		t.Errorf("unexpected response body: %v", result.Response.Body)
	}
}

func TestGenerateObject_Object_ToJSONResponse(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	jsonBytes, err := result.ToJSONResponse()
	if err != nil {
		t.Fatalf("unexpected error from ToJSONResponse: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("unexpected error unmarshaling JSON response: %v", err)
	}
	if parsed["content"] != "Hello, world!" {
		t.Errorf("unexpected JSON content: %v", parsed["content"])
	}
}

func TestGenerateObject_Object_ProviderMetadata(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			r := dummyResponseResult(`{ "content": "Hello, world!" }`)
			r.ProviderMetadata = ProviderMetadata{
				"exampleProvider": {
					"a": float64(10),
					"b": float64(20),
				},
			}
			return r, nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ProviderMetadata == nil {
		t.Fatal("expected provider metadata to be non-nil")
	}
	ep := result.ProviderMetadata["exampleProvider"]
	if ep == nil {
		t.Fatal("expected exampleProvider metadata")
	}
	if ep["a"] != float64(10) {
		t.Errorf("expected a=10, got %v", ep["a"])
	}
	if ep["b"] != float64(20) {
		t.Errorf("expected b=20, got %v", ep["b"])
	}
}

func TestGenerateObject_Object_Headers(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			if opts.Headers == nil {
				t.Error("expected headers to be non-nil")
			} else if opts.Headers["custom-request-header"] != "request-header-value" {
				t.Errorf("unexpected header value: %s", opts.Headers["custom-request-header"])
			}
			return dummyResponseResult(`{ "content": "headers test" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
		Headers: map[string]string{
			"custom-request-header": "request-header-value",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "headers test" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_RepairText_JSONParseError(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "provider metadata test" `), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
		RepairText: func(text string, parseError error) (string, error) {
			if text != `{ "content": "provider metadata test" ` {
				t.Errorf("unexpected text: %q", text)
			}
			return text + "}", nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "provider metadata test" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_ProviderOptions(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			if opts.ProviderOptions == nil {
				t.Error("expected provider options to be non-nil")
			} else if opts.ProviderOptions["aProvider"] == nil {
				t.Error("expected aProvider to be non-nil")
			} else if opts.ProviderOptions["aProvider"]["someKey"] != "someValue" {
				t.Errorf("unexpected provider option: %v", opts.ProviderOptions["aProvider"]["someKey"])
			}
			return dummyResponseResult(`{ "content": "provider metadata test" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
		ProviderOptions: map[string]map[string]any{
			"aProvider": {"someKey": "someValue"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "provider metadata test" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_ErrorOnInvalidJSON(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ broken json`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	_, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestGenerateObject_Object_ErrorOnEmptyText(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(""), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	_, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestGenerateObject_Array_ShouldGenerate3Elements(t *testing.T) {
	arrayJSON := `{"elements":[{"content":"element 1"},{"content":"element 2"},{"content":"element 3"}]}`
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(arrayJSON), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Output: "array",
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	arr, ok := result.Object.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result.Object)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr))
	}

	for i, expected := range []string{"element 1", "element 2", "element 3"} {
		elem, ok := arr[i].(map[string]any)
		if !ok {
			t.Fatalf("element %d: expected map, got %T", i, arr[i])
		}
		if elem["content"] != expected {
			t.Errorf("element %d: expected content %q, got %v", i, expected, elem["content"])
		}
	}
}

func TestGenerateObject_Enum_ShouldGenerateEnumValue(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{"result":"sunny"}`), nil
		},
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:      model,
		Output:     "enum",
		EnumValues: []string{"sunny", "rainy", "snowy"},
		Prompt:     "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object != "sunny" {
		t.Errorf("expected 'sunny', got %v", result.Object)
	}
}

func TestGenerateObject_NoSchema_ShouldGenerateObject(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Output: "no-schema",
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "Hello, world!" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_Reasoning(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			r := dummyResponseResult(`{ "content": "Hello, world!" }`)
			r.Reasoning = "This is a test reasoning.\nThis is another test reasoning."
			return r, nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Reasoning != "This is a test reasoning.\nThis is another test reasoning." {
		t.Errorf("unexpected reasoning: %q", result.Reasoning)
	}

	obj, ok := result.Object.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result.Object)
	}
	if obj["content"] != "Hello, world!" {
		t.Errorf("unexpected content: %v", obj["content"])
	}
}

func TestGenerateObject_Object_FinishReason(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.FinishReason != "stop" {
		t.Errorf("expected finish reason 'stop', got %q", result.FinishReason)
	}
}

func TestGenerateObject_Object_Usage(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	result, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Usage.PromptTokens != 10 {
		t.Errorf("expected prompt tokens 10, got %d", result.Usage.PromptTokens)
	}
	if result.Usage.CompletionTokens != 20 {
		t.Errorf("expected completion tokens 20, got %d", result.Usage.CompletionTokens)
	}
	if result.Usage.TotalTokens != 30 {
		t.Errorf("expected total tokens 30, got %d", result.Usage.TotalTokens)
	}
}

func TestGenerateObject_Object_ModePassedToModel(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			if opts.Mode != "json" {
				t.Errorf("expected mode 'json', got %q", opts.Mode)
			}
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	_, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateObject_Object_ToolMode(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			if opts.Mode != "tool" {
				t.Errorf("expected mode 'tool', got %q", opts.Mode)
			}
			return dummyResponseResult(`{ "content": "Hello, world!" }`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	_, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Mode:   "tool",
		Prompt: "prompt",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenerateObject_Object_RepairTextFailedRepairKeepsOriginalError(t *testing.T) {
	model := &mockLanguageModel{
		doGenerate: func(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error) {
			return dummyResponseResult(`{ broken json`), nil
		},
	}

	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}

	_, err := GenerateObject(context.Background(), GenerateObjectOptions{
		Model:  model,
		Schema: schema,
		Prompt: "prompt",
		RepairText: func(text string, parseError error) (string, error) {
			// Return a still-broken repair.
			return text + "{", nil
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
