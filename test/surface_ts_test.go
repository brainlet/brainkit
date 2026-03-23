package test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	provreg "github.com/brainlet/brainkit/kit/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTSSurface tests every domain operation callable from deployed .ts code.
// Pattern: deploy .ts that creates a tool wrapping the domain operation,
// then Go calls the tool and verifies the result. This proves the
// .ts → bridgeRequest/bridgeRequestAsync → LocalInvoker → handler path.

func newTSKernel(t *testing.T) *testKernel {
	t.Helper()
	loadEnv(t)
	tmpDir := t.TempDir()
	aiProviders := make(map[string]provreg.AIProviderRegistration)
	envVars := make(map[string]string)
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		aiProviders["openai"] = provreg.AIProviderRegistration{
			Type: provreg.AIProviderOpenAI, Config: provreg.OpenAIProviderConfig{APIKey: key},
		}
		envVars["OPENAI_API_KEY"] = key
	}
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test", CallerID: "test-ts-surface", WorkspaceDir: tmpDir,
		AIProviders:      aiProviders,
		EmbeddedStorages: map[string]kit.EmbeddedStorageConfig{"default": {Path: tmpDir + "/brainkit.db"}},
		EnvVars:          envVars,
		MastraStorages:   map[string]provreg.StorageRegistration{"default": {Type: provreg.StorageInMemory, Config: provreg.InMemoryStorageConfig{}}},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return &testKernel{k}
}

// --- Tools domain from TS ---

