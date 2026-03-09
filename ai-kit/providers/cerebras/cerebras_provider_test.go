// Ported from: packages/cerebras/src/cerebras-provider.test.ts
package cerebras

import (
	"os"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/providers/openaicompatible"
)

func TestCreateCerebras(t *testing.T) {
	// Save and restore original VERSION so tests are deterministic.
	origVersion := VERSION
	VERSION = "0.0.0-test"
	defer func() { VERSION = origVersion }()

	t.Run("should create a CerebrasProvider instance with default options", func(t *testing.T) {
		// Set the env var so LoadApiKey succeeds inside getHeaders().
		t.Setenv("CEREBRAS_API_KEY", "test-env-key")

		provider := NewCerebrasProvider()

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.baseURL != "https://api.cerebras.ai/v1" {
			t.Errorf("expected default base URL, got %q", provider.baseURL)
		}

		// Verify headers can be generated (triggers LoadApiKey internally).
		headers := provider.getHeaders()
		// Headers are normalized to lowercase by providerutils.NormalizeHeaders.
		if headers["authorization"] != "Bearer test-env-key" {
			t.Errorf("expected authorization header with env key, got %q", headers["authorization"])
		}
	})

	t.Run("should create a CerebrasProvider instance with custom options", func(t *testing.T) {
		customKey := "custom-key"
		customURL := "https://custom.url"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey:  &customKey,
			BaseURL: &customURL,
			Headers: map[string]string{"Custom-Header": "value"},
		})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}
		if provider.baseURL != "https://custom.url" {
			t.Errorf("expected custom base URL, got %q", provider.baseURL)
		}

		headers := provider.getHeaders()
		if headers["authorization"] != "Bearer custom-key" {
			t.Errorf("expected authorization with custom key, got %q", headers["authorization"])
		}
		if headers["custom-header"] != "value" {
			t.Errorf("expected custom-header, got %q", headers["custom-header"])
		}
	})

	t.Run("should pass user-agent header with version", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

		headers := provider.getHeaders()
		ua := headers["user-agent"]
		expected := "ai-sdk/cerebras/0.0.0-test"
		if !strings.Contains(ua, expected) {
			t.Errorf("expected user-agent to contain %q, got %q", expected, ua)
		}
	})

	t.Run("should return a chat model when called as a function", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Chat("foo-model-id")
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		// Verify it's a *openaicompatible.ChatLanguageModel by type assertion.
		var _ *openaicompatible.ChatLanguageModel = model
	})
}

func TestLanguageModel(t *testing.T) {
	t.Run("should construct a language model with correct configuration", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

		model, err := provider.LanguageModel("foo-model-id")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		// Verify it satisfies the LanguageModel interface and has correct metadata.
		if model.ModelID() != "foo-model-id" {
			t.Errorf("expected model ID %q, got %q", "foo-model-id", model.ModelID())
		}
		if model.Provider() != "cerebras.chat" {
			t.Errorf("expected provider %q, got %q", "cerebras.chat", model.Provider())
		}
	})
}

func TestEmbeddingModel(t *testing.T) {
	t.Run("should return NoSuchModelError when attempting to create embedding model", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

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

		expectedMsg := "No such embeddingModel: any-model"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("expected error message to contain %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestChat(t *testing.T) {
	t.Run("should construct a chat model with correct configuration", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

		model := provider.Chat("foo-model-id")
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		// Verify it's a *openaicompatible.ChatLanguageModel.
		var _ *openaicompatible.ChatLanguageModel = model

		if model.ModelID() != "foo-model-id" {
			t.Errorf("expected model ID %q, got %q", "foo-model-id", model.ModelID())
		}
	})
}

func TestSpecificationVersion(t *testing.T) {
	t.Run("should return v3", func(t *testing.T) {
		apiKey := "mock-api-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &apiKey,
		})

		if v := provider.SpecificationVersion(); v != "v3" {
			t.Errorf("expected specification version %q, got %q", "v3", v)
		}
	})
}

func TestUnsupportedModels(t *testing.T) {
	apiKey := "mock-api-key"
	provider := NewCerebrasProvider(CerebrasProviderSettings{
		APIKey: &apiKey,
	})

	t.Run("ImageModel should return NoSuchModelError", func(t *testing.T) {
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

func TestBaseURLWithTrailingSlash(t *testing.T) {
	t.Run("should strip trailing slash from custom base URL", func(t *testing.T) {
		apiKey := "mock-api-key"
		url := "https://custom.url/v1/"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey:  &apiKey,
			BaseURL: &url,
		})
		// WithoutTrailingSlash should have stripped the trailing slash.
		if strings.HasSuffix(provider.baseURL, "/") {
			t.Errorf("expected base URL without trailing slash, got %q", provider.baseURL)
		}
	})
}

func TestLoadApiKeyFromEnvironment(t *testing.T) {
	t.Run("should load API key from CEREBRAS_API_KEY environment variable", func(t *testing.T) {
		t.Setenv("CEREBRAS_API_KEY", "env-api-key-123")
		provider := NewCerebrasProvider()

		headers := provider.getHeaders()
		if headers["authorization"] != "Bearer env-api-key-123" {
			t.Errorf("expected Authorization with env key, got %q", headers["authorization"])
		}
	})

	t.Run("should panic when no API key is provided and env var is unset", func(t *testing.T) {
		// Ensure the env var is unset.
		os.Unsetenv("CEREBRAS_API_KEY")
		t.Setenv("CEREBRAS_API_KEY", "")
		os.Unsetenv("CEREBRAS_API_KEY")

		provider := NewCerebrasProvider()

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic when no API key is available")
			}
		}()

		// Calling getHeaders triggers LoadApiKey which should panic.
		provider.getHeaders()
	})

	t.Run("should prefer explicit API key over environment variable", func(t *testing.T) {
		t.Setenv("CEREBRAS_API_KEY", "env-key")
		explicitKey := "explicit-key"
		provider := NewCerebrasProvider(CerebrasProviderSettings{
			APIKey: &explicitKey,
		})

		headers := provider.getHeaders()
		if headers["authorization"] != "Bearer explicit-key" {
			t.Errorf("expected explicit key to take precedence, got %q", headers["authorization"])
		}
	})
}
