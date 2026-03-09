// Ported from: packages/xai/src/responses/xai-responses-language-model.test.ts
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
)

// responsesTestPrompt is the standard prompt for responses API tests.
var responsesTestPrompt = languagemodel.Prompt{
	languagemodel.UserMessage{
		Content: []languagemodel.UserMessagePart{
			languagemodel.TextPart{Text: "hello"},
		},
	},
}

// createResponsesTestServer creates a JSON test server for the responses API.
func createResponsesTestServer(body map[string]any, headers map[string]string) (*httptest.Server, *requestCapture) {
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

// createResponsesSSETestServer creates an SSE test server for the responses API.
func createResponsesSSETestServer(chunks []string, headers map[string]string) (*httptest.Server, *requestCapture) {
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

// createResponsesModel creates an xAI responses language model pointing at a test server.
func createResponsesModel(serverURL string, modelId ...string) *XaiResponsesLanguageModel {
	id := "grok-4-fast-non-reasoning"
	if len(modelId) > 0 {
		id = modelId[0]
	}
	idCounter := 0
	return NewXaiResponsesLanguageModel(id, XaiResponsesConfig{
		Provider: "xai.responses",
		BaseURL:  serverURL,
		Headers:  func() map[string]string { return map[string]string{"Authorization": "Bearer test-key"} },
		GenerateID: func() string {
			result := fmt.Sprintf("id-%d", idCounter)
			idCounter++
			return result
		},
	})
}

// responsesTextFixture returns a basic responses API text response.
func responsesTextFixture() map[string]any {
	return map[string]any{
		"id":         "resp_123",
		"object":     "response",
		"created_at": float64(1700000000),
		"status":     "completed",
		"model":      "grok-4-fast-non-reasoning",
		"output": []any{
			map[string]any{
				"type":   "message",
				"id":     "msg_123",
				"status": "completed",
				"role":   "assistant",
				"content": []any{
					map[string]any{
						"type":        "output_text",
						"text":        "hello world",
						"annotations": []any{},
					},
				},
			},
		},
		"usage": map[string]any{
			"input_tokens":  float64(10),
			"output_tokens": float64(5),
			"total_tokens":  float64(15),
		},
	}
}

// emptyResponsesFixture returns a minimal responses API response with no output.
func emptyResponsesFixture() map[string]any {
	return map[string]any{
		"id":     "resp_123",
		"object": "response",
		"status": "completed",
		"model":  "grok-4-fast-non-reasoning",
		"output": []any{},
		"usage": map[string]any{
			"input_tokens":  float64(10),
			"output_tokens": float64(5),
		},
	}
}

// ---- DoGenerate Tests ----

func TestResponsesDoGenerate_BasicText(t *testing.T) {
	t.Run("should generate text content", func(t *testing.T) {
		server, _ := createResponsesTestServer(responsesTextFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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
		if text.Text != "hello world" {
			t.Errorf("expected 'hello world', got %q", text.Text)
		}
	})
}

func TestResponsesDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage correctly", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{},
			"usage": map[string]any{
				"input_tokens":  float64(345),
				"output_tokens": float64(538),
				"total_tokens":  float64(883),
				"output_tokens_details": map[string]any{
					"reasoning_tokens": float64(123),
				},
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if intVal(result.Usage.InputTokens.Total) != 345 {
			t.Errorf("expected InputTokens.Total 345, got %d", intVal(result.Usage.InputTokens.Total))
		}
		if intVal(result.Usage.InputTokens.NoCache) != 345 {
			t.Errorf("expected InputTokens.NoCache 345, got %d", intVal(result.Usage.InputTokens.NoCache))
		}
		if intVal(result.Usage.InputTokens.CacheRead) != 0 {
			t.Errorf("expected InputTokens.CacheRead 0, got %d", intVal(result.Usage.InputTokens.CacheRead))
		}
		if intVal(result.Usage.OutputTokens.Total) != 538 {
			t.Errorf("expected OutputTokens.Total 538, got %d", intVal(result.Usage.OutputTokens.Total))
		}
		if intVal(result.Usage.OutputTokens.Reasoning) != 123 {
			t.Errorf("expected OutputTokens.Reasoning 123, got %d", intVal(result.Usage.OutputTokens.Reasoning))
		}
		// text = 538 - 123 = 415
		if intVal(result.Usage.OutputTokens.Text) != 415 {
			t.Errorf("expected OutputTokens.Text 415, got %d", intVal(result.Usage.OutputTokens.Text))
		}
	})
}

func TestResponsesDoGenerate_FinishReason(t *testing.T) {
	t.Run("should extract finish reason from status", func(t *testing.T) {
		server, _ := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected finish reason stop, got %v", result.FinishReason.Unified)
		}
		if *result.FinishReason.Raw != "completed" {
			t.Errorf("expected raw finish reason 'completed', got %v", result.FinishReason.Raw)
		}
	})
}

func TestResponsesDoGenerate_ReasoningWithEncrypted(t *testing.T) {
	t.Run("should extract reasoning with encrypted content when store=false", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":   "reasoning",
					"id":     "rs_456",
					"status": "completed",
					"summary": []any{
						map[string]any{
							"type": "summary_text",
							"text": "First, analyze the question carefully.",
						},
					},
					"encrypted_content": "abc123encryptedcontent",
				},
				map[string]any{
					"type":   "message",
					"id":     "msg_123",
					"status": "completed",
					"role":   "assistant",
					"content": []any{
						map[string]any{
							"type":        "output_text",
							"text":        "The answer is 42.",
							"annotations": []any{},
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(20),
				"output_tokens_details": map[string]any{
					"reasoning_tokens": float64(15),
				},
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(result.Content))
		}

		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[0])
		}
		if reasoning.Text != "First, analyze the question carefully." {
			t.Errorf("expected reasoning text, got %q", reasoning.Text)
		}
		if reasoning.ProviderMetadata == nil {
			t.Fatal("expected non-nil provider metadata")
		}
		xaiMeta, ok := reasoning.ProviderMetadata["xai"]
		if !ok {
			t.Fatal("expected xai provider metadata")
		}
		if xaiMeta["reasoningEncryptedContent"] != "abc123encryptedcontent" {
			t.Errorf("expected encrypted content, got %v", xaiMeta["reasoningEncryptedContent"])
		}
		if xaiMeta["itemId"] != "rs_456" {
			t.Errorf("expected itemId 'rs_456', got %v", xaiMeta["itemId"])
		}

		text, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[1])
		}
		if text.Text != "The answer is 42." {
			t.Errorf("expected text 'The answer is 42.', got %q", text.Text)
		}
	})
}

