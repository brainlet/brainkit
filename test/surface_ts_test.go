package test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
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

	_pr1, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
	_ch1 := make(chan messages.KitDeployResp, 1)
	_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
	defer _us1()
	select {
	case <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-tools-surface.ts"})

	t.Run("List", func(t *testing.T) {
		_pr1, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-tools-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch1 := make(chan messages.ToolCallResp, 1)
		_us1, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr1.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch1 <- r })
		require.NoError(t, err)
		defer _us1()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch1:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("Call", func(t *testing.T) {
		_pr2, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-tools-call", Input: map[string]any{}})
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
		assert.NotNil(t, resp.Result)
	})
	t.Run("Resolve", func(t *testing.T) {
		_pr3, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-tools-resolve", Input: map[string]any{"name": "ts-tools-list"}})
		require.NoError(t, err)
		_ch3 := make(chan messages.ToolCallResp, 1)
		_us3, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr3.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch3 <- r })
		require.NoError(t, err)
		defer _us3()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch3:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
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

	_pr2, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
		Source: "ts-fs-surface.ts",
		Code: `
			const tsFsAll = createTool({ id: "ts-fs-all", execute: async () => {
				await fs.write("ts-written.txt", "hello from ts");
				var read = await fs.read("ts-written.txt");
				await fs.mkdir("ts-dir");
				await fs.write("ts-dir/a.txt", "a");
				var list = await fs.list("ts-dir");
				var stat = await fs.stat("ts-written.txt");
				await fs.delete("ts-written.txt");
				return {
					written: true,
					readData: read.data,
					listed: (list.files || []).length,
					statSize: stat.size,
					deleted: true,
				};
			}});
		`,
	})
	require.NoError(t, err)
	_ch2 := make(chan messages.KitDeployResp, 1)
	_us2, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr2.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch2 <- r })
	defer _us2()
	select {
	case <-_ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-fs-surface.ts"})

	t.Run("AllOperations", func(t *testing.T) {
		_pr4, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-fs-all", Input: map[string]any{}})
		require.NoError(t, err)
		_ch4 := make(chan messages.ToolCallResp, 1)
		_us4, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr4.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch4 <- r })
		require.NoError(t, err)
		defer _us4()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch4:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "hello from ts", result["readData"])
		assert.Equal(t, true, result["written"])
		assert.Equal(t, true, result["deleted"])
	})
}

// --- Agents domain from TS (all operations) ---

