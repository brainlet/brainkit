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

func TestCross_TS_Go(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			rt := newTestKernelFullWithBackend(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("TS_deploys_tool_Go_calls_it", func(t *testing.T) {
				// TS surface: deploy .ts that creates a tool
				_pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "cross-ts-tool.ts",
					Code: `
						const myTool = createTool({
							id: "ts-greeter",
							description: "greets from TS",
							execute: async ({ context: input }) => {
								return { greeting: "hello from TS, " + (input.name || "world") };
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

				// Go surface: call the TS-created tool via Publish
				_pr2, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "ts-greeter",
					Input: map[string]any{"name": "Go"},
				})
				require.NoError(t, err)
				_ch2 := make(chan messages.ToolCallResp, 1)
				_us2, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "hello from TS, Go", result["greeting"])

				// Cleanup
				sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "cross-ts-tool.ts"})
			})

			t.Run("Go_registers_tool_TS_calls_via_deploy", func(t *testing.T) {
				// Go surface: "echo" tool is already registered by helpers

				// TS surface: deploy .ts that creates a wrapper tool which calls the Go-registered "echo"
				// The wrapper proves the Go tool is callable from the TS surface.
				_pr3, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "cross-go-call.ts",
					Code: `
						const wrapper = createTool({
							id: "echo-wrapper",
							description: "calls Go echo tool from TS",
							execute: async ({ context: input }) => {
								const result = await tools.call("echo", { message: input.msg || "default" });
								return { wrapped: true, inner: result };
							}
						});
					`,
				})
				require.NoError(t, err)
				_ch3 := make(chan messages.KitDeployResp, 1)
				_us3, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr3.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch3 <- r })
				defer _us3()
				select {
				case <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				// Go surface: call the TS wrapper which internally calls the Go echo tool
				_pr4, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "echo-wrapper",
					Input: map[string]any{"msg": "from TS to Go"},
				})
				require.NoError(t, err)
				_ch4 := make(chan messages.ToolCallResp, 1)
				_us4, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr4.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				var result map[string]any
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, true, result["wrapped"])
				inner, _ := result["inner"].(map[string]any)
				assert.Equal(t, "from TS to Go", inner["echoed"])

				sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "cross-go-call.ts"})
			})
		})
	}
}