func TestResponsesDoGenerate_ReasoningWithoutEncrypted(t *testing.T) {
	t.Run("should handle reasoning without encrypted content", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":   "reasoning",
					"id":     "rs_456",
					"status": "completed",
					"summary": []any{
						map[string]any{
							"type": "summary_text",
							"text": "Thinking through the problem.",
						},
					},
				},
				map[string]any{
					"type":   "message",
					"id":     "msg_123",
					"status": "completed",
					"role":   "assistant",
					"content": []any{
						map[string]any{
							"type":        "output_text",
							"text":        "Solution found.",
							"annotations": []any{},
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(15),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(result.Content))
		}

		reasoning, ok := result.Content[0].(languagemodel.Reasoning)
		if !ok {
			t.Fatalf("expected Reasoning content, got %T", result.Content[0])
		}
		if reasoning.Text != "Thinking through the problem." {
			t.Errorf("expected reasoning text, got %q", reasoning.Text)
		}
		// Should have provider metadata with itemId but no encrypted content
		if reasoning.ProviderMetadata == nil {
			t.Fatal("expected non-nil provider metadata")
		}
		xaiMeta := reasoning.ProviderMetadata["xai"]
		if xaiMeta["itemId"] != "rs_456" {
			t.Errorf("expected itemId 'rs_456', got %v", xaiMeta["itemId"])
		}
		if _, hasEncrypted := xaiMeta["reasoningEncryptedContent"]; hasEncrypted {
			t.Error("expected no encrypted content")
		}

		text, ok := result.Content[1].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text content, got %T", result.Content[1])
		}
		if text.Text != "Solution found." {
			t.Errorf("expected 'Solution found.', got %q", text.Text)
		}
	})
}

func TestResponsesDoGenerate_SettingsAndOptions(t *testing.T) {
	t.Run("should send model id and settings", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL, "grok-4-fast-non-reasoning")
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
		if body["max_output_tokens"] != float64(100) {
			t.Errorf("expected max_output_tokens 100, got %v", body["max_output_tokens"])
		}
	})
}

func TestResponsesDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("reasoningEffort", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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
		reasoning, ok := body["reasoning"].(map[string]interface{})
		if !ok {
			t.Fatal("expected reasoning in body")
		}
		if reasoning["effort"] != "high" {
			t.Errorf("expected effort 'high', got %v", reasoning["effort"])
		}
	})

	t.Run("logprobs and topLogprobs", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"logprobs":    true,
					"topLogprobs": float64(6),
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
		if body["top_logprobs"] != float64(6) {
			t.Errorf("expected top_logprobs 6, got %v", body["top_logprobs"])
		}
	})

	t.Run("topLogprobs enables logprobs", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"topLogprobs": float64(2),
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
		if body["top_logprobs"] != float64(2) {
			t.Errorf("expected top_logprobs 2, got %v", body["top_logprobs"])
		}
	})

	t.Run("store:true should not include store or include", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"store": true,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, hasStore := body["store"]; hasStore {
			t.Error("expected no store in body when store=true")
		}
		if _, hasInclude := body["include"]; hasInclude {
			t.Error("expected no include in body when store=true")
		}
	})

	t.Run("store:false should include store and encrypted content include", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"store": false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["store"] != false {
			t.Errorf("expected store false, got %v", body["store"])
		}
		include, ok := body["include"].([]interface{})
		if !ok {
			t.Fatal("expected include array in body")
		}
		if len(include) != 1 || include[0] != "reasoning.encrypted_content" {
			t.Errorf("expected include ['reasoning.encrypted_content'], got %v", include)
		}
	})

	t.Run("previousResponseId", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"previousResponseId": "resp_456",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["previous_response_id"] != "resp_456" {
			t.Errorf("expected previous_response_id 'resp_456', got %v", body["previous_response_id"])
		}
	})

	t.Run("include with file_search_call.results", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"include": []interface{}{"file_search_call.results"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		include, ok := body["include"].([]interface{})
		if !ok {
			t.Fatal("expected include array")
		}
		if len(include) != 1 || include[0] != "file_search_call.results" {
			t.Errorf("expected include ['file_search_call.results'], got %v", include)
		}
	})

	t.Run("include with file_search_call.results and store:false", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"include": []interface{}{"file_search_call.results"},
					"store":   false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["store"] != false {
			t.Errorf("expected store false, got %v", body["store"])
		}
		include, ok := body["include"].([]interface{})
		if !ok {
			t.Fatal("expected include array")
		}
		if len(include) != 2 {
			t.Fatalf("expected 2 include items, got %d", len(include))
		}
		if include[0] != "file_search_call.results" {
			t.Errorf("expected first include 'file_search_call.results', got %v", include[0])
		}
		if include[1] != "reasoning.encrypted_content" {
			t.Errorf("expected second include 'reasoning.encrypted_content', got %v", include[1])
		}
	})
}

func TestResponsesDoGenerate_StopSequencesWarning(t *testing.T) {
	t.Run("should warn about unsupported stopSequences", func(t *testing.T) {
		server, _ := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		stopSeqs := []string{"\n\n", "STOP"}
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:        responsesTestPrompt,
			Ctx:           context.Background(),
			StopSequences: stopSeqs,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var stopWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "stopSequences" {
				stopWarning = true
			}
		}
		if !stopWarning {
			t.Error("expected unsupported warning for stopSequences")
		}
	})
}

