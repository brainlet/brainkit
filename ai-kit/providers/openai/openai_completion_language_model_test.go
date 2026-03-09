// Ported from: packages/openai/src/completion/openai-completion-language-model.test.ts
package openai

import (
	"context"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func createCompletionModel(baseURL string) *OpenAICompletionLanguageModel {
	return NewOpenAICompletionLanguageModel("gpt-3.5-turbo-instruct", OpenAICompletionConfig{
		Provider: "openai.completion",
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

func completionTextFixture() map[string]any {
	return map[string]any{
		"id":      "cmpl-96cAM1v77r4jXa4qb2NSmRREV5oWB",
		"object":  "text_completion",
		"created": float64(1711363706),
		"model":   "gpt-3.5-turbo-instruct",
		"choices": []any{
			map[string]any{
				"text":          "Hello, World!",
				"index":         float64(0),
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

func TestCompletionDoGenerate_TextResponse(t *testing.T) {
	t.Run("should extract text response", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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
			t.Fatalf("expected Text, got %T", result.Content[0])
		}
		if text.Text != "Hello, World!" {
			t.Errorf("expected 'Hello, World!', got %q", text.Text)
		}
	})
}

func TestCompletionDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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
}

func TestCompletionDoGenerate_FinishReason(t *testing.T) {
	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != languagemodel.FinishReasonStop {
			t.Errorf("expected stop, got %v", result.FinishReason.Unified)
		}
	})
}

func TestCompletionDoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass model and prompt", func(t *testing.T) {
		server, capture := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "gpt-3.5-turbo-instruct" {
			t.Errorf("expected model 'gpt-3.5-turbo-instruct', got %v", body["model"])
		}
		if body["prompt"] == nil {
			t.Error("expected prompt to be set")
		}
	})

	t.Run("should pass temperature", func(t *testing.T) {
		server, capture := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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
		if body["temperature"] != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", body["temperature"])
		}
	})

	t.Run("should pass max_tokens", func(t *testing.T) {
		server, capture := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		maxTokens := 200
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:          testPrompt,
			MaxOutputTokens: &maxTokens,
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["max_tokens"] != float64(200) {
			t.Errorf("expected max_tokens 200, got %v", body["max_tokens"])
		}
	})
}

func TestCompletionDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), map[string]string{
			"X-Completion-Header": "completion-value",
		})
		defer server.Close()
		model := createCompletionModel(server.URL)

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
		if result.Response.Headers["X-Completion-Header"] != "completion-value" {
			t.Errorf("expected X-Completion-Header, got %q", result.Response.Headers["X-Completion-Header"])
		}
	})
}

func TestCompletionDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ID == nil || *result.Response.ID != "cmpl-96cAM1v77r4jXa4qb2NSmRREV5oWB" {
			t.Errorf("unexpected response ID: %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "gpt-3.5-turbo-instruct" {
			t.Errorf("unexpected model ID: %v", result.Response.ModelID)
		}
	})
}

func TestCompletionDoGenerate_Logprobs(t *testing.T) {
	t.Run("should extract logprobs", func(t *testing.T) {
		fixture := completionTextFixture()
		choices := fixture["choices"].([]any)
		choices[0].(map[string]any)["logprobs"] = map[string]any{
			"tokens":         []any{" ever", " after"},
			"token_logprobs": []any{-0.0664508, -0.014520033},
			"top_logprobs": []any{
				map[string]any{" ever": -0.0664508},
				map[string]any{" after": -0.014520033},
			},
		}

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, ok := result.ProviderMetadata["openai"]["logprobs"]
		if !ok {
			t.Error("expected logprobs in provider metadata")
		}
	})
}

func TestCompletionDoGenerate_Warnings(t *testing.T) {
	t.Run("should warn about unsupported topK", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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

	t.Run("should warn about unsupported tools", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		desc := "test"
		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{
					Name:        "test",
					Description: &desc,
					InputSchema: map[string]any{},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "tools" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for tools")
		}
	})
}

func TestCompletionDoGenerate_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createJSONTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		headerVal := "custom-val"
		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Headers: map[string]*string{
				"X-Custom": &headerVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Custom") != "custom-val" {
			t.Errorf("expected X-Custom 'custom-val', got %q", capture.Headers.Get("X-Custom"))
		}
	})
}

// --- DoStream tests ---

func TestCompletionDoStream_TextStreaming(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"object\":\"text_completion\",\"created\":1711363706,\"model\":\"gpt-3.5-turbo-instruct\",\"choices\":[{\"text\":\"Hello\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"object\":\"text_completion\",\"created\":1711363706,\"model\":\"gpt-3.5-turbo-instruct\",\"choices\":[{\"text\":\", World!\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"object\":\"text_completion\",\"created\":1711363706,\"model\":\"gpt-3.5-turbo-instruct\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":6,\"total_tokens\":10}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

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
	})
}

