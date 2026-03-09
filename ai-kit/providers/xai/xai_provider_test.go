// Ported from: packages/xai/src/xai-provider.test.ts
package xai

import (
	"testing"
)

func TestCreateXai_DefaultOptions(t *testing.T) {
	t.Run("should create provider with default options", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.baseURL != "https://api.x.ai/v1" {
			t.Errorf("expected baseURL 'https://api.x.ai/v1', got %q", provider.baseURL)
		}
	})
}

func TestCreateXai_CustomBaseURL(t *testing.T) {
	t.Run("should use custom base URL without trailing slash", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey:  "test-key",
			BaseURL: "https://custom.api.com/v2/",
		})

		if provider.baseURL != "https://custom.api.com/v2" {
			t.Errorf("expected baseURL 'https://custom.api.com/v2', got %q", provider.baseURL)
		}
	})
}

func TestCreateXai_ChatModel(t *testing.T) {
	t.Run("should create a chat model with correct provider", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		model := provider.Chat("grok-4-fast-non-reasoning")
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "grok-4-fast-non-reasoning" {
			t.Errorf("expected model ID 'grok-4-fast-non-reasoning', got %q", model.ModelID())
		}
		if model.Provider() != "xai.chat" {
			t.Errorf("expected provider 'xai.chat', got %q", model.Provider())
		}
	})
}

func TestCreateXai_ResponsesModel(t *testing.T) {
	t.Run("should create a responses model with correct provider", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		model := provider.Responses("grok-4-fast-non-reasoning")
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "grok-4-fast-non-reasoning" {
			t.Errorf("expected model ID 'grok-4-fast-non-reasoning', got %q", model.ModelID())
		}
		if model.Provider() != "xai.responses" {
			t.Errorf("expected provider 'xai.responses', got %q", model.Provider())
		}
	})
}

func TestCreateXai_ImageModel(t *testing.T) {
	t.Run("should create an image model with correct provider", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		model := provider.Image("grok-2-image")
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "grok-2-image" {
			t.Errorf("expected model ID 'grok-2-image', got %q", model.ModelID())
		}
		if model.Provider() != "xai.image" {
			t.Errorf("expected provider 'xai.image', got %q", model.Provider())
		}
	})
}

func TestCreateXai_VideoModel(t *testing.T) {
	t.Run("should create a video model with correct provider", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		model := provider.Video("grok-2-video")
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "grok-2-video" {
			t.Errorf("expected model ID 'grok-2-video', got %q", model.ModelID())
		}
		if model.Provider() != "xai.video" {
			t.Errorf("expected provider 'xai.video', got %q", model.Provider())
		}
	})
}

func TestCreateXai_Headers(t *testing.T) {
	t.Run("should include custom headers in header function", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
			Headers: map[string]string{
				"X-Custom": "custom-value",
			},
		})

		headers := provider.getHeaders()
		if headers["x-custom"] != "custom-value" {
			t.Errorf("expected x-custom 'custom-value', got %q", headers["x-custom"])
		}
		if headers["authorization"] != "Bearer test-key" {
			t.Errorf("expected authorization 'Bearer test-key', got %q", headers["authorization"])
		}
	})
}

func TestCreateXai_LanguageModel(t *testing.T) {
	t.Run("should create a language model via LanguageModel method", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		model, err := provider.LanguageModel("grok-4-fast-non-reasoning")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model.ModelID() != "grok-4-fast-non-reasoning" {
			t.Errorf("expected model ID 'grok-4-fast-non-reasoning', got %q", model.ModelID())
		}
	})
}

func TestCreateXai_EmbeddingModel(t *testing.T) {
	t.Run("should return NoSuchModelError for embedding models", func(t *testing.T) {
		provider := CreateXai(XaiProviderSettings{
			APIKey: "test-key",
		})

		_, err := provider.EmbeddingModel("test-embedding")
		if err == nil {
			t.Fatal("expected error for embedding model")
		}
	})
}
