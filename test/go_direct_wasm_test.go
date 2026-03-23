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
				_pr1, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source:  `export function run(): i32 { return 42; }`,
					Options: &messages.WasmCompileOpts{Name: "gd-run"},
				})
				require.NoError(t, err)
				_ch1 := make(chan messages.WasmCompileResp, 1)
				_us1, err := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr1.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch1 <- r })
				require.NoError(t, err)
				defer _us1()
				var compResp messages.WasmCompileResp
				select {
				case compResp = <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "gd-run", compResp.Name)
				assert.Greater(t, compResp.Size, 0)

				_pr2, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "gd-run"})
				require.NoError(t, err)
				_ch2 := make(chan messages.WasmRunResp, 1)
				_us2, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr2.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var runResp messages.WasmRunResp
				select {
				case runResp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, 42, runResp.ExitCode)
			})

			t.Run("List", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-list-a"},
				})
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 2; }`, Options: &messages.WasmCompileOpts{Name: "gd-list-b"},
				})

				_pr3, err := sdk.Publish(rt, ctx, messages.WasmListMsg{})
				require.NoError(t, err)
				_ch3 := make(chan messages.WasmListResp, 1)
				_us3, err := sdk.SubscribeTo[messages.WasmListResp](rt, ctx, _pr3.ReplyTo, func(r messages.WasmListResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var resp messages.WasmListResp
				select {
				case resp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				names := make(map[string]bool)
				for _, m := range resp.Modules {
					names[m.Name] = true
				}
				assert.True(t, names["gd-list-a"])
				assert.True(t, names["gd-list-b"])
			})

			t.Run("Get", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-get"},
				})

				_pr4, err := sdk.Publish(rt, ctx, messages.WasmGetMsg{Name: "gd-get"})
				require.NoError(t, err)
				_ch4 := make(chan messages.WasmGetResp, 1)
				_us4, err := sdk.SubscribeTo[messages.WasmGetResp](rt, ctx, _pr4.ReplyTo, func(r messages.WasmGetResp, m messages.Message) { _ch4 <- r })
				require.NoError(t, err)
				defer _us4()
				var resp messages.WasmGetResp
				select {
				case resp = <-_ch4:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				require.NotNil(t, resp.Module)
				assert.Equal(t, "gd-get", resp.Module.Name)
			})

			t.Run("Get_NotFound", func(t *testing.T) {
				_pr5, err := sdk.Publish(rt, ctx, messages.WasmGetMsg{Name: "nope"})
				require.NoError(t, err)
				_ch5 := make(chan messages.WasmGetResp, 1)
				_us5, err := sdk.SubscribeTo[messages.WasmGetResp](rt, ctx, _pr5.ReplyTo, func(r messages.WasmGetResp, m messages.Message) { _ch5 <- r })
				require.NoError(t, err)
				defer _us5()
				var resp messages.WasmGetResp
				select {
				case resp = <-_ch5:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Nil(t, resp.Module)
			})

			t.Run("Remove", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `export function run(): i32 { return 1; }`, Options: &messages.WasmCompileOpts{Name: "gd-remove"},
				})

				_pr6, err := sdk.Publish(rt, ctx, messages.WasmRemoveMsg{Name: "gd-remove"})
				require.NoError(t, err)
				_ch6 := make(chan messages.WasmRemoveResp, 1)
				_us6, err := sdk.SubscribeTo[messages.WasmRemoveResp](rt, ctx, _pr6.ReplyTo, func(r messages.WasmRemoveResp, m messages.Message) { _ch6 <- r })
				require.NoError(t, err)
				defer _us6()
				var resp messages.WasmRemoveResp
				select {
				case resp = <-_ch6:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, resp.Removed)

				_, _ := sdk.Publish(rt, ctx, messages.WasmGetMsg{Name: "gd-remove"})
				assert.Nil(t, getResp.Module)
			})

			t.Run("Deploy_Undeploy_Describe", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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

				_pr8, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				_ch8 := make(chan messages.WasmDeployResp, 1)
				_us8, err := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr8.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch8 <- r })
				require.NoError(t, err)
				defer _us8()
				var deployResp messages.WasmDeployResp
				select {
				case deployResp = <-_ch8:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "gd-deploy", deployResp.Module)
				assert.Equal(t, "stateless", deployResp.Mode)

				_pr9, err := sdk.Publish(rt, ctx, messages.WasmDescribeMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				_ch9 := make(chan messages.WasmDescribeResp, 1)
				_us9, err := sdk.SubscribeTo[messages.WasmDescribeResp](rt, ctx, _pr9.ReplyTo, func(r messages.WasmDescribeResp, m messages.Message) { _ch9 <- r })
				require.NoError(t, err)
				defer _us9()
				var descResp messages.WasmDescribeResp
				select {
				case descResp = <-_ch9:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, "stateless", descResp.Mode)
				assert.Contains(t, descResp.Handlers, "gd.test.event")

				_pr10, err := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "gd-deploy"})
				require.NoError(t, err)
				_ch10 := make(chan messages.WasmUndeployResp, 1)
				_us10, err := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _pr10.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _ch10 <- r })
				require.NoError(t, err)
				defer _us10()
				var undeployResp messages.WasmUndeployResp
				select {
				case undeployResp = <-_ch10:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.True(t, undeployResp.Undeployed)
			})

			t.Run("Run_HostFunctions", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
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

				_pr11, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "gd-hostfn"})
				require.NoError(t, err)
				_ch11 := make(chan messages.WasmRunResp, 1)
				_us11, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr11.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch11 <- r })
				require.NoError(t, err)
				defer _us11()
				var runResp messages.WasmRunResp
				select {
				case runResp = <-_ch11:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, 1, runResp.ExitCode, "host functions should work correctly")
			})

			t.Run("Run_NotFound", func(t *testing.T) {
				pr, _ := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "nope"})
				errCh := make(chan string, 1)
				un, _ := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
					var r struct { Error string `json:"error"` }
					json.Unmarshal(msg.Payload, &r)
					errCh <- r.Error
				})
				defer un()
				select {
				case errMsg := <-errCh:
					assert.NotEmpty(t, errMsg)
				case <-ctx.Done():
					t.Fatal("timeout")
				}
			})

			t.Run("Remove_WhileDeployed", func(t *testing.T) {
				sdk.Publish(rt, ctx, messages.WasmCompileMsg{
					Source: `
						import { _on, _setMode } from "brainkit";
						export function init(): void { _setMode("stateless"); _on("x.ev", "h"); }
						export function h(t: usize, p: usize): void {}
					`,
					Options: &messages.WasmCompileOpts{Name: "gd-rm-deployed"},
				})
				sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "gd-rm-deployed"})

				pr, _ := sdk.Publish(rt, ctx, messages.WasmRemoveMsg{Name: "gd-rm-deployed"})
				errCh := make(chan string, 1)
				un, _ := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
					var r struct { Error string `json:"error"` }
					json.Unmarshal(msg.Payload, &r)
					errCh <- r.Error
				})
				defer un()
				select {
				case errMsg := <-errCh:
					assert.NotEmpty(t, errMsg, "cannot remove deployed module")
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "gd-rm-deployed"})
				sdk.Publish(rt, ctx, messages.WasmRemoveMsg{Name: "gd-rm-deployed"})
			})
		})
	}
}
