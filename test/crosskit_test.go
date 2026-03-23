package test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crossKitCall is a helper for typed cross-Kit PublishTo calls.
// Kit A calls an operation on Kit B's namespace.
func crossKitCall[Req messages.BrainkitMessage, Resp any](t *testing.T, kitA sdk.Runtime, ctx context.Context, targetNS string, req Req) Resp {
	t.Helper()
	_pr1, err := sdk.PublishTo(kitA, ctx, targetNS, req)
	require.NoError(t, err)
	_ch1 := make(chan Resp, 1)
	_us1, err := sdk.SubscribeTo[Resp](kitA, ctx, _pr1.ReplyTo, func(r Resp, m messages.Message) { _ch1 <- r })
	require.NoError(t, err)
	defer _us1()
	var resp Resp
	select {
	case resp = <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	return resp
}

// --- Raw pub/sub ---

func TestCrossKit_BasicRoundTrip(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			received := make(chan []byte, 1)
			unsub, err := kitB.SubscribeRaw(ctx, "crosskit.ping", func(msg messages.Message) {
				received <- msg.Payload
			})
			require.NoError(t, err)
			defer unsub()

			xrtA := kitA.(sdk.CrossNamespaceRuntime)
			_, err = xrtA.PublishRawTo(ctx, "kit-b", "crosskit.ping", json.RawMessage(`{"from":"kit-a"}`))
			require.NoError(t, err)

			select {
			case payload := <-received:
				var msg map[string]string
				json.Unmarshal(payload, &msg)
				assert.Equal(t, "kit-a", msg["from"])
			case <-ctx.Done():
				t.Fatal("timeout")
			}
		})
	}
}

// --- Tools domain cross-Kit ---

func TestCrossKit_Tools(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Call_A_to_B", func(t *testing.T) {
				resp := crossKitCall[messages.ToolCallMsg, messages.ToolCallResp](t, kitA, ctx, "kit-b", messages.ToolCallMsg{
					Name: "echo", Input: map[string]any{"message": "from-A"},
				})
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "from-A", result["echoed"])
				assert.Equal(t, "kit-b", result["from"])
			})

			t.Run("Call_B_to_A", func(t *testing.T) {
				resp := crossKitCall[messages.ToolCallMsg, messages.ToolCallResp](t, kitB, ctx, "kit-a", messages.ToolCallMsg{
					Name: "echo", Input: map[string]any{"message": "from-B"},
				})
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "from-B", result["echoed"])
				assert.Equal(t, "kit-a", result["from"])
			})

			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.ToolListMsg, messages.ToolListResp](t, kitA, ctx, "kit-b", messages.ToolListMsg{})
				found := false
				for _, tool := range resp.Tools {
					if tool.ShortName == "echo" {
						found = true
					}
				}
				assert.True(t, found, "Kit B's echo tool should be visible from Kit A")
			})
		})
	}
}

// --- FS domain cross-Kit ---

func TestCrossKit_FS(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Write on Kit B, read from Kit A via cross-Kit
			crossKitCall[messages.FsWriteMsg, messages.FsWriteResp](t, kitA, ctx, "kit-b", messages.FsWriteMsg{
				Path: "crosskit.txt", Data: "written-by-A-on-B",
			})

			resp := crossKitCall[messages.FsReadMsg, messages.FsReadResp](t, kitA, ctx, "kit-b", messages.FsReadMsg{Path: "crosskit.txt"})
			assert.Equal(t, "written-by-A-on-B", resp.Data)

			// Verify Kit B sees the file locally too
			_pr2, err := sdk.Publish(kitB, ctx, messages.FsReadMsg{Path: "crosskit.txt"})
			require.NoError(t, err)
			_ch2 := make(chan messages.FsReadResp, 1)
			_us2, err := sdk.SubscribeTo[messages.FsReadResp](kitB, ctx, _pr2.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch2 <- r })
			require.NoError(t, err)
			defer _us2()
			var localResp messages.FsReadResp
			select {
			case localResp = <-_ch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			assert.Equal(t, "written-by-A-on-B", localResp.Data)
		})
	}
}

// --- Agents domain cross-Kit ---

func TestCrossKit_Agents(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AgentListMsg, messages.AgentListResp](t, kitA, ctx, "kit-b", messages.AgentListMsg{})
				assert.NotNil(t, resp.Agents)
			})
			t.Run("Discover_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AgentDiscoverMsg, messages.AgentDiscoverResp](t, kitA, ctx, "kit-b", messages.AgentDiscoverMsg{})
				assert.NotNil(t, resp.Agents)
			})
		})
	}
}

// --- Kit lifecycle cross-Kit ---

