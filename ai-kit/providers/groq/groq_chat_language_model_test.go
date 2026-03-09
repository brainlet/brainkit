// Ported from: packages/groq/src/groq-chat-language-model.test.ts
package groq

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

// requestCapture captures HTTP request details.
type requestCapture struct {
	Body    []byte
	Headers http.Header
}

func (rc *requestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

// groqTextFixture returns the JSON fixture for groq-text responses.
func groqTextFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-09d64d2a-ed1c-4473-829f-78db43f45d13",
		"object":  "chat.completion",
		"created": float64(1770770798),
		"model":   "llama-3.3-70b-versatile",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "I'd like to introduce \"Luminaria\" - a joyous holiday that celebrates the magic of light, community, and personal growth.",
				},
				"logprobs":      nil,
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(45),
			"completion_tokens": float64(607),
			"total_tokens":      float64(652),
		},
	}
}

// groqToolCallFixture returns the JSON fixture for groq-tool-call responses.
func groqToolCallFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-1fd017fc-60b8-44eb-a736-375b8e1bc3e7",
		"object":  "chat.completion",
		"created": float64(1770770815),
		"model":   "llama-3.3-70b-versatile",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{
						map[string]any{
							"id":   "ax9fskhev",
							"type": "function",
							"function": map[string]any{
								"name":      "weather",
								"arguments": "{}",
							},
						},
					},
				},
				"logprobs":      nil,
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(218),
			"completion_tokens": float64(15),
			"total_tokens":      float64(233),
		},
	}
}

// groqReasoningFixture returns the JSON fixture for groq-reasoning responses.
func groqReasoningFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-73cf8a54-d54e-400c-88b8-603d1a346d96",
		"object":  "chat.completion",
		"created": float64(1770770833),
		"model":   "qwen/qwen3-32b",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":      "assistant",
					"content":   "The word \"strawberry\" contains **3** instances of the letter **R**.",
					"reasoning": "Okay, let me count the R's in strawberry...",
				},
				"logprobs":      nil,
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(17),
			"completion_tokens": float64(649),
			"total_tokens":      float64(666),
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": float64(570),
			},
		},
	}
}

// createJSONTestServer creates an httptest server that serves the given JSON body.
func createJSONTestServer(body any, headers map[string]string) (*httptest.Server, *requestCapture) {
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

// createTestModel creates a GroqChatLanguageModel targeting a test server.
func createTestModel(baseURL string) *GroqChatLanguageModel {
	return NewGroqChatLanguageModel("gemma2-9b-it", GroqChatConfig{
		Provider: "groq.chat",
		URL: func(_ string, path string) string {
			return baseURL + path
		},
		Headers: func() map[string]string {
			return map[string]string{
				"authorization": "Bearer test-api-key",
				"content-type":  "application/json",
			}
		},
	})
}

// createTestModelWithHeaders creates a GroqChatLanguageModel with custom headers.
func createTestModelWithHeaders(baseURL string, headers map[string]string) *GroqChatLanguageModel {
	return NewGroqChatLanguageModel("gemma2-9b-it", GroqChatConfig{
		Provider: "groq.chat",
		URL: func(_ string, path string) string {
			return baseURL + path
		},
		Headers: func() map[string]string {
			return headers
		},
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

// ===== DoGenerate tests =====

func TestDoGenerate_Text(t *testing.T) {
	t.Run("should extract text content", func(t *testing.T) {
		server, _ := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var text string
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.Text); ok {
				text += tc.Text
			}
		}
		if text == "" {
			t.Fatal("expected non-empty text content")
		}
		if !strings.Contains(text, "Luminaria") {
			t.Errorf("expected text to contain 'Luminaria', got: %s", text)
		}
	})

	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "gemma2-9b-it" {
			t.Errorf("expected model 'gemma2-9b-it', got %v", body["model"])
		}
		messages := body["messages"].([]any)
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
		if msg["content"] != "Hello" {
			t.Errorf("expected content 'Hello', got %v", msg["content"])
		}
	})
}

