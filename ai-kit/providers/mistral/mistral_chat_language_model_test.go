// Ported from: packages/mistral/src/mistral-chat-language-model.test.ts
package mistral

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// testPrompt is the standard test prompt used across tests.
var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

// mockIDCounter creates a deterministic ID generator for tests.
func mockIDCounter() func() string {
	counter := 0
	return func() string {
		counter++
		return fmt.Sprintf("id-%d", counter)
	}
}

// strPtr returns a pointer to a string.
func strPtr(s string) *string { return &s }

// boolPtr returns a pointer to a bool.
func boolPtr(b bool) *bool { return &b }

// requestCapture captures HTTP request data for assertions.
type requestCapture struct {
	Body    []byte
	Headers http.Header
}

func (rc *requestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

// createTestServer creates an httptest server that serves the given JSON body.
func createTestServer(body any, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")

		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

// createSSETestServer creates an httptest server that serves SSE chunks.
func createSSETestServer(chunks []string, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		for k, v := range headers {
			w.Header().Set(k, v)
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprint(w, chunk)
			flusher.Flush()
		}
	}))
	return server, capture
}

// createModel creates a ChatLanguageModel that targets a test server.
func createModel(baseURL string) *ChatLanguageModel {
	return NewChatLanguageModel("mistral-small-latest", ChatConfig{
		Provider: "mistral.chat",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return map[string]string{
				"authorization": "Bearer test-api-key",
				"content-type":  "application/json",
			}
		},
		GenerateID: mockIDCounter(),
	})
}

// createModelWithHeaders creates a ChatLanguageModel with custom headers.
func createModelWithHeaders(baseURL string, headers map[string]string) *ChatLanguageModel {
	return NewChatLanguageModel("mistral-small-latest", ChatConfig{
		Provider: "mistral.chat",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return headers
		},
		GenerateID: mockIDCounter(),
	})
}

// collectStreamParts drains a stream channel into a slice.
func collectStreamParts(stream <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for part := range stream {
		parts = append(parts, part)
	}
	return parts
}

// --- Fixtures ---

func mistralTextFixture() map[string]any {
	return map[string]any{
		"id":      "5319bd0299614c679a0068a4f2c8ffd0",
		"created": float64(1769088720),
		"model":   "mistral-small-latest",
		"usage": map[string]any{
			"prompt_tokens":     float64(13),
			"total_tokens":      float64(447),
			"completion_tokens": float64(434),
		},
		"object": "chat.completion",
		"choices": []any{
			map[string]any{
				"index":         float64(0),
				"finish_reason": "stop",
				"message": map[string]any{
					"role":       "assistant",
					"tool_calls": nil,
					"content":    "Hello, world! This is a test response.",
				},
			},
		},
	}
}

func mistralToolCallFixture() map[string]any {
	return map[string]any{
		"id":      "b3999b8c93e04e11bcbff7bcab829667",
		"created": float64(1769088854),
		"model":   "mistral-small-latest",
		"usage": map[string]any{
			"prompt_tokens":     float64(124),
			"total_tokens":      float64(146),
			"completion_tokens": float64(22),
		},
		"object": "chat.completion",
		"choices": []any{
			map[string]any{
				"index":         float64(0),
				"finish_reason": "tool_calls",
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{
						map[string]any{
							"id": "gSIMJiOkT",
							"function": map[string]any{
								"name":      "weather",
								"arguments": `{"location": "San Francisco"}`,
							},
						},
					},
				},
			},
		},
	}
}

func mistralReasoningFixture() map[string]any {
	return map[string]any{
		"id":      "a4e29c5b82f94d67b23e108a7c9df6e1",
		"created": float64(1769088912),
		"model":   "magistral-medium-2507",
		"usage": map[string]any{
			"prompt_tokens":     float64(10),
			"total_tokens":      float64(56),
			"completion_tokens": float64(46),
		},
		"object": "chat.completion",
		"choices": []any{
			map[string]any{
				"index":         float64(0),
				"finish_reason": "stop",
				"message": map[string]any{
					"role": "assistant",
					"content": []any{
						map[string]any{
							"type": "thinking",
							"thinking": []any{
								map[string]any{
									"type": "text",
									"text": "The user is asking for 2+2. This is basic arithmetic. 2+2=4.",
								},
							},
						},
						map[string]any{
							"type": "text",
							"text": "2 + 2 = 4",
						},
					},
				},
			},
		},
	}
}

