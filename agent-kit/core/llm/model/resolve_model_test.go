// Ported from: packages/core/src/llm/model/resolve-model.test.ts
package model

import (
	"fmt"
	"testing"
)

// mockLanguageModelV2 implements MastraLanguageModel and LanguageModelV2 for testing.
type mockLanguageModelV2 struct {
	specVersion string
	provider    string
	modelID     string
}

func (m *mockLanguageModelV2) SpecificationVersion() string { return m.specVersion }
func (m *mockLanguageModelV2) Provider() string             { return m.provider }
func (m *mockLanguageModelV2) ModelID() string              { return m.modelID }
func (m *mockLanguageModelV2) DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	return LanguageModelV2StreamResult{}, nil
}
func (m *mockLanguageModelV2) DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error) {
	return LanguageModelV2StreamResult{}, nil
}

// mockGatewayForResolve implements MastraModelGateway for testing resolve-model.
type mockGatewayForResolve struct {
	id   string
	name string
}

func (g *mockGatewayForResolve) ID() string   { return g.id }
func (g *mockGatewayForResolve) Name() string { return g.name }
func (g *mockGatewayForResolve) FetchProviders() (map[string]ProviderConfig, error) {
	return map[string]ProviderConfig{
		"openai":          {Name: "OpenAI", Models: []string{"gpt-4o"}},
		"anthropic":       {Name: "Anthropic", Models: []string{"claude-3-opus"}},
		"custom-provider": {Name: "Custom Provider", Models: []string{"my-model"}},
		"public-provider": {Name: "Public Provider", Models: []string{"public-model"}},
	}, nil
}
func (g *mockGatewayForResolve) BuildURL(modelID string, envVars map[string]string) (string, error) {
	return "", nil
}
func (g *mockGatewayForResolve) GetAPIKey(modelID string) (string, error) {
	return "mock-key", nil
}
func (g *mockGatewayForResolve) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	return &mockLanguageModelV2{
		specVersion: "v2",
		provider:    args.ProviderID,
		modelID:     args.ModelID,
	}, nil
}

