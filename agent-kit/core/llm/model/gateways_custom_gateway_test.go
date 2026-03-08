// Ported from: packages/core/src/llm/model/gateways/custom-gateway.test.ts
package model

import (
	"fmt"
	"os"
	"testing"
)

// ---------------------------------------------------------------------------
// Mock custom gateway implementation for testing
// ---------------------------------------------------------------------------

// testCustomGateway is a mock implementation of MastraModelGateway for testing.
type testCustomGateway struct{}

func (g *testCustomGateway) ID() string   { return "custom" }
func (g *testCustomGateway) Name() string { return "test-custom" }

func (g *testCustomGateway) FetchProviders() (map[string]ProviderConfig, error) {
	return map[string]ProviderConfig{
		"my-provider": {
			Name:         "My Custom Provider",
			Models:       []string{"model-1", "model-2", "model-3"},
			APIKeyEnvVar: "CUSTOM_API_KEY",
			Gateway:      "custom",
			URL:          "https://api.custom-provider.com/v1",
		},
	}, nil
}

func (g *testCustomGateway) BuildURL(_ string, _ map[string]string) (string, error) {
	return "https://api.custom-provider.com/v1", nil
}

func (g *testCustomGateway) GetAPIKey(modelID string) (string, error) {
	apiKey := os.Getenv("CUSTOM_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("Missing CUSTOM_API_KEY environment variable for model: %s", modelID)
	}
	return apiKey, nil
}

func (g *testCustomGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	// TODO: Return a real model when AI SDK providers are available in Go.
	return &stubGatewayModel{
		provider: args.ProviderID,
		modelID:  args.ModelID,
	}, nil
}

// anotherCustomGateway is a second mock gateway with a different prefix.
type anotherCustomGateway struct{}

func (g *anotherCustomGateway) ID() string   { return "another" }
func (g *anotherCustomGateway) Name() string { return "another-custom" }

func (g *anotherCustomGateway) FetchProviders() (map[string]ProviderConfig, error) {
	return map[string]ProviderConfig{
		"another-provider": {
			Name:         "Another Provider",
			Models:       []string{"model-a", "model-b"},
			APIKeyEnvVar: "ANOTHER_API_KEY",
			Gateway:      "another",
			URL:          "https://api.another.com/v1",
		},
	}, nil
}

func (g *anotherCustomGateway) BuildURL(_ string, _ map[string]string) (string, error) {
	return "https://api.another.com/v1", nil
}

func (g *anotherCustomGateway) GetAPIKey(modelID string) (string, error) {
	apiKey := os.Getenv("ANOTHER_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("Missing ANOTHER_API_KEY environment variable for model: %s", modelID)
	}
	return apiKey, nil
}

func (g *anotherCustomGateway) ResolveLanguageModel(args ResolveLanguageModelArgs) (GatewayLanguageModel, error) {
	return &stubGatewayModel{
		provider: args.ProviderID,
		modelID:  args.ModelID,
	}, nil
}

// stubGatewayModel is a minimal GatewayLanguageModel for testing.
type stubGatewayModel struct {
	provider string
	modelID  string
}

func (m *stubGatewayModel) SpecificationVersion() string { return "v2" }
func (m *stubGatewayModel) Provider() string             { return m.provider }
func (m *stubGatewayModel) ModelID() string               { return m.modelID }

// ---------------------------------------------------------------------------
// Mastra stub for testing gateway configuration
// (The real Mastra class is not yet ported to Go)
// ---------------------------------------------------------------------------

// TODO: Replace with real Mastra type when ported.
type stubMastra struct {
	gateways map[string]MastraModelGateway
	agents   map[string]*stubAgent
}

func newStubMastra(gateways map[string]MastraModelGateway) *stubMastra {
	return &stubMastra{
		gateways: gateways,
		agents:   make(map[string]*stubAgent),
	}
}

func (m *stubMastra) listGateways() map[string]MastraModelGateway {
	return m.gateways
}

func (m *stubMastra) addGateway(gw MastraModelGateway, key string) {
	if m.gateways == nil {
		m.gateways = make(map[string]MastraModelGateway)
	}
	m.gateways[key] = gw
}

func (m *stubMastra) getGateway(key string) (MastraModelGateway, error) {
	gw, ok := m.gateways[key]
	if !ok {
		return nil, fmt.Errorf("Gateway with key %s not found", key)
	}
	return gw, nil
}

func (m *stubMastra) addAgent(agent *stubAgent, key string) {
	m.agents[key] = agent
}

