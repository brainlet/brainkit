// Ported from: packages/huggingface/src/responses/huggingface-responses-language-model.test.ts
package huggingface

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Test helpers ---

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
		id := fmt.Sprintf("id-%d", counter)
		counter++
		return id
	}
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
func createSSEServer(chunks []string, headers map[string]string) (*httptest.Server, *requestCapture) {
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

// createTestModel creates a ResponsesLanguageModel pointing at a test server.
func createTestModel(modelID string, baseURL string) *ResponsesLanguageModel {
	return NewResponsesLanguageModel(modelID, Config{
		Provider: "huggingface.responses",
		URL: func(opts URLOptions) string {
			return fmt.Sprintf("%s%s", baseURL, opts.Path)
		},
		Headers: func() map[string]string {
			return map[string]string{"Authorization": "Bearer APIKEY"}
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

// basicTextResponseFixture returns a standard text response fixture.
func basicTextResponseFixture() map[string]any {
	return map[string]any{
		"id":                 "resp_67c97c0203188190a025beb4a75242bc",
		"model":              "deepseek-ai/DeepSeek-V3-0324",
		"object":             "response",
		"created_at":         float64(1741257730),
		"status":             "completed",
		"error":              nil,
		"instructions":       nil,
		"max_output_tokens":  nil,
		"metadata":           nil,
		"tool_choice":        "auto",
		"tools":              []any{},
		"temperature":        1.0,
		"top_p":              1.0,
		"incomplete_details": nil,
		"usage": map[string]any{
			"input_tokens":  float64(12),
			"output_tokens": float64(25),
			"total_tokens":  float64(37),
		},
		"output": []any{
			map[string]any{
				"id":     "msg_67c97c02656c81908e080dfdf4a03cd1",
				"type":   "message",
				"role":   "assistant",
				"status": "completed",
				"content": []any{
					map[string]any{
						"type": "output_text",
						"text": "Hello! How can I help you today?",
					},
				},
			},
		},
		"output_text": "Hello! How can I help you today?",
	}
}

// emptyResponseFixture returns a minimal response for message conversion tests.
func emptyResponseFixture() map[string]any {
	return map[string]any{
		"id":                 "resp_test",
		"model":              "moonshotai/Kimi-K2-Instruct",
		"object":             "response",
		"created_at":         float64(1741257730),
		"status":             "completed",
		"error":              nil,
		"instructions":       nil,
		"max_output_tokens":  nil,
		"metadata":           nil,
		"tool_choice":        "auto",
		"tools":              []any{},
		"temperature":        1.0,
		"top_p":              1.0,
		"incomplete_details": nil,
		"usage":              nil,
		"output":             []any{},
		"output_text":        "Test response",
	}
}

// ===== DoGenerate tests =====

func TestDoGenerate_BasicTextResponse(t *testing.T) {
	t.Run("should generate text", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

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
		if textContent.Text != "Hello! How can I help you today?" {
			t.Errorf("expected text 'Hello! How can I help you today?', got %q", textContent.Text)
		}

		// Check provider metadata
		meta := textContent.ProviderMetadata["huggingface"]
		if meta["itemId"] != "msg_67c97c02656c81908e080dfdf4a03cd1" {
			t.Errorf("expected itemId 'msg_67c97c02656c81908e080dfdf4a03cd1', got %v", meta["itemId"])
		}
	})

	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 12 {
			t.Errorf("expected input total tokens 12, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.InputTokens.NoCache == nil || *result.Usage.InputTokens.NoCache != 12 {
			t.Errorf("expected input noCache tokens 12, got %v", result.Usage.InputTokens.NoCache)
		}
		if result.Usage.InputTokens.CacheRead == nil || *result.Usage.InputTokens.CacheRead != 0 {
			t.Errorf("expected input cacheRead tokens 0, got %v", result.Usage.InputTokens.CacheRead)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 25 {
			t.Errorf("expected output total tokens 25, got %v", result.Usage.OutputTokens.Total)
		}
		if result.Usage.OutputTokens.Text == nil || *result.Usage.OutputTokens.Text != 25 {
			t.Errorf("expected output text tokens 25, got %v", result.Usage.OutputTokens.Text)
		}
		if result.Usage.OutputTokens.Reasoning == nil || *result.Usage.OutputTokens.Reasoning != 0 {
			t.Errorf("expected output reasoning tokens 0, got %v", result.Usage.OutputTokens.Reasoning)
		}
	})

	t.Run("should extract text from output array when output_text is missing", func(t *testing.T) {
		fixture := basicTextResponseFixture()
		fixture["id"] = "resp_test"
		fixture["output_text"] = nil
		fixture["usage"] = nil
		output := fixture["output"].([]any)
		msg := output[0].(map[string]any)
		msg["id"] = "msg_test"
		content := msg["content"].([]any)
		textContent := content[0].(map[string]any)
		textContent["text"] = "Extracted from output array"

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

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
		textPart, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if textPart.Text != "Extracted from output array" {
			t.Errorf("expected text 'Extracted from output array', got %q", textPart.Text)
		}
	})

	t.Run("should handle missing usage gracefully", func(t *testing.T) {
		fixture := basicTextResponseFixture()
		fixture["id"] = "resp_test"
		fixture["usage"] = nil
		fixture["output"] = []any{}
		fixture["output_text"] = "Test response"

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// When usage is nil, all token counts should be nil.
		if result.Usage.InputTokens.Total != nil {
			t.Errorf("expected input total nil, got %v", *result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total != nil {
			t.Errorf("expected output total nil, got %v", *result.Usage.OutputTokens.Total)
		}
		if result.Usage.Raw != nil {
			t.Errorf("expected raw nil, got %v", result.Usage.Raw)
		}
	})

	t.Run("should send model id, settings, and input", func(t *testing.T) {
		server, capture := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		temp := 0.5
		topP := 0.3
		maxTokens := 100
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "You are a helpful assistant."},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
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
		if body["model"] != "deepseek-ai/DeepSeek-V3-0324" {
			t.Errorf("expected model 'deepseek-ai/DeepSeek-V3-0324', got %v", body["model"])
		}
		if body["temperature"] != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", body["temperature"])
		}
		if body["top_p"] != 0.3 {
			t.Errorf("expected top_p 0.3, got %v", body["top_p"])
		}
		if body["max_output_tokens"] != float64(100) {
			t.Errorf("expected max_output_tokens 100, got %v", body["max_output_tokens"])
		}
		if body["stream"] != false {
			t.Errorf("expected stream false, got %v", body["stream"])
		}

		input, ok := body["input"].([]any)
		if !ok {
			t.Fatalf("expected input to be []any, got %T", body["input"])
		}
		if len(input) != 2 {
			t.Fatalf("expected 2 input messages, got %d", len(input))
		}

		// First message: system
		sysMsg := input[0].(map[string]any)
		if sysMsg["role"] != "system" {
			t.Errorf("expected role 'system', got %v", sysMsg["role"])
		}
		if sysMsg["content"] != "You are a helpful assistant." {
			t.Errorf("expected content 'You are a helpful assistant.', got %v", sysMsg["content"])
		}

		// Second message: user
		userMsg := input[1].(map[string]any)
		if userMsg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", userMsg["role"])
		}
		userContent := userMsg["content"].([]any)
		part := userContent[0].(map[string]any)
		if part["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", part["type"])
		}
		if part["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", part["text"])
		}
	})

	t.Run("should handle unsupported settings with warnings", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		topK := 10
		seed := 123
		presencePenalty := 0.5
		frequencyPenalty := 0.3
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:           testPrompt,
			TopK:             &topK,
			Seed:             &seed,
			PresencePenalty:  &presencePenalty,
			FrequencyPenalty: &frequencyPenalty,
			StopSequences:    []string{"stop"},
			Ctx:              context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedWarnings := []string{"topK", "seed", "presencePenalty", "frequencyPenalty", "stopSequences"}
		if len(result.Warnings) < len(expectedWarnings) {
			t.Fatalf("expected at least %d warnings, got %d", len(expectedWarnings), len(result.Warnings))
		}

		for i, expected := range expectedWarnings {
			w, ok := result.Warnings[i].(shared.UnsupportedWarning)
			if !ok {
				t.Errorf("warning %d: expected UnsupportedWarning, got %T", i, result.Warnings[i])
				continue
			}
			if w.Feature != expected {
				t.Errorf("warning %d: expected feature %q, got %q", i, expected, w.Feature)
			}
		}
	})
}