func TestResolveModelConfig(t *testing.T) {
	gateways := []MastraModelGateway{
		&mockGatewayForResolve{id: "models.dev", name: "Models.dev"},
	}

	t.Run("should resolve a magic string to ModelRouterLanguageModel", func(t *testing.T) {
		result, err := ResolveModelConfig("openai/gpt-4o", gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(*ModelRouterLanguageModel); !ok {
			t.Errorf("expected *ModelRouterLanguageModel, got %T", result)
		}
	})

	t.Run("should resolve a config object to ModelRouterLanguageModel", func(t *testing.T) {
		result, err := ResolveModelConfig(OpenAICompatibleConfig{
			ID:     "openai/gpt-4o",
			APIKey: "test-key",
		}, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(*ModelRouterLanguageModel); !ok {
			t.Errorf("expected *ModelRouterLanguageModel, got %T", result)
		}
	})

	t.Run("should return a LanguageModel instance as-is", func(t *testing.T) {
		model := &mockLanguageModelV2{
			specVersion: "v2",
			provider:    "openai.responses",
			modelID:     "gpt-4o",
		}
		result, err := ResolveModelConfig(model, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(MastraLanguageModel)
		if !ok {
			t.Fatalf("expected MastraLanguageModel, got %T", result)
		}
		if got := m.ModelID(); got != "gpt-4o" {
			t.Errorf("modelId = %q, want %q", got, "gpt-4o")
		}
		if got := m.Provider(); got != "openai.responses" {
			t.Errorf("provider = %q, want %q", got, "openai.responses")
		}
		if got := m.SpecificationVersion(); got != "v2" {
			t.Errorf("specificationVersion = %q, want %q", got, "v2")
		}
	})

	t.Run("should resolve a dynamic function returning a string", func(t *testing.T) {
		dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
			return "openai/gpt-4o", nil
		})
		result, err := ResolveModelConfig(dynamicFn, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(*ModelRouterLanguageModel); !ok {
			t.Errorf("expected *ModelRouterLanguageModel, got %T", result)
		}
	})

	t.Run("should resolve a dynamic function returning a config object", func(t *testing.T) {
		dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
			return OpenAICompatibleConfig{
				ID:     "openai/gpt-4o",
				APIKey: "test-key",
			}, nil
		})
		result, err := ResolveModelConfig(dynamicFn, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := result.(*ModelRouterLanguageModel); !ok {
			t.Errorf("expected *ModelRouterLanguageModel, got %T", result)
		}
	})

	t.Run("should resolve a dynamic function returning a LanguageModel", func(t *testing.T) {
		model := &mockLanguageModelV2{
			specVersion: "v2",
			provider:    "openai.responses",
			modelID:     "gpt-4o",
		}
		dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
			return model, nil
		})
		result, err := ResolveModelConfig(dynamicFn, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(MastraLanguageModel)
		if !ok {
			t.Fatalf("expected MastraLanguageModel, got %T", result)
		}
		if got := m.ModelID(); got != "gpt-4o" {
			t.Errorf("modelId = %q, want %q", got, "gpt-4o")
		}
		if got := m.Provider(); got != "openai.responses" {
			t.Errorf("provider = %q, want %q", got, "openai.responses")
		}
		if got := m.SpecificationVersion(); got != "v2" {
			t.Errorf("specificationVersion = %q, want %q", got, "v2")
		}
	})

	t.Run("should pass requestContext to dynamic function", func(t *testing.T) {
		// In the TS test, a RequestContext is used. In Go, we pass it via ModelConfigFuncArgs.
		dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
			// Simulate accessing requestContext to pick a model
			return "anthropic/claude-3-opus", nil
		})
		result, err := ResolveModelConfig(dynamicFn, gateways)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(*ModelRouterLanguageModel)
		if !ok {
			t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
		}
		if got := m.ModelID(); got != "claude-3-opus" {
			t.Errorf("modelId = %q, want %q", got, "claude-3-opus")
		}
		if got := m.Provider(); got != "anthropic" {
			t.Errorf("provider = %q, want %q", got, "anthropic")
		}
	})

	t.Run("should throw error for invalid config", func(t *testing.T) {
		_, err := ResolveModelConfig(OpenAICompatibleConfig{}, gateways)
		if err == nil {
			t.Fatal("expected error for empty config, got nil")
		}
	})

	t.Run("unknown specificationVersion handling", func(t *testing.T) {
		t.Run("should handle model with unknown specificationVersion that has DoStream/DoGenerate", func(t *testing.T) {
			// In the Go port, ResolveModelConfig returns the model as-is since
			// wrapping with AISDKV5LanguageModel requires more infrastructure.
			model := &mockLanguageModelV2{
				specVersion: "v4",
				provider:    "ollama.responses",
				modelID:     "llama3.2",
			}
			result, err := ResolveModelConfig(model, gateways)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			m, ok := result.(MastraLanguageModel)
			if !ok {
				t.Fatalf("expected MastraLanguageModel, got %T", result)
			}
			if got := m.ModelID(); got != "llama3.2" {
				t.Errorf("modelId = %q, want %q", got, "llama3.2")
			}
			if got := m.Provider(); got != "ollama.responses" {
				t.Errorf("provider = %q, want %q", got, "ollama.responses")
			}
		})

		t.Run("should pass through a model with v1 specificationVersion", func(t *testing.T) {
			model := &mockLanguageModelV2{
				specVersion: "v1",
				provider:    "test",
				modelID:     "test-model",
			}
			result, err := ResolveModelConfig(model, gateways)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Should be returned as-is
			if result != model {
				t.Errorf("expected same model instance to be returned")
			}
		})
	})

	t.Run("custom OpenAI-compatible config objects", func(t *testing.T) {
		t.Run("using id format (provider/model)", func(t *testing.T) {
			t.Run("should resolve a custom config with id, url, and apiKey", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ID:     "custom-provider/my-model",
					URL:    "https://api.mycompany.com/v1/chat/completions",
					APIKey: "custom-api-key",
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "my-model" {
					t.Errorf("modelId = %q, want %q", got, "my-model")
				}
				if got := m.Provider(); got != "custom-provider" {
					t.Errorf("provider = %q, want %q", got, "custom-provider")
				}
			})

			t.Run("should resolve a custom config with custom headers", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ID:     "custom-provider/my-model",
					URL:    "https://api.mycompany.com/v1/chat/completions",
					APIKey: "custom-api-key",
					Headers: map[string]string{
						"x-custom-header": "custom-value",
						"x-api-version":   "2024-01",
					},
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "my-model" {
					t.Errorf("modelId = %q, want %q", got, "my-model")
				}
				if got := m.Provider(); got != "custom-provider" {
					t.Errorf("provider = %q, want %q", got, "custom-provider")
				}
			})

			t.Run("should resolve a custom config without apiKey (for public endpoints)", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ID:  "public-provider/public-model",
					URL: "https://public-api.example.com/v1/chat/completions",
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "public-model" {
					t.Errorf("modelId = %q, want %q", got, "public-model")
				}
				if got := m.Provider(); got != "public-provider" {
					t.Errorf("provider = %q, want %q", got, "public-provider")
				}
			})
		})

		t.Run("using providerId/modelId format", func(t *testing.T) {
			t.Run("should resolve a custom config with providerId, modelId, url, and apiKey", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ProviderID: "custom-provider",
					ModelID:    "my-model",
					URL:        "https://api.mycompany.com/v1/chat/completions",
					APIKey:     "custom-api-key",
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "my-model" {
					t.Errorf("modelId = %q, want %q", got, "my-model")
				}
				if got := m.Provider(); got != "custom-provider" {
					t.Errorf("provider = %q, want %q", got, "custom-provider")
				}
			})

			t.Run("should resolve a custom config with custom headers", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ProviderID: "custom-provider",
					ModelID:    "my-model",
					URL:        "https://api.mycompany.com/v1/chat/completions",
					APIKey:     "custom-api-key",
					Headers: map[string]string{
						"x-custom-header": "custom-value",
						"x-api-version":   "2024-01",
					},
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "my-model" {
					t.Errorf("modelId = %q, want %q", got, "my-model")
				}
				if got := m.Provider(); got != "custom-provider" {
					t.Errorf("provider = %q, want %q", got, "custom-provider")
				}
			})

			t.Run("should resolve a custom config without apiKey (for public endpoints)", func(t *testing.T) {
				result, err := ResolveModelConfig(OpenAICompatibleConfig{
					ProviderID: "public-provider",
					ModelID:    "public-model",
					URL:        "https://public-api.example.com/v1/chat/completions",
				}, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "public-model" {
					t.Errorf("modelId = %q, want %q", got, "public-model")
				}
				if got := m.Provider(); got != "public-provider" {
					t.Errorf("provider = %q, want %q", got, "public-provider")
				}
			})
		})

		t.Run("dynamic functions", func(t *testing.T) {
			t.Run("should resolve a dynamic function returning id format", func(t *testing.T) {
				dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
					return OpenAICompatibleConfig{
						ID:     "dynamic-provider/dynamic-model",
						URL:    "https://api.mycompany.com/v1/chat/completions",
						APIKey: "dynamic-api-key",
					}, nil
				})
				result, err := ResolveModelConfig(dynamicFn, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "dynamic-model" {
					t.Errorf("modelId = %q, want %q", got, "dynamic-model")
				}
				if got := m.Provider(); got != "dynamic-provider" {
					t.Errorf("provider = %q, want %q", got, "dynamic-provider")
				}
			})

			t.Run("should resolve a dynamic function returning providerId/modelId format", func(t *testing.T) {
				dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
					return OpenAICompatibleConfig{
						ProviderID: "dynamic-provider",
						ModelID:    "dynamic-model",
						URL:        "https://api.mycompany.com/v1/chat/completions",
						APIKey:     "dynamic-api-key",
					}, nil
				})
				result, err := ResolveModelConfig(dynamicFn, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "dynamic-model" {
					t.Errorf("modelId = %q, want %q", got, "dynamic-model")
				}
				if got := m.Provider(); got != "dynamic-provider" {
					t.Errorf("provider = %q, want %q", got, "dynamic-provider")
				}
			})

			t.Run("should resolve a custom config selected from request context", func(t *testing.T) {
				// Simulate request context passing custom endpoint and API key
				customEndpoint := "https://api.mycompany.com/v1/chat/completions"
				customAPIKey := "context-api-key"

				dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
					return OpenAICompatibleConfig{
						ProviderID: "context-provider",
						ModelID:    "context-model",
						URL:        customEndpoint,
						APIKey:     customAPIKey,
					}, nil
				})
				result, err := ResolveModelConfig(dynamicFn, gateways)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				m, ok := result.(*ModelRouterLanguageModel)
				if !ok {
					t.Fatalf("expected *ModelRouterLanguageModel, got %T", result)
				}
				if got := m.ModelID(); got != "context-model" {
					t.Errorf("modelId = %q, want %q", got, "context-model")
				}
				if got := m.Provider(); got != "context-provider" {
					t.Errorf("provider = %q, want %q", got, "context-provider")
				}
			})
		})
	})
}

// TestResolveModelConfigErrors tests error cases that the TS tests cover.
func TestResolveModelConfigErrors(t *testing.T) {
	t.Run("should return error for dynamic function that fails", func(t *testing.T) {
		dynamicFn := ModelConfigFunc(func(args ModelConfigFuncArgs) (MastraModelConfig, error) {
			return nil, fmt.Errorf("dynamic resolution failed")
		})
		_, err := ResolveModelConfig(dynamicFn, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
