// Ported from: packages/anthropic/src/anthropic-messages-language-model.test.ts
package anthropic

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

// --- Test Helpers ---

var testPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "Hello"},
		},
	},
}

type lmRequestCapture struct {
	Body    []byte
	Headers http.Header
}

func (rc *lmRequestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

func createLMJSONTestServer(body any) (*httptest.Server, *lmRequestCapture) {
	capture := &lmRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

func createLMSSETestServer(chunks []string) (*httptest.Server, *lmRequestCapture) {
	capture := &lmRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
	}))
	return server, capture
}

func anthropicTextFixture() map[string]any {
	return map[string]any{
		"model": "claude-3-haiku-20240307",
		"id":    "msg_01VdEjxAP5ahtHKrrRdNBteQ",
		"type":  "message",
		"role":  "assistant",
		"content": []any{
			map[string]any{"type": "text", "text": "Hello! How are you doing today?"},
		},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":                float64(12),
			"cache_creation_input_tokens": float64(0),
			"cache_read_input_tokens":     float64(0),
			"output_tokens":               float64(29),
		},
	}
}

func anthropicToolUseFixture() map[string]any {
	return map[string]any{
		"model": "claude-3-haiku-20240307",
		"id":    "msg_tooluse123",
		"type":  "message",
		"role":  "assistant",
		"content": []any{
			map[string]any{
				"type":  "tool_use",
				"id":    "toolu_01Q9ExVZnzZj7E2QQYHYtNUa",
				"name":  "weather",
				"input": map[string]any{"location": "San Francisco"},
			},
		},
		"stop_reason":   "tool_use",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  float64(100),
			"output_tokens": float64(50),
		},
	}
}

func anthropicThinkingFixture() map[string]any {
	return map[string]any{
		"id":   "msg_017TfcQ4AgGxKyBduUpqYPZn",
		"type": "message",
		"role": "assistant",
		"content": []any{
			map[string]any{
				"type":      "thinking",
				"thinking":  "I am thinking...",
				"signature": "1234567890",
			},
			map[string]any{
				"type": "text",
				"text": "Hello, World!",
			},
		},
		"model":         "claude-3-haiku-20240307",
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]any{
			"input_tokens":  float64(4),
			"output_tokens": float64(30),
		},
	}
}

func anthropicJSONToolFixture() map[string]any {
	return map[string]any{
		"model": "claude-haiku-4-5-20251001",
		"id":    "msg_0191iYfpERYfS27xLsdW2nbb",
		"type":  "message",
		"role":  "assistant",
		"content": []any{
			map[string]any{
				"type": "tool_use",
				"id":   "toolu_01Q9ExVZnzZj7E2QQYHYtNUa",
				"name": "json",
				"input": map[string]any{
					"name": "test-name",
				},
			},
		},
		"stop_reason": "tool_use",
		"usage": map[string]any{
			"input_tokens":  float64(1151),
			"output_tokens": float64(87),
		},
	}
}

func createTestModel(baseURL string) *AnthropicMessagesLanguageModel {
	provider, _ := CreateAnthropic(AnthropicProviderSettings{
		ApiKey:  strPtr("test-api-key"),
		BaseURL: strPtr(baseURL),
	})
	return provider.createChatModel("claude-3-haiku-20240307")
}

func createTestModelWithID(baseURL string, modelID string) *AnthropicMessagesLanguageModel {
	provider, _ := CreateAnthropic(AnthropicProviderSettings{
		ApiKey:  strPtr("test-api-key"),
		BaseURL: strPtr(baseURL),
	})
	return provider.createChatModel(AnthropicMessagesModelId(modelID))
}

// --- DoGenerate Tests ---

func TestDoGenerate_TextResponse(t *testing.T) {
	t.Run("should extract text content from response", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) == 0 {
			t.Fatal("expected at least one content part")
		}
		textContent, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[0])
		}
		if textContent.Text != "Hello! How are you doing today?" {
			t.Errorf("expected text content, got %q", textContent.Text)
		}
	})

	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 12 {
			t.Errorf("expected input tokens total 12, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 29 {
			t.Errorf("expected output tokens total 29, got %v", result.Usage.OutputTokens.Total)
		}
	})

	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != "stop" {
			t.Errorf("expected finish reason type 'stop', got %q", result.FinishReason.Unified)
		}
	})
}

func TestDoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass model and messages to the API", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
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
		if body["model"] != "claude-3-haiku-20240307" {
			t.Errorf("expected model 'claude-3-haiku-20240307', got %v", body["model"])
		}
		messages, _ := body["messages"].([]any)
		if len(messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(messages))
		}
		msg := messages[0].(map[string]any)
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
	})

	t.Run("should set max_tokens from maxOutputTokens option", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		maxTokens := 500
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			MaxOutputTokens: &maxTokens,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		gotMax := int(body["max_tokens"].(float64))
		if gotMax != 500 {
			t.Errorf("expected max_tokens 500, got %d", gotMax)
		}
	})

	t.Run("should pass temperature", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		temp := 0.7
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		gotTemp := body["temperature"].(float64)
		if gotTemp != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", gotTemp)
		}
	})

	t.Run("should pass stop sequences", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:        testPrompt,
			StopSequences: []string{"STOP", "END"},
			Ctx:           context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		stopSeqs, ok := body["stop_sequences"].([]any)
		if !ok {
			t.Fatal("expected stop_sequences in body")
		}
		if len(stopSeqs) != 2 {
			t.Fatalf("expected 2 stop sequences, got %d", len(stopSeqs))
		}
		if stopSeqs[0] != "STOP" || stopSeqs[1] != "END" {
			t.Errorf("expected stop sequences ['STOP', 'END'], got %v", stopSeqs)
		}
	})

	t.Run("should pass headers to the API", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiKey := capture.Headers.Get("X-Api-Key")
		if apiKey != "test-api-key" {
			t.Errorf("expected x-api-key 'test-api-key', got %q", apiKey)
		}
	})
}

func TestDoGenerate_TemperatureClamping(t *testing.T) {
	t.Run("should clamp temperature above 1 to 1", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		temp := 1.5
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		gotTemp := body["temperature"].(float64)
		if gotTemp != 1.0 {
			t.Errorf("expected clamped temperature 1.0, got %v", gotTemp)
		}

		// Should have a warning
		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "temperature" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected temperature clamping warning")
		}
	})

	t.Run("should clamp temperature below 0 to 0", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		temp := -0.5
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		gotTemp := body["temperature"].(float64)
		if gotTemp != 0.0 {
			t.Errorf("expected clamped temperature 0.0, got %v", gotTemp)
		}
	})
}

func TestDoGenerate_TemperatureTopPMutualExclusivity(t *testing.T) {
	t.Run("should remove topP when temperature is set", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		temp := 0.5
		topP := 0.7
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Temperature: &temp,
			TopP:        &topP,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["top_p"]; ok {
			t.Error("expected top_p to be removed when temperature is set")
		}
		if body["temperature"].(float64) != 0.5 {
			t.Errorf("expected temperature 0.5, got %v", body["temperature"])
		}

		// Should have a warning
		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "topP" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected topP removal warning")
		}
	})
}

func TestDoGenerate_UnsupportedSettings(t *testing.T) {
	t.Run("should warn on frequencyPenalty", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		fp := 0.5
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:           testPrompt,
			FrequencyPenalty: &fp,
			Ctx:              context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "frequencyPenalty" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected frequencyPenalty warning")
		}
	})

	t.Run("should warn on presencePenalty", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		pp := 0.5
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			PresencePenalty: &pp,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "presencePenalty" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected presencePenalty warning")
		}
	})

	t.Run("should warn on seed", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		seed := 42
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Seed:   &seed,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "seed" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected seed warning")
		}
	})
}

