// Ported from: packages/openai/src/chat/openai-chat-language-model.test.ts
package openai

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

// --- Test infrastructure ---

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

// createBinaryTestServer creates a test server that returns raw bytes.
func createBinaryTestServer(data []byte, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}))
	return server, capture
}

// createFormDataTestServer creates a test server for multipart form data that returns JSON.
func createFormDataTestServer(body any, headers map[string]string) (*httptest.Server, *requestCapture) {
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

func createChatModel(baseURL string) *OpenAIChatLanguageModel {
	return NewOpenAIChatLanguageModel("gpt-3.5-turbo", OpenAIChatConfig{
		Provider: "openai.chat",
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return baseURL + options.Path
		},
		Headers: func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-api-key",
				"Content-Type":  "application/json",
			}
		},
	})
}

func createChatModelWithID(baseURL, modelID string) *OpenAIChatLanguageModel {
	return NewOpenAIChatLanguageModel(modelID, OpenAIChatConfig{
		Provider: "openai.chat",
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return baseURL + options.Path
		},
		Headers: func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-api-key",
				"Content-Type":  "application/json",
			}
		},
	})
}

// chatTextFixture returns a basic text response fixture.
func chatTextFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-95ZTZkhr0mHNKqerQfiwkuox3PHAd",
		"object":  "chat.completion",
		"created": float64(1711115037),
		"model":   "gpt-3.5-turbo-0125",
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
			"total_tokens":      float64(34),
			"completion_tokens": float64(30),
		},
	}
}

// chatToolCallFixture returns a tool call response fixture.
func chatToolCallFixture() map[string]any {
	return map[string]any{
		"id":      "chatcmpl-tool-123",
		"object":  "chat.completion",
		"created": float64(1711115037),
		"model":   "gpt-3.5-turbo-0125",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role": "assistant",
					"tool_calls": []any{
						map[string]any{
							"id":   "call_abc123",
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
			"prompt_tokens":     float64(50),
			"total_tokens":      float64(65),
			"completion_tokens": float64(15),
		},
	}
}

func collectStreamParts(stream <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for part := range stream {
		parts = append(parts, part)
	}
	return parts
}

// --- DoGenerate tests ---

func TestChatDoGenerate_TextResponse(t *testing.T) {
	t.Run("should extract text response", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content item, got %d", len(result.Content))
		}
		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if text.Text != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", text.Text)
		}
	})
}

func TestChatDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), nil)
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
			t.Errorf("expected input tokens 4, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 30 {
			t.Errorf("expected output tokens 30, got %v", result.Usage.OutputTokens.Total)
		}
	})

	t.Run("should extract usage with cached tokens", func(t *testing.T) {
		fixture := chatTextFixture()
		fixture["usage"] = map[string]any{
			"prompt_tokens":     float64(20),
			"completion_tokens": float64(5),
			"total_tokens":      float64(25),
			"prompt_tokens_details": map[string]any{
				"cached_tokens": float64(10),
			},
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": float64(2),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

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
		if result.Usage.InputTokens.CacheRead == nil || *result.Usage.InputTokens.CacheRead != 10 {
			t.Errorf("expected cache read 10, got %v", result.Usage.InputTokens.CacheRead)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 10 {
			t.Errorf("expected no-cache 10, got %v", result.Usage.InputTokens.NoCache)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 2 {
			t.Errorf("expected reasoning tokens 2, got %v", result.Usage.OutputTokens.Reasoning)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 3 {
			t.Errorf("expected text tokens 3 (5-2), got %v", result.Usage.OutputTokens.Text)
		}
	})
}

func TestChatDoGenerate_FinishReason(t *testing.T) {
	tests := []struct {
		name           string
		finishReason   string
		expectedUnified languagemodel.UnifiedFinishReason
	}{
		{"stop", "stop", languagemodel.FinishReasonStop},
		{"length", "length", languagemodel.FinishReasonLength},
		{"content_filter", "content_filter", languagemodel.FinishReasonContentFilter},
		{"tool_calls", "tool_calls", languagemodel.FinishReasonToolCalls},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := chatTextFixture()
			choices := fixture["choices"].([]any)
			choices[0].(map[string]any)["finish_reason"] = tt.finishReason

			server, _ := createJSONTestServer(fixture, nil)
			defer server.Close()
			model := createChatModel(server.URL)

			result, err := model.DoGenerate(languagemodel.CallOptions{
				Prompt: testPrompt,
				Ctx:    context.Background(),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.FinishReason.Unified != tt.expectedUnified {
				t.Errorf("expected finish reason %v, got %v", tt.expectedUnified, result.FinishReason.Unified)
			}
		})
	}
}

func TestChatDoGenerate_ToolCalls(t *testing.T) {
	t.Run("should extract tool calls", func(t *testing.T) {
		server, _ := createJSONTestServer(chatToolCallFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content item, got %d", len(result.Content))
		}
		tc, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if tc.ToolCallID != "call_abc123" {
			t.Errorf("expected tool call ID 'call_abc123', got %q", tc.ToolCallID)
		}
		if tc.ToolName != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", tc.ToolName)
		}
		if tc.Input != `{"city":"San Francisco"}` {
			t.Errorf("expected input '{\"city\":\"San Francisco\"}', got %q", tc.Input)
		}
	})
}

func TestChatDoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass model and messages", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
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
		if body["model"] != "gpt-3.5-turbo" {
			t.Errorf("expected model 'gpt-3.5-turbo', got %v", body["model"])
		}
		messages, ok := body["messages"].([]any)
		if !ok {
			t.Fatalf("expected messages array, got %T", body["messages"])
		}
		if len(messages) == 0 {
			t.Fatal("expected at least 1 message")
		}
	})

	t.Run("should pass temperature", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		temp := 0.5
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["temperature"] != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", body["temperature"])
		}
	})

	t.Run("should pass max_tokens", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		maxTokens := 100
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			MaxOutputTokens: &maxTokens,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["max_tokens"] != float64(100) {
			t.Errorf("expected max_tokens 100, got %v", body["max_tokens"])
		}
	})

	t.Run("should pass top_p", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		topP := 0.9
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			TopP:   &topP,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["top_p"] != 0.9 {
			t.Errorf("expected top_p 0.9, got %v", body["top_p"])
		}
	})

	t.Run("should pass frequency_penalty", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		fp := 0.5
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:           testPrompt,
			FrequencyPenalty: &fp,
			Ctx:              context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["frequency_penalty"] != 0.5 {
			t.Errorf("expected frequency_penalty 0.5, got %v", body["frequency_penalty"])
		}
	})

	t.Run("should pass presence_penalty", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		pp := 0.5
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			PresencePenalty: &pp,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["presence_penalty"] != 0.5 {
			t.Errorf("expected presence_penalty 0.5, got %v", body["presence_penalty"])
		}
	})

	t.Run("should pass stop sequences", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:        testPrompt,
			StopSequences: []string{"END", "STOP"},
			Ctx:           context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		stop, ok := body["stop"].([]any)
		if !ok {
			t.Fatalf("expected stop array, got %T", body["stop"])
		}
		if len(stop) != 2 {
			t.Errorf("expected 2 stop sequences, got %d", len(stop))
		}
	})

	t.Run("should pass seed", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		seed := 42
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Seed:   &seed,
			Ctx:    context.Background(),
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

func TestChatDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), map[string]string{
			"X-Custom-Header": "test-value",
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
		if result.Response.Headers["X-Custom-Header"] != "test-value" {
			t.Errorf("expected X-Custom-Header 'test-value', got %q", result.Response.Headers["X-Custom-Header"])
		}
	})
}

func TestChatDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), nil)
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
		if result.Response.ID == nil || *result.Response.ID != "chatcmpl-95ZTZkhr0mHNKqerQfiwkuox3PHAd" {
			t.Errorf("expected response ID, got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "gpt-3.5-turbo-0125" {
			t.Errorf("expected model ID 'gpt-3.5-turbo-0125', got %v", result.Response.ModelID)
		}
	})
}

func TestChatDoGenerate_Logprobs(t *testing.T) {
	t.Run("should extract logprobs", func(t *testing.T) {
		fixture := chatTextFixture()
		choices := fixture["choices"].([]any)
		choices[0].(map[string]any)["logprobs"] = map[string]any{
			"content": []any{
				map[string]any{
					"token":   "Hello",
					"logprob": -0.0009994634,
					"top_logprobs": []any{
						map[string]any{
							"token":   "Hello",
							"logprob": -0.0009994634,
						},
					},
				},
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		logprobs, ok := result.ProviderMetadata["openai"]["logprobs"]
		if !ok {
			t.Fatal("expected logprobs in provider metadata")
		}
		logprobArr, ok := logprobs.([]openaiLogprobContent)
		if !ok {
			t.Fatalf("expected []openaiLogprobContent, got %T", logprobs)
		}
		if len(logprobArr) != 1 {
			t.Fatalf("expected 1 logprob entry, got %d", len(logprobArr))
		}
		if logprobArr[0].Token != "Hello" {
			t.Errorf("expected token 'Hello', got %q", logprobArr[0].Token)
		}
	})
}

func TestChatDoGenerate_Annotations(t *testing.T) {
	t.Run("should extract URL citations", func(t *testing.T) {
		fixture := chatTextFixture()
		choices := fixture["choices"].([]any)
		choices[0].(map[string]any)["message"].(map[string]any)["annotations"] = []any{
			map[string]any{
				"type": "url_citation",
				"url_citation": map[string]any{
					"start_index": float64(0),
					"end_index":   float64(10),
					"url":         "https://example.com",
					"title":       "Example",
				},
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Content should have text + source URL
		found := false
		for _, c := range result.Content {
			if src, ok := c.(languagemodel.SourceURL); ok {
				found = true
				if src.URL != "https://example.com" {
					t.Errorf("expected URL 'https://example.com', got %q", src.URL)
				}
				if src.Title == nil || *src.Title != "Example" {
					t.Errorf("expected title 'Example', got %v", src.Title)
				}
			}
		}
		if !found {
			t.Error("expected SourceURL in content")
		}
	})
}

func TestChatDoGenerate_ResponseFormat(t *testing.T) {
	t.Run("should pass json_object format", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:         testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{},
			Ctx:            context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("expected response_format map, got %T", body["response_format"])
		}
		if rf["type"] != "json_object" {
			t.Errorf("expected type 'json_object', got %v", rf["type"])
		}
	})

	t.Run("should pass json_schema format with schema", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		name := "test_schema"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
				Name: &name,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		rf, ok := body["response_format"].(map[string]any)
		if !ok {
			t.Fatalf("expected response_format map, got %T", body["response_format"])
		}
		if rf["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", rf["type"])
		}
		jsonSchema, ok := rf["json_schema"].(map[string]any)
		if !ok {
			t.Fatalf("expected json_schema map")
		}
		if jsonSchema["name"] != "test_schema" {
			t.Errorf("expected name 'test_schema', got %v", jsonSchema["name"])
		}
		if jsonSchema["strict"] != true {
			t.Errorf("expected strict true, got %v", jsonSchema["strict"])
		}
	})
}

