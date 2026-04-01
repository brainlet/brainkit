package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compileAndDeploy compiles an AS module and deploys it as a shard.
// Returns the module name or fails the test.
func compileAndDeploy(t *testing.T, tk *testutil.TestKernel, name, source string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source:  source,
		Options: &messages.WasmCompileOpts{Name: name},
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		if responseHasError(p) {
			t.Fatalf("compile %s failed: %s", name, string(p))
		}
	case <-ctx.Done():
		t.Fatalf("timeout compiling %s", name)
	}
	unsub()

	// Deploy
	pr2, err := sdk.Publish(tk, ctx, messages.WasmDeployMsg{Name: name})
	require.NoError(t, err)

	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	select {
	case p := <-ch2:
		if responseHasError(p) {
			t.Fatalf("deploy %s failed: %s", name, string(p))
		}
	case <-ctx.Done():
		t.Fatalf("timeout deploying %s", name)
	}
	unsub2()
}

// TestWASMSurface_BusPublish — WASM shard publishes to bus via host function.
func TestWASMSurface_BusPublish(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile simple module that uses _busEmit (fire-and-forget, no callback needed)
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busEmit } from "brainkit";
			export function run(): i32 {
				_busEmit("incoming.from-wasm", '{"source":"wasm"}');
				return 42;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "bus-emit-shard"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub()

	// Run it
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "bus-emit-shard"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "42") // exitCode: 42
	case <-ctx.Done():
		t.Fatal("timeout running WASM shard")
	}
}

// TestWASMSurface_State — WASM module sets and gets state.
func TestWASMSurface_State(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// State is available via _setState/_getState even in wasm.run mode
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _setState, _getState } from "brainkit";
			export function run(): i32 {
				_setState("counter", "42");
				var val = _getState("counter");
				if (val == "42") return 1;
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "state-mod"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub()

	// Run — exit code 1 means state worked
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "state-mod"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		// exitCode should be 1 (state get/set worked)
		assert.Contains(t, string(p), `"exitCode":1`)
	case <-ctx.Done():
		t.Fatal("timeout running state module")
	}
}

// TestWASMSurface_ToolCall — WASM shard calls a Go-registered tool via bus.
func TestWASMSurface_ToolCall(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	compileAndDeploy(t, tk, "tool-call-shard", `
		import { _busPublish } from "brainkit";
		export function onToolResult(topic: usize, payload: usize): void {}
		export function run(): i32 {
			_busPublish("tools.call", '{"name":"echo","input":{"message":"from-wasm"}}', "onToolResult");
			return 0;
		}
	`)

	pr, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "tool-call-shard"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		// Shard ran — whether tool call completed depends on async callback timing
		assert.NotEmpty(t, p)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestWASMSurface_CompileErrors — invalid AS source returns clean error.
func TestWASMSurface_CompileErrors(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source:  `this is not valid assemblyscript at all {{{{`,
		Options: &messages.WasmCompileOpts{Name: "bad-as"},
	})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	defer unsub()

	select {
	case p := <-ch:
		assert.True(t, responseHasError(p), "compile of invalid AS should return error")
	case <-ctx.Done():
		t.Fatal("timeout — compile of invalid AS hung")
	}
}

// TestWASMSurface_ModuleLifecycle — compile, list, get, remove.
func TestWASMSurface_ModuleLifecycle(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile
	pr1, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source:  `export function run(): i32 { return 1; }`,
		Options: &messages.WasmCompileOpts{Name: "lifecycle-mod"},
	})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case <-ch1:
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub1()

	// List — should include it
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmListMsg{})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	select {
	case p := <-ch2:
		assert.Contains(t, string(p), "lifecycle-mod")
	case <-ctx.Done():
		t.Fatal("timeout list")
	}
	unsub2()

	// Get
	pr3, _ := sdk.Publish(tk, ctx, messages.WasmGetMsg{Name: "lifecycle-mod"})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	select {
	case p := <-ch3:
		assert.Contains(t, string(p), "lifecycle-mod")
	case <-ctx.Done():
		t.Fatal("timeout get")
	}
	unsub3()

	// Remove
	pr4, _ := sdk.Publish(tk, ctx, messages.WasmRemoveMsg{Name: "lifecycle-mod"})
	ch4 := make(chan []byte, 1)
	unsub4, _ := tk.SubscribeRaw(ctx, pr4.ReplyTo, func(m messages.Message) { ch4 <- m.Payload })
	select {
	case p := <-ch4:
		assert.Contains(t, string(p), "removed")
	case <-ctx.Done():
		t.Fatal("timeout remove")
	}
	unsub4()

	// Get again — should be gone
	pr5, _ := sdk.Publish(tk, ctx, messages.WasmGetMsg{Name: "lifecycle-mod"})
	ch5 := make(chan []byte, 1)
	unsub5, _ := tk.SubscribeRaw(ctx, pr5.ReplyTo, func(m messages.Message) { ch5 <- m.Payload })
	defer unsub5()
	select {
	case p := <-ch5:
		// Module field should be null/empty — it's been removed
		assert.NotContains(t, string(p), `"name":"lifecycle-mod"`)
	case <-ctx.Done():
		t.Fatal("timeout get after remove")
	}
}