func TestDoGenerate_Thinking(t *testing.T) {
	t.Run("should pass thinking config and add budget tokens", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModelWithID(server.URL, "claude-sonnet-4-5")
		maxTokens := 20000
		temp := 0.5
		topP := 0.7
		topK := 1
		budgetTokens := 1000
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			MaxOutputTokens: &maxTokens,
			Temperature:     &temp,
			TopP:            &topP,
			TopK:            &topK,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"thinking": map[string]any{"type": "enabled", "budgetTokens": float64(1000)},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()

		// Max tokens should include budget
		gotMax := int(body["max_tokens"].(float64))
		if gotMax != 20000+budgetTokens {
			t.Errorf("expected max_tokens %d, got %d", 20000+budgetTokens, gotMax)
		}

		// Thinking config
		thinking := body["thinking"].(map[string]any)
		if thinking["type"] != "enabled" {
			t.Errorf("expected thinking type 'enabled', got %v", thinking["type"])
		}
		if int(thinking["budget_tokens"].(float64)) != 1000 {
			t.Errorf("expected budget_tokens 1000, got %v", thinking["budget_tokens"])
		}

		// Temperature, topP, topK should be removed when thinking enabled
		if _, ok := body["temperature"]; ok {
			t.Error("expected temperature to be removed when thinking enabled")
		}
		if _, ok := body["top_p"]; ok {
			t.Error("expected top_p to be removed when thinking enabled")
		}
		if _, ok := body["top_k"]; ok {
			t.Error("expected top_k to be removed when thinking enabled")
		}

		// Should have warnings for removed params
		warningFeatures := map[string]bool{}
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok {
				warningFeatures[uw.Feature] = true
			}
		}
		if !warningFeatures["temperature"] {
			t.Error("expected temperature warning")
		}
		if !warningFeatures["topK"] {
			t.Error("expected topK warning")
		}
		if !warningFeatures["topP"] {
			t.Error("expected topP warning")
		}
	})

	t.Run("should extract reasoning response", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicThinkingFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(result.Content))
		}

		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[0])
		}
		if reasoning.Text != "I am thinking..." {
			t.Errorf("expected reasoning text 'I am thinking...', got %q", reasoning.Text)
		}
		if reasoning.ProviderMetadata == nil {
			t.Fatal("expected provider metadata")
		}
		antMeta := reasoning.ProviderMetadata["anthropic"]
		if antMeta == nil {
			t.Fatal("expected anthropic metadata")
		}

		text, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[1])
		}
		if text.Text != "Hello, World!" {
			t.Errorf("expected text 'Hello, World!', got %q", text.Text)
		}
	})

	t.Run("should use default budget when thinking enabled without budgetTokens", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModelWithID(server.URL, "claude-sonnet-4-5")
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"thinking": map[string]any{"type": "enabled"},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		thinking := body["thinking"].(map[string]any)
		if thinking["type"] != "enabled" {
			t.Errorf("expected thinking type 'enabled', got %v", thinking["type"])
		}
		if int(thinking["budget_tokens"].(float64)) != 1024 {
			t.Errorf("expected default budget_tokens 1024, got %v", thinking["budget_tokens"])
		}

		// Should have compatibility warning
		hasWarning := false
		for _, w := range result.Warnings {
			if cw, ok := w.(shared.CompatibilityWarning); ok && cw.Feature == "extended thinking" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected compatibility warning for missing budgetTokens")
		}
	})

	t.Run("should send adaptive thinking without budget_tokens", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModelWithID(server.URL, "claude-opus-4-6")
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"thinking": map[string]any{"type": "adaptive"},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		thinking := body["thinking"].(map[string]any)
		if thinking["type"] != "adaptive" {
			t.Errorf("expected thinking type 'adaptive', got %v", thinking["type"])
		}
		if _, ok := thinking["budget_tokens"]; ok {
			t.Error("expected no budget_tokens for adaptive thinking")
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected no warnings for adaptive thinking, got %v", result.Warnings)
		}
	})
}

func TestDoGenerate_JSONSchemaResponseFormat(t *testing.T) {
	t.Run("should pass json schema response format as a tool for unsupported models", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicJSONToolFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required":             []any{"name"},
					"additionalProperties": false,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()

		// Should have tools with json tool
		tools := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		tool := tools[0].(map[string]any)
		if tool["name"] != "json" {
			t.Errorf("expected tool name 'json', got %v", tool["name"])
		}
		if tool["description"] != "Respond with a JSON object." {
			t.Errorf("expected tool description, got %v", tool["description"])
		}

		// Should have tool_choice for any with disable parallel
		toolChoice := body["tool_choice"].(map[string]any)
		if toolChoice["type"] != "any" {
			t.Errorf("expected tool_choice type 'any', got %v", toolChoice["type"])
		}
		if toolChoice["disable_parallel_tool_use"] != true {
			t.Error("expected disable_parallel_tool_use true")
		}
	})

	t.Run("should return the json response from tool use as text", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicJSONToolFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{"type": "string"},
					},
					"required": []any{"name"},
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
		// The json tool response should be extracted as text
		if !strings.Contains(text.Text, "test-name") {
			t.Errorf("expected text to contain 'test-name', got %q", text.Text)
		}
	})
}