func TestResponsesDoGenerate_ResponseFormat(t *testing.T) {
	t.Run("should send json_schema response format", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		name := "recipe"
		desc := "A recipe object"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Name:        &name,
				Description: &desc,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name":        map[string]any{"type": "string"},
						"ingredients": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required":             []any{"name", "ingredients"},
					"additionalProperties": false,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		textMap, ok := body["text"].(map[string]interface{})
		if !ok {
			t.Fatal("expected text in body")
		}
		formatMap, ok := textMap["format"].(map[string]interface{})
		if !ok {
			t.Fatal("expected format in text")
		}
		if formatMap["type"] != "json_schema" {
			t.Errorf("expected type 'json_schema', got %v", formatMap["type"])
		}
		if formatMap["strict"] != true {
			t.Errorf("expected strict true, got %v", formatMap["strict"])
		}
		if formatMap["name"] != "recipe" {
			t.Errorf("expected name 'recipe', got %v", formatMap["name"])
		}
		if formatMap["description"] != "A recipe object" {
			t.Errorf("expected description 'A recipe object', got %v", formatMap["description"])
		}
	})

	t.Run("should send json_object when no schema", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:         responsesTestPrompt,
			Ctx:            context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		textMap := body["text"].(map[string]interface{})
		formatMap := textMap["format"].(map[string]interface{})
		if formatMap["type"] != "json_object" {
			t.Errorf("expected type 'json_object', got %v", formatMap["type"])
		}
	})

	t.Run("should use default name when not provided", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			ResponseFormat: languagemodel.ResponseFormatJSON{
				Schema: map[string]any{
					"type":       "object",
					"properties": map[string]any{"value": map[string]any{"type": "string"}},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		textMap := body["text"].(map[string]interface{})
		formatMap := textMap["format"].(map[string]interface{})
		if formatMap["name"] != "response" {
			t.Errorf("expected default name 'response', got %v", formatMap["name"])
		}
	})
}

func TestResponsesDoGenerate_WebSearchTool(t *testing.T) {
	t.Run("should send web_search tool in request", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]interface{}{
						"allowedDomains":          []interface{}{"wikipedia.org"},
						"enableImageUnderstanding": true,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		tools, ok := body["tools"].([]interface{})
		if !ok || len(tools) == 0 {
			t.Fatal("expected tools in body")
		}
		tool := tools[0].(map[string]interface{})
		if tool["type"] != "web_search" {
			t.Errorf("expected type 'web_search', got %v", tool["type"])
		}
	})

	t.Run("should include web_search tool call with providerExecuted true", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":      "web_search_call",
					"id":        "ws_123",
					"name":      "web_search",
					"arguments": `{"query":"test"}`,
					"call_id":   "",
					"status":    "completed",
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(5),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]interface{}{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) < 1 {
			t.Fatal("expected at least 1 content part")
		}
		tc, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if tc.ToolName != "web_search" {
			t.Errorf("expected toolName 'web_search', got %q", tc.ToolName)
		}
		if tc.ToolCallID != "ws_123" {
			t.Errorf("expected toolCallId 'ws_123', got %q", tc.ToolCallID)
		}
		if tc.ProviderExecuted == nil || !*tc.ProviderExecuted {
			t.Error("expected providerExecuted true")
		}
	})
}

func TestResponsesDoGenerate_FileSearchTool(t *testing.T) {
	t.Run("should include file_search tool call and result", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":    "file_search_call",
					"id":      "fs_123",
					"status":  "completed",
					"queries": []any{"AI safety research"},
					"results": []any{
						map[string]any{
							"file_id":  "file_abc123",
							"filename": "ai-safety-paper.pdf",
							"score":    0.95,
							"text":     "Recent advances in AI safety research have focused on alignment techniques...",
						},
						map[string]any{
							"file_id":  "file_def456",
							"filename": "research-notes.md",
							"score":    0.82,
							"text":     "Key findings from the AI safety workshop include recommendations for...",
						},
					},
				},
				map[string]any{
					"type":   "message",
					"id":     "msg_123",
					"status": "completed",
					"role":   "assistant",
					"content": []any{
						map[string]any{
							"type":        "output_text",
							"text":        "Based on the documents in your collection.",
							"annotations": []any{},
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(100),
				"output_tokens": float64(20),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]interface{}{
						"vectorStoreIds": []interface{}{"collection_test123"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have tool-call, tool-result, and text
		if len(result.Content) < 3 {
			t.Fatalf("expected at least 3 content parts, got %d", len(result.Content))
		}

		// Check tool call
		tc, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if tc.ToolName != "file_search" {
			t.Errorf("expected toolName 'file_search', got %q", tc.ToolName)
		}
		if tc.ToolCallID != "fs_123" {
			t.Errorf("expected toolCallId 'fs_123', got %q", tc.ToolCallID)
		}
		if tc.ProviderExecuted == nil || !*tc.ProviderExecuted {
			t.Error("expected providerExecuted true")
		}

		// Check tool result
		tr, ok := result.Content[1].(languagemodel.ToolResult)
		if !ok {
			t.Fatalf("expected ToolResult, got %T", result.Content[1])
		}
		if tr.ToolName != "file_search" {
			t.Errorf("expected toolName 'file_search', got %q", tr.ToolName)
		}
		resultMap, ok := tr.Result.(map[string]interface{})
		if !ok {
			t.Fatal("expected result to be map")
		}
		queries := resultMap["queries"].([]string)
		if len(queries) != 1 || queries[0] != "AI safety research" {
			t.Errorf("expected queries ['AI safety research'], got %v", queries)
		}
		results := resultMap["results"].([]map[string]interface{})
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		if results[0]["fileId"] != "file_abc123" {
			t.Errorf("expected first result fileId 'file_abc123', got %v", results[0]["fileId"])
		}
	})

	t.Run("should handle file_search with null results", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":    "file_search_call",
					"id":      "fs_456",
					"status":  "completed",
					"queries": []any{"nonexistent topic"},
					"results": nil,
				},
				map[string]any{
					"type":   "message",
					"id":     "msg_456",
					"status": "completed",
					"role":   "assistant",
					"content": []any{
						map[string]any{
							"type":        "output_text",
							"text":        "No relevant documents found.",
							"annotations": []any{},
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(100),
				"output_tokens": float64(20),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]interface{}{
						"vectorStoreIds": []interface{}{"collection_test123"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check tool result has null results
		if len(result.Content) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(result.Content))
		}
		tr, ok := result.Content[1].(languagemodel.ToolResult)
		if !ok {
			t.Fatalf("expected ToolResult, got %T", result.Content[1])
		}
		resultMap := tr.Result.(map[string]interface{})
		if resultMap["results"] != nil {
			t.Errorf("expected nil results, got %v", resultMap["results"])
		}
	})
}

func TestResponsesDoGenerate_FunctionTools(t *testing.T) {
	t.Run("should include function tool calls without providerExecuted", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":      "function_call",
					"id":        "fc_123",
					"name":      "weather",
					"arguments": `{"location":"sf"}`,
					"call_id":   "call_123",
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(5),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		desc := "get weather"
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content, got %d", len(result.Content))
		}
		tc, ok := result.Content[0].(languagemodel.ToolCall)
		if !ok {
			t.Fatalf("expected ToolCall, got %T", result.Content[0])
		}
		if tc.ToolName != "weather" {
			t.Errorf("expected toolName 'weather', got %q", tc.ToolName)
		}
		if tc.ToolCallID != "call_123" {
			t.Errorf("expected toolCallId 'call_123', got %q", tc.ToolCallID)
		}
		if tc.Input != `{"location":"sf"}` {
			t.Errorf("expected input, got %q", tc.Input)
		}
		if tc.ProviderExecuted != nil {
			t.Error("expected nil providerExecuted for function calls")
		}
	})
}

