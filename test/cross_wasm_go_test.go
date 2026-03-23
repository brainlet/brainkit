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
			tk := newTestKernelFull(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			t.Run("WASM_calls_Go_tool_via_invokeAsync", func(t *testing.T) {
				// Go surface: "add" tool is already registered

				// WASM surface: compile module that calls Go tool via invokeAsync
				_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
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

				// Run the WASM module — it calls the Go "add" tool and the callback fires
				runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "cross-wasm-go"})
				require.NoError(t, err)
				assert.Equal(t, 0, runResp.ExitCode)
				// The callback was called (pendingInvokes.Wait ensures this)
			})

			t.Run("Go_injects_event_WASM_shard_handles", func(t *testing.T) {
				// WASM surface: compile and deploy a shard that handles events
				_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
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

				_, err = sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "cross-go-shard"})
				require.NoError(t, err)

				// Go surface: inject event into the shard
				result, err := tk.InjectWASMEvent("cross-go-shard", "cross.go.event", json.RawMessage(`{"from":"go"}`))
				require.NoError(t, err)

				var resp map[string]any
				json.Unmarshal([]byte(result.ReplyPayload), &resp)
				assert.Equal(t, true, resp["handled"])
				assert.Equal(t, "wasm", resp["source"])

				// Cleanup
				sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "cross-go-shard"})
			})
		})
	}
}