func TestChatDoGenerate_ResponseFormatText(t *testing.T) {
	t.Run("should not send response_format for text format", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:         testPrompt,
			ResponseFormat: languagemodel.ResponseFormatText{},
			Ctx:            context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, exists := body["response_format"]; exists {
			t.Errorf("expected response_format to NOT be sent for text format, but it was: %v", body["response_format"])
		}
	})
}

func TestChatDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should pass user", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"user": "test-user",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["user"] != "test-user" {
			t.Errorf("expected user 'test-user', got %v", body["user"])
		}
	})

	t.Run("should pass parallelToolCalls", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"parallelToolCalls": false,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["parallel_tool_calls"] != false {
			t.Errorf("expected parallel_tool_calls false, got %v", body["parallel_tool_calls"])
		}
	})

	t.Run("should pass reasoningEffort", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o1")

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"reasoningEffort": "medium",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["reasoning_effort"] != "medium" {
			t.Errorf("expected reasoning_effort 'medium', got %v", body["reasoning_effort"])
		}
	})

	t.Run("should pass serviceTier", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "auto",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["service_tier"] != "auto" {
			t.Errorf("expected service_tier 'auto', got %v", body["service_tier"])
		}
	})
}

func TestChatDoGenerate_PredictionTokens(t *testing.T) {
	t.Run("should extract prediction tokens from provider metadata", func(t *testing.T) {
		fixture := chatTextFixture()
		fixture["usage"] = map[string]any{
			"prompt_tokens":     float64(10),
			"completion_tokens": float64(20),
			"total_tokens":      float64(30),
			"completion_tokens_details": map[string]any{
				"accepted_prediction_tokens": float64(5),
				"rejected_prediction_tokens": float64(3),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		openaiMeta := result.ProviderMetadata["openai"]
		if openaiMeta["acceptedPredictionTokens"] != 5 {
			t.Errorf("expected acceptedPredictionTokens 5, got %v (%T)", openaiMeta["acceptedPredictionTokens"], openaiMeta["acceptedPredictionTokens"])
		}
		if openaiMeta["rejectedPredictionTokens"] != 3 {
			t.Errorf("expected rejectedPredictionTokens 3, got %v (%T)", openaiMeta["rejectedPredictionTokens"], openaiMeta["rejectedPredictionTokens"])
		}
	})
}

func TestChatDoGenerate_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		headerVal := "custom-value"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Headers: map[string]*string{
				"X-Custom-Header": &headerVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Custom-Header") != "custom-value" {
			t.Errorf("expected X-Custom-Header 'custom-value', got %q", capture.Headers.Get("X-Custom-Header"))
		}
	})
}

func TestChatDoGenerate_Warnings(t *testing.T) {
	t.Run("should warn about unsupported topK", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		topK := 5
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			TopK:   &topK,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "topK" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for topK")
		}
	})
}

func TestChatDoGenerate_ReasoningModel(t *testing.T) {
	t.Run("should remove temperature for reasoning models", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o1")

		temp := 0.5
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["temperature"]; ok {
			t.Error("expected temperature to be removed for reasoning model")
		}

		// Should have a warning
		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "temperature" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for temperature")
		}
	})

	t.Run("should convert max_tokens to max_completion_tokens for reasoning models", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o1")

		maxTokens := 100
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			MaxOutputTokens: &maxTokens,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["max_tokens"]; ok {
			t.Error("expected max_tokens to be removed for reasoning model")
		}
		if body["max_completion_tokens"] != float64(100) {
			t.Errorf("expected max_completion_tokens 100, got %v", body["max_completion_tokens"])
		}
	})

	t.Run("should use developer messages for reasoning models", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o1")

		prompt := languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "You are helpful"},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatal("expected messages")
		}
		firstMsg := messages[0].(map[string]any)
		if firstMsg["role"] != "developer" {
			t.Errorf("expected role 'developer' for system message in reasoning model, got %v", firstMsg["role"])
		}
	})
}

func TestChatDoGenerate_Tools(t *testing.T) {
	t.Run("should pass tools in request body", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		desc := "Get weather"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "get_weather",
					Description: &desc,
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"city": map[string]any{"type": "string"},
						},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		tools, ok := body["tools"].([]any)
		if !ok {
			t.Fatalf("expected tools array, got %T", body["tools"])
		}
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
	})

	t.Run("should pass tool_choice", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		desc := "Get weather"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "get_weather",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			ToolChoice: languagemodel.ToolChoiceRequired{},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["tool_choice"] != "required" {
			t.Errorf("expected tool_choice 'required', got %v", body["tool_choice"])
		}
	})
}

// --- DoStream tests ---

func TestChatDoStream_TextStreaming(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo-0125\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo-0125\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo-0125\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\", World!\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo-0125\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo-0125\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":6,\"total_tokens\":10}}\n\n",
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
		textDeltas := []string{}
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, td.Delta)
			}
		}

		combined := strings.Join(textDeltas, "")
		if combined != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", combined)
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
			t.Errorf("expected stop finish reason, got %v", finishPart.FinishReason.Unified)
		}
	})
}

