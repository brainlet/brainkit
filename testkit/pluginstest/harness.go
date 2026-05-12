// Package pluginstest provides shared end-to-end test helpers for Brainkit
// subprocess plugins.
package pluginstest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	pluginsmod "github.com/brainlet/brainkit/modules/plugins"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/presets/standard"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/transports"
)

const (
	defaultStartTimeout = 15 * time.Second
	defaultWaitTimeout  = 30 * time.Second
)

// Config describes one plugin e2e harness.
type Config struct {
	PluginName    string
	BinaryName    string
	SourceDir     string
	Namespace     string
	FSRoot        string
	Env           map[string]string
	StartTimeout  time.Duration
	WaitTimeout   time.Duration
	ExpectedTools []string
}

// Harness owns a Kit booted with one subprocess plugin.
type Harness struct {
	T           testing.TB
	Kit         *brainkit.Kit
	BinaryPath  string
	FSRoot      string
	PluginName  string
	WaitTimeout time.Duration
}

// New builds the plugin binary, boots a Kit with modules/plugins mounted, and
// waits for ExpectedTools when provided.
func New(t testing.TB, cfg Config) *Harness {
	t.Helper()
	if cfg.PluginName == "" {
		t.Fatalf("pluginstest: PluginName is required")
	}
	waitTimeout := cfg.WaitTimeout
	if waitTimeout <= 0 {
		waitTimeout = defaultWaitTimeout
	}
	fsRoot := cfg.FSRoot
	if fsRoot == "" {
		fsRoot = t.TempDir()
	}
	binaryPath := BuildBinary(t, BuildConfig{
		BinaryName: cfg.BinaryName,
		SourceDir:  cfg.SourceDir,
	})
	kit, err := brainkit.New(brainkit.Config{
		Namespace: namespaceOrDefault(cfg.Namespace, cfg.PluginName),
		Transport: transports.EmbeddedNATS(),
		FSRoot:    fsRoot,
		Modules:   Modules(cfg.PluginName, binaryPath, cfg),
	})
	if err != nil {
		t.Fatalf("brainkit.New: %v", err)
	}
	t.Cleanup(func() { _ = kit.Close() })
	h := &Harness{
		T:           t,
		Kit:         kit,
		BinaryPath:  binaryPath,
		FSRoot:      fsRoot,
		PluginName:  cfg.PluginName,
		WaitTimeout: waitTimeout,
	}
	if len(cfg.ExpectedTools) > 0 {
		h.WaitForTools(cfg.ExpectedTools...)
	}
	return h
}

// BuildConfig controls plugin binary compilation.
type BuildConfig struct {
	BinaryName string
	SourceDir  string
}

// BuildBinary compiles a plugin package and returns the temporary binary path.
func BuildBinary(t testing.TB, cfg BuildConfig) string {
	t.Helper()
	sourceDir := cfg.SourceDir
	if sourceDir == "" {
		sourceDir = "."
	}
	binaryName := cfg.BinaryName
	if binaryName == "" {
		binaryName = filepath.Base(sourceDir)
		if binaryName == "." || binaryName == string(filepath.Separator) {
			binaryName = "plugin"
		}
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)
	build := exec.Command("go", "build", "-o", binaryPath, ".")
	build.Dir = sourceDir
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("go build plugin: %v", err)
	}
	return binaryPath
}

// Modules returns the standard command modules plus a configured plugins
// module. Each call returns fresh module instances.
func Modules(pluginName, binaryPath string, cfg Config) []bkmodule.Module {
	startTimeout := cfg.StartTimeout
	if startTimeout <= 0 {
		startTimeout = defaultStartTimeout
	}
	mods := append([]bkmodule.Module{}, standard.CommandSet()...)
	return append(mods, pluginsmod.NewModule(pluginsmod.Config{
		Plugins: []pluginsmod.PluginConfig{{
			Name:         pluginName,
			Binary:       binaryPath,
			Env:          copyEnv(cfg.Env),
			AutoRestart:  false,
			StartTimeout: startTimeout,
		}},
	}))
}

// InlineDeployMsg creates a package.deploy message for one inline .ts file.
func InlineDeployMsg(entry, code string) packagemsg.PackageDeployMsg {
	name := strings.TrimSuffix(entry, ".ts")
	name = strings.TrimSuffix(name, ".js")
	manifest, _ := json.Marshal(map[string]string{"name": name, "entry": entry})
	return packagemsg.PackageDeployMsg{
		Manifest: manifest,
		Files:    map[string]string{entry: code},
	}
}

// WaitForTools polls tools.list until all short tool names are present.
func (h *Harness) WaitForTools(names ...string) map[string]bool {
	h.T.Helper()
	return WaitForTools(h.T, h.Kit, h.WaitTimeout, names...)
}

// WaitForTools polls tools.list until all short tool names are present.
func WaitForTools(t testing.TB, rt sdk.CallerRuntime, timeout time.Duration, names ...string) map[string]bool {
	t.Helper()
	if timeout <= 0 {
		timeout = defaultWaitTimeout
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	var found map[string]bool
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatalf("pluginstest: tools %v did not register within %v; found=%v lastErr=%v", names, timeout, found, lastErr)
		}
		ctx, cancel := context.WithTimeout(context.Background(), minDuration(2*time.Second, remaining))
		resp, err := toolmsg.CallToolList(rt, ctx, toolmsg.ToolListMsg{})
		cancel()
		if err == nil {
			found = map[string]bool{}
			for _, tool := range resp.Tools {
				found[tool.ShortName] = true
			}
			if hasAll(found, names) {
				return found
			}
		} else {
			lastErr = err
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// CallTool invokes tools.call and returns the raw JSON tool result.
func (h *Harness) CallTool(name string, input any) json.RawMessage {
	h.T.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), h.WaitTimeout)
	defer cancel()
	resp, err := toolmsg.CallToolCall(h.Kit, ctx, toolmsg.ToolCallMsg{Name: name, Input: input})
	if err != nil {
		h.T.Fatalf("tools.call %q: %v", name, err)
	}
	return resp.Result
}

// DeployInlineTS deploys one inline TypeScript entry file.
func (h *Harness) DeployInlineTS(entry, code string) packagemsg.PackageDeployResp {
	h.T.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), h.WaitTimeout)
	defer cancel()
	resp, err := packagemsg.CallPackageDeploy(h.Kit, ctx, InlineDeployMsg(entry, code))
	if err != nil {
		h.T.Fatalf("package.deploy %q: %v", entry, err)
	}
	if !resp.Deployed {
		h.T.Fatalf("package.deploy %q returned deployed=false", entry)
	}
	return resp
}

// CallTopic sends a request/reply message to a raw topic through the shared
// caller path.
func (h *Harness) CallTopic(topic string, payload json.RawMessage) json.RawMessage {
	h.T.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), h.WaitTimeout)
	defer cancel()
	resp, err := sdk.Call[sdk.CustomMsg, json.RawMessage](h.Kit, ctx, sdk.CustomMsg{
		Topic:   topic,
		Payload: payload,
	})
	if err != nil {
		h.T.Fatalf("call topic %q: %v", topic, err)
	}
	return resp
}

func namespaceOrDefault(namespace, pluginName string) string {
	if namespace != "" {
		return namespace
	}
	return fmt.Sprintf("test-%s", pluginName)
}

func copyEnv(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func hasAll(found map[string]bool, names []string) bool {
	for _, name := range names {
		if !found[name] {
			return false
		}
	}
	return true
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
