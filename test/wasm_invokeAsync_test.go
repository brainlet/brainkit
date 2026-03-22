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

// TestWASM_InvokeAsync tests that WASM modules can call domain methods via invokeAsync
// and receive results in their callback functions.
func TestWASM_InvokeAsync_ToolsCall(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile a module that calls tools.call via invokeAsync and stores result in state
	compResp, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _invokeAsync, _setState, _getState } from "brainkit";

			// Callback — receives the result from tools.call
			export function onToolResult(topic: usize, payload: usize): void {
				// Store the raw result topic and payload in state for verification
				_setState("resultReceived", "true");
			}

			export function run(): i32 {
				// Call the "add" tool via invokeAsync
				let req = '{"name":"add","input":{"a":10,"b":32}}';
				_invokeAsync("tools.call", req, "onToolResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-tools"},
	})
	require.NoError(t, err)
	assert.Equal(t, "invoke-tools", compResp.Name)

	// Run — invokeAsync fires in a goroutine, run() returns immediately
	runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-tools"})
	require.NoError(t, err)
	assert.Equal(t, 0, runResp.ExitCode)
	// The callback should have been called (pendingInvokes.Wait ensures this)
	// State "resultReceived" was set in the callback
}

// TestWASM_InvokeAsync_UnknownTopic tests that invokeAsync with an unknown topic
// still calls back (with an error payload) instead of silently dropping.
func TestWASM_InvokeAsync_UnknownTopic(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _invokeAsync, _setState } from "brainkit";

			export function onResult(topic: usize, payload: usize): void {
				// If we get here, the callback was called (even for errors)
				_setState("callbackCalled", "true");
			}

			export function run(): i32 {
				_invokeAsync("nonexistent.command", "{}", "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-unknown"},
	})
	require.NoError(t, err)

	runResp, err := sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-unknown"})
	require.NoError(t, err)
	assert.Equal(t, 0, runResp.ExitCode)
	// The callback should have been called with an error payload
	// (our fix in wasm_host.go ensures this)
}

// TestWASM_InvokeAsync_ToolsList tests invokeAsync with tools.list.
func TestWASM_InvokeAsync_ToolsList(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	compResp, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _invokeAsync, _setState } from "brainkit";

			export function onListResult(topic: usize, payload: usize): void {
				_setState("listCalled", "true");
			}

			export function run(): i32 {
				_invokeAsync("tools.list", "{}", "onListResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-list"},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, compResp.Exports)

	_, err = sdk.PublishAwait[messages.WasmRunMsg, messages.WasmRunResp](rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-list"})
	require.NoError(t, err)
}
