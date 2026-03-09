// Ported from: packages/openai-compatible/src/chat/openai-compatible-chat-language-model.test.ts
package openaicompatible

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Test helpers ---

// testPrompt is the standard test prompt used across tests.
var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

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

// createChatModel creates a ChatLanguageModel targeting a test server.
func createChatModel(baseURL string) *ChatLanguageModel {
	return NewChatLanguageModel("test-model", ChatConfig{
		Provider: "test-provider.chat",
		URL: func(path string) string {
			return baseURL + path
		},
		Headers: func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-key",
				"Content-Type":  "application/json",
			}
		},
	})
}

// createChatModelWithConfig creates a ChatLanguageModel with custom config.
func createChatModelWithConfig(baseURL string, config ChatConfig) *ChatLanguageModel {
	if config.URL == nil {
		config.URL = func(path string) string {
			return baseURL + path
		}
	}
	if config.Headers == nil {
		config.Headers = func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-key",
				"Content-Type":  "application/json",
			}
		}
	}
	return NewChatLanguageModel("test-model", config)
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

func chatTextFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-test-id",
		"model":   "test-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello, World!",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(4),
			"completion_tokens": float64(30),
			"total_tokens":      float64(34),
		},
	}
}

func chatToolCallFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-tool-id",
		"model":   "test-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": nil,
					"tool_calls": []any{
						map[string]any{
							"id":   "call-1",
							"type": "function",
							"function": map[string]any{
								"name":      "get_weather",
								"arguments": `{"city":"San Francisco"}`,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(10),
			"completion_tokens": float64(20),
			"total_tokens":      float64(30),
		},
	}
}

func chatUsageDetailedFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-usage-id",
		"model":   "test-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "Response with detailed usage",
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(100),
			"completion_tokens": float64(200),
			"total_tokens":      float64(300),
			"prompt_tokens_details": map[string]any{
				"cached_tokens": float64(30),
			},
			"completion_tokens_details": map[string]any{
				"reasoning_tokens":           float64(50),
				"accepted_prediction_tokens": float64(10),
				"rejected_prediction_tokens": float64(5),
			},
		},
	}
}

func chatReasoningContentFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-reasoning-id",
		"model":   "test-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":              "assistant",
					"content":           "The answer is 42.",
					"reasoning_content": "Let me think about this...",
				},
				"finish_reason": "stop",
			},
		},
	}
}

func chatReasoningFieldFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-reasoning-field-id",
		"model":   "test-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":      "assistant",
					"content":   "The answer is 42.",
					"reasoning": "Using reasoning field...",
				},
				"finish_reason": "stop",
			},
		},
	}
}

// --- Config tests ---

func TestChatLanguageModel_Config(t *testing.T) {
	t.Run("should extract provider options name from provider", func(t *testing.T) {
		model := NewChatLanguageModel("test-model", ChatConfig{
			Provider: "anthropic.beta",
		})
		name := model.providerOptionsName()
		if name != "anthropic" {
			t.Errorf("expected 'anthropic', got %q", name)
		}
	})

	t.Run("should extract simple provider options name", func(t *testing.T) {
		model := NewChatLanguageModel("test-model", ChatConfig{
			Provider: "openai.chat",
		})
		name := model.providerOptionsName()
		if name != "openai" {
			t.Errorf("expected 'openai', got %q", name)
		}
	})
}

// --- DoGenerate tests ---

func TestChatDoGenerate_Text(t *testing.T) {
	t.Run("should extract text content", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

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
		if text != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got %q", text)
		}
	})

	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected FinishReasonStop, got %v", result.FinishReason.Unified)
		}
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != "stop" {
			t.Errorf("expected raw finish reason 'stop', got %v", result.FinishReason.Raw)
		}
	})
}

func TestChatDoGenerate_ToolCall(t *testing.T) {
	t.Run("should extract tool calls", func(t *testing.T) {
		server, _ := createTestServer(chatToolCallFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var toolCalls []languagemodel.ToolCall
		for _, c := range result.Content {
			if tc, ok := c.(languagemodel.ToolCall); ok {
				toolCalls = append(toolCalls, tc)
			}
		}

		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
		if toolCalls[0].ToolCallID != "call-1" {
			t.Errorf("expected tool call ID 'call-1', got %q", toolCalls[0].ToolCallID)
		}
		if toolCalls[0].ToolName != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", toolCalls[0].ToolName)
		}
		if result.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected FinishReasonToolCalls, got %v", result.FinishReason.Unified)
		}
	})
}

