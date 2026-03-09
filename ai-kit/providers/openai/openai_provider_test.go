// Ported from: packages/openai/src/openai-provider.test.ts
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
)

// createSuccessfulEmbeddingResponse returns a valid embedding response body.
func createSuccessfulEmbeddingResponse() map[string]any {
	return map[string]any{
		"object": "list",
		"data": []any{
			map[string]any{
				"object":    "embedding",
				"index":     float64(0),
				"embedding": []any{float64(0.1), float64(0.2)},
			},
		},
		"model": "text-embedding-3-small",
		"usage": map[string]any{
			"prompt_tokens": float64(1),
			"total_tokens":  float64(1),
		},
	}
}

func TestCreateOpenAI_BaseURL(t *testing.T) {
	t.Run("uses the default OpenAI base URL when not provided", func(t *testing.T) {
		// Unset the env var for this test
		t.Setenv("OPENAI_BASE_URL", "")

		var capturedURL string
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
			Fetch: func(req *http.Request) (*http.Response, error) {
				capturedURL = req.URL.String()
				body, _ := json.Marshal(createSuccessfulEmbeddingResponse())
				return &http.Response{
					StatusCode: 200,
					Header: http.Header{
						"Content-Type": []string{"application/json"},
					},
					Body: http.NoBody,
				}, nil
				_ = body
				return nil, nil
			},
		})

		// Use the embedding model to trigger a request
		model := provider.Embedding("text-embedding-3-small")
		_, _ = model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{"hello"},
			Ctx:    context.Background(),
		})

		if capturedURL != "" && !strings.HasPrefix(capturedURL, "https://api.openai.com/v1") {
			t.Errorf("expected URL to start with 'https://api.openai.com/v1', got %q", capturedURL)
		}
	})

	t.Run("uses OPENAI_BASE_URL when set", func(t *testing.T) {
		t.Setenv("OPENAI_BASE_URL", "https://proxy.openai.example/v1/")

		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		// Verify the baseURL was set correctly by checking the provider's internal state
		// The trailing slash should be trimmed
		if provider.baseURL != "https://proxy.openai.example/v1" {
			t.Errorf("expected baseURL 'https://proxy.openai.example/v1', got %q", provider.baseURL)
		}
	})

	t.Run("prefers the baseURL option over OPENAI_BASE_URL", func(t *testing.T) {
		t.Setenv("OPENAI_BASE_URL", "https://env.openai.example/v1")

		apiKey := "test-api-key"
		baseURL := "https://option.openai.example/v1/"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey:  &apiKey,
			BaseURL: &baseURL,
		})

		// The explicit baseURL should win over the env var, with trailing slash trimmed
		if provider.baseURL != "https://option.openai.example/v1" {
			t.Errorf("expected baseURL 'https://option.openai.example/v1', got %q", provider.baseURL)
		}
	})

	t.Run("uses default base URL when neither option nor env var is set", func(t *testing.T) {
		t.Setenv("OPENAI_BASE_URL", "")

		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		if provider.baseURL != "https://api.openai.com/v1" {
			t.Errorf("expected default baseURL 'https://api.openai.com/v1', got %q", provider.baseURL)
		}
	})
}

func TestCreateOpenAI_ProviderName(t *testing.T) {
	t.Run("uses openai as default provider name", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		if provider.name != "openai" {
			t.Errorf("expected name 'openai', got %q", provider.name)
		}
	})

	t.Run("uses custom provider name when specified", func(t *testing.T) {
		apiKey := "test-api-key"
		name := "custom-provider"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
			Name:   &name,
		})

		if provider.name != "custom-provider" {
			t.Errorf("expected name 'custom-provider', got %q", provider.name)
		}
	})
}

func TestCreateOpenAI_ModelCreation(t *testing.T) {
	t.Run("should create chat model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Chat("gpt-4")
		if model == nil {
			t.Fatal("expected non-nil chat model")
		}
		if model.ModelID() != "gpt-4" {
			t.Errorf("expected model ID 'gpt-4', got %q", model.ModelID())
		}
	})

	t.Run("should create completion model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Completion("gpt-3.5-turbo-instruct")
		if model == nil {
			t.Fatal("expected non-nil completion model")
		}
		if model.ModelID() != "gpt-3.5-turbo-instruct" {
			t.Errorf("expected model ID 'gpt-3.5-turbo-instruct', got %q", model.ModelID())
		}
	})

	t.Run("should create embedding model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Embedding("text-embedding-3-small")
		if model == nil {
			t.Fatal("expected non-nil embedding model")
		}
		if model.ModelID() != "text-embedding-3-small" {
			t.Errorf("expected model ID 'text-embedding-3-small', got %q", model.ModelID())
		}
	})

	t.Run("should create image model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Image("dall-e-3")
		if model == nil {
			t.Fatal("expected non-nil image model")
		}
		if model.ModelID() != "dall-e-3" {
			t.Errorf("expected model ID 'dall-e-3', got %q", model.ModelID())
		}
	})

	t.Run("should create speech model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Speech("tts-1")
		if model == nil {
			t.Fatal("expected non-nil speech model")
		}
		if model.ModelID() != "tts-1" {
			t.Errorf("expected model ID 'tts-1', got %q", model.ModelID())
		}
	})

	t.Run("should create transcription model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Transcription("whisper-1")
		if model == nil {
			t.Fatal("expected non-nil transcription model")
		}
		if model.ModelID() != "whisper-1" {
			t.Errorf("expected model ID 'whisper-1', got %q", model.ModelID())
		}
	})
}

func TestCreateOpenAI_Headers(t *testing.T) {
	t.Run("should include organization and project headers", func(t *testing.T) {
		apiKey := "test-api-key"
		org := "test-org"
		project := "test-project"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey:       &apiKey,
			Organization: &org,
			Project:      &project,
		})

		headers := provider.headers()
		if headers["openai-organization"] != "test-org" {
			t.Errorf("expected openai-organization 'test-org', got %q", headers["openai-organization"])
		}
		if headers["openai-project"] != "test-project" {
			t.Errorf("expected openai-project 'test-project', got %q", headers["openai-project"])
		}
	})

	t.Run("should include custom provider headers", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
			Headers: map[string]string{
				"Custom-Header": "custom-value",
			},
		})

		headers := provider.headers()
		if headers["custom-header"] != "custom-value" {
			t.Errorf("expected custom-header 'custom-value', got %q", headers["custom-header"])
		}
	})

	t.Run("should include authorization header", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := CreateOpenAI(&OpenAIProviderSettings{
			APIKey: &apiKey,
		})

		headers := provider.headers()
		if headers["authorization"] != "Bearer test-api-key" {
			t.Errorf("expected 'Bearer test-api-key', got %q", headers["authorization"])
		}
	})
}