func TestChatDoStream_ToolCallStreaming(t *testing.T) {
	t.Run("should stream tool call deltas", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"tool_calls\":[{\"index\":0,\"id\":\"call_abc\",\"type\":\"function\",\"function\":{\"name\":\"get_weather\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"ci\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"ty\\\":\\\"SF\\\"}\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711115037,\"model\":\"gpt-3.5-turbo\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n",
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

		// Check for tool input start
		var toolStart *languagemodel.StreamPartToolInputStart
		for _, p := range parts {
			if ts, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				toolStart = &ts
			}
		}
		if toolStart == nil {
			t.Fatal("expected tool input start")
		}
		if toolStart.ToolName != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", toolStart.ToolName)
		}
		if toolStart.ID != "call_abc" {
			t.Errorf("expected tool call ID 'call_abc', got %q", toolStart.ID)
		}

		// Check for tool call
		var toolCall *languagemodel.ToolCall
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}
		if toolCall == nil {
			t.Fatal("expected tool call")
		}
		if toolCall.ToolName != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", toolCall.ToolName)
		}
		if toolCall.Input != `{"city":"SF"}` {
			t.Errorf("expected input '{\"city\":\"SF\"}', got %q", toolCall.Input)
		}
	})
}

func TestChatDoStream_RequestBody(t *testing.T) {
	t.Run("should include stream and stream_options in request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
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

		// Drain the stream
		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
		streamOpts, ok := body["stream_options"].(map[string]any)
		if !ok {
			t.Fatalf("expected stream_options map, got %T", body["stream_options"])
		}
		if streamOpts["include_usage"] != true {
			t.Errorf("expected include_usage true, got %v", streamOpts["include_usage"])
		}
	})
}

func TestChatDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers from stream", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
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
		if result.Response.Headers["X-Stream-Header"] != "stream-value" {
			t.Errorf("expected X-Stream-Header 'stream-value', got %q", result.Response.Headers["X-Stream-Header"])
		}
	})
}

func TestChatDoStream_ResponseMetadata(t *testing.T) {
	t.Run("should emit response metadata", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-metadata\",\"model\":\"gpt-3.5-turbo-0125\",\"created\":1711115037,\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-metadata\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
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
		if metaPart.ID == nil || *metaPart.ID != "chatcmpl-metadata" {
			t.Errorf("expected ID 'chatcmpl-metadata', got %v", metaPart.ID)
		}
		if metaPart.ModelID == nil || *metaPart.ModelID != "gpt-3.5-turbo-0125" {
			t.Errorf("expected model 'gpt-3.5-turbo-0125', got %v", metaPart.ModelID)
		}
	})
}

func TestChatDoStream_ErrorChunks(t *testing.T) {
	t.Run("should handle error in stream", func(t *testing.T) {
		chunks := []string{
			"data: {\"error\":{\"message\":\"Rate limit exceeded\",\"code\":429}}\n\n",
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

		var errorPart *languagemodel.StreamPartError
		for _, p := range parts {
			if ep, ok := p.(languagemodel.StreamPartError); ok {
				errorPart = &ep
			}
		}
		if errorPart == nil {
			t.Fatal("expected error part in stream")
		}
	})
}

func TestChatDoStream_RawChunks(t *testing.T) {
	t.Run("should include raw chunks when enabled", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		includeRaw := true
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt:           testPrompt,
			IncludeRawChunks: &includeRaw,
			Ctx:              context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		rawCount := 0
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartRaw); ok {
				rawCount++
			}
		}
		if rawCount == 0 {
			t.Error("expected raw chunks when includeRawChunks is true")
		}
	})
}

func TestChatDoStream_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers in stream request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		headerVal := "stream-custom"
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Headers: map[string]*string{
				"X-Stream-Custom": &headerVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		if capture.Headers.Get("X-Stream-Custom") != "stream-custom" {
			t.Errorf("expected X-Stream-Custom 'stream-custom', got %q", capture.Headers.Get("X-Stream-Custom"))
		}
	})
}

func TestChatDoStream_Usage(t *testing.T) {
	t.Run("should extract usage from stream", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":6,\"total_tokens\":10}}\n\n",
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

		var finishPart *languagemodel.StreamPartFinish
		for _, p := range parts {
			if fp, ok := p.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 4 {
			t.Errorf("expected input tokens 4, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 6 {
			t.Errorf("expected output tokens 6, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}

// --- Additional provider options tests ---

func TestChatDoGenerate_StoreOption(t *testing.T) {
	t.Run("should send store extension setting", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"store": true,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["store"] != true {
			t.Errorf("expected store true, got %v", body["store"])
		}
	})
}

func TestChatDoGenerate_MetadataOption(t *testing.T) {
	t.Run("should send metadata extension values", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"metadata": map[string]any{
						"custom": "value",
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("expected metadata map, got %T", body["metadata"])
		}
		if metadata["custom"] != "value" {
			t.Errorf("expected metadata custom='value', got %v", metadata["custom"])
		}
	})
}

func TestChatDoGenerate_PredictionOption(t *testing.T) {
	t.Run("should send prediction extension setting", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"prediction": map[string]any{
						"type":    "content",
						"content": "Hello, World!",
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		prediction, ok := body["prediction"].(map[string]any)
		if !ok {
			t.Fatalf("expected prediction map, got %T", body["prediction"])
		}
		if prediction["type"] != "content" {
			t.Errorf("expected prediction type='content', got %v", prediction["type"])
		}
		if prediction["content"] != "Hello, World!" {
			t.Errorf("expected prediction content, got %v", prediction["content"])
		}
	})
}

func TestChatDoGenerate_PromptCacheKeyOption(t *testing.T) {
	t.Run("should send promptCacheKey extension value", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"promptCacheKey": "test-cache-key-123",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["prompt_cache_key"] != "test-cache-key-123" {
			t.Errorf("expected prompt_cache_key, got %v", body["prompt_cache_key"])
		}
	})
}

func TestChatDoGenerate_PromptCacheRetentionOption(t *testing.T) {
	t.Run("should send promptCacheRetention extension value", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"promptCacheRetention": "24h",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["prompt_cache_retention"] != "24h" {
			t.Errorf("expected prompt_cache_retention '24h', got %v", body["prompt_cache_retention"])
		}
	})
}

func TestChatDoGenerate_SafetyIdentifierOption(t *testing.T) {
	t.Run("should send safetyIdentifier extension value", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"safetyIdentifier": "test-safety-id-123",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["safety_identifier"] != "test-safety-id-123" {
			t.Errorf("expected safety_identifier, got %v", body["safety_identifier"])
		}
	})
}

func TestChatDoGenerate_MaxCompletionTokensOption(t *testing.T) {
	t.Run("should send max_completion_tokens extension setting", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"maxCompletionTokens": float64(255),
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["max_completion_tokens"] != float64(255) {
			t.Errorf("expected max_completion_tokens 255, got %v", body["max_completion_tokens"])
		}
	})
}