func TestDoGenerate_Annotations(t *testing.T) {
	t.Run("should generate text and sources from annotations", func(t *testing.T) {
		fixture := map[string]any{
			"id":                 "resp_test_annotations",
			"model":              "deepseek-ai/DeepSeek-V3-0324",
			"object":             "response",
			"created_at":         float64(1741257730),
			"status":             "completed",
			"error":              nil,
			"instructions":       nil,
			"max_output_tokens":  nil,
			"metadata":           nil,
			"tool_choice":        "auto",
			"tools":              []any{},
			"temperature":        1.0,
			"top_p":              1.0,
			"incomplete_details": nil,
			"usage": map[string]any{
				"input_tokens":  float64(20),
				"output_tokens": float64(50),
				"total_tokens":  float64(70),
			},
			"output": []any{
				map[string]any{
					"id":     "msg_test_annotations",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "Here are some recent articles about AI.",
							"annotations": []any{
								map[string]any{
									"type":  "url_citation",
									"url":   "https://example.com/article1",
									"title": "AI Developments Article",
								},
								map[string]any{
									"type":  "url_citation",
									"url":   "https://test.com/article2",
									"title": "Industry Trends Report",
								},
							},
						},
					},
				},
			},
			"output_text": nil,
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 3 content items: 1 text + 2 sources
		if len(result.Content) != 3 {
			t.Fatalf("expected 3 content items, got %d", len(result.Content))
		}

		// First: text
		textPart, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text, got %T", result.Content[0])
		}
		if textPart.Text != "Here are some recent articles about AI." {
			t.Errorf("unexpected text: %q", textPart.Text)
		}

		// Second: source URL
		source1, ok := result.Content[1].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[1])
		}
		if source1.URL != "https://example.com/article1" {
			t.Errorf("expected URL 'https://example.com/article1', got %q", source1.URL)
		}
		if source1.Title == nil || *source1.Title != "AI Developments Article" {
			t.Errorf("expected title 'AI Developments Article', got %v", source1.Title)
		}
		if source1.ID != "id-0" {
			t.Errorf("expected ID 'id-0', got %q", source1.ID)
		}

		// Third: source URL
		source2, ok := result.Content[2].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[2])
		}
		if source2.URL != "https://test.com/article2" {
			t.Errorf("expected URL 'https://test.com/article2', got %q", source2.URL)
		}
		if source2.ID != "id-1" {
			t.Errorf("expected ID 'id-1', got %q", source2.ID)
		}
	})
}

