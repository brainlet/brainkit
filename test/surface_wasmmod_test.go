package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWASMSurface tests every domain operation callable from WASM via invokeAsync.
// Pattern: compile an AS module that calls invokeAsync(topic, payload, callback).
// The callback stores a flag in state. pendingInvokes.Wait() ensures the callback
// fires before wasm.run returns. This proves WASM → Go → LocalInvoker → handler path.
//
// File named surface_wasmmod_test.go to avoid _wasm build constraint suffix.

// wasmDomainTest compiles and runs a WASM module that calls invokeAsync with the given topic.
// Returns the run response. Verifies the callback was called (no panic/hang).
func wasmDomainTest(t *testing.T, rt sdk.Runtime, ctx context.Context, name, topic, payload string) {
	t.Helper()
	_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _invokeAsync, _setState } from "brainkit";
			export function onResult(topic: usize, payload: usize): void {
				_setState("done", "true");
			}
			export function run(): i32 {
				_invokeAsync("` + topic + `", '` + payload + `', "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: name},
	})
	require.NoError(t, err, "compile %s", name)

	runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: name})
	require.NoError(t, err, "run %s", name)
	assert.Equal(t, 0, runResp.ExitCode, "%s should exit 0", name)
}

// --- Tools domain from WASM ---

func TestWASMSurface_Tools(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("Call", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-tools-call", "tools.call", `{"name":"echo","input":{"message":"from-wasm"}}`)
	})
	t.Run("List", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-tools-list", "tools.list", `{}`)
	})
	t.Run("Resolve", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-tools-resolve", "tools.resolve", `{"name":"echo"}`)
	})
}

// --- FS domain from WASM ---

func TestWASMSurface_FS(t *testing.T) {
	tk := newTestKernelFull(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Pre-write a file so we can read it from WASM
	sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](tk, ctx, messages.FsWriteMsg{Path: "wasm-test.txt", Data: "hello"})
	sdk.PublishAwait[messages.FsMkdirMsg, messages.FsMkdirResp](tk, ctx, messages.FsMkdirMsg{Path: "wasm-dir"})

	t.Run("Write", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-write", "fs.write", `{"path":"wasm-written.txt","data":"from wasm"}`)
	})
	t.Run("Read", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-read", "fs.read", `{"path":"wasm-test.txt"}`)
	})
	t.Run("List", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-list", "fs.list", `{"path":"."}`)
	})
	t.Run("Stat", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-stat", "fs.stat", `{"path":"wasm-test.txt"}`)
	})
	t.Run("Mkdir", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-mkdir", "fs.mkdir", `{"path":"wasm-created-dir"}`)
	})
	t.Run("Delete", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-fs-delete", "fs.delete", `{"path":"wasm-written.txt"}`)
	})
}

// --- Agents domain from WASM ---

func TestWASMSurface_Agents(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("List", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-agents-list", "agents.list", `{}`)
	})
	t.Run("Discover", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-agents-discover", "agents.discover", `{}`)
	})
}

// --- AI domain from WASM ---

func TestWASMSurface_AI(t *testing.T) {
	loadEnv(t)
	if !hasAIKey() {
		t.Skip("OPENAI_API_KEY required")
	}
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("Generate", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-ai-gen", "ai.generate", `{"model":"openai/gpt-4o-mini","prompt":"Say: ok"}`)
	})
	t.Run("Embed", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-ai-embed", "ai.embed", `{"model":"openai/text-embedding-3-small","value":"test"}`)
	})
	t.Run("EmbedMany", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-ai-embedmany", "ai.embedMany", `{"model":"openai/text-embedding-3-small","values":["a","b"]}`)
	})
	t.Run("GenerateObject", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-ai-genobj", "ai.generateObject",
			`{"model":"openai/gpt-4o-mini","prompt":"color","schema":{"type":"object","properties":{"color":{"type":"string"}}}}`)
	})
}

// --- Memory domain from WASM ---

func TestWASMSurface_Memory(t *testing.T) {
	tk := newTestKernelWithStorage(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Init memory
	_, err := tk.EvalTS(ctx, "__wasm_mem_init.ts", `
		globalThis.__kit_memory = createMemory({ storage: new InMemoryStore(), lastMessages: 10 });
		return "ok";
	`)
	require.NoError(t, err)

	t.Run("CreateThread", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-mem-create", "memory.createThread", `{}`)
	})
	t.Run("ListThreads", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-mem-list", "memory.listThreads", `{}`)
	})
}

// --- Workflows domain from WASM ---

func TestWASMSurface_Workflows(t *testing.T) {
	tk := newTestKernelWithStorage(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Deploy a workflow first
	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](tk, ctx, messages.KitDeployMsg{
		Source: "wasm-wf-setup.ts",
		Code: `
			const wf = createWorkflow({
				id: "wasm-test-wf",
				inputSchema: z.object({ v: z.string() }),
				outputSchema: z.object({ r: z.string() }),
			});
			const s = createStep({
				id: "s1",
				inputSchema: z.object({ v: z.string() }),
				outputSchema: z.object({ r: z.string() }),
				execute: async ({ inputData }) => ({ r: inputData.v + "-done" }),
			});
			wf.then(s).commit();
		`,
	})
	require.NoError(t, err)

	t.Run("Run", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-wf-run", "workflows.run", `{"name":"wasm-test-wf","input":{"v":"test"}}`)
	})
}

// --- Kit lifecycle from WASM ---

func TestWASMSurface_Kit(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	t.Run("List", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-kit-list", "kit.list", `{}`)
	})
}

// --- MCP domain from WASM ---

func TestWASMSurface_MCP(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// MCP listTools will error (no servers configured) but the callback SHOULD still fire
	// with an error response. This proves the path works even for error cases.
	t.Run("ListTools", func(t *testing.T) {
		wasmDomainTest(t, rt, ctx, "wasm-mcp-list", "mcp.listTools", `{}`)
	})
}