func TestChatDoGenerate_ForceReasoning(t *testing.T) {
	t.Run("should allow forcing reasoning behavior via providerOptions", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "stealth-reasoning-model")

		temp := 0.5
		topP := 0.7
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			TopP:        &topP,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"forceReasoning": true,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["temperature"]; ok {
			t.Error("expected temperature to be removed when forceReasoning")
		}
		if _, ok := body["top_p"]; ok {
			t.Error("expected top_p to be removed when forceReasoning")
		}

		// Should have warnings for removed settings
		tempWarning := false
		topPWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok {
				if uw.Feature == "temperature" {
					tempWarning = true
				}
				if uw.Feature == "topP" {
					topPWarning = true
				}
			}
		}
		if !tempWarning {
			t.Error("expected warning for temperature")
		}
		if !topPWarning {
			t.Error("expected warning for topP")
		}
	})

	t.Run("should default systemMessageMode to developer when forcing reasoning", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "stealth-reasoning-model")

		prompt := languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"forceReasoning": true,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatal("expected messages")
		}
		firstMsg := messages[0].(map[string]any)
		if firstMsg["role"] != "developer" {
			t.Errorf("expected role 'developer' when forceReasoning, got %v", firstMsg["role"])
		}
	})
}

func TestChatDoGenerate_SystemMessageMode(t *testing.T) {
	t.Run("should allow overriding systemMessageMode via providerOptions", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		prompt := languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"systemMessageMode": "developer",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatal("expected messages")
		}
		firstMsg := messages[0].(map[string]any)
		if firstMsg["role"] != "developer" {
			t.Errorf("expected role 'developer', got %v", firstMsg["role"])
		}
	})

	t.Run("should use default systemMessageMode when not overridden", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		prompt := languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: prompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		messages, ok := body["messages"].([]any)
		if !ok || len(messages) == 0 {
			t.Fatal("expected messages")
		}
		firstMsg := messages[0].(map[string]any)
		if firstMsg["role"] != "system" {
			t.Errorf("expected role 'system' by default, got %v", firstMsg["role"])
		}
	})
}

func TestChatDoGenerate_ServiceTier(t *testing.T) {
	t.Run("should send serviceTier flex processing setting", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "flex",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["service_tier"] != "flex" {
			t.Errorf("expected service_tier 'flex', got %v", body["service_tier"])
		}
	})

	t.Run("should show warning when using flex processing with unsupported model", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "gpt-4o-mini")

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "flex",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["service_tier"]; ok {
			t.Error("expected service_tier to be removed for unsupported model")
		}

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "serviceTier" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for serviceTier flex")
		}
	})

	t.Run("should allow flex processing with o4-mini model without warnings", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "flex",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["service_tier"] != "flex" {
			t.Errorf("expected service_tier 'flex', got %v", body["service_tier"])
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should send serviceTier priority processing setting", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "gpt-4o-mini")

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "priority",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["service_tier"] != "priority" {
			t.Errorf("expected service_tier 'priority', got %v", body["service_tier"])
		}
	})

	t.Run("should show warning when using priority processing with unsupported model", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "gpt-3.5-turbo")

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "priority",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["service_tier"]; ok {
			t.Error("expected service_tier to be removed for unsupported model")
		}

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "serviceTier" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for serviceTier priority")
		}
	})

	t.Run("should allow priority processing with gpt-4o model without warnings", func(t *testing.T) {
		server, _ := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "gpt-4o")

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "priority",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestChatDoGenerate_SearchPreviewTemperature(t *testing.T) {
	searchPreviewModels := []string{
		"gpt-4o-search-preview",
		"gpt-4o-mini-search-preview",
		"gpt-4o-mini-search-preview-2025-03-11",
	}

	for _, modelID := range searchPreviewModels {
		t.Run("should remove temperature for "+modelID, func(t *testing.T) {
			server, capture := createJSONTestServer(chatTextFixture(), nil)
			defer server.Close()
			model := createChatModelWithID(server.URL, modelID)

			temp := 0.7
			result, err := model.DoGenerate(languagemodel.CallOptions{
				Prompt:      testPrompt,
				Temperature: &temp,
				Ctx:         context.Background(),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			body := capture.BodyJSON()
			if _, ok := body["temperature"]; ok {
				t.Error("expected temperature to be removed for search preview model")
			}

			found := false
			for _, w := range result.Warnings {
				if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "temperature" {
					found = true
				}
			}
			if !found {
				t.Error("expected unsupported warning for temperature")
			}
		})
	}
}

func TestChatDoGenerate_ReasoningTokens(t *testing.T) {
	t.Run("should return reasoning tokens in usage", func(t *testing.T) {
		fixture := chatTextFixture()
		fixture["usage"] = map[string]any{
			"prompt_tokens":     float64(15),
			"completion_tokens": float64(20),
			"total_tokens":      float64(35),
			"completion_tokens_details": map[string]any{
				"reasoning_tokens": float64(10),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 10 {
			t.Errorf("expected reasoning tokens 10, got %v", result.Usage.OutputTokens.Reasoning)
		}
	})
}

func TestChatDoGenerate_ReasoningModelAllWarnings(t *testing.T) {
	t.Run("should clear out temperature, top_p, frequency_penalty, presence_penalty and return warnings", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		temp := 0.5
		topP := 0.7
		freqPenalty := 0.2
		presPenalty := 0.3
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:           testPrompt,
			Temperature:      &temp,
			TopP:             &topP,
			FrequencyPenalty: &freqPenalty,
			PresencePenalty:  &presPenalty,
			Ctx:              context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["temperature"]; ok {
			t.Error("expected temperature to be removed")
		}
		if _, ok := body["top_p"]; ok {
			t.Error("expected top_p to be removed")
		}

		warningFeatures := map[string]bool{}
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok {
				warningFeatures[uw.Feature] = true
			}
		}
		for _, feature := range []string{"temperature", "topP", "frequencyPenalty", "presencePenalty"} {
			if !warningFeatures[feature] {
				t.Errorf("expected warning for %s", feature)
			}
		}
	})
}

func TestChatDoStream_ServiceTier(t *testing.T) {
	t.Run("should send serviceTier flex processing setting in streaming", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "o4-mini")

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "flex",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["service_tier"] != "flex" {
			t.Errorf("expected service_tier 'flex', got %v", body["service_tier"])
		}
	})

	t.Run("should send serviceTier priority processing setting in streaming", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModelWithID(server.URL, "gpt-4o")

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"serviceTier": "priority",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["service_tier"] != "priority" {
			t.Errorf("expected service_tier 'priority', got %v", body["service_tier"])
		}
	})
}

