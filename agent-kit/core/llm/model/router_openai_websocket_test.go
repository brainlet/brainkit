// Ported from: packages/core/src/llm/model/router-openai-websocket.test.ts
package model

import (
	"testing"
)

// The TS test uses vi.mock to mock @ai-sdk/openai-v5 and ./openai-websocket-fetch.js,
// then tests the interaction between Agent, ModelRouterLanguageModel, and WebSocket transport.
// In Go, Agent is not ported yet, so we test the router-level WebSocket transport logic directly.

// ---------------------------------------------------------------------------
// Mock types for WebSocket transport tests
// ---------------------------------------------------------------------------

// mockOpenAIGateway implements MastraModelGateway for testing WebSocket transport.
type mockOpenAIGateway struct {
	id                     string
	name                   string
	resolveLanguageModelFn func(args ResolveLanguageModelArgs) (GatewayLanguageModel, error)
}

func (g *mockOpenAIGateway) ID() string   { return g.id }
func (g *mockOpenAIGateway) Name() string { return g.name }
func (g *mockOpenAIGateway) FetchProviders() (map[string]ProviderConfig, error) {
	return nil, nil
}
func (g *mockOpenAIGateway) BuildURL(modelID string, envVars map[string]string) (string, error) {
	return "", nil
}
func (g *mockOpenAIGateway) GetAPIKey(modelID string) (string, error) {
	return "test-openai-key", nil
}
func (g *mockOpenAIGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	if g.resolveLanguageModelFn != nil {
		return g.resolveLanguageModelFn(args)
	}
	return &mockLanguageModelV2{
		specVersion: "v2",
		provider:    "openai.responses",
		modelID:     args.ModelID,
	}, nil
}

