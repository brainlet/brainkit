package togetherai

import (
	"os"
	"testing"
)

func TestNewProvider_DefaultOptions(t *testing.T) {
	// Ensure env vars are clean
	origKey := os.Getenv("TOGETHER_API_KEY")
	origOldKey := os.Getenv("TOGETHER_AI_API_KEY")
	os.Setenv("TOGETHER_API_KEY", "test-env-key")
	os.Unsetenv("TOGETHER_AI_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("TOGETHER_API_KEY", origKey)
		} else {
			os.Unsetenv("TOGETHER_API_KEY")
		}
		if origOldKey != "" {
			os.Setenv("TOGETHER_AI_API_KEY", origOldKey)
		} else {
			os.Unsetenv("TOGETHER_AI_API_KEY")
		}
	}()

	provider := NewProvider(ProviderSettings{})

	if provider.baseURL != "https://api.together.xyz/v1" {
		t.Errorf("expected default base URL 'https://api.together.xyz/v1', got %q", provider.baseURL)
	}

	headers := provider.getHeaders()
	// Headers are normalized to lowercase by WithUserAgentSuffix/NormalizeHeaders
	if headers["authorization"] != "Bearer test-env-key" {
		t.Errorf("expected authorization header 'Bearer test-env-key', got %q", headers["authorization"])
	}
}

func TestNewProvider_CustomOptions(t *testing.T) {
	apiKey := "custom-key"
	baseURL := "https://custom.url"

	provider := NewProvider(ProviderSettings{
		APIKey:  &apiKey,
		BaseURL: &baseURL,
		Headers: map[string]string{"Custom-Header": "value"},
	})

	if provider.baseURL != "https://custom.url" {
		t.Errorf("expected base URL 'https://custom.url', got %q", provider.baseURL)
	}

	headers := provider.getHeaders()
	if headers["authorization"] != "Bearer custom-key" {
		t.Errorf("expected authorization 'Bearer custom-key', got %q", headers["authorization"])
	}
	if headers["custom-header"] != "value" {
		t.Errorf("expected custom-header 'value', got %q", headers["custom-header"])
	}
}

func TestNewProvider_FallbackToDeprecatedEnvVar(t *testing.T) {
	origKey := os.Getenv("TOGETHER_API_KEY")
	origOldKey := os.Getenv("TOGETHER_AI_API_KEY")
	os.Unsetenv("TOGETHER_API_KEY")
	os.Setenv("TOGETHER_AI_API_KEY", "old-key")
	defer func() {
		if origKey != "" {
			os.Setenv("TOGETHER_API_KEY", origKey)
		} else {
			os.Unsetenv("TOGETHER_API_KEY")
		}
		if origOldKey != "" {
			os.Setenv("TOGETHER_AI_API_KEY", origOldKey)
		} else {
			os.Unsetenv("TOGETHER_AI_API_KEY")
		}
	}()

	provider := NewProvider(ProviderSettings{})

	headers := provider.getHeaders()
	if headers["authorization"] != "Bearer old-key" {
		t.Errorf("expected authorization 'Bearer old-key', got %q", headers["authorization"])
	}
}

func TestNewProvider_PreferPrimaryOverDeprecatedEnvVar(t *testing.T) {
	origKey := os.Getenv("TOGETHER_API_KEY")
	origOldKey := os.Getenv("TOGETHER_AI_API_KEY")
	os.Setenv("TOGETHER_API_KEY", "new-key")
	os.Setenv("TOGETHER_AI_API_KEY", "old-key")
	defer func() {
		if origKey != "" {
			os.Setenv("TOGETHER_API_KEY", origKey)
		} else {
			os.Unsetenv("TOGETHER_API_KEY")
		}
		if origOldKey != "" {
			os.Setenv("TOGETHER_AI_API_KEY", origOldKey)
		} else {
			os.Unsetenv("TOGETHER_AI_API_KEY")
		}
	}()

	provider := NewProvider(ProviderSettings{})

	headers := provider.getHeaders()
	if headers["authorization"] != "Bearer new-key" {
		t.Errorf("expected authorization 'Bearer new-key', got %q", headers["authorization"])
	}
}

func TestNewProvider_PreferExplicitKeyOverDeprecatedEnvVar(t *testing.T) {
	origOldKey := os.Getenv("TOGETHER_AI_API_KEY")
	os.Setenv("TOGETHER_AI_API_KEY", "old-key")
	defer func() {
		if origOldKey != "" {
			os.Setenv("TOGETHER_AI_API_KEY", origOldKey)
		} else {
			os.Unsetenv("TOGETHER_AI_API_KEY")
		}
	}()

	apiKey := "explicit-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	headers := provider.getHeaders()
	if headers["authorization"] != "Bearer explicit-key" {
		t.Errorf("expected authorization 'Bearer explicit-key', got %q", headers["authorization"])
	}
}

func TestProvider_ChatModel(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	model := provider.ChatModel("together-chat-model")
	if model == nil {
		t.Fatal("expected non-nil chat model")
	}
}

func TestProvider_CompletionModel(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	model := provider.CompletionModel("together-completion-model")
	if model == nil {
		t.Fatal("expected non-nil completion model")
	}
}

func TestProvider_EmbeddingModel(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	model, err := provider.EmbeddingModel("together-embedding-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model == nil {
		t.Fatal("expected non-nil embedding model")
	}
}

func TestProvider_ImageModel(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	model := provider.Image("stabilityai/stable-diffusion-xl")
	if model == nil {
		t.Fatal("expected non-nil image model")
	}
	if model.Provider() != "togetherai.image" {
		t.Errorf("expected provider 'togetherai.image', got %q", model.Provider())
	}
}

func TestProvider_ImageModel_CustomBaseURL(t *testing.T) {
	apiKey := "test-key"
	baseURL := "https://custom.url"
	provider := NewProvider(ProviderSettings{
		APIKey:  &apiKey,
		BaseURL: &baseURL,
	})

	model := provider.Image("stabilityai/stable-diffusion-xl")
	if model == nil {
		t.Fatal("expected non-nil image model")
	}
	if model.config.BaseURL != "https://custom.url" {
		t.Errorf("expected base URL 'https://custom.url', got %q", model.config.BaseURL)
	}
}

func TestProvider_RerankingModel(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	model := provider.Reranking("Salesforce/Llama-Rank-v1")
	if model == nil {
		t.Fatal("expected non-nil reranking model")
	}
	if model.config.BaseURL != "https://api.together.xyz/v1" {
		t.Errorf("expected base URL 'https://api.together.xyz/v1', got %q", model.config.BaseURL)
	}
}

func TestProvider_CallAsFunction(t *testing.T) {
	apiKey := "test-key"
	provider := NewProvider(ProviderSettings{APIKey: &apiKey})

	// In TS, provider('model-id') returns a chat model.
	// In Go, the equivalent is provider.ChatModel('model-id').
	model := provider.ChatModel("foo-model-id")
	if model == nil {
		t.Fatal("expected non-nil model from ChatModel")
	}
}

func TestProvider_BaseURL_TrailingSlashRemoved(t *testing.T) {
	apiKey := "test-key"
	baseURL := "https://custom.url/"
	provider := NewProvider(ProviderSettings{
		APIKey:  &apiKey,
		BaseURL: &baseURL,
	})

	if provider.baseURL != "https://custom.url" {
		t.Errorf("expected trailing slash removed, got %q", provider.baseURL)
	}
}