func TestChatDoStream_StoreOption(t *testing.T) {
	t.Run("should send store extension setting in streaming", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"store": true,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["store"] != true {
			t.Errorf("expected store true, got %v", body["store"])
		}
	})
}

func TestChatDoStream_MetadataOption(t *testing.T) {
	t.Run("should send metadata extension values in streaming", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"metadata": map[string]any{
						"custom": "value",
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("expected metadata map, got %T", body["metadata"])
		}
		if metadata["custom"] != "value" {
			t.Errorf("expected metadata custom='value', got %v", metadata["custom"])
		}
	})
}

// --- Additional streaming tests ported from TypeScript ---

func TestChatDoStream_AnnotationsCitations(t *testing.T) {
	t.Run("should stream annotations/citations", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":null,\"choices\":[{\"index\":1,\"delta\":{\"content\":\"Based on search results\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":null,\"choices\":[{\"index\":1,\"delta\":{\"annotations\":[{\"type\":\"url_citation\",\"url_citation\":{\"start_index\":24,\"end_index\":29,\"url\":\"https://example.com/doc1.pdf\",\"title\":\"Document 1\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[],\"usage\":{\"prompt_tokens\":17,\"completion_tokens\":227,\"total_tokens\":244}}\n\n",
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

		// Verify we got a source URL part
		var sourceFound bool
		for _, p := range parts {
			if src, ok := p.(languagemodel.SourceURL); ok {
				sourceFound = true
				if src.URL != "https://example.com/doc1.pdf" {
					t.Errorf("expected URL 'https://example.com/doc1.pdf', got %q", src.URL)
				}
				title := "Document 1"
				if src.Title == nil || *src.Title != title {
					t.Errorf("expected title 'Document 1', got %v", src.Title)
				}
			}
		}
		if !sourceFound {
			t.Error("expected a SourceURL stream part from annotations")
		}

		// Verify text deltas were emitted
		var textDeltas []string
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textDeltas = append(textDeltas, td.Delta)
			}
		}
		if len(textDeltas) < 1 {
			t.Error("expected at least one text delta")
		}
	})
}

func TestChatDoStream_ToolCallMissingType(t *testing.T) {
	t.Run("should stream tool call with missing type field (Azure AI Foundry / Mistral)", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-azure-001\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"mistral-large\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null,\"tool_calls\":[{\"index\":0,\"id\":\"call_abc123\",\"function\":{\"name\":\"test-tool\",\"arguments\":\"\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-azure-001\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"mistral-large\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"value\\\"\"}}]},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-azure-001\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"mistral-large\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\":\\\"hello\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n",
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

		var toolStart *languagemodel.StreamPartToolInputStart
		var toolCall *languagemodel.ToolCall
		for _, p := range parts {
			if ts, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				toolStart = &ts
			}
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}

		if toolStart == nil {
			t.Fatal("expected tool input start")
		}
		if toolStart.ID != "call_abc123" {
			t.Errorf("expected tool call ID 'call_abc123', got %q", toolStart.ID)
		}
		if toolStart.ToolName != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got %q", toolStart.ToolName)
		}

		if toolCall == nil {
			t.Fatal("expected tool call")
		}
		if toolCall.ToolCallID != "call_abc123" {
			t.Errorf("expected tool call ID 'call_abc123', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got %q", toolCall.ToolName)
		}
		if toolCall.Input != `{"value":"hello"}` {
			t.Errorf("expected input '{\"value\":\"hello\"}', got %q", toolCall.Input)
		}
	})
}

