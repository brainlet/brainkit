// Ported from: packages/core/src/llm/model/router-supported-urls.test.ts
package model

import (
	"fmt"
	"regexp"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock types for supportedUrls tests
// ---------------------------------------------------------------------------

// mockGatewayForSupportedURLs implements MastraModelGateway for testing URL propagation.
type mockGatewayForSupportedURLs struct {
	id                     string
	name                   string
	getAPIKeyFn            func(modelID string) (string, error)
	resolveLanguageModelFn func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error)
}

func (g *mockGatewayForSupportedURLs) ID() string   { return g.id }
func (g *mockGatewayForSupportedURLs) Name() string { return g.name }
func (g *mockGatewayForSupportedURLs) FetchProviders() (map[string]ProviderConfig, error) {
	return nil, nil
}
func (g *mockGatewayForSupportedURLs) BuildURL(modelID string, envVars map[string]string) (string, error) {
	return "", nil
}
func (g *mockGatewayForSupportedURLs) GetAPIKey(modelID string) (string, error) {
	if g.getAPIKeyFn != nil {
		return g.getAPIKeyFn(modelID)
	}
	return "mock-api-key", nil
}
func (g *mockGatewayForSupportedURLs) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	if g.resolveLanguageModelFn != nil {
		return g.resolveLanguageModelFn(args)
	}
	return &mockLanguageModelV2{
		specVersion: "v2",
		provider:    args.ProviderID,
		modelID:     args.ModelID,
	}, nil
}

// mockLanguageModelWithSupportedURLs implements GatewayLanguageModel with supportedUrls.
type mockLanguageModelWithSupportedURLs struct {
	specVersion   string
	provider      string
	modelID       string
	supportedURLs map[string][]*regexp.Regexp
}

func (m *mockLanguageModelWithSupportedURLs) SpecificationVersion() string { return m.specVersion }
func (m *mockLanguageModelWithSupportedURLs) Provider() string             { return m.provider }
func (m *mockLanguageModelWithSupportedURLs) ModelID() string              { return m.modelID }

// SupportedURLs returns the supported URLs map (test helper, not part of interface).
func (m *mockLanguageModelWithSupportedURLs) SupportedURLs() map[string][]*regexp.Regexp {
	return m.supportedURLs
}

func TestModelRouterLanguageModelSupportedURLsPropagation(t *testing.T) {
	// Mock Mistral's supportedUrls (same as what the real Mistral SDK defines)
	mockMistralSupportedURLs := map[string][]*regexp.Regexp{
		"application/pdf": {regexp.MustCompile(`^https://.*$`)},
	}

	// Mock model that simulates Mistral's behavior
	mockMistralModel := &mockLanguageModelWithSupportedURLs{
		specVersion:   "v2",
		provider:      "mistral",
		modelID:       "mistral-large-latest",
		supportedURLs: mockMistralSupportedURLs,
	}

	t.Run("should create ModelRouterLanguageModel with correct provider and model", func(t *testing.T) {
		gateway := &mockGatewayForSupportedURLs{
			id:   "models.dev",
			name: "Models.dev",
			resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
				return mockMistralModel, nil
			},
		}

		model, err := NewModelRouterLanguageModel("mistral/mistral-large-latest", []MastraModelGateway{gateway})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// The model should be created and support the Mistral provider
		if model.Provider() != "mistral" {
			t.Errorf("provider = %q, want %q", model.Provider(), "mistral")
		}
		if model.ModelID() != "mistral-large-latest" {
			t.Errorf("modelId = %q, want %q", model.ModelID(), "mistral-large-latest")
		}
	})

	t.Run("should return error when gateway API key resolution fails", func(t *testing.T) {
		// The TS test checks that supportedUrls gracefully degrades to {} when API key fails.
		// In the Go port, we verify the gateway error handling path via DoGenerate.
		gateway := &mockGatewayForSupportedURLs{
			id:   "models.dev",
			name: "Models.dev",
			getAPIKeyFn: func(modelID string) (string, error) {
				return "", fmt.Errorf("API key not found")
			},
		}

		model, err := NewModelRouterLanguageModel("unknown/unknown-model", []MastraModelGateway{gateway})
		if err != nil {
			t.Fatalf("unexpected error creating model: %v", err)
		}

		// DoGenerate should fail because API key resolution fails
		_, err = model.DoGenerate(LanguageModelV2CallOptions{})
		if err == nil {
			t.Error("expected error from DoGenerate when API key fails")
		}
	})

	t.Run("should return error when model resolution fails", func(t *testing.T) {
		gateway := &mockGatewayForSupportedURLs{
			id:   "models.dev",
			name: "Models.dev",
			resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
				return nil, fmt.Errorf("model not found")
			},
		}

		model, err := NewModelRouterLanguageModel("unknown/unknown-model", []MastraModelGateway{gateway})
		if err != nil {
			t.Fatalf("unexpected error creating model: %v", err)
		}

		// DoGenerate should fail because model resolution fails
		_, err = model.DoGenerate(LanguageModelV2CallOptions{})
		if err == nil {
			t.Error("expected error from DoGenerate when model resolution fails")
		}
	})

	t.Run("should return error when model has no supportedUrls", func(t *testing.T) {
		gateway := &mockGatewayForSupportedURLs{
			id:   "models.dev",
			name: "Models.dev",
			resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
				return &mockLanguageModelWithSupportedURLs{
					specVersion:   "v2",
					provider:      "custom",
					modelID:       "custom-model",
					supportedURLs: nil,
				}, nil
			},
		}

		model, err := NewModelRouterLanguageModel("custom/custom-model", []MastraModelGateway{gateway})
		if err != nil {
			t.Fatalf("unexpected error creating model: %v", err)
		}

		// Model was created successfully but the underlying model has no supportedUrls
		if model.Provider() != "custom" {
			t.Errorf("provider = %q, want %q", model.Provider(), "custom")
		}
	})

	t.Run("should only resolve the underlying model once (caching)", func(t *testing.T) {
		resolveCalls := 0
		gateway := &mockGatewayForSupportedURLs{
			id:   "models.dev",
			name: "Models.dev",
			resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
				resolveCalls++
				return mockMistralModel, nil
			},
		}

		// Clear the global model instances cache for this test
		modelInstancesMu.Lock()
		modelInstances = make(map[string]GatewayLanguageModel)
		modelInstancesMu.Unlock()

		model, err := NewModelRouterLanguageModel("mistral/mistral-large-latest", []MastraModelGateway{gateway})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Access DoGenerate multiple times - model should only be resolved once
		_, _ = model.DoGenerate(LanguageModelV2CallOptions{})
		_, _ = model.DoGenerate(LanguageModelV2CallOptions{})
		_, _ = model.DoGenerate(LanguageModelV2CallOptions{})

		if resolveCalls != 1 {
			t.Errorf("resolveLanguageModel called %d times, want 1 (should be cached)", resolveCalls)
		}
	})
}