func mistralTextStreamChunks() []string {
	return []string{
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\", \"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"world!\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" This\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" is a test\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" response.\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"5319bd0299614c679a0068a4f2c8ffd0\",\"object\":\"chat.completion.chunk\",\"created\":1769088720,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"},\"finish_reason\":\"stop\",\"logprobs\":null}],\"usage\":{\"prompt_tokens\":13,\"total_tokens\":21,\"completion_tokens\":8}}\n\n",
		"data: [DONE]\n\n",
	}
}

func mistralToolCallStreamChunks() []string {
	return []string{
		"data: {\"id\":\"b3999b8c93e04e11bcbff7bcab829667\",\"object\":\"chat.completion.chunk\",\"created\":1769088854,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
		"data: {\"id\":\"b3999b8c93e04e11bcbff7bcab829667\",\"object\":\"chat.completion.chunk\",\"created\":1769088854,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":null,\"tool_calls\":[{\"id\":\"gSIMJiOkT\",\"function\":{\"name\":\"weather\",\"arguments\":\"{\\\"location\\\": \\\"San Francisco\\\"}\"}}]},\"finish_reason\":\"tool_calls\",\"logprobs\":null}],\"usage\":{\"prompt_tokens\":124,\"total_tokens\":146,\"completion_tokens\":22}}\n\n",
		"data: [DONE]\n\n",
	}
}

func mistralReasoningStreamChunks() []string {
	return []string{
		"data: {\"id\":\"a4e29c5b82f94d67b23e108a7c9df6e1\",\"object\":\"chat.completion.chunk\",\"created\":1769088912,\"model\":\"magistral-medium-2507\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"thinking\",\"thinking\":[{\"type\":\"text\",\"text\":\"The user is asking\"}]}]},\"finish_reason\":null}]}\n\n",
		"data: {\"id\":\"a4e29c5b82f94d67b23e108a7c9df6e1\",\"object\":\"chat.completion.chunk\",\"created\":1769088912,\"model\":\"magistral-medium-2507\",\"choices\":[{\"index\":0,\"delta\":{\"content\":[{\"type\":\"thinking\",\"thinking\":[{\"type\":\"text\",\"text\":\" for 2+2. This is basic arithmetic. 2+2=4.\"}]}]},\"finish_reason\":null}]}\n\n",
		"data: {\"id\":\"a4e29c5b82f94d67b23e108a7c9df6e1\",\"object\":\"chat.completion.chunk\",\"created\":1769088912,\"model\":\"magistral-medium-2507\",\"choices\":[{\"index\":0,\"delta\":{\"content\":[{\"type\":\"text\",\"text\":\"2 + 2 = 4\"}]},\"finish_reason\":null}]}\n\n",
		"data: {\"id\":\"a4e29c5b82f94d67b23e108a7c9df6e1\",\"object\":\"chat.completion.chunk\",\"created\":1769088912,\"model\":\"magistral-medium-2507\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"total_tokens\":56,\"completion_tokens\":46}}\n\n",
		"data: [DONE]\n\n",
	}
}

// ===== DoGenerate tests =====

func TestDoGenerate_Text(t *testing.T) {
	t.Run("should extract text content", func(t *testing.T) {
		server, _ := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) == 0 {
			t.Fatal("expected non-empty content")
		}

		textContent, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if textContent.Text != "Hello, world! This is a test response." {
			t.Errorf("unexpected text content: %q", textContent.Text)
		}
	})

	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "mistral-small-latest" {
			t.Errorf("expected model 'mistral-small-latest', got %v", body["model"])
		}

		messages, ok := body["messages"].([]any)
		if !ok {
			t.Fatalf("expected messages to be []any, got %T", body["messages"])
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
	})
}

func TestDoGenerate_ToolCall(t *testing.T) {
	t.Run("should extract tool call content", func(t *testing.T) {
		server, _ := createTestServer(mistralToolCallFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Find tool call content
		var toolCall *languagemodel.ToolCall
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}
		if toolCall == nil {
			t.Fatal("expected tool call content")
		}
		if toolCall.ToolCallID != "gSIMJiOkT" {
			t.Errorf("expected tool call ID 'gSIMJiOkT', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "weather" {
			t.Errorf("expected tool name 'weather', got %q", toolCall.ToolName)
		}
		if !strings.Contains(toolCall.Input, "San Francisco") {
			t.Errorf("expected input to contain 'San Francisco', got %q", toolCall.Input)
		}

		// Should have tool_calls finish reason
		if result.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected finish reason 'tool-calls', got %q", result.FinishReason.Unified)
		}
	})
}