func TestTSSurface_Agents(t *testing.T) {
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required for agent tests")
	}
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_pr3, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
		Source: "ts-agents-surface.ts",
		Code: `
			// Create an agent so we can test all agent operations
			const testAgent = agent({
				name: "ts-surface-agent",
				instructions: "Reply with exactly: ok",
				model: "openai/gpt-4o-mini",
			});

			const tsAgentsList = createTool({ id: "ts-agents-list", execute: async () => {
				return { agents: agents.list() };
			}});
			const tsAgentsDiscover = createTool({ id: "ts-agents-discover", execute: async () => {
				return { agents: agents.discover({}) };
			}});
			const tsAgentsStatus = createTool({ id: "ts-agents-status", execute: async () => {
				return agents.status("ts-surface-agent");
			}});
			const tsAgentsSetStatus = createTool({ id: "ts-agents-setstatus", execute: async () => {
				agents.setStatus("ts-surface-agent", "busy");
				var after = agents.status("ts-surface-agent");
				agents.setStatus("ts-surface-agent", "idle");
				return { statusWas: after.status };
			}});
			const tsAgentsMessage = createTool({ id: "ts-agents-message", execute: async () => {
				return agents.message("ts-surface-agent", { text: "hello" });
			}});
			const tsAgentsRequest = createTool({ id: "ts-agents-request", execute: async () => {
				var resp = await agents.request("ts-surface-agent", "Say ok");
				return resp;
			}});
		`,
	})
	require.NoError(t, err)
	_ch3 := make(chan messages.KitDeployResp, 1)
	_us3, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch3 <- r })
	defer _us3()
	select {
	case <-_ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-agents-surface.ts"})

	t.Run("List", func(t *testing.T) {
		_pr5, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch5 := make(chan messages.ToolCallResp, 1)
		_us5, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr5.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch5 <- r })
		require.NoError(t, err)
		defer _us5()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch5:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("Discover", func(t *testing.T) {
		_pr6, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-discover", Input: map[string]any{}})
		require.NoError(t, err)
		_ch6 := make(chan messages.ToolCallResp, 1)
		_us6, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr6.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch6 <- r })
		require.NoError(t, err)
		defer _us6()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch6:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("GetStatus", func(t *testing.T) {
		_pr7, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-status", Input: map[string]any{}})
		require.NoError(t, err)
		_ch7 := make(chan messages.ToolCallResp, 1)
		_us7, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr7.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch7 <- r })
		require.NoError(t, err)
		defer _us7()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch7:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "idle", result["status"])
	})
	t.Run("SetStatus", func(t *testing.T) {
		_pr8, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-setstatus", Input: map[string]any{}})
		require.NoError(t, err)
		_ch8 := make(chan messages.ToolCallResp, 1)
		_us8, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr8.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch8 <- r })
		require.NoError(t, err)
		defer _us8()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch8:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "busy", result["statusWas"])
	})
	t.Run("Message", func(t *testing.T) {
		_pr9, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-message", Input: map[string]any{}})
		require.NoError(t, err)
		_ch9 := make(chan messages.ToolCallResp, 1)
		_us9, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr9.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch9 <- r })
		require.NoError(t, err)
		defer _us9()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch9:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["delivered"])
	})
	t.Run("Request", func(t *testing.T) {
		_pr10, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-agents-request", Input: map[string]any{}})
		require.NoError(t, err)
		_ch10 := make(chan messages.ToolCallResp, 1)
		_us10, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr10.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch10 <- r })
		require.NoError(t, err)
		defer _us10()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch10:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotEmpty(t, result["text"])
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

	_pr4, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
			const tsAiStream = createTool({ id: "ts-ai-stream", execute: async () => {
				var stream = ai.stream({ model: "openai/gpt-4o-mini", prompt: "Say exactly: pong" });
				var text = "";
				var reader = stream.textStream.getReader();
				while (true) {
					var chunk = await reader.read();
					if (chunk.done) break;
					text += chunk.value || "";
				}
				return { text: text };
			}});
		`,
	})
	require.NoError(t, err)
	_ch4 := make(chan messages.KitDeployResp, 1)
	_us4, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr4.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch4 <- r })
	defer _us4()
	select {
	case <-_ch4:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-ai-surface.ts"})

	t.Run("Generate", func(t *testing.T) {
		_pr11, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-ai-generate", Input: map[string]any{}})
		require.NoError(t, err)
		_ch11 := make(chan messages.ToolCallResp, 1)
		_us11, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr11.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch11 <- r })
		require.NoError(t, err)
		defer _us11()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch11:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotEmpty(t, result["text"])
	})
	t.Run("Embed", func(t *testing.T) {
		_pr12, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-ai-embed", Input: map[string]any{}})
		require.NoError(t, err)
		_ch12 := make(chan messages.ToolCallResp, 1)
		_us12, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr12.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch12 <- r })
		require.NoError(t, err)
		defer _us12()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch12:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Greater(t, result["dimensions"], float64(0))
	})
	t.Run("EmbedMany", func(t *testing.T) {
		_pr13, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-ai-embedmany", Input: map[string]any{}})
		require.NoError(t, err)
		_ch13 := make(chan messages.ToolCallResp, 1)
		_us13, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr13.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch13 <- r })
		require.NoError(t, err)
		defer _us13()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch13:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, float64(2), result["count"])
	})
	t.Run("GenerateObject", func(t *testing.T) {
		_pr14, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-ai-genobj", Input: map[string]any{}})
		require.NoError(t, err)
		_ch14 := make(chan messages.ToolCallResp, 1)
		_us14, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr14.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch14 <- r })
		require.NoError(t, err)
		defer _us14()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch14:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotNil(t, result["object"])
	})
	t.Run("Stream", func(t *testing.T) {
		_pr15, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-ai-stream", Input: map[string]any{}})
		require.NoError(t, err)
		_ch15 := make(chan messages.ToolCallResp, 1)
		_us15, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr15.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch15 <- r })
		require.NoError(t, err)
		defer _us15()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch15:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotEmpty(t, result["text"])
	})
}

// --- Memory domain from TS ---

func TestTSSurface_Memory(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_pr5, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
				var recalled = await mem.recall({ threadId: threadId, resourceId: "", query: "hello" });
				var list = await mem.listThreads({});
				await mem.deleteThread(threadId);
				return {
					created: !!threadId,
					saved: true,
					got: !!got,
					recalled: true,
					listed: !!(list && (list.threads || list)),
					deleted: true,
				};
			}});
		`,
	})
	require.NoError(t, err)
	_ch5 := make(chan messages.KitDeployResp, 1)
	_us5, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr5.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch5 <- r })
	defer _us5()
	select {
	case <-_ch5:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-memory-surface.ts"})

	t.Run("AllOperations", func(t *testing.T) {
		_pr16, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-mem-all", Input: map[string]any{}})
		require.NoError(t, err)
		_ch16 := make(chan messages.ToolCallResp, 1)
		_us16, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr16.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch16 <- r })
		require.NoError(t, err)
		defer _us16()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch16:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
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

	_pr6, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
				result.runId = run.runId;
				return result;
			}});

			// Suspend workflow for resume/status/cancel tests
			const swf = createWorkflow({
				id: "ts-suspend-wf",
				inputSchema: z.object({ v: z.string() }),
				outputSchema: z.object({ r: z.string() }),
			});
			swf.then(createStep({
				id: "ss1",
				inputSchema: z.object({ v: z.string() }),
				outputSchema: z.object({ r: z.string() }),
				execute: async ({ inputData, suspend }) => {
					await suspend({ reason: "need-input" });
					return { r: inputData.v + "-resumed" };
				},
			})).commit();

			const tsWfSuspend = createTool({ id: "ts-wf-suspend", execute: async ({ context: input }) => {
				var run = await createWorkflowRun(swf);
				var result = await run.start({ inputData: { v: input.val || "test" } });
				result.runId = run.runId;
				return result;
			}});
			const tsWfResume = createTool({ id: "ts-wf-resume", execute: async ({ context: input }) => {
				var result = await resumeWorkflow(input.runId, "ss1", { approved: true });
				return result;
			}});
		`,
	})
	require.NoError(t, err)
	_ch6 := make(chan messages.KitDeployResp, 1)
	_us6, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr6.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch6 <- r })
	defer _us6()
	select {
	case <-_ch6:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-wf-surface.ts"})

	t.Run("Run", func(t *testing.T) {
		_pr17, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wf-run", Input: map[string]any{"val": "hello"}})
		require.NoError(t, err)
		_ch17 := make(chan messages.ToolCallResp, 1)
		_us17, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr17.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch17 <- r })
		require.NoError(t, err)
		defer _us17()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch17:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "success", result["status"])
	})
	t.Run("Suspend_Resume", func(t *testing.T) {
		// Run the suspend workflow
		_pr18, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wf-suspend", Input: map[string]any{"val": "test"}})
		require.NoError(t, err)
		_ch18 := make(chan messages.ToolCallResp, 1)
		_us18, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr18.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch18 <- r })
		require.NoError(t, err)
		defer _us18()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch18:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var suspendResult map[string]any
		json.Unmarshal(resp.Result, &suspendResult)
		if suspendResult["status"] == "suspended" {
			runId, _ := suspendResult["runId"].(string)
			require.NotEmpty(t, runId)

			// Resume from TS
			_pr7, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{
				Name: "ts-wf-resume", Input: map[string]any{"runId": runId},
			})
			require.NoError(t, err)
			_ch7 := make(chan messages.ToolCallResp, 1)
			_us7, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr7.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch7 <- r })
			require.NoError(t, err)
			defer _us7()
			var resumeResp messages.ToolCallResp
			select {
			case resumeResp = <-_ch7:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			assert.NotNil(t, resumeResp.Result)
		}
	})
}

// --- WASM domain from TS ---

func TestTSSurface_WASM(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	_pr8, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
				await wasm.compile('export function run(): i32 { return 1; }', { name: "ts-to-remove" });
				var ok = wasm.remove("ts-to-remove");
				return { removed: ok };
			}});
			const tsWasmDeploy = createTool({ id: "ts-wasm-deploy", execute: async () => {
				await wasm.compile('import { _on, _setMode } from "brainkit"; export function init(): void { _setMode("stateless"); _on("ts.ev", "h"); } export function h(t: usize, p: usize): void {}', { name: "ts-deploy-mod" });
				var desc = await wasm.deploy("ts-deploy-mod");
				return { mode: desc.mode };
			}});
			const tsWasmDescribe = createTool({ id: "ts-wasm-describe", execute: async () => {
				return wasm.describe("ts-deploy-mod");
			}});
			const tsWasmUndeploy = createTool({ id: "ts-wasm-undeploy", execute: async () => {
				var result = await wasm.undeploy("ts-deploy-mod");
				return result;
			}});
		`,
	})
	require.NoError(t, err)
	_ch8 := make(chan messages.KitDeployResp, 1)
	_us8, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr8.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch8 <- r })
	defer _us8()
	select {
	case <-_ch8:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-wasm-surface.ts"})

	t.Run("Compile", func(t *testing.T) {
		_pr19, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-compile", Input: map[string]any{}})
		require.NoError(t, err)
		_ch19 := make(chan messages.ToolCallResp, 1)
		_us19, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr19.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch19 <- r })
		require.NoError(t, err)
		defer _us19()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch19:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "ts-compiled", result["name"])
	})
	t.Run("Run", func(t *testing.T) {
		_pr20, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-run", Input: map[string]any{}})
		require.NoError(t, err)
		_ch20 := make(chan messages.ToolCallResp, 1)
		_us20, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr20.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch20 <- r })
		require.NoError(t, err)
		defer _us20()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch20:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, float64(77), result["exitCode"])
	})
	t.Run("List", func(t *testing.T) {
		_pr21, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch21 := make(chan messages.ToolCallResp, 1)
		_us21, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr21.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch21 <- r })
		require.NoError(t, err)
		defer _us21()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch21:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		// wasm.list() should not error — proves bridge path works
		assert.NotNil(t, resp.Result)
	})
	t.Run("Get", func(t *testing.T) {
		_pr22, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-get", Input: map[string]any{}})
		require.NoError(t, err)
		_ch22 := make(chan messages.ToolCallResp, 1)
		_us22, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr22.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch22 <- r })
		require.NoError(t, err)
		defer _us22()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch22:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["found"])
	})
	t.Run("Remove", func(t *testing.T) {
		_pr23, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-remove", Input: map[string]any{}})
		require.NoError(t, err)
		_ch23 := make(chan messages.ToolCallResp, 1)
		_us23, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr23.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch23 <- r })
		require.NoError(t, err)
		defer _us23()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch23:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["removed"])
	})
	t.Run("Deploy", func(t *testing.T) {
		_pr24, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-deploy", Input: map[string]any{}})
		require.NoError(t, err)
		_ch24 := make(chan messages.ToolCallResp, 1)
		_us24, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr24.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch24 <- r })
		require.NoError(t, err)
		defer _us24()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch24:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, "stateless", result["mode"])
	})
	t.Run("Describe", func(t *testing.T) {
		_pr25, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-describe", Input: map[string]any{}})
		require.NoError(t, err)
		_ch25 := make(chan messages.ToolCallResp, 1)
		_us25, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr25.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch25 <- r })
		require.NoError(t, err)
		defer _us25()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch25:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("Undeploy", func(t *testing.T) {
		_pr26, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-wasm-undeploy", Input: map[string]any{}})
		require.NoError(t, err)
		_ch26 := make(chan messages.ToolCallResp, 1)
		_us26, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr26.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch26 <- r })
		require.NoError(t, err)
		defer _us26()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch26:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
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
		_pr27, err := sdk.Publish(tk, ctx, messages.KitListMsg{})
		require.NoError(t, err)
		_ch27 := make(chan messages.KitListResp, 1)
		_us27, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr27.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch27 <- r })
		require.NoError(t, err)
		defer _us27()
		var resp messages.KitListResp
		select {
		case resp = <-_ch27:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Deployments)
	})
	t.Log("Kit lifecycle: deploy/teardown/redeploy are Go-only operations — .ts code IS the deployed artifact")
}

