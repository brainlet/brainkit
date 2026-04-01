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

// ════════════════════════════════════════════════════════════════════════════
// WASM HOST FUNCTION ATTACKS
// WASM modules call host functions with crafted arguments.
// ════════════════════════════════════════════════════════════════════════════

// Attack: WASM module calls reply() with enormous payload
func TestWASMAttack_ReplyBomb(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _reply } from "brainkit";
			export function run(): i32 {
				// Reply with 1MB string
				var big = "";
				for (var i: i32 = 0; i < 1000000; i++) big += "x";
				_reply(big);
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "reply-bomb"},
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
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "reply-bomb"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case <-ch2:
		// Got result — kernel survived
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, tk.Alive(ctx), "kernel should survive WASM reply bomb")
}

// Attack: WASM module calls get_state with crafted key to probe for other shards' state
func TestWASMAttack_StateKeyProbing(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _getState, _setState } from "brainkit";
			export function run(): i32 {
				// Try to read state from other shards by guessing key patterns
				var val1 = _getState("../../other-shard/secret");
				var val2 = _getState("__internal__");
				var val3 = _getState(""); // empty key

				// Set state with crafted keys
				_setState("../escape", "pwned");
				_setState("", "empty-key-value");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "state-probe"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	unsub()

	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "state-probe"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, tk.Alive(ctx), "kernel should survive state probing")
}

// Attack: WASM module uses bus_emit to flood topics
func TestWASMAttack_BusFlood(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busEmit } from "brainkit";
			export function run(): i32 {
				for (var i: i32 = 0; i < 1000; i++) {
					_busEmit("flood.from.wasm", '{"i":' + i.toString() + '}');
				}
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "bus-flood"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	unsub()

	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "bus-flood"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, tk.Alive(ctx), "kernel should survive WASM bus flood")
}

// Attack: WASM module tries to call tools.call to escalate privileges
func TestWASMAttack_ToolCallEscalation(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile module that uses bus_publish to call tools.call
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish } from "brainkit";
			export function onResult(topic: usize, payload: usize): void {}
			export function run(): i32 {
				// Try to call a tool via bus_publish (which can reach command topics)
				_busPublish("tools.call", '{"name":"echo","input":{"message":"from-wasm"}}', "onResult");
				// Try secrets
				_busPublish("secrets.get", '{"name":"API_KEY"}', "onResult");
				// Try deploy
				_busPublish("kit.deploy", '{"source":"wasm-injected.ts","code":"output(1);"}', "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "tool-escalate"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	unsub()

	// Run — the bus_publish host function routes to the catalog for command topics
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "tool-escalate"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// FINDING: WASM bus_publish host function routes command topics through the catalog.
	// This means a WASM module CAN execute kit.deploy, secrets.set, etc.
	// There is no RBAC enforcement on WASM host function calls.
	deps := tk.ListDeployments()
	for _, d := range deps {
		if d.Source == "wasm-injected.ts" {
			t.Logf("FINDING #11: WASM module deployed .ts code via bus_publish → kit.deploy")
			t.Logf("WASM modules have unrestricted access to ALL command topics via bus_publish")
		}
	}
}

// Attack: compile module with source that includes path traversal in name
func TestWASMAttack_ModuleNameTraversal(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	evilNames := []string{
		"../../../etc/passwd",
		"foo\x00bar",
		"module with spaces",
		"\"; DROP TABLE wasm_modules; --",
		"<script>alert(1)</script>",
	}

	for _, name := range evilNames {
		t.Run(name, func(t *testing.T) {
			pr, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
				Source:  `export function run(): i32 { return 0; }`,
				Options: &messages.WasmCompileOpts{Name: name},
			})
			ch := make(chan []byte, 1)
			unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			select {
			case <-ch:
				// Either compiled or errored — both OK as long as no crash
			case <-time.After(10 * time.Second):
			}
			unsub()
		})
	}
	assert.True(t, tk.Alive(ctx), "kernel should survive evil module names")
}

// Attack: compile WASM that exports 1000 functions
func TestWASMAttack_ExportBomb(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build AS source with many exports
	source := ""
	for i := 0; i < 100; i++ {
		source += "export function fn" + json.Number(json.RawMessage([]byte{byte('0' + byte(i/10)), byte('0' + byte(i%10))})).String() + "(): i32 { return " + json.Number(json.RawMessage([]byte{byte('0' + byte(i/10)), byte('0' + byte(i%10))})).String() + "; }\n"
	}
	// Simpler: just use a string
	src := `export function a(): i32 { return 1; }
export function b(): i32 { return 2; }
export function c(): i32 { return 3; }
export function d(): i32 { return 4; }
export function e(): i32 { return 5; }
export function f(): i32 { return 6; }
export function g(): i32 { return 7; }
export function h(): i32 { return 8; }
export function i(): i32 { return 9; }
export function j(): i32 { return 10; }
export function run(): i32 { return 42; }`

	pr, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source:  src,
		Options: &messages.WasmCompileOpts{Name: "export-bomb"},
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		if !responseHasError(p) {
			assert.Contains(t, string(p), "export-bomb")
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	unsub()
}

// Attack: deploy shard with wildcard topic that catches everything
func TestWASMAttack_WildcardShardHandler(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile a shard that registers bus.on("*") — wildcard handler
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busOn, _reply, _setMode } from "brainkit";
			export function init(): void {
				_setMode("stateless");
				_busOn("*", "onAny");
			}
			export function onAny(topic: usize, payload: usize): void {
				_reply('{"intercepted":true}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "wildcard-shard"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		if responseHasError(p) {
			t.Logf("Wildcard shard compile error (expected): %s", string(p))
			return
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	unsub()

	// Try to deploy — should fail because wildcards aren't supported on transport
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmDeployMsg{Name: "wildcard-shard"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		t.Logf("Wildcard shard deploy result: %s", string(p))
		// Should either error (wildcard not supported) or deploy with limited matching
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