func TestDoGenerate_Reasoning(t *testing.T) {
	t.Run("should extract reasoning content", func(t *testing.T) {
		server, _ := createTestServer(mistralReasoningFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(result.Content))
		}

		// First: reasoning
		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[0])
		}
		if !strings.Contains(reasoning.Text, "2+2") {
			t.Errorf("expected reasoning to contain '2+2', got %q", reasoning.Text)
		}

		// Second: text
		text, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[1])
		}
		if text.Text != "2 + 2 = 4" {
			t.Errorf("expected text '2 + 2 = 4', got %q", text.Text)
		}
	})
}

func TestDoGenerate_PassToolsAndToolChoice(t *testing.T) {
	t.Run("should pass tools and toolChoice", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"value": map[string]any{"type": "string"}},
						"required":             []any{"value"},
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "test-tool"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "mistral-small-latest" {
			t.Errorf("expected model 'mistral-small-latest', got %v", body["model"])
		}

		// Check tool_choice
		if body["tool_choice"] != "any" {
			t.Errorf("expected tool_choice 'any', got %v", body["tool_choice"])
		}

		// Check tools
		tools, ok := body["tools"].([]any)
		if !ok {
			t.Fatalf("expected tools to be []any, got %T", body["tools"])
		}
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		tool := tools[0].(map[string]any)
		if tool["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tool["type"])
		}
		fn := tool["function"].(map[string]any)
		if fn["name"] != "test-tool" {
			t.Errorf("expected function name 'test-tool', got %v", fn["name"])
		}
	})
}

func TestDoGenerate_PassHeaders(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModelWithHeaders(server.URL, map[string]string{
			"authorization":          "Bearer test-api-key",
			"content-type":           "application/json",
			"Custom-Provider-Header": "provider-header-value",
		})

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q",
				capture.Headers.Get("Authorization"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createTestServer(mistralTextFixture(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header 'test-value', got %q", result.Response.Headers["Test-Header"])
		}
	})
}

func TestDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Input tokens
		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 13 {
			t.Errorf("expected input total tokens 13, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 13 {
			t.Errorf("expected input noCache tokens 13, got %v", result.Usage.InputTokens.NoCache)
		}

		// Output tokens
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 434 {
			t.Errorf("expected output total tokens 434, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 434 {
			t.Errorf("expected output text tokens 434, got %v", result.Usage.OutputTokens.Text)
		}

		// Raw usage
		if result.Usage.Raw == nil {
			t.Fatal("expected non-nil raw usage")
		}
		if result.Usage.Raw["prompt_tokens"] != 13 {
			t.Errorf("expected raw prompt_tokens 13, got %v", result.Usage.Raw["prompt_tokens"])
		}
		if result.Usage.Raw["completion_tokens"] != 434 {
			t.Errorf("expected raw completion_tokens 434, got %v", result.Usage.Raw["completion_tokens"])
		}
		if result.Usage.Raw["total_tokens"] != 447 {
			t.Errorf("expected raw total_tokens 447, got %v", result.Usage.Raw["total_tokens"])
		}
	})
}

func TestDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should send additional response information", func(t *testing.T) {
		server, _ := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if result.Response.ID == nil || *result.Response.ID != "5319bd0299614c679a0068a4f2c8ffd0" {
			t.Errorf("expected ID '5319bd0299614c679a0068a4f2c8ffd0', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "mistral-small-latest" {
			t.Errorf("expected ModelID 'mistral-small-latest', got %v", result.Response.ModelID)
		}
		expectedTime := time.Unix(1769088720, 0)
		if result.Response.Timestamp == nil || !result.Response.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, result.Response.Timestamp)
		}
	})
}

func TestDoGenerate_ParallelToolCalls(t *testing.T) {
	t.Run("should pass parallelToolCalls option", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"value": map[string]any{"type": "string"}},
						"required":             []any{"value"},
						"additionalProperties": false,
						"$schema":              "http://json-schema.org/draft-07/schema#",
					},
				},
			},
			ProviderOptions: shared.ProviderOptions{
				"mistral": map[string]any{
					"parallelToolCalls": false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["parallel_tool_calls"] != false {
			t.Errorf("expected parallel_tool_calls false, got %v", body["parallel_tool_calls"])
		}
	})
}