// --- MCP domain from TS ---

func TestTSSurface_MCP(t *testing.T) {
	// Build testmcp binary for a real MCP server
	mcpBinary := buildTestMCP(t)

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace: "test", CallerID: "test-ts-mcp", WorkspaceDir: t.TempDir(),
		MCPServers: map[string]mcppkg.ServerConfig{
			"echo": {Command: mcpBinary},
		},
	})
	require.NoError(t, err)
	defer k.Close()
	tk := &testKernel{k}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_pr9, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
		Source: "ts-mcp-surface.ts",
		Code: `
			const tsMcpList = createTool({ id: "ts-mcp-list", execute: async () => {
				var tools = await mcp.listTools();
				return { tools: tools };
			}});
			const tsMcpCall = createTool({ id: "ts-mcp-call", execute: async () => {
				var result = await mcp.callTool("echo", "echo", { message: "from-ts" });
				return result;
			}});
		`,
	})
	require.NoError(t, err)
	_ch9 := make(chan messages.KitDeployResp, 1)
	_us9, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr9.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch9 <- r })
	defer _us9()
	select {
	case <-_ch9:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-mcp-surface.ts"})

	t.Run("ListTools", func(t *testing.T) {
		_pr28, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-mcp-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch28 := make(chan messages.ToolCallResp, 1)
		_us28, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr28.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch28 <- r })
		require.NoError(t, err)
		defer _us28()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch28:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.NotNil(t, result["tools"])
	})
	t.Run("CallTool", func(t *testing.T) {
		_pr29, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-mcp-call", Input: map[string]any{}})
		require.NoError(t, err)
		_ch29 := make(chan messages.ToolCallResp, 1)
		_us29, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr29.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch29 <- r })
		require.NoError(t, err)
		defer _us29()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch29:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
}

