// Ported from: packages/google/src/google-provider.test.ts
package google

import (
	"strings"
	"testing"
)

func TestGoogleProvider(t *testing.T) {
	t.Run("should create a language model with default settings", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		lm, err := provider.LanguageModel("gemini-pro")
		if err != nil {
			t.Fatal(err)
		}
		if lm == nil {
			t.Fatal("expected non-nil language model")
		}
		if lm.ModelID() != "gemini-pro" {
			t.Errorf("expected modelID 'gemini-pro', got %q", lm.ModelID())
		}
		if lm.Provider() != "google.generative-ai" {
			t.Errorf("expected provider 'google.generative-ai', got %q", lm.Provider())
		}
	})

	t.Run("should create an embedding model with correct settings", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		em, err := provider.EmbeddingModel("embedding-001")
		if err != nil {
			t.Fatal(err)
		}
		if em == nil {
			t.Fatal("expected non-nil embedding model")
		}
		if em.ModelID() != "embedding-001" {
			t.Errorf("expected modelID 'embedding-001', got %q", em.ModelID())
		}
		if em.Provider() != "google.generative-ai" {
			t.Errorf("expected provider 'google.generative-ai', got %q", em.Provider())
		}
	})

	t.Run("should pass custom headers to the model constructor", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
			Headers: map[string]string{
				"Custom-Header": "custom-value",
			},
		})
		model := provider.ChatModel("gemini-pro")
		headers := model.config.Headers()
		if headers["x-goog-api-key"] != "test-api-key" {
			t.Errorf("expected x-goog-api-key header")
		}
		if headers["custom-header"] != "custom-value" {
			t.Errorf("expected custom-header header, got headers: %v", headers)
		}
	})

	t.Run("should pass custom generateId function to the model constructor", func(t *testing.T) {
		apiKey := "test-api-key"
		customGenerateID := func() string { return "custom-id" }
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey:     &apiKey,
			GenerateID: customGenerateID,
		})
		model := provider.ChatModel("gemini-pro")
		id := model.generateID()
		if id != "custom-id" {
			t.Errorf("expected 'custom-id', got %q", id)
		}
	})

	t.Run("should use chat method to create a model", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		model := provider.ChatModel("gemini-pro")
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "gemini-pro" {
			t.Errorf("expected modelID 'gemini-pro', got %q", model.ModelID())
		}
	})

	t.Run("should use custom baseURL when provided", func(t *testing.T) {
		apiKey := "test-api-key"
		customBaseURL := "https://custom-endpoint.example.com"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey:  &apiKey,
			BaseURL: customBaseURL,
		})
		model := provider.ChatModel("gemini-pro")
		if model.config.BaseURL != customBaseURL {
			t.Errorf("expected baseURL %q, got %q", customBaseURL, model.config.BaseURL)
		}
	})

	t.Run("should create an image model with default settings", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		im, err := provider.ImageModel("imagen-3.0-generate-002")
		if err != nil {
			t.Fatal(err)
		}
		if im == nil {
			t.Fatal("expected non-nil image model")
		}
		if im.ModelID() != "imagen-3.0-generate-002" {
			t.Errorf("expected modelID 'imagen-3.0-generate-002', got %q", im.ModelID())
		}
	})

	t.Run("should support deprecated methods", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})

		lm := provider.GenerativeAI("gemini-pro")
		if lm == nil {
			t.Error("expected non-nil from GenerativeAI")
		}

		em := provider.Embedding("embedding-001")
		if em == nil {
			t.Error("expected non-nil from Embedding")
		}

		em2 := provider.TextEmbedding("embedding-001")
		if em2 == nil {
			t.Error("expected non-nil from TextEmbedding")
		}
	})

	t.Run("should include YouTube URLs in supportedUrls", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		model := provider.ChatModel("gemini-pro")

		supportedUrls, err := model.SupportedUrls()
		if err != nil {
			t.Fatal(err)
		}

		patterns, ok := supportedUrls["*"]
		if !ok {
			t.Fatal("expected '*' key in supportedUrls")
		}

		supportedURLs := []string{
			"https://generativelanguage.googleapis.com/v1beta/files/test123",
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			"https://youtube.com/watch?v=dQw4w9WgXcQ",
			"https://youtu.be/dQw4w9WgXcQ",
		}
		unsupportedURLs := []string{
			"https://example.com",
			"https://vimeo.com/123456789",
			"https://youtube.com/channel/UCdQw4w9WgXcQ",
		}

		for _, u := range supportedURLs {
			matched := false
			for _, p := range patterns {
				if p.MatchString(u) {
					matched = true
					break
				}
			}
			if !matched {
				t.Errorf("expected %q to be supported", u)
			}
		}
		for _, u := range unsupportedURLs {
			matched := false
			for _, p := range patterns {
				if p.MatchString(u) {
					matched = true
					break
				}
			}
			if matched {
				t.Errorf("expected %q to NOT be supported", u)
			}
		}
	})
}

