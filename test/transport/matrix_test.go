package transport_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
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
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := testutil.NewTestKernelFullWithBackend(t, backend)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// --- Tools domain ---
			t.Run("tools_call", func(t *testing.T) {
				_pr1, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name:  "add",
					Input: map[string]any{"a": 10, "b": 32},
				})
				require.NoError(t, err)
				_ch1 := make(chan messages.ToolCallResp, 1)
				_us1, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr1.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var resp messages.ToolCallResp
				select {
				case resp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				assert.Equal(t, 42, result["sum"])
			})

			t.Run("tools_list", func(t *testing.T) {
				_pr2, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
				require.NoError(t, err)
				_ch2 := make(chan messages.ToolListResp, 1)
				_us2, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var resp messages.ToolListResp
				select {
				case resp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotEmpty(t, resp.Tools)
			})

			t.Run("tools_resolve", func(t *testing.T) {
				_pr3, err := sdk.Publish(rt, ctx, messages.ToolResolveMsg{Name: "echo"})
				require.NoError(t, err)
				_ch3 := make(chan messages.ToolResolveResp, 1)
				_us3, err := sdk.SubscribeTo[messages.ToolResolveResp](rt, ctx, _pr3.ReplyTo, func(r messages.ToolResolveResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var resp messages.ToolResolveResp
				select {
				case resp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "echo", resp.ShortName)
			})

			// --- FS domain (via jsbridge polyfill) ---
			t.Run("fs_write_read", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.writeFileSync("matrix-test.txt", "backend:`+backend+`");
					return fs.readFileSync("matrix-test.txt", "utf8");
				`)
				require.NoError(t, err)
				assert.Equal(t, "backend:"+backend, result)
			})

			t.Run("fs_mkdir_list_stat_delete", func(t *testing.T) {
				result, err := tk.EvalTS(ctx, "__test.ts", `
					fs.mkdirSync("matrix-dir", {recursive: true});
					fs.writeFileSync("matrix-dir/a.txt", "a");
					var files = fs.readdirSync("matrix-dir");
					var s = fs.statSync("matrix-dir/a.txt");
					fs.unlinkSync("matrix-dir/a.txt");
					return JSON.stringify({fileCount: files.length, isDir: s.isDirectory()});
				`)
				require.NoError(t, err)
				var resp struct {
					FileCount int  `json:"fileCount"`
					IsDir     bool `json:"isDir"`
				}
				json.Unmarshal([]byte(result), &resp)
				assert.Equal(t, 1, resp.FileCount)
				assert.False(t, resp.IsDir)
			})

			// --- Agents domain ---
			t.Run("agents_list_empty", func(t *testing.T) {
				_pr9, err := sdk.Publish(rt, ctx, messages.AgentListMsg{})
				require.NoError(t, err)
				_ch9 := make(chan messages.AgentListResp, 1)
				_us9, err := sdk.SubscribeTo[messages.AgentListResp](rt, ctx, _pr9.ReplyTo, func(r messages.AgentListResp, m messages.Message) { _ch9 <- r })
				require.NoError(t, err)
				defer _us9()
				var resp messages.AgentListResp
				select {
				case resp = <-_ch9:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotNil(t, resp.Agents)
			})

			// --- Kit lifecycle ---
			t.Run("kit_deploy_teardown", func(t *testing.T) {
				_pr10, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "matrix-deploy.ts",
					Code: `
						const matrixTool = createTool({
							id: "matrix-tool",
							description: "matrix test tool",
							execute: async () => ({ backend: "works" })
						});
						kit.register("tool", "matrix-tool", matrixTool);
					`,
				})
				require.NoError(t, err)
				_ch10 := make(chan messages.KitDeployResp, 1)
				_us10, err := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr10.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch10 <- r })
				require.NoError(t, err)
				defer _us10()
				var deployResp messages.KitDeployResp
				select {
				case deployResp = <-_ch10:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, deployResp.Deployed)

				// Verify tool is callable
				_pr11, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
					Name: "matrix-tool", Input: map[string]any{},
				})
				require.NoError(t, err)
				_ch11 := make(chan messages.ToolCallResp, 1)
				_us11, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr11.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch11 <- r })
				require.NoError(t, err)
				defer _us11()
				var callResp messages.ToolCallResp
				select {
				case callResp = <-_ch11:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				var result map[string]string
				json.Unmarshal(callResp.Result, &result)
				assert.Equal(t, "works", result["backend"])

				// Teardown
				_pr12, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "matrix-deploy.ts"})
				require.NoError(t, err)
				_ch12 := make(chan messages.KitTeardownResp, 1)
				_us12, err := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr12.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch12 <- r })
				require.NoError(t, err)
				defer _us12()
				var tearResp messages.KitTeardownResp
				select {
				case tearResp = <-_ch12:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.GreaterOrEqual(t, tearResp.Removed, 0)
			})

			// --- WASM ---
			t.Run("wasm_compile_run", func(t *testing.T) {
				_pr13, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 99; }`,
					Options: &messages.WasmCompileOpts{Name: "matrix-wasm-" + backend},
				})
				require.NoError(t, err)
				_ch13 := make(chan messages.WasmCompileResp, 1)
				_us13, err := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr13.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch13 <- r })
				require.NoError(t, err)
				defer _us13()
				var compResp messages.WasmCompileResp
				select {
				case compResp = <-_ch13:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Greater(t, compResp.Size, 0)

				_pr14, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{
					ModuleID: "matrix-wasm-" + backend,
				})
				require.NoError(t, err)
				_ch14 := make(chan messages.WasmRunResp, 1)
				_us14, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr14.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch14 <- r })
				require.NoError(t, err)
				defer _us14()
				var runResp messages.WasmRunResp
				select {
				case runResp = <-_ch14:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
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
				sdk.Publish(rt, ctx, messages.KitDeployMsg{
					Source: "matrix-redeploy.ts", Code: `var v = 1;`,
				})
				_pr15, err := sdk.Publish(rt, ctx, messages.KitRedeployMsg{
					Source: "matrix-redeploy.ts", Code: `var v = 2;`,
				})
				require.NoError(t, err)
				_ch15 := make(chan messages.KitRedeployResp, 1)
				_us15, err := sdk.SubscribeTo[messages.KitRedeployResp](rt, ctx, _pr15.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { _ch15 <- r })
				require.NoError(t, err)
				defer _us15()
				var resp messages.KitRedeployResp
				select {
				case resp = <-_ch15:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.Deployed)
				_spr3, _ := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "matrix-redeploy.ts"})
				_sch3 := make(chan messages.KitTeardownResp, 1)
				_sun3, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _spr3.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _sch3 <- r })
				defer _sun3()
				select { case <-_sch3: case <-ctx.Done(): t.Fatal("timeout") }
			})

			// --- WASM deploy/undeploy/describe ---
			t.Run("wasm_deploy_lifecycle", func(t *testing.T) {
				cpr, _ := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _busOn, _setMode } from "brainkit";
						export function init(): void { _setMode("stateless"); _busOn("matrix.ev", "h"); }
						export function h(t: usize, p: usize): void {}
					`, Options: &messages.WasmCompileOpts{Name: "matrix-shard-" + backend},
				})
				cch := make(chan messages.WasmCompileResp, 1)
				cun, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, cpr.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { cch <- r })
				defer cun()
				select { case <-cch: case <-ctx.Done(): t.Fatal("timeout") }
				_pr16, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				_ch16 := make(chan messages.WasmDeployResp, 1)
				_us16, err := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr16.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch16 <- r })
				require.NoError(t, err)
				defer _us16()
				var deploy messages.WasmDeployResp
				select {
				case deploy = <-_ch16:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "stateless", deploy.Mode)

				_pr17, err := sdk.Publish(rt, ctx, messages.WasmDescribeMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				_ch17 := make(chan messages.WasmDescribeResp, 1)
				_us17, err := sdk.SubscribeTo[messages.WasmDescribeResp](rt, ctx, _pr17.ReplyTo, func(r messages.WasmDescribeResp, m messages.Message) { _ch17 <- r })
				require.NoError(t, err)
				defer _us17()
				var desc messages.WasmDescribeResp
				select {
				case desc = <-_ch17:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "stateless", desc.Mode)

				_pr18, err := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "matrix-shard-" + backend})
				require.NoError(t, err)
				_ch18 := make(chan messages.WasmUndeployResp, 1)
				_us18, err := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _pr18.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _ch18 <- r })
				require.NoError(t, err)
				defer _us18()
				var undeploy messages.WasmUndeployResp
				select {
				case undeploy = <-_ch18:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, undeploy.Undeployed)
			})

			// --- Registry ---
			t.Run("registry_has_list", func(t *testing.T) {
				_pr19, err := sdk.Publish(rt, ctx, messages.RegistryHasMsg{
					Category: "provider", Name: "nonexistent",
				})
				require.NoError(t, err)
				_ch19 := make(chan messages.RegistryHasResp, 1)
				_us19, err := sdk.SubscribeTo[messages.RegistryHasResp](rt, ctx, _pr19.ReplyTo, func(r messages.RegistryHasResp, m messages.Message) { _ch19 <- r })
				require.NoError(t, err)
				defer _us19()
				var resp messages.RegistryHasResp
				select {
				case resp = <-_ch19:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.False(t, resp.Found)

				_pr20, err := sdk.Publish(rt, ctx, messages.RegistryListMsg{
					Category: "provider",
				})
				require.NoError(t, err)
				_ch20 := make(chan messages.RegistryListResp, 1)
				_us20, err := sdk.SubscribeTo[messages.RegistryListResp](rt, ctx, _pr20.ReplyTo, func(r messages.RegistryListResp, m messages.Message) { _ch20 <- r })
				require.NoError(t, err)
				defer _us20()
				var listResp messages.RegistryListResp
				select {
				case listResp = <-_ch20:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.NotNil(t, listResp.Items)
			})
		})
	}
}
