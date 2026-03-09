// Ported from: packages/xai/src/xai-chat-language-model.test.ts
package xai

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
	"github.com/brainlet/brainkit/ai-kit/providerutils"
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

// createTestServer creates a JSON test server that returns the given body and captures requests.
func createTestServer(body map[string]any, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header.Clone()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

// createSSETestServer creates an SSE test server that streams chunks.
func createSSETestServer(chunks []string, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header.Clone()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, _ := w.(http.Flusher)
		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			if flusher != nil {
				flusher.Flush()
			}
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		if flusher != nil {
			flusher.Flush()
		}
	}))
	return server, capture
}

// createModel creates an xAI chat model pointing at a test server.
func createModel(serverURL string, modelId ...string) *XaiChatLanguageModel {
	id := "grok-4-fast-non-reasoning"
	if len(modelId) > 0 {
		id = modelId[0]
	}
	idCounter := 0
	return NewXaiChatLanguageModel(id, XaiChatConfig{
		Provider: "xai.chat",
		BaseURL:  serverURL,
		Headers:  func() map[string]string { return map[string]string{"Authorization": "Bearer test-key"} },
		GenerateID: func() string {
			result := fmt.Sprintf("id-%d", idCounter)
			idCounter++
			return result
		},
	})
}

// collectStreamParts reads all parts from a stream channel.
func collectStreamParts(stream <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for part := range stream {
		parts = append(parts, part)
	}
	return parts
}

// xaiTextFixture returns a basic text response.
func xaiTextFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-123",
		"object":  "chat.completion",
		"created": float64(1700000000),
		"model":   "grok-4-fast-non-reasoning",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": "Hello, world!",
				},
				"logprobs":      nil,
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(10),
			"completion_tokens": float64(20),
			"total_tokens":      float64(30),
		},
	}
}

// xaiToolCallFixture returns a tool call response.
func xaiToolCallFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-456",
		"object":  "chat.completion",
		"created": float64(1700000000),
		"model":   "grok-4-fast-non-reasoning",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{
						map[string]any{
							"id":   "call_abc",
							"type": "function",
							"function": map[string]any{
								"name":      "weather",
								"arguments": `{"location":"sf"}`,
							},
						},
					},
				},
				"logprobs":      nil,
				"finish_reason": "tool_calls",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(50),
			"completion_tokens": float64(10),
			"total_tokens":      float64(60),
		},
	}
}

func TestDoGenerate_TextExtraction(t *testing.T) {
	t.Run("should extract text from response", func(t *testing.T) {
		server, _ := createTestServer(xaiTextFixture(), nil)
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
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Hello, world!" {
			t.Errorf("expected 'Hello, world!', got %q", text.Text)
		}
	})
}

func TestDoGenerate_ToolCalls(t *testing.T) {
	t.Run("should extract tool calls from response", func(t *testing.T) {
		server, _ := createTestServer(xaiToolCallFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) < 1 {
			t.Fatalf("expected at least 1 content, got %d", len(result.Content))
		}

		toolCall, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if toolCall.ToolName != "weather" {
			t.Errorf("expected toolName 'weather', got %q", toolCall.ToolName)
		}
		if toolCall.ToolCallID != "call_abc" {
			t.Errorf("expected toolCallId 'call_abc', got %q", toolCall.ToolCallID)
		}
	})
}

func TestDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage from response", func(t *testing.T) {
		server, _ := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if intVal(result.Usage.InputTokens.Total) != 10 {
			t.Errorf("expected InputTokens.Total 10, got %d", intVal(result.Usage.InputTokens.Total))
		}
	})
}

func TestDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createTestServer(xaiTextFixture(), nil)
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
			t.Fatal("expected non-nil response")
		}
		if result.Response.ResponseMetadata.ModelID == nil || *result.Response.ResponseMetadata.ModelID != "grok-4-fast-non-reasoning" {
			t.Errorf("expected model ID 'grok-4-fast-non-reasoning', got %v", result.Response.ResponseMetadata.ModelID)
		}
	})
}

func TestDoGenerate_Headers(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		customHeader := "custom-value"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Custom": &customHeader,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom 'custom-value', got %q", capture.Headers.Get("X-Custom"))
		}
	})
}

func TestDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should return response headers", func(t *testing.T) {
		server, _ := createTestServer(xaiTextFixture(), map[string]string{
			"X-Response-Custom": "response-value",
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
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers["X-Response-Custom"] != "response-value" {
			t.Errorf("expected X-Response-Custom 'response-value', got %q", result.Response.Headers["X-Response-Custom"])
		}
	})
}

func TestDoGenerate_RequestBody(t *testing.T) {
	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		temp := float64(0.5)
		topP := float64(0.9)
		maxTokens := 100
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "you are helpful"},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "hello"},
					},
				},
			},
			Temperature:     &temp,
			TopP:            &topP,
			MaxOutputTokens: &maxTokens,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "grok-4-fast-non-reasoning" {
			t.Errorf("expected model 'grok-4-fast-non-reasoning', got %v", body["model"])
		}
		if body["temperature"] != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", body["temperature"])
		}
		if body["top_p"] != 0.9 {
			t.Errorf("expected top_p 0.9, got %v", body["top_p"])
		}
		if body["max_completion_tokens"] != float64(100) {
			t.Errorf("expected max_completion_tokens 100, got %v", body["max_completion_tokens"])
		}
	})
}

func TestDoGenerate_ReasoningContent(t *testing.T) {
	t.Run("should extract reasoning content", func(t *testing.T) {
		fixture := xaiTextFixture()
		choices := fixture["choices"].([]any)
		msg := choices[0].(map[string]any)["message"].(map[string]any)
		msg["reasoning_content"] = "Let me think about this..."
		msg["content"] = "The answer is 42."

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

		if len(result.Content) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(result.Content))
		}

		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "The answer is 42." {
			t.Errorf("expected text 'The answer is 42.', got %q", text.Text)
		}

		reasoning, ok := result.Content[1].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[1])
		}
		if reasoning.Text != "Let me think about this..." {
			t.Errorf("expected reasoning 'Let me think about this...', got %q", reasoning.Text)
		}
	})
}

func TestDoGenerate_SearchParameters(t *testing.T) {
	t.Run("should include search parameters in request", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"searchParameters": map[string]interface{}{
						"mode":            "auto",
						"returnCitations": true,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		sp, ok := body["search_parameters"].(map[string]interface{})
		if !ok {
			t.Fatal("expected search_parameters in body")
		}
		if sp["mode"] != "auto" {
			t.Errorf("expected mode 'auto', got %v", sp["mode"])
		}
	})
}

func TestDoGenerate_Citations(t *testing.T) {
	t.Run("should extract citations from response", func(t *testing.T) {
		fixture := xaiTextFixture()
		fixture["citations"] = []any{"https://example.com", "https://test.com"}

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

		// Check for source content
		var sourceCount int
		for _, c := range result.Content {
			if _, ok := c.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount != 2 {
			t.Errorf("expected 2 sources, got %d", sourceCount)
		}
	})
}

func TestDoGenerate_ResponseFormat(t *testing.T) {
	t.Run("should send json_schema response format", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		name := "recipe"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Name: &name,
				Schema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"name": map[string]any{"type": "string"}},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf, ok := body["response_format"].(map[string]interface{})
		if !ok {
			t.Fatal("expected response_format in body")
		}
		if rf["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", rf["type"])
		}
	})

	t.Run("should send json_object response format when no schema", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:         testPrompt,
			Ctx:            context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf, ok := body["response_format"].(map[string]interface{})
		if !ok {
			t.Fatal("expected response_format in body")
		}
		if rf["type"] != "json_object" {
			t.Errorf("expected type 'json_object', got %v", rf["type"])
		}
	})
}

func TestDoGenerate_Logprobs(t *testing.T) {
	t.Run("should include logprobs in request body", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"logprobs":    true,
					"topLogprobs": float64(5),
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["logprobs"] != true {
			t.Errorf("expected logprobs true, got %v", body["logprobs"])
		}
		if body["top_logprobs"] != float64(5) {
			t.Errorf("expected top_logprobs 5, got %v", body["top_logprobs"])
		}
	})
}

