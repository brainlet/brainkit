package registry

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	provreg "github.com/brainlet/brainkit/internal/providers"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryEnv creates a kernel with explicit provider/storage/vector config for registry tests.
func registryEnv(t *testing.T) *brainkit.Kernel {
	t.Helper()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-registry",
		FSRoot:    t.TempDir(),
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
	t.Cleanup(func() { k.Close() })
	return k
}

func testGoSideRegisterAndList(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)

	providers := k.ListAIProviders()
	require.Len(t, providers, 1)
	assert.Equal(t, "openai", providers[0].Name)
	assert.Equal(t, provreg.AIProviderOpenAI, providers[0].Type)
	assert.True(t, providers[0].Capabilities.Chat)
	assert.True(t, providers[0].Capabilities.Embedding)

	vectors := k.ListVectorStores()
	require.Len(t, vectors, 1)
	assert.Equal(t, "main", vectors[0].Name)
	assert.Equal(t, provreg.VectorStorePg, vectors[0].Type)

	storages := k.ListStorages()
	require.Len(t, storages, 1)
	assert.Equal(t, "default", storages[0].Name)
	assert.Equal(t, provreg.StorageInMemory, storages[0].Type)
}

func testGoSideRuntimeRegisterUnregister(t *testing.T, _ *suite.TestEnv) {
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

	assert.Empty(t, k.ListAIProviders())
	assert.Empty(t, k.ListVectorStores())
	assert.Empty(t, k.ListStorages())

	err = k.RegisterAIProvider("anthropic", provreg.AIProviderAnthropic, provreg.AnthropicProviderConfig{APIKey: "sk-ant"})
	require.NoError(t, err)
	assert.Len(t, k.ListAIProviders(), 1)

	err = k.RegisterVectorStore("qdrant", provreg.VectorStoreQdrant, provreg.QdrantVectorConfig{URL: "http://localhost:6333"})
	require.NoError(t, err)
	assert.Len(t, k.ListVectorStores(), 1)

	k.UnregisterAIProvider("anthropic")
	assert.Empty(t, k.ListAIProviders())

	k.UnregisterVectorStore("qdrant")
	assert.Empty(t, k.ListVectorStores())
}

func testJSBridgeHas(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := k.EvalTS(ctx, "__test_reg_has.ts", `
		var hasOpenAI = registry.has("provider", "openai");
		var hasAnthropic = registry.has("provider", "anthropic");
		return JSON.stringify({ hasOpenAI: hasOpenAI, hasAnthropic: hasAnthropic });
	`)
	require.NoError(t, err)
	assert.Contains(t, result, `"hasOpenAI":true`)
	assert.Contains(t, result, `"hasAnthropic":false`)
}

func testJSBridgeList(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test",
		CallerID:  "test-registry-list",
		FSRoot:    t.TempDir(),
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

func testJSBridgeResolve(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func testWithDeployedTS(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)
	rt := sdk.Runtime(k)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
	ch := make(chan messages.KitDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{Name: "check-providers", Input: map[string]any{}})
	require.NoError(t, err)
	ch2 := make(chan messages.ToolCallResp, 1)
	unsub2, _ := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch2 <- r })
	defer unsub2()
	var resp messages.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, true, result["hasOpenAI"])
}