func TestDoGenerate_MCPToolsWithAnnotations(t *testing.T) {
	t.Run("should handle MCP tools with annotations", func(t *testing.T) {
		fixture := map[string]any{
			"id":                 "resp_mcp_test",
			"model":              "deepseek-ai/DeepSeek-V3-0324",
			"object":             "response",
			"created_at":         float64(1741257730),
			"status":             "completed",
			"error":              nil,
			"instructions":       nil,
			"max_output_tokens":  nil,
			"metadata":           nil,
			"tool_choice":        "auto",
			"tools":              []any{},
			"temperature":        1.0,
			"top_p":              1.0,
			"incomplete_details": nil,
			"usage": map[string]any{
				"input_tokens":  float64(50),
				"output_tokens": float64(100),
				"total_tokens":  float64(150),
			},
			"output": []any{
				map[string]any{
					"id":           "mcp_search_test",
					"type":         "mcp_call",
					"server_label": "web_search",
					"name":         "search",
					"arguments":    `{"query": "San Francisco tech events"}`,
					"output":       "Found 25 tech events in San Francisco",
				},
				map[string]any{
					"id":     "msg_mcp_response",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "Based on the search results, here are the latest tech events.",
							"annotations": []any{
								map[string]any{
									"type":  "url_citation",
									"url":   "https://techevents.com/sf-ai",
									"title": "SF AI Conference 2025",
								},
								map[string]any{
									"type":  "url_citation",
									"url":   "https://eventbrite.com/sf-startups",
									"title": "SF Startup Meetups",
								},
							},
						},
					},
				},
			},
			"output_text": nil,
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expected: tool-call, tool-result, text, source, source = 5 items
		if len(result.Content) != 5 {
			t.Fatalf("expected 5 content items, got %d", len(result.Content))
		}

		// First: tool call
		toolCall, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if toolCall.ToolCallID != "mcp_search_test" {
			t.Errorf("expected toolCallId 'mcp_search_test', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", toolCall.ToolName)
		}
		if toolCall.ProviderExecuted == nil || *toolCall.ProviderExecuted != true {
			t.Error("expected providerExecuted to be true")
		}

		// Second: tool result
		toolResult, ok := result.Content[1].(languagemodel.ToolResult)
		if !ok {
			t.Fatalf("expected ToolResult, got %T", result.Content[1])
		}
		if toolResult.ToolCallID != "mcp_search_test" {
			t.Errorf("expected toolCallId 'mcp_search_test', got %q", toolResult.ToolCallID)
		}
		if toolResult.Result != "Found 25 tech events in San Francisco" {
			t.Errorf("unexpected result: %v", toolResult.Result)
		}

		// Third: text
		textPart, ok := result.Content[2].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text, got %T", result.Content[2])
		}
		if textPart.ProviderMetadata["huggingface"]["itemId"] != "msg_mcp_response" {
			t.Errorf("expected itemId 'msg_mcp_response', got %v", textPart.ProviderMetadata["huggingface"]["itemId"])
		}

		// Fourth & Fifth: sources
		source1, ok := result.Content[3].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[3])
		}
		if source1.URL != "https://techevents.com/sf-ai" {
			t.Errorf("expected URL 'https://techevents.com/sf-ai', got %q", source1.URL)
		}

		source2, ok := result.Content[4].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[4])
		}
		if source2.URL != "https://eventbrite.com/sf-startups" {
			t.Errorf("expected URL 'https://eventbrite.com/sf-startups', got %q", source2.URL)
		}
	})
}

func TestDoGenerate_FunctionCall(t *testing.T) {
	t.Run("should handle function_call tool responses", func(t *testing.T) {
		fixture := map[string]any{
			"id":                 "resp_tool_test",
			"model":              "deepseek-ai/DeepSeek-V3-0324",
			"object":             "response",
			"created_at":         float64(1741257730),
			"status":             "completed",
			"error":              nil,
			"instructions":       nil,
			"max_output_tokens":  nil,
			"metadata":           nil,
			"tool_choice":        "auto",
			"tools":              []any{},
			"temperature":        1.0,
			"top_p":              1.0,
			"incomplete_details": nil,
			"usage": map[string]any{
				"input_tokens":  float64(50),
				"output_tokens": float64(30),
				"total_tokens":  float64(80),
			},
			"output": []any{
				map[string]any{
					"id":        "fc_test",
					"type":      "function_call",
					"call_id":   "call_123",
					"name":      "getWeather",
					"arguments": `{"location": "New York"}`,
					"output":    `{"temperature": "72°F", "condition": "sunny"}`,
				},
				map[string]any{
					"id":     "msg_after_tool",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "The weather in New York is 72°F and sunny.",
						},
					},
				},
			},
			"output_text": nil,
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Expected: tool-call, tool-result, text = 3 items
		if len(result.Content) != 3 {
			t.Fatalf("expected 3 content items, got %d", len(result.Content))
		}

		toolCall, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if toolCall.ToolCallID != "call_123" {
			t.Errorf("expected toolCallId 'call_123', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "getWeather" {
			t.Errorf("expected toolName 'getWeather', got %q", toolCall.ToolName)
		}
		if toolCall.Input != `{"location": "New York"}` {
			t.Errorf("unexpected input: %v", toolCall.Input)
		}

		toolResult, ok := result.Content[1].(languagemodel.ToolResult)
		if !ok {
			t.Fatalf("expected ToolResult, got %T", result.Content[1])
		}
		if toolResult.ToolCallID != "call_123" {
			t.Errorf("expected toolCallId 'call_123', got %q", toolResult.ToolCallID)
		}

		textPart, ok := result.Content[2].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text, got %T", result.Content[2])
		}
		if textPart.Text != "The weather in New York is 72°F and sunny." {
			t.Errorf("unexpected text: %q", textPart.Text)
		}
	})
}

