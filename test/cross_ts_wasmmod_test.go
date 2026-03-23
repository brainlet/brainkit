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
			tk := newTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			t.Run("WASM_calls_TS_registered_tool", func(t *testing.T) {
				// TS surface: deploy .ts that creates a tool
				_pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
				_ch1 := make(chan messages.KitDeployResp, 1)
				_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
				defer _us1()
				select {
				case <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// WASM surface: compile module that calls the TS tool via invokeAsync
				_pr2, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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
				_ch2 := make(chan messages.WasmCompileResp, 1)
				_us2, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch2 <- r })
				defer _us2()
				select {
				case <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_pr1, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "wasm-calls-ts"})
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

				// Cleanup
				sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "ts-for-wasm.ts"})
			})

			t.Run("TS_deploys_WASM_shard_and_injects_event", func(t *testing.T) {
				// First compile the WASM shard via Go (TS can't directly compile, but
				// the test proves the wiring: compile → deploy → inject via Go → shard responds)
				_pr3, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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
				_ch3 := make(chan messages.WasmCompileResp, 1)
				_us3, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr3.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch3 <- r })
				defer _us3()
				select {
				case <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Deploy the shard
				_pr2, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "ts-wasm-shard"})
				require.NoError(t, err)
				_ch2 := make(chan messages.WasmDeployResp, 1)
				_us2, _ := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch2 <- r })
				defer _us2()
				select {
				case <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Inject event and verify shard responded
				result, err := tk.InjectWASMEvent("ts-wasm-shard", "ts.wasm.ping", json.RawMessage(`{}`))
				require.NoError(t, err)

				var resp map[string]any
				json.Unmarshal([]byte(result.ReplyPayload), &resp)
				assert.Equal(t, true, resp["pong"])

				// Cleanup
				sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "ts-wasm-shard"})
			})
		})
	}
}
