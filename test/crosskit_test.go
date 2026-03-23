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

// crossKitCall is a helper for typed cross-Kit PublishAwaitTo calls.
// Kit A calls an operation on Kit B's namespace.
func crossKitCall[Req, Resp messages.BrainkitMessage](t *testing.T, kitA sdk.Runtime, ctx context.Context, targetNS string, req Req) Resp {
	t.Helper()
	resp, err := sdk.PublishAwaitTo[Req, Resp](kitA, ctx, targetNS, req)
	require.NoError(t, err)
	return resp
}

// --- Raw pub/sub ---

func TestCrossKit_BasicRoundTrip(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			received := make(chan []byte, 1)
			unsub, err := kitB.SubscribeRaw(ctx, "crosskit.ping", func(msg messages.Message) {
				received <- msg.Payload
			})
			require.NoError(t, err)
			defer unsub()

			xrtA := kitA.(sdk.CrossNamespaceRuntime)
			_, err = xrtA.PublishRawTo(ctx, "kit-b", "crosskit.ping", json.RawMessage(`{"from":"kit-a"}`))
			require.NoError(t, err)

			select {
			case payload := <-received:
				var msg map[string]string
				json.Unmarshal(payload, &msg)
				assert.Equal(t, "kit-a", msg["from"])
			case <-ctx.Done():
				t.Fatal("timeout")
			}
		})
	}
}

// --- Tools domain cross-Kit ---

func TestCrossKit_Tools(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Call_A_to_B", func(t *testing.T) {
				resp := crossKitCall[messages.ToolCallMsg, messages.ToolCallResp](t, kitA, ctx, "kit-b", messages.ToolCallMsg{
					Name: "echo", Input: map[string]any{"message": "from-A"},
				})
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "from-A", result["echoed"])
				assert.Equal(t, "kit-b", result["from"])
			})

			t.Run("Call_B_to_A", func(t *testing.T) {
				resp := crossKitCall[messages.ToolCallMsg, messages.ToolCallResp](t, kitB, ctx, "kit-a", messages.ToolCallMsg{
					Name: "echo", Input: map[string]any{"message": "from-B"},
				})
				var result map[string]string
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, "from-B", result["echoed"])
				assert.Equal(t, "kit-a", result["from"])
			})

			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.ToolListMsg, messages.ToolListResp](t, kitA, ctx, "kit-b", messages.ToolListMsg{})
				found := false
				for _, tool := range resp.Tools {
					if tool.ShortName == "echo" {
						found = true
					}
				}
				assert.True(t, found, "Kit B's echo tool should be visible from Kit A")
			})
		})
	}
}

// --- FS domain cross-Kit ---

func TestCrossKit_FS(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Write on Kit B, read from Kit A via cross-Kit
			crossKitCall[messages.FsWriteMsg, messages.FsWriteResp](t, kitA, ctx, "kit-b", messages.FsWriteMsg{
				Path: "crosskit.txt", Data: "written-by-A-on-B",
			})

			resp := crossKitCall[messages.FsReadMsg, messages.FsReadResp](t, kitA, ctx, "kit-b", messages.FsReadMsg{Path: "crosskit.txt"})
			assert.Equal(t, "written-by-A-on-B", resp.Data)

			// Verify Kit B sees the file locally too
			localResp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](kitB, ctx, messages.FsReadMsg{Path: "crosskit.txt"})
			require.NoError(t, err)
			assert.Equal(t, "written-by-A-on-B", localResp.Data)
		})
	}
}

// --- Agents domain cross-Kit ---

func TestCrossKit_Agents(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AgentListMsg, messages.AgentListResp](t, kitA, ctx, "kit-b", messages.AgentListMsg{})
				assert.NotNil(t, resp.Agents)
			})
			t.Run("Discover_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.AgentDiscoverMsg, messages.AgentDiscoverResp](t, kitA, ctx, "kit-b", messages.AgentDiscoverMsg{})
				assert.NotNil(t, resp.Agents)
			})
		})
	}
}

// --- Kit lifecycle cross-Kit ---

func TestCrossKit_Kit(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Deploy_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitDeployMsg, messages.KitDeployResp](t, kitA, ctx, "kit-b", messages.KitDeployMsg{
					Source: "crosskit-deploy.ts",
					Code:   `var x = 1;`,
				})
				assert.True(t, resp.Deployed)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitListMsg, messages.KitListResp](t, kitA, ctx, "kit-b", messages.KitListMsg{})
				assert.NotNil(t, resp.Deployments)
			})
			t.Run("Teardown_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.KitTeardownMsg, messages.KitTeardownResp](t, kitA, ctx, "kit-b", messages.KitTeardownMsg{
					Source: "crosskit-deploy.ts",
				})
				assert.GreaterOrEqual(t, resp.Removed, 0)
			})
		})
	}
}

// --- WASM domain cross-Kit ---

func TestCrossKit_WASM(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()

			t.Run("Compile_Run_Remote", func(t *testing.T) {
				comp := crossKitCall[messages.WasmCompileMsg, messages.WasmCompileResp](t, kitA, ctx, "kit-b", messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 88; }`,
					Options: &messages.WasmCompileOpts{Name: "crosskit-mod"},
				})
				assert.Greater(t, comp.Size, 0)

				run := crossKitCall[messages.WasmRunMsg, messages.WasmRunResp](t, kitA, ctx, "kit-b", messages.WasmRunMsg{
					ModuleID: "crosskit-mod",
				})
				assert.Equal(t, 88, run.ExitCode)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.WasmListMsg, messages.WasmListResp](t, kitA, ctx, "kit-b", messages.WasmListMsg{})
				assert.NotEmpty(t, resp.Modules)
			})
		})
	}
}

// --- Registry cross-Kit ---

func TestCrossKit_Registry(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, _ := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			t.Run("Has_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.RegistryHasMsg, messages.RegistryHasResp](t, kitA, ctx, "kit-b", messages.RegistryHasMsg{
					Category: "provider", Name: "nonexistent",
				})
				assert.False(t, resp.Found)
			})
			t.Run("List_Remote", func(t *testing.T) {
				resp := crossKitCall[messages.RegistryListMsg, messages.RegistryListResp](t, kitA, ctx, "kit-b", messages.RegistryListMsg{
					Category: "provider",
				})
				assert.NotNil(t, resp.Items)
			})
		})
	}
}