func TestChatDoGenerate_Usage(t *testing.T) {
	t.Run("should extract basic usage", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 4 {
			t.Errorf("expected input total 4, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 30 {
			t.Errorf("expected output total 30, got %v", result.Usage.OutputTokens.Total)
		}
	})

	t.Run("should extract detailed usage with cache and reasoning tokens", func(t *testing.T) {
		server, _ := createTestServer(chatUsageDetailedFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 100 {
			t.Errorf("expected input total 100, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.InputTokens.CacheRead == nil || *result.Usage.InputTokens.CacheRead != 30 {
			t.Errorf("expected cache read 30, got %v", result.Usage.InputTokens.CacheRead)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 70 {
			t.Errorf("expected no-cache 70, got %v", result.Usage.InputTokens.NoCache)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 200 {
			t.Errorf("expected output total 200, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 50 {
			t.Errorf("expected reasoning 50, got %v", result.Usage.OutputTokens.Reasoning)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 150 {
			t.Errorf("expected text 150, got %v", result.Usage.OutputTokens.Text)
		}
	})

	t.Run("should include prediction tokens in provider metadata", func(t *testing.T) {
		server, _ := createTestServer(chatUsageDetailedFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		providerMeta := result.ProviderMetadata["test-provider"]
		if providerMeta == nil {
			t.Fatal("expected provider metadata for 'test-provider'")
		}
		if providerMeta["acceptedPredictionTokens"] != 10 {
			t.Errorf("expected acceptedPredictionTokens 10, got %v", providerMeta["acceptedPredictionTokens"])
		}
		if providerMeta["rejectedPredictionTokens"] != 5 {
			t.Errorf("expected rejectedPredictionTokens 5, got %v", providerMeta["rejectedPredictionTokens"])
		}
	})
}

func TestChatDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.ID == nil || *result.Response.ID != "chatcmpl-test-id" {
			t.Errorf("expected response ID 'chatcmpl-test-id', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "test-model" {
			t.Errorf("expected model ID 'test-model', got %v", result.Response.ModelID)
		}
		if result.Response.Timestamp == nil {
			t.Error("expected non-nil timestamp")
		}
	})
}

func TestChatDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), map[string]string{
			"X-Custom-Header": "custom-value",
		})
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers == nil {
			t.Fatal("expected non-nil headers")
		}
		customHeader := result.Response.Headers["X-Custom-Header"]
		if customHeader != "custom-value" {
			t.Errorf("expected X-Custom-Header 'custom-value', got %q", customHeader)
		}
	})
}

func TestChatDoGenerate_Headers(t *testing.T) {
	t.Run("should pass request headers", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

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

		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header, got %q", capture.Headers.Get("Custom-Request-Header"))
		}
		if capture.Headers.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header, got %q", capture.Headers.Get("Authorization"))
		}
	})
}

func TestChatDoGenerate_RequestBody(t *testing.T) {
	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", body["model"])
		}

		messages, ok := body["messages"].([]any)
		if !ok {
			t.Fatalf("expected messages to be []any, got %T", body["messages"])
		}
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
	})

	t.Run("should include temperature in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)
		temp := 0.5

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Ctx:         context.Background(),
			Temperature: &temp,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["temperature"] != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", body["temperature"])
		}
	})

	t.Run("should include max_tokens in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)
		maxTokens := 100

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			Ctx:             context.Background(),
			MaxOutputTokens: &maxTokens,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["max_tokens"] != float64(100) {
			t.Errorf("expected max_tokens 100, got %v", body["max_tokens"])
		}
	})

	t.Run("should include stop sequences in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:        testPrompt,
			Ctx:           context.Background(),
			StopSequences: []string{"STOP1", "STOP2"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		stop, ok := body["stop"].([]any)
		if !ok {
			t.Fatalf("expected stop to be []any, got %T", body["stop"])
		}
		if len(stop) != 2 {
			t.Fatalf("expected 2 stop sequences, got %d", len(stop))
		}
	})

	t.Run("should include seed in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)
		seed := 42

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Seed:   &seed,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["seed"] != float64(42) {
			t.Errorf("expected seed 42, got %v", body["seed"])
		}
	})
}

func TestChatDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should pass through provider options", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"custom_field": "custom_value",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["custom_field"] != "custom_value" {
			t.Errorf("expected custom_field 'custom_value', got %v", body["custom_field"])
		}
	})

	t.Run("should pass user option through provider options", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"user": "test-user-id",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["user"] != "test-user-id" {
			t.Errorf("expected user 'test-user-id', got %v", body["user"])
		}
	})
}

