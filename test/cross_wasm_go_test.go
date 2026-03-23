package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCross_WASM_Go(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			t.Run("WASM_calls_Go_tool_via_invokeAsync", func(t *testing.T) {
				// Go surface: "add" tool is already registered

				// WASM surface: compile module that calls Go tool via invokeAsync
				_pr1, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _invokeAsync, _setState } from "brainkit";

						export function onResult(topic: usize, payload: usize): void {
							_setState("gotResult", "true");
						}

						export function run(): i32 {
							_invokeAsync("tools.call", '{"name":"add","input":{"a":7,"b":8}}', "onResult");
							return 0;
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "cross-wasm-go"},
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

				// Run the WASM module — it calls the Go "add" tool and the callback fires
				_pr1, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "cross-wasm-go"})
				require.NoError(t, err)
				_ch1 := make(chan messages.WasmRunResp, 1)
				_us1, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr1.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var runResp messages.WasmRunResp
				select {
				case runResp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, 0, runResp.ExitCode)
				// The callback was called (pendingInvokes.Wait ensures this)
			})

			t.Run("Go_injects_event_WASM_shard_handles", func(t *testing.T) {
				// WASM surface: compile and deploy a shard that handles events
				_pr2, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode, _reply } from "brainkit";

						export function init(): void {
							_setMode("stateless");
							_on("cross.go.event", "handleEvent");
						}

						export function handleEvent(topic: usize, payload: usize): void {
							_reply('{"handled":true,"source":"wasm"}');
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "cross-go-shard"},
				})
				require.NoError(t, err)
				_ch2 := make(chan messages.WasmCompileResp, 1)
				_us2, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch2 <- r })
				defer _us2()
				select {
				case <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_pr2, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "cross-go-shard"})
				require.NoError(t, err)
				_ch2 := make(chan messages.WasmDeployResp, 1)
				_us2, _ := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch2 <- r })
				defer _us2()
				select {
				case <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Go surface: inject event into the shard
				result, err := tk.InjectWASMEvent("cross-go-shard", "cross.go.event", json.RawMessage(`{"from":"go"}`))
				require.NoError(t, err)

				var resp map[string]any
				json.Unmarshal([]byte(result.ReplyPayload), &resp)
				assert.Equal(t, true, resp["handled"])
				assert.Equal(t, "wasm", resp["source"])

				// Cleanup
				sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "cross-go-shard"})
			})
		})
	}
}