func TestDoGenerate_ToolCall(t *testing.T) {
	t.Run("should extract tool call content", func(t *testing.T) {
		server, _ := createJSONTestServer(groqToolCallFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var toolCallCount int
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.ToolCall); ok {
				toolCallCount++
				if tc.ToolName != "weather" {
					t.Errorf("expected tool name 'weather', got %q", tc.ToolName)
				}
				if tc.ToolCallID != "ax9fskhev" {
					t.Errorf("expected tool call ID 'ax9fskhev', got %q", tc.ToolCallID)
				}
			}
		}
		if toolCallCount != 1 {
			t.Errorf("expected 1 tool call, got %d", toolCallCount)
		}
	})
}

func TestDoGenerate_Reasoning(t *testing.T) {
	t.Run("should extract reasoning content", func(t *testing.T) {
		server, _ := createJSONTestServer(groqReasoningFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var hasReasoning bool
		var hasText bool
		for _, c := range result.Content {
			if _, ok := c.(languagemodel.Reasoning); ok {
				hasReasoning = true
			}
			if _, ok := c.(languagemodel.Text); ok {
				hasText = true
			}
		}
		if !hasReasoning {
			t.Error("expected reasoning content")
		}
		if !hasText {
			t.Error("expected text content")
		}
	})
}

func TestDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 45 {
			t.Errorf("expected input total tokens 45, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 45 {
			t.Errorf("expected input noCache tokens 45, got %v", result.Usage.InputTokens.NoCache)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 607 {
			t.Errorf("expected output total tokens 607, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 607 {
			t.Errorf("expected output text tokens 607, got %v", result.Usage.OutputTokens.Text)
		}
	})
}

func TestDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should send additional response information", func(t *testing.T) {
		server, _ := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

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
		if result.Response.ID == nil || *result.Response.ID != "chatcmpl-09d64d2a-ed1c-4473-829f-78db43f45d13" {
			t.Errorf("expected ID 'chatcmpl-09d64d2a-ed1c-4473-829f-78db43f45d13', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "llama-3.3-70b-versatile" {
			t.Errorf("expected ModelID 'llama-3.3-70b-versatile', got %v", result.Response.ModelID)
		}
		expectedTime := time.Unix(1770770798, 0)
		if result.Response.Timestamp == nil || !result.Response.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, result.Response.Timestamp)
		}
	})
}

func TestDoGenerate_PartialUsage(t *testing.T) {
	t.Run("should support partial usage", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": float64(1711115037),
			"model":   "gemma2-9b-it",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens": float64(20),
				"total_tokens":  float64(20),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 20 {
			t.Errorf("expected input total 20, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 0 {
			t.Errorf("expected output total 0, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 0 {
			t.Errorf("expected output text 0, got %v", result.Usage.OutputTokens.Text)
		}
	})
}

func TestDoGenerate_ReasoningTokens(t *testing.T) {
	t.Run("should extract reasoning tokens from completion_tokens_details", func(t *testing.T) {
		server, _ := createJSONTestServer(groqReasoningFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 17 {
			t.Errorf("expected input total 17, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 649 {
			t.Errorf("expected output total 649, got %v", result.Usage.OutputTokens.Total)
		}
		// text = 649 - 570 = 79
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 79 {
			t.Errorf("expected output text 79, got %v", result.Usage.OutputTokens.Text)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 570 {
			t.Errorf("expected output reasoning 570, got %v", result.Usage.OutputTokens.Reasoning)
		}
	})
}

func TestDoGenerate_UnknownFinishReason(t *testing.T) {
	t.Run("should support unknown finish reason", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": float64(1711115037),
			"model":   "gemma2-9b-it",
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":    "assistant",
						"content": "",
					},
					"finish_reason": "eos",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(4),
				"total_tokens":      float64(34),
				"completion_tokens": float64(30),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonOther {
			t.Errorf("expected unified finish reason 'other', got %q", result.FinishReason.Unified)
		}
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != "eos" {
			t.Errorf("expected raw finish reason 'eos', got %v", result.FinishReason.Raw)
		}
	})
}

func TestDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createJSONTestServer(groqTextFixture(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createTestModel(server.URL)

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

func TestDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should pass provider options", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"groq": {
					"reasoningFormat":   "hidden",
					"user":              "test-user-id",
					"parallelToolCalls": false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["reasoning_format"] != "hidden" {
			t.Errorf("expected reasoning_format 'hidden', got %v", body["reasoning_format"])
		}
		if body["user"] != "test-user-id" {
			t.Errorf("expected user 'test-user-id', got %v", body["user"])
		}
		if body["parallel_tool_calls"] != false {
			t.Errorf("expected parallel_tool_calls false, got %v", body["parallel_tool_calls"])
		}
	})

	t.Run("should pass serviceTier provider option", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"groq": {
					"serviceTier": "flex",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["service_tier"] != "flex" {
			t.Errorf("expected service_tier 'flex', got %v", body["service_tier"])
		}
	})
}

func TestDoGenerate_ToolsAndToolChoice(t *testing.T) {
	t.Run("should pass tools and toolChoice", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
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
			Prompt:     testPrompt,
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		tools, ok := body["tools"].([]any)
		if !ok {
			t.Fatalf("expected tools to be []any, got %T", body["tools"])
		}
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}

		toolChoice, ok := body["tool_choice"].(map[string]any)
		if !ok {
			t.Fatalf("expected tool_choice to be map[string]any, got %T", body["tool_choice"])
		}
		if toolChoice["type"] != "function" {
			t.Errorf("expected tool_choice type 'function', got %v", toolChoice["type"])
		}
		fn := toolChoice["function"].(map[string]any)
		if fn["name"] != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got %v", fn["name"])
		}
	})
}