func TestDoGenerate_Reasoning(t *testing.T) {
	t.Run("should handle reasoning content in responses", func(t *testing.T) {
		fixture := map[string]any{
			"id":                 "resp_reasoning",
			"model":              "deepseek-ai/DeepSeek-R1",
			"object":             "response",
			"created_at":         float64(1741257730),
			"status":             "completed",
			"error":              nil,
			"instructions":       nil,
			"max_output_tokens":  nil,
			"metadata":           nil,
			"tool_choice":        "auto",
			"tools":              []any{},
			"temperature":        1.0,
			"top_p":              1.0,
			"incomplete_details": nil,
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(50),
				"total_tokens":  float64(60),
			},
			"output": []any{
				map[string]any{
					"id":   "reasoning_1",
					"type": "reasoning",
					"content": []any{
						map[string]any{
							"type": "reasoning_text",
							"text": "Let me think about this problem step by step...",
						},
					},
				},
				map[string]any{
					"id":     "msg_after_reasoning",
					"type":   "message",
					"role":   "assistant",
					"status": "completed",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "The answer is 42.",
						},
					},
				},
			},
			"output_text": nil,
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-R1", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 2 {
			t.Fatalf("expected 2 content items, got %d", len(result.Content))
		}

		// First: reasoning
		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning, got %T", result.Content[0])
		}
		if reasoning.Text != "Let me think about this problem step by step..." {
			t.Errorf("unexpected reasoning text: %q", reasoning.Text)
		}
		if reasoning.ProviderMetadata["huggingface"]["itemId"] != "reasoning_1" {
			t.Errorf("expected itemId 'reasoning_1', got %v", reasoning.ProviderMetadata["huggingface"]["itemId"])
		}

		// Second: text
		textPart, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text, got %T", result.Content[1])
		}
		if textPart.Text != "The answer is 42." {
			t.Errorf("unexpected text: %q", textPart.Text)
		}
	})
}

func TestDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should return response metadata", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

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
		if result.Response.ID == nil || *result.Response.ID != "resp_67c97c0203188190a025beb4a75242bc" {
			t.Errorf("expected ID 'resp_67c97c0203188190a025beb4a75242bc', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "deepseek-ai/DeepSeek-V3-0324" {
			t.Errorf("expected ModelID 'deepseek-ai/DeepSeek-V3-0324', got %v", result.Response.ModelID)
		}
		expectedTime := time.Unix(1741257730, 0)
		if result.Response.Timestamp == nil || !result.Response.Timestamp.Equal(expectedTime) {
			t.Errorf("expected timestamp %v, got %v", expectedTime, result.Response.Timestamp)
		}
	})
}

func TestDoGenerate_FinishReason(t *testing.T) {
	t.Run("should default to stop when no incomplete_details", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", result.FinishReason.Unified)
		}
	})
}

func TestDoGenerate_ProviderMetadata(t *testing.T) {
	t.Run("should include responseId in provider metadata", func(t *testing.T) {
		server, _ := createJSONTestServer(basicTextResponseFixture(), nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hfMeta := result.ProviderMetadata["huggingface"]
		if hfMeta["responseId"] != "resp_67c97c0203188190a025beb4a75242bc" {
			t.Errorf("expected responseId 'resp_67c97c0203188190a025beb4a75242bc', got %v", hfMeta["responseId"])
		}
	})
}

func TestDoGenerate_StructuredOutput(t *testing.T) {
	t.Run("should send text.format for structured output", func(t *testing.T) {
		fixture := emptyResponseFixture()
		fixture["id"] = "resp_structured"
		fixture["model"] = "moonshotai/Kimi-K2-Instruct"
		fixture["output"] = []any{
			map[string]any{
				"id":     "msg_structured",
				"type":   "message",
				"role":   "assistant",
				"status": "completed",
				"content": []any{
					map[string]any{
						"type": "output_text",
						"text": `{"name": "John Doe", "age": 30}`,
					},
				},
			},
		}

		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("moonshotai/Kimi-K2-Instruct", server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
						"age":  map[string]any{"type": "number"},
					},
					"required": []any{"name", "age"},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		textObj, ok := body["text"].(map[string]any)
		if !ok {
			t.Fatalf("expected text object, got %T", body["text"])
		}
		formatObj, ok := textObj["format"].(map[string]any)
		if !ok {
			t.Fatalf("expected format object, got %T", textObj["format"])
		}
		if formatObj["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", formatObj["type"])
		}
		if formatObj["name"] != "response" {
			t.Errorf("expected name 'response', got %v", formatObj["name"])
		}
		if formatObj["strict"] != false {
			t.Errorf("expected strict false, got %v", formatObj["strict"])
		}
	})

	t.Run("should handle structured output with custom name and description", func(t *testing.T) {
		fixture := emptyResponseFixture()
		fixture["id"] = "resp_structured"
		fixture["model"] = "moonshotai/Kimi-K2-Instruct"

		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("moonshotai/Kimi-K2-Instruct", server.URL)

		name := "person_profile"
		desc := "A person profile with basic information"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
				},
				Name:        &name,
				Description: &desc,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		textObj := body["text"].(map[string]any)
		formatObj := textObj["format"].(map[string]any)
		if formatObj["name"] != "person_profile" {
			t.Errorf("expected name 'person_profile', got %v", formatObj["name"])
		}
		if formatObj["description"] != "A person profile with basic information" {
			t.Errorf("expected description 'A person profile with basic information', got %v", formatObj["description"])
		}
	})
}

func TestDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should send provider-specific options", func(t *testing.T) {
		fixture := emptyResponseFixture()

		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"huggingface": {
					"metadata":         map[string]any{"key": "value"},
					"instructions":     "Be concise",
					"strictJsonSchema":  true,
				},
			},
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{"type": "object"},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		metadata, ok := body["metadata"].(map[string]any)
		if !ok {
			t.Fatalf("expected metadata to be map, got %T", body["metadata"])
		}
		if metadata["key"] != "value" {
			t.Errorf("expected metadata key 'value', got %v", metadata["key"])
		}
		if body["instructions"] != "Be concise" {
			t.Errorf("expected instructions 'Be concise', got %v", body["instructions"])
		}

		// Check that strictJsonSchema affects the text.format.strict field
		textObj := body["text"].(map[string]any)
		formatObj := textObj["format"].(map[string]any)
		if formatObj["strict"] != true {
			t.Errorf("expected strict true, got %v", formatObj["strict"])
		}
	})
}

func TestDoGenerate_ToolPreparation(t *testing.T) {
	t.Run("should prepare tools correctly", func(t *testing.T) {
		fixture := emptyResponseFixture()
		fixture["id"] = "resp_tools"
		fixture["model"] = "deepseek-ai/DeepSeek-V3-0324"

		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		desc := "Get weather information"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "getWeather",
					Description: &desc,
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
						},
						"required": []any{"location"},
					},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{
				ToolName: "getWeather",
			},
			Ctx: context.Background(),
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

		tool := tools[0].(map[string]any)
		if tool["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tool["type"])
		}
		if tool["name"] != "getWeather" {
			t.Errorf("expected name 'getWeather', got %v", tool["name"])
		}
		if tool["description"] != "Get weather information" {
			t.Errorf("expected description 'Get weather information', got %v", tool["description"])
		}

		params := tool["parameters"].(map[string]any)
		if params["type"] != "object" {
			t.Errorf("expected parameters type 'object', got %v", params["type"])
		}

		// Check tool_choice
		toolChoice, ok := body["tool_choice"].(map[string]any)
		if !ok {
			t.Fatalf("expected tool_choice to be map, got %T", body["tool_choice"])
		}
		if toolChoice["type"] != "function" {
			t.Errorf("expected tool_choice type 'function', got %v", toolChoice["type"])
		}
		fn := toolChoice["function"].(map[string]any)
		if fn["name"] != "getWeather" {
			t.Errorf("expected function name 'getWeather', got %v", fn["name"])
		}
	})

	t.Run("should handle auto and required tool choices", func(t *testing.T) {
		fixture := emptyResponseFixture()
		fixture["id"] = "resp_tools"
		fixture["model"] = "deepseek-ai/DeepSeek-V3-0324"

		// Test auto
		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "test",
					InputSchema: map[string]any{"type": "object"},
				},
			},
			ToolChoice: languagemodel.ToolChoiceAuto{},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["tool_choice"] != "auto" {
			t.Errorf("expected tool_choice 'auto', got %v", body["tool_choice"])
		}

		// Test required
		server2, capture2 := createJSONTestServer(fixture, nil)
		defer server2.Close()
		model2 := createTestModel("deepseek-ai/DeepSeek-V3-0324", server2.URL)

		_, err = model2.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "test",
					InputSchema: map[string]any{"type": "object"},
				},
			},
			ToolChoice: languagemodel.ToolChoiceRequired{},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body2 := capture2.BodyJSON()
		if body2["tool_choice"] != "required" {
			t.Errorf("expected tool_choice 'required', got %v", body2["tool_choice"])
		}
	})
}

// ===== Message conversion tests =====

func TestDoGenerate_MessageConversion(t *testing.T) {
	t.Run("should convert user messages with images", func(t *testing.T) {
		fixture := emptyResponseFixture()
		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "What do you see?"},
						languagemodel.FilePart{
							MediaType: "image/jpeg",
							Data:      languagemodel.DataContentString{Value: "AQIDBA=="},
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
		input := body["input"].([]any)
		msg := input[0].(map[string]any)
		content := msg["content"].([]any)

		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}

		textPart := content[0].(map[string]any)
		if textPart["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", textPart["type"])
		}
		if textPart["text"] != "What do you see?" {
			t.Errorf("expected text 'What do you see?', got %v", textPart["text"])
		}

		imagePart := content[1].(map[string]any)
		if imagePart["type"] != "input_image" {
			t.Errorf("expected type 'input_image', got %v", imagePart["type"])
		}
		if imagePart["image_url"] != "data:image/jpeg;base64,AQIDBA==" {
			t.Errorf("expected image_url 'data:image/jpeg;base64,AQIDBA==', got %v", imagePart["image_url"])
		}
	})

	t.Run("should handle assistant messages", func(t *testing.T) {
		fixture := emptyResponseFixture()
		server, capture := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "Hi there!"},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "How are you?"},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		input := body["input"].([]any)
		if len(input) != 3 {
			t.Fatalf("expected 3 input messages, got %d", len(input))
		}

		// Check user message
		userMsg := input[0].(map[string]any)
		if userMsg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", userMsg["role"])
		}

		// Check assistant message
		assistantMsg := input[1].(map[string]any)
		if assistantMsg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", assistantMsg["role"])
		}
		assistantContent := assistantMsg["content"].([]any)
		assistantPart := assistantContent[0].(map[string]any)
		if assistantPart["type"] != "output_text" {
			t.Errorf("expected type 'output_text', got %v", assistantPart["type"])
		}
		if assistantPart["text"] != "Hi there!" {
			t.Errorf("expected text 'Hi there!', got %v", assistantPart["text"])
		}

		// Check second user message
		userMsg2 := input[2].(map[string]any)
		if userMsg2["role"] != "user" {
			t.Errorf("expected role 'user', got %v", userMsg2["role"])
		}
	})

	t.Run("should warn about tool messages", func(t *testing.T) {
		fixture := emptyResponseFixture()
		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "test",
							ToolName:   "test",
							Output:     languagemodel.ToolResultOutputText{Value: "test"},
						},
					},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "tool messages" {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected warning about unsupported tool messages")
		}
	})
}

