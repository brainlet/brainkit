// Ported from: packages/openai-compatible/src/completion/openai-compatible-completion-language-model.test.ts
package openaicompatible

import (
	"context"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Completion test helpers ---

func createCompletionModel(baseURL string) *CompletionLanguageModel {
	return NewCompletionLanguageModel("test-completion-model", CompletionConfig{
		Provider: "test-provider.completion",
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

func completionTextFixture() map[string]any {
	return map[string]any{
		"id":      "cmpl-test-id",
		"model":   "test-completion-model",
		"created": float64(1700000000),
		"choices": []any{
			map[string]any{
				"text":          "Hello, World!",
				"finish_reason": "stop",
				"index":         float64(0),
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(5),
			"completion_tokens": float64(10),
		},
	}
}

// --- Config tests ---

func TestCompletionLanguageModel_Config(t *testing.T) {
	t.Run("should extract provider options name", func(t *testing.T) {
		model := NewCompletionLanguageModel("test", CompletionConfig{
			Provider: "openai.completion",
		})
		name := model.providerOptionsName()
		if name != "openai" {
			t.Errorf("expected 'openai', got %q", name)
		}
	})
}

// --- DoGenerate tests ---

func TestCompletionDoGenerate_Text(t *testing.T) {
	t.Run("should extract text content", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

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
}

func TestCompletionDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage.InputTokens.Total == nil || *result.Usage.InputTokens.Total != 5 {
			t.Errorf("expected input total 5, got %v", result.Usage.InputTokens.Total)
		}
		if result.Usage.OutputTokens.Total == nil || *result.Usage.OutputTokens.Total != 10 {
			t.Errorf("expected output total 10, got %v", result.Usage.OutputTokens.Total)
		}
	})
}

func TestCompletionDoGenerate_FinishReason(t *testing.T) {
	t.Run("should extract finish reason", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), nil)
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
			t.Errorf("expected FinishReasonStop, got %v", result.FinishReason.Unified)
		}
	})
}

func TestCompletionDoGenerate_RequestBody(t *testing.T) {
	t.Run("should send correct request body with completion prompt", func(t *testing.T) {
		server, capture := createTestServer(completionTextFixture(), nil)
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
		if body["model"] != "test-completion-model" {
			t.Errorf("expected model 'test-completion-model', got %v", body["model"])
		}
		// Prompt should be a string (converted from chat format)
		prompt, ok := body["prompt"].(string)
		if !ok {
			t.Fatalf("expected prompt to be string, got %T", body["prompt"])
		}
		if !strings.Contains(prompt, "Hello") {
			t.Errorf("expected prompt to contain 'Hello', got %q", prompt)
		}
	})

	t.Run("should include temperature", func(t *testing.T) {
		server, capture := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)
		temp := 0.7

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt:      testPrompt,
			Ctx:         context.Background(),
			Temperature: &temp,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["temperature"] != 0.7 {
			t.Errorf("expected temperature 0.7, got %v", body["temperature"])
		}
	})
}

func TestCompletionDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), map[string]string{
			"X-Custom-Header": "custom-value",
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

		if result.Response.Headers["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected X-Custom-Header 'custom-value', got %q", result.Response.Headers["X-Custom-Header"])
		}
	})
}

func TestCompletionDoGenerate_Headers(t *testing.T) {
	t.Run("should pass request headers", func(t *testing.T) {
		server, capture := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Custom-Request-Header") != "request-value" {
			t.Errorf("expected Custom-Request-Header, got %q", capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestCompletionDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should pass through provider options", func(t *testing.T) {
		server, capture := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		_, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"suffix": "-end",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["suffix"] != "-end" {
			t.Errorf("expected suffix '-end', got %v", body["suffix"])
		}
	})
}

func TestCompletionDoGenerate_Warnings(t *testing.T) {
	t.Run("should warn about unsupported topK", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)
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
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for topK")
		}
	})

	t.Run("should warn about unsupported tools", func(t *testing.T) {
		server, _ := createTestServer(completionTextFixture(), nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoGenerate(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Tools: []languagemodel.Tool{
				languagemodel.FunctionTool{Name: "test"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "tools" {
				hasWarning = true
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for tools")
		}
	})
}

// --- DoStream tests ---

func TestCompletionDoStream_Text(t *testing.T) {
	t.Run("should stream text deltas", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"cmpl-stream-1","model":"test-completion-model","created":1700000000,"choices":[{"text":"Hello","index":0}]}` + "\n\n",
			`data: {"id":"cmpl-stream-1","model":"test-completion-model","created":1700000000,"choices":[{"text":", World!","index":0}]}` + "\n\n",
			`data: {"id":"cmpl-stream-1","model":"test-completion-model","created":1700000000,"choices":[{"text":"","index":0,"finish_reason":"stop"}]}` + "\n\n",
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
	})
}

func TestCompletionDoStream_RequestBody(t *testing.T) {
	t.Run("should set stream=true in request body", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"cmpl-stream","model":"test-completion-model","created":1700000000,"choices":[{"text":"Hi","index":0,"finish_reason":"stop"}]}` + "\n\n",
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
			t.Errorf("expected stream=true, got %v", body["stream"])
		}
	})
}

func TestCompletionDoStream_FinishReason(t *testing.T) {
	t.Run("should extract finish reason from stream", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"cmpl-stream","model":"test-completion-model","created":1700000000,"choices":[{"text":"Hi","index":0,"finish_reason":"stop"}]}` + "\n\n",
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

func TestCompletionDoStream_Headers(t *testing.T) {
	t.Run("should pass headers in streaming request", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"cmpl-stream","model":"test-completion-model","created":1700000000,"choices":[{"text":"Hi","index":0,"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, capture := createSSETestServer(chunks, nil)
		defer server.Close()
		model := createCompletionModel(server.URL)

		result, err := model.DoStream(languagemodel.CallOptions{
			Prompt: testPrompt,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Stream-Header": strPtr("stream-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		collectStreamParts(result.Stream)

		if capture.Headers.Get("X-Stream-Header") != "stream-value" {
			t.Errorf("expected X-Stream-Header, got %q", capture.Headers.Get("X-Stream-Header"))
		}
	})
}

func TestCompletionDoStream_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers from stream", func(t *testing.T) {
		chunks := []string{
			`data: {"id":"cmpl-stream","model":"test-completion-model","created":1700000000,"choices":[{"text":"Hi","index":0,"finish_reason":"stop"}]}` + "\n\n",
			"data: [DONE]\n\n",
		}

		server, _ := createSSETestServer(chunks, map[string]string{
			"X-Response-Header": "response-value",
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

		if result.Response.Headers["X-Response-Header"] != "response-value" {
			t.Errorf("expected X-Response-Header 'response-value', got %q", result.Response.Headers["X-Response-Header"])
		}
	})
}
