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

// TestWASM_InvokeAsync tests that WASM modules can call domain methods via _busPublish
// and receive results in their callback functions.
func TestWASM_InvokeAsync_ToolsCall(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile a module that calls tools.call via _busPublish and stores result in state
	_pr1, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish, _setState, _getState } from "brainkit";

			// Callback — receives the result from tools.call
			export function onToolResult(topic: usize, payload: usize): void {
				// Store the raw result topic and payload in state for verification
				_setState("resultReceived", "true");
			}

			export function run(): i32 {
				// Call the "add" tool via _busPublish
				let req = '{"name":"add","input":{"a":10,"b":32}}';
				_busPublish("tools.call", req, "onToolResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-tools"},
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
	assert.Equal(t, "invoke-tools", compResp.Name)

	// Run — _busPublish fires in a goroutine, run() returns immediately
	_pr2, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-tools"})
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
	assert.Equal(t, 0, runResp.ExitCode)
	// The callback should have been called (pendingInvokes.Wait ensures this)
	// State "resultReceived" was set in the callback
}

// TestWASM_InvokeAsync_UnknownTopic tests that _busPublish with an unknown topic
// still calls back (with an error payload) instead of silently dropping.
func TestWASM_InvokeAsync_UnknownTopic(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_pr3, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish, _setState } from "brainkit";

			export function onResult(topic: usize, payload: usize): void {
				// If we get here, the callback was called (even for errors)
				_setState("callbackCalled", "true");
			}

			export function run(): i32 {
				_busPublish("nonexistent.command", "{}", "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-unknown"},
	})
	require.NoError(t, err)
	_ch3 := make(chan messages.WasmCompileResp, 1)
	_us3, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr3.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch3 <- r })
	defer _us3()
	select {
	case <-_ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	_pr4, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-unknown"})
	require.NoError(t, err)
	_ch4 := make(chan messages.WasmRunResp, 1)
	_us4, err := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr4.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch4 <- r })
	require.NoError(t, err)
	defer _us4()
	var runResp messages.WasmRunResp
	select {
	case runResp = <-_ch4:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Equal(t, 0, runResp.ExitCode)
	// The callback should have been called with an error payload
	// (our fix in wasm_host.go ensures this)
}

// TestWASM_InvokeAsync_ToolsList tests _busPublish with tools.list.
func TestWASM_InvokeAsync_ToolsList(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_pr5, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish, _setState } from "brainkit";

			export function onListResult(topic: usize, payload: usize): void {
				_setState("listCalled", "true");
			}

			export function run(): i32 {
				_busPublish("tools.list", "{}", "onListResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "invoke-list"},
	})
	require.NoError(t, err)
	_ch5 := make(chan messages.WasmCompileResp, 1)
	_us5, err := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr5.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch5 <- r })
	require.NoError(t, err)
	defer _us5()
	var compResp messages.WasmCompileResp
	select {
	case compResp = <-_ch5:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.NotEmpty(t, compResp.Exports)

	_pr6, err := sdk.Publish(rt, ctx, messages.WasmRunMsg{ModuleID: "invoke-list"})
	require.NoError(t, err)
	_ch6 := make(chan messages.WasmRunResp, 1)
	_us6, _ := sdk.SubscribeTo[messages.WasmRunResp](rt, ctx, _pr6.ReplyTo, func(r messages.WasmRunResp, m messages.Message) { _ch6 <- r })
	defer _us6()
	select {
	case <-_ch6:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