func TestChatDoGenerate_ReasoningContent(t *testing.T) {
	t.Run("should extract reasoning_content", func(t *testing.T) {
		server, _ := createTestServer(chatReasoningContentFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var reasoning string
		for _, c := range result.Content {
			if r, ok := c.(languagemodel.Reasoning); ok {
				reasoning += r.Text
			}
		}
		if reasoning != "Let me think about this..." {
			t.Errorf("expected reasoning 'Let me think about this...', got %q", reasoning)
		}
	})

	t.Run("should fall back to reasoning field", func(t *testing.T) {
		server, _ := createTestServer(chatReasoningFieldFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var reasoning string
		for _, c := range result.Content {
			if r, ok := c.(languagemodel.Reasoning); ok {
				reasoning += r.Text
			}
		}
		if reasoning != "Using reasoning field..." {
			t.Errorf("expected reasoning 'Using reasoning field...', got %q", reasoning)
		}
	})

	t.Run("should prioritize reasoning_content over reasoning field", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "chatcmpl-both-id",
			"model":   "test-model",
			"created": float64(1700000000),
			"choices": []any{
				map[string]any{
					"index": float64(0),
					"message": map[string]any{
						"role":              "assistant",
						"content":           "Answer",
						"reasoning_content": "Primary reasoning",
						"reasoning":         "Fallback reasoning",
					},
					"finish_reason": "stop",
				},
			},
		}

		server, _ := createTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var reasoning string
		for _, c := range result.Content {
			if r, ok := c.(languagemodel.Reasoning); ok {
				reasoning += r.Text
			}
		}
		if reasoning != "Primary reasoning" {
			t.Errorf("expected 'Primary reasoning', got %q", reasoning)
		}
	})
}

func TestChatDoGenerate_ResponseFormat(t *testing.T) {
	t.Run("should send json_object response format", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:         testPrompt,
			Ctx:            context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		if rf["type"] != "json_object" {
			t.Errorf("expected type 'json_object', got %v", rf["type"])
		}
	})

	t.Run("should send json_schema response format when structured outputs enabled", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		supports := true
		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider:                  "test-provider.chat",
			SupportsStructuredOutputs: &supports,
		})

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
				Name: strPtr("test-schema"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		if rf["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", rf["type"])
		}
		jsonSchema := rf["json_schema"].(map[string]any)
		if jsonSchema["name"] != "test-schema" {
			t.Errorf("expected name 'test-schema', got %v", jsonSchema["name"])
		}
		if jsonSchema["strict"] != true {
			t.Errorf("expected strict true, got %v", jsonSchema["strict"])
		}
		if jsonSchema["schema"] == nil {
			t.Error("expected non-nil schema")
		}
	})

	t.Run("should warn when schema provided without structured outputs", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "responseFormat" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for responseFormat")
		}
	})

	t.Run("should use default name 'response' when name not provided", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		supports := true
		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider:                  "test-provider.chat",
			SupportsStructuredOutputs: &supports,
		})

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		jsonSchema := rf["json_schema"].(map[string]any)
		if jsonSchema["name"] != "response" {
			t.Errorf("expected default name 'response', got %v", jsonSchema["name"])
		}
	})
}

func TestChatDoGenerate_ReasoningEffort(t *testing.T) {
	t.Run("should include reasoningEffort in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"reasoningEffort": "high",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["reasoning_effort"] != "high" {
			t.Errorf("expected reasoning_effort 'high', got %v", body["reasoning_effort"])
		}
	})
}

func TestChatDoGenerate_TextVerbosity(t *testing.T) {
	t.Run("should include verbosity in request body", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"textVerbosity": "verbose",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["verbosity"] != "verbose" {
			t.Errorf("expected verbosity 'verbose', got %v", body["verbosity"])
		}
	})
}

func TestChatDoGenerate_StrictJsonSchema(t *testing.T) {
	t.Run("should set strict to false when strictJsonSchema is false", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		supports := true
		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider:                  "test-provider.chat",
			SupportsStructuredOutputs: &supports,
		})

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{"type": "object"},
			},
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"strictJsonSchema": false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf := body["response_format"].(map[string]any)
		jsonSchema := rf["json_schema"].(map[string]any)
		if jsonSchema["strict"] != false {
			t.Errorf("expected strict false, got %v", jsonSchema["strict"])
		}
	})
}

func TestChatDoGenerate_TopKWarning(t *testing.T) {
	t.Run("should warn about topK", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)
		topK := 10

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			TopK:   &topK,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "topK" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for topK")
		}
	})
}