func TestDoGenerate_ToolCalls(t *testing.T) {
	t.Run("should extract tool calls from response", func(t *testing.T) {
		server, _ := createLMJSONTestServer(anthropicToolUseFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{"type": "object", "properties": map[string]any{"location": map[string]any{"type": "string"}}},
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
		tc, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall content, got %T", result.Content[0])
		}
		if tc.ToolCallID != "toolu_01Q9ExVZnzZj7E2QQYHYtNUa" {
			t.Errorf("expected tool call ID, got %q", tc.ToolCallID)
		}
		if tc.ToolName != "weather" {
			t.Errorf("expected tool name 'weather', got %q", tc.ToolName)
		}
		if !strings.Contains(tc.Input, "San Francisco") {
			t.Errorf("expected input to contain 'San Francisco', got %v", tc.Input)
		}
	})

	t.Run("should pass tools to the API", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]any{"type": "string"},
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
		tools := body["tools"].([]any)
		if len(tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(tools))
		}
		tool := tools[0].(map[string]any)
		if tool["name"] != "weather" {
			t.Errorf("expected tool name 'weather', got %v", tool["name"])
		}
	})

	t.Run("should pass tool choice auto", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
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
		toolChoice := body["tool_choice"].(map[string]any)
		if toolChoice["type"] != "auto" {
			t.Errorf("expected tool_choice type 'auto', got %v", toolChoice["type"])
		}
	})

	t.Run("should pass tool choice required", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{"type": "object"},
				},
			},
			ToolChoice: languagemodel.ToolChoiceRequired{},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		toolChoice := body["tool_choice"].(map[string]any)
		if toolChoice["type"] != "any" {
			t.Errorf("expected tool_choice type 'any', got %v", toolChoice["type"])
		}
	})

	t.Run("should pass tool choice none", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{"type": "object"},
				},
			},
			ToolChoice: languagemodel.ToolChoiceNone{},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		// When tool choice is none, tools should not be sent
		if _, ok := body["tools"]; ok {
			t.Error("expected no tools when tool choice is none")
		}
	})

	t.Run("should pass specific tool choice", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{"type": "object"},
				},
			},
			ToolChoice: languagemodel.ToolChoiceTool{ToolName: "weather"},
			Ctx:        context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		toolChoice := body["tool_choice"].(map[string]any)
		if toolChoice["type"] != "tool" {
			t.Errorf("expected tool_choice type 'tool', got %v", toolChoice["type"])
		}
		if toolChoice["name"] != "weather" {
			t.Errorf("expected tool_choice name 'weather', got %v", toolChoice["name"])
		}
	})
}

func TestDoGenerate_Effort(t *testing.T) {
	t.Run("should pass effort setting", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"effort": "high",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		outputConfig, ok := body["output_config"].(map[string]any)
		if !ok {
			t.Fatal("expected output_config in body")
		}
		if outputConfig["effort"] != "high" {
			t.Errorf("expected effort 'high', got %v", outputConfig["effort"])
		}
	})
}

func TestDoGenerate_Speed(t *testing.T) {
	t.Run("should pass speed setting", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"speed": "fast",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["speed"] != "fast" {
			t.Errorf("expected speed 'fast', got %v", body["speed"])
		}
	})
}