func TestResponsesDoGenerate_Citations(t *testing.T) {
	t.Run("should extract citations from annotations", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":   "message",
					"id":     "msg_123",
					"status": "completed",
					"role":   "assistant",
					"content": []any{
						map[string]any{
							"type": "output_text",
							"text": "based on research",
							"annotations": []any{
								map[string]any{
									"type":  "url_citation",
									"url":   "https://example.com",
									"title": "example title",
								},
								map[string]any{
									"type": "url_citation",
									"url":  "https://test.com",
								},
							},
						},
					},
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(5),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have: text, source, source
		if len(result.Content) != 3 {
			t.Fatalf("expected 3 content parts, got %d", len(result.Content))
		}

		text, ok := result.Content[0].(languagemodel.Text)
		if !ok {
			t.Fatalf("expected Text, got %T", result.Content[0])
		}
		if text.Text != "based on research" {
			t.Errorf("expected 'based on research', got %q", text.Text)
		}

		source1, ok := result.Content[1].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[1])
		}
		if source1.URL != "https://example.com" {
			t.Errorf("expected URL 'https://example.com', got %q", source1.URL)
		}
		if source1.Title == nil || *source1.Title != "example title" {
			t.Errorf("expected title 'example title', got %v", source1.Title)
		}
		if source1.ID != "id-0" {
			t.Errorf("expected id 'id-0', got %q", source1.ID)
		}

		source2, ok := result.Content[2].(languagemodel.SourceURL)
		if !ok {
			t.Fatalf("expected SourceURL, got %T", result.Content[2])
		}
		if source2.URL != "https://test.com" {
			t.Errorf("expected URL 'https://test.com', got %q", source2.URL)
		}
		// When no title, should use URL as title
		if source2.Title == nil || *source2.Title != "https://test.com" {
			t.Errorf("expected title 'https://test.com', got %v", source2.Title)
		}
	})
}

func TestResponsesDoGenerate_MultipleTools(t *testing.T) {
	t.Run("should handle multiple server-side tools", func(t *testing.T) {
		fixture := map[string]any{
			"id":     "resp_123",
			"object": "response",
			"status": "completed",
			"model":  "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type":      "web_search_call",
					"id":        "ws_123",
					"name":      "web_search",
					"arguments": "{}",
					"call_id":   "",
					"status":    "completed",
				},
				map[string]any{
					"type":      "code_interpreter_call",
					"id":        "code_123",
					"name":      "code_execution",
					"arguments": "{}",
					"call_id":   "",
					"status":    "completed",
				},
			},
			"usage": map[string]any{
				"input_tokens":  float64(10),
				"output_tokens": float64(5),
			},
		}

		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.web_search",
					Name: "web_search",
					Args: map[string]interface{}{},
				},
				languagemodel.ProviderTool{
					ID:   "xai.code_execution",
					Name: "code_execution",
					Args: map[string]interface{}{},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(result.Content))
		}
		if _, ok := result.Content[0].(languagemodel.ToolCall); !ok {
			t.Errorf("expected ToolCall for first content, got %T", result.Content[0])
		}
		if _, ok := result.Content[1].(languagemodel.ToolCall); !ok {
			t.Errorf("expected ToolCall for second content, got %T", result.Content[1])
		}
	})
}

func TestResponsesDoGenerate_ToolNameMapping(t *testing.T) {
	t.Run("should map web_search_call type to web_search", func(t *testing.T) {
		fixture := map[string]any{
			"id": "resp_123", "object": "response", "status": "completed",
			"model": "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type": "web_search_call", "id": "ws_123", "name": "",
					"arguments": `{"query":"test"}`, "call_id": "", "status": "completed",
				},
			},
			"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
		}
		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.web_search", Name: "web_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc := result.Content[0].(languagemodel.ToolCall)
		if tc.ToolName != "web_search" {
			t.Errorf("expected 'web_search', got %q", tc.ToolName)
		}
	})

	t.Run("should map x_search_call type to x_search", func(t *testing.T) {
		fixture := map[string]any{
			"id": "resp_123", "object": "response", "status": "completed",
			"model": "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type": "x_search_call", "id": "xs_123", "name": "",
					"arguments": `{"query":"test"}`, "call_id": "", "status": "completed",
				},
			},
			"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
		}
		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.x_search", Name: "x_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc := result.Content[0].(languagemodel.ToolCall)
		if tc.ToolName != "x_search" {
			t.Errorf("expected 'x_search', got %q", tc.ToolName)
		}
	})

	t.Run("should map code_interpreter_call type to code_execution", func(t *testing.T) {
		fixture := map[string]any{
			"id": "resp_123", "object": "response", "status": "completed",
			"model": "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type": "code_interpreter_call", "id": "ci_123", "name": "",
					"arguments": "{}", "call_id": "", "status": "completed",
				},
			},
			"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
		}
		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.code_execution", Name: "code_execution", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc := result.Content[0].(languagemodel.ToolCall)
		if tc.ToolName != "code_execution" {
			t.Errorf("expected 'code_execution', got %q", tc.ToolName)
		}
		if tc.ToolCallID != "ci_123" {
			t.Errorf("expected 'ci_123', got %q", tc.ToolCallID)
		}
	})

	t.Run("should map code_execution_call type to code_execution", func(t *testing.T) {
		fixture := map[string]any{
			"id": "resp_123", "object": "response", "status": "completed",
			"model": "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type": "code_execution_call", "id": "ce_123", "name": "",
					"arguments": "{}", "call_id": "", "status": "completed",
				},
			},
			"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
		}
		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.code_execution", Name: "code_execution", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc := result.Content[0].(languagemodel.ToolCall)
		if tc.ToolName != "code_execution" {
			t.Errorf("expected 'code_execution', got %q", tc.ToolName)
		}
		if tc.ToolCallID != "ce_123" {
			t.Errorf("expected 'ce_123', got %q", tc.ToolCallID)
		}
	})

	t.Run("should use custom tool name from provider tool when type matches", func(t *testing.T) {
		fixture := map[string]any{
			"id": "resp_123", "object": "response", "status": "completed",
			"model": "grok-4-fast-non-reasoning",
			"output": []any{
				map[string]any{
					"type": "web_search_call", "id": "ws_123", "name": "",
					"arguments": "{}", "call_id": "", "status": "completed",
				},
			},
			"usage": map[string]any{"input_tokens": float64(10), "output_tokens": float64(5)},
		}
		server, _ := createResponsesTestServer(fixture, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.web_search", Name: "my_custom_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tc := result.Content[0].(languagemodel.ToolCall)
		if tc.ToolName != "my_custom_search" {
			t.Errorf("expected 'my_custom_search', got %q", tc.ToolName)
		}
	})
}

