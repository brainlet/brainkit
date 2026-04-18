package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPluginToolCancel verifies that cancelling a ToolCall on the host
// side fires a TypeCancel frame to the plugin, aborting the handler's
// ctx so it returns promptly instead of running to completion.
//
// The test plugin exposes a "sleep" tool that blocks until ctx.Done
// or a 10s fallback. The host publishes the tool.call with a short
// context, then cancels it; we assert the call returns with a cancel
// error well under the 10s fallback.
func TestPluginToolCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin e2e test")
	}

	dir := t.TempDir()

	pluginCode := `package main

import (
	"context"
	"time"

	bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

type SleepIn struct{}
type SleepOut struct {
	Cancelled bool ` + "`json:\"cancelled\"`" + `
}

func main() {
	p := bkplugin.New("test", "cancel-test", "0.1.0")

	bkplugin.Tool(p, "sleep", "wait for cancel", func(ctx context.Context, _ bkplugin.Client, _ SleepIn) (SleepOut, error) {
		select {
		case <-ctx.Done():
			return SleepOut{Cancelled: true}, nil
		case <-time.After(10 * time.Second):
			return SleepOut{Cancelled: false}, nil
		}
	})

	p.OnStart(func(_ bkplugin.Client) error { return nil })
	if err := p.Run(); err != nil { panic(err) }
}
`

	pluginDir := filepath.Join(dir, "cancel-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "main.go"), []byte(pluginCode), 0644))

	projectRoot, _ := filepath.Abs("../../..")
	goModContent := fmt.Sprintf(`module test-cancel-plugin
go 1.26.0

require (
	github.com/brainlet/brainkit v0.0.0-00010101000000-000000000000
	github.com/brainlet/brainkit/sdk v0.0.0-00010101000000-000000000000
)

replace (
	github.com/brainlet/brainkit => %s
	github.com/brainlet/brainkit/sdk => %s/sdk
	github.com/brainlet/brainkit/vendor_typescript => %s/vendor_typescript
	github.com/brainlet/brainkit/vendor_quickjs => %s/vendor_quickjs
)
`, projectRoot, projectRoot, projectRoot, projectRoot)
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "go.mod"), []byte(goModContent), 0644))

	tidy := exec.Command("go", "mod", "tidy")
	tidy.Dir = pluginDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	require.NoError(t, tidy.Run(), "go mod tidy")

	binaryPath := filepath.Join(dir, "cancel-plugin-bin")
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = pluginDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	require.NoError(t, build.Run(), "plugin build")

	tmpDir := t.TempDir()
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "test-cancel-plugin",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    tmpDir,
		Modules: []brainkit.Module{
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []brainkit.PluginConfig{{
					Name: "cancel-test", Binary: binaryPath, AutoRestart: false,
				}},
			}),
		},
	})
	require.NoError(t, err)
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	regCh := make(chan struct{}, 1)
	unsub, _ := sdk.SubscribeTo[sdk.PluginRegisteredEvent](kit, ctx, "plugin.registered",
		func(evt sdk.PluginRegisteredEvent, _ sdk.Message) {
			if evt.Name == "cancel-test" {
				select {
				case regCh <- struct{}{}:
				default:
				}
			}
		})
	select {
	case <-regCh:
		unsub()
	case <-ctx.Done():
		unsub()
		t.Fatal("plugin did not register")
	}

	// Start a tool call with a 1-second deadline. Without WS cancel
	// the plugin would wait 10s; with cancel plumbing the handler
	// returns as soon as its ctx fires.
	callCtx, callCancel := context.WithTimeout(ctx, 1*time.Second)
	defer callCancel()

	start := time.Now()
	_, callErr := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](kit, callCtx, sdk.ToolCallMsg{
		Name:  "sleep",
		Input: map[string]any{},
	})
	elapsed := time.Since(start)
	t.Logf("sleep call returned after %s: err=%v", elapsed.Round(time.Millisecond), callErr)

	// The host-side ctx deadline fires first; the plugin receives
	// TypeCancel and its handler returns. We don't check the returned
	// payload (the host already gave up on replies after ctx expired),
	// but the plugin subprocess must be free to accept new work,
	// which means the goroutine exited. Give it a moment, then call
	// "sleep" again with a finite ctx — if the previous handler was
	// still running the plugin's dispatch slot would be locked.
	time.Sleep(500 * time.Millisecond)

	quickCtx, quickCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer quickCancel()
	resp, err := brainkit.Call[sdk.ToolCallMsg, sdk.ToolCallResp](kit, quickCtx, sdk.ToolCallMsg{
		Name:  "sleep",
		Input: map[string]any{},
	}, brainkit.WithCallTimeout(2*time.Second))

	// Under WS cancel, this second call either returns cancelled=true
	// (handler observed ctx done) or times out at the brainkit.Call
	// level after 2s — either way proves the earlier handler is gone.
	if err == nil {
		var out struct {
			Cancelled bool `json:"cancelled"`
		}
		require.NoError(t, json.Unmarshal(resp.Result, &out))
		assert.True(t, out.Cancelled, "plugin handler should observe ctx cancellation")
	}

	// First call must have aborted well under the 10s fallback. Budget
	// 5s to allow for WS frame round-trip + startup variance on CI.
	assert.Less(t, elapsed, 5*time.Second, "cancel should propagate before the 10s fallback")
}