func TestDoGenerate_TrailingAssistantMessage(t *testing.T) {
	t.Run("should avoid duplication when trailing assistant message", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "16362f24e60340d0994dd205c267a43a",
			"created": float64(1711113008),
			"model":   "mistral-small-latest",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":       "assistant",
						"content":    "prefix and more content",
						"tool_calls": nil,
					},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "prefix "},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "prefix and more content" {
			t.Errorf("expected text 'prefix and more content', got %q", text.Text)
		}
	})
}

func TestDoGenerate_MixedThinkingAndText(t *testing.T) {
	t.Run("should preserve ordering of mixed thinking and text", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "mixed-content-test",
			"object":  "chat.completion",
			"created": float64(1722349660),
			"model":   "magistral-medium-2507",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{
								"type": "thinking",
								"thinking": []any{
									map[string]any{"type": "text", "text": "First thought."},
								},
							},
							map[string]any{
								"type": "text",
								"text": "Partial answer.",
							},
							map[string]any{
								"type": "thinking",
								"thinking": []any{
									map[string]any{"type": "text", "text": "Second thought."},
								},
							},
							map[string]any{
								"type": "text",
								"text": "Final answer.",
							},
						},
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"total_tokens":      float64(30),
				"completion_tokens": float64(20),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 4 {
			t.Fatalf("expected 4 content parts, got %d", len(result.Content))
		}

		// First: reasoning
		r1, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning at index 0, got %T", result.Content[0])
		}
		if r1.Text != "First thought." {
			t.Errorf("expected 'First thought.', got %q", r1.Text)
		}

		// Second: text
		t1, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text at index 1, got %T", result.Content[1])
		}
		if t1.Text != "Partial answer." {
			t.Errorf("expected 'Partial answer.', got %q", t1.Text)
		}

		// Third: reasoning
		r2, ok := result.Content[2].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning at index 2, got %T", result.Content[2])
		}
		if r2.Text != "Second thought." {
			t.Errorf("expected 'Second thought.', got %q", r2.Text)
		}

		// Fourth: text
		t2, ok := result.Content[3].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text at index 3, got %T", result.Content[3])
		}
		if t2.Text != "Final answer." {
			t.Errorf("expected 'Final answer.', got %q", t2.Text)
		}
	})
}

func TestDoGenerate_EmptyThinkingContent(t *testing.T) {
	t.Run("should handle empty thinking content", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "empty-thinking-test",
			"object":  "chat.completion",
			"created": float64(1722349660),
			"model":   "magistral-medium-2507",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{
								"type":     "thinking",
								"thinking": []any{},
							},
							map[string]any{
								"type": "text",
								"text": "Just the answer.",
							},
						},
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"total_tokens":      float64(30),
				"completion_tokens": float64(20),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Just the answer." {
			t.Errorf("expected 'Just the answer.', got %q", text.Text)
		}
	})
}

func TestDoGenerate_ContentObjectArray(t *testing.T) {
	t.Run("should extract content when message content is a content object", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "object-id",
			"created": float64(1711113008),
			"model":   "mistral-small-latest",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{
								"type": "text",
								"text": "Hello from object",
							},
						},
						"tool_calls": nil,
					},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Hello from object" {
			t.Errorf("expected 'Hello from object', got %q", text.Text)
		}
	})
}

func TestDoGenerate_RawTextWithThinkTags(t *testing.T) {
	t.Run("should return raw text with think tags", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "raw-think-id",
			"created": float64(1711113008),
			"model":   "magistral-small-2506",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":       "assistant",
						"content":    "<think>\nLet me think about this problem step by step.\nFirst, I need to understand what the user is asking.\nThen I can provide a helpful response.\n</think>\n\nHello! I'm ready to help you with your question.",
						"tool_calls": nil,
					},
					"finish_reason": "stop",
					"logprobs":      nil,
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := NewChatLanguageModel("magistral-small-2506", ChatConfig{
			Provider: "mistral.chat",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{
					"authorization": "Bearer test-api-key",
					"content-type":  "application/json",
				}
			},
			GenerateID: mockIDCounter(),
		})

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		// Raw text with think tags should be returned as-is
		if !strings.Contains(text.Text, "<think>") {
			t.Errorf("expected text to contain '<think>', got %q", text.Text)
		}
		if !strings.Contains(text.Text, "Hello! I'm ready to help you with your question.") {
			t.Errorf("expected text to contain final answer, got %q", text.Text)
		}
	})
}