// ---- DoStream Tests ----

func TestResponsesDoStream_TextStreaming(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"message","id":"msg_123","status":"in_progress","role":"assistant","content":[]},"output_index":0}`,
			`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"Hello"}`,
			`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":" world"}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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

		if len(textDeltas) != 2 {
			t.Fatalf("expected 2 text deltas, got %d", len(textDeltas))
		}
		if textDeltas[0] != "Hello" {
			t.Errorf("expected first delta 'Hello', got %q", textDeltas[0])
		}
		if textDeltas[1] != " world" {
			t.Errorf("expected second delta ' world', got %q", textDeltas[1])
		}
	})
}

func TestResponsesDoStream_EncryptedReasoning(t *testing.T) {
	t.Run("should include encrypted content in reasoning-end providerMetadata", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_456","status":"in_progress","summary":[]},"output_index":0}`,
			`{"type":"response.reasoning_summary_part.added","item_id":"rs_456","output_index":0,"summary_index":0,"part":{"type":"summary_text","text":""}}`,
			`{"type":"response.reasoning_summary_text.delta","item_id":"rs_456","output_index":0,"summary_index":0,"delta":"Analyzing..."}`,
			`{"type":"response.reasoning_summary_text.done","item_id":"rs_456","output_index":0,"summary_index":0,"text":"Analyzing..."}`,
			`{"type":"response.output_item.done","item":{"type":"reasoning","id":"rs_456","status":"completed","summary":[{"type":"summary_text","text":"Analyzing..."}],"encrypted_content":"encrypted_data_abc123"},"output_index":0}`,
			`{"type":"response.output_item.added","item":{"type":"message","id":"msg_789","role":"assistant","status":"in_progress","content":[]},"output_index":1}`,
			`{"type":"response.output_text.delta","item_id":"msg_789","output_index":1,"content_index":0,"delta":"Result."}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":20}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var reasoningEnd *languagemodel.StreamPartReasoningEnd
		for _, p := range parts {
			if re, ok := p.(languagemodel.StreamPartReasoningEnd); ok {
				reasoningEnd = &re
			}
		}

		if reasoningEnd == nil {
			t.Fatal("expected reasoning-end part")
		}
		if reasoningEnd.ID != "reasoning-rs_456" {
			t.Errorf("expected id 'reasoning-rs_456', got %q", reasoningEnd.ID)
		}
		xaiMeta := reasoningEnd.ProviderMetadata["xai"]
		if xaiMeta["reasoningEncryptedContent"] != "encrypted_data_abc123" {
			t.Errorf("expected encrypted content, got %v", xaiMeta["reasoningEncryptedContent"])
		}
		if xaiMeta["itemId"] != "rs_456" {
			t.Errorf("expected itemId 'rs_456', got %v", xaiMeta["itemId"])
		}
	})
}

func TestResponsesDoStream_ReasoningStartBeforeEnd(t *testing.T) {
	t.Run("should emit reasoning-start before reasoning-end when summary_part.added not sent", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_456","status":"in_progress","summary":[]},"output_index":0}`,
			// No reasoning_summary_part.added -- encrypted reasoning with store=false
			`{"type":"response.output_item.done","item":{"type":"reasoning","id":"rs_456","status":"completed","summary":[],"encrypted_content":"encrypted_reasoning_content_xyz"},"output_index":0}`,
			`{"type":"response.output_item.added","item":{"type":"message","id":"msg_789","role":"assistant","status":"in_progress","content":[]},"output_index":1}`,
			`{"type":"response.output_text.delta","item_id":"msg_789","output_index":1,"content_index":0,"delta":"The answer is 42."}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":20}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		reasoningStartIdx := -1
		reasoningEndIdx := -1
		for i, p := range parts {
			if rs, ok := p.(languagemodel.StreamPartReasoningStart); ok {
				reasoningStartIdx = i
				if rs.ID != "reasoning-rs_456" {
					t.Errorf("expected reasoning-start id 'reasoning-rs_456', got %q", rs.ID)
				}
			}
			if re, ok := p.(languagemodel.StreamPartReasoningEnd); ok {
				reasoningEndIdx = i
				xaiMeta := re.ProviderMetadata["xai"]
				if xaiMeta["reasoningEncryptedContent"] != "encrypted_reasoning_content_xyz" {
					t.Errorf("expected encrypted content in reasoning-end")
				}
			}
		}

		if reasoningStartIdx == -1 {
			t.Fatal("expected reasoning-start")
		}
		if reasoningEndIdx == -1 {
			t.Fatal("expected reasoning-end")
		}
		if reasoningStartIdx >= reasoningEndIdx {
			t.Error("expected reasoning-start before reasoning-end")
		}
	})
}

func TestResponsesDoStream_ReasoningTextDeltas(t *testing.T) {
	t.Run("should stream reasoning text deltas (response.reasoning_text.delta)", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-code-fast-1","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_456","status":"in_progress","summary":[]},"output_index":0}`,
			`{"type":"response.reasoning_text.delta","item_id":"rs_456","output_index":0,"content_index":0,"delta":"First"}`,
			`{"type":"response.reasoning_text.delta","item_id":"rs_456","output_index":0,"content_index":0,"delta":", analyze the question."}`,
			`{"type":"response.reasoning_text.done","item_id":"rs_456","output_index":0,"content_index":0,"text":"First, analyze the question."}`,
			`{"type":"response.output_item.done","item":{"type":"reasoning","id":"rs_456","status":"completed","summary":[{"type":"summary_text","text":"First, analyze the question."}]},"output_index":0}`,
			`{"type":"response.output_item.added","item":{"type":"message","id":"msg_789","role":"assistant","status":"in_progress","content":[]},"output_index":1}`,
			`{"type":"response.output_text.delta","item_id":"msg_789","output_index":1,"content_index":0,"delta":"The answer."}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","model":"grok-code-fast-1","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":20,"output_tokens_details":{"reasoning_tokens":15}}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL, "grok-code-fast-1")
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Check reasoning start
		var reasoningStart *languagemodel.StreamPartReasoningStart
		for _, p := range parts {
			if rs, ok := p.(languagemodel.StreamPartReasoningStart); ok {
				reasoningStart = &rs
				break
			}
		}
		if reasoningStart == nil {
			t.Fatal("expected reasoning-start")
		}
		if reasoningStart.ID != "reasoning-rs_456" {
			t.Errorf("expected id 'reasoning-rs_456', got %q", reasoningStart.ID)
		}

		// Check reasoning deltas
		var reasoningDeltas []languagemodel.StreamPartReasoningDelta
		for _, p := range parts {
			if rd, ok := p.(languagemodel.StreamPartReasoningDelta); ok {
				reasoningDeltas = append(reasoningDeltas, rd)
			}
		}
		if len(reasoningDeltas) != 2 {
			t.Fatalf("expected 2 reasoning deltas, got %d", len(reasoningDeltas))
		}
		if reasoningDeltas[0].Delta != "First" {
			t.Errorf("expected first delta 'First', got %q", reasoningDeltas[0].Delta)
		}
		if reasoningDeltas[1].Delta != ", analyze the question." {
			t.Errorf("expected second delta ', analyze the question.', got %q", reasoningDeltas[1].Delta)
		}

		// Check ordering: reasoning-start < reasoning-deltas < reasoning-end < text-delta
		startIdx, endIdx, textIdx := -1, -1, -1
		firstDeltaIdx := -1
		for i, p := range parts {
			if _, ok := p.(languagemodel.StreamPartReasoningStart); ok {
				startIdx = i
			}
			if _, ok := p.(languagemodel.StreamPartReasoningDelta); ok && firstDeltaIdx == -1 {
				firstDeltaIdx = i
			}
			if _, ok := p.(languagemodel.StreamPartReasoningEnd); ok {
				endIdx = i
			}
			if _, ok := p.(languagemodel.StreamPartTextDelta); ok && textIdx == -1 {
				textIdx = i
			}
		}
		if startIdx >= firstDeltaIdx {
			t.Error("expected reasoning-start before reasoning-delta")
		}
		if firstDeltaIdx >= endIdx {
			t.Error("expected reasoning-delta before reasoning-end")
		}
		if endIdx >= textIdx {
			t.Error("expected reasoning-end before text-delta")
		}
	})
}

func TestResponsesDoStream_NoDuplicateTextDelta(t *testing.T) {
	t.Run("should not emit duplicate text-delta from output_item.done", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"message","id":"msg_123","status":"in_progress","role":"assistant","content":[]},"output_index":0}`,
			`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"Hello"}`,
			`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":" "}`,
			`{"type":"response.output_text.delta","item_id":"msg_123","output_index":0,"content_index":0,"delta":"world"}`,
			`{"type":"response.output_item.done","item":{"type":"message","id":"msg_123","status":"completed","role":"assistant","content":[{"type":"output_text","text":"Hello world","annotations":[]}]},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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

		// Should only have 3 deltas from streaming, not 4 with duplicate full text
		if len(textDeltas) != 3 {
			t.Fatalf("expected 3 text deltas, got %d: %v", len(textDeltas), textDeltas)
		}

		expected := []string{"Hello", " ", "world"}
		for i, exp := range expected {
			if textDeltas[i] != exp {
				t.Errorf("expected delta[%d] %q, got %q", i, exp, textDeltas[i])
			}
		}

		// No delta with full accumulated text
		for _, d := range textDeltas {
			if d == "Hello world" {
				t.Error("should not have full text delta")
			}
		}
	})
}

func TestResponsesDoStream_WebSearchToolCall(t *testing.T) {
	t.Run("should stream web_search tool calls", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"web_search_call","id":"ws_123","name":"web_search","arguments":"{\"query\":\"test\"}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.web_search", Name: "web_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var toolCall *languagemodel.ToolCall
		for _, p := range parts {
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}
		if toolCall == nil {
			t.Fatal("expected tool-call")
		}
		if toolCall.ToolCallID != "ws_123" {
			t.Errorf("expected 'ws_123', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "web_search" {
			t.Errorf("expected 'web_search', got %q", toolCall.ToolName)
		}
		if toolCall.ProviderExecuted == nil || !*toolCall.ProviderExecuted {
			t.Error("expected providerExecuted true")
		}
	})
}

func TestResponsesDoStream_FunctionCallArguments(t *testing.T) {
	t.Run("should stream function tool call arguments", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","output_index":0,"item":{"type":"function_call","id":"fc_123","call_id":"call_123","name":"weather","arguments":"","status":"in_progress"}}`,
			`{"type":"response.function_call_arguments.delta","item_id":"fc_123","output_index":0,"delta":"{\"location\""}`,
			`{"type":"response.function_call_arguments.delta","item_id":"fc_123","output_index":0,"delta":":\"sf\"}"}`,
			`{"type":"response.function_call_arguments.done","item_id":"fc_123","output_index":0,"arguments":"{\"location\":\"sf\"}"}`,
			`{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","id":"fc_123","call_id":"call_123","name":"weather","arguments":"{\"location\":\"sf\"}","status":"completed"}}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		desc := "get weather"
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasToolStart, hasToolEnd bool
		var toolInputDeltas []string
		var toolCall *languagemodel.ToolCall
		for _, p := range parts {
			if ts, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				hasToolStart = true
				if ts.ToolName != "weather" {
					t.Errorf("expected tool name 'weather', got %q", ts.ToolName)
				}
				if ts.ID != "call_123" {
					t.Errorf("expected id 'call_123', got %q", ts.ID)
				}
			}
			if te, ok := p.(languagemodel.StreamPartToolInputEnd); ok {
				hasToolEnd = true
				if te.ID != "call_123" {
					t.Errorf("expected id 'call_123', got %q", te.ID)
				}
			}
			if td, ok := p.(languagemodel.StreamPartToolInputDelta); ok {
				toolInputDeltas = append(toolInputDeltas, td.Delta)
			}
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
		}

		if !hasToolStart {
			t.Error("expected tool-input-start")
		}
		if !hasToolEnd {
			t.Error("expected tool-input-end")
		}
		if len(toolInputDeltas) < 2 {
			t.Fatalf("expected at least 2 tool input deltas, got %d", len(toolInputDeltas))
		}
		if toolInputDeltas[0] != `{"location"` {
			t.Errorf("expected first delta, got %q", toolInputDeltas[0])
		}
		if toolInputDeltas[1] != `:"sf"}` {
			t.Errorf("expected second delta, got %q", toolInputDeltas[1])
		}
		if toolCall == nil {
			t.Fatal("expected tool-call")
		}
		if toolCall.ToolCallID != "call_123" {
			t.Errorf("expected 'call_123', got %q", toolCall.ToolCallID)
		}
		if toolCall.ToolName != "weather" {
			t.Errorf("expected 'weather', got %q", toolCall.ToolName)
		}
		if toolCall.Input != `{"location":"sf"}` {
			t.Errorf("expected input, got %q", toolCall.Input)
		}
	})
}

func TestResponsesDoStream_FileSearchToolCall(t *testing.T) {
	t.Run("should stream file_search tool call and result", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"file_search_call","id":"fs_stream_123","status":"in_progress","queries":["search query"],"results":null},"output_index":0}`,
			`{"type":"response.output_item.done","item":{"type":"file_search_call","id":"fs_stream_123","status":"completed","queries":["search query"],"results":[{"file_id":"file_abc","filename":"doc.txt","score":0.9,"text":"Found text content"}]},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{
					ID:   "xai.file_search",
					Name: "file_search",
					Args: map[string]interface{}{
						"vectorStoreIds": []interface{}{"collection_123"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasToolInputStart, hasToolInputEnd bool
		var toolCall *languagemodel.ToolCall
		var toolResult *languagemodel.ToolResult
		for _, p := range parts {
			if ts, ok := p.(languagemodel.StreamPartToolInputStart); ok {
				hasToolInputStart = true
				if ts.ToolName != "file_search" {
					t.Errorf("expected 'file_search', got %q", ts.ToolName)
				}
			}
			if _, ok := p.(languagemodel.StreamPartToolInputEnd); ok {
				hasToolInputEnd = true
			}
			if tc, ok := p.(languagemodel.ToolCall); ok {
				toolCall = &tc
			}
			if tr, ok := p.(languagemodel.ToolResult); ok {
				toolResult = &tr
			}
		}

		if !hasToolInputStart {
			t.Error("expected tool-input-start")
		}
		if !hasToolInputEnd {
			t.Error("expected tool-input-end")
		}
		if toolCall == nil {
			t.Fatal("expected tool-call")
		}
		if toolCall.ToolCallID != "fs_stream_123" {
			t.Errorf("expected 'fs_stream_123', got %q", toolCall.ToolCallID)
		}
		if toolCall.ProviderExecuted == nil || !*toolCall.ProviderExecuted {
			t.Error("expected providerExecuted true")
		}
		if toolResult == nil {
			t.Fatal("expected tool-result")
		}
		if toolResult.ToolCallID != "fs_stream_123" {
			t.Errorf("expected 'fs_stream_123', got %q", toolResult.ToolCallID)
		}
		resultMap := toolResult.Result.(map[string]interface{})
		results := resultMap["results"].([]map[string]interface{})
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0]["fileId"] != "file_abc" {
			t.Errorf("expected fileId 'file_abc', got %v", results[0]["fileId"])
		}
	})
}

func TestResponsesDoStream_StreamToolNameMapping(t *testing.T) {
	t.Run("should map web_search_call type to web_search in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"web_search_call","id":"ws_123","name":"","arguments":"{\"query\":\"test\"}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}
		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.web_search", Name: "web_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts := collectStreamParts(result.Stream)

		var tc *languagemodel.ToolCall
		for _, p := range parts {
			if c, ok := p.(languagemodel.ToolCall); ok {
				tc = &c
			}
		}
		if tc == nil || tc.ToolName != "web_search" {
			t.Errorf("expected web_search tool call")
		}
	})

	t.Run("should map x_search_call type to x_search in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"x_search_call","id":"xs_123","name":"","arguments":"{\"query\":\"test\"}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}
		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.x_search", Name: "x_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts := collectStreamParts(result.Stream)

		var tc *languagemodel.ToolCall
		for _, p := range parts {
			if c, ok := p.(languagemodel.ToolCall); ok {
				tc = &c
			}
		}
		if tc == nil || tc.ToolName != "x_search" {
			t.Errorf("expected x_search tool call")
		}
	})

	t.Run("should map code_interpreter_call type to code_execution in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"code_interpreter_call","id":"ci_123","name":"","arguments":"{}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}
		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.code_execution", Name: "code_execution", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts := collectStreamParts(result.Stream)

		var tc *languagemodel.ToolCall
		for _, p := range parts {
			if c, ok := p.(languagemodel.ToolCall); ok {
				tc = &c
			}
		}
		if tc == nil || tc.ToolName != "code_execution" {
			t.Errorf("expected code_execution tool call")
		}
	})

	t.Run("should map code_execution_call type to code_execution in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"code_execution_call","id":"ce_123","name":"","arguments":"{}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}
		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.code_execution", Name: "code_execution", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts := collectStreamParts(result.Stream)

		var tc *languagemodel.ToolCall
		for _, p := range parts {
			if c, ok := p.(languagemodel.ToolCall); ok {
				tc = &c
			}
		}
		if tc == nil || tc.ToolName != "code_execution" {
			t.Errorf("expected code_execution tool call")
		}
	})

	t.Run("should use custom tool name in stream", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_item.added","item":{"type":"web_search_call","id":"ws_123","name":"","arguments":"{}","call_id":"","status":"completed"},"output_index":0}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}
		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt, Ctx: context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.ProviderTool{ID: "xai.web_search", Name: "my_custom_search", Args: map[string]interface{}{}},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parts := collectStreamParts(result.Stream)

		var tc *languagemodel.ToolCall
		for _, p := range parts {
			if c, ok := p.(languagemodel.ToolCall); ok {
				tc = &c
			}
		}
		if tc == nil || tc.ToolName != "my_custom_search" {
			t.Errorf("expected 'my_custom_search' tool call")
		}
	})
}

func TestResponsesDoStream_CitationStreaming(t *testing.T) {
	t.Run("should stream citations as sources", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","status":"in_progress","output":[]}}`,
			`{"type":"response.output_text.annotation.added","item_id":"msg_123","output_index":0,"content_index":0,"annotation_index":0,"annotation":{"type":"url_citation","url":"https://example.com","title":"example"}}`,
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var source *languagemodel.SourceURL
		for _, p := range parts {
			if s, ok := p.(languagemodel.SourceURL); ok {
				source = &s
			}
		}
		if source == nil {
			t.Fatal("expected source")
		}
		if source.URL != "https://example.com" {
			t.Errorf("expected URL 'https://example.com', got %q", source.URL)
		}
		if source.Title == nil || *source.Title != "example" {
			t.Errorf("expected title 'example', got %v", source.Title)
		}
		if source.ID != "id-0" {
			t.Errorf("expected id 'id-0', got %q", source.ID)
		}
	})
}