func TestChatDoStream_ToolCallInOneChunk(t *testing.T) {
	t.Run("should stream tool call that is sent in one chunk", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null,\"tool_calls\":[{\"index\":0,\"id\":\"call_O17Uplv4lJvD6DVdIvFFeRMw\",\"type\":\"function\",\"function\":{\"name\":\"test-tool\",\"arguments\":\"{\\\"value\\\":\\\"Sparkle Day\\\"}\"}}]},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{},\"logprobs\":null,\"finish_reason\":\"tool_calls\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[],\"usage\":{\"prompt_tokens\":53,\"completion_tokens\":17,\"total_tokens\":70}}\n\n",
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

		var toolStart *languagemodel.StreamPartToolInputStart
		var toolCall *languagemodel.ToolCall
		var toolDeltas []string
		for _, p := range parts {
			if ts, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				toolStart = &ts
			}
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
			if td, ok := p.(languagemodel.StreamPartToolInputDelta); ok {
				toolDeltas = append(toolDeltas, td.Delta)
			}
		}

		if toolStart == nil {
			t.Fatal("expected tool input start")
		}
		if toolStart.ToolName != "test-tool" {
			t.Errorf("expected tool name 'test-tool', got %q", toolStart.ToolName)
		}

		if toolCall == nil {
			t.Fatal("expected tool call")
		}
		if toolCall.Input != `{"value":"Sparkle Day"}` {
			t.Errorf("expected input '{\"value\":\"Sparkle Day\"}', got %q", toolCall.Input)
		}

		// All arguments should come in a single delta since the whole call is in one chunk
		if len(toolDeltas) != 1 {
			t.Errorf("expected 1 tool input delta, got %d", len(toolDeltas))
		}
	})
}

func TestChatDoStream_ToolCallArgsInFirstChunk(t *testing.T) {
	t.Run("should stream tool call deltas when arguments are in the first chunk", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":null,\"tool_calls\":[{\"index\":0,\"id\":\"call_O17\",\"type\":\"function\",\"function\":{\"name\":\"test-tool\",\"arguments\":\"{\\\"\"}}]},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"value\"}}]},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\":\\\"test\\\"\"}}]},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"}\"}}]},\"logprobs\":null,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[{\"index\":0,\"delta\":{},\"logprobs\":null,\"finish_reason\":\"tool_calls\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1711357598,\"model\":\"gpt-3.5-turbo-0125\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[],\"usage\":{\"prompt_tokens\":53,\"completion_tokens\":17,\"total_tokens\":70}}\n\n",
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

		var toolCall *languagemodel.ToolCall
		var toolDeltas []string
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
			if td, ok := p.(languagemodel.StreamPartToolInputDelta); ok {
				toolDeltas = append(toolDeltas, td.Delta)
			}
		}

		if toolCall == nil {
			t.Fatal("expected tool call")
		}
		if toolCall.Input != `{"value":"test"}` {
			t.Errorf("expected input '{\"value\":\"test\"}', got %q", toolCall.Input)
		}

		// First delta should contain the initial arguments from the first chunk
		if len(toolDeltas) < 2 {
			t.Errorf("expected at least 2 tool input deltas, got %d", len(toolDeltas))
		}
	})
}

func TestChatDoStream_DuplicateToolCallPrevention(t *testing.T) {
	t.Run("should not duplicate tool calls with empty chunk after completion", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"logprobs\":null,\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":226,\"completion_tokens\":0}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"id\":\"tool-call-001\",\"type\":\"function\",\"index\":0,\"function\":{\"name\":\"searchGoogle\"}}]},\"logprobs\":null,\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":233,\"completion_tokens\":7}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"query\\\": \\\"\"}}]},\"logprobs\":null,\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":241,\"completion_tokens\":15}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"latest news on ai\\\"\"}}]},\"logprobs\":null,\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":245,\"completion_tokens\":19}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"}\"}}]},\"logprobs\":null,\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":245,\"completion_tokens\":19}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\"}}]},\"logprobs\":null,\"finish_reason\":\"tool_calls\",\"stop_reason\":128008}],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":246,\"completion_tokens\":20}}\n\n",
			"data: {\"id\":\"chat-001\",\"object\":\"chat.completion.chunk\",\"created\":1733162241,\"model\":\"meta/llama-3.1-8b-instruct:fp8\",\"choices\":[],\"usage\":{\"prompt_tokens\":226,\"total_tokens\":246,\"completion_tokens\":20}}\n\n",
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

		// Count tool calls - should only be one despite empty chunk
		toolCallCount := 0
		for _, p := range parts {
			if _, ok := p.(languagemodel.ToolCall); ok {
				toolCallCount++
			}
		}
		if toolCallCount != 1 {
			t.Errorf("expected exactly 1 tool call, got %d", toolCallCount)
		}

		// Verify the tool call has correct input
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				if tc.ToolName != "searchGoogle" {
					t.Errorf("expected tool name 'searchGoogle', got %q", tc.ToolName)
				}
				if tc.Input != `{"query": "latest news on ai"}` {
					t.Errorf("expected input '{\"query\": \"latest news on ai\"}', got %q", tc.Input)
				}
			}
		}
	})
}