func TestDoGenerate_JSONResponseFormat(t *testing.T) {
	t.Run("should inject JSON instruction for JSON response format", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		responseFormat, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("expected response_format to be map, got %T", body["response_format"])
		}
		if responseFormat["type"] != "json_object" {
			t.Errorf("expected response_format type 'json_object', got %v", responseFormat["type"])
		}
	})

	t.Run("should inject JSON instruction for JSON response format with schema", func(t *testing.T) {
		server, capture := createTestServer(mistralTextFixture(), nil)
		defer server.Close()
		model := createModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		responseFormat, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("expected response_format to be map, got %T", body["response_format"])
		}
		if responseFormat["type"] != "json_schema" {
			t.Errorf("expected response_format type 'json_schema', got %v", responseFormat["type"])
		}
		jsonSchema, ok := responseFormat["json_schema"].(map[string]any)
		if !ok {
			t.Fatalf("expected json_schema to be map, got %T", responseFormat["json_schema"])
		}
		if jsonSchema["name"] != "response" {
			t.Errorf("expected json_schema name 'response', got %v", jsonSchema["name"])
		}
		if jsonSchema["strict"] != false {
			t.Errorf("expected json_schema strict false, got %v", jsonSchema["strict"])
		}
	})
}

func TestDoGenerate_ToolResultFormat(t *testing.T) {
	t.Run("should handle LanguageModelV3ToolResultOutput format", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "test-id",
			"object":  "chat.completion",
			"created": float64(1234567890),
			"model":   "mistral-small",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":       "assistant",
						"content":    "Here is the result",
						"tool_calls": nil,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(10),
				"completion_tokens": float64(5),
				"total_tokens":      float64(15),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							ToolCallID: "call-1",
							ToolName:   "test-tool",
							Input:      map[string]any{"query": "test"},
						},
					},
				},
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "test-tool",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"result": "success"}},
						},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Here is the result" {
			t.Errorf("expected 'Here is the result', got %q", text.Text)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected finish reason 'stop', got %q", result.FinishReason.Unified)
		}
	})
}

func TestDoGenerate_ReferenceContentParsing(t *testing.T) {
	t.Run("should handle reference_ids as numbers", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "test-id",
			"created": float64(1711113008),
			"model":   "mistral-small-latest",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{"type": "text", "text": "Here is the info"},
							map[string]any{"type": "reference", "reference_ids": []any{float64(1), float64(2), float64(3)}},
						},
						"tool_calls": nil,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have text content, reference type should be ignored
		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Here is the info" {
			t.Errorf("expected 'Here is the info', got %q", text.Text)
		}
	})

	t.Run("should handle reference_ids as strings", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "test-id",
			"created": float64(1711113008),
			"model":   "mistral-small-latest",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{"type": "text", "text": "Here is the info"},
							map[string]any{"type": "reference", "reference_ids": []any{"ref-1", "ref-2", "ref-3"}},
						},
						"tool_calls": nil,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Here is the info" {
			t.Errorf("expected 'Here is the info', got %q", text.Text)
		}
	})

	t.Run("should handle mixed reference_ids", func(t *testing.T) {
		fixture := map[string]any{
			"object":  "chat.completion",
			"id":      "test-id",
			"created": float64(1711113008),
			"model":   "mistral-small-latest",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role": "assistant",
						"content": []any{
							map[string]any{"type": "text", "text": "Here is the info"},
							map[string]any{"type": "reference", "reference_ids": []any{float64(1), "ref-2", float64(3)}},
						},
						"tool_calls": nil,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Here is the info" {
			t.Errorf("expected 'Here is the info', got %q", text.Text)
		}
	})
}

// ===== DoStream tests =====