// ===== DoStream tests =====

func TestDoStream_TextDeltas(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.created","response":{"id":"resp_test","object":"response","created_at":1741269019,"status":"in_progress","model":"deepseek-ai/DeepSeek-V3-0324"}}` + "\n\n",
			`data:{"type":"response.in_progress","response":{"id":"resp_test","object":"response","created_at":1741269019,"status":"in_progress"}}` + "\n\n",
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"msg_test","type":"message","role":"assistant","status":"in_progress","content":[]},"sequence_number":1}` + "\n\n",
			`data:{"type":"response.output_text.delta","item_id":"msg_test","output_index":0,"content_index":0,"delta":"Hello,","sequence_number":2}` + "\n\n",
			`data:{"type":"response.output_text.delta","item_id":"msg_test","output_index":0,"content_index":0,"delta":" World!","sequence_number":3}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":0,"item":{"id":"msg_test","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello, World!"}]},"sequence_number":4}` + "\n\n",
			`data:{"type":"response.completed","response":{"id":"resp_test","model":"deepseek-ai/DeepSeek-V3-0324","object":"response","created_at":1741269112,"status":"completed","incomplete_details":null,"usage":{"input_tokens":12,"output_tokens":25,"total_tokens":37},"output":[{"id":"msg_test","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello, World!"}]}]}}` + "\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Check stream-start
		if _, ok := parts[0].(languagemodel.StreamPartStreamStart); !ok {
			t.Errorf("expected StreamPartStreamStart first, got %T", parts[0])
		}

		// Check response-metadata
		var metadataPart *languagemodel.StreamPartResponseMetadata
		for _, part := range parts {
			if mp, ok := part.(languagemodel.StreamPartResponseMetadata); ok {
				metadataPart = &mp
				break
			}
		}
		if metadataPart == nil {
			t.Fatal("expected response metadata part")
		}
		if metadataPart.ID == nil || *metadataPart.ID != "resp_test" {
			t.Errorf("expected ID 'resp_test', got %v", metadataPart.ID)
		}
		if metadataPart.ModelID == nil || *metadataPart.ModelID != "deepseek-ai/DeepSeek-V3-0324" {
			t.Errorf("expected ModelID 'deepseek-ai/DeepSeek-V3-0324', got %v", metadataPart.ModelID)
		}

		// Check text-start
		var textStart *languagemodel.StreamPartTextStart
		for _, part := range parts {
			if ts, ok := part.(languagemodel.StreamPartTextStart); ok {
				textStart = &ts
				break
			}
		}
		if textStart == nil {
			t.Fatal("expected text-start part")
		}
		if textStart.ID != "msg_test" {
			t.Errorf("expected ID 'msg_test', got %q", textStart.ID)
		}

		// Check text deltas
		var fullText string
		for _, part := range parts {
			if delta, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += delta.Delta
			}
		}
		if fullText != "Hello, World!" {
			t.Errorf("expected full text 'Hello, World!', got %q", fullText)
		}

		// Check text-end
		var textEnd *languagemodel.StreamPartTextEnd
		for _, part := range parts {
			if te, ok := part.(languagemodel.StreamPartTextEnd); ok {
				textEnd = &te
				break
			}
		}
		if textEnd == nil {
			t.Fatal("expected text-end part")
		}
		if textEnd.ID != "msg_test" {
			t.Errorf("expected ID 'msg_test', got %q", textEnd.ID)
		}

		// Check finish
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
				break
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", finishPart.FinishReason.Unified)
		}

		// Check finish usage
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 12 {
			t.Errorf("expected input total tokens 12, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 25 {
			t.Errorf("expected output total tokens 25, got %v", finishPart.Usage.OutputTokens.Total)
		}

		// Check provider metadata on finish
		hfMeta := finishPart.ProviderMetadata["huggingface"]
		responseID, ok := hfMeta["responseId"].(*string)
		if !ok || responseID == nil || *responseID != "resp_test" {
			t.Errorf("expected responseId 'resp_test', got %v", hfMeta["responseId"])
		}
	})
}

func TestDoStream_WithoutUsage(t *testing.T) {
	t.Run("should handle streaming without usage", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"msg_test","type":"message","role":"assistant","status":"in_progress"},"sequence_number":1}` + "\n\n",
			`data:{"type":"response.output_text.delta","item_id":"msg_test","output_index":0,"content_index":0,"delta":"Hi!","sequence_number":2}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":0,"item":{"id":"msg_test","type":"message","role":"assistant","status":"completed"},"sequence_number":3}` + "\n\n",
			`data:{"type":"response.completed","response":{"id":"resp_test","status":"completed","incomplete_details":null,"usage":null},"sequence_number":4}` + "\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

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

		// When usage is nil, token counts should be nil.
		if finishPart.Usage.InputTokens.Total != nil {
			t.Errorf("expected input total nil, got %v", *finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total != nil {
			t.Errorf("expected output total nil, got %v", *finishPart.Usage.OutputTokens.Total)
		}
		if finishPart.Usage.Raw != nil {
			t.Errorf("expected raw nil, got %v", finishPart.Usage.Raw)
		}
	})
}

