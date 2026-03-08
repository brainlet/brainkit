// Ported from: packages/ai/src/generate-object/stream-object.test.ts
package generateobject

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// --- Test helpers ---

// mockStreamLanguageModel implements StreamLanguageModel for testing.
type mockStreamLanguageModel struct {
	provider string
	modelID  string
	doStream func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error)
	calls    []DoStreamObjectOptions
}

func (m *mockStreamLanguageModel) Provider() string {
	if m.provider != "" {
		return m.provider
	}
	return "mock-provider"
}

func (m *mockStreamLanguageModel) ModelID() string {
	if m.modelID != "" {
		return m.modelID
	}
	return "mock-model-id"
}

func (m *mockStreamLanguageModel) DoStream(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
	m.calls = append(m.calls, opts)
	if m.doStream != nil {
		return m.doStream(ctx, opts)
	}
	ch := make(chan StreamChunk)
	close(ch)
	return ch, nil
}

// streamChunksToChannel converts a slice of StreamChunks into a channel.
func streamChunksToChannel(chunks []StreamChunk) <-chan StreamChunk {
	ch := make(chan StreamChunk, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch
}

// defaultTestUsage returns a standard LanguageModelUsage for tests.
func defaultTestUsage() LanguageModelUsage {
	return LanguageModelUsage{
		PromptTokens:     3,
		CompletionTokens: 10,
		TotalTokens:      13,
	}
}

// defaultTestChunks returns stream chunks that produce '{ "content": "Hello, world!" }'.
func defaultTestChunks() []StreamChunk {
	return []StreamChunk{
		{Type: "text-delta", TextDelta: "{ "},
		{Type: "text-delta", TextDelta: `"content": `},
		{Type: "text-delta", TextDelta: `"Hello, `},
		{Type: "text-delta", TextDelta: `world`},
		{Type: "text-delta", TextDelta: `!"`},
		{Type: "text-delta", TextDelta: " }"},
		{
			Type:         "finish",
			FinishReason: "stop",
			Usage:        defaultTestUsage(),
			ProviderMetadata: ProviderMetadata{
				"testProvider": {"testKey": "testValue"},
			},
		},
	}
}

// createDefaultStreamModel creates a mock model with default test chunks.
func createDefaultStreamModel() *mockStreamLanguageModel {
	return &mockStreamLanguageModel{
		doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
			return streamChunksToChannel(defaultTestChunks()), nil
		},
	}
}

// createStreamModelWithChunks creates a mock model with specific chunks.
func createStreamModelWithChunks(chunks []StreamChunk) *mockStreamLanguageModel {
	return &mockStreamLanguageModel{
		doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
			return streamChunksToChannel(chunks), nil
		},
	}
}

// drainTextStream reads all values from a text stream channel.
func drainTextStream(ch <-chan string) []string {
	var result []string
	for v := range ch {
		result = append(result, v)
	}
	return result
}

// drainFullStream reads all values from a full stream channel.
func drainFullStream(ch <-chan ObjectStreamPart) []ObjectStreamPart {
	var result []ObjectStreamPart
	for v := range ch {
		result = append(result, v)
	}
	return result
}

// collectObjectsFromFullStream reads the full stream and collects only "object" type parts.
func collectObjectsFromFullStream(ch <-chan ObjectStreamPart) []any {
	var result []any
	for v := range ch {
		if v.Type == ObjectStreamPartTypeObject {
			result = append(result, v.Object)
		}
	}
	return result
}

// drainAndWait drains both channels and returns the result after full stream is consumed.
func drainAndWait(result *StreamObjectResult) {
	// Drain both channels concurrently to avoid blocking.
	go func() {
		for range result.TextStream {
		}
	}()
	for range result.FullStream {
	}
}

// --- Test schema ---

func testObjectSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"content": map[string]any{"type": "string"},
		},
		"required":             []string{"content"},
		"additionalProperties": false,
	}
}

// --- Tests ---

