// Ported from: packages/openai-compatible/src/openai-compatible-provider.test.ts
package openaicompatible

import (
	"strings"
	"testing"
)

func TestNewProvider(t *testing.T) {
	t.Run("should create provider with default settings", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
		})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.baseURL != "https://api.example.com/v1" {
			t.Errorf("expected base URL 'https://api.example.com/v1', got %q", provider.baseURL)
		}
	})

	t.Run("should strip trailing slash from base URL", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1/",
			Name:    "test-provider",
		})

		if strings.HasSuffix(provider.baseURL, "/") {
			t.Errorf("expected base URL without trailing slash, got %q", provider.baseURL)
		}
	})

	t.Run("should create chat model", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-key",
		})

		model := provider.ChatModel("test-model")
		if model == nil {
			t.Fatal("expected non-nil chat model")
		}
		if model.ModelID() != "test-model" {
			t.Errorf("expected model ID 'test-model', got %q", model.ModelID())
		}
		if model.Provider() != "test-provider.chat" {
			t.Errorf("expected provider 'test-provider.chat', got %q", model.Provider())
		}
	})

	t.Run("should create completion model", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-key",
		})

		model := provider.CompletionModel("test-model")
		if model == nil {
			t.Fatal("expected non-nil completion model")
		}
		if model.ModelID() != "test-model" {
			t.Errorf("expected model ID 'test-model', got %q", model.ModelID())
		}
		if model.Provider() != "test-provider.completion" {
			t.Errorf("expected provider 'test-provider.completion', got %q", model.Provider())
		}
	})

	t.Run("should create embedding model", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-key",
		})

		model, err := provider.EmbeddingModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil embedding model")
		}
		if model.ModelID() != "test-model" {
			t.Errorf("expected model ID 'test-model', got %q", model.ModelID())
		}
	})

	t.Run("should create image model", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-key",
		})

		model, err := provider.ImageModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil image model")
		}
		if model.ModelID() != "test-model" {
			t.Errorf("expected model ID 'test-model', got %q", model.ModelID())
		}
	})

	t.Run("should return language model through LanguageModel method", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-key",
		})

		model, err := provider.LanguageModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil language model")
		}
		if model.ModelID() != "test-model" {
			t.Errorf("expected model ID 'test-model', got %q", model.ModelID())
		}
	})

	t.Run("should set authorization header from API key", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			APIKey:  "test-api-key",
		})

		headers := provider.headers()
		if headers["authorization"] != "Bearer test-api-key" {
			t.Errorf("expected Authorization header, got %q", headers["authorization"])
		}
	})

	t.Run("should include custom headers", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			Headers: map[string]string{
				"X-Custom-Header": "custom-value",
			},
		})

		headers := provider.headers()
		if headers["x-custom-header"] != "custom-value" {
			t.Errorf("expected custom header, got %q", headers["x-custom-header"])
		}
	})

	t.Run("should include user-agent header with version", func(t *testing.T) {
		origVersion := VERSION
		VERSION = "0.0.0-test"
		defer func() { VERSION = origVersion }()

		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
		})

		headers := provider.headers()
		ua := headers["user-agent"]
		if !strings.Contains(ua, "ai-sdk/openai-compatible/0.0.0-test") {
			t.Errorf("expected user-agent to contain version, got %q", ua)
		}
	})

	t.Run("should pass query params in URL", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
			QueryParams: map[string]string{
				"api-version": "2024-01-01",
			},
		})

		cfg := provider.getCommonModelConfig("chat")
		url := cfg.URL("/chat/completions")
		if !strings.Contains(url, "api-version=2024-01-01") {
			t.Errorf("expected URL to contain query param, got %q", url)
		}
	})

	t.Run("should propagate includeUsage setting to chat model", func(t *testing.T) {
		includeUsage := true
		provider := NewProvider(ProviderSettings{
			BaseURL:      "https://api.example.com/v1",
			Name:         "test-provider",
			IncludeUsage: &includeUsage,
		})

		model := provider.ChatModel("test-model")
		if model.config.IncludeUsage == nil || *model.config.IncludeUsage != true {
			t.Error("expected IncludeUsage to be true on chat model")
		}
	})

	t.Run("should propagate includeUsage false to chat model", func(t *testing.T) {
		includeUsage := false
		provider := NewProvider(ProviderSettings{
			BaseURL:      "https://api.example.com/v1",
			Name:         "test-provider",
			IncludeUsage: &includeUsage,
		})

		model := provider.ChatModel("test-model")
		if model.config.IncludeUsage == nil || *model.config.IncludeUsage != false {
			t.Error("expected IncludeUsage to be false on chat model")
		}
	})

	t.Run("should propagate includeUsage nil to chat model", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
		})

		model := provider.ChatModel("test-model")
		if model.config.IncludeUsage != nil {
			t.Error("expected IncludeUsage to be nil on chat model")
		}
	})

	t.Run("should propagate supportsStructuredOutputs to chat model", func(t *testing.T) {
		supports := true
		provider := NewProvider(ProviderSettings{
			BaseURL:                   "https://api.example.com/v1",
			Name:                      "test-provider",
			SupportsStructuredOutputs: &supports,
		})

		model := provider.ChatModel("test-model")
		if !model.supportsStructured {
			t.Error("expected supportsStructured to be true on chat model")
		}
	})

	t.Run("should return specification version v3", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
		})

		if v := provider.SpecificationVersion(); v != "v3" {
			t.Errorf("expected specification version 'v3', got %q", v)
		}
	})

	t.Run("unsupported model types", func(t *testing.T) {
		provider := NewProvider(ProviderSettings{
			BaseURL: "https://api.example.com/v1",
			Name:    "test-provider",
		})

		t.Run("TranscriptionModel should return error", func(t *testing.T) {
			_, err := provider.TranscriptionModel("any-model")
			if err == nil {
				t.Fatal("expected error")
			}
		})

		t.Run("SpeechModel should return error", func(t *testing.T) {
			_, err := provider.SpeechModel("any-model")
			if err == nil {
				t.Fatal("expected error")
			}
		})

		t.Run("RerankingModel should return error", func(t *testing.T) {
			_, err := provider.RerankingModel("any-model")
			if err == nil {
				t.Fatal("expected error")
			}
		})
	})
}