func TestResponsesDoStream_MissingUsage(t *testing.T) {
	t.Run("should handle missing usage in streaming response", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"in_progress","output":[]}}`,
			`{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"Hello"}`,
			`{"type":"response.completed","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"completed","output":[{"type":"message","id":"msg_001","role":"assistant","status":"completed","content":[{"type":"output_text","text":"Hello"}]}]}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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
		if finishPart.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected stop finish reason, got %v", finishPart.FinishReason.Unified)
		}
		if intVal(finishPart.Usage.InputTokens.Total) != 0 {
			t.Errorf("expected InputTokens.Total 0, got %d", intVal(finishPart.Usage.InputTokens.Total))
		}
		if intVal(finishPart.Usage.OutputTokens.Total) != 0 {
			t.Errorf("expected OutputTokens.Total 0, got %d", intVal(finishPart.Usage.OutputTokens.Total))
		}
	})
}

func TestResponsesDoStream_SchemaValidation(t *testing.T) {
	t.Run("should accept response.created with usage: null", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"in_progress","output":[],"usage":null}}`,
			`{"type":"response.output_item.added","item":{"id":"msg_001","type":"message","role":"assistant","content":[],"status":"in_progress"},"output_index":0}`,
			`{"type":"response.output_text.delta","item_id":"msg_001","output_index":0,"content_index":0,"delta":"Hello"}`,
			`{"type":"response.completed","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"completed","output":[{"id":"msg_001","type":"message","role":"assistant","content":[{"type":"output_text","text":"Hello"}],"status":"completed"}],"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasTextDelta, hasFinish bool
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok && td.Delta == "Hello" {
				hasTextDelta = true
			}
			if _, ok := p.(languagemodel.StreamPartFinish); ok {
				hasFinish = true
			}
		}
		if !hasTextDelta {
			t.Error("expected text-delta with 'Hello'")
		}
		if !hasFinish {
			t.Error("expected finish part")
		}
	})

	t.Run("should accept response.in_progress with usage: null", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.created","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"in_progress","output":[],"usage":null}}`,
			`{"type":"response.in_progress","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"in_progress","output":[],"usage":null}}`,
			`{"type":"response.output_item.added","item":{"id":"msg_001","type":"message","role":"assistant","content":[],"status":"in_progress"},"output_index":0}`,
			`{"type":"response.output_text.delta","item_id":"msg_001","output_index":0,"content_index":0,"delta":"Hi"}`,
			`{"type":"response.completed","response":{"id":"resp_123","object":"response","model":"grok-4-fast-non-reasoning","created_at":1700000000,"status":"completed","output":[{"id":"msg_001","type":"message","role":"assistant","content":[{"type":"output_text","text":"Hi"}],"status":"completed"}],"usage":{"input_tokens":5,"output_tokens":1,"total_tokens":6}}}`,
		}

		server, _ := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		var hasTextDelta bool
		for _, p := range parts {
			if td, ok := p.(languagemodel.StreamPartTextDelta); ok && td.Delta == "Hi" {
				hasTextDelta = true
			}
		}
		if !hasTextDelta {
			t.Error("expected text-delta with 'Hi'")
		}
	})
}

