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

// FINDING #9: WASM shard bus.on handlers don't work on standalone Kernel.
// restoreTransportSubscriptions() is only called by Node.Start(), not by standalone Kernel.
// Shard compiles and deploys but the bus subscription is never wired.
// TODO: Wire shard subscriptions in standalone Kernel mode.

// TestWASMShard_DeployWithHandler — compile, deploy shard, send message, get reply.
// NOTE: Requires Node — standalone Kernel doesn't wire shard subscriptions.
func TestWASMShard_DeployWithHandler(t *testing.T) {
	t.Skip("FINDING #9: WASM shard handlers need Node.Start() — standalone Kernel doesn't wire subscriptions")
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile shard with bus handler
	pr1, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busOn, _reply, _setMode } from "brainkit";
			export function init(): void {
				_setMode("stateless");
				_busOn("wasm-shard.echo", "onEcho");
			}
			export function onEcho(topic: usize, payload: usize): void {
				_reply('{"from":"wasm-shard"}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "echo-shard"},
	})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case p := <-ch1:
		require.False(t, responseHasError(p), "compile failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub1()

	// Deploy shard
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmDeployMsg{Name: "echo-shard"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	select {
	case p := <-ch2:
		require.False(t, responseHasError(p), "deploy failed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout deploy")
	}
	unsub2()

	// Send message to shard handler
	pr3, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "wasm-shard.echo", Payload: json.RawMessage(`{"test":true}`),
	})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()

	select {
	case p := <-ch3:
		assert.Contains(t, string(p), "wasm-shard")
	case <-ctx.Done():
		t.Fatal("timeout — shard handler didn't reply")
	}
}

// TestWASMShard_PersistentState — persistent shard maintains state across calls.
// NOTE: Same issue as DeployWithHandler — needs Node.
func TestWASMShard_PersistentState(t *testing.T) {
	t.Skip("FINDING #9: needs Node.Start() for shard subscriptions")
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr1, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busOn, _reply, _setMode, _setState, _getState } from "brainkit";
			export function init(): void {
				_setMode("persistent");
				_busOn("state-shard.inc", "onInc");
			}
			export function onInc(topic: usize, payload: usize): void {
				var current = _getState("count");
				var n: i32 = 0;
				if (current.length > 0) {
					n = parseInt(current) as i32;
				}
				n = n + 1;
				_setState("count", n.toString());
				_reply('{"count":' + n.toString() + '}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "state-inc-shard"},
	})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case p := <-ch1:
		require.False(t, responseHasError(p), "compile: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub1()

	pr2, _ := sdk.Publish(tk, ctx, messages.WasmDeployMsg{Name: "state-inc-shard"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	select {
	case p := <-ch2:
		require.False(t, responseHasError(p), "deploy: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout deploy")
	}
	unsub2()

	// Call 3 times — state should increment
	for i := 1; i <= 3; i++ {
		pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
			Topic: "state-shard.inc", Payload: json.RawMessage(`{}`),
		})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })

		select {
		case p := <-ch:
			var resp struct{ Count int `json:"count"` }
			json.Unmarshal(p, &resp)
			assert.Equal(t, i, resp.Count, "call %d should return count=%d", i, i)
		case <-ctx.Done():
			t.Fatalf("timeout on call %d", i)
		}
		unsub()
	}
}

// TestWASMShard_Undeploy — undeploy stops handler.
// NOTE: Same issue — needs Node.
func TestWASMShard_Undeploy(t *testing.T) {
	t.Skip("FINDING #9: needs Node.Start() for shard subscriptions")
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Compile + deploy
	compileAndDeploy(t, tk, "undeploy-shard", `
		import { _busOn, _reply, _setMode } from "brainkit";
		export function init(): void {
			_setMode("stateless");
			_busOn("undeploy-shard.ping", "onPing");
		}
		export function onPing(topic: usize, payload: usize): void {
			_reply('{"alive":true}');
		}
	`)

	// Verify it responds
	pr1, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "undeploy-shard.ping", Payload: json.RawMessage(`{}`),
	})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	select {
	case p := <-ch1:
		assert.Contains(t, string(p), "alive")
	case <-ctx.Done():
		t.Fatal("timeout before undeploy")
	}
	unsub1()

	// Undeploy
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmUndeployMsg{Name: "undeploy-shard"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	select {
	case <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout undeploy")
	}
	unsub2()

	// Should no longer respond
	pr3, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "undeploy-shard.ping", Payload: json.RawMessage(`{}`),
	})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()

	select {
	case <-ch3:
		t.Fatal("shard should not respond after undeploy")
	case <-time.After(1 * time.Second):
		// Good — no response after undeploy
	}
}
