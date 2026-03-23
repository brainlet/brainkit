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
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
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

				// Go surface: call the TS-created tool via PublishAwait
				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "ts-greeter",
					Input: map[string]any{"name": "Go"},
				})
				require.NoError(t, err)

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
				_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
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

				// Go surface: call the TS wrapper which internally calls the Go echo tool
				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "echo-wrapper",
					Input: map[string]any{"msg": "from TS to Go"},
				})
				require.NoError(t, err)

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
