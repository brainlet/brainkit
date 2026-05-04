package registry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	bkmodule "github.com/brainlet/brainkit/module"
	jsruntimemod "github.com/brainlet/brainkit/modules/jsruntime"
	"github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	registrymod "github.com/brainlet/brainkit/modules/registry"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	secretsmod "github.com/brainlet/brainkit/modules/secrets"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/protocol"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// registryEnv creates a kit with explicit provider/storage/vector config for registry tests.
func registryEnv(t *testing.T) *brainkit.Kit {
	t.Helper()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-registry",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{
			brainkit.OpenAI("test-key"),
		},
		Vectors: map[string]brainkit.VectorConfig{
			"main": brainkit.PgVectorStore("pg://test"),
		},
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.InMemoryStorage(),
		},
		Modules: []bkmodule.Module{registrymod.New(), toolsmod.New(), packages.New()},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

func testGoSideRegisterAndList(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verify provider is registered via registry.list bus command
	pr, err := protocol.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "provider"})
	require.NoError(t, err)
	listCh := make(chan registrymsg.RegistryListResp, 1)
	unsub, _ := sdk.SubscribeTo[registrymsg.RegistryListResp](k, ctx, pr.ReplyTo,
		func(resp registrymsg.RegistryListResp, _ sdk.Message) { listCh <- resp })
	defer unsub()

	select {
	case resp := <-listCh:
		assert.Contains(t, string(resp.Items), "openai")
	case <-ctx.Done():
		t.Fatal("timeout listing providers")
	}

	// Verify vector store via registry.list
	pr2, _ := protocol.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "vectorStore"})
	vecCh := make(chan registrymsg.RegistryListResp, 1)
	unsub2, _ := sdk.SubscribeTo[registrymsg.RegistryListResp](k, ctx, pr2.ReplyTo,
		func(resp registrymsg.RegistryListResp, _ sdk.Message) { vecCh <- resp })
	defer unsub2()

	select {
	case resp := <-vecCh:
		assert.Contains(t, string(resp.Items), "main")
	case <-ctx.Done():
		t.Fatal("timeout listing vectors")
	}

	// Verify storage via registry.list
	pr3, _ := protocol.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "storage"})
	storCh := make(chan registrymsg.RegistryListResp, 1)
	unsub3, _ := sdk.SubscribeTo[registrymsg.RegistryListResp](k, ctx, pr3.ReplyTo,
		func(resp registrymsg.RegistryListResp, _ sdk.Message) { storCh <- resp })
	defer unsub3()

	select {
	case resp := <-storCh:
		assert.Contains(t, string(resp.Items), "default")
	case <-ctx.Done():
		t.Fatal("timeout listing storages")
	}
}

func testGoSideRuntimeRegisterUnregister(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-registry-dynamic",
		FSRoot:    t.TempDir(),
		Modules:   []bkmodule.Module{registrymod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Add provider via bus
	addResp, err := sdk.Call[registrymsg.ProviderAddMsg, registrymsg.ProviderAddResp](k, ctx, registrymsg.ProviderAddMsg{
		Name: "anthropic", Type: "anthropic", Config: json.RawMessage(`{"APIKey":"sk-ant"}`),
	})
	require.NoError(t, err)
	require.True(t, addResp.Added)

	// Remove via bus
	rmResp, err := sdk.Call[registrymsg.ProviderRemoveMsg, registrymsg.ProviderRemoveResp](k, ctx, registrymsg.ProviderRemoveMsg{Name: "anthropic"})
	require.NoError(t, err)
	assert.True(t, rmResp.Removed)
}

func testJSBridgeHas(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)

	result := testutil.EvalTS(t, k, "__test_reg_has.ts", `
		var hasOpenAI = registry.has("provider", "openai");
		var hasAnthropic = registry.has("provider", "anthropic");
		return JSON.stringify({ hasOpenAI: hasOpenAI, hasAnthropic: hasAnthropic });
	`)
	assert.Contains(t, result, `"hasOpenAI":true`)
	assert.Contains(t, result, `"hasAnthropic":false`)
}

func testJSBridgeList(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-registry-list",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{
			brainkit.OpenAI("test1"),
			brainkit.Anthropic("test2"),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	result := testutil.EvalTS(t, k, "__test_reg_list.ts", `
		var providers = registry.list("provider");
		return JSON.stringify({ count: providers.length, names: providers.map(function(p) { return p.name; }).sort() });
	`)
	assert.Contains(t, result, `"count":2`)
	assert.Contains(t, result, `"anthropic"`)
	assert.Contains(t, result, `"openai"`)
}