func TestDoStream_Text(t *testing.T) {
	t.Run("should stream text", func(t *testing.T) {
		server, _ := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Collect text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText == "" {
			t.Fatal("expected non-empty streamed text")
		}
		if !strings.Contains(fullText, "Hello") {
			t.Errorf("expected text to contain 'Hello', got %q", fullText)
		}
		if !strings.Contains(fullText, "world!") {
			t.Errorf("expected text to contain 'world!', got %q", fullText)
		}

		// Should have a finish part
		var gotFinish bool
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartFinish); ok {
				gotFinish = true
			}
		}
		if !gotFinish {
			t.Error("expected finish part in stream")
		}
	})
}

func TestDoStream_ToolCall(t *testing.T) {
	t.Run("should stream tool call", func(t *testing.T) {
		server, _ := createSSETestServer(mistralToolCallStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have tool input start
		var toolStart *languagemodel.StreamPartToolInputStart
		for _, part := range parts {
			if ts, ok := part.(languagemodel.StreamPartToolInputStart); ok {
				toolStart = &ts
			}
		}
		if toolStart == nil {
			t.Fatal("expected tool input start part")
		}
		if toolStart.ToolName != "weather" {
			t.Errorf("expected tool name 'weather', got %q", toolStart.ToolName)
		}

		// Should have tool call
		var toolCall *languagemodel.ToolCall
		for _, part := range parts {
			if tc, ok := part.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}
		if toolCall == nil {
			t.Fatal("expected tool call part")
		}
		if toolCall.ToolName != "weather" {
			t.Errorf("expected tool name 'weather', got %q", toolCall.ToolName)
		}

		// Should have finish with tool_calls reason
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected finish reason 'tool-calls', got %q", finishPart.FinishReason.Unified)
		}
	})
}

func TestDoStream_Reasoning(t *testing.T) {
	t.Run("should stream reasoning", func(t *testing.T) {
		server, _ := createSSETestServer(mistralReasoningStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Collect reasoning deltas
		var reasoningText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningText += delta.Delta
			}
		}
		if reasoningText == "" {
			t.Fatal("expected non-empty reasoning text")
		}
		if !strings.Contains(reasoningText, "2+2") {
			t.Errorf("expected reasoning to contain '2+2', got %q", reasoningText)
		}

		// Collect text deltas
		var textContent string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				textContent += delta.Delta
			}
		}
		if textContent != "2 + 2 = 4" {
			t.Errorf("expected text '2 + 2 = 4', got %q", textContent)
		}

		// Should have reasoning start/end
		var hasReasoningStart, hasReasoningEnd bool
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartReasoningStart); ok {
				hasReasoningStart = true
			}
			if _, ok := part.(languagemodel.StreamPartReasoningEnd); ok {
				hasReasoningEnd = true
			}
		}
		if !hasReasoningStart {
			t.Error("expected reasoning start part")
		}
		if !hasReasoningEnd {
			t.Error("expected reasoning end part")
		}
	})
}

func TestDoStream_PassMessages(t *testing.T) {
	t.Run("should pass the messages", func(t *testing.T) {
		server, capture := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		collectStreamParts(streamResult.Stream)

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
		if body["model"] != "mistral-small-latest" {
			t.Errorf("expected model 'mistral-small-latest', got %v", body["model"])
		}

		messages, ok := body["messages"].([]any)
		if !ok {
			t.Fatalf("expected messages to be []any, got %T", body["messages"])
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
	})
}

func TestDoStream_PassHeaders(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModelWithHeaders(server.URL, map[string]string{
			"authorization":          "Bearer test-api-key",
			"content-type":           "application/json",
			"Custom-Provider-Header": "provider-header-value",
		})

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		collectStreamParts(streamResult.Stream)

		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q",
				capture.Headers.Get("Authorization"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createSSETestServer(mistralTextStreamChunks(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		collectStreamParts(streamResult.Stream)

		if streamResult.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if streamResult.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header 'test-value', got %q",
				streamResult.Response.Headers["Test-Header"])
		}
	})
}