func TestCompletionDoStream_PassModelAndPrompt(t *testing.T) {
	t.Run("should pass model and prompt in stream request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"Hi\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		body := capture.BodyJSON()
		if body["model"] != "gpt-3.5-turbo-instruct" {
			t.Errorf("expected model 'gpt-3.5-turbo-instruct', got %v", body["model"])
		}
		if body["prompt"] == nil {
			t.Error("expected prompt to be present in request body")
		}
	})
}

func TestCompletionDoStream_RequestBody(t *testing.T) {
	t.Run("should include stream in request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"Hi\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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
			t.Errorf("expected stream true, got %v", body["stream"])
		}
	})
}

func TestCompletionDoGenerate_UnknownFinishReason(t *testing.T) {
	t.Run("should support unknown finish reason", func(t *testing.T) {
		fixture := completionTextFixture()
		choices := fixture["choices"].([]any)
		choice := choices[0].(map[string]any)
		choice["finish_reason"] = "eos"

		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.FinishReason.Unified != "other" && result.FinishReason.Unified != "unknown" {
			t.Errorf("expected unified finish reason 'other' or 'unknown', got %q", result.FinishReason.Unified)
		}
		raw := "eos"
		if result.FinishReason.Raw == nil || *result.FinishReason.Raw != raw {
			t.Errorf("expected raw finish reason 'eos', got %v", result.FinishReason.Raw)
		}
	})
}

func TestCompletionDoStream_ErrorHandling(t *testing.T) {
	t.Run("should handle error stream parts", func(t *testing.T) {
		chunks := []string{
			"data: {\"error\":{\"message\":\"The server had an error processing your request.\",\"type\":\"server_error\",\"param\":null,\"code\":null}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Verify we got an error part
		var errorFound bool
		for _, p := range parts {
			if _, ok := p.(languagemodel.StreamPartError); ok {
				errorFound = true
			}
		}
		if !errorFound {
			t.Error("expected an error stream part")
		}

		// Verify finish reason is error
		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.FinishReason.Unified != "error" {
					t.Errorf("expected unified finish reason 'error', got %q", f.FinishReason.Unified)
				}
			}
		}
	})
}

func TestCompletionDoGenerate_ResponseInfo(t *testing.T) {
	t.Run("should send additional response information", func(t *testing.T) {
		server, _ := createJSONTestServer(completionTextFixture(), map[string]string{
			"x-request-id": "req-123",
		})
		defer server.Close()
		model := createCompletionModel(server.URL)

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
		if result.Response.ID == nil || *result.Response.ID != "cmpl-96cAM1v77r4jXa4qb2NSmRREV5oWB" {
			t.Errorf("expected response ID 'cmpl-96cAM1v77r4jXa4qb2NSmRREV5oWB', got %v", result.Response.ID)
		}
		if result.Response.ModelID == nil || *result.Response.ModelID != "gpt-3.5-turbo-instruct" {
			t.Errorf("expected model ID 'gpt-3.5-turbo-instruct', got %v", result.Response.ModelID)
		}
	})
}

func TestCompletionDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers from stream", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"Hi\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, map[string]string{
			"x-custom-header": "custom-value",
		})
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers == nil {
			t.Fatal("expected non-nil response headers")
		}
		if result.Response.Headers["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected custom header value, got %q", result.Response.Headers["X-Custom-Header"])
		}
	})
}

func TestCompletionDoStream_Usage(t *testing.T) {
	t.Run("should extract usage from stream", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"Hello\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":6,\"total_tokens\":10}}\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		parts := collectStreamParts(result.Stream)

		// Find the finish part and check usage
		for _, p := range parts {
			if f, ok := p.(languagemodel.StreamPartFinish); ok {
				if f.Usage.InputTokens.Total == nil || *f.Usage.InputTokens.Total != 4 {
					t.Errorf("expected input tokens 4, got %v", f.Usage.InputTokens.Total)
				}
				if f.Usage.OutputTokens.Total == nil || *f.Usage.OutputTokens.Total != 6 {
					t.Errorf("expected output tokens 6, got %v", f.Usage.OutputTokens.Total)
				}
			}
		}
	})
}

func TestCompletionDoStream_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers in stream request", func(t *testing.T) {
		chunks := []string{
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"Hi\",\"index\":0,\"finish_reason\":null}]}\n\n",
			"data: {\"id\":\"cmpl-123\",\"choices\":[{\"text\":\"\",\"index\":0,\"finish_reason\":\"stop\"}]}\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()

		model := NewOpenAICompletionLanguageModel("gpt-3.5-turbo-instruct", OpenAICompletionConfig{
			Provider: "openai.completion",
			URL: func(options struct {
				ModelID string
				Path    string
			}) string {
				return server.URL + options.Path
			},
			Headers: func() map[string]string {
				return map[string]string{
					"Authorization":  "Bearer test-api-key",
					"Content-Type":   "application/json",
					"X-Custom-Test":  "test-value",
				}
			},
		})

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		if capture.Headers.Get("X-Custom-Test") != "test-value" {
			t.Errorf("expected custom header 'test-value', got %q", capture.Headers.Get("X-Custom-Test"))
		}
	})
}
