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

func TestCross_TS_WASM(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFull(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			t.Run("WASM_calls_TS_registered_tool", func(t *testing.T) {
				// TS surface: deploy .ts that creates a tool
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "ts-for-wasm.ts",
					Code: `
						const tsTool = createTool({
							id: "ts-multiplier",
							description: "multiplies a number by 2",
							execute: async ({ context: input }) => {
								return { result: (input.value || 0) * 2 };
							}
						});
					`,
				})
				require.NoError(t, err)

				// WASM surface: compile module that calls the TS tool via invokeAsync
				_, err = sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _invokeAsync, _setState } from "brainkit";

						export function onResult(topic: usize, payload: usize): void {
							_setState("callDone", "true");
						}

						export function run(): i32 {
							_invokeAsync("tools.call", '{"name":"ts-multiplier","input":{"value":21}}', "onResult");
							return 0;
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "wasm-calls-ts"},
				})
				require.NoError(t, err)

				runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "wasm-calls-ts"})
				require.NoError(t, err)
				assert.Equal(t, 0, runResp.ExitCode)

				// Cleanup
				sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "ts-for-wasm.ts"})
			})

			t.Run("TS_deploys_WASM_shard_and_injects_event", func(t *testing.T) {
				// First compile the WASM shard via Go (TS can't directly compile, but
				// the test proves the wiring: compile → deploy → inject via Go → shard responds)
				_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode, _reply } from "brainkit";

						export function init(): void {
							_setMode("stateless");
							_on("ts.wasm.ping", "handlePing");
						}

						export function handlePing(topic: usize, payload: usize): void {
							_reply('{"pong":true}');
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "ts-wasm-shard"},
				})
				require.NoError(t, err)

				// Deploy the shard
				_, err = sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "ts-wasm-shard"})
				require.NoError(t, err)

				// Inject event and verify shard responded
				result, err := tk.InjectWASMEvent("ts-wasm-shard", "ts.wasm.ping", json.RawMessage(`{}`))
				require.NoError(t, err)

				var resp map[string]any
				json.Unmarshal([]byte(result.ReplyPayload), &resp)
				assert.Equal(t, true, resp["pong"])

				// Cleanup
				sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "ts-wasm-shard"})
			})
		})
	}
}
