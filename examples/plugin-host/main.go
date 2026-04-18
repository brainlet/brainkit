// Command host drives a live round trip against the sibling
// examples/plugin-author binary. It builds the plugin from source
// into a temp directory, boots a Kit with modules/plugins wired
// to that binary, waits for the plugin.registered event, invokes
// the `echo` tool through brainkit.CallToolCall, and prints the
// reply.
//
// Run from the repo root:
//
//	go run ./examples/plugin-host
//
// Expected output (roughly):
//
//	building plugin …
//	plugin registered: test/plugin-author@0.1.0
//	echo reply: {"echoed":"ping"}
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/sdk"
)

// pluginSourceDir is the path to examples/plugin-author relative
// to the repo root. The host runner expects to be invoked from
// there (`go run ./examples/plugin-host`).
const pluginSourceDir = "./examples/plugin-author"

func main() {
	if err := run(); err != nil {
		log.Fatalf("plugin host: %v", err)
	}
}

func run() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	binDir, err := os.MkdirTemp("", "brainkit-plugin-host-")
	if err != nil {
		return fmt.Errorf("tempdir: %w", err)
	}
	defer os.RemoveAll(binDir)

	binaryPath := filepath.Join(binDir, "plugin-author")
	if err := buildPlugin(ctx, binaryPath); err != nil {
		return fmt.Errorf("build plugin: %w", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "plugin-host-demo",
		Transport: brainkit.EmbeddedNATS(),
		FSRoot:    binDir,
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
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	if err := waitForPluginRegistered(ctx, kit, "demo", 20*time.Second); err != nil {
		return fmt.Errorf("plugin registration: %w", err)
	}
	fmt.Println("plugin registered: test/plugin-author@0.1.0")

	resp, err := brainkit.CallToolCall(kit, ctx, sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"text": "ping"},
	}, brainkit.WithCallTimeout(10*time.Second))
	if err != nil {
		return fmt.Errorf("call echo: %w", err)
	}

	fmt.Printf("echo reply: %s\n", string(resp.Result))
	return nil
}

// buildPlugin compiles the sibling plugin source into binaryPath.
// Requires that the caller launched from the repo root; the
// plugin's go.mod has relative replace directives that only
// resolve from there.
func buildPlugin(ctx context.Context, binaryPath string) error {
	fmt.Println("building plugin …")

	tidy := exec.CommandContext(ctx, "go", "mod", "tidy")
	tidy.Dir = pluginSourceDir
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		return fmt.Errorf("go mod tidy: %w", err)
	}

	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Dir = pluginSourceDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	return build.Run()
}

// waitForPluginRegistered blocks until plugin.list includes
// pluginName with an active status, or the deadline elapses.
//
// Polling `plugin.list` beats subscribing to `plugin.registered`
// because the registration event can fire before the host's
// subscribe call finishes — the kit boots the plugin supervisor
// as part of brainkit.New. Polling a query is race-free.
func waitForPluginRegistered(ctx context.Context, kit *brainkit.Kit, pluginName string, timeout time.Duration) error {
	deadline, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		resp, err := brainkit.CallPluginListRunning(kit, deadline, sdk.PluginListRunningMsg{},
			brainkit.WithCallTimeout(500*time.Millisecond))
		if err == nil {
			for _, p := range resp.Plugins {
				if p.Name == pluginName && p.Status != "crashed" && p.Status != "stopped" {
					return nil
				}
			}
		}
		select {
		case <-deadline.Done():
			return fmt.Errorf("timeout waiting for plugin %q to register", pluginName)
		case <-tick.C:
		}
	}
}