func TestCrossKit_Kit(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Deploy_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitDeployMsg, messages.KitDeployResp](t, kitA, ctx, "kit-b", messages.KitDeployMsg{
					Source: "crosskit-deploy.ts",
					Code:   `var x = 1;`,
				})
				assert.True(t, resp.Deployed)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitListMsg, messages.KitListResp](t, kitA, ctx, "kit-b", messages.KitListMsg{})
				assert.NotNil(t, resp.Deployments)
			})
			t.Run("Teardown_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitTeardownMsg, messages.KitTeardownResp](t, kitA, ctx, "kit-b", messages.KitTeardownMsg{
					Source: "crosskit-deploy.ts",
				})
				assert.GreaterOrEqual(t, resp.Removed, 0)
			})
		})
	}
}

// --- WASM domain cross-Kit ---

func TestCrossKit_WASM(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			t.Run("Compile_Run_Remote", func(t *testing.T) {
				comp := crossKitCall[messages.WasmCompileMsg, messages.WasmCompileResp](t, kitA, ctx, "kit-b", messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 88; }`,
					Options: &messages.WasmCompileOpts{Name: "crosskit-mod"},
				})
				assert.Greater(t, comp.Size, 0)

				run := crossKitCall[messages.WasmRunMsg, messages.WasmRunResp](t, kitA, ctx, "kit-b", messages.WasmRunMsg{
					ModuleID: "crosskit-mod",
				})
				assert.Equal(t, 88, run.ExitCode)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.WasmListMsg, messages.WasmListResp](t, kitA, ctx, "kit-b", messages.WasmListMsg{})
				assert.NotEmpty(t, resp.Modules)
			})
		})
	}
}

// --- Registry cross-Kit ---

func TestCrossKit_Registry(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Has_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.RegistryHasMsg, messages.RegistryHasResp](t, kitA, ctx, "kit-b", messages.RegistryHasMsg{
					Category: "provider", Name: "nonexistent",
				})
				assert.False(t, resp.Found)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.RegistryListMsg, messages.RegistryListResp](t, kitA, ctx, "kit-b", messages.RegistryListMsg{
					Category: "provider",
				})
				assert.NotNil(t, resp.Items)
			})
		})
	}
}

// --- AI domain cross-Kit ---

func TestCrossKit_AI(t *testing.T) {
	loadEnv(t)
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required")
	}
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPairFull(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			t.Run("Generate_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AiGenerateMsg, messages.AiGenerateResp](t, kitA, ctx, "kit-b", messages.AiGenerateMsg{
					Model: "openai/gpt-4o-mini", Prompt: "Say exactly: pong",
				})
				assert.NotEmpty(t, resp.Text)
			})
			t.Run("Embed_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AiEmbedMsg, messages.AiEmbedResp](t, kitA, ctx, "kit-b", messages.AiEmbedMsg{
					Model: "openai/text-embedding-3-small", Value: "test",
				})
				assert.NotEmpty(t, resp.Embedding)
			})
		})
	}
}

// --- Memory domain cross-Kit ---

func TestCrossKit_Memory(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPairFull(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Init memory on Kit B so cross-Kit memory calls have a target
			_, err := kitB.EvalTS(ctx, "__crosskit_mem_init.ts", `
				globalThis.__kit_memory = createMemory({ storage: new InMemoryStore(), lastMessages: 10 });
				return "ok";
			`)
			require.NoError(t, err)

			t.Run("CreateThread_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](t, kitA, ctx, "kit-b", messages.MemoryCreateThreadMsg{})
				assert.NotEmpty(t, resp.ThreadID)
			})
			t.Run("Save_Get_Remote", func(t *testing.T) {
				create := crossKitCall[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](t, kitA, ctx, "kit-b", messages.MemoryCreateThreadMsg{})
				crossKitCall[messages.MemorySaveMsg, messages.MemorySaveResp](t, kitA, ctx, "kit-b", messages.MemorySaveMsg{
					ThreadID: create.ThreadID,
					Messages: []messages.MemoryMessage{{Role: "user", Content: "cross-kit msg"}},
				})
				get := crossKitCall[messages.MemoryGetThreadMsg, messages.MemoryGetThreadResp](t, kitA, ctx, "kit-b", messages.MemoryGetThreadMsg{
					ThreadID: create.ThreadID,
				})
				assert.NotNil(t, get.Thread)
			})
			t.Run("ListThreads_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.MemoryListThreadsMsg, messages.MemoryListThreadsResp](t, kitA, ctx, "kit-b", messages.MemoryListThreadsMsg{})
				assert.NotNil(t, resp.Threads)
			})
			t.Run("DeleteThread_Remote", func(t *testing.T) {
				create := crossKitCall[messages.MemoryCreateThreadMsg, messages.MemoryCreateThreadResp](t, kitA, ctx, "kit-b", messages.MemoryCreateThreadMsg{})
				resp := crossKitCall[messages.MemoryDeleteThreadMsg, messages.MemoryDeleteThreadResp](t, kitA, ctx, "kit-b", messages.MemoryDeleteThreadMsg{
					ThreadID: create.ThreadID,
				})
				assert.True(t, resp.OK)
			})
		})
	}
}

// --- Workflows domain cross-Kit ---

func TestCrossKit_Workflows(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPairFull(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Deploy workflow on Kit B
			_pr3, err := sdk.Publish(kitB, ctx, messages.KitDeployMsg{
				Source: "crosskit-wf.ts",
				Code: `
					const wf = createWorkflow({
						id: "crosskit-wf", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
					});
					wf.then(createStep({
						id: "s1", inputSchema: z.object({ v: z.string() }), outputSchema: z.object({ r: z.string() }),
						execute: async ({ inputData }) => ({ r: inputData.v + "-crosskit" }),
					})).commit();
				`,
			})
			require.NoError(t, err)
			_ch3 := make(chan messages.KitDeployResp, 1)
			_us3, _ := sdk.SubscribeTo[messages.KitDeployResp](kitA, ctx, _pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch3 <- r })
			defer _us3()
			select {
			case <-_ch3:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			t.Run("Run_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.WorkflowRunMsg, messages.WorkflowRunResp](t, kitA, ctx, "kit-b", messages.WorkflowRunMsg{
					Name: "crosskit-wf", Input: map[string]any{"v": "hello"},
				})
				assert.NotNil(t, resp.Result)
				var result map[string]any
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "success", result["status"])
			})

			_spr1, _ := sdk.Publish(kitB, ctx, messages.KitTeardownMsg{Source: "crosskit-wf.ts"})
			_sch1 := make(chan messages.KitTeardownResp, 1)
			_sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](kitB, ctx, _spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch1 <- r })
			defer _sun1()
			select { case <-_sch1: case <-ctx.Done(): t.Fatal("timeout") }
		})
	}
}

// --- MCP domain cross-Kit ---

func TestCrossKit_MCP(t *testing.T) {
	mcpBinary := buildTestMCP(t)
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			loadEnv(t)
			cfg := transportConfigForBackend(t, backend)
			transport := mustCreateTransport(t, cfg)
			t.Cleanup(func() { transport.Close() })

			// Kit B has MCP server, Kit A calls remotely
			tmpA := t.TempDir()
			kitA, err := kit.NewKernel(kit.KernelConfig{
				Namespace: "kit-a", CallerID: "kit-a-caller", WorkspaceDir: tmpA, Transport: transport,
			})
			require.NoError(t, err)
			t.Cleanup(func() { kitA.Close() })

			tmpB := t.TempDir()
			kitB, err := kit.NewKernel(kit.KernelConfig{
				Namespace: "kit-b", CallerID: "kit-b-caller", WorkspaceDir: tmpB, Transport: transport,
				MCPServers: map[string]mcppkg.ServerConfig{"echo": {Command: mcpBinary}},
			})
			require.NoError(t, err)
			t.Cleanup(func() { kitB.Close() })

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("ListTools_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.McpListToolsMsg, messages.McpListToolsResp](t, kitA, ctx, "kit-b", messages.McpListToolsMsg{})
				assert.NotEmpty(t, resp.Tools)
			})
			t.Run("CallTool_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.McpCallToolMsg, messages.McpCallToolResp](t, kitA, ctx, "kit-b", messages.McpCallToolMsg{
					Server: "echo", Tool: "echo", Args: map[string]any{"message": "cross-kit-mcp"},
				})
				assert.NotNil(t, resp.Result)
			})
		})
	}
}

// --- Vectors domain cross-Kit ---

func TestCrossKit_Vectors(t *testing.T) {
	if !podmanAvailable() {
		t.Skip("Podman required for pgvector")
	}
	pgConnStr := startPgVectorContainer(t)

	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPairFull(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Init PgVector on Kit B
			_, err := kitB.EvalTS(ctx, "__crosskit_vec_init.ts", fmt.Sprintf(`
				var vs = new PgVector({ id: "crosskit_vec", connectionString: %q });
				globalThis.__kit_vector_store = vs;
				return "ok";
			`, pgConnStr))
			require.NoError(t, err)

			idxName := "crosskit_" + sanitizeIdent(backend)

			t.Run("CreateIndex_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.VectorCreateIndexMsg, messages.VectorCreateIndexResp](t, kitA, ctx, "kit-b", messages.VectorCreateIndexMsg{
					Name: idxName, Dimension: 3, Metric: "cosine",
				})
				assert.True(t, resp.OK)
			})
		})
	}
}