func TestDoGenerate_ContextManagement(t *testing.T) {
	t.Run("should pass clear_tool_uses context management", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"contextManagement": map[string]any{
						"edits": []any{
							map[string]any{
								"type": "clear_tool_uses_20250919",
								"trigger": map[string]any{
									"type":  "token_count",
									"value": float64(100000),
								},
							},
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
		cm, ok := body["context_management"].(map[string]any)
		if !ok {
			t.Fatal("expected context_management in body")
		}
		edits := cm["edits"].([]any)
		if len(edits) != 1 {
			t.Fatalf("expected 1 edit, got %d", len(edits))
		}
		edit := edits[0].(map[string]any)
		if edit["type"] != "clear_tool_uses_20250919" {
			t.Errorf("expected edit type, got %v", edit["type"])
		}
	})

	t.Run("should pass clear_thinking context management", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"contextManagement": map[string]any{
						"edits": []any{
							map[string]any{
								"type": "clear_thinking_20251015",
								"keep": map[string]any{
									"type":  "last_n",
									"value": float64(2),
								},
							},
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
		cm := body["context_management"].(map[string]any)
		edits := cm["edits"].([]any)
		edit := edits[0].(map[string]any)
		if edit["type"] != "clear_thinking_20251015" {
			t.Errorf("expected type 'clear_thinking_20251015', got %v", edit["type"])
		}
		keep := edit["keep"].(map[string]any)
		if keep["type"] != "last_n" {
			t.Errorf("expected keep type 'last_n', got %v", keep["type"])
		}
	})
}

func TestDoGenerate_MCPServers(t *testing.T) {
	t.Run("should pass MCP server configuration", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		model := createTestModel(server.URL)
		authToken := "test-mcp-token"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"anthropic": map[string]any{
					"mcpServers": []any{
						map[string]any{
							"type":               "url",
							"name":               "test-mcp",
							"url":                "https://mcp.example.com",
							"authorizationToken": authToken,
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
		mcpServers := body["mcp_servers"].([]any)
		if len(mcpServers) != 1 {
			t.Fatalf("expected 1 MCP server, got %d", len(mcpServers))
		}
		srv := mcpServers[0].(map[string]any)
		if srv["type"] != "url" {
			t.Errorf("expected type 'url', got %v", srv["type"])
		}
		if srv["name"] != "test-mcp" {
			t.Errorf("expected name 'test-mcp', got %v", srv["name"])
		}
		if srv["url"] != "https://mcp.example.com" {
			t.Errorf("expected url, got %v", srv["url"])
		}

		// Should add MCP beta header
		betaHeader := capture.Headers.Get("Anthropic-Beta")
		if !strings.Contains(betaHeader, "mcp-client-2025-04-04") {
			t.Errorf("expected mcp beta header, got %q", betaHeader)
		}
	})
}

func TestDoGenerate_FinishReasons(t *testing.T) {
	finishReasonTests := []struct {
		apiReason    string
		expectedType languagemodel.UnifiedFinishReason
	}{
		{"end_turn", languagemodel.FinishReasonStop},
		{"stop_sequence", languagemodel.FinishReasonStop},
		{"tool_use", languagemodel.FinishReasonToolCalls},
		{"max_tokens", languagemodel.FinishReasonLength},
	}

	for _, tc := range finishReasonTests {
		t.Run(fmt.Sprintf("should map %s to %s", tc.apiReason, string(tc.expectedType)), func(t *testing.T) {
			fixture := anthropicTextFixture()
			fixture["stop_reason"] = tc.apiReason
			server, _ := createLMJSONTestServer(fixture)
			defer server.Close()

			model := createTestModel(server.URL)
			result, err := model.DoGenerate(languagemodel.CallOptions{
				Prompt: testPrompt,
				Ctx:    context.Background(),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.FinishReason.Unified != tc.expectedType {
				t.Errorf("expected finish reason '%s', got %q", tc.expectedType, result.FinishReason.Unified)
			}
		})
	}
}

func TestDoGenerate_RedactedThinking(t *testing.T) {
	t.Run("should handle redacted thinking blocks", func(t *testing.T) {
		fixture := map[string]any{
			"id":   "msg_test",
			"type": "message",
			"role": "assistant",
			"content": []any{
				map[string]any{
					"type": "redacted_thinking",
					"data": "encrypted-data-blob",
				},
				map[string]any{
					"type": "text",
					"text": "I've thought about it.",
				},
			},
			"model":         "claude-3-haiku-20240307",
			"stop_reason":   "end_turn",
			"stop_sequence": nil,
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(20),
			},
		}

		server, _ := createLMJSONTestServer(fixture)
		defer server.Close()

		model := createTestModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(result.Content))
		}

		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[0])
		}
		// Redacted thinking should have empty text and redactedData in metadata
		if reasoning.Text != "" {
			t.Errorf("expected empty text for redacted thinking, got %q", reasoning.Text)
		}
		antMeta := reasoning.ProviderMetadata["anthropic"]
		if antMeta == nil {
			t.Fatal("expected anthropic metadata")
		}
	})
}

func TestDoGenerate_APIError(t *testing.T) {
	t.Run("should handle API errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]any{
				"type": "error",
				"error": map[string]any{
					"type":    "rate_limit_error",
					"message": "Rate limit exceeded",
				},
			})
		}))
		defer server.Close()

		model := createTestModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err == nil {
			t.Fatal("expected error for rate limit response")
		}
	})
}