func TestDoStream_TrailingAssistantMessage(t *testing.T) {
	t.Run("should avoid duplication when trailing assistant message", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"53ff663126294946a6b7a4747b70597e\",\"object\":\"chat.completion.chunk\",\"created\":1750537996,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"53ff663126294946a6b7a4747b70597e\",\"object\":\"chat.completion.chunk\",\"created\":1750537996,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"prefix\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"53ff663126294946a6b7a4747b70597e\",\"object\":\"chat.completion.chunk\",\"created\":1750537996,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\" and\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"53ff663126294946a6b7a4747b70597e\",\"object\":\"chat.completion.chunk\",\"created\":1750537996,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\" more content\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"53ff663126294946a6b7a4747b70597e\",\"object\":\"chat.completion.chunk\",\"created\":1750537996,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"},\"finish_reason\":\"stop\",\"logprobs\":null}],\"usage\":{\"prompt_tokens\":4,\"total_tokens\":36,\"completion_tokens\":32}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "prefix "},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Collect text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if !strings.Contains(fullText, "prefix") {
			t.Errorf("expected text to contain 'prefix', got %q", fullText)
		}
		if !strings.Contains(fullText, " and") {
			t.Errorf("expected text to contain ' and', got %q", fullText)
		}
		if !strings.Contains(fullText, " more content") {
			t.Errorf("expected text to contain ' more content', got %q", fullText)
		}

		// Verify finish
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected finish reason 'stop', got %q", finishPart.FinishReason.Unified)
		}
	})
}

func TestDoStream_TextWithContentObjects(t *testing.T) {
	t.Run("should stream text with content objects", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"b9e43f82d6c74a1e9f5b2c8e7a9d4f6b\",\"object\":\"chat.completion.chunk\",\"created\":1750538500,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"\"}]},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"b9e43f82d6c74a1e9f5b2c8e7a9d4f6b\",\"object\":\"chat.completion.chunk\",\"created\":1750538500,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":[{\"type\":\"text\",\"text\":\"Hello\"}]},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"b9e43f82d6c74a1e9f5b2c8e7a9d4f6b\",\"object\":\"chat.completion.chunk\",\"created\":1750538500,\"model\":\"mistral-small-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":[{\"type\":\"text\",\"text\":\", world!\"}]},\"finish_reason\":\"stop\",\"logprobs\":null}],\"usage\":{\"prompt_tokens\":4,\"total_tokens\":36,\"completion_tokens\":32}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText != "Hello, world!" {
			t.Errorf("expected text 'Hello, world!', got %q", fullText)
		}
	})
}

func TestDoStream_InterleavedThinkingAndText(t *testing.T) {
	t.Run("should handle interleaved thinking and text", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"interleaved-test\",\"object\":\"chat.completion.chunk\",\"created\":1750538000,\"model\":\"magistral-small-2507\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"thinking\",\"thinking\":[{\"type\":\"text\",\"text\":\"First thought.\"}]}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"interleaved-test\",\"object\":\"chat.completion.chunk\",\"created\":1750538000,\"model\":\"magistral-small-2507\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"Partial answer.\"}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"interleaved-test\",\"object\":\"chat.completion.chunk\",\"created\":1750538000,\"model\":\"magistral-small-2507\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"thinking\",\"thinking\":[{\"type\":\"text\",\"text\":\"Second thought.\"}]}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"interleaved-test\",\"object\":\"chat.completion.chunk\",\"created\":1750538000,\"model\":\"magistral-small-2507\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":[{\"type\":\"text\",\"text\":\"Final answer.\"}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"interleaved-test\",\"object\":\"chat.completion.chunk\",\"created\":1750538000,\"model\":\"magistral-small-2507\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":10,\"total_tokens\":40,\"completion_tokens\":30}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Collect reasoning deltas
		var reasoningDeltas []string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, delta.Delta)
			}
		}
		if len(reasoningDeltas) != 2 {
			t.Fatalf("expected 2 reasoning deltas, got %d", len(reasoningDeltas))
		}
		if reasoningDeltas[0] != "First thought." {
			t.Errorf("expected first reasoning delta 'First thought.', got %q", reasoningDeltas[0])
		}
		if reasoningDeltas[1] != "Second thought." {
			t.Errorf("expected second reasoning delta 'Second thought.', got %q", reasoningDeltas[1])
		}

		// Collect text deltas
		var textDeltas []string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, delta.Delta)
			}
		}
		if len(textDeltas) != 2 {
			t.Fatalf("expected 2 text deltas, got %d", len(textDeltas))
		}
		if textDeltas[0] != "Partial answer." {
			t.Errorf("expected first text delta 'Partial answer.', got %q", textDeltas[0])
		}
		if textDeltas[1] != "Final answer." {
			t.Errorf("expected second text delta 'Final answer.', got %q", textDeltas[1])
		}

		// Should have reasoning start, reasoning end, text start, text end events
		var reasoningStartCount, reasoningEndCount int
		var textStartCount, textEndCount int
		for _, part := range parts {
			switch part.(type) {
			case languagemodel.StreamPartReasoningStart:
				reasoningStartCount++
			case languagemodel.StreamPartReasoningEnd:
				reasoningEndCount++
			case languagemodel.StreamPartTextStart:
				textStartCount++
			case languagemodel.StreamPartTextEnd:
				textEndCount++
			}
		}
		if reasoningStartCount != 2 {
			t.Errorf("expected 2 reasoning starts, got %d", reasoningStartCount)
		}
		if reasoningEndCount != 2 {
			t.Errorf("expected 2 reasoning ends, got %d", reasoningEndCount)
		}
		if textStartCount != 2 {
			t.Errorf("expected 2 text starts, got %d", textStartCount)
		}
		if textEndCount != 2 {
			t.Errorf("expected 2 text ends, got %d", textEndCount)
		}
	})
}

