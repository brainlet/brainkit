package test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogHandler_TSCompartment(t *testing.T) {
	var mu sync.Mutex
	var logs []kit.LogEntry

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-log",
		WorkspaceDir: t.TempDir(),
		LogHandler: func(e kit.LogEntry) {
			mu.Lock()
			logs = append(logs, e)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy .ts that logs at different levels
	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](k, ctx, messages.KitDeployMsg{
		Source: "log-test.ts",
		Code:   `console.log("hello from ts"); console.warn("warning!"); console.error("error!");`,
	})
	require.NoError(t, err)

	// Check captured logs
	mu.Lock()
	defer mu.Unlock()

	var tagged []string
	for _, l := range logs {
		if l.Source == "log-test.ts" {
			tagged = append(tagged, l.Level+":"+l.Message)
		}
	}
	assert.Contains(t, tagged, "log:hello from ts")
	assert.Contains(t, tagged, "warn:warning!")
	assert.Contains(t, tagged, "error:error!")
}

func TestLogHandler_TSCompartment_MultipleFiles(t *testing.T) {
	var mu sync.Mutex
	var logs []kit.LogEntry

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-log-multi",
		WorkspaceDir: t.TempDir(),
		LogHandler: func(e kit.LogEntry) {
			mu.Lock()
			logs = append(logs, e)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy two different .ts files
	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](k, ctx, messages.KitDeployMsg{
		Source: "file-a.ts",
		Code:   `console.log("from file A");`,
	})
	require.NoError(t, err)

	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](k, ctx, messages.KitDeployMsg{
		Source: "file-b.ts",
		Code:   `console.log("from file B");`,
	})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()

	var fromA, fromB []string
	for _, l := range logs {
		if l.Source == "file-a.ts" {
			fromA = append(fromA, l.Message)
		}
		if l.Source == "file-b.ts" {
			fromB = append(fromB, l.Message)
		}
	}
	assert.Contains(t, fromA, "from file A", "file-a.ts logs should be tagged")
	assert.Contains(t, fromB, "from file B", "file-b.ts logs should be tagged")
}

func TestLogHandler_WASMModule(t *testing.T) {
	var mu sync.Mutex
	var logs []kit.LogEntry

	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-wasm-log",
		WorkspaceDir: t.TempDir(),
		LogHandler: func(e kit.LogEntry) {
			mu.Lock()
			logs = append(logs, e)
			mu.Unlock()
		},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile and run a WASM module that logs
	_, err = sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](k, ctx, messages.WasmCompileMsg{
		Source: `
			import { _log } from "brainkit";
			export function run(): i32 {
				_log("hello from wasm", 1);
				_log("wasm warning", 2);
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "log-mod"},
	})
	require.NoError(t, err)

	_, err = sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](k, ctx, messages.WasmRunMsg{ModuleID: "log-mod"})
	require.NoError(t, err)

	// Check captured logs
	mu.Lock()
	defer mu.Unlock()

	var wasmLogs []string
	for _, l := range logs {
		if strings.HasPrefix(l.Source, "wasm:") {
			wasmLogs = append(wasmLogs, l.Source+"|"+l.Level+"|"+l.Message)
		}
	}
	assert.Contains(t, wasmLogs, "wasm:log-mod|info|hello from wasm")
	assert.Contains(t, wasmLogs, "wasm:log-mod|warn|wasm warning")
}

func TestLogHandler_NilDefault(t *testing.T) {
	// When LogHandler is nil, logs should go to default (stdout) without panicking
	k, err := kit.NewKernel(kit.KernelConfig{
		Namespace:    "test",
		CallerID:     "test-nil-log",
		WorkspaceDir: t.TempDir(),
		// LogHandler: nil — default
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Should not panic
	_, err = sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](k, ctx, messages.KitDeployMsg{
		Source: "nil-test.ts",
		Code:   `console.log("should not panic");`,
	})
	require.NoError(t, err)
}