func TestResponsesDoStream_RequestBody(t *testing.T) {
	t.Run("should include stream=true in request body", func(t *testing.T) {
		chunks := []string{
			`{"type":"response.done","response":{"id":"resp_123","object":"response","status":"completed","output":[],"usage":{"input_tokens":10,"output_tokens":5}}}`,
		}

		server, capture := createResponsesSSETestServer(chunks, nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_ = collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["stream"] != true {
			t.Errorf("expected stream true, got %v", body["stream"])
		}
	})
}

func TestResponsesDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createResponsesTestServer(responsesTextFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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

func TestResponsesDoGenerate_SupportedUrls(t *testing.T) {
	t.Run("should return supported URL patterns", func(t *testing.T) {
		model := createResponsesModel("http://unused")
		urls, err := model.SupportedUrls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		patterns, ok := urls["image/*"]
		if !ok || len(patterns) == 0 {
			t.Fatal("expected image/* patterns")
		}
		if !patterns[0].MatchString("https://example.com/image.jpg") {
			t.Error("expected URL pattern to match https URLs")
		}
	})
}

func TestResponsesDoGenerate_ModelIdAndProvider(t *testing.T) {
	t.Run("should return correct model ID", func(t *testing.T) {
		model := createResponsesModel("http://unused", "grok-4-fast-non-reasoning")
		if model.ModelID() != "grok-4-fast-non-reasoning" {
			t.Errorf("expected 'grok-4-fast-non-reasoning', got %q", model.ModelID())
		}
	})

	t.Run("should return correct provider", func(t *testing.T) {
		model := createResponsesModel("http://unused")
		if model.Provider() != "xai.responses" {
			t.Errorf("expected 'xai.responses', got %q", model.Provider())
		}
	})
}

func TestResponsesDoGenerate_Headers(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createResponsesTestServer(emptyResponsesFixture(), nil)
		defer server.Close()

		model := createResponsesModel(server.URL)
		customHeader := "custom-value"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: responsesTestPrompt,
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

// assertBodyContainsStr is a helper to verify a string in captured body.
func assertBodyContainsStr(t *testing.T, capture *requestCapture, key string) {
	t.Helper()
	if !strings.Contains(string(capture.Body), key) {
		t.Errorf("expected body to contain %q, body: %s", key, string(capture.Body))
	}
}