func testJSBridgeResolve(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)

	result := testutil.EvalTS(t, k, "__test_reg_resolve.ts", `
		try {
			var p = provider("openai");
			return JSON.stringify({ resolved: true, type: typeof p });
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	assert.Contains(t, result, `"resolved":true`)
}

func testWithDeployedTS(t *testing.T, _ *suite.TestEnv) {
	k := registryEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mp, _ := json.Marshal(map[string]string{"name": "registry-user", "entry": "registry-user.ts"})
	pr, err := protocol.Publish(k, ctx, packagemsg.PackageDeployMsg{
		Manifest: mp,
		Files: map[string]string{"registry-user.ts": `
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
		`},
	})
	require.NoError(t, err)
	ch := make(chan packagemsg.PackageDeployResp, 1)
	unsub, _ := sdk.SubscribeTo[packagemsg.PackageDeployResp](k, ctx, pr.ReplyTo, func(r packagemsg.PackageDeployResp, m sdk.Message) { ch <- r })
	defer unsub()
	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	pr2, err := protocol.Publish(k, ctx, toolmsg.ToolCallMsg{Name: "check-providers", Input: map[string]any{}})
	require.NoError(t, err)
	ch2 := make(chan toolmsg.ToolCallResp, 1)
	unsub2, _ := sdk.SubscribeTo[toolmsg.ToolCallResp](k, ctx, pr2.ReplyTo, func(r toolmsg.ToolCallResp, m sdk.Message) { ch2 <- r })
	defer unsub2()
	var resp toolmsg.ToolCallResp
	select {
	case resp = <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	var result map[string]any
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, true, result["hasOpenAI"])
}

func testJSDynamicProviderRegisterUsableByModel(t *testing.T, _ *suite.TestEnv) {
	env := suite.NewEnv(t, suite.EnvConfig{FSRoot: true})

	result := testutil.EvalTS(t, env.Kit, "__test_dynamic_provider.ts", `
		const originalCreateOpenAI = globalThis.__agent_embed.createOpenAI;
		const seen = [];
		globalThis.__agent_embed.createOpenAI = function(opts) {
			seen.push(opts);
			return function(modelId) {
				return { modelId: modelId, opts: opts };
			};
		};
		try {
			registry.register("provider", "proxy-openai", {
				type: "openai",
				apiKey: "dynamic-key",
				baseURL: "https://proxy.internal/v1",
			});

			const resolved = registry.resolve("provider", "proxy-openai");
			const modelRef = model("proxy-openai", "gpt-dynamic");
			const providerRef = provider("proxy-openai");

			return JSON.stringify({
				modelId: modelRef.modelId,
				modelApiKey: modelRef.opts.apiKey,
				modelBaseURL: modelRef.opts.baseURL,
				providerType: typeof providerRef,
				publicAPIKey: resolved.config.APIKey || resolved.config.apiKey,
				createCalls: seen.length,
			});
		} finally {
			registry.unregister("provider", "proxy-openai");
			globalThis.__agent_embed.createOpenAI = originalCreateOpenAI;
		}
	`)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "gpt-dynamic", parsed["modelId"])
	assert.Equal(t, "dynamic-key", parsed["modelApiKey"])
	assert.Equal(t, "https://proxy.internal/v1", parsed["modelBaseURL"])
	assert.Equal(t, "function", parsed["providerType"])
	assert.Equal(t, "****", parsed["publicAPIKey"])
	assert.GreaterOrEqual(t, parsed["createCalls"].(float64), float64(2))
}

func testJSDynamicProviderRegisterPropagatesValidation(t *testing.T, _ *suite.TestEnv) {
	env := suite.NewEnv(t, suite.EnvConfig{FSRoot: true})

	_, err := testutil.EvalTSErr(env.Kit, "__test_bad_dynamic_provider.ts", `
		registry.register("provider", "", {
			type: "openai",
			apiKey: "dynamic-key",
		});
		return "no-error";
	`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func testSecretRotateRefreshesJSProviderCache(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-provider-secret-refresh",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{
			brainkit.OpenAI("initial-key"),
		},
		Modules: []bkmodule.Module{jsruntimemod.New(), registrymod.New(), secretsmod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	setup := testutil.EvalTS(t, k, "__test_provider_secret_refresh_setup.ts", `
		globalThis.__test_provider_seen = [];
		globalThis.__test_original_create_openai = globalThis.__agent_embed.createOpenAI;
		globalThis.__agent_embed.createOpenAI = function(opts) {
			globalThis.__test_provider_seen.push(opts);
			return function(modelId) {
				return { modelId: modelId, opts: opts };
			};
		};
		const before = provider("openai")("before");
		return JSON.stringify({ apiKey: before.opts.apiKey, calls: globalThis.__test_provider_seen.length });
	`)
	var before map[string]any
	require.NoError(t, json.Unmarshal([]byte(setup), &before))
	assert.Equal(t, "initial-key", before["apiKey"])
	assert.Equal(t, float64(1), before["calls"])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := secretmsg.CallSecretsRotate(k, ctx, secretmsg.SecretsRotateMsg{
		Name:     "OPENAI_API_KEY",
		NewValue: "rotated-key",
	})
	require.NoError(t, err)
	require.True(t, resp.Rotated)

	result := testutil.EvalTS(t, k, "__test_provider_secret_refresh_after.ts", `
		try {
			const resolved = registry.resolve("provider", "openai");
			const after = provider("openai")("after");
			return JSON.stringify({
				apiKey: after.opts.apiKey,
				publicAPIKey: resolved.config.APIKey || resolved.config.apiKey,
				calls: globalThis.__test_provider_seen.length,
			});
		} finally {
			globalThis.__agent_embed.createOpenAI = globalThis.__test_original_create_openai;
			delete globalThis.__test_original_create_openai;
			delete globalThis.__test_provider_seen;
		}
	`)
	var after map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &after))
	assert.Equal(t, "rotated-key", after["apiKey"])
	assert.Equal(t, "****", after["publicAPIKey"])
	assert.Equal(t, float64(2), after["calls"])
}

func testBusProviderAddInvalidatesJSProviderCache(t *testing.T, _ *suite.TestEnv) {
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test-provider-bus-cache",
		FSRoot:    t.TempDir(),
		Providers: []brainkit.ProviderConfig{
			brainkit.OpenAI("initial-key"),
		},
		Modules: []bkmodule.Module{jsruntimemod.New(), registrymod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	setup := testutil.EvalTS(t, k, "__test_provider_bus_cache_setup.ts", `
		globalThis.__test_bus_provider_seen = [];
		globalThis.__test_bus_original_create_openai = globalThis.__agent_embed.createOpenAI;
		globalThis.__agent_embed.createOpenAI = function(opts) {
			globalThis.__test_bus_provider_seen.push(opts);
			return function(modelId) {
				return { modelId: modelId, opts: opts };
			};
		};
		const before = provider("openai")("before");
		return JSON.stringify({ apiKey: before.opts.apiKey, calls: globalThis.__test_bus_provider_seen.length });
	`)
	var before map[string]any
	require.NoError(t, json.Unmarshal([]byte(setup), &before))
	assert.Equal(t, "initial-key", before["apiKey"])
	assert.Equal(t, float64(1), before["calls"])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := registrymsg.CallProviderAdd(k, ctx, registrymsg.ProviderAddMsg{
		Name:   "openai",
		Type:   "openai",
		Config: json.RawMessage(`{"APIKey":"bus-key"}`),
	})
	require.NoError(t, err)
	require.True(t, resp.Added)

	result := testutil.EvalTS(t, k, "__test_provider_bus_cache_after.ts", `
		try {
			const resolved = registry.resolve("provider", "openai");
			const after = provider("openai")("after");
			return JSON.stringify({
				apiKey: after.opts.apiKey,
				publicAPIKey: resolved.config.APIKey || resolved.config.apiKey,
				calls: globalThis.__test_bus_provider_seen.length,
			});
		} finally {
			globalThis.__agent_embed.createOpenAI = globalThis.__test_bus_original_create_openai;
			delete globalThis.__test_bus_original_create_openai;
			delete globalThis.__test_bus_provider_seen;
		}
	`)
	var after map[string]any
	require.NoError(t, json.Unmarshal([]byte(result), &after))
	assert.Equal(t, "bus-key", after["apiKey"])
	assert.Equal(t, "****", after["publicAPIKey"])
	assert.Equal(t, float64(2), after["calls"])
}