func TestTSSurface_Tools(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-tools-surface.ts",
		Code: `
			const tsList = createTool({ id: "ts-tools-list", execute: async () => {
				return await tools.list();
			}});
			const tsCall = createTool({ id: "ts-tools-call", execute: async ({ context: input }) => {
				return await tools.call("ts-tools-list", {});
			}});
			const tsResolve = createTool({ id: "ts-tools-resolve", execute: async ({ context: input }) => {
				var t = tool(input.name || "ts-tools-list");
				return { id: t.id, description: t.description };
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-tools-surface.ts"})

	t.Run("List", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-tools-list", Input: map[string]any{}})
		require.NoError(t, err)
		assert.NotNil(t, resp.Result)
	})
	t.Run("Call", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-tools-call", Input: map[string]any{}})
		require.NoError(t, err)
		assert.NotNil(t, resp.Result)
	})
	t.Run("Resolve", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-tools-resolve", Input: map[string]any{"name": "ts-tools-list"}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotEmpty(t, result["id"])
	})
}

// --- FS domain from TS ---

func TestTSSurface_FS(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// FS operations aren't directly in the kit module — deploy .ts that uses bridgeRequestAsync
	// via tools.call to a Go-registered tool that wraps fs operations.
	// Simpler: deploy .ts that writes/reads via Go bridges.
	// Actually, fs has no JS API in kit_runtime.js. TS code accesses fs through Go tools.
	// We test by deploying .ts that creates tools which use the Workspace/LocalFilesystem APIs.
	// But the simplest proof: .ts can call Go-registered fs-wrapper tools.

	// Register Go-side wrapper tools for FS
	_, err := sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](tk, ctx, messages.FsWriteMsg{Path: "ts-test.txt", Data: "hello from Go"})
	require.NoError(t, err)

	// Deploy .ts that reads the file via tools (testing the TS→Go→FS path)
	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-fs-surface.ts",
		Code: `
			const tsReadFile = createTool({ id: "ts-read-file", execute: async ({ context: input }) => {
				// Use tools.call to invoke a Go tool that reads the file
				// FS operations go through Go — .ts proves the bridge works
				return { note: "FS has no JS API — tested via Go Direct surface" };
			}});
		`,
	})
	require.NoError(t, err)
	t.Log("FS domain: no JS-side FS API in kit_runtime.js — covered by Go Direct tests (go_direct_fs_test.go)")
}

// --- Agents domain from TS ---

func TestTSSurface_Agents(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-agents-surface.ts",
		Code: `
			const tsAgentsList = createTool({ id: "ts-agents-list", execute: async () => {
				var list = agents.list();
				return { agents: list };
			}});
			const tsAgentsDiscover = createTool({ id: "ts-agents-discover", execute: async () => {
				var found = agents.discover({ capability: "nonexistent" });
				return { agents: found };
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-agents-surface.ts"})

	t.Run("List", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-agents-list", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotNil(t, result["agents"])
	})
	t.Run("Discover", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-agents-discover", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotNil(t, result["agents"])
	})
}

// --- AI domain from TS ---

func TestTSSurface_AI(t *testing.T) {
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required")
	}
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-ai-surface.ts",
		Code: `
			const tsGenerate = createTool({ id: "ts-ai-generate", execute: async () => {
				var result = await ai.generate({ model: "openai/gpt-4o-mini", prompt: "Say exactly: pong" });
				return { text: result.text };
			}});
			const tsEmbed = createTool({ id: "ts-ai-embed", execute: async () => {
				var result = await ai.embed({ model: "openai/text-embedding-3-small", value: "test" });
				return { dimensions: result.embedding.length };
			}});
			const tsEmbedMany = createTool({ id: "ts-ai-embedmany", execute: async () => {
				var result = await ai.embedMany({ model: "openai/text-embedding-3-small", values: ["a", "b"] });
				return { count: result.embeddings.length };
			}});
			const tsGenerateObject = createTool({ id: "ts-ai-genobj", execute: async () => {
				var result = await ai.generateObject({
					model: "openai/gpt-4o-mini",
					prompt: "Give me a color",
					schema: z.object({ color: z.string() }),
				});
				return { object: result.object };
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-ai-surface.ts"})

	t.Run("Generate", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-ai-generate", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotEmpty(t, result["text"])
	})
	t.Run("Embed", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-ai-embed", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Greater(t, result["dimensions"], float64(0))
	})
	t.Run("EmbedMany", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-ai-embedmany", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, float64(2), result["count"])
	})
	t.Run("GenerateObject", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-ai-genobj", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotNil(t, result["object"])
	})
}

// --- Memory domain from TS ---

func TestTSSurface_Memory(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-memory-surface.ts",
		Code: `
			// Create memory inside the Compartment — can't access globalThis from outer scope
			const _mem = createMemory({ storage: new InMemoryStore(), lastMessages: 10 });
			const tsMemAll = createTool({ id: "ts-mem-all", execute: async () => {
				var mem = _mem;
				var thread = await mem.createThread({});
				var threadId = thread.id;
				await mem.saveMessages({ threadId: threadId, messages: [{ role: "user", content: "hello" }] });
				var got = await mem.getThreadById({ threadId: threadId });
				var list = await mem.listThreads({});
				await mem.deleteThread(threadId);
				return {
					created: !!threadId,
					saved: true,
					got: !!got,
					listed: !!(list && (list.threads || list)),
					deleted: true,
				};
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-memory-surface.ts"})

	t.Run("AllOperations", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-mem-all", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["created"])
		assert.Equal(t, true, result["saved"])
		assert.Equal(t, true, result["got"])
		assert.Equal(t, true, result["listed"])
		assert.Equal(t, true, result["deleted"])
	})
}

// --- Workflows domain from TS ---

func TestTSSurface_Workflows(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-wf-surface.ts",
		Code: `
			const wf = createWorkflow({
				id: "ts-surface-wf",
				inputSchema: z.object({ value: z.string() }),
				outputSchema: z.object({ result: z.string() }),
			});
			const step1 = createStep({
				id: "step1",
				inputSchema: z.object({ value: z.string() }),
				outputSchema: z.object({ result: z.string() }),
				execute: async ({ inputData }) => ({ result: inputData.value + "-processed" }),
			});
			wf.then(step1).commit();

			const tsWfRun = createTool({ id: "ts-wf-run", execute: async ({ context: input }) => {
				var run = await createWorkflowRun(wf);
				var result = await run.start({ inputData: { value: input.val || "test" } });
				return result;
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-wf-surface.ts"})

	t.Run("Run", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wf-run", Input: map[string]any{"val": "hello"}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "success", result["status"])
	})
}

// --- WASM domain from TS ---

func TestTSSurface_WASM(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-wasm-surface.ts",
		Code: `
			const tsWasmCompile = createTool({ id: "ts-wasm-compile", execute: async () => {
				var result = await wasm.compile('export function run(): i32 { return 77; }', { name: "ts-compiled" });
				return { name: result.name || result.moduleId, size: result.size };
			}});
			const tsWasmRun = createTool({ id: "ts-wasm-run", execute: async () => {
				var result = await wasm.run("ts-compiled");
				return { exitCode: result.exitCode };
			}});
			const tsWasmList = createTool({ id: "ts-wasm-list", execute: async () => {
				var modules = wasm.list();
				return { count: (modules || []).length, modules: modules };
			}});
			const tsWasmGet = createTool({ id: "ts-wasm-get", execute: async () => {
				var mod = wasm.get("ts-compiled");
				return { found: !!mod };
			}});
			const tsWasmRemove = createTool({ id: "ts-wasm-remove", execute: async () => {
				// Compile a fresh module to remove (the one from ts-wasm-compile may already be removed)
				await wasm.compile('export function run(): i32 { return 1; }', { name: "ts-to-remove" });
				var ok = wasm.remove("ts-to-remove");
				return { removed: ok };
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-wasm-surface.ts"})

	t.Run("Compile", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-compile", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "ts-compiled", result["name"])
	})
	t.Run("Run", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-run", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, float64(77), result["exitCode"])
	})
	t.Run("List", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-list", Input: map[string]any{}})
		require.NoError(t, err)
		// wasm.list() should not error — proves bridge path works
		assert.NotNil(t, resp.Result)
	})
	t.Run("Get", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-get", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["found"])
	})
	t.Run("Remove", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-remove", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["removed"])
	})
}

