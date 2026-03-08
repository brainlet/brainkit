// Ported from: packages/core/src/llm/model/router-custom-provider.test.ts
package model

import (
	"fmt"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock types for custom provider tests
// ---------------------------------------------------------------------------

// mockCustomProviderGateway implements MastraModelGateway for testing custom providers.
type mockCustomProviderGateway struct {
	id                     string
	name                   string
	resolveLanguageModelFn func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error)
}

func (g *mockCustomProviderGateway) ID() string   { return g.id }
func (g *mockCustomProviderGateway) Name() string { return g.name }
func (g *mockCustomProviderGateway) FetchProviders() (map[string]ProviderConfig, error) {
	return nil, nil
}
func (g *mockCustomProviderGateway) BuildURL(modelID string, envVars map[string]string) (string, error) {
	return "", nil
}
func (g *mockCustomProviderGateway) GetAPIKey(modelID string) (string, error) {
	return "", fmt.Errorf("Could not find config for provider %s", modelID)
}
func (g *mockCustomProviderGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	if g.resolveLanguageModelFn != nil {
		return g.resolveLanguageModelFn(args)
	}
	return &mockLanguageModelV2{
		specVersion: "v2",
		provider:    args.ProviderID,
		modelID:     args.ModelID,
	}, nil
}

func TestModelRouterCustomProviderSupport(t *testing.T) {
	t.Run("Mock verification", func(t *testing.T) {
		t.Run("should create ModelRouterLanguageModel with correct parameters", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "my-custom-provider",
					ModelID:    "my-model",
					URL:        "http://fake-test-server-that-does-not-exist.local:9999/v1",
					APIKey:     "test-key",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.Provider() != "my-custom-provider" {
				t.Errorf("provider = %q, want %q", model.Provider(), "my-custom-provider")
			}
			if model.ModelID() != "my-model" {
				t.Errorf("modelId = %q, want %q", model.ModelID(), "my-model")
			}
			if model.config.URL != "http://fake-test-server-that-does-not-exist.local:9999/v1" {
				t.Errorf("url = %q, want %q", model.config.URL, "http://fake-test-server-that-does-not-exist.local:9999/v1")
			}
			if model.config.APIKey != "test-key" {
				t.Errorf("apiKey = %q, want %q", model.config.APIKey, "test-key")
			}
		})
	})

	t.Run("Unknown provider with custom URL", func(t *testing.T) {
		t.Run("should allow unknown provider when URL is provided", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "ollama",
					ModelID:    "llama3.2",
					URL:        "http://localhost:11434/v1",
					APIKey:     "not-needed-for-ollama",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
		})

		t.Run("should allow unknown provider with id format when URL is provided", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:     "ollama/llama3.2",
					URL:    "http://localhost:11434/v1",
					APIKey: "not-needed-for-ollama",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
		})

		t.Run("should allow any custom provider name when URL is provided", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			customProviders := []string{"my-custom-provider", "local-llm", "custom-ai-service", "test-provider-123"}

			for _, providerID := range customProviders {
				model, err := NewModelRouterLanguageModel(
					OpenAICompatibleConfig{
						ProviderID: providerID,
						ModelID:    "test-model",
						URL:        "http://localhost:8080/v1",
						APIKey:     "test-key",
					},
					[]MastraModelGateway{gateway},
				)
				if err != nil {
					t.Fatalf("unexpected error for provider %q: %v", providerID, err)
				}
				if model == nil {
					t.Fatalf("expected non-nil model for provider %q", providerID)
				}
			}
		})

		t.Run("should work with LMStudio provider", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "lmstudio",
					ModelID:    "custom-model",
					URL:        "http://localhost:1234/v1",
					APIKey:     "not-needed",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
		})

		t.Run("should handle custom headers with unknown provider", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:     "custom-provider/custom-model",
					URL:    "http://localhost:8080/v1",
					APIKey: "test-key",
					Headers: map[string]string{
						"X-Custom-Header": "custom-value",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
			if model.config.Headers["X-Custom-Header"] != "custom-value" {
				t.Errorf("X-Custom-Header = %q, want %q", model.config.Headers["X-Custom-Header"], "custom-value")
			}
		})

		t.Run("should create model with custom URL that can be used for DoGenerate", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "ollama",
					ModelID:    "llama3.2",
					URL:        "http://fake-ollama-server.local:9999/v1",
					APIKey:     "not-needed-for-ollama",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if model == nil {
				t.Fatal("expected non-nil model")
			}
			// Verify the URL is stored on the config
			if model.config.URL != "http://fake-ollama-server.local:9999/v1" {
				t.Errorf("url = %q, want %q", model.config.URL, "http://fake-ollama-server.local:9999/v1")
			}
		})
	})

	t.Run("Unknown provider without custom URL", func(t *testing.T) {
		t.Run("should return error for unknown provider without URL during DoGenerate", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "unknown-provider",
					ModelID:    "unknown-model",
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error creating model: %v", err)
			}

			// Clear cache so we hit the gateway
			modelInstancesMu.Lock()
			modelInstances = make(map[string]GatewayLanguageModel)
			modelInstancesMu.Unlock()

			// The error should happen when trying to use the model (API key resolution fails)
			_, err = model.DoGenerate(LanguageModelV2CallOptions{})
			if err == nil {
				t.Fatal("expected error when using unknown provider without URL")
			}
		})

		t.Run("should return error for unknown provider in id format without URL during DoGenerate", func(t *testing.T) {
			gateway := &mockCustomProviderGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				"unknown-provider/unknown-model",
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error creating model: %v", err)
			}

			// Clear cache so we hit the gateway
			modelInstancesMu.Lock()
			modelInstances = make(map[string]GatewayLanguageModel)
			modelInstancesMu.Unlock()

			// The error should happen when trying to use the model (API key resolution fails)
			_, err = model.DoGenerate(LanguageModelV2CallOptions{})
			if err == nil {
				t.Fatal("expected error when using unknown provider without URL")
			}
		})
	})
}
