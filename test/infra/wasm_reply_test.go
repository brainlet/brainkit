package infra_test

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

// TestWASM_Reply tests that a shard handler can use reply() to send a response
// back through the host function, and the Go caller receives the reply payload.
func TestWASM_Reply(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile a shard that replies with a JSON payload
	_pr1, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busOn, _setMode, _reply } from "brainkit";

			export function init(): void {
				_setMode("stateless");
				
_busOn("reply.test.event", "handleReply");
			}

			export function handleReply(topic: usize, payload: usize): void {
				_reply('{"response":"hello from wasm"}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "reply-shard"},
	})
	require.NoError(t, err)
	_ch1 := make(chan messages.WasmCompileResp, 1)
	_us1, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr1.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch1 <- r })
	defer _us1()
	select {
	case <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Deploy
	_pr2, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "reply-shard"})
	require.NoError(t, err)
	_ch2 := make(chan messages.WasmDeployResp, 1)
	_us2, _ := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch2 <- r })
	defer _us2()
	select {
	case <-_ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Inject event and check reply — using the exported InjectWASMEvent on Kernel
	result, err := tk.InjectWASMEvent("reply-shard", "reply.test.event", json.RawMessage(`{}`))
	require.NoError(t, err)
	assert.Equal(t, `{"response":"hello from wasm"}`, result.ReplyPayload)

	// Cleanup
	_spr1, _ := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "reply-shard"})
	_sch1 := make(chan messages.WasmUndeployResp, 1)
	_sun1, _ := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _spr1.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _sch1 <- r })
	defer _sun1()
	select { case <-_sch1: case <-ctx.Done(): t.Fatal("timeout") }
}

// TestWASM_Reply_WithState tests reply in persistent mode with state.
func TestWASM_Reply_WithState(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_pr3, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busOn, _setMode, _reply, _getState, _setState, _hasState } from "brainkit";

			export function init(): void {
				_setMode("persistent");
				
_busOn("counter.event", "handleCounter");
			}

			export function handleCounter(topic: usize, payload: usize): void {
				let count: i32 = 0;
				if (_hasState("count") != 0) {
					let s = _getState("count");
					// Parse integer from string
					count = parseInt(s) as i32;
				}
				count = count + 1;
				_setState("count", count.toString());
				_reply('{"count":' + count.toString() + '}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "counter-shard"},
	})
	require.NoError(t, err)
	_ch3 := make(chan messages.WasmCompileResp, 1)
	_us3, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr3.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch3 <- r })
	defer _us3()
	select {
	case <-_ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	_pr4, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "counter-shard"})
	require.NoError(t, err)
	_ch4 := make(chan messages.WasmDeployResp, 1)
	_us4, _ := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr4.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch4 <- r })
	defer _us4()
	select {
	case <-_ch4:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Invoke 3 times — counter should increment
	for i := 1; i <= 3; i++ {
		result, err := tk.InjectWASMEvent("counter-shard", "counter.event", json.RawMessage(`{}`))
		require.NoError(t, err)

		var resp struct {
			Count int `json:"count"`
		}
		json.Unmarshal([]byte(result.ReplyPayload), &resp)
		assert.Equal(t, i, resp.Count, "invocation %d should have count %d", i, i)
	}

	_spr2, _ := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "counter-shard"})
	_sch2 := make(chan messages.WasmUndeployResp, 1)
	_sun2, _ := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _spr2.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _sch2 <- r })
	defer _sun2()
	select { case <-_sch2: case <-ctx.Done(): t.Fatal("timeout") }
}
