package test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCross_WASM_Plugin tests WASM modules calling plugin-registered tools via invokeAsync.
// Note: file named cross_wasmmod_plugin_test.go to avoid _wasm build constraint suffix.
func TestCross_WASM_Plugin(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping plugin tests in short mode")
	}

	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			requiresNetworkTransport(t, backend)

			if backend != "nats" {
				t.Skipf("WASM↔Plugin cross-surface currently tested on NATS only")
			}

			pluginBinary := filepath.Join(t.TempDir(), "testplugin")
			buildCmd := exec.Command("go", "build", "-o", pluginBinary, "./test/testplugin/")
			buildCmd.Dir = filepath.Join("..")
			buildCmd.Stdout = os.Stdout
			buildCmd.Stderr = os.Stderr
			if err := buildCmd.Run(); err != nil {
				t.Fatalf("build test plugin: %v", err)
			}

			natsURL := startNATSContainer(t)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()

			tmpDir := t.TempDir()
			node, err := kit.NewNode(kit.NodeConfig{
				Kernel: kit.KernelConfig{
					Namespace:    "wasm-plugin-cross",
					CallerID:     "host",
					WorkspaceDir: tmpDir,
				},
				Messaging: kit.MessagingConfig{
					Transport: "nats",
					NATSURL:   natsURL,
					NATSName:  "brainkit-wasm-plugin",
				},
				Plugins: []kit.PluginConfig{
					{
						Name:         "testplugin",
						Binary:       pluginBinary,
						StartTimeout: 30 * time.Second,
					},
				},
			})
			require.NoError(t, err)
			defer node.Close()

			err = node.Start(ctx)
			require.NoError(t, err)
			time.Sleep(2 * time.Second)

			t.Run("WASM_calls_plugin_tool_via_invokeAsync", func(t *testing.T) {
				wasmCtx, wasmCancel := context.WithTimeout(ctx, 60*time.Second)
				defer wasmCancel()

				// Compile WASM that calls the plugin's "concat" tool via invokeAsync
				_pr1, err := sdk.Publish(node, wasmCtx, messages.WasmCompileMsg{
					Source: `
						import { _invokeAsync, _setState } from "brainkit";

						export function onConcatResult(topic: usize, payload: usize): void {
							_setState("pluginCallDone", "true");
						}

						export function run(): i32 {
							_invokeAsync("tools.call", '{"name":"concat","input":{"a":"wasm","b":"plugin"}}', "onConcatResult");
							return 0;
						}
					`,
					Options: &messages.WasmCompileOpts{Name: "wasm-calls-plugin"},
				})
				require.NoError(t, err)
				_ch1 := make(chan messages.WasmCompileResp, 1)
				_us1, _ := sdk.SubscribeTo[messages.WasmCompileResp](node, ctx, _pr1.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch1 <- r })
				defer _us1()
				select {
				case <-_ch1:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				_pr2, err := sdk.Publish(node, wasmCtx, messages.WasmRunMsg{ModuleID: "wasm-calls-plugin"})
				require.NoError(t, err)
				_ch2 := make(chan messages.WasmRunResp, 1)
				_us2, err := sdk.SubscribeTo[messages.WasmRunResp](node, ctx, _pr2.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch2 <- r })
				require.NoError(t, err)
				defer _us2()
				var runResp messages.WasmRunResp
				select {
				case runResp = <-_ch2:
				case <-ctx.Done():
					t.Fatal("timeout")
				}
				assert.Equal(t, 0, runResp.ExitCode)
				// pendingInvokes.Wait() ensures the callback was called
			})

			t.Run("WASM_and_plugin_tools_both_listed", func(t *testing.T) {
				listCtx, listCancel := context.WithTimeout(ctx, 10*time.Second)
				defer listCancel()

				_pr3, err := sdk.Publish(node, listCtx, messages.ToolListMsg{})
				require.NoError(t, err)
				_ch3 := make(chan messages.ToolListResp, 1)
				_us3, err := sdk.SubscribeTo[messages.ToolListResp](node, ctx, _pr3.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch3 <- r })
				require.NoError(t, err)
				defer _us3()
				var resp messages.ToolListResp
				select {
				case resp = <-_ch3:
				case <-ctx.Done():
					t.Fatal("timeout")
				}

				names := make(map[string]bool)
				for _, tool := range resp.Tools {
					names[tool.ShortName] = true
				}
				assert.True(t, names["echo"], "plugin echo")
				assert.True(t, names["concat"], "plugin concat")
			})
		})
	}
}