func TestChatDoStream_ReasoningModelStreaming(t *testing.T) {
	t.Run("should stream text delta for reasoning models", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":1,\"delta\":{\"content\":\"Hello, World!\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\",\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[],\"usage\":{\"prompt_tokens\":17,\"total_tokens\":244,\"completion_tokens\":227}}\n\n",
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

		var textContent string
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok {
				textContent += td.Delta
			}
		}
		if textContent != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", textContent)
		}

		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.FinishReason.Unified != "stop" {
					t.Errorf("expected finish reason 'stop', got %q", f.FinishReason.Unified)
				}
			}
		}
	})

	t.Run("should send reasoning tokens in streaming", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":1,\"delta\":{\"content\":\"Hello, World!\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":null,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\",\"logprobs\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"o4-mini\",\"system_fingerprint\":\"fp_3bc1b5746c\",\"choices\":[],\"usage\":{\"prompt_tokens\":15,\"completion_tokens\":20,\"total_tokens\":35,\"completion_tokens_details\":{\"reasoning_tokens\":10}}}\n\n",
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

		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.Usage.OutputTokens.Reasoning == nil || *f.Usage.OutputTokens.Reasoning != 10 {
					t.Errorf("expected reasoning tokens 10, got %v", f.Usage.OutputTokens.Reasoning)
				}
				if f.Usage.OutputTokens.Total == nil || *f.Usage.OutputTokens.Total != 20 {
					t.Errorf("expected total output tokens 20, got %v", f.Usage.OutputTokens.Total)
				}
				if f.Usage.OutputTokens.Text == nil || *f.Usage.OutputTokens.Text != 10 {
					t.Errorf("expected text tokens 10, got %v", f.Usage.OutputTokens.Text)
				}
			}
		}
	})
}

func TestChatDoStream_CachedTokens(t *testing.T) {
	t.Run("should return cached tokens in streaming providerMetadata", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":5,\"total_tokens\":25,\"prompt_tokens_details\":{\"cached_tokens\":10}}}\n\n",
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

		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.Usage.InputTokens.CacheRead == nil || *f.Usage.InputTokens.CacheRead != 10 {
					t.Errorf("expected cached tokens 10, got %v", f.Usage.InputTokens.CacheRead)
				}
				if f.Usage.InputTokens.Total == nil || *f.Usage.InputTokens.Total != 20 {
					t.Errorf("expected total input tokens 20, got %v", f.Usage.InputTokens.Total)
				}
			}
		}
	})
}

func TestChatDoStream_PredictionTokens(t *testing.T) {
	t.Run("should return prediction tokens in streaming providerMetadata", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-3.5-turbo\",\"choices\":[],\"usage\":{\"prompt_tokens\":20,\"completion_tokens\":15,\"total_tokens\":35,\"completion_tokens_details\":{\"accepted_prediction_tokens\":5,\"rejected_prediction_tokens\":3}}}\n\n",
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

		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.ProviderMetadata == nil {
					t.Fatal("expected non-nil providerMetadata")
				}
				metaMap, ok := f.ProviderMetadata["openai"]
				if !ok {
					t.Fatal("expected openai key in providerMetadata")
				}
				// Check for accepted/rejected prediction tokens
				if v, exists := metaMap["acceptedPredictionTokens"]; exists {
					if fmt.Sprintf("%v", v) != "5" {
						t.Errorf("expected acceptedPredictionTokens 5, got %v", v)
					}
				}
				if v, exists := metaMap["rejectedPredictionTokens"]; exists {
					if fmt.Sprintf("%v", v) != "3" {
						t.Errorf("expected rejectedPredictionTokens 3, got %v", v)
					}
				}
			}
		}
	})
}

func TestChatDoGenerate_UnknownFinishReason(t *testing.T) {
	t.Run("should support unknown finish reason", func(t *testing.T) {
		fixture := chatTextFixture()
		choices := fixture["choices"].([]any)
		choice := choices[0].(map[string]any)
		choice["finish_reason"] = "custom_reason"

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

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
		raw := "custom_reason"
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != raw {
			t.Errorf("expected raw finish reason 'custom_reason', got %v", result.FinishReason.Raw)
		}
	})
}

func TestChatDoGenerate_PartialUsage(t *testing.T) {
	t.Run("should support partial usage", func(t *testing.T) {
		fixture := map[string]any{
			"id":      "chatcmpl-123",
			"object":  "chat.completion",
			"created": float64(1702657020),
			"model":   "gpt-3.5-turbo-0613",
			"choices": []any{
				map[string]any{
					"index":         float64(0),
					"message":       map[string]any{"role": "assistant", "content": "Hello, World!"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     float64(20),
				"total_tokens":      float64(25),
				"completion_tokens": float64(5),
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createChatModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 20 {
			t.Errorf("expected input tokens 20, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 5 {
			t.Errorf("expected output tokens 5, got %v", result.Usage.OutputTokens.Total)
		}
	})
}

func TestChatDoGenerate_TextVerbosity(t *testing.T) {
	t.Run("should pass textVerbosity setting from provider options", func(t *testing.T) {
		server, capture := createJSONTestServer(chatTextFixture(), nil)
		defer server.Close()
		model := createChatModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"textVerbosity": "concise",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["verbosity"] != "concise" {
			t.Errorf("expected verbosity 'concise', got %v", body["verbosity"])
		}
	})
}

func TestChatDoStream_ModelRouterModelId(t *testing.T) {
	t.Run("should set modelId for model-router request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-4o-2024-08-06\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-4o-2024-08-06\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"chatcmpl-123\",\"object\":\"chat.completion.chunk\",\"created\":1702657020,\"model\":\"gpt-4o-2024-08-06\",\"choices\":[],\"usage\":{\"prompt_tokens\":17,\"total_tokens\":244,\"completion_tokens\":227}}\n\n",
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

		for _, p := range parts {
			if meta, ok := p.(languagemodel.StreamPartResponseMetadata); ok {
				if meta.ModelID == nil || *meta.ModelID != "gpt-4o-2024-08-06" {
					t.Errorf("expected modelId 'gpt-4o-2024-08-06', got %v", meta.ModelID)
				}
			}
		}
	})
}
