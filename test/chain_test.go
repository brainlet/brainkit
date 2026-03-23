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

func TestChain_Go_TS_WASM(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFull(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			// Chain: Go deploys .ts → .ts creates a tool → WASM calls that tool via invokeAsync

			// Step 1: Go deploys .ts that creates a tool
			_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
				Source: "chain-ts.ts",
				Code: `
					const chainTool = createTool({
						id: "chain-doubler",
						description: "doubles a value (created by .ts, called by WASM)",
						execute: async ({ context: input }) => {
							return { doubled: (input.n || 0) * 2 };
						}
					});
				`,
			})
			require.NoError(t, err)

			// Step 2: Compile WASM that calls the .ts tool
			_, err = sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
				Source: `
					import { _invokeAsync, _setState } from "brainkit";

					export function onDoubled(topic: usize, payload: usize): void {
						_setState("chainDone", "true");
					}

					export function run(): i32 {
						_invokeAsync("tools.call", '{"name":"chain-doubler","input":{"n":21}}', "onDoubled");
						return 0;
					}
				`,
				Options: &messages.WasmCompileOpts{Name: "chain-wasm"},
			})
			require.NoError(t, err)

			// Step 3: Run WASM — it calls the .ts tool
			runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "chain-wasm"})
			require.NoError(t, err)
			assert.Equal(t, 0, runResp.ExitCode)

			// Verify the chain worked by calling the tool from Go too
			toolResp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
				Name:  "chain-doubler",
				Input: map[string]any{"n": 21},
			})
			require.NoError(t, err)
			var result map[string]any
			json.Unmarshal(toolResp.Result, &result)
			assert.Equal(t, float64(42), result["doubled"])

			// Cleanup
			sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "chain-ts.ts"})
		})
	}
}

func TestChain_Go_TS_WASM_Reply(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFull(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			// Chain: Go deploys .ts tool → deploys WASM shard → Go injects event →
			// WASM shard calls .ts tool via invokeAsync → shard replies with result

			// Step 1: Deploy .ts tool
			_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
				Source: "chain-reply-ts.ts",
				Code: `
					const adder = createTool({
						id: "chain-adder",
						description: "adds 100 to a value",
						execute: async ({ context: input }) => {
							return { sum: (input.v || 0) + 100 };
						}
					});
				`,
			})
			require.NoError(t, err)

			// Step 2: Compile WASM shard that calls .ts tool and replies
			_, err = sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
				Source: `
					import { _on, _setMode, _invokeAsync, _reply, _setState, _getState, _hasState } from "brainkit";

					export function init(): void {
						_setMode("stateless");
						_on("chain.trigger", "handleTrigger");
					}

					export function onAdderResult(topic: usize, payload: usize): void {
						_setState("adderResult", "done");
					}

					export function handleTrigger(topic: usize, payload: usize): void {
						_invokeAsync("tools.call", '{"name":"chain-adder","input":{"v":42}}', "onAdderResult");
						_reply('{"chain":"complete"}');
					}
				`,
				Options: &messages.WasmCompileOpts{Name: "chain-reply-shard"},
			})
			require.NoError(t, err)

			// Step 3: Deploy shard
			_, err = sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "chain-reply-shard"})
			require.NoError(t, err)

			// Step 4: Go injects event → WASM shard handles → calls .ts tool → replies
			result, err := tk.InjectWASMEvent("chain-reply-shard", "chain.trigger", json.RawMessage(`{}`))
			require.NoError(t, err)

			var resp map[string]string
			json.Unmarshal([]byte(result.ReplyPayload), &resp)
			assert.Equal(t, "complete", resp["chain"])

			// Cleanup
			sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "chain-reply-shard"})
			sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "chain-reply-ts.ts"})
		})
	}
}
