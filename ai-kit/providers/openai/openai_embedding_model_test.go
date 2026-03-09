// Ported from: packages/openai/src/embedding/openai-embedding-model.test.ts
package openai

import (
	"context"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func createEmbeddingTestModel(baseURL string) *OpenAIEmbeddingModel {
	return NewOpenAIEmbeddingModel("text-embedding-3-large", OpenAIConfig{
		Provider: "openai.embedding",
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

func embeddingFixture() map[string]any {
	return map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{
				"object":    "embedding",
				"index":     float64(0),
				"embedding": []any{float64(0.0057293195), float64(-0.012727811), float64(0.020042092), float64(-0.013437585), float64(0.022833068)},
			},
			map[string]any{
				"object":    "embedding",
				"index":     float64(1),
				"embedding": []any{float64(-0.037104916), float64(-0.05178114), float64(-0.008340587), float64(0.001164541), float64(-0.0035253682)},
			},
		},
		"model": "text-embedding-3-large",
		"usage": map[string]any{
			"prompt_tokens": float64(12),
			"total_tokens":  float64(12),
		},
	}
}

func TestEmbeddingDoEmbed_Embeddings(t *testing.T) {
	t.Run("should extract embeddings", func(t *testing.T) {
		server, _ := createJSONTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"sunny day at the beach", "rainy day in the city"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Embeddings) != 2 {
			t.Fatalf("expected 2 embeddings, got %d", len(result.Embeddings))
		}
		if len(result.Embeddings[0]) != 5 {
			t.Fatalf("expected 5 dimensions, got %d", len(result.Embeddings[0]))
		}
		if result.Embeddings[0][0] != 0.0057293195 {
			t.Errorf("expected first value 0.0057293195, got %v", result.Embeddings[0][0])
		}
		if result.Embeddings[1][0] != -0.037104916 {
			t.Errorf("expected second embedding first value -0.037104916, got %v", result.Embeddings[1][0])
		}
	})
}

func TestEmbeddingDoEmbed_ResponseHeaders(t *testing.T) {
	t.Run("should expose the raw response headers", func(t *testing.T) {
		server, _ := createJSONTestServer(embeddingFixture(), map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"test"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response == nil {
			t.Fatal("expected non-nil response")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected Test-Header 'test-value', got %q", result.Response.Headers["Test-Header"])
		}
	})
}

func TestEmbeddingDoEmbed_Usage(t *testing.T) {
	t.Run("should extract usage", func(t *testing.T) {
		server, _ := createJSONTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"sunny day at the beach", "rainy day in the city"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage == nil {
			t.Fatal("expected non-nil usage")
		}
		if result.Usage.Tokens != 12 {
			t.Errorf("expected 12 tokens, got %d", result.Usage.Tokens)
		}
	})
}

func TestEmbeddingDoEmbed_RequestBody(t *testing.T) {
	t.Run("should pass model and input", func(t *testing.T) {
		server, capture := createJSONTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"sunny day at the beach", "rainy day in the city"},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "text-embedding-3-large" {
			t.Errorf("expected model 'text-embedding-3-large', got %v", body["model"])
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

	t.Run("should pass dimensions setting", func(t *testing.T) {
		server, capture := createJSONTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"test"},
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"dimensions": float64(64),
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["dimensions"] != float64(64) {
			t.Errorf("expected dimensions 64, got %v", body["dimensions"])
		}
	})
}

func TestEmbeddingDoEmbed_Headers(t *testing.T) {
	t.Run("should pass request headers", func(t *testing.T) {
		server, capture := createJSONTestServer(embeddingFixture(), nil)
		defer server.Close()
		model := createEmbeddingTestModel(server.URL)

		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"test"},
			Headers: map[string]string{
				"Custom-Embed-Header": "embed-value",
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Custom-Embed-Header") != "embed-value" {
			t.Errorf("expected Custom-Embed-Header, got %q", capture.Headers.Get("Custom-Embed-Header"))
		}
	})
}

func TestEmbeddingModel_MaxEmbeddingsPerCall(t *testing.T) {
	t.Run("should return 2048", func(t *testing.T) {
		model := NewOpenAIEmbeddingModel("test", OpenAIConfig{
			Provider: "openai.embedding",
		})

		max, err := model.MaxEmbeddingsPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 2048 {
			t.Errorf("expected 2048, got %v", max)
		}
	})
}

func TestEmbeddingModel_SupportsParallelCalls(t *testing.T) {
	t.Run("should return true", func(t *testing.T) {
		model := NewOpenAIEmbeddingModel("test", OpenAIConfig{
			Provider: "openai.embedding",
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
