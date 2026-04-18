package main

import (
	"context"
	"encoding/json"
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

// TestPluginRoundTrip drives the same flow as main.go under the
// test harness. Builds the plugin binary, boots a Kit, waits for
// registration, invokes the `echo` tool, asserts the reply shape.
//
// Skipped under -short because it shells out to `go build`, which
// is expensive and requires the Go toolchain on the runner.
func TestPluginRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("plugin-host integration test: requires `go build`")
	}

	// Resolve the repo root. The test runs from
	// examples/plugin-host; the repo root is two levels up.
	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	pluginSourceDir := filepath.Join(repoRoot, "examples", "plugin-author")

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	binaryPath := filepath.Join(t.TempDir(), "plugin-author")

	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = pluginSourceDir
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, string(out))
	}

	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Dir = pluginSourceDir
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build plugin: %v\n%s", err, string(out))
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-host-test",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    t.TempDir(),
		Modules: []brainkit.Module{
			pluginsmod.NewModule(pluginsmod.Config{
				Plugins: []brainkit.PluginConfig{{
					Name:         "demo",
					Binary:       binaryPath,
					AutoRestart:  false,
					StartTimeout: 15 * time.Second,
				}},
			}),
		},
	})
	require.NoError(t, err, "kit init")
	t.Cleanup(func() { kit.Close() })

	// Poll plugin.list until the plugin is reported as running.
	// Subscribe-after-boot races against the plugin.registered
	// emit window; polling avoids the race entirely and works
	// regardless of subscription timing.
	require.Eventually(t, func() bool {
		resp, err := brainkit.CallPluginListRunning(kit, ctx, sdk.PluginListRunningMsg{},
			brainkit.WithCallTimeout(2*time.Second))
		if err != nil {
			return false
		}
		for _, p := range resp.Plugins {
			if p.Name == "demo" {
				return true
			}
		}
		return false
	}, 30*time.Second, 250*time.Millisecond, "plugin did not register within 30s")

	resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"text": "ping"},
	}, brainkit.WithCallTimeout(10*time.Second))
	require.NoError(t, err, "echo round-trip")

	var out struct {
		Echoed string `json:"echoed"`
	}
	require.NoError(t, json.Unmarshal(resp.Result, &out), "decode echo reply: %s", string(resp.Result))
	assert.Equal(t, "ping", out.Echoed, "echo must return the input text")
}
