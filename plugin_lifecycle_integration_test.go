package brainkit_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/modules/plugins/pluginmsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/testkit/pluginstest"
	"github.com/brainlet/brainkit/transports"
	"github.com/stretchr/testify/require"
)

func TestPluginModuleUnmountStopsProcessAndClearsTools(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin lifecycle integration test skipped in short mode")
	}

	binaryPath := pluginstest.BuildBinary(t, pluginstest.BuildConfig{
		BinaryName: "testplugin",
		SourceDir:  "./test/testplugin",
	})
	pluginMod := testPluginModule(binaryPath)
	kit := newPluginLifecycleKit(t, pluginMod)
	t.Cleanup(func() { _ = kit.Close() })

	pluginstest.WaitForTools(t, kit, 30*time.Second, "echo", "concat")
	requirePluginSnapshot(t, pluginMod, func(s pluginsmod.DebugSnapshot) bool {
		return s.RunningProcesses == 1 &&
			s.RegisteredTools == 2 &&
			s.WebSocket.ActiveConnections == 1 &&
			s.WebSocket.RegisteredTools == 2
	}, "plugin process and websocket tools did not become active")

	resp := callEchoTool(t, kit, "before-unmount")
	require.Equal(t, "before-unmount", resp["echoed"])
	require.Equal(t, "testplugin", resp["plugin"])

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, kit.Unmount(ctx, "plugins"))

	requirePluginSnapshot(t, pluginMod, func(s pluginsmod.DebugSnapshot) bool {
		return s.RunningProcesses == 0 &&
			s.RegisteredTools == 0 &&
			s.PluginToolReferences == 0 &&
			!s.WebSocket.Listening &&
			s.WebSocket.ActiveConnections == 0 &&
			s.WebSocket.RegisteredTools == 0
	}, "plugin module did not release resources after unmount")
	requireToolCallFails(t, kit, "echo")

	nextMod := testPluginModule(binaryPath)
	require.NoError(t, kit.Mount(ctx, nextMod))
	pluginstest.WaitForTools(t, kit, 30*time.Second, "echo", "concat")
	resp = callEchoTool(t, kit, "after-remount")
	require.Equal(t, "after-remount", resp["echoed"])
	require.Equal(t, "testplugin", resp["plugin"])
}

func TestPluginModuleKitCloseStopsProcessAndClearsTools(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin lifecycle integration test skipped in short mode")
	}

	binaryPath := pluginstest.BuildBinary(t, pluginstest.BuildConfig{
		BinaryName: "testplugin",
		SourceDir:  "./test/testplugin",
	})
	pluginMod := testPluginModule(binaryPath)
	kit := newPluginLifecycleKit(t, pluginMod)
	pluginstest.WaitForTools(t, kit, 30*time.Second, "echo", "concat")

	require.NoError(t, kit.Close())
	requirePluginSnapshot(t, pluginMod, func(s pluginsmod.DebugSnapshot) bool {
		return s.RunningProcesses == 0 &&
			s.RegisteredTools == 0 &&
			s.PluginToolReferences == 0 &&
			!s.WebSocket.Listening &&
			s.WebSocket.ActiveConnections == 0 &&
			s.WebSocket.RegisteredTools == 0
	}, "plugin module did not release resources after Kit close")
}

func TestPluginLifecycleCommandsRestartAndStopUpdateTools(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin lifecycle integration test skipped in short mode")
	}

	binaryPath := pluginstest.BuildBinary(t, pluginstest.BuildConfig{
		BinaryName: "testplugin",
		SourceDir:  "./test/testplugin",
	})
	pluginMod := testPluginModule(binaryPath)
	kit := newPluginLifecycleKit(t, pluginMod)
	t.Cleanup(func() { _ = kit.Close() })
	pluginstest.WaitForTools(t, kit, 30*time.Second, "echo", "concat")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	before := pluginStatusPID(t, kit, ctx)

	restartResp, err := pluginmsg.CallPluginRestart(kit, ctx, pluginmsg.PluginRestartMsg{Name: "testplugin"})
	require.NoError(t, err)
	require.True(t, restartResp.Restarted)
	require.NotZero(t, restartResp.PID)
	pluginstest.WaitForTools(t, kit, 30*time.Second, "echo", "concat")
	after := pluginStatusPID(t, kit, ctx)
	require.NotZero(t, after)
	require.NotEqual(t, before, after, "plugin.restart should replace the subprocess")

	resp := callEchoTool(t, kit, "after-restart")
	require.Equal(t, "after-restart", resp["echoed"])

	stopResp, err := pluginmsg.CallPluginStop(kit, ctx, pluginmsg.PluginStopMsg{Name: "testplugin"})
	require.NoError(t, err)
	require.True(t, stopResp.Stopped)
	requireToolCallFails(t, kit, "echo")
	requirePluginSnapshot(t, pluginMod, func(s pluginsmod.DebugSnapshot) bool {
		return s.RunningProcesses == 0 &&
			s.RegisteredTools == 0 &&
			s.PluginToolReferences == 0 &&
			s.WebSocket.ActiveConnections == 0 &&
			s.WebSocket.RegisteredTools == 0
	}, "plugin.stop did not release process/tool state")
}

func testPluginModule(binaryPath string) *pluginsmod.Module {
	return pluginsmod.NewModule(pluginsmod.Config{
		Plugins: []pluginsmod.PluginConfig{{
			Name:         "testplugin",
			Binary:       binaryPath,
			AutoRestart:  false,
			StartTimeout: 30 * time.Second,
		}},
	})
}

func newPluginLifecycleKit(t *testing.T, pluginMod *pluginsmod.Module) *brainkit.Kit {
	t.Helper()
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-plugin-lifecycle",
		Transport: transports.EmbeddedNATS(),
		FSRoot:    t.TempDir(),
		Modules:   []bkmodule.Module{pluginMod},
	})
	require.NoError(t, err)
	return kit
}

func pluginStatusPID(t *testing.T, kit *brainkit.Kit, ctx context.Context) int {
	t.Helper()
	resp, err := pluginmsg.CallPluginStatus(kit, ctx, pluginmsg.PluginStatusMsg{Name: "testplugin"})
	require.NoError(t, err)
	require.Equal(t, "running", resp.Status)
	require.NotZero(t, resp.PID)
	return resp.PID
}

func requirePluginSnapshot(t *testing.T, pluginMod *pluginsmod.Module, ok func(pluginsmod.DebugSnapshot) bool, msg string) {
	t.Helper()
	require.Eventually(t, func() bool {
		return ok(pluginMod.DebugSnapshot())
	}, 10*time.Second, 100*time.Millisecond, "%s; snapshot=%+v", msg, pluginMod.DebugSnapshot())
}

func callEchoTool(t *testing.T, kit *brainkit.Kit, message string) map[string]string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := toolmsg.CallToolCall(kit, ctx, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": message},
	})
	require.NoError(t, err)
	var out map[string]string
	require.NoError(t, json.Unmarshal(resp.Result, &out))
	return out
}

func requireToolCallFails(t *testing.T, kit *brainkit.Kit, name string) {
	t.Helper()
	require.Eventually(t, func() bool {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_, err := toolmsg.CallToolCall(kit, ctx, toolmsg.ToolCallMsg{
			Name:  name,
			Input: map[string]any{"message": "after-unmount"},
		})
		return err != nil
	}, 5*time.Second, 100*time.Millisecond)
}