func TestDoStream_NonMessageItems(t *testing.T) {
	t.Run("should handle non-message item types", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"mcp_test","type":"mcp_list_tools","server_label":"test"},"sequence_number":1}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":0,"item":{"id":"mcp_test","type":"mcp_list_tools","server_label":"test"},"sequence_number":2}` + "\n\n",
			`data:{"type":"response.completed","response":{"id":"resp_test","status":"completed","incomplete_details":null},"sequence_number":3}` + "\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Should only have stream-start and finish events (no text events)
		var types []string
		for _, part := range parts {
			switch part.(type) {
			case languagemodel.StreamPartStreamStart:
				types = append(types, "stream-start")
			case languagemodel.StreamPartFinish:
				types = append(types, "finish")
			case languagemodel.StreamPartTextStart:
				types = append(types, "text-start")
			case languagemodel.StreamPartTextDelta:
				types = append(types, "text-delta")
			case languagemodel.StreamPartTextEnd:
				types = append(types, "text-end")
			}
		}

		if len(types) != 2 || types[0] != "stream-start" || types[1] != "finish" {
			t.Errorf("expected [stream-start, finish], got %v", types)
		}
	})
}

func TestDoStream_StreamingErrors(t *testing.T) {
	t.Run("should handle streaming errors", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"msg_test","type":"message","role":"assistant"},"sequence_number":1}` + "\n\n",
			"data:invalid json}\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

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

		// Finish should have error finish reason
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part in stream")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonError {
			t.Errorf("expected unified finish reason 'error', got %q", finishPart.FinishReason.Unified)
		}
	})
}

func TestDoStream_RequestBody(t *testing.T) {
	t.Run("should send correct streaming request", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.completed","response":{"id":"resp_test","status":"completed"},"sequence_number":1}` + "\n\n",
		}

		server, capture := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		temp := 0.7
		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain the stream
		collectStreamParts(streamResult.Stream)

		body := capture.BodyJSON()
		if body["model"] != "deepseek-ai/DeepSeek-V3-0324" {
			t.Errorf("expected model 'deepseek-ai/DeepSeek-V3-0324', got %v", body["model"])
		}
		if body["temperature"] != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", body["temperature"])
		}
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}

		input := body["input"].([]any)
		if len(input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(input))
		}
		msg := input[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
	})
}