func TestModelRouterOpenAIWebSocketTransport(t *testing.T) {
	t.Run("uses WebSocket transport configuration", func(t *testing.T) {
		// Test getOpenAITransport function which parses provider options for WebSocket config
		t.Run("should return fetch transport by default", func(t *testing.T) {
			transport, wsOpts := getOpenAITransport(nil)
			if transport != OpenAITransportFetch {
				t.Errorf("transport = %q, want %q", transport, OpenAITransportFetch)
			}
			if wsOpts != nil {
				t.Error("expected nil wsOpts for default transport")
			}
		})

		t.Run("should return fetch when no openai options provided", func(t *testing.T) {
			transport, wsOpts := getOpenAITransport(map[string]any{
				"anthropic": map[string]any{"thinking": true},
			})
			if transport != OpenAITransportFetch {
				t.Errorf("transport = %q, want %q", transport, OpenAITransportFetch)
			}
			if wsOpts != nil {
				t.Error("expected nil wsOpts when no openai options")
			}
		})

		t.Run("should detect websocket transport", func(t *testing.T) {
			transport, wsOpts := getOpenAITransport(map[string]any{
				"openai": map[string]any{
					"transport": "websocket",
					"websocket": map[string]any{
						"url": "wss://api.openai.com/v1/responses",
					},
				},
			})
			if transport != OpenAITransportWebSocket {
				t.Errorf("transport = %q, want %q", transport, OpenAITransportWebSocket)
			}
			if wsOpts == nil {
				t.Fatal("expected non-nil wsOpts for websocket transport")
			}
			if wsOpts.URL != "wss://api.openai.com/v1/responses" {
				t.Errorf("wsOpts.URL = %q, want %q", wsOpts.URL, "wss://api.openai.com/v1/responses")
			}
		})

		t.Run("should parse closeOnFinish option", func(t *testing.T) {
			closeOnFinish := false
			transport, wsOpts := getOpenAITransport(map[string]any{
				"openai": map[string]any{
					"transport": "websocket",
					"websocket": map[string]any{
						"closeOnFinish": closeOnFinish,
					},
				},
			})
			if transport != OpenAITransportWebSocket {
				t.Errorf("transport = %q, want %q", transport, OpenAITransportWebSocket)
			}
			if wsOpts == nil {
				t.Fatal("expected non-nil wsOpts")
			}
			if wsOpts.CloseOnFinish == nil {
				t.Fatal("expected non-nil CloseOnFinish")
			}
			if *wsOpts.CloseOnFinish != false {
				t.Errorf("CloseOnFinish = %v, want false", *wsOpts.CloseOnFinish)
			}
		})

		t.Run("should parse websocket headers", func(t *testing.T) {
			_, wsOpts := getOpenAITransport(map[string]any{
				"openai": map[string]any{
					"transport": "websocket",
					"websocket": map[string]any{
						"url": "wss://api.openai.com/v1/responses",
						"headers": map[string]string{
							"X-Test": "ws-header",
						},
					},
				},
			})
			if wsOpts == nil {
				t.Fatal("expected non-nil wsOpts")
			}
			if wsOpts.Headers["X-Test"] != "ws-header" {
				t.Errorf("headers[X-Test] = %q, want %q", wsOpts.Headers["X-Test"], "ws-header")
			}
		})
	})

	t.Run("isOpenAIBaseURL", func(t *testing.T) {
		t.Run("should return true for empty URL", func(t *testing.T) {
			if !isOpenAIBaseURL("") {
				t.Error("expected true for empty URL")
			}
		})

		t.Run("should return true for api.openai.com", func(t *testing.T) {
			if !isOpenAIBaseURL("https://api.openai.com/v1") {
				t.Error("expected true for api.openai.com")
			}
		})

		t.Run("should return false for custom URL", func(t *testing.T) {
			if isOpenAIBaseURL("https://custom-api.example.com/v1") {
				t.Error("expected false for custom URL")
			}
		})
	})

	t.Run("stableHeaderKey", func(t *testing.T) {
		t.Run("should return empty string for nil headers", func(t *testing.T) {
			key := stableHeaderKey(nil)
			if key != "" {
				t.Errorf("expected empty string, got %q", key)
			}
		})

		t.Run("should return empty string for empty headers", func(t *testing.T) {
			key := stableHeaderKey(map[string]string{})
			if key != "" {
				t.Errorf("expected empty string, got %q", key)
			}
		})

		t.Run("should produce same key regardless of insertion order", func(t *testing.T) {
			headers1 := map[string]string{
				"X-First":  "1",
				"X-Second": "2",
				"X-Third":  "3",
			}
			headers2 := map[string]string{
				"X-Third":  "3",
				"X-First":  "1",
				"X-Second": "2",
			}
			key1 := stableHeaderKey(headers1)
			key2 := stableHeaderKey(headers2)
			if key1 != key2 {
				t.Errorf("expected same key for same headers in different order: %q vs %q", key1, key2)
			}
		})
	})

	t.Run("WebSocket fetch integration", func(t *testing.T) {
		t.Run("should create OpenAIWebSocketFetch with default URL", func(t *testing.T) {
			wsFetch := NewOpenAIWebSocketFetch(nil)
			if wsFetch == nil {
				t.Fatal("expected non-nil OpenAIWebSocketFetch")
			}
			if wsFetch.wsURL != "wss://api.openai.com/v1/responses" {
				t.Errorf("wsURL = %q, want %q", wsFetch.wsURL, "wss://api.openai.com/v1/responses")
			}
		})

		t.Run("should create OpenAIWebSocketFetch with custom URL", func(t *testing.T) {
			wsFetch := NewOpenAIWebSocketFetch(&CreateOpenAIWebSocketFetchOptions{
				URL: "wss://custom.api.com/v1/responses",
			})
			if wsFetch.wsURL != "wss://custom.api.com/v1/responses" {
				t.Errorf("wsURL = %q, want %q", wsFetch.wsURL, "wss://custom.api.com/v1/responses")
			}
		})

		t.Run("should create OpenAIWebSocketFetch with custom headers", func(t *testing.T) {
			wsFetch := NewOpenAIWebSocketFetch(&CreateOpenAIWebSocketFetchOptions{
				Headers: map[string]string{
					"X-Custom": "test",
				},
			})
			if wsFetch.baseHeaders["X-Custom"] != "test" {
				t.Errorf("baseHeaders[X-Custom] = %q, want %q", wsFetch.baseHeaders["X-Custom"], "test")
			}
		})

		t.Run("Close should be safe to call multiple times", func(t *testing.T) {
			wsFetch := NewOpenAIWebSocketFetch(nil)
			// Should not panic
			wsFetch.Close()
			wsFetch.Close()
		})
	})

	t.Run("openAIWSAllowlist", func(t *testing.T) {
		t.Run("should include openai", func(t *testing.T) {
			if !openAIWSAllowlist["openai"] {
				t.Error("expected openai to be in the WebSocket allowlist")
			}
		})

		t.Run("should not include other providers", func(t *testing.T) {
			if openAIWSAllowlist["anthropic"] {
				t.Error("expected anthropic to NOT be in the WebSocket allowlist")
			}
		})
	})

	t.Run("ModelRouterLanguageModel with WebSocket transport", func(t *testing.T) {
		t.Run("should create model that accepts websocket provider options", func(t *testing.T) {
			gateway := &mockOpenAIGateway{
				id:   "models.dev",
				name: "Models.dev",
			}

			model, err := NewModelRouterLanguageModel(
				OpenAICompatibleConfig{
					ID: "openai/gpt-4o",
					Headers: map[string]string{
						"X-Test": "ws",
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
			if model.SpecificationVersion() != "v2" {
				t.Errorf("specificationVersion = %q, want %q", model.SpecificationVersion(), "v2")
			}
		})
	})
}