func TestDoGenerate_Headers(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModelWithHeaders(server.URL, map[string]string{
			"authorization":        "Bearer test-api-key",
			"content-type":         "application/json",
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
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q", capture.Headers.Get("Authorization"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q", capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q", capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestDoGenerate_ResponseFormat_StructuredOutputs(t *testing.T) {
	t.Run("should pass response format information as json_schema when structuredOutputs enabled by default", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Name:        strPtr("test-name"),
				Description: strPtr("test description"),
				Schema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{"value": map[string]any{"type": "string"}},
					"required":             []any{"value"},
					"additionalProperties": false,
					"$schema":              "http://json-schema.org/draft-07/schema#",
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("expected response_format to be map, got %T", body["response_format"])
		}
		if rf["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", rf["type"])
		}
		js := rf["json_schema"].(map[string]any)
		if js["name"] != "test-name" {
			t.Errorf("expected name 'test-name', got %v", js["name"])
		}
		if js["description"] != "test description" {
			t.Errorf("expected description 'test description', got %v", js["description"])
		}
		if js["strict"] != true {
			t.Errorf("expected strict true, got %v", js["strict"])
		}
	})

	t.Run("should pass response format information as json_object when structuredOutputs explicitly disabled", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			ProviderOptions: shared.ProviderOptions{
				"groq": {
					"structuredOutputs": false,
				},
			},
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Name:        strPtr("test-name"),
				Description: strPtr("test description"),
				Schema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{"value": map[string]any{"type": "string"}},
					"required":             []any{"value"},
					"additionalProperties": false,
					"$schema":              "http://json-schema.org/draft-07/schema#",
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		if rf["type"] != "json_object" {
			t.Errorf("expected type 'json_object', got %v", rf["type"])
		}

		// Should have warnings
		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		w, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if w.Feature != "responseFormat" {
			t.Errorf("expected feature 'responseFormat', got %q", w.Feature)
		}
		if w.Details == nil || !strings.Contains(*w.Details, "structuredOutputs") {
			t.Errorf("expected details about structuredOutputs, got %v", w.Details)
		}
	})

	t.Run("should send strict: false when strictJsonSchema is explicitly disabled", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			ProviderOptions: shared.ProviderOptions{
				"groq": {
					"strictJsonSchema": false,
				},
			},
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Name:        strPtr("test-name"),
				Description: strPtr("test description"),
				Schema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{"value": map[string]any{"type": "string"}},
					"required":             []any{"value"},
					"additionalProperties": false,
					"$schema":              "http://json-schema.org/draft-07/schema#",
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		js := rf["json_schema"].(map[string]any)
		if js["strict"] != false {
			t.Errorf("expected strict false, got %v", js["strict"])
		}
	})

	t.Run("should allow explicit structuredOutputs override with default name", func(t *testing.T) {
		server, capture := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			ProviderOptions: shared.ProviderOptions{
				"groq": {
					"structuredOutputs": true,
				},
			},
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{"value": map[string]any{"type": "string"}},
					"required":             []any{"value"},
					"additionalProperties": false,
					"$schema":              "http://json-schema.org/draft-07/schema#",
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		js := rf["json_schema"].(map[string]any)
		if js["name"] != "response" {
			t.Errorf("expected default name 'response', got %v", js["name"])
		}
	})
}