func TestChatDoGenerate_MetadataExtractor(t *testing.T) {
	t.Run("should pass metadata extractor results to provider metadata", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()

		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider: "test-provider.chat",
			MetadataExtractor: &testMetadataExtractor{
				extractFunc: func(parsedBody interface{}) (*shared.ProviderMetadata, error) {
					return &shared.ProviderMetadata{
						"custom": map[string]any{
							"extractedField": "extracted-value",
						},
					}, nil
				},
			},
		})

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProviderMetadata["custom"] == nil {
			t.Fatal("expected custom provider metadata")
		}
		if result.ProviderMetadata["custom"]["extractedField"] != "extracted-value" {
			t.Errorf("expected extractedField 'extracted-value', got %v", result.ProviderMetadata["custom"]["extractedField"])
		}
	})
}

// testMetadataExtractor is a test implementation of MetadataExtractor.
type testMetadataExtractor struct {
	extractFunc func(parsedBody interface{}) (*shared.ProviderMetadata, error)
}

func (e *testMetadataExtractor) ExtractMetadata(parsedBody interface{}) (*shared.ProviderMetadata, error) {
	if e.extractFunc != nil {
		return e.extractFunc(parsedBody)
	}
	return nil, nil
}

func (e *testMetadataExtractor) CreateStreamExtractor() StreamExtractor {
	return &testStreamExtractor{}
}

type testStreamExtractor struct {
	chunks []interface{}
}

func (e *testStreamExtractor) ProcessChunk(chunk interface{}) {
	e.chunks = append(e.chunks, chunk)
}

func (e *testStreamExtractor) BuildMetadata() *shared.ProviderMetadata {
	return nil
}

// --- DoStream tests ---

func TestChatDoStream_Text(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-1","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-1","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":", World!"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-1","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Check for text deltas
		var textDeltas []string
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, td.Delta)
			}
		}
		combined := strings.Join(textDeltas, "")
		if combined != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got %q", combined)
		}

		// Check for finish part
		var finishPart *languagemodel.StreamPartFinish
		for _, p := range parts {
			if fp, ok := p.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected FinishReasonStop, got %v", finishPart.FinishReason.Unified)
		}
	})
}

func TestChatDoStream_ToolCall(t *testing.T) {
	t.Run("should stream tool call deltas", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-tc","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call-1","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-tc","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"ci"}}]}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-tc","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ty\":\""}}]}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-tc","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"SF\"}"}}]}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-tc","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Check for tool call start
		var toolStartCount int
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				toolStartCount++
			}
		}
		if toolStartCount != 1 {
			t.Errorf("expected 1 tool input start, got %d", toolStartCount)
		}

		// Check for tool call content
		var toolCalls []languagemodel.ToolCall
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCalls = append(toolCalls, tc)
			}
		}
		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
		if toolCalls[0].ToolName != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", toolCalls[0].ToolName)
		}
		if toolCalls[0].ToolCallID != "call-1" {
			t.Errorf("expected tool call ID 'call-1', got %q", toolCalls[0].ToolCallID)
		}

		// Check finish reason
		var finishPart *languagemodel.StreamPartFinish
		for _, p := range parts {
			if fp, ok := p.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonToolCalls {
			t.Errorf("expected FinishReasonToolCalls, got %v", finishPart.FinishReason.Unified)
		}
	})
}

func TestChatDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, map[string]string{
			"X-Stream-Header": "stream-value",
		})
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(result.Stream)

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers == nil {
			t.Fatal("expected non-nil headers")
		}
		if result.Response.Headers["X-Stream-Header"] != "stream-value" {
			t.Errorf("expected X-Stream-Header 'stream-value', got %q", result.Response.Headers["X-Stream-Header"])
		}
	})
}

func TestChatDoStream_IncludeUsage(t *testing.T) {
	t.Run("should include stream_options when includeUsage is true", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":1,"total_tokens":6}}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		includeUsage := true
		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider:     "test-provider.chat",
			IncludeUsage: &includeUsage,
		})

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Verify stream_options was sent
		body := capture.BodyJSON()
		streamOpts, ok := body["stream_options"].(map[string]any)
		if !ok {
			t.Fatal("expected stream_options in request body")
		}
		if streamOpts["include_usage"] != true {
			t.Errorf("expected include_usage true, got %v", streamOpts["include_usage"])
		}

		// Verify usage is in finish part
		var finishPart *languagemodel.StreamPartFinish
		for _, p := range parts {
			if fp, ok := p.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 5 {
			t.Errorf("expected input total 5, got %v", finishPart.Usage.InputTokens.Total)
		}
	})

	t.Run("should not include stream_options when includeUsage is nil", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if _, ok := body["stream_options"]; ok {
			t.Error("expected no stream_options in request body")
		}
	})
}

