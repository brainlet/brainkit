// Ported from: packages/core/src/llm/model/router-openrouter-headers.test.ts
package model

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Mock types for OpenRouter header tests
// ---------------------------------------------------------------------------

// mockOpenRouterGateway implements MastraModelGateway simulating OpenRouter behavior.
type mockOpenRouterGateway struct {
	id                     string
	name                   string
	resolveLanguageModelFn func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error)
	lastResolveArgs        *ResolveLanguageModelArgs
}

func (g *mockOpenRouterGateway) ID() string   { return g.id }
func (g *mockOpenRouterGateway) Name() string { return g.name }
func (g *mockOpenRouterGateway) FetchProviders() (map[string]ProviderConfig, error) {
	return nil, nil
}
func (g *mockOpenRouterGateway) BuildURL(modelID string, envVars map[string]string) (string, error) {
	return "", nil
}
func (g *mockOpenRouterGateway) GetAPIKey(modelID string) (string, error) {
	return "test-openrouter-key", nil
}
func (g *mockOpenRouterGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	g.lastResolveArgs = &args
	if g.resolveLanguageModelFn != nil {
		return g.resolveLanguageModelFn(args)
	}
	return &mockLanguageModelV2{
		specVersion: "v2",
		provider:    "openrouter",
		modelID:     args.ModelID,
	}, nil
}

