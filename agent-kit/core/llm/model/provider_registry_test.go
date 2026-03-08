// Ported from: packages/core/src/llm/model/provider-registry.test.ts
package model

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// Helper to reset the singleton GatewayRegistry between tests
// ---------------------------------------------------------------------------

func resetGatewayRegistry() {
	// Reset the sync.Once and instance to allow re-creation
	gatewayRegistryOnce = sync.Once{}
	gatewayRegistryInstance = nil
}

// ---------------------------------------------------------------------------
// Tests for GatewayRegistry (ported from provider-registry.test.ts)
// ---------------------------------------------------------------------------

func TestGatewayRegistry(t *testing.T) {
	// The TS test file tests auto-refresh with cache files, env vars, and intervals.
	// Many of those are deeply tied to Node.js fs and process.env.
	// We port what is testable in Go.

	t.Run("should return nil providers when no registry data is set", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()
		providers := registry.GetProviders()
		if providers != nil {
			t.Errorf("expected nil providers, got %v", providers)
		}
	})

	t.Run("should set and get registry data", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()
		registry.SetRegistryData(&RegistryData{
			Providers: map[string]ProviderConfig{
				"test-provider": {
					Name:   "Test Provider",
					Models: []string{"model-a", "model-b"},
				},
			},
			Models: map[string][]string{
				"test-provider": {"model-a", "model-b"},
			},
			Version: "1.0.0",
		})

		providers := registry.GetProviders()
		if providers == nil {
			t.Fatal("expected non-nil providers")
		}
		if _, ok := providers["test-provider"]; !ok {
			t.Error("expected 'test-provider' to be in providers")
		}
	})

	t.Run("should get provider config by ID", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()
		registry.SetRegistryData(&RegistryData{
			Providers: map[string]ProviderConfig{
				"openai": {
					Name:   "OpenAI",
					Models: []string{"gpt-4o", "gpt-3.5-turbo"},
				},
			},
			Models: map[string][]string{
				"openai": {"gpt-4o", "gpt-3.5-turbo"},
			},
		})

		cfg, ok := registry.GetProviderConfig("openai")
		if !ok {
			t.Fatal("expected provider config for 'openai'")
		}
		if cfg.Name != "OpenAI" {
			t.Errorf("name = %q, want %q", cfg.Name, "OpenAI")
		}

		_, ok = registry.GetProviderConfig("nonexistent")
		if ok {
			t.Error("expected no provider config for 'nonexistent'")
		}
	})

	t.Run("should check if provider is registered", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()
		registry.SetRegistryData(&RegistryData{
			Providers: map[string]ProviderConfig{
				"anthropic": {Name: "Anthropic", Models: []string{"claude-3-opus"}},
			},
		})

		if !registry.IsProviderRegistered("anthropic") {
			t.Error("expected 'anthropic' to be registered")
		}
		if registry.IsProviderRegistered("unknown") {
			t.Error("expected 'unknown' to NOT be registered")
		}
	})

	t.Run("should get models grouped by provider", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()
		registry.SetRegistryData(&RegistryData{
			Models: map[string][]string{
				"openai":    {"gpt-4o", "gpt-3.5-turbo"},
				"anthropic": {"claude-3-opus", "claude-3-sonnet"},
			},
		})

		models := registry.GetModels()
		if len(models) != 2 {
			t.Errorf("expected 2 providers in models, got %d", len(models))
		}
		if len(models["openai"]) != 2 {
			t.Errorf("expected 2 openai models, got %d", len(models["openai"]))
		}
	})

	t.Run("should register and get custom gateways", func(t *testing.T) {
		resetGatewayRegistry()
		registry := GetGatewayRegistry()

		mockGW := &mockGatewayForResolve{id: "custom-gateway", name: "Custom Gateway"}
		registry.RegisterCustomGateways([]MastraModelGateway{mockGW})

		gateways := registry.GetCustomGateways()
		if len(gateways) != 1 {
			t.Fatalf("expected 1 custom gateway, got %d", len(gateways))
		}
		if gateways[0].ID() != "custom-gateway" {
			t.Errorf("gateway ID = %q, want %q", gateways[0].ID(), "custom-gateway")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests for ParseModelString
// ---------------------------------------------------------------------------

func TestParseModelString(t *testing.T) {
	t.Run("should parse provider/model format", func(t *testing.T) {
		result := ParseModelString("openai/gpt-4o")
		if result.Provider != "openai" {
			t.Errorf("provider = %q, want %q", result.Provider, "openai")
		}
		if result.ModelID != "gpt-4o" {
			t.Errorf("modelId = %q, want %q", result.ModelID, "gpt-4o")
		}
	})

	t.Run("should handle multi-segment model IDs", func(t *testing.T) {
		result := ParseModelString("fireworks/accounts/etc/model")
		if result.Provider != "fireworks" {
			t.Errorf("provider = %q, want %q", result.Provider, "fireworks")
		}
		if result.ModelID != "accounts/etc/model" {
			t.Errorf("modelId = %q, want %q", result.ModelID, "accounts/etc/model")
		}
	})

	t.Run("should handle model-only string", func(t *testing.T) {
		result := ParseModelString("gpt-4o")
		if result.Provider != "" {
			t.Errorf("provider = %q, want empty string", result.Provider)
		}
		if result.ModelID != "gpt-4o" {
			t.Errorf("modelId = %q, want %q", result.ModelID, "gpt-4o")
		}
	})
}

// ---------------------------------------------------------------------------
// Tests ported from "Corrupted JSON recovery" and "Issue #10434" sections
// ---------------------------------------------------------------------------

func TestCorruptedJSONRecovery(t *testing.T) {
	t.Run("should detect corrupted JSON content", func(t *testing.T) {
		corruptedContent := `{
  "providers": {
    "test": { "name": "Test", "models": ["a", "b"] }
  },
  "models": { "test": ["a", "b"] },
  "version": "1.0.0"
}
}
]}
}`
		var data RegistryData
		err := json.Unmarshal([]byte(corruptedContent), &data)
		if err == nil {
			t.Error("expected JSON parse error for corrupted content, got nil")
		}
	})

	t.Run("should parse valid JSON content", func(t *testing.T) {
		validContent := `{
  "providers": {
    "test": { "name": "Test", "models": ["a", "b"] }
  },
  "models": { "test": ["a", "b"] },
  "version": "1.0.0"
}`
		var data RegistryData
		err := json.Unmarshal([]byte(validContent), &data)
		if err != nil {
			t.Fatalf("unexpected error parsing valid JSON: %v", err)
		}
		if _, ok := data.Providers["test"]; !ok {
			t.Error("expected 'test' provider in parsed data")
		}
	})

	t.Run("should detect unquoted numeric provider names in .d.ts content", func(t *testing.T) {
		// The validation regex used in syncGlobalCacheToLocal to detect corrupted .d.ts files
		re := regexp.MustCompile(`readonly\s+\d`)

		// Corrupted content: unquoted "302ai" starts with a digit - invalid TypeScript
		corruptedDtsContent := `export type ProviderModelsMap = {
  readonly openai: readonly ['gpt-4o'];
  readonly 302ai: readonly ['model-1'];
};`

		// Valid content: "302ai" is properly quoted
		validDtsContent := `export type ProviderModelsMap = {
  readonly openai: readonly ['gpt-4o'];
  readonly '302ai': readonly ['model-1'];
};`

		// Content with no numeric providers at all
		nothingToQuote := `export type ProviderModelsMap = {
  readonly openai: readonly ['gpt-4o'];
  readonly anthropic: readonly ['claude-3'];
};`

		if !re.MatchString(corruptedDtsContent) {
			t.Error("expected regex to match corrupted .d.ts content")
		}
		if re.MatchString(validDtsContent) {
			t.Error("expected regex to NOT match valid .d.ts content")
		}
		if re.MatchString(nothingToQuote) {
			t.Error("expected regex to NOT match content without numeric providers")
		}
	})
}

func TestConcurrentWriteCorruption(t *testing.T) {
	t.Run("should not corrupt JSON file when using atomic writes for concurrent operations", func(t *testing.T) {
		tempDir := t.TempDir()
		testJSONPath := filepath.Join(tempDir, "provider-registry.json")

		jsonContent1, _ := json.MarshalIndent(map[string]any{
			"providers": map[string]any{
				"openai": map[string]any{"name": "OpenAI", "models": []string{"gpt-4", "gpt-3.5-turbo"}},
			},
			"models":  map[string]any{"openai": []string{"gpt-4", "gpt-3.5-turbo"}},
			"version": "1.0.0",
		}, "", "  ")

		jsonContent2, _ := json.MarshalIndent(map[string]any{
			"providers": map[string]any{
				"anthropic": map[string]any{"name": "Anthropic", "models": []string{"claude-3-opus", "claude-3-sonnet"}},
			},
			"models":  map[string]any{"anthropic": []string{"claude-3-opus", "claude-3-sonnet"}},
			"version": "1.0.0",
		}, "", "  ")

		iterations := 50
		corruptionDetected := false

		for i := 0; i < iterations && !corruptionDetected; i++ {
			// Start both atomic writes "simultaneously"
			errCh1 := make(chan error, 1)
			errCh2 := make(chan error, 1)

			go func() {
				errCh1 <- AtomicWriteFile(testJSONPath, string(jsonContent1))
			}()
			go func() {
				errCh2 <- AtomicWriteFile(testJSONPath, string(jsonContent2))
			}()

			err1 := <-errCh1
			err2 := <-errCh2

			if err1 != nil {
				t.Fatalf("atomic write 1 failed: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("atomic write 2 failed: %v", err2)
			}

			// Check if the file is valid JSON
			content, err := os.ReadFile(testJSONPath)
			if err != nil {
				t.Fatalf("failed to read file: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(content, &parsed); err != nil {
				corruptionDetected = true
				t.Errorf("corruption detected on iteration %d: %v", i+1, err)
			}
		}

		if corruptionDetected {
			t.Error("with atomic writes, corruption should NEVER occur")
		}
	})

	t.Run("should handle concurrent syncGlobalCacheToLocal and writeRegistryFiles calls with atomic writes", func(t *testing.T) {
		tempDir := t.TempDir()
		globalCacheDir := filepath.Join(tempDir, "global-cache")
		distDir := filepath.Join(tempDir, "dist")
		globalJSONPath := filepath.Join(globalCacheDir, "provider-registry.json")
		distJSONPath := filepath.Join(distDir, "provider-registry.json")

		if err := os.MkdirAll(globalCacheDir, 0755); err != nil {
			t.Fatalf("failed to create global cache dir: %v", err)
		}
		if err := os.MkdirAll(distDir, 0755); err != nil {
			t.Fatalf("failed to create dist dir: %v", err)
		}

		globalContent, _ := json.MarshalIndent(map[string]any{
			"providers": map[string]any{
				"cached-provider": map[string]any{"name": "Cached", "models": []string{"model-a", "model-b", "model-c"}},
			},
			"models":  map[string]any{"cached-provider": []string{"model-a", "model-b", "model-c"}},
			"version": "1.0.0",
		}, "", "  ")
		if err := os.WriteFile(globalJSONPath, globalContent, 0644); err != nil {
			t.Fatalf("failed to write global cache: %v", err)
		}

		freshContent, _ := json.MarshalIndent(map[string]any{
			"providers": map[string]any{
				"fresh-provider": map[string]any{"name": "Fresh", "models": []string{"new-model-1", "new-model-2"}},
			},
			"models":  map[string]any{"fresh-provider": []string{"new-model-1", "new-model-2"}},
			"version": "1.0.0",
		}, "", "  ")

		iterations := 50
		corruptionDetected := false

		for i := 0; i < iterations && !corruptionDetected; i++ {
			errCh1 := make(chan error, 1)
			errCh2 := make(chan error, 1)

			// Simulate syncGlobalCacheToLocal with atomic write
			go func() {
				content, err := os.ReadFile(globalJSONPath)
				if err != nil {
					errCh1 <- err
					return
				}
				errCh1 <- AtomicWriteFile(distJSONPath, string(content))
			}()

			// Simulate writeRegistryFiles with atomic write
			go func() {
				errCh2 <- AtomicWriteFile(distJSONPath, string(freshContent))
			}()

			err1 := <-errCh1
			err2 := <-errCh2
			if err1 != nil {
				t.Fatalf("sync global to local failed: %v", err1)
			}
			if err2 != nil {
				t.Fatalf("write registry files failed: %v", err2)
			}

			resultContent, err := os.ReadFile(distJSONPath)
			if err != nil {
				t.Fatalf("failed to read dist file: %v", err)
			}

			var parsed map[string]any
			if err := json.Unmarshal(resultContent, &parsed); err != nil {
				corruptionDetected = true
				t.Errorf("corruption detected on iteration %d: %v", i+1, err)
			}
		}

		if corruptionDetected {
			t.Error("with atomic writes, corruption should NEVER occur")
		}
	})
}
