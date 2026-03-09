// Ported from: packages/openai-compatible/src/embedding/openai-compatible-embedding-model.test.ts
package openaicompatible

import (
	"context"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Embedding test helpers ---

func createEmbeddingModel(baseURL string) *EmbeddingModel {
	return NewEmbeddingModel("test-embedding-model", EmbeddingConfig{
		Provider: "test-provider.embedding",
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

func embeddingFixture() map[string]any {
	return map[string]any{
		"data": []any{
			map[string]any{
				"embedding": []any{float64(0.1), float64(0.2), float64(0.3)},
			},
			map[string]any{
				"embedding": []any{float64(0.4), float64(0.5), float64(0.6)},
			},
		},
		"usage": map[string]any{
			"prompt_tokens": float64(8),
		},
	}
}

// --- DoEmbed tests ---

func TestEmbeddingDoEmbed_Embeddings(t *testing.T) {
	t.Run("should extract embeddings", func(t *testing.T) {
		server, _ := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello", "World"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Embeddings) != 2 {
			t.Fatalf("expected 2 embeddings, got %d", len(result.Embeddings))
		}
		if len(result.Embeddings[0]) != 3 {
			t.Fatalf("expected 3 dimensions, got %d", len(result.Embeddings[0]))
		}
		if result.Embeddings[0][0] != 0.1 {
			t.Errorf("expected first value 0.1, got %v", result.Embeddings[0][0])
		}
		if result.Embeddings[1][0] != 0.4 {
			t.Errorf("expected second embedding first value 0.4, got %v", result.Embeddings[1][0])
		}
	})
}

func TestEmbeddingDoEmbed_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createTestServer(embeddingFixture(), map[string]string{
			"X-Embed-Header": "embed-value",
		})
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers["X-Embed-Header"] != "embed-value" {
			t.Errorf("expected X-Embed-Header 'embed-value', got %q", result.Response.Headers["X-Embed-Header"])
		}
	})
}

func TestEmbeddingDoEmbed_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello", "World"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage == nil {
			t.Fatal("expected non-nil usage")
		}
		if result.Usage.Tokens != 8 {
			t.Errorf("expected 8 tokens, got %d", result.Usage.Tokens)
		}
	})
}

func TestEmbeddingDoEmbed_RequestBody(t *testing.T) {
	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello", "World"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "test-embedding-model" {
			t.Errorf("expected model 'test-embedding-model', got %v", body["model"])
		}
		if body["encoding_format"] != "float" {
			t.Errorf("expected encoding_format 'float', got %v", body["encoding_format"])
		}
		input, ok := body["input"].([]any)
		if !ok {
			t.Fatalf("expected input to be []any, got %T", body["input"])
		}
		if len(input) != 2 {
			t.Fatalf("expected 2 inputs, got %d", len(input))
		}
	})

	t.Run("should include dimensions when set", func(t *testing.T) {
		server, capture := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello"},
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"dimensions": float64(256),
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["dimensions"] != float64(256) {
			t.Errorf("expected dimensions 256, got %v", body["dimensions"])
		}
	})
}

func TestEmbeddingDoEmbed_DeprecatedKey(t *testing.T) {
	t.Run("should warn about deprecated openai-compatible key", func(t *testing.T) {
		server, _ := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello"},
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"openai-compatible": {
					"dimensions": float64(256),
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

func TestEmbeddingDoEmbed_Headers(t *testing.T) {
	t.Run("should pass headers to request", func(t *testing.T) {
		server, capture := createTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"Hello"},
			Ctx:    context.Background(),
			Headers: map[string]string{
				"X-Embed-Request": "request-value",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Embed-Request") != "request-value" {
			t.Errorf("expected X-Embed-Request header, got %q", capture.Headers.Get("X-Embed-Request"))
		}
	})
}

func TestEmbeddingModel_MaxEmbeddingsPerCall(t *testing.T) {
	t.Run("should return default max 2048", func(t *testing.T) {
		model := NewEmbeddingModel("test", EmbeddingConfig{
			Provider: "test.embedding",
		})

		max, err := model.MaxEmbeddingsPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 2048 {
			t.Errorf("expected max 2048, got %v", max)
		}
	})

	t.Run("should return custom max when set", func(t *testing.T) {
		customMax := 100
		model := NewEmbeddingModel("test", EmbeddingConfig{
			Provider:             "test.embedding",
			MaxEmbeddingsPerCall: &customMax,
		})

		max, err := model.MaxEmbeddingsPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 100 {
			t.Errorf("expected max 100, got %v", max)
		}
	})
}

func TestEmbeddingModel_SupportsParallelCalls(t *testing.T) {
	t.Run("should return true by default", func(t *testing.T) {
		model := NewEmbeddingModel("test", EmbeddingConfig{
			Provider: "test.embedding",
		})

		supports, err := model.SupportsParallelCalls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !supports {
			t.Error("expected supports parallel calls to be true")
		}
	})
}
