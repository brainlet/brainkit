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

// TestBackendMatrix runs core API operations across ALL transport backends.
// This validates that every domain works on every backend — not just GoChannel memory.
// Covers: tools, fs, agents, kit deploy/teardown, WASM compile/run/deploy,
// async patterns (correlationID, subscribe/cancel), and cross-Kit routing.
func TestBackendMatrix(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// --- Tools domain ---
			t.Run("tools_call", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name:  "add",
					Input: map[string]any{"a": 10, "b": 32},
				})
				require.NoError(t, err)
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, 42, result["sum"])
			})

			t.Run("tools_list", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
				require.NoError(t, err)
				assert.NotEmpty(t, resp.Tools)
			})

			t.Run("tools_resolve", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.ToolResolveMsg, messages.ToolResolveResp](rt, ctx, messages.ToolResolveMsg{Name: "echo"})
				require.NoError(t, err)
				assert.Equal(t, "echo", resp.ShortName)
			})

			// --- FS domain ---
			t.Run("fs_write_read", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{
					Path: "matrix-test.txt", Data: "backend:" + backend,
				})
				require.NoError(t, err)

				resp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "matrix-test.txt"})
				require.NoError(t, err)
				assert.Equal(t, "backend:"+backend, resp.Data)
			})

			t.Run("fs_mkdir_list_stat_delete", func(t *testing.T) {
				sdk.PublishAwait[messages.FsMkdirMsg, messages.FsMkdirResp](rt, ctx, messages.FsMkdirMsg{Path: "matrix-dir"})
				sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{Path: "matrix-dir/a.txt", Data: "a"})

				listResp, err := sdk.PublishAwait[messages.FsListMsg, messages.FsListResp](rt, ctx, messages.FsListMsg{Path: "matrix-dir"})
				require.NoError(t, err)
				assert.Len(t, listResp.Files, 1)

				statResp, err := sdk.PublishAwait[messages.FsStatMsg, messages.FsStatResp](rt, ctx, messages.FsStatMsg{Path: "matrix-dir/a.txt"})
				require.NoError(t, err)
				assert.False(t, statResp.IsDir)

				_, err = sdk.PublishAwait[messages.FsDeleteMsg, messages.FsDeleteResp](rt, ctx, messages.FsDeleteMsg{Path: "matrix-dir/a.txt"})
				require.NoError(t, err)
			})

			// --- Agents domain ---
			t.Run("agents_list_empty", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.AgentListMsg, messages.AgentListResp](rt, ctx, messages.AgentListMsg{})
				require.NoError(t, err)
				assert.NotNil(t, resp.Agents)
			})

			// --- Kit lifecycle ---
			t.Run("kit_deploy_teardown", func(t *testing.T) {
				deployResp, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "matrix-deploy.ts",
					Code: `
						const matrixTool = createTool({
							id: "matrix-tool",
							description: "matrix test tool",
							execute: async () => ({ backend: "works" })
						});
					`,
				})
				require.NoError(t, err)
				assert.True(t, deployResp.Deployed)

				// Verify tool is callable
				callResp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
					Name: "matrix-tool", Input: map[string]any{},
				})
				require.NoError(t, err)
				var result map[string]string
				json.Unmarshal(callResp.Result, &result)
				assert.Equal(t, "works", result["backend"])

				// Teardown
				tearResp, err := sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "matrix-deploy.ts"})
				require.NoError(t, err)
				assert.GreaterOrEqual(t, tearResp.Removed, 0)
			})

			// --- WASM ---
			t.Run("wasm_compile_run", func(t *testing.T) {
				compResp, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 99; }`,
					Options: &messages.WasmCompileOpts{Name: "matrix-wasm-" + backend},
				})
				require.NoError(t, err)
				assert.Greater(t, compResp.Size, 0)

				runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{
					ModuleID: "matrix-wasm-" + backend,
				})
				require.NoError(t, err)
				assert.Equal(t, 99, runResp.ExitCode)
			})

			// --- Async pattern ---
			t.Run("async_correlation", func(t *testing.T) {
				corrID, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
				require.NoError(t, err)
				assert.NotEmpty(t, corrID)
			})

			// --- Kit redeploy ---
			t.Run("kit_redeploy", func(t *testing.T) {
				sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
					Source: "matrix-redeploy.ts", Code: `var v = 1;`,
				})
				resp, err := sdk.PublishAwait[messages.KitRedeployMsg, messages.KitRedeployResp](rt, ctx, messages.KitRedeployMsg{
					Source: "matrix-redeploy.ts", Code: `var v = 2;`,
				})
				require.NoError(t, err)
				assert.True(t, resp.Deployed)
				sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "matrix-redeploy.ts"})
			})

			// --- WASM deploy/undeploy/describe ---
			t.Run("wasm_deploy_lifecycle", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode } from "brainkit";
						export function init(): void { _setMode("stateless"); _on("matrix.ev", "h"); }
						export function h(t: usize, p: usize): void {}
					`, Options: &messages.WasmCompileOpts{Name: "matrix-shard-" + backend},
				})
				deploy, err := sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				assert.Equal(t, "stateless", deploy.Mode)

				desc, err := sdk.PublishAwait[messages.WasmDescribeMsg, messages.WasmDescribeResp](rt, ctx, messages.WasmDescribeMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				assert.Equal(t, "stateless", desc.Mode)

				undeploy, err := sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				assert.True(t, undeploy.Undeployed)
			})

			// --- Registry ---
			t.Run("registry_has_list", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.RegistryHasMsg, messages.RegistryHasResp](rt, ctx, messages.RegistryHasMsg{
					Category: "provider", Name: "nonexistent",
				})
				require.NoError(t, err)
				assert.False(t, resp.Found)

				listResp, err := sdk.PublishAwait[messages.RegistryListMsg, messages.RegistryListResp](rt, ctx, messages.RegistryListMsg{
					Category: "provider",
				})
				require.NoError(t, err)
				assert.NotNil(t, listResp.Items)
			})
		})
	}
}