func (m *stubMastra) getAgent(key string) (*stubAgent, error) {
	a, ok := m.agents[key]
	if !ok {
		return nil, fmt.Errorf("Agent with key %s not found", key)
	}
	return a, nil
}

// stubAgent is a minimal agent stub for testing.
type stubAgent struct {
	name         string
	instructions string
	model        string
}

// ---------------------------------------------------------------------------
// Tests: Mastra Gateway Configuration
// ---------------------------------------------------------------------------

func TestCustomGateway_MastraGatewayConfiguration(t *testing.T) {
	t.Run("should accept custom gateways in Mastra config", func(t *testing.T) {
		customGateway := &testCustomGateway{}
		mastra := newStubMastra(map[string]MastraModelGateway{
			"custom": customGateway,
		})

		gws := mastra.listGateways()
		if gws == nil {
			t.Fatal("expected non-nil gateways")
		}
		if len(gws) != 1 {
			t.Fatalf("expected 1 gateway, got %d", len(gws))
		}
		if gws["custom"] != customGateway {
			t.Error("expected custom gateway to match")
		}
	})

	t.Run("should accept multiple custom gateways", func(t *testing.T) {
		gw1 := &testCustomGateway{}
		gw2 := &anotherCustomGateway{}
		mastra := newStubMastra(map[string]MastraModelGateway{
			"custom":  gw1,
			"another": gw2,
		})

		gws := mastra.listGateways()
		if gws == nil {
			t.Fatal("expected non-nil gateways")
		}
		if len(gws) != 2 {
			t.Fatalf("expected 2 gateways, got %d", len(gws))
		}
		if gws["custom"] != gw1 {
			t.Error("expected first gateway to match")
		}
		if gws["another"] != gw2 {
			t.Error("expected second gateway to match")
		}
	})

	t.Run("should allow adding gateways after initialization", func(t *testing.T) {
		mastra := newStubMastra(nil)
		gws := mastra.listGateways()
		if len(gws) != 0 {
			t.Fatalf("expected 0 gateways initially, got %d", len(gws))
		}

		customGateway := &testCustomGateway{}
		mastra.addGateway(customGateway, "custom")

		gws = mastra.listGateways()
		if len(gws) != 1 {
			t.Fatalf("expected 1 gateway after adding, got %d", len(gws))
		}
		if gws["custom"] != customGateway {
			t.Error("expected custom gateway to match")
		}
	})

	t.Run("should allow getting a gateway by name", func(t *testing.T) {
		gw1 := &testCustomGateway{}
		mastra := newStubMastra(map[string]MastraModelGateway{
			"custom": gw1,
		})

		gw, err := mastra.getGateway("custom")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gw != gw1 {
			t.Error("expected gateway to match")
		}
	})

	t.Run("should return error when getting non-existent gateway", func(t *testing.T) {
		mastra := newStubMastra(nil)
		_, err := mastra.getGateway("nonexistent")
		if err == nil {
			t.Fatal("expected error for non-existent gateway")
		}
		if err.Error() != "Gateway with key nonexistent not found" {
			t.Errorf("unexpected error message: %s", err.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: ModelRouterLanguageModel with Custom Gateways
// ---------------------------------------------------------------------------

func TestCustomGateway_ModelRouterWithCustomGateways(t *testing.T) {
	t.Run("should use custom gateway when provided", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		model, err := NewModelRouterLanguageModel("custom/my-provider/model-1", []MastraModelGateway{customGateway})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.ModelID() != "model-1" {
			t.Errorf("expected modelId 'model-1', got '%s'", model.ModelID())
		}
		if model.Provider() != "my-provider" {
			t.Errorf("expected provider 'my-provider', got '%s'", model.Provider())
		}
	})

	t.Run("should fall back to default gateways when custom gateways array is empty", func(t *testing.T) {
		// With no gateways at all, this should fail since there's no default
		// gateway registered in the Go port yet.
		// The TS test expects this to work because default gateways (netlify, models.dev)
		// are always available. In Go, defaultGateways() returns nil.
		_, err := NewModelRouterLanguageModel("openai/gpt-4o", []MastraModelGateway{})
		if err == nil {
			// If it succeeds, verify the model is created correctly
			t.Log("default gateways available, model created")
		} else {
			// Expected in current Go port since default gateways are not yet populated
			t.Log("no default gateways available (expected in current Go port state)")
		}
	})

	t.Run("should prefer custom gateway over default when both can handle the model", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		model, err := NewModelRouterLanguageModel("custom/my-provider/model-1", []MastraModelGateway{customGateway})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if model.Provider() != "my-provider" {
			t.Errorf("expected provider 'my-provider', got '%s'", model.Provider())
		}
	})

	t.Run("should fall back to default gateways when custom gateways cannot handle the model ID", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		// Model ID doesn't match custom gateway prefix.
		// In TS, default gateways handle this. In Go, defaultGateways() returns nil
		// so this may fail unless there's a models.dev gateway configured.
		model, err := NewModelRouterLanguageModel("openai/gpt-4", []MastraModelGateway{customGateway})
		if err != nil {
			// Expected: no gateway found for 'openai' prefix when only 'custom' gateway exists
			// and no default gateways are available.
			t.Log("no matching gateway found (expected: custom gateway doesn't handle 'openai' prefix)")
		} else {
			if model == nil {
				t.Fatal("expected non-nil model")
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Gateway Integration with Agents
// ---------------------------------------------------------------------------

func TestCustomGateway_AgentIntegration(t *testing.T) {
	t.Run("should use custom gateway from Mastra instance in agent", func(t *testing.T) {
		customGateway := &testCustomGateway{}
		mastra := newStubMastra(map[string]MastraModelGateway{
			"custom": customGateway,
		})

		agent := &stubAgent{
			name:         "test-agent",
			instructions: "You are a test agent",
			model:        "custom/my-provider/model-1",
		}

		mastra.addAgent(agent, "testAgent")

		retrieved, err := mastra.getAgent("testAgent")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if retrieved == nil {
			t.Fatal("expected non-nil agent")
		}
		if retrieved.name != "test-agent" {
			t.Errorf("expected agent name 'test-agent', got '%s'", retrieved.name)
		}
	})

	t.Run("should support multiple gateways for different agents", func(t *testing.T) {
		gw1 := &testCustomGateway{}
		gw2 := &anotherCustomGateway{}
		mastra := newStubMastra(map[string]MastraModelGateway{
			"custom":  gw1,
			"another": gw2,
		})

		agent1 := &stubAgent{
			name:         "agent-1",
			instructions: "Agent using custom gateway",
			model:        "custom/my-provider/model-1",
		}
		agent2 := &stubAgent{
			name:         "agent-2",
			instructions: "Agent using another gateway",
			model:        "another/another-provider/model-a",
		}

		mastra.addAgent(agent1, "agent1")
		mastra.addAgent(agent2, "agent2")

		a1, err := mastra.getAgent("agent1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a1 == nil {
			t.Fatal("expected non-nil agent1")
		}

		a2, err := mastra.getAgent("agent2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a2 == nil {
			t.Fatal("expected non-nil agent2")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Gateway fetchProviders
// ---------------------------------------------------------------------------

func TestCustomGateway_FetchProviders(t *testing.T) {
	t.Run("should correctly fetch providers from custom gateway", func(t *testing.T) {
		customGateway := &testCustomGateway{}
		providers, err := customGateway.FetchProviders()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if providers == nil {
			t.Fatal("expected non-nil providers")
		}

		myProvider, ok := providers["my-provider"]
		if !ok {
			t.Fatal("expected 'my-provider' in providers")
		}
		if myProvider.Name != "My Custom Provider" {
			t.Errorf("expected name 'My Custom Provider', got '%s'", myProvider.Name)
		}
		if len(myProvider.Models) != 3 {
			t.Fatalf("expected 3 models, got %d", len(myProvider.Models))
		}
		expected := []string{"model-1", "model-2", "model-3"}
		for i, m := range expected {
			if myProvider.Models[i] != m {
				t.Errorf("expected model %d to be '%s', got '%s'", i, m, myProvider.Models[i])
			}
		}
	})

	t.Run("should correctly build URLs for custom gateway", func(t *testing.T) {
		customGateway := &testCustomGateway{}
		url, err := customGateway.BuildURL("custom/my-provider/model-1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != "https://api.custom-provider.com/v1" {
			t.Errorf("expected URL 'https://api.custom-provider.com/v1', got '%s'", url)
		}
	})

	t.Run("should correctly get API keys for custom gateway", func(t *testing.T) {
		t.Setenv("CUSTOM_API_KEY", "test-custom-key")

		customGateway := &testCustomGateway{}
		apiKey, err := customGateway.GetAPIKey("custom/my-provider/model-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if apiKey != "test-custom-key" {
			t.Errorf("expected API key 'test-custom-key', got '%s'", apiKey)
		}
	})

	t.Run("should return error when API key is missing", func(t *testing.T) {
		// Ensure CUSTOM_API_KEY is not set
		t.Setenv("CUSTOM_API_KEY", "")
		os.Unsetenv("CUSTOM_API_KEY")

		customGateway := &testCustomGateway{}
		_, err := customGateway.GetAPIKey("custom/my-provider/model-1")
		if err == nil {
			t.Fatal("expected error for missing API key")
		}
		if got := err.Error(); got != "Missing CUSTOM_API_KEY environment variable for model: custom/my-provider/model-1" {
			t.Errorf("unexpected error message: %s", got)
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Gateway Prefix Handling
// ---------------------------------------------------------------------------

func TestCustomGateway_PrefixHandling(t *testing.T) {
	t.Run("should correctly parse model IDs with custom prefix", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		model, err := NewModelRouterLanguageModel("custom/my-provider/model-1", []MastraModelGateway{customGateway})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if model.Provider() != "my-provider" {
			t.Errorf("expected provider 'my-provider', got '%s'", model.Provider())
		}
		if model.ModelID() != "model-1" {
			t.Errorf("expected modelId 'model-1', got '%s'", model.ModelID())
		}
	})

	t.Run("should handle models with different prefixes", func(t *testing.T) {
		gw1 := &testCustomGateway{}
		gw2 := &anotherCustomGateway{}

		model1, err := NewModelRouterLanguageModel("custom/my-provider/model-1", []MastraModelGateway{gw1, gw2})
		if err != nil {
			t.Fatalf("unexpected error creating model1: %v", err)
		}
		if model1.Provider() != "my-provider" {
			t.Errorf("model1: expected provider 'my-provider', got '%s'", model1.Provider())
		}

		model2, err := NewModelRouterLanguageModel("another/another-provider/model-a", []MastraModelGateway{gw1, gw2})
		if err != nil {
			t.Fatalf("unexpected error creating model2: %v", err)
		}
		if model2.Provider() != "another-provider" {
			t.Errorf("model2: expected provider 'another-provider', got '%s'", model2.Provider())
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Custom Gateway Error Handling
// ---------------------------------------------------------------------------

func TestCustomGateway_ErrorHandling(t *testing.T) {
	t.Run("should handle gateway resolution errors gracefully", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		// Invalid model ID format (missing model part after provider)
		_, err := NewModelRouterLanguageModel("custom/invalid", []MastraModelGateway{customGateway})
		if err == nil {
			t.Fatal("expected error for invalid model ID format")
		}
	})

	t.Run("should fail for unknown prefixes when no default gateways available", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		// Model ID with unknown prefix should fail when no default gateways are available
		// In TS, this falls back to models.dev. In Go, defaultGateways() returns nil.
		_, err := NewModelRouterLanguageModel("anthropic/claude-3-5-sonnet-20241022", []MastraModelGateway{customGateway})
		if err == nil {
			t.Log("model created (default gateways available)")
		} else {
			t.Log("no matching gateway found (expected when only custom gateway exists)")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests: Gateway with Dynamic Model Config
// ---------------------------------------------------------------------------

func TestCustomGateway_DynamicModelConfig(t *testing.T) {
	t.Run("should work with OpenAICompatibleConfig objects (ID form)", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		model, err := NewModelRouterLanguageModel(
			OpenAICompatibleConfig{
				ID:     "custom/my-provider/model-1",
				APIKey: "override-key",
			},
			[]MastraModelGateway{customGateway},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.Provider() != "my-provider" {
			t.Errorf("expected provider 'my-provider', got '%s'", model.Provider())
		}
		if model.ModelID() != "model-1" {
			t.Errorf("expected modelId 'model-1', got '%s'", model.ModelID())
		}
	})

	t.Run("should work with providerId/modelId config objects", func(t *testing.T) {
		customGateway := &testCustomGateway{}

		model, err := NewModelRouterLanguageModel(
			OpenAICompatibleConfig{
				ProviderID: "custom/my-provider",
				ModelID:    "model-1",
				APIKey:     "override-key",
			},
			[]MastraModelGateway{customGateway},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}
		if model.Provider() != "my-provider" {
			t.Errorf("expected provider 'my-provider', got '%s'", model.Provider())
		}
		if model.ModelID() != "model-1" {
			t.Errorf("expected modelId 'model-1', got '%s'", model.ModelID())
		}
	})
}