func TestDoGenerate_RequestBody(t *testing.T) {
	t.Run("should send request body", func(t *testing.T) {
		server, _ := createJSONTestServer(groqTextFixture(), nil)
		defer server.Close()
		model := createTestModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Request == nil {
			t.Fatal("expected request to be non-nil")
		}
		bodyStr, ok := result.Request.Body.(string)
		if !ok || bodyStr == "" {
			t.Fatal("expected non-empty request body string")
		}

		// Parse the request body to verify it contains the expected fields
		var body map[string]any
		if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
			t.Fatalf("failed to parse request body: %v", err)
		}
		if body["model"] != "gemma2-9b-it" {
			t.Errorf("expected model 'gemma2-9b-it', got %v", body["model"])
		}
	})
}

// ===== DoStream tests =====

func TestDoStream_Text(t *testing.T) {
	t.Run("should stream text", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

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
		if fullText != "Hello world" {
			t.Errorf("expected text 'Hello world', got %q", fullText)
		}

		// Check for finish part
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
	})
}

func TestDoStream_ToolCallDeltas(t *testing.T) {
	t.Run("should stream tool call deltas when tool call arguments are passed in chunks", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_O17Uplv4","type":"function","function":{"name":"test-tool","arguments":"{\"" }}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"va"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"lue"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\":\"" }}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"Spark"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"le"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" Day"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"}"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1729171479,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"tool_calls"}],"x_groq":{"id":"req_test","usage":{"prompt_tokens":18,"completion_tokens":439,"total_tokens":457}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"value": map[string]any{"type": "string"}},
						"required":             []any{"value"},
						"additionalProperties": false,
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Check for tool-input-start
		var toolStartCount int
		for _, part := range parts {
			if ts, ok := part.(languagemodel.StreamPartToolInputStart); ok {
				toolStartCount++
				if ts.ToolName != "test-tool" {
					t.Errorf("expected tool name 'test-tool', got %q", ts.ToolName)
				}
				if ts.ID != "call_O17Uplv4" {
					t.Errorf("expected tool ID 'call_O17Uplv4', got %q", ts.ID)
				}
			}
		}
		if toolStartCount != 1 {
			t.Errorf("expected 1 tool-input-start, got %d", toolStartCount)
		}

		// Check for tool-input-delta parts
		var toolArgDeltas string
		for _, part := range parts {
			if td, ok := part.(languagemodel.StreamPartToolInputDelta); ok {
				toolArgDeltas += td.Delta
			}
		}
		if toolArgDeltas != `{"value":"Sparkle Day"}` {
			t.Errorf("expected tool args '{\"value\":\"Sparkle Day\"}', got %q", toolArgDeltas)
		}

		// Check for tool-call
		var toolCallCount int
		for _, part := range parts {
			if tc, ok := part.(languagemodel.ToolCall); ok {
				toolCallCount++
				if tc.ToolName != "test-tool" {
					t.Errorf("expected tool name 'test-tool', got %q", tc.ToolName)
				}
				if tc.Input != `{"value":"Sparkle Day"}` {
					t.Errorf("expected tool input '{\"value\":\"Sparkle Day\"}', got %q", tc.Input)
				}
			}
		}
		if toolCallCount != 1 {
			t.Errorf("expected 1 tool-call, got %d", toolCallCount)
		}

		// Check finish reason
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected unified finish reason 'tool-calls', got %q", finishPart.FinishReason.Unified)
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 18 {
			t.Errorf("expected input total 18, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 439 {
			t.Errorf("expected output total 439, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}

func TestDoStream_ToolCallSingleChunk(t *testing.T) {
	t.Run("should stream tool call that is sent in one chunk", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1711357598,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_O17Uplv4","type":"function","function":{"name":"test-tool","arguments":"{\"value\":\"Sparkle Day\"}"}}]},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-e7f8e220","object":"chat.completion.chunk","created":1729171479,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"logprobs":null,"finish_reason":"tool_calls"}],"x_groq":{"id":"req_test","usage":{"prompt_tokens":18,"completion_tokens":439,"total_tokens":457}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "test-tool",
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"value": map[string]any{"type": "string"}},
						"required":             []any{"value"},
						"additionalProperties": false,
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have tool-input-start, tool-input-delta, tool-input-end, tool-call
		var toolCallCount int
		for _, part := range parts {
			if tc, ok := part.(languagemodel.ToolCall); ok {
				toolCallCount++
				if tc.ToolName != "test-tool" {
					t.Errorf("expected tool name 'test-tool', got %q", tc.ToolName)
				}
				if tc.Input != `{"value":"Sparkle Day"}` {
					t.Errorf("expected tool input, got %q", tc.Input)
				}
			}
		}
		if toolCallCount != 1 {
			t.Errorf("expected 1 tool-call, got %d", toolCallCount)
		}
	})
}