func TestDoStream_ToolCalls(t *testing.T) {
	t.Run("should stream tool calls", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.created","response":{"id":"resp_tool_stream","object":"response","created_at":1741269019,"status":"in_progress","model":"deepseek-ai/DeepSeek-V3-0324"}}` + "\n\n",
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"fc_stream","type":"function_call","call_id":"call_456","name":"calculator","arguments":""},"sequence_number":1}` + "\n\n",
			`data:{"type":"response.function_call_arguments.delta","item_id":"fc_stream","output_index":0,"delta":"{\"operation\"","sequence_number":2}` + "\n\n",
			`data:{"type":"response.function_call_arguments.delta","item_id":"fc_stream","output_index":0,"delta":": \"add\", \"a\": 5, \"b\": 3}","sequence_number":3}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":0,"item":{"id":"fc_stream","type":"function_call","call_id":"call_456","name":"calculator","arguments":"{\"operation\": \"add\", \"a\": 5, \"b\": 3}","output":"8"},"sequence_number":4}` + "\n\n",
			`data:{"type":"response.completed","response":{"id":"resp_tool_stream","status":"completed","usage":{"input_tokens":20,"output_tokens":15,"total_tokens":35}},"sequence_number":5}` + "\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-V3-0324", server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Check for tool-input-start
		var toolInputStart *languagemodel.StreamPartToolInputStart
		for _, part := range parts {
			if tis, ok := part.(languagemodel.StreamPartToolInputStart); ok {
				toolInputStart = &tis
			}
		}
		if toolInputStart == nil {
			t.Fatal("expected tool-input-start part")
		}
		if toolInputStart.ID != "call_456" {
			t.Errorf("expected ID 'call_456', got %q", toolInputStart.ID)
		}
		if toolInputStart.ToolName != "calculator" {
			t.Errorf("expected toolName 'calculator', got %q", toolInputStart.ToolName)
		}

		// Check for tool-input-end
		var toolInputEnd *languagemodel.StreamPartToolInputEnd
		for _, part := range parts {
			if tie, ok := part.(languagemodel.StreamPartToolInputEnd); ok {
				toolInputEnd = &tie
			}
		}
		if toolInputEnd == nil {
			t.Fatal("expected tool-input-end part")
		}

		// Check for tool call
		var toolCall *languagemodel.ToolCall
		for _, part := range parts {
			if tc, ok := part.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}
		if toolCall == nil {
			t.Fatal("expected tool-call part")
		}
		if toolCall.ToolCallID != "call_456" {
			t.Errorf("expected toolCallId 'call_456', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "calculator" {
			t.Errorf("expected toolName 'calculator', got %q", toolCall.ToolName)
		}

		// Check for tool result
		var toolResult *languagemodel.ToolResult
		for _, part := range parts {
			if tr, ok := part.(languagemodel.ToolResult); ok {
				toolResult = &tr
			}
		}
		if toolResult == nil {
			t.Fatal("expected tool-result part")
		}
		if toolResult.Result != "8" {
			t.Errorf("expected result '8', got %v", toolResult.Result)
		}

		// Check finish usage
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 20 {
			t.Errorf("expected input total tokens 20, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 15 {
			t.Errorf("expected output total tokens 15, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}

func TestDoStream_Reasoning(t *testing.T) {
	t.Run("should stream reasoning content", func(t *testing.T) {
		chunks := []string{
			`data:{"type":"response.created","response":{"id":"resp_reasoning_stream","object":"response","created_at":1741269019,"status":"in_progress","model":"deepseek-ai/DeepSeek-R1"}}` + "\n\n",
			`data:{"type":"response.output_item.added","output_index":0,"item":{"id":"reasoning_stream","type":"reasoning"},"sequence_number":1}` + "\n\n",
			`data:{"type":"response.reasoning_text.delta","item_id":"reasoning_stream","output_index":0,"content_index":0,"delta":"Thinking about","sequence_number":2}` + "\n\n",
			`data:{"type":"response.reasoning_text.delta","item_id":"reasoning_stream","output_index":0,"content_index":0,"delta":" the problem...","sequence_number":3}` + "\n\n",
			`data:{"type":"response.reasoning_text.done","item_id":"reasoning_stream","output_index":0,"content_index":0,"text":"Thinking about the problem...","sequence_number":4}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":0,"item":{"id":"reasoning_stream","type":"reasoning","content":[{"type":"reasoning_text","text":"Thinking about the problem..."}]},"sequence_number":5}` + "\n\n",
			`data:{"type":"response.output_item.added","output_index":1,"item":{"id":"msg_stream","type":"message","role":"assistant"},"sequence_number":6}` + "\n\n",
			`data:{"type":"response.output_text.delta","item_id":"msg_stream","output_index":1,"content_index":0,"delta":"The solution is","sequence_number":7}` + "\n\n",
			`data:{"type":"response.output_text.delta","item_id":"msg_stream","output_index":1,"content_index":0,"delta":" simple.","sequence_number":8}` + "\n\n",
			`data:{"type":"response.output_item.done","output_index":1,"item":{"id":"msg_stream","type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":"The solution is simple."}]},"sequence_number":9}` + "\n\n",
			`data:{"type":"response.completed","response":{"id":"resp_reasoning_stream","status":"completed","usage":{"input_tokens":10,"output_tokens":20,"total_tokens":30}},"sequence_number":10}` + "\n\n",
		}

		server, _ := createSSEServer(chunks, nil)
		defer server.Close()
		model := createTestModel("deepseek-ai/DeepSeek-R1", server.URL)

		streamResult, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(streamResult.Stream)

		// Collect part types in order
		var types []string
		for _, part := range parts {
			switch part.(type) {
			case languagemodel.StreamPartStreamStart:
				types = append(types, "stream-start")
			case languagemodel.StreamPartResponseMetadata:
				types = append(types, "response-metadata")
			case languagemodel.StreamPartReasoningStart:
				types = append(types, "reasoning-start")
			case languagemodel.StreamPartReasoningDelta:
				types = append(types, "reasoning-delta")
			case languagemodel.StreamPartReasoningEnd:
				types = append(types, "reasoning-end")
			case languagemodel.StreamPartTextStart:
				types = append(types, "text-start")
			case languagemodel.StreamPartTextDelta:
				types = append(types, "text-delta")
			case languagemodel.StreamPartTextEnd:
				types = append(types, "text-end")
			case languagemodel.StreamPartFinish:
				types = append(types, "finish")
			}
		}

		expectedTypes := []string{
			"stream-start",
			"response-metadata",
			"reasoning-start",
			"reasoning-delta",
			"reasoning-delta",
			"reasoning-end",
			"text-start",
			"text-delta",
			"text-delta",
			"text-end",
			"finish",
		}

		if len(types) != len(expectedTypes) {
			t.Fatalf("expected %d parts, got %d: %v", len(expectedTypes), len(types), types)
		}

		for i, expected := range expectedTypes {
			if types[i] != expected {
				t.Errorf("part %d: expected %q, got %q", i, expected, types[i])
			}
		}

		// Check reasoning deltas
		var reasoningText string
		for _, part := range parts {
			if rd, ok := part.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningText += rd.Delta
			}
		}
		if reasoningText != "Thinking about the problem..." {
			t.Errorf("expected reasoning text 'Thinking about the problem...', got %q", reasoningText)
		}

		// Check text deltas
		var fullText string
		for _, part := range parts {
			if td, ok := part.(languagemodel.StreamPartTextDelta); ok {
				fullText += td.Delta
			}
		}
		if fullText != "The solution is simple." {
			t.Errorf("expected text 'The solution is simple.', got %q", fullText)
		}

		// Check finish
		var finishPart *languagemodel.StreamPartFinish
		for _, part := range parts {
			if fp, ok := part.(languagemodel.StreamPartFinish); ok {
				finishPart = &fp
			}
		}
		if finishPart == nil {
			t.Fatal("expected finish part")
		}
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected unified finish reason 'stop', got %q", finishPart.FinishReason.Unified)
		}
		if finishPart.Usage.InputTokens.Total == nil || *finishPart.Usage.InputTokens.Total != 10 {
			t.Errorf("expected input total 10, got %v", finishPart.Usage.InputTokens.Total)
		}
		if finishPart.Usage.OutputTokens.Total == nil || *finishPart.Usage.OutputTokens.Total != 20 {
			t.Errorf("expected output total 20, got %v", finishPart.Usage.OutputTokens.Total)
		}
	})
}