func TestDoGenerate_CustomProviderName(t *testing.T) {
	t.Run("should use custom provider name for provider options lookup", func(t *testing.T) {
		server, capture := createLMJSONTestServer(anthropicTextFixture())
		defer server.Close()

		customName := "my-proxy"
		provider, _ := CreateAnthropic(AnthropicProviderSettings{
			ApiKey:  strPtr("test-api-key"),
			BaseURL: strPtr(server.URL),
			Name:    &customName,
		})
		model := provider.createChatModel("claude-3-haiku-20240307")

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			ProviderOptions: shared.ProviderOptions{
				"my-proxy": map[string]any{
					"speed": "fast",
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["speed"] != "fast" {
			t.Errorf("expected speed 'fast' from custom provider options, got %v", body["speed"])
		}
	})
}

// --- DoStream Tests ---

func TestDoStream_TextResponse(t *testing.T) {
	t.Run("should stream text content", func(t *testing.T) {
		chunks := []string{
			`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-haiku-20240307","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" World"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`,
			`{"type":"message_stop"}`,
		}

		server, _ := createLMSSETestServer(chunks)
		defer server.Close()

		model := createTestModel(server.URL)
		stream, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parts []languagemodel.StreamPart
		for part := range stream.Stream {
			parts = append(parts, part)
		}

		if len(parts) == 0 {
			t.Fatal("expected stream parts")
		}

		// Should contain text deltas
		hasTextDelta := false
		fullText := ""
		for _, part := range parts {
			if td, ok := part.(languagemodel.StreamPartTextDelta); ok {
				hasTextDelta = true
				fullText += td.Delta
			}
		}
		if !hasTextDelta {
			t.Error("expected at least one text delta")
		}
		if fullText != "Hello World" {
			t.Errorf("expected full text 'Hello World', got %q", fullText)
		}
	})

	t.Run("should include finish reason in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-haiku-20240307","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":1}}`,
			`{"type":"message_stop"}`,
		}

		server, _ := createLMSSETestServer(chunks)
		defer server.Close()

		model := createTestModel(server.URL)
		stream, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parts []languagemodel.StreamPart
		for part := range stream.Stream {
			parts = append(parts, part)
		}

		hasFinishReason := false
		for _, part := range parts {
			if fr, ok := part.(languagemodel.StreamPartFinish); ok {
				hasFinishReason = true
				if fr.FinishReason.Unified != languagemodel.FinishReasonStop {
					t.Errorf("expected finish reason 'stop', got %q", fr.FinishReason.Unified)
				}
				break
			}
		}
		if !hasFinishReason {
			t.Error("expected finish reason in stream")
		}
	})

	t.Run("should pass request body with stream true", func(t *testing.T) {
		chunks := []string{
			`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-haiku-20240307","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":0}}`,
			`{"type":"message_stop"}`,
		}

		server, capture := createLMSSETestServer(chunks)
		defer server.Close()

		model := createTestModel(server.URL)
		stream, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Drain stream
		for range stream.Stream {
		}

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Error("expected stream=true in request body")
		}
	})
}

func TestDoStream_ThinkingResponse(t *testing.T) {
	t.Run("should stream thinking content", func(t *testing.T) {
		chunks := []string{
			`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-haiku-20240307","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"Let me think..."}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`,
			`{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Answer"}}`,
			`{"type":"content_block_stop","index":1}`,
			`{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":10}}`,
			`{"type":"message_stop"}`,
		}

		server, _ := createLMSSETestServer(chunks)
		defer server.Close()

		model := createTestModel(server.URL)
		stream, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parts []languagemodel.StreamPart
		for part := range stream.Stream {
			parts = append(parts, part)
		}

		// Should contain reasoning delta
		hasReasoningDelta := false
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartReasoningDelta); ok {
				hasReasoningDelta = true
				break
			}
		}
		if !hasReasoningDelta {
			t.Error("expected reasoning delta in stream")
		}
	})
}

func TestDoStream_ToolCallResponse(t *testing.T) {
	t.Run("should stream tool call content", func(t *testing.T) {
		chunks := []string{
			`{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-haiku-20240307","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_123","name":"weather","input":{}}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"location\":"}}`,
			`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"SF\"}"}}`,
			`{"type":"content_block_stop","index":0}`,
			`{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":20}}`,
			`{"type":"message_stop"}`,
		}

		server, _ := createLMSSETestServer(chunks)
		defer server.Close()

		model := createTestModel(server.URL)
		stream, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "weather",
					Description: strPtr("Get weather"),
					InputSchema: map[string]any{"type": "object"},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parts []languagemodel.StreamPart
		for part := range stream.Stream {
			parts = append(parts, part)
		}

		// Should contain tool call start and/or delta
		hasToolCall := false
		for _, part := range parts {
			if _, ok := part.(languagemodel.StreamPartToolInputStart); ok {
				hasToolCall = true
				break
			}
			if _, ok := part.(languagemodel.StreamPartToolInputDelta); ok {
				hasToolCall = true
				break
			}
		}
		if !hasToolCall {
			t.Error("expected tool call in stream")
		}
	})
}