func TestDoStream_ErrorParts(t *testing.T) {
	t.Run("should handle error stream parts", func(t *testing.T) {
		chunks := []string{
			`data: {"error":{"message": "The server had an error processing your request. Sorry about that!","type":"invalid_request_error"}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have an error part
		var hasError bool
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartError); ok {
				hasError = true
			}
		}
		if !hasError {
			t.Error("expected error part in stream")
		}

		// Should have finish with error reason
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonError {
			t.Errorf("expected unified finish reason 'error', got %q", finishPart.FinishReason.Unified)
		}
	})
}

func TestDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		if streamResult.Response == nil {
			t.Fatal("expected response to be non-nil")
		}
		if streamResult.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header 'test-value', got %q", streamResult.Response.Headers["Test-Header"])
		}
	})
}

func TestDoStream_StreamingRequestBody(t *testing.T) {
	t.Run("should send correct streaming request body", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		body := capture.BodyJSON()
		if body["model"] != "gemma2-9b-it" {
			t.Errorf("expected model 'gemma2-9b-it', got %v", body["model"])
		}
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
		messages := body["messages"].([]any)
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["content"] != "Hello" {
			t.Errorf("expected content 'Hello', got %v", msg["content"])
		}
	})
}

func TestDoStream_StreamHeaders(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModelWithHeaders(server.URL, map[string]string{
			"authorization":        "Bearer test-api-key",
			"content-type":         "application/json",
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

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected Authorization 'Bearer test-api-key', got %q", capture.Headers.Get("Authorization"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q", capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q", capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestDoStream_RequestBody(t *testing.T) {
	t.Run("should send request body", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		if streamResult.Request == nil {
			t.Fatal("expected request to be non-nil")
		}
		// The request body is a map[string]any for streaming
		body, ok := streamResult.Request.Body.(map[string]any)
		if !ok {
			t.Fatalf("expected request body to be map[string]any, got %T", streamResult.Request.Body)
		}
		if body["model"] != "gemma2-9b-it" {
			t.Errorf("expected model 'gemma2-9b-it', got %v", body["model"])
		}
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
	})
}

func TestDoStream_ResponseMetadata(t *testing.T) {
	t.Run("should stream response metadata", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

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
		if metadataPart.ID == nil || *metadataPart.ID != "chatcmpl-123" {
			t.Errorf("expected ID 'chatcmpl-123', got %v", metadataPart.ID)
		}
		if metadataPart.ModelID == nil || *metadataPart.ModelID != "gemma2-9b-it" {
			t.Errorf("expected ModelID 'gemma2-9b-it', got %v", metadataPart.ModelID)
		}
		expectedTime := time.Unix(1234567890, 0)
		if metadataPart.Timestamp == nil || !metadataPart.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, metadataPart.Timestamp)
		}
	})
}

func TestDoStream_Reasoning(t *testing.T) {
	t.Run("should stream reasoning", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"qwen/qwen3-32b","choices":[{"index":0,"delta":{"role":"assistant","reasoning":"Let me think..."},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"qwen/qwen3-32b","choices":[{"index":0,"delta":{"reasoning":" about this."},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"qwen/qwen3-32b","choices":[{"index":0,"delta":{"content":"The answer is 3."},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"qwen/qwen3-32b","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := NewGroqChatLanguageModel("qwen/qwen3-32b", GroqChatConfig{
			Provider: "groq.chat",
			URL: func(_ string, path string) string {
				return server.URL + path
			},
			Headers: func() map[string]string {
				return map[string]string{
					"authorization": "Bearer test-api-key",
					"content-type":  "application/json",
				}
			},
		})

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Check for reasoning parts
		var reasoningText string
		for _, part := range parts {
			if rd, ok := part.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningText += rd.Delta
			}
		}
		if reasoningText != "Let me think... about this." {
			t.Errorf("expected reasoning text 'Let me think... about this.', got %q", reasoningText)
		}

		// Check for text parts
		var textContent string
		for _, part := range parts {
			if td, ok := part.(languagemodel.StreamPartTextDelta); ok {
				textContent += td.Delta
			}
		}
		if textContent != "The answer is 3." {
			t.Errorf("expected text 'The answer is 3.', got %q", textContent)
		}

		// Check reasoning start/end
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
			t.Error("expected reasoning-start part")
		}
		if !hasReasoningEnd {
			t.Error("expected reasoning-end part")
		}
	})
}

func TestDoStream_RawChunks(t *testing.T) {
	t.Run("should stream raw chunks when includeRawChunks is true", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-123","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-456","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chatcmpl-789","object":"chat.completion.chunk","created":1234567890,"model":"gemma2-9b-it","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"x_groq":{"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		includeRaw := true
		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt:          testPrompt,
			Ctx:             context.Background(),
			IncludeRawChunks: &includeRaw,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Count raw parts
		var rawCount int
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartRaw); ok {
				rawCount++
			}
		}
		// Should have 3 raw parts (one per data chunk, excluding [DONE])
		if rawCount != 3 {
			t.Errorf("expected 3 raw parts, got %d", rawCount)
		}

		// Text should still work
		var fullText string
		for _, part := range parts {
			if td, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += td.Delta
			}
		}
		if fullText != "Hello world" {
			t.Errorf("expected text 'Hello world', got %q", fullText)
		}
	})
}

func TestDoStream_DuplicateToolCalls(t *testing.T) {
	t.Run("should not duplicate tool calls when there is an additional empty chunk", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"role":"assistant","content":""},"logprobs":null,"finish_reason":null}],"usage":{"prompt_tokens":226,"total_tokens":226,"completion_tokens":0}}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"id":"tool-id-123","type":"function","index":0,"function":{"name":"searchGoogle"}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"query\": \""}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"latest"}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" news"}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" on"}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":" ai\"}"}}]},"logprobs":null,"finish_reason":null}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":""}}]},"logprobs":null,"finish_reason":"tool_calls","stop_reason":128008}]}` + "\n\n",
			`data: {"id":"chat-test","object":"chat.completion.chunk","created":1733162241,"model":"meta/llama-3.1-8b-instruct:fp8","choices":[],"usage":{"prompt_tokens":226,"total_tokens":246,"completion_tokens":20}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createTestModel(server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name: "searchGoogle",
					InputSchema: map[string]any{
						"type":                 "object",
						"properties":           map[string]any{"query": map[string]any{"type": "string"}},
						"required":             []any{"query"},
						"additionalProperties": false,
					},
				},
			},
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should have exactly 1 tool-call (no duplicates)
		var toolCallCount int
		for _, part := range parts {
			if tc, ok := part.(languagemodel.ToolCall); ok {
				toolCallCount++
				if tc.ToolName != "searchGoogle" {
					t.Errorf("expected tool name 'searchGoogle', got %q", tc.ToolName)
				}
			}
		}
		if toolCallCount != 1 {
			t.Errorf("expected exactly 1 tool-call (no duplicates), got %d", toolCallCount)
		}
	})
}
