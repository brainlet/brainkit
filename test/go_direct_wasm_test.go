package test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoDirect_WASM(t *testing.T) {
	for _, factory := range []struct {
		name string
		make func(t *testing.T) sdk.Runtime
	}{
		{"Kernel", newTestKernel},
		{"Node", newTestNode},
	} {
		t.Run(factory.name, func(t *testing.T) {
			rt := factory.make(t)
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			t.Run("Compile_Run", func(t *testing.T) {
				compResp, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 42; }`,
					Options: &messages.WasmCompileOpts{Name: "gd-run"},
				})
				require.NoError(t, err)
				assert.Equal(t, "gd-run", compResp.Name)
				assert.Greater(t, compResp.Size, 0)

				runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "gd-run"})
				require.NoError(t, err)
				assert.Equal(t, 42, runResp.ExitCode)
			})

			t.Run("List", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-list-a"},
				})
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 2; }`, Options: &messages.WasmCompileOpts{Name: "gd-list-b"},
				})

				resp, err := sdk.PublishAwait[messages.WasmListMsg, messages.WasmListResp](rt, ctx, messages.WasmListMsg{})
				require.NoError(t, err)
				names := make(map[string]bool)
				for _, m := range resp.Modules {
					names[m.Name] = true
				}
				assert.True(t, names["gd-list-a"])
				assert.True(t, names["gd-list-b"])
			})

			t.Run("Get", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-get"},
				})

				resp, err := sdk.PublishAwait[messages.WasmGetMsg, messages.WasmGetResp](rt, ctx, messages.WasmGetMsg{Name: "gd-get"})
				require.NoError(t, err)
				require.NotNil(t, resp.Module)
				assert.Equal(t, "gd-get", resp.Module.Name)
			})

			t.Run("Get_NotFound", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.WasmGetMsg, messages.WasmGetResp](rt, ctx, messages.WasmGetMsg{Name: "nope"})
				require.NoError(t, err)
				assert.Nil(t, resp.Module)
			})

			t.Run("Remove", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-remove"},
				})

				resp, err := sdk.PublishAwait[messages.WasmRemoveMsg, messages.WasmRemoveResp](rt, ctx, messages.WasmRemoveMsg{Name: "gd-remove"})
				require.NoError(t, err)
				assert.True(t, resp.Removed)

				getResp, _ := sdk.PublishAwait[messages.WasmGetMsg, messages.WasmGetResp](rt, ctx, messages.WasmGetMsg{Name: "gd-remove"})
				assert.Nil(t, getResp.Module)
			})

			t.Run("Deploy_Undeploy_Describe", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode } from "brainkit";
						export function init(): void {
							_setMode("stateless");
							_on("gd.test.event", "handle");
						}
						export function handle(t: usize, p: usize): void {}
					`,
					Options: &messages.WasmCompileOpts{Name: "gd-deploy"},
				})

				deployResp, err := sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				assert.Equal(t, "gd-deploy", deployResp.Module)
				assert.Equal(t, "stateless", deployResp.Mode)

				descResp, err := sdk.PublishAwait[messages.WasmDescribeMsg, messages.WasmDescribeResp](rt, ctx, messages.WasmDescribeMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				assert.Equal(t, "stateless", descResp.Mode)
				assert.Contains(t, descResp.Handlers, "gd.test.event")

				undeployResp, err := sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				assert.True(t, undeployResp.Undeployed)
			})

			t.Run("Run_HostFunctions", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _log, _setState, _getState, _hasState } from "brainkit";
						export function run(): i32 {
							_log("test log", 1);
							_setState("k", "v");
							if (_hasState("k") == 0) return 0;
							if (_getState("k") != "v") return 0;
							return 1;
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "gd-hostfn"},
				})

				runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "gd-hostfn"})
				require.NoError(t, err)
				assert.Equal(t, 1, runResp.ExitCode, "host functions should work correctly")
			})

			t.Run("Run_NotFound", func(t *testing.T) {
				_, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "nope"})
				assert.Error(t, err)
			})

			t.Run("Remove_WhileDeployed", func(t *testing.T) {
				sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode } from "brainkit";
						export function init(): void { _setMode("stateless"); _on("x.ev", "h"); }
						export function h(t: usize, p: usize): void {}
					`,
					Options: &messages.WasmCompileOpts{Name: "gd-rm-deployed"},
				})
				sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "gd-rm-deployed"})

				_, err := sdk.PublishAwait[messages.WasmRemoveMsg, messages.WasmRemoveResp](rt, ctx, messages.WasmRemoveMsg{Name: "gd-rm-deployed"})
				assert.Error(t, err, "cannot remove deployed module")

				sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "gd-rm-deployed"})
				sdk.PublishAwait[messages.WasmRemoveMsg, messages.WasmRemoveResp](rt, ctx, messages.WasmRemoveMsg{Name: "gd-rm-deployed"})
			})
		})
	}
}