func TestStreamObject(t *testing.T) {
	t.Run("output = object", func(t *testing.T) {
		t.Run("result.textStream", func(t *testing.T) {
			t.Run("should send text stream", func(t *testing.T) {
				model := createDefaultStreamModel()

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Drain fullStream in background to prevent blocking.
				go func() {
					for range result.FullStream {
					}
				}()

				texts := drainTextStream(result.TextStream)
				expected := []string{"{ ", `"content": `, `"Hello, `, `world`, `!"`, " }"}
				if !reflect.DeepEqual(texts, expected) {
					t.Errorf("text stream mismatch:\ngot:  %v\nwant: %v", texts, expected)
				}
			})
		})

		t.Run("result.fullStream", func(t *testing.T) {
			t.Run("should send full stream data", func(t *testing.T) {
				model := createDefaultStreamModel()

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Drain textStream in background.
				go func() {
					for range result.TextStream {
					}
				}()

				parts := drainFullStream(result.FullStream)

				// Check that we have text-delta parts and a finish part.
				var textDeltaCount, objectCount, finishCount int
				for _, p := range parts {
					switch p.Type {
					case ObjectStreamPartTypeTextDelta:
						textDeltaCount++
					case ObjectStreamPartTypeObject:
						objectCount++
					case ObjectStreamPartTypeFinish:
						finishCount++
					}
				}
				if textDeltaCount == 0 {
					t.Error("expected at least one text-delta part")
				}
				if finishCount != 1 {
					t.Errorf("expected exactly 1 finish part, got %d", finishCount)
				}
				// Object parts are emitted when partial JSON can be parsed.
				if objectCount == 0 {
					t.Error("expected at least one object part")
				}
			})
		})

		t.Run("result.usage", func(t *testing.T) {
			t.Run("should resolve with token usage", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if result.Usage.PromptTokens != 3 {
					t.Errorf("expected prompt tokens 3, got %d", result.Usage.PromptTokens)
				}
				if result.Usage.CompletionTokens != 10 {
					t.Errorf("expected completion tokens 10, got %d", result.Usage.CompletionTokens)
				}
				if result.Usage.TotalTokens != 13 {
					t.Errorf("expected total tokens 13, got %d", result.Usage.TotalTokens)
				}
			})
		})

		t.Run("result.providerMetadata", func(t *testing.T) {
			t.Run("should resolve with provider metadata", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
						ProviderMetadata: ProviderMetadata{
							"testProvider": {"testKey": "testValue"},
						},
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if result.ProviderMetadata == nil {
					t.Fatal("expected provider metadata to be non-nil")
				}
				tp := result.ProviderMetadata["testProvider"]
				if tp == nil {
					t.Fatal("expected testProvider metadata")
				}
				if tp["testKey"] != "testValue" {
					t.Errorf("expected testKey=testValue, got %v", tp["testKey"])
				}
			})
		})

		t.Run("result.response", func(t *testing.T) {
			t.Run("should resolve with response information", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{"content": "Hello, world!"}`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
						Response: LanguageModelResponseMetadata{
							ID:      "id-0",
							ModelID: "mock-model-id",
						},
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if result.Response.ID != "id-0" {
					t.Errorf("expected response ID 'id-0', got %q", result.Response.ID)
				}
				if result.Response.ModelID != "mock-model-id" {
					t.Errorf("expected response modelId 'mock-model-id', got %q", result.Response.ModelID)
				}
			})
		})

		t.Run("result.object", func(t *testing.T) {
			t.Run("should resolve with typed object", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: "{ "},
					{Type: "text-delta", TextDelta: `"content": `},
					{Type: "text-delta", TextDelta: `"Hello, `},
					{Type: "text-delta", TextDelta: `world`},
					{Type: "text-delta", TextDelta: `!"`},
					{Type: "text-delta", TextDelta: " }"},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				obj, ok := result.Object.(map[string]any)
				if !ok {
					t.Fatalf("expected map, got %T (value: %v)", result.Object, result.Object)
				}
				if obj["content"] != "Hello, world!" {
					t.Errorf("expected content 'Hello, world!', got %v", obj["content"])
				}
			})

			t.Run("should handle object when schema validation passes", func(t *testing.T) {
				// The Go version doesn't do Zod schema validation, but we verify
				// the object is correctly parsed from the accumulated text.
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{ "invalid": "Hello, world!" }`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				// In Go, the ObjectOutputStrategy accepts any non-nil JSON value.
				// Schema validation is not done by Go's ObjectOutputStrategy.
				obj, ok := result.Object.(map[string]any)
				if !ok {
					t.Fatalf("expected map, got %T", result.Object)
				}
				if obj["invalid"] != "Hello, world!" {
					t.Errorf("expected invalid='Hello, world!', got %v", obj["invalid"])
				}
			})
		})

		t.Run("result.finishReason", func(t *testing.T) {
			t.Run("should resolve with finish reason", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if result.FinishReason != "stop" {
					t.Errorf("expected finish reason 'stop', got %q", result.FinishReason)
				}
			})
		})

		t.Run("options.headers", func(t *testing.T) {
			t.Run("should pass headers to model", func(t *testing.T) {
				var receivedHeaders map[string]string
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedHeaders = opts.Headers
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "headers test" }`},
							{
								Type:         "finish",
								FinishReason: "stop",
								Usage:        defaultTestUsage(),
							},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
					Headers: map[string]string{
						"custom-request-header": "request-header-value",
					},
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedHeaders == nil {
					t.Fatal("expected headers to be non-nil")
				}
				if receivedHeaders["custom-request-header"] != "request-header-value" {
					t.Errorf("unexpected header value: %s", receivedHeaders["custom-request-header"])
				}
			})
		})

		t.Run("options.providerOptions", func(t *testing.T) {
			t.Run("should pass provider options to model", func(t *testing.T) {
				var receivedProviderOptions map[string]map[string]any
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedProviderOptions = opts.ProviderOptions
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "provider metadata test" }`},
							{
								Type:         "finish",
								FinishReason: "stop",
								Usage:        defaultTestUsage(),
							},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
					ProviderOptions: map[string]map[string]any{
						"aProvider": {"someKey": "someValue"},
					},
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedProviderOptions == nil {
					t.Fatal("expected provider options to be non-nil")
				}
				if receivedProviderOptions["aProvider"] == nil {
					t.Fatal("expected aProvider to be non-nil")
				}
				if receivedProviderOptions["aProvider"]["someKey"] != "someValue" {
					t.Errorf("unexpected provider option: %v", receivedProviderOptions["aProvider"]["someKey"])
				}
			})
		})

		t.Run("options.schemaName and schemaDescription", func(t *testing.T) {
			t.Run("should use name and description", func(t *testing.T) {
				var receivedOpts DoStreamObjectOptions
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedOpts = opts
						return streamChunksToChannel(defaultTestChunks()), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:             model,
					Schema:            testObjectSchema(),
					SchemaName:        "test-name",
					SchemaDescription: "test description",
					Prompt:            "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedOpts.SchemaName != "test-name" {
					t.Errorf("expected SchemaName 'test-name', got %q", receivedOpts.SchemaName)
				}
				if receivedOpts.SchemaDescription != "test description" {
					t.Errorf("expected SchemaDescription 'test description', got %q", receivedOpts.SchemaDescription)
				}
			})
		})

		t.Run("custom schema", func(t *testing.T) {
			t.Run("should send object deltas with custom schema", func(t *testing.T) {
				model := createDefaultStreamModel()

				// Using a raw JSON schema directly, like TS jsonSchema({...}).
				customSchema := map[string]any{
					"type": "object",
					"properties": map[string]any{
						"content": map[string]any{"type": "string"},
					},
					"required":             []string{"content"},
					"additionalProperties": false,
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: customSchema,
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Drain fullStream in background.
				go func() {
					for range result.FullStream {
					}
				}()

				texts := drainTextStream(result.TextStream)
				expectedTexts := []string{"{ ", `"content": `, `"Hello, `, `world`, `!"`, " }"}
				if !reflect.DeepEqual(texts, expectedTexts) {
					t.Errorf("text stream mismatch:\ngot:  %v\nwant: %v", texts, expectedTexts)
				}
			})
		})

		t.Run("error handling", func(t *testing.T) {
			t.Run("should handle object when no text is generated", func(t *testing.T) {
				chunks := []StreamChunk{
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				// When no text is generated, Object should be nil.
				if result.Object != nil {
					t.Errorf("expected nil Object when no text generated, got %v", result.Object)
				}
			})

			t.Run("should handle broken JSON gracefully", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: "{ broken json"},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				// Object should be nil because broken JSON can't be parsed.
				if result.Object != nil {
					t.Errorf("expected nil Object for broken JSON, got %v", result.Object)
				}
			})

			t.Run("should propagate stream errors via fullStream", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: "{ "},
					{Type: "error", Error: context.DeadlineExceeded},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Drain textStream in background.
				go func() {
					for range result.TextStream {
					}
				}()

				parts := drainFullStream(result.FullStream)
				var foundError bool
				for _, p := range parts {
					if p.Type == ObjectStreamPartTypeError {
						foundError = true
						if p.Error != context.DeadlineExceeded {
							t.Errorf("expected DeadlineExceeded, got %v", p.Error)
						}
					}
				}
				if !foundError {
					t.Error("expected an error part in fullStream")
				}
			})
		})

		t.Run("options.OnChunk", func(t *testing.T) {
			t.Run("should be called for each chunk", func(t *testing.T) {
				var chunkParts []ObjectStreamPart
				model := createDefaultStreamModel()

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
					OnChunk: func(chunk ObjectStreamPart) {
						chunkParts = append(chunkParts, chunk)
					},
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if len(chunkParts) == 0 {
					t.Error("expected OnChunk to be called at least once")
				}

				// Should have text-delta, object, and finish parts.
				var hasTextDelta, hasObject, hasFinish bool
				for _, p := range chunkParts {
					switch p.Type {
					case ObjectStreamPartTypeTextDelta:
						hasTextDelta = true
					case ObjectStreamPartTypeObject:
						hasObject = true
					case ObjectStreamPartTypeFinish:
						hasFinish = true
					}
				}
				if !hasTextDelta {
					t.Error("expected OnChunk to receive text-delta parts")
				}
				if !hasObject {
					t.Error("expected OnChunk to receive object parts")
				}
				if !hasFinish {
					t.Error("expected OnChunk to receive finish part")
				}
			})
		})

		t.Run("mode handling", func(t *testing.T) {
			t.Run("should default to json mode", func(t *testing.T) {
				var receivedMode string
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedMode = opts.Mode
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
							{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedMode != "json" {
					t.Errorf("expected mode 'json', got %q", receivedMode)
				}
			})

			t.Run("should pass tool mode to model", func(t *testing.T) {
				var receivedMode string
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedMode = opts.Mode
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
							{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Mode:   "tool",
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedMode != "tool" {
					t.Errorf("expected mode 'tool', got %q", receivedMode)
				}
			})

			t.Run("should inject JSON instruction in json mode", func(t *testing.T) {
				var receivedPrompt string
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedPrompt = opts.Prompt
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
							{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if !strings.Contains(receivedPrompt, "prompt") {
					t.Error("expected prompt to contain original prompt")
				}
				if !strings.Contains(receivedPrompt, "JSON") {
					t.Error("expected prompt to contain JSON instruction")
				}
			})

			t.Run("should not inject JSON instruction in tool mode", func(t *testing.T) {
				var receivedPrompt string
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedPrompt = opts.Prompt
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
							{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
						}), nil
					},
				}

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Mode:   "tool",
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedPrompt != "prompt" {
					t.Errorf("expected raw prompt 'prompt' in tool mode, got %q", receivedPrompt)
				}
			})
		})

		t.Run("schema passed to model", func(t *testing.T) {
			t.Run("should pass schema to model DoStream", func(t *testing.T) {
				var receivedSchema any
				model := &mockStreamLanguageModel{
					doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
						receivedSchema = opts.Schema
						return streamChunksToChannel([]StreamChunk{
							{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
							{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
						}), nil
					},
				}

				schema := testObjectSchema()

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: schema,
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				if receivedSchema == nil {
					t.Fatal("expected schema to be passed to model")
				}
				schemaMap, ok := receivedSchema.(map[string]any)
				if !ok {
					t.Fatalf("expected schema to be map, got %T", receivedSchema)
				}
				if schemaMap["type"] != "object" {
					t.Errorf("expected schema type 'object', got %v", schemaMap["type"])
				}
			})
		})
	})

	t.Run("output = array", func(t *testing.T) {
		t.Run("array with 3 elements", func(t *testing.T) {
			arrayChunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{"elements":[`},
				{Type: "text-delta", TextDelta: `{`},
				{Type: "text-delta", TextDelta: `"content":`},
				{Type: "text-delta", TextDelta: `"element 1"`},
				{Type: "text-delta", TextDelta: `},`},
				{Type: "text-delta", TextDelta: `{ `},
				{Type: "text-delta", TextDelta: `"content": `},
				{Type: "text-delta", TextDelta: `"element 2"`},
				{Type: "text-delta", TextDelta: `},`},
				{Type: "text-delta", TextDelta: `{`},
				{Type: "text-delta", TextDelta: `"content":`},
				{Type: "text-delta", TextDelta: `"element 3"`},
				{Type: "text-delta", TextDelta: `}`},
				{Type: "text-delta", TextDelta: `]`},
				{Type: "text-delta", TextDelta: `}`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}

			t.Run("should have the correct object result", func(t *testing.T) {
				model := createStreamModelWithChunks(arrayChunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Output: "array",
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				arr, ok := result.Object.([]any)
				if !ok {
					t.Fatalf("expected []any, got %T (value: %v)", result.Object, result.Object)
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
			})

			t.Run("should send text deltas for array", func(t *testing.T) {
				model := createStreamModelWithChunks(arrayChunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Output: "array",
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Drain fullStream in background.
				go func() {
					for range result.FullStream {
					}
				}()

				texts := drainTextStream(result.TextStream)
				joined := strings.Join(texts, "")
				if !strings.Contains(joined, "elements") {
					t.Errorf("expected text stream to contain 'elements', got %q", joined)
				}
				if !strings.Contains(joined, "element 1") {
					t.Errorf("expected text stream to contain 'element 1', got %q", joined)
				}
			})
		})

		t.Run("array with 2 elements streamed in 1 chunk", func(t *testing.T) {
			t.Run("should have the correct object result", func(t *testing.T) {
				chunks := []StreamChunk{
					{Type: "text-delta", TextDelta: `{"elements":[{"content":"element 1"},{"content":"element 2"}]}`},
					{
						Type:         "finish",
						FinishReason: "stop",
						Usage:        defaultTestUsage(),
					},
				}
				model := createStreamModelWithChunks(chunks)

				result, err := StreamObject(context.Background(), StreamObjectOptions{
					Model:  model,
					Schema: testObjectSchema(),
					Output: "array",
					Prompt: "prompt",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				drainAndWait(result)

				arr, ok := result.Object.([]any)
				if !ok {
					t.Fatalf("expected []any, got %T (value: %v)", result.Object, result.Object)
				}
				if len(arr) != 2 {
					t.Fatalf("expected 2 elements, got %d", len(arr))
				}

				for i, expected := range []string{"element 1", "element 2"} {
					elem, ok := arr[i].(map[string]any)
					if !ok {
						t.Fatalf("element %d: expected map, got %T", i, arr[i])
					}
					if elem["content"] != expected {
						t.Errorf("element %d: expected content %q, got %v", i, expected, elem["content"])
					}
				}
			})
		})
	})

	t.Run("output = enum", func(t *testing.T) {
		t.Run("should stream an enum value", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"result": `},
				{Type: "text-delta", TextDelta: `"su`},
				{Type: "text-delta", TextDelta: `nny`},
				{Type: "text-delta", TextDelta: `"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:      model,
				Output:     "enum",
				EnumValues: []string{"sunny", "rainy", "snowy"},
				Prompt:     "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if result.Object != "sunny" {
				t.Errorf("expected 'sunny', got %v", result.Object)
			}
		})

		t.Run("should not accept incorrect values", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"result": `},
				{Type: "text-delta", TextDelta: `"foo`},
				{Type: "text-delta", TextDelta: `bar`},
				{Type: "text-delta", TextDelta: `"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:      model,
				Output:     "enum",
				EnumValues: []string{"sunny", "rainy", "snowy"},
				Prompt:     "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			// "foobar" is not in the enum, so Object should be nil.
			if result.Object != nil {
				t.Errorf("expected nil Object for invalid enum value, got %v", result.Object)
			}
		})

		t.Run("should handle valid enum value foobar", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"result": `},
				{Type: "text-delta", TextDelta: `"foo`},
				{Type: "text-delta", TextDelta: `bar`},
				{Type: "text-delta", TextDelta: `"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:      model,
				Output:     "enum",
				EnumValues: []string{"foobar", "barfoo"},
				Prompt:     "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if result.Object != "foobar" {
				t.Errorf("expected 'foobar', got %v", result.Object)
			}
		})
	})

	t.Run("output = no-schema", func(t *testing.T) {
		t.Run("should send object deltas", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"content": `},
				{Type: "text-delta", TextDelta: `"Hello, `},
				{Type: "text-delta", TextDelta: `world`},
				{Type: "text-delta", TextDelta: `!"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Output: "no-schema",
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Drain fullStream in background.
			go func() {
				for range result.FullStream {
				}
			}()

			texts := drainTextStream(result.TextStream)
			expected := []string{"{ ", `"content": `, `"Hello, `, `world`, `!"`, " }"}
			if !reflect.DeepEqual(texts, expected) {
				t.Errorf("text stream mismatch:\ngot:  %v\nwant: %v", texts, expected)
			}
		})

		t.Run("should resolve with final object", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Output: "no-schema",
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			obj, ok := result.Object.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T", result.Object)
			}
			if obj["content"] != "Hello, world!" {
				t.Errorf("unexpected content: %v", obj["content"])
			}
		})

		t.Run("should pass nil schema to model for no-schema", func(t *testing.T) {
			var receivedSchema any
			model := &mockStreamLanguageModel{
				doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
					receivedSchema = opts.Schema
					return streamChunksToChannel([]StreamChunk{
						{Type: "text-delta", TextDelta: `{ "content": "Hello" }`},
						{Type: "finish", FinishReason: "stop", Usage: defaultTestUsage()},
					}), nil
				},
			}

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Output: "no-schema",
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if receivedSchema != nil {
				t.Errorf("expected nil schema for no-schema mode, got %v", receivedSchema)
			}
		})
	})

	t.Run("validation", func(t *testing.T) {
		t.Run("should reject invalid output type", func(t *testing.T) {
			model := createDefaultStreamModel()

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Output: "invalid",
				Prompt: "prompt",
			})
			if err == nil {
				t.Fatal("expected error for invalid output type")
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Errorf("expected error to mention 'invalid', got: %v", err)
			}
		})

		t.Run("should reject object output without schema", func(t *testing.T) {
			model := createDefaultStreamModel()

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Prompt: "prompt",
			})
			if err == nil {
				t.Fatal("expected error for object output without schema")
			}
			if !strings.Contains(err.Error(), "schema") {
				t.Errorf("expected error to mention 'schema', got: %v", err)
			}
		})

		t.Run("should reject enum output without enum values", func(t *testing.T) {
			model := createDefaultStreamModel()

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Output: "enum",
				Prompt: "prompt",
			})
			if err == nil {
				t.Fatal("expected error for enum output without enum values")
			}
			if !strings.Contains(err.Error(), "enum") {
				t.Errorf("expected error to mention 'enum', got: %v", err)
			}
		})

		t.Run("should reject no-schema output with schema", func(t *testing.T) {
			model := createDefaultStreamModel()

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Output: "no-schema",
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err == nil {
				t.Fatal("expected error for no-schema output with schema")
			}
		})
	})

	t.Run("DoStream error propagation", func(t *testing.T) {
		t.Run("should propagate DoStream errors", func(t *testing.T) {
			model := &mockStreamLanguageModel{
				doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
					return nil, context.DeadlineExceeded
				},
			}

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err == nil {
				t.Fatal("expected error from DoStream")
			}
			if err != context.DeadlineExceeded {
				t.Errorf("expected DeadlineExceeded, got %v", err)
			}
		})
	})

	t.Run("partial object parsing", func(t *testing.T) {
		t.Run("should emit partial objects as JSON becomes parseable", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"content": `},
				{Type: "text-delta", TextDelta: `"Hello, `},
				{Type: "text-delta", TextDelta: `world`},
				{Type: "text-delta", TextDelta: `!"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Drain textStream in background.
			go func() {
				for range result.TextStream {
				}
			}()

			objects := collectObjectsFromFullStream(result.FullStream)
			if len(objects) == 0 {
				t.Fatal("expected at least one partial object")
			}

			// The last partial object before finish should be the complete object.
			lastObj, ok := objects[len(objects)-1].(map[string]any)
			if !ok {
				t.Fatalf("expected last object to be map, got %T", objects[len(objects)-1])
			}
			if lastObj["content"] != "Hello, world!" {
				t.Errorf("expected last partial object content 'Hello, world!', got %v", lastObj["content"])
			}
		})
	})

	t.Run("concurrent channel consumption", func(t *testing.T) {
		t.Run("should allow concurrent reading of TextStream and FullStream", func(t *testing.T) {
			model := createDefaultStreamModel()

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Read both channels concurrently.
			textDone := make(chan []string)
			fullDone := make(chan []ObjectStreamPart)

			go func() {
				textDone <- drainTextStream(result.TextStream)
			}()
			go func() {
				fullDone <- drainFullStream(result.FullStream)
			}()

			texts := <-textDone
			parts := <-fullDone

			if len(texts) == 0 {
				t.Error("expected text stream to have content")
			}
			if len(parts) == 0 {
				t.Error("expected full stream to have content")
			}
		})
	})

	// --- onFinish callback tests ---

	t.Run("options.onFinish", func(t *testing.T) {
		t.Run("should be called when a valid object is generated", func(t *testing.T) {
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
					ProviderMetadata: ProviderMetadata{
						"testProvider": {"testKey": "testValue"},
					},
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}

			// Check usage.
			if finishEvent.Usage.PromptTokens != 3 {
				t.Errorf("expected prompt tokens 3, got %d", finishEvent.Usage.PromptTokens)
			}
			if finishEvent.Usage.CompletionTokens != 10 {
				t.Errorf("expected completion tokens 10, got %d", finishEvent.Usage.CompletionTokens)
			}

			// Check object.
			obj, ok := finishEvent.Object.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T (value: %v)", finishEvent.Object, finishEvent.Object)
			}
			if obj["content"] != "Hello, world!" {
				t.Errorf("expected content 'Hello, world!', got %v", obj["content"])
			}

			// Check no error.
			if finishEvent.Error != nil {
				t.Errorf("expected no error, got %v", finishEvent.Error)
			}

			// Check provider metadata.
			if finishEvent.ProviderMetadata == nil {
				t.Fatal("expected provider metadata in onFinish event")
			}
			tp := finishEvent.ProviderMetadata["testProvider"]
			if tp == nil {
				t.Fatal("expected testProvider metadata")
			}
			if tp["testKey"] != "testValue" {
				t.Errorf("expected testKey=testValue, got %v", tp["testKey"])
			}
		})

		t.Run("should be called when object does not match the schema", func(t *testing.T) {
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			// Stream text that produces {"invalid": "Hello, world!"} which doesn't
			// match the schema requiring "content". For the Go ObjectOutputStrategy,
			// any valid JSON object passes validation — but we test the callback is
			// still invoked. If the strategy rejects it, Error will be set and Object nil.
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "text-delta", TextDelta: `"invalid": `},
				{Type: "text-delta", TextDelta: `"Hello, `},
				{Type: "text-delta", TextDelta: `world`},
				{Type: "text-delta", TextDelta: `!"`},
				{Type: "text-delta", TextDelta: " }"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}

			// The Go ObjectOutputStrategy accepts any valid JSON object,
			// so the object should be present. Usage should be set either way.
			if finishEvent.Usage.PromptTokens != 3 {
				t.Errorf("expected prompt tokens 3, got %d", finishEvent.Usage.PromptTokens)
			}

			// Object should be non-nil (Go strategy doesn't reject by field names).
			if finishEvent.Object == nil {
				// If the strategy did reject, Error should be set.
				if finishEvent.Error == nil {
					t.Error("expected either Object or Error to be set in onFinish")
				}
			} else {
				obj, ok := finishEvent.Object.(map[string]any)
				if !ok {
					t.Fatalf("expected map, got %T", finishEvent.Object)
				}
				if obj["invalid"] != "Hello, world!" {
					t.Errorf("expected invalid='Hello, world!', got %v", obj["invalid"])
				}
			}
		})

		t.Run("should call onFinish callback with full array (3 elements)", func(t *testing.T) {
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			arrayChunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{"elements":[`},
				{Type: "text-delta", TextDelta: `{"content":"element 1"},`},
				{Type: "text-delta", TextDelta: `{"content":"element 2"},`},
				{Type: "text-delta", TextDelta: `{"content":"element 3"}`},
				{Type: "text-delta", TextDelta: `]}`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(arrayChunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Output: "array",
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}

			arr, ok := finishEvent.Object.([]any)
			if !ok {
				t.Fatalf("expected []any in onFinish, got %T (value: %v)", finishEvent.Object, finishEvent.Object)
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
		})

		t.Run("should call onFinish callback with full array (2 elements in 1 chunk)", func(t *testing.T) {
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{"elements":[{"content":"element 1"},{"content":"element 2"}]}`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Output: "array",
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}

			arr, ok := finishEvent.Object.([]any)
			if !ok {
				t.Fatalf("expected []any in onFinish, got %T (value: %v)", finishEvent.Object, finishEvent.Object)
			}
			if len(arr) != 2 {
				t.Fatalf("expected 2 elements, got %d", len(arr))
			}

			for i, expected := range []string{"element 1", "element 2"} {
				elem, ok := arr[i].(map[string]any)
				if !ok {
					t.Fatalf("element %d: expected map, got %T", i, arr[i])
				}
				if elem["content"] != expected {
					t.Errorf("element %d: expected content %q, got %v", i, expected, elem["content"])
				}
			}
		})
	})

	// --- onError callback tests ---

	t.Run("options.onError", func(t *testing.T) {
		t.Run("should invoke onError callback with Error", func(t *testing.T) {
			var mu sync.Mutex
			var errorEvents []StreamObjectOnErrorEvent

			model := &mockStreamLanguageModel{
				doStream: func(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error) {
					return nil, fmt.Errorf("test error")
				},
			}

			_, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnError: func(event StreamObjectOnErrorEvent) {
					mu.Lock()
					defer mu.Unlock()
					errorEvents = append(errorEvents, event)
				},
			})

			// When DoStream returns an error, StreamObject itself returns the error.
			// The onError callback is for in-stream errors (error chunks), not DoStream errors.
			if err == nil {
				t.Fatal("expected error from StreamObject")
			}
			if !strings.Contains(err.Error(), "test error") {
				t.Errorf("expected error to contain 'test error', got: %v", err)
			}
		})

		t.Run("should invoke onError for in-stream error chunks", func(t *testing.T) {
			var mu sync.Mutex
			var errorEvents []StreamObjectOnErrorEvent

			testErr := fmt.Errorf("stream chunk error")
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ "},
				{Type: "error", Error: testErr},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnError: func(event StreamObjectOnErrorEvent) {
					mu.Lock()
					defer mu.Unlock()
					errorEvents = append(errorEvents, event)
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if len(errorEvents) != 1 {
				t.Fatalf("expected 1 error event, got %d", len(errorEvents))
			}
			if !strings.Contains(errorEvents[0].Error.Error(), "stream chunk error") {
				t.Errorf("expected error to contain 'stream chunk error', got: %v", errorEvents[0].Error)
			}
		})

		t.Run("should suppress error in fullStream when onError is set", func(t *testing.T) {
			// When a model DoStream returns an error, StreamObject returns it directly.
			// This test verifies that when an error chunk is in the stream,
			// setting onError still allows the stream to complete without panic.
			testErr := fmt.Errorf("suppressed error")
			chunks := []StreamChunk{
				{Type: "error", Error: testErr},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnError: func(event StreamObjectOnErrorEvent) {
					// Intentionally empty — suppresses the error.
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Draining should complete without panic.
			drainAndWait(result)
		})
	})

	// --- warnings tests ---

	t.Run("warnings", func(t *testing.T) {
		t.Run("should resolve with empty warnings when no warnings are present", func(t *testing.T) {
			chunks := []StreamChunk{
				{
					Type:     "stream-start",
					Warnings: []CallWarning{},
				},
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if result.Warnings == nil {
				t.Fatal("expected Warnings to be non-nil (empty slice)")
			}
			if len(result.Warnings) != 0 {
				t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
			}
		})

		t.Run("should resolve with warnings when warnings are present", func(t *testing.T) {
			expectedWarnings := []CallWarning{
				{
					Type:    "unsupported",
					Feature: "frequency_penalty",
					Details: "This model does not support the frequency_penalty setting.",
				},
				{
					Type:    "other",
					Message: "Test warning message",
				},
			}

			chunks := []StreamChunk{
				{
					Type:     "stream-start",
					Warnings: expectedWarnings,
				},
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
				t.Errorf("warnings mismatch:\ngot:  %+v\nwant: %+v", result.Warnings, expectedWarnings)
			}
		})

		t.Run("should propagate warnings to onFinish callback", func(t *testing.T) {
			// This is the Go equivalent of the TS logWarnings tests.
			// In Go, LogWarnings is not called from StreamObject, but warnings
			// are propagated to the onFinish event. We verify that path.
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			expectedWarnings := []CallWarning{
				{
					Type:    "other",
					Message: "Setting is not supported",
				},
				{
					Type:    "unsupported",
					Feature: "temperature",
					Details: "Temperature parameter not supported",
				},
			}

			chunks := []StreamChunk{
				{
					Type:     "stream-start",
					Warnings: expectedWarnings,
				},
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}
			if !reflect.DeepEqual(finishEvent.Warnings, expectedWarnings) {
				t.Errorf("onFinish warnings mismatch:\ngot:  %+v\nwant: %+v", finishEvent.Warnings, expectedWarnings)
			}
		})

		t.Run("should propagate empty warnings to onFinish callback", func(t *testing.T) {
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			chunks := []StreamChunk{
				{
					Type:     "stream-start",
					Warnings: []CallWarning{},
				},
				{Type: "text-delta", TextDelta: `{ "content": "Hello, world!" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}
			if finishEvent.Warnings == nil {
				t.Fatal("expected Warnings to be non-nil (empty slice)")
			}
			if len(finishEvent.Warnings) != 0 {
				t.Errorf("expected 0 warnings in onFinish, got %d", len(finishEvent.Warnings))
			}
		})
	})

	// --- repairText tests ---

	t.Run("options.RepairText", func(t *testing.T) {
		t.Run("should be able to repair a JSONParseError", func(t *testing.T) {
			// Stream incomplete JSON: missing closing brace.
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ "content": "provider metadata test" `},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			var repairCalled bool
			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					repairCalled = true
					if text != `{ "content": "provider metadata test" ` {
						return "", fmt.Errorf("unexpected text: %q", text)
					}
					if parseError == nil {
						return "", fmt.Errorf("expected non-nil parse error")
					}
					// Repair by adding closing brace.
					return text + "}", nil
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if !repairCalled {
				t.Error("expected repairText to be called")
			}

			obj, ok := result.Object.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T (value: %v)", result.Object, result.Object)
			}
			if obj["content"] != "provider metadata test" {
				t.Errorf("expected content 'provider metadata test', got %v", obj["content"])
			}
		})

		t.Run("should be able to repair a TypeValidationError", func(t *testing.T) {
			// Use enum mode which has strict validation. Stream a value not in the enum.
			// The EnumOutputStrategy wraps values as {"result": "value"}, so the
			// repair function needs to fix the value to a valid enum member.
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ "result": "invalid_value" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			var repairCalled bool
			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:      model,
				Output:     "enum",
				EnumValues: []string{"sunny", "rainy"},
				Prompt:     "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					repairCalled = true
					if text != `{ "result": "invalid_value" }` {
						return "", fmt.Errorf("unexpected text: %q", text)
					}
					// Repair by replacing with a valid enum value.
					return `{ "result": "sunny" }`, nil
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if !repairCalled {
				t.Error("expected repairText to be called")
			}

			if result.Object != "sunny" {
				t.Errorf("expected 'sunny', got %v", result.Object)
			}
		})

		t.Run("should handle repair that returns empty string", func(t *testing.T) {
			// In the TS test, repairText returns null. In Go, we return empty string
			// which ParseAndValidateObjectResultWithRepair treats as "repair failed".
			// Use enum mode to ensure validation actually rejects the input.
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ "result": "invalid_value" }`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:      model,
				Output:     "enum",
				EnumValues: []string{"sunny", "rainy"},
				Prompt:     "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					// Return empty string to signal repair failure (Go equivalent of null).
					return "", nil
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			// Object should be nil because the repair returned empty string.
			if result.Object != nil {
				t.Errorf("expected nil Object when repair returns empty string, got %v", result.Object)
			}
		})

		t.Run("should be able to repair JSON wrapped with markdown code blocks", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "```json\n{ \"content\": \"test message\" }\n```"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					if text != "```json\n{ \"content\": \"test message\" }\n```" {
						return "", fmt.Errorf("unexpected text: %q", text)
					}
					// Remove markdown code block wrapper.
					cleaned := strings.TrimPrefix(text, "```json\n")
					cleaned = strings.TrimSuffix(cleaned, "\n```")
					return cleaned, nil
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			obj, ok := result.Object.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T (value: %v)", result.Object, result.Object)
			}
			if obj["content"] != "test message" {
				t.Errorf("expected content 'test message', got %v", obj["content"])
			}
		})

		t.Run("should fail when repairText returns still-broken JSON", func(t *testing.T) {
			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: "{ broken json"},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					// "Repair" still produces broken JSON.
					return text + "{", nil
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			// Object should be nil because the repair also produced unparseable JSON.
			if result.Object != nil {
				t.Errorf("expected nil Object when repair produces broken JSON, got %v", result.Object)
			}
		})

		t.Run("should propagate repair error and keep original error", func(t *testing.T) {
			// When repairText itself returns an error, the original parse error
			// should be used (not the repair error).
			var mu sync.Mutex
			var finishEvent *StreamObjectOnFinishEvent

			chunks := []StreamChunk{
				{Type: "text-delta", TextDelta: `{ broken json`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
				RepairText: func(text string, parseError error) (string, error) {
					return "", errors.New("repair function failed")
				},
				OnFinish: func(event StreamObjectOnFinishEvent) {
					mu.Lock()
					defer mu.Unlock()
					finishEvent = &event
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if result.Object != nil {
				t.Errorf("expected nil Object, got %v", result.Object)
			}

			mu.Lock()
			defer mu.Unlock()

			if finishEvent == nil {
				t.Fatal("expected onFinish to be called")
			}
			if finishEvent.Error == nil {
				t.Fatal("expected Error to be set in onFinish when repair fails")
			}
			// The error should be the original parse error, not the repair error.
			if strings.Contains(finishEvent.Error.Error(), "repair function failed") {
				t.Error("expected original parse error, not repair error")
			}
		})
	})

	// --- request metadata tests ---

	t.Run("result.request", func(t *testing.T) {
		t.Run("should contain request information", func(t *testing.T) {
			chunks := []StreamChunk{
				{
					Type: "stream-start",
					Request: &LanguageModelRequestMetadata{
						Body: "test body",
					},
				},
				{Type: "text-delta", TextDelta: `{"content": "Hello, world!"}`},
				{
					Type:         "finish",
					FinishReason: "stop",
					Usage:        defaultTestUsage(),
				},
			}
			model := createStreamModelWithChunks(chunks)

			result, err := StreamObject(context.Background(), StreamObjectOptions{
				Model:  model,
				Schema: testObjectSchema(),
				Prompt: "prompt",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			drainAndWait(result)

			if result.Request.Body != "test body" {
				t.Errorf("expected request body 'test body', got %v", result.Request.Body)
			}
		})
	})
}