func TestDoStream_ResponseMetadata(t *testing.T) {
	t.Run("should stream response metadata", func(t *testing.T) {
		server, _ := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		var metadataPart *languagemodel.StreamPartResponseMetadata
		for _, part := range parts {
			if mp, ok := part.(languagemodel.StreamPartResponseMetadata); ok {
				metadataPart = &mp
			}
		}
		if metadataPart == nil {
			t.Fatal("expected response metadata part in stream")
		}
		if metadataPart.ID == nil || *metadataPart.ID != "5319bd0299614c679a0068a4f2c8ffd0" {
			t.Errorf("expected ID '5319bd0299614c679a0068a4f2c8ffd0', got %v", metadataPart.ID)
		}
		if metadataPart.ModelID == nil || *metadataPart.ModelID != "mistral-small-latest" {
			t.Errorf("expected ModelID 'mistral-small-latest', got %v", metadataPart.ModelID)
		}
		expectedTime := time.Unix(1769088720, 0)
		if metadataPart.Timestamp == nil || !metadataPart.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, metadataPart.Timestamp)
		}
	})
}

func TestDoStream_FinishReason(t *testing.T) {
	t.Run("should extract finish reason from stream", func(t *testing.T) {
		server, _ := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", finishPart.FinishReason.Unified)
		}
		if finishPart.FinishReason.Raw == nil || *finishPart.FinishReason.Raw != "stop" {
			t.Errorf("expected raw finish reason 'stop', got %v", finishPart.FinishReason.Raw)
		}
	})
}

func TestDoStream_Usage(t *testing.T) {
	t.Run("should extract usage from stream", func(t *testing.T) {
		server, _ := createSSETestServer(mistralTextStreamChunks(), nil)
		defer server.Close()
		model := createModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}

		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 13 {
			t.Errorf("expected input total tokens 13, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 8 {
			t.Errorf("expected output total tokens 8, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}

func TestDoStream_RawChunks(t *testing.T) {
	t.Run("should stream raw chunks", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"c7d54e93f8a64b2e9c1f5a8b7d9e2f4c\",\"object\":\"chat.completion.chunk\",\"created\":1750538600,\"model\":\"mistral-large-latest\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hello\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"d8e65fa4g9b75c3f0d2g6b9c8e0f3g5d\",\"object\":\"chat.completion.chunk\",\"created\":1750538601,\"model\":\"mistral-large-latest\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null,\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"e9f76gb5h0c86d4g1e3h7c0d9f1g4h6e\",\"object\":\"chat.completion.chunk\",\"created\":1750538602,\"model\":\"mistral-large-latest\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\",\"logprobs\":null}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createModel(server.URL)

		includeRaw := true
		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt:           testPrompt,
			Ctx:              context.Background(),
			IncludeRawChunks: &includeRaw,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have raw chunks
		var rawCount int
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartRaw); ok {
				rawCount++
			}
		}
		if rawCount != 3 {
			t.Errorf("expected 3 raw chunks, got %d", rawCount)
		}

		// Should still have text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText != "Hello world" {
			t.Errorf("expected text 'Hello world', got %q", fullText)
		}

		// Should have finish with correct usage
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 10 {
			t.Errorf("expected input total tokens 10, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 5 {
			t.Errorf("expected output total tokens 5, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}