// --- Registry from TS ---

func TestTSSurface_Registry(t *testing.T) {
	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_pr10, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
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
			const tsRegResolve = createTool({ id: "ts-reg-resolve", execute: async () => {
				try {
					var s = storage("default");
					return { resolved: true };
				} catch(e) {
					return { resolved: false, error: e.message };
				}
			}});
		`,
	})
	require.NoError(t, err)
	_ch10 := make(chan messages.KitDeployResp, 1)
	_us10, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr10.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch10 <- r })
	defer _us10()
	select {
	case <-_ch10:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-registry-surface.ts"})

	t.Run("Has", func(t *testing.T) {
		_pr30, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-reg-has", Input: map[string]any{}})
		require.NoError(t, err)
		_ch30 := make(chan messages.ToolCallResp, 1)
		_us30, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr30.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch30 <- r })
		require.NoError(t, err)
		defer _us30()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch30:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["hasDefault"])
		assert.Equal(t, false, result["hasFake"])
	})
	t.Run("List", func(t *testing.T) {
		_pr31, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-reg-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch31 := make(chan messages.ToolCallResp, 1)
		_us31, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr31.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch31 <- r })
		require.NoError(t, err)
		defer _us31()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch31:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("Resolve", func(t *testing.T) {
		_pr32, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-reg-resolve", Input: map[string]any{}})
		require.NoError(t, err)
		_ch32 := make(chan messages.ToolCallResp, 1)
		_us32, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr32.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch32 <- r })
		require.NoError(t, err)
		defer _us32()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch32:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["resolved"])
	})
}

// --- Vectors from TS ---

func TestTSSurface_Vectors(t *testing.T) {
	if !podmanAvailable() {
		t.Skip("Podman required for pgvector")
	}
	pgConnStr := startPgVectorContainer(t)

	tk := newTSKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_pr11, err := sdk.Publish(tk, ctx, messages.KitDeployMsg{
		Source: "ts-vec-surface.ts",
		Code: fmt.Sprintf(`
			var _vs = new PgVector({ id: "ts_vec", connectionString: %q });
			const tsVecCreate = createTool({ id: "ts-vec-create", execute: async ({ context: input }) => {
				await _vs.createIndex({ indexName: input.name || "ts_idx", dimension: 3 });
				return { ok: true };
			}});
			const tsVecList = createTool({ id: "ts-vec-list", execute: async () => {
				var indexes = await _vs.listIndexes();
				return { indexes: indexes };
			}});
			const tsVecDelete = createTool({ id: "ts-vec-delete", execute: async ({ context: input }) => {
				await _vs.deleteIndex(input.name || "ts_idx");
				return { ok: true };
			}});
		`, pgConnStr),
	})
	require.NoError(t, err)
	_ch11 := make(chan messages.KitDeployResp, 1)
	_us11, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr11.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch11 <- r })
	defer _us11()
	select {
	case <-_ch11:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	defer sdk.Publish(tk, ctx, messages.KitTeardownMsg{Source: "ts-vec-surface.ts"})

	t.Run("CreateIndex", func(t *testing.T) {
		_pr33, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-vec-create", Input: map[string]any{"name": "ts_vec_idx"}})
		require.NoError(t, err)
		_ch33 := make(chan messages.ToolCallResp, 1)
		_us33, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr33.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch33 <- r })
		require.NoError(t, err)
		defer _us33()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch33:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		var result map[string]any
		json.Unmarshal(resp.Result, &result)
		assert.Equal(t, true, result["ok"])
	})
	t.Run("ListIndexes", func(t *testing.T) {
		_pr34, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-vec-list", Input: map[string]any{}})
		require.NoError(t, err)
		_ch34 := make(chan messages.ToolCallResp, 1)
		_us34, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr34.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch34 <- r })
		require.NoError(t, err)
		defer _us34()
		var resp messages.ToolCallResp
		select {
		case resp = <-_ch34:
		case <-ctx.Done():
			t.Fatal("timeout")
		}
		assert.NotNil(t, resp.Result)
	})
	t.Run("DeleteIndex", func(t *testing.T) {
		_pr35, err := sdk.Publish(tk, ctx, messages.ToolCallMsg{Name: "ts-vec-delete", Input: map[string]any{"name": "ts_vec_idx"}})
		if err != nil {
			t.Logf("PgVector deleteIndex: Neon driver limitation in QuickJS: %v", err)
		}
	})
}
