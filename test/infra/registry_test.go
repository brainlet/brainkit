package infra_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	provreg "github.com/brainlet/brainkit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_GoSide_RegisterAndList(t *testing.T) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry",
		FSRoot: t.TempDir(),
		AIProviders: map[string]provreg.AIProviderRegistration{
			"openai": {Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: "test-key"}},
		},
		Vectors: map[string]brainkit.VectorConfig{
			"main": brainkit.PgVectorStore("pg://test"),
		},
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
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
	// Clear env vars so auto-detect doesn't register providers
	origKey := os.Getenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	defer func() {
		if origKey != "" {
			os.Setenv("OPENAI_API_KEY", origKey)
		}
	}()

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-registry-dynamic",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	defer k.Close()

	// Initially empty (no auto-detected providers since env was cleared)
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-js",
		FSRoot: t.TempDir(),
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-list",
		FSRoot: t.TempDir(),
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-resolve",
		FSRoot: t.TempDir(),
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
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-registry-deploy",
		FSRoot: t.TempDir(),
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
	_pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
			kit.register("tool", "check-providers", registryTool);
		`,
	})
	require.NoError(t, err)
	_ch1 := make(chan messages.KitDeployResp, 1)
	_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
	defer _us1()
	select {
	case <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Call the tool — it uses registry from inside a Compartment
	_pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
		Name: "check-providers", Input: map[string]any{},
	})
	require.NoError(t, err)
	_ch2 := make(chan messages.ToolCallResp, 1)
	_us2, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch2 <- r })
	require.NoError(t, err)
	defer _us2()
	var resp messages.ToolCallResp
	select {
	case resp = <-_ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, true, result["hasOpenAI"])

	_spr1, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "registry-user.ts"})
	_sch1 := make(chan messages.KitTeardownResp, 1)
	_sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch1 <- r })
	defer _sun1()
	select { case <-_sch1: case <-ctx.Done(): t.Fatal("timeout") }
}