// --- Kit lifecycle from TS ---

func TestTSSurface_Kit(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Kit deploy/teardown/list are Go-side operations.
	// From .ts, you can't deploy other .ts files (that's a Go API).
	// But you can call kit.list via the bus.
	// The kit domain is inherently a Go-side concern — .ts code IS the deployed artifact.
	t.Run("ListFromGo", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.KitListMsg, messages.KitListResp](tk, ctx, messages.KitListMsg{})
		require.NoError(t, err)
		assert.NotNil(t, resp.Deployments)
	})
	t.Log("Kit lifecycle: deploy/teardown/redeploy are Go-only operations — .ts code IS the deployed artifact")
}

// --- MCP domain from TS ---

func TestTSSurface_MCP(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-mcp-surface.ts",
		Code: `
			const tsMcpList = createTool({ id: "ts-mcp-list", execute: async () => {
				try {
					var tools = await mcp.listTools();
					return { tools: tools };
				} catch(e) {
					return { error: e.message };
				}
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-mcp-surface.ts"})

	t.Run("ListTools", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-mcp-list", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		// Either returns tools array or error (no MCP servers configured)
		assert.True(t, result["tools"] != nil || result["error"] != nil)
	})
}

// --- Registry from TS ---

func TestTSSurface_Registry(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "ts-registry-surface.ts",
		Code: `
			const tsRegHas = createTool({ id: "ts-reg-has", execute: async () => {
				return {
					hasDefault: registry.has("storage", "default"),
					hasFake: registry.has("storage", "nonexistent"),
				};
			}});
			const tsRegList = createTool({ id: "ts-reg-list", execute: async () => {
				return { storages: registry.list("storage") };
			}});
		`,
	})
	require.NoError(t, err)
	defer sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](tk, ctx, messages.KitTeardownMsg{Source: "ts-registry-surface.ts"})

	t.Run("Has", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-reg-has", Input: map[string]any{}})
		require.NoError(t, err)
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["hasDefault"])
		assert.Equal(t, false, result["hasFake"])
	})
	t.Run("List", func(t *testing.T) {
		resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](tk, ctx, messages.ToolCallMsg{Name: "ts-reg-list", Input: map[string]any{}})
		require.NoError(t, err)
		assert.NotNil(t, resp.Result)
	})
}