func TestDoGenerate_MissingUsage(t *testing.T) {
	t.Run("should handle missing usage with zero values", func(t *testing.T) {
		fixture := xaiTextFixture()
		delete(fixture, "usage")

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

		if intVal(result.Usage.InputTokens.Total) != 0 {
			t.Errorf("expected InputTokens.Total 0, got %d", intVal(result.Usage.InputTokens.Total))
		}
		if intVal(result.Usage.OutputTokens.Total) != 0 {
			t.Errorf("expected OutputTokens.Total 0, got %d", intVal(result.Usage.OutputTokens.Total))
		}
	})
}

func TestDoGenerate_FinishReason(t *testing.T) {
	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected finish reason 'stop', got %v", result.FinishReason.Unified)
		}
	})
}

func TestDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should include reasoning effort in request", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
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

// ---- DoStream Tests ----

func TestDoStream_TextStreaming(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`,
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var textDeltas []string
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, td.Delta)
			}
		}

		if len(textDeltas) < 2 {
			t.Fatalf("expected at least 2 text deltas, got %d", len(textDeltas))
		}
	})
}

func TestDoStream_ToolCallStreaming(t *testing.T) {
	t.Run("should stream tool call deltas", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"weather","arguments":""}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"ation\":\"sf\"}"}}]},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":50,"completion_tokens":10,"total_tokens":60}}`,
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasToolStart, hasToolEnd bool
		var toolInputDeltas []string
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				hasToolStart = true
			}
			if _, ok := p.(languagemodel.StreamPartToolInputEnd); ok {
				hasToolEnd = true
			}
			if td, ok := p.(languagemodel.StreamPartToolInputDelta); ok {
				toolInputDeltas = append(toolInputDeltas, td.Delta)
			}
		}

		if !hasToolStart {
			t.Error("expected tool-input-start")
		}
		if !hasToolEnd {
			t.Error("expected tool-input-end")
		}
		if len(toolInputDeltas) == 0 {
			t.Error("expected at least one tool input delta")
		}
	})
}

func TestDoStream_Headers(t *testing.T) {
	t.Run("should pass custom headers to stream request", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11}}`,
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		customHeader := "stream-custom"
		_, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Stream-Custom": &customHeader,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Stream-Custom") != "stream-custom" {
			t.Errorf("expected X-Stream-Custom 'stream-custom', got %q", capture.Headers.Get("X-Stream-Custom"))
		}
	})
}

func TestDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should return response headers from stream", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11}}`,
		}

		server, _ := createSSETestServer(chunks, map[string]string{
			"X-Stream-Response": "stream-resp",
		})
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Consume the stream
		_ = collectStreamParts(result.Stream)

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers["X-Stream-Response"] != "stream-resp" {
			t.Errorf("expected X-Stream-Response 'stream-resp', got %q", result.Response.Headers["X-Stream-Response"])
		}
	})
}

func TestDoStream_RequestBody(t *testing.T) {
	t.Run("should include stream=true in request body", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11}}`,
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Consume the stream to ensure request was made
		_ = collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
	})
}

func TestDoStream_ReasoningStreaming(t *testing.T) {
	t.Run("should stream reasoning content", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"role":"assistant","reasoning_content":"Let me think"},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"reasoning_content":"..."},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"The answer is 42."},"finish_reason":null}]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`,
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasReasoningStart, hasReasoningEnd bool
		var reasoningDeltas []string
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartReasoningStart); ok {
				hasReasoningStart = true
			}
			if _, ok := p.(languagemodel.StreamPartReasoningEnd); ok {
				hasReasoningEnd = true
			}
			if rd, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, rd.Delta)
			}
		}

		if !hasReasoningStart {
			t.Error("expected reasoning-start")
		}
		if !hasReasoningEnd {
			t.Error("expected reasoning-end")
		}
		if len(reasoningDeltas) == 0 {
			t.Error("expected at least one reasoning delta")
		}
	})
}