func TestGoogleProviderCustomName(t *testing.T) {
	t.Run("should use custom provider name when specified", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
			Name:   "my-gemini-proxy",
		})
		model := provider.ChatModel("gemini-pro")
		if model.Provider() != "my-gemini-proxy" {
			t.Errorf("expected provider 'my-gemini-proxy', got %q", model.Provider())
		}
	})

	t.Run("should default to google.generative-ai when name not specified", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		model := provider.ChatModel("gemini-pro")
		if model.Provider() != "google.generative-ai" {
			t.Errorf("expected provider 'google.generative-ai', got %q", model.Provider())
		}
	})
}

func TestGoogleProviderVideo(t *testing.T) {
	t.Run("should create a video model with default settings", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		vm, err := provider.VideoModel("veo-3.1-generate-preview")
		if err != nil {
			t.Fatal(err)
		}
		if vm == nil {
			t.Fatal("expected non-nil video model")
		}
		if vm.ModelID() != "veo-3.1-generate-preview" {
			t.Errorf("expected modelID 'veo-3.1-generate-preview', got %q", vm.ModelID())
		}
		if vm.Provider() != "google.generative-ai" {
			t.Errorf("expected provider 'google.generative-ai', got %q", vm.Provider())
		}
	})

	t.Run("should use custom baseURL for video model when provided", func(t *testing.T) {
		apiKey := "test-api-key"
		customBaseURL := "https://custom-endpoint.example.com"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey:  &apiKey,
			BaseURL: customBaseURL,
		})
		vm := provider.Video("veo-3.1-generate")
		if vm.config.BaseURL != customBaseURL {
			t.Errorf("expected baseURL %q, got %q", customBaseURL, vm.config.BaseURL)
		}
	})

	t.Run("should pass custom generateId to video model", func(t *testing.T) {
		apiKey := "test-api-key"
		customGenerateID := func() string { return "custom-video-id" }
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey:     &apiKey,
			GenerateID: customGenerateID,
		})
		vm := provider.Video("veo-3.1-generate-preview")
		if vm.config.GenerateID() != "custom-video-id" {
			t.Errorf("expected 'custom-video-id'")
		}
	})

	t.Run("should have user agent suffix", func(t *testing.T) {
		apiKey := "test-api-key"
		provider := NewGoogleProvider(GoogleProviderSettings{
			APIKey: &apiKey,
		})
		model := provider.ChatModel("gemini-pro")
		headers := model.config.Headers()
		// Check for user-agent header containing version info
		hasUserAgent := false
		for k, v := range headers {
			if strings.EqualFold(k, "user-agent") && strings.Contains(v, "ai-sdk/google/") {
				hasUserAgent = true
				break
			}
		}
		if !hasUserAgent {
			t.Errorf("expected user-agent header with ai-sdk/google/ prefix, got headers: %v", headers)
		}
	})
}
