package registry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	packagesmod "github.com/brainlet/brainkit/modules/packages"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	registrymod "github.com/brainlet/brainkit/modules/registry"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
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
		Modules: []brainkit.Module{registrymod.New(), toolsmod.New(), packagesmod.New()},
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
	pr, err := sdk.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "provider"})
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
	pr2, _ := sdk.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "vectorStore"})
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
	pr3, _ := sdk.Publish(k, ctx, registrymsg.RegistryListMsg{Category: "storage"})
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
		Modules:   []brainkit.Module{registrymod.New()},
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
	pr, err := sdk.Publish(k, ctx, packagemsg.PackageDeployMsg{
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

	pr2, err := sdk.Publish(k, ctx, toolmsg.ToolCallMsg{Name: "check-providers", Input: map[string]any{}})
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