func TestDoStream_CitationsInStream(t *testing.T) {
	t.Run("should stream citations as sources", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"test"},"finish_reason":null}],"citations":["https://example.com"]}`,
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":1,"total_tokens":11}}`,
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var sourceCount int
		for _, p := range parts {
			if _, ok := p.(languagemodel.SourceURL); ok {
				sourceCount++
			}
		}
		if sourceCount == 0 {
			t.Error("expected at least one source in stream")
		}
	})
}

func TestDoGenerate_SupportedUrls(t *testing.T) {
	t.Run("should return supported URL patterns", func(t *testing.T) {
		model := NewXaiChatLanguageModel("grok-4-fast-non-reasoning", XaiChatConfig{
			Provider: "xai.chat",
			BaseURL:  "https://api.x.ai/v1",
			Headers: func() map[string]string {
				return map[string]string{}
			},
			GenerateID: providerutils.GenerateId,
		})

		urls, err := model.SupportedUrls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if urls == nil {
			t.Fatal("expected non-nil supported URLs")
		}
		patterns, ok := urls["image/*"]
		if !ok || len(patterns) == 0 {
			t.Fatal("expected image/* patterns")
		}

		// Should match http/https URLs
		if !patterns[0].MatchString("https://example.com/image.jpg") {
			t.Error("expected URL pattern to match https URLs")
		}
	})
}

func TestDoGenerate_ToolsAndToolChoice(t *testing.T) {
	t.Run("should include tools in request body", func(t *testing.T) {
		server, capture := createTestServer(xaiTextFixture(), nil)
		defer server.Close()

		model := createModel(server.URL)
		desc := "Get weather"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: &desc,
					InputSchema: map[string]any{
						"type":       "object",
						"properties": map[string]any{"location": map[string]any{"type": "string"}},
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceAuto{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		tools, ok := body["tools"].([]interface{})
		if !ok || len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %v", body["tools"])
		}
		if body["tool_choice"] != "auto" {
			t.Errorf("expected tool_choice 'auto', got %v", body["tool_choice"])
		}
	})
}

func TestDoGenerate_ModelId(t *testing.T) {
	t.Run("should return correct model ID", func(t *testing.T) {
		model := NewXaiChatLanguageModel("grok-4-fast-non-reasoning", XaiChatConfig{
			Provider: "xai.chat",
			BaseURL:  "https://api.x.ai/v1",
			Headers: func() map[string]string {
				return map[string]string{}
			},
			GenerateID: providerutils.GenerateId,
		})

		if model.ModelID() != "grok-4-fast-non-reasoning" {
			t.Errorf("expected 'grok-4-fast-non-reasoning', got %q", model.ModelID())
		}
	})
}

func TestDoGenerate_Provider(t *testing.T) {
	t.Run("should return correct provider", func(t *testing.T) {
		model := NewXaiChatLanguageModel("grok-4-fast-non-reasoning", XaiChatConfig{
			Provider: "xai.chat",
			BaseURL:  "https://api.x.ai/v1",
			Headers: func() map[string]string {
				return map[string]string{}
			},
			GenerateID: providerutils.GenerateId,
		})

		if model.Provider() != "xai.chat" {
			t.Errorf("expected 'xai.chat', got %q", model.Provider())
		}
	})
}

func TestDoStream_MissingUsage(t *testing.T) {
	t.Run("should handle missing usage in streaming response", func(t *testing.T) {
		chunks := []string{
			`{"id":"chatcmpl-123","object":"chat.completion.chunk","created":1700000000,"model":"grok-4-fast-non-reasoning","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":"stop"}]}`,
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()

		model := createModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var finishPart *languagemodel.StreamPartFinish
		for _, p := range parts {
			if fp, ok := p.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}

		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		// Should have zero usage values
		if intVal(finishPart.Usage.InputTokens.Total) != 0 {
			t.Errorf("expected InputTokens.Total 0, got %d", intVal(finishPart.Usage.InputTokens.Total))
		}
	})
}

// Test helper: verify string in captured body
func assertBodyContains(t *testing.T, capture *requestCapture, key string) {
	t.Helper()
	if !strings.Contains(string(capture.Body), key) {
		t.Errorf("expected body to contain %q, body: %s", key, string(capture.Body))
	}
}
