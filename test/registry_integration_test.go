package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_GoSide_RegisterAndList(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "test-key"}},
		},
		VectorStores: map[string]provreg.VectorStoreRegistration{
			"main": {Type: provreg.VectorStorePg, Config: provreg.PgVectorConfig{ConnectionString: "pg://test"}},
		},
		MastraStorages: map[string]provreg.StorageRegistration{
			"default": {Type: provreg.StorageInMemory, Config: provreg.InMemoryStorageConfig{}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// List providers
	providers := k.ListAIProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "openai", providers[0].Name)
	assert.Equal(t, provreg.AIProviderOpenAI, providers[0].Type)
	assert.True(t, providers[0].Capabilities.Chat)
	assert.True(t, providers[0].Capabilities.Embedding)

	// List vector stores
	vectors := k.ListVectorStores()
	require.Len(t, vectors, 1)
	assert.Equal(t, "main", vectors[0].Name)
	assert.Equal(t, provreg.VectorStorePg, vectors[0].Type)

	// List storages
	storages := k.ListStorages()
	require.Len(t, storages, 1)
	assert.Equal(t, "default", storages[0].Name)
	assert.Equal(t, provreg.StorageInMemory, storages[0].Type)
}

func TestRegistry_GoSide_RuntimeRegisterUnregister(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-dynamic",
		WorkspaceDir: t.TempDir(),
	})
	require.NoError(t, err)
	defer k.Close()

	// Initially empty
	assert.Empty(t, k.ListAIProviders())
	assert.Empty(t, k.ListVectorStores())
	assert.Empty(t, k.ListStorages())

	// Register at runtime
	err = k.RegisterAIProvider("anthropic", provreg.AIProviderAnthropic, provreg.AnthropicProviderConfig{APIKey: "sk-ant"})
	require.NoError(t, err)
	assert.Len(t, k.ListAIProviders(), 1)

	err = k.RegisterVectorStore("qdrant", provreg.VectorStoreQdrant, provreg.QdrantVectorConfig{URL: "http://localhost:6333"})
	require.NoError(t, err)
	assert.Len(t, k.ListVectorStores(), 1)

	// Unregister
	k.UnregisterAIProvider("anthropic")
	assert.Empty(t, k.ListAIProviders())

	k.UnregisterVectorStore("qdrant")
	assert.Empty(t, k.ListVectorStores())
}

func TestRegistry_JSBridge_Has(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-js",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "test"}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test registry.has from .ts
	result, err := k.EvalTS(ctx, "__test_reg_has.ts", `
		var hasOpenAI = registry.has("provider", "openai");
		var hasAnthropic = registry.has("provider", "anthropic");
		return JSON.stringify({ hasOpenAI: hasOpenAI, hasAnthropic: hasAnthropic });
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"hasOpenAI":true`)
	assert.Contains(t, result, `"hasAnthropic":false`)
}

func TestRegistry_JSBridge_List(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-list",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai":    {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "test1"}},
			"anthropic": {Type: provreg.AIProviderAnthropic, Config: provreg.AnthropicProviderConfig{APIKey: "test2"}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := k.EvalTS(ctx, "__test_reg_list.ts", `
		var providers = registry.list("provider");
		return JSON.stringify({ count: providers.length, names: providers.map(function(p) { return p.name; }).sort() });
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"count":2`)
	assert.Contains(t, result, `"anthropic"`)
	assert.Contains(t, result, `"openai"`)
}

func TestRegistry_JSBridge_Resolve(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-resolve",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "sk-test-key"}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test provider() resolution from .ts — should get a provider factory
	result, err := k.EvalTS(ctx, "__test_reg_resolve.ts", `
		try {
			var p = provider("openai");
			return JSON.stringify({ resolved: true, type: typeof p });
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"resolved":true`)
}

func TestRegistry_WithDeployedTS(t *testing.T) {
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-deploy",
		WorkspaceDir: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "test"}},
		},
	})
	require.NoError(t, err)
	defer k.Close()

	rt := sdk.Runtime(k)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy .ts that uses registry.has
	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
		Source: "registry-user.ts",
		Code: `
			const registryTool = createTool({
				id: "check-providers",
				description: "checks which providers are registered",
				execute: async () => {
					return {
						hasOpenAI: registry.has("provider", "openai"),
						providers: registry.list("provider").map(function(p) { return p.name; }),
					};
				}
			});
		`,
	})
	require.NoError(t, err)

	// Call the tool — it uses registry from inside a Compartment
	resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
		Name: "check-providers", Input: map[string]any{},
	})
	require.NoError(t, err)

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, true, result["hasOpenAI"])

	sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "registry-user.ts"})
}
