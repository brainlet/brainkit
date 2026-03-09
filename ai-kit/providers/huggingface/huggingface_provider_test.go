// Ported from: packages/huggingface/src/huggingface-provider.test.ts
package huggingface

import (
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
)

func TestCreateHuggingFaceProvider(t *testing.T) {
	t.Run("should create provider with default configuration", func(t *testing.T) {
		t.Setenv("HUGGINGFACE_API_KEY", "test-key")
		provider := NewProvider(ProviderSettings{})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}

		// Verify Responses method is available.
		model := provider.Responses("test-model")
		if model == nil {
			t.Fatal("expected non-nil model from Responses()")
		}

		// Verify LanguageModel method is available.
		lm, err := provider.LanguageModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lm == nil {
			t.Fatal("expected non-nil model from LanguageModel()")
		}
	})

	t.Run("should create provider with custom settings", func(t *testing.T) {
		apiKey := "custom-key"
		baseURL := "https://custom.url"
		provider := NewProvider(ProviderSettings{
			APIKey:  &apiKey,
			BaseURL: &baseURL,
			Headers: map[string]string{"Custom-Header": "test"},
		})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}

		// Verify Responses method is available.
		model := provider.Responses("test-model")
		if model == nil {
			t.Fatal("expected non-nil model from Responses()")
		}

		// Verify LanguageModel method is available.
		lm, err := provider.LanguageModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lm == nil {
			t.Fatal("expected non-nil model from LanguageModel()")
		}
	})
}

func TestModelCreationMethods(t *testing.T) {
	t.Run("should expose responses method", func(t *testing.T) {
		t.Setenv("HUGGINGFACE_API_KEY", "test-key")
		provider := NewProvider(ProviderSettings{})

		model := provider.Responses("test-model")
		if model == nil {
			t.Fatal("expected non-nil model from Responses()")
		}
	})

	t.Run("should expose languageModel method", func(t *testing.T) {
		t.Setenv("HUGGINGFACE_API_KEY", "test-key")
		provider := NewProvider(ProviderSettings{})

		model, err := provider.LanguageModel("test-model")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model from LanguageModel()")
		}
	})
}

func TestUnsupportedFunctionality(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{
		APIKey: &apiKey,
	})

	t.Run("should return error for text embedding models", func(t *testing.T) {
		model, err := provider.EmbeddingModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}

		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}

		expectedMsg := "Hugging Face Responses API does not support text embeddings"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("expected error message to contain %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("should return error for image models", func(t *testing.T) {
		model, err := provider.ImageModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}

		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}

		expectedMsg := "Hugging Face Responses API does not support image generation"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("expected error message to contain %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("TranscriptionModel should return NoSuchModelError", func(t *testing.T) {
		model, err := provider.TranscriptionModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}
	})

	t.Run("SpeechModel should return NoSuchModelError", func(t *testing.T) {
		model, err := provider.SpeechModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}
	})

	t.Run("RerankingModel should return NoSuchModelError", func(t *testing.T) {
		model, err := provider.RerankingModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}
		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}
	})
}

func TestSpecificationVersion(t *testing.T) {
	t.Run("should return v3", func(t *testing.T) {
		apiKey := "test-key"
		provider := NewProvider(ProviderSettings{
			APIKey: &apiKey,
		})
		if v := provider.SpecificationVersion(); v != "v3" {
			t.Errorf("expected specification version %q, got %q", "v3", v)
		}
	})
}