func TestChatDoStream_ReasoningContent(t *testing.T) {
	t.Run("should stream reasoning content with reasoning_content field", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"reasoning_content":"Think"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"reasoning_content":"ing..."}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Answer"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Check reasoning parts
		var reasoningDeltas []string
		for _, p := range parts {
			if rd, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, rd.Delta)
			}
		}
		combined := strings.Join(reasoningDeltas, "")
		if combined != "Thinking..." {
			t.Errorf("expected reasoning 'Thinking...', got %q", combined)
		}

		// Verify reasoning start and end
		var hasReasoningStart, hasReasoningEnd bool
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartReasoningStart); ok {
				hasReasoningStart = true
			}
			if _, ok := p.(languagemodel.StreamPartReasoningEnd); ok {
				hasReasoningEnd = true
			}
		}
		if !hasReasoningStart {
			t.Error("expected reasoning start part")
		}
		if !hasReasoningEnd {
			t.Error("expected reasoning end part")
		}

		// Check text comes after reasoning
		var textDeltas []string
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, td.Delta)
			}
		}
		if strings.Join(textDeltas, "") != "Answer" {
			t.Errorf("expected text 'Answer', got %q", strings.Join(textDeltas, ""))
		}
	})

	t.Run("should fall back to reasoning field for stream", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"reasoning":"Fallback reasoning"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Answer"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var reasoningDeltas []string
		for _, p := range parts {
			if rd, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, rd.Delta)
			}
		}
		if strings.Join(reasoningDeltas, "") != "Fallback reasoning" {
			t.Errorf("expected reasoning 'Fallback reasoning', got %q", strings.Join(reasoningDeltas, ""))
		}
	})

	t.Run("should prioritize reasoning_content over reasoning in stream", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"reasoning_content":"Primary","reasoning":"Fallback"}}]}` + "\n\n",
			`data: {"id":"chatcmpl-stream-r","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var reasoningDeltas []string
		for _, p := range parts {
			if rd, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, rd.Delta)
			}
		}
		if strings.Join(reasoningDeltas, "") != "Primary" {
			t.Errorf("expected reasoning 'Primary', got %q", strings.Join(reasoningDeltas, ""))
		}
	})
}

func TestChatDoStream_RequestBody(t *testing.T) {
	t.Run("should set stream=true in request body", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Errorf("expected stream=true, got %v", body["stream"])
		}
	})

	t.Run("should pass headers in streaming request", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Stream-Request": strPtr("stream-request-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		if capture.Headers.Get("X-Stream-Request") != "stream-request-value" {
			t.Errorf("expected X-Stream-Request header, got %q", capture.Headers.Get("X-Stream-Request"))
		}
	})
}

func TestChatDoStream_ResponseMetadata(t *testing.T) {
	t.Run("should emit response metadata from first chunk", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"chatcmpl-stream-meta","model":"test-model","created":1700000000,"choices":[{"index":0,"delta":{"content":"Hi"}}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var metaPart *languagemodel.StreamPartResponseMetadata
		for _, p := range parts {
			if mp, ok := p.(languagemodel.StreamPartResponseMetadata); ok {
				metaPart = &mp
			}
		}
		if metaPart == nil {
			t.Fatal("expected response metadata part")
		}
		if metaPart.ID == nil || *metaPart.ID != "chatcmpl-stream-meta" {
			t.Errorf("expected ID 'chatcmpl-stream-meta', got %v", metaPart.ID)
		}
		if metaPart.ModelID == nil || *metaPart.ModelID != "test-model" {
			t.Errorf("expected ModelID 'test-model', got %v", metaPart.ModelID)
		}
	})
}

func TestChatDoGenerate_DeprecatedProviderOptions(t *testing.T) {
	t.Run("should warn about deprecated openai-compatible key", func(t *testing.T) {
		server, _ := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"openai-compatible": {
					"user": "test-user",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if ow, ok := w.(shared.OtherWarning); ok {
				if strings.Contains(ow.Message, "deprecated") {
					hasWarning = true
					break
				}
			}
		}
		if !hasWarning {
			t.Error("expected deprecation warning for 'openai-compatible' key")
		}
	})
}

func TestChatDoGenerate_TransformRequestBody(t *testing.T) {
	t.Run("should apply request body transform", func(t *testing.T) {
		server, capture := createTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithConfig(server.URL, ChatConfig{
			Provider: "test-provider.chat",
			TransformRequestBody: func(body map[string]any) map[string]any {
				body["custom_transform"] = "transformed"
				return body
			},
		})

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["custom_transform"] != "transformed" {
			t.Errorf("expected custom_transform 'transformed', got %v", body["custom_transform"])
		}
	})
}