func TestModelRouterOpenRouterHeaders(t *testing.T) {
	t.Run("Headers passing", func(t *testing.T) {
		t.Run("should pass headers when creating ModelRouterLanguageModel with id format", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID: "openrouter/anthropic/claude-3-5-sonnet-20241022",
					Headers: map[string]string{
						"HTTP-Referer": "http://my-service/",
						"X-Title":      "my-application-name",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.Provider() != "anthropic" {
				t.Errorf("provider = %q, want %q", model.Provider(), "anthropic")
			}

			// Verify that headers are stored on the model's config
			if model.config.Headers == nil {
				t.Fatal("expected headers to be set on model config")
			}
			if model.config.Headers["HTTP-Referer"] != "http://my-service/" {
				t.Errorf("HTTP-Referer = %q, want %q", model.config.Headers["HTTP-Referer"], "http://my-service/")
			}
			if model.config.Headers["X-Title"] != "my-application-name" {
				t.Errorf("X-Title = %q, want %q", model.config.Headers["X-Title"], "my-application-name")
			}
		})

		t.Run("should pass headers when using providerId/modelId format", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ProviderID: "openrouter/openai",
					ModelID:    "gpt-4o",
					Headers: map[string]string{
						"HTTP-Referer": "https://myapp.com",
						"X-Title":      "MyApp",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.config.Headers == nil {
				t.Fatal("expected headers to be set")
			}
			if model.config.Headers["HTTP-Referer"] != "https://myapp.com" {
				t.Errorf("HTTP-Referer = %q, want %q", model.config.Headers["HTTP-Referer"], "https://myapp.com")
			}
		})

		t.Run("should work without headers (backward compatibility)", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				"openrouter/anthropic/claude-3-5-sonnet-20241022",
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Headers should be nil when not provided
			if model.config.Headers != nil {
				t.Errorf("expected nil headers, got %v", model.config.Headers)
			}
		})

		t.Run("should pass custom API key along with headers", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:     "openrouter/meta-llama/llama-3.1-8b-instruct",
					APIKey: "custom-openrouter-key-123",
					Headers: map[string]string{
						"HTTP-Referer": "https://example.com",
						"X-Title":      "Example App",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if model.config.APIKey != "custom-openrouter-key-123" {
				t.Errorf("apiKey = %q, want %q", model.config.APIKey, "custom-openrouter-key-123")
			}
			if model.config.Headers["HTTP-Referer"] != "https://example.com" {
				t.Errorf("HTTP-Referer = %q, want %q", model.config.Headers["HTTP-Referer"], "https://example.com")
			}
		})

		t.Run("should handle multiple header fields", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID: "openrouter/google/gemini-pro",
					Headers: map[string]string{
						"HTTP-Referer":    "https://myapp.com",
						"X-Title":         "My Application",
						"X-Custom-Header": "custom-value",
						"X-User-ID":       "user-123",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(model.config.Headers) != 4 {
				t.Errorf("expected 4 headers, got %d", len(model.config.Headers))
			}
			if model.config.Headers["X-Custom-Header"] != "custom-value" {
				t.Errorf("X-Custom-Header = %q, want %q", model.config.Headers["X-Custom-Header"], "custom-value")
			}
			if model.config.Headers["X-User-ID"] != "user-123" {
				t.Errorf("X-User-ID = %q, want %q", model.config.Headers["X-User-ID"], "user-123")
			}
		})
	})

	t.Run("Model caching with headers", func(t *testing.T) {
		t.Run("should create different model instances for different headers", func(t *testing.T) {
			resolveCount := 0
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
				resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
					resolveCount++
					return &mockLanguageModelV2{
						specVersion: "v2",
						provider:    "openrouter",
						modelID:     args.ModelID,
					}, nil
				},
			}

			// Clear cache
			modelInstancesMu.Lock()
			modelInstances = make(map[string]GatewayLanguageModel)
			modelInstancesMu.Unlock()

			model1, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID: "openrouter/anthropic/claude-3-5-sonnet-20241022",
					Headers: map[string]string{
						"X-Title": "App1",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			model2, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID: "openrouter/anthropic/claude-3-5-sonnet-20241022",
					Headers: map[string]string{
						"X-Title": "App2",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Trigger resolution for both
			_, _ = model1.DoGenerate(LanguageModelV2CallOptions{})
			_, _ = model2.DoGenerate(LanguageModelV2CallOptions{})

			// Both should have been resolved (different headers = different cache keys)
			if resolveCount != 2 {
				t.Errorf("resolveLanguageModel called %d times, want 2 (different headers)", resolveCount)
			}
		})

		t.Run("should reuse model instance for same headers", func(t *testing.T) {
			resolveCount := 0
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
				resolveLanguageModelFn: func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
					resolveCount++
					return &mockLanguageModelV2{
						specVersion: "v2",
						provider:    "openrouter",
						modelID:     args.ModelID,
					}, nil
				},
			}

			// Clear cache
			modelInstancesMu.Lock()
			modelInstances = make(map[string]GatewayLanguageModel)
			modelInstancesMu.Unlock()

			sharedHeaders := map[string]string{
				"HTTP-Referer": "https://shared.com",
				"X-Title":      "Shared App",
			}

			model1, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:      "openrouter/anthropic/claude-3-5-sonnet-20241022",
					Headers: sharedHeaders,
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			model2, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:      "openrouter/anthropic/claude-3-5-sonnet-20241022",
					Headers: sharedHeaders,
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Trigger resolution for both
			_, _ = model1.DoGenerate(LanguageModelV2CallOptions{})
			_, _ = model2.DoGenerate(LanguageModelV2CallOptions{})

			// Should only be called once since headers are the same
			if resolveCount != 1 {
				t.Errorf("resolveLanguageModel called %d times, want 1 (same headers should be cached)", resolveCount)
			}
		})
	})

	t.Run("Error handling", func(t *testing.T) {
		t.Run("should pass headers even when using custom API key (no env var needed)", func(t *testing.T) {
			gateway := &mockOpenRouterGateway{
				id:   "openrouter",
				name: "OpenRouter",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID:     "openrouter/anthropic/claude-3-5-sonnet-20241022",
					APIKey: "custom-key-no-env",
					Headers: map[string]string{
						"HTTP-Referer": "http://my-service/",
						"X-Title":      "my-application-name",
					},
				},
				[]MastraModelGateway{gateway},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should not panic during model creation
			if model == nil {
				t.Fatal("expected non-nil model")
			}

			// Verify API key and headers are set correctly
			if model.config.APIKey != "custom-key-no-env" {
				t.Errorf("apiKey = %q, want %q", model.config.APIKey, "custom-key-no-env")
			}
			if model.config.Headers["HTTP-Referer"] != "http://my-service/" {
				t.Errorf("HTTP-Referer = %q, want %q", model.config.Headers["HTTP-Referer"], "http://my-service/")
			}
		})
	})
}
