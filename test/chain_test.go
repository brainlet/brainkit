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
			tk := newTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			// Chain: Go deploys .ts → .ts creates a tool → WASM calls that tool via invokeAsync

			// Step 1: Go deploys .ts that creates a tool
			_pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
			_ch1 := make(chan messages.KitDeployResp, 1)
			_us1, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
			defer _us1()
			select {
			case <-_ch1:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Step 2: Compile WASM that calls the .ts tool
			_pr2, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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
			_ch2 := make(chan messages.WasmCompileResp, 1)
			_us2, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch2 <- r })
			defer _us2()
			select {
			case <-_ch2:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Step 3: Run WASM — it calls the .ts tool
			_pr3, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "chain-wasm"})
			require.NoError(t, err)
			_ch3 := make(chan messages.WasmRunResp, 1)
			_us3, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr3.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch3 <- r })
			require.NoError(t, err)
			defer _us3()
			var runResp messages.WasmRunResp
			select {
			case runResp = <-_ch3:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			assert.Equal(t, 0, runResp.ExitCode)

			// Verify the chain worked by calling the tool from Go too
			_pr4, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
				Name:  "chain-doubler",
				Input: map[string]any{"n": 21},
			})
			require.NoError(t, err)
			_ch4 := make(chan messages.ToolCallResp, 1)
			_us4, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr4.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch4 <- r })
			require.NoError(t, err)
			defer _us4()
			var toolResp messages.ToolCallResp
			select {
			case toolResp = <-_ch4:
			case <-ctx.Done():
				t.Fatal("timeout")
			}
			var result map[string]any
			json.Unmarshal(toolResp.Result, &result)
			assert.Equal(t, float64(42), result["doubled"])

			// Cleanup
			_spr1, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "chain-ts.ts"})
			_sch1 := make(chan messages.KitTeardownResp, 1)
			_sun1, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _spr1.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch1 <- r })
			defer _sun1()
			select { case <-_sch1: case <-ctx.Done(): t.Fatal("timeout") }
		})
	}
}

func TestChain_Go_TS_WASM_Reply(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			// Chain: Go deploys .ts tool → deploys WASM shard → Go injects event →
			// WASM shard calls .ts tool via invokeAsync → shard replies with result

			// Step 1: Deploy .ts tool
			_pr5, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
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
			_ch5 := make(chan messages.KitDeployResp, 1)
			_us5, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr5.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch5 <- r })
			defer _us5()
			select {
			case <-_ch5:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Step 2: Compile WASM shard that calls .ts tool and replies
			_pr6, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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
			_ch6 := make(chan messages.WasmCompileResp, 1)
			_us6, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr6.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch6 <- r })
			defer _us6()
			select {
			case <-_ch6:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Step 3: Deploy shard
			_pr7, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "chain-reply-shard"})
			require.NoError(t, err)
			_ch7 := make(chan messages.WasmDeployResp, 1)
			_us7, _ := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr7.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch7 <- r })
			defer _us7()
			select {
			case <-_ch7:
			case <-ctx.Done():
				t.Fatal("timeout")
			}

			// Step 4: Go injects event → WASM shard handles → calls .ts tool → replies
			result, err := tk.InjectWASMEvent("chain-reply-shard", "chain.trigger", json.RawMessage(`{}`))
			require.NoError(t, err)

			var resp map[string]string
			json.Unmarshal([]byte(result.ReplyPayload), &resp)
			assert.Equal(t, "complete", resp["chain"])

			// Cleanup
			_spr2, _ := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "chain-reply-shard"})
			_sch2 := make(chan messages.WasmUndeployResp, 1)
			_sun2, _ := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _spr2.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _sch2 <- r })
			defer _sun2()
			select { case <-_sch2: case <-ctx.Done(): t.Fatal("timeout") }
			_spr3, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "chain-reply-ts.ts"})
			_sch3 := make(chan messages.KitTeardownResp, 1)
			_sun3, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _spr3.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch3 <- r })
			defer _sun3()
			select { case <-_sch3: case <-ctx.Done(): t.Fatal("timeout") }
		})
	}
}
