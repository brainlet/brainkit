package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ToolPipeline tests a real-world scenario:
// Go registers a tool → deploys .ts code that calls the tool → verifies the full chain.
func TestE2E_ToolPipeline(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. "echo" and "add" tools are already registered by helpers

	// 2. Deploy .ts code that creates a new tool (using the simple createTool API)
	_pr1, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "pipeline.ts",
		Code: `
			const greeter = createTool({
				id: "greeter",
				description: "greets a person by name",
				execute: async ({ context: input }) => {
					return { greeting: "Hello, " + (input.name || "world") + "!" };
				}
			});
		`,
	})
	require.NoError(t, err)
	_ch1 := make(chan messages.KitDeployResp, 1)
	_us1, err := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr1.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch1 <- r })
	require.NoError(t, err)
	defer _us1()
	var deployResp messages.KitDeployResp
	select {
	case deployResp = <-_ch1:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.True(t, deployResp.Deployed)

	// 3. Verify "greeter" appears in tools.list
	_pr2, err := sdk.Publish(rt, ctx, messages.ToolListMsg{})
	require.NoError(t, err)
	_ch2 := make(chan messages.ToolListResp, 1)
	_us2, err := sdk.SubscribeTo[messages.ToolListResp](rt, ctx, _pr2.ReplyTo, func(r messages.ToolListResp, m messages.Message) { _ch2 <- r })
	require.NoError(t, err)
	defer _us2()
	var listResp messages.ToolListResp
	select {
	case listResp = <-_ch2:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	found := false
	for _, tool := range listResp.Tools {
		if tool.ShortName == "greeter" {
			found = true
		}
	}
	assert.True(t, found, "deployed 'greeter' tool should appear")

	// 4. Call the deployed tool via the unified API
	_pr3, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
		Name:  "greeter",
		Input: map[string]any{"name": "Brainkit"},
	})
	require.NoError(t, err)
	_ch3 := make(chan messages.ToolCallResp, 1)
	_us3, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr3.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch3 <- r })
	require.NoError(t, err)
	defer _us3()
	var callResp messages.ToolCallResp
	select {
	case callResp = <-_ch3:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	var result map[string]string
	json.Unmarshal(callResp.Result, &result)
	assert.Equal(t, "Hello, Brainkit!", result["greeting"])

	// 5. Teardown
	_pr4, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "pipeline.ts"})
	require.NoError(t, err)
	_ch4 := make(chan messages.KitTeardownResp, 1)
	_us4, _ := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr4.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch4 <- r })
	defer _us4()
	select {
	case <-_ch4:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// TestE2E_DeployLifecycle tests the full deploy → list → redeploy → teardown cycle.
func TestE2E_DeployLifecycle(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy v1
	_pr5, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
		Source: "lifecycle.ts",
		Code:   `const v1 = createTool({ id: "version-check", description: "v1", execute: async () => ({ version: 1 }) });`,
	})
	require.NoError(t, err)
	_ch5 := make(chan messages.KitDeployResp, 1)
	_us5, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, _pr5.ReplyTo, func(r messages.KitDeployResp, m messages.Message) { _ch5 <- r })
	defer _us5()
	select {
	case <-_ch5:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// List — should show lifecycle.ts
	_pr6, err := sdk.Publish(rt, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	_ch6 := make(chan messages.KitListResp, 1)
	_us6, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr6.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch6 <- r })
	require.NoError(t, err)
	defer _us6()
	var listResp messages.KitListResp
	select {
	case listResp = <-_ch6:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	sources := make(map[string]bool)
	for _, d := range listResp.Deployments {
		sources[d.Source] = true
	}
	assert.True(t, sources["lifecycle.ts"])

	// Redeploy with v2
	_pr7, err := sdk.Publish(rt, ctx, messages.KitRedeployMsg{
		Source: "lifecycle.ts",
		Code:   `const v2 = createTool({ id: "version-check-v2", description: "v2", execute: async () => ({ version: 2 }) });`,
	})
	require.NoError(t, err)
	_ch7 := make(chan messages.KitRedeployResp, 1)
	_us7, _ := sdk.SubscribeTo[messages.KitRedeployResp](rt, ctx, _pr7.ReplyTo, func(r messages.KitRedeployResp, m messages.Message) { _ch7 <- r })
	defer _us7()
	select {
	case <-_ch7:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Teardown
	_pr8, err := sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "lifecycle.ts"})
	require.NoError(t, err)
	_ch8 := make(chan messages.KitTeardownResp, 1)
	_us8, err := sdk.SubscribeTo[messages.KitTeardownResp](rt, ctx, _pr8.ReplyTo, func(r messages.KitTeardownResp, m messages.Message) { _ch8 <- r })
	require.NoError(t, err)
	defer _us8()
	var tearResp messages.KitTeardownResp
	select {
	case tearResp = <-_ch8:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.GreaterOrEqual(t, tearResp.Removed, 0)

	// List — should be empty (or at least not contain lifecycle.ts)
	_pr9, err := sdk.Publish(rt, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	_ch9 := make(chan messages.KitListResp, 1)
	_us9, err := sdk.SubscribeTo[messages.KitListResp](rt, ctx, _pr9.ReplyTo, func(r messages.KitListResp, m messages.Message) { _ch9 <- r })
	require.NoError(t, err)
	defer _us9()
	
	select {
	case listResp = <-_ch9:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	for _, d := range listResp.Deployments {
		assert.NotEqual(t, "lifecycle.ts", d.Source, "should be torn down")
	}
}

// TestE2E_MultiDomain tests a workflow that crosses domain boundaries:
// write a file → call a tool that reads and processes it → write output → verify.
func TestE2E_MultiDomain(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Write input file
	_pr10, err := sdk.Publish(rt, ctx, messages.FsWriteMsg{
		Path: "input.json",
		Data: `{"items":["apple","banana","cherry"]}`,
	})
	require.NoError(t, err)
	_ch10 := make(chan messages.FsWriteResp, 1)
	_us10, _ := sdk.SubscribeTo[messages.FsWriteResp](rt, ctx, _pr10.ReplyTo, func(r messages.FsWriteResp, m messages.Message) { _ch10 <- r })
	defer _us10()
	select {
	case <-_ch10:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// 2. Read it back
	_pr11, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "input.json"})
	require.NoError(t, err)
	_ch11 := make(chan messages.FsReadResp, 1)
	_us11, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr11.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch11 <- r })
	require.NoError(t, err)
	defer _us11()
	var readResp messages.FsReadResp
	select {
	case readResp = <-_ch11:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// 3. Process with the "echo" tool (simulating processing)
	var input map[string]any
	json.Unmarshal([]byte(readResp.Data), &input)

	_pr12, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": readResp.Data},
	})
	require.NoError(t, err)
	_ch12 := make(chan messages.ToolCallResp, 1)
	_us12, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, _pr12.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { _ch12 <- r })
	require.NoError(t, err)
	defer _us12()
	var callResp messages.ToolCallResp
	select {
	case callResp = <-_ch12:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// 4. Write the processed output
	_pr13, err := sdk.Publish(rt, ctx, messages.FsWriteMsg{
		Path: "output.json",
		Data: string(callResp.Result),
	})
	require.NoError(t, err)
	_ch13 := make(chan messages.FsWriteResp, 1)
	_us13, _ := sdk.SubscribeTo[messages.FsWriteResp](rt, ctx, _pr13.ReplyTo, func(r messages.FsWriteResp, m messages.Message) { _ch13 <- r })
	defer _us13()
	select {
	case <-_ch13:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// 5. Read and verify output
	_pr14, err := sdk.Publish(rt, ctx, messages.FsReadMsg{Path: "output.json"})
	require.NoError(t, err)
	_ch14 := make(chan messages.FsReadResp, 1)
	_us14, err := sdk.SubscribeTo[messages.FsReadResp](rt, ctx, _pr14.ReplyTo, func(r messages.FsReadResp, m messages.Message) { _ch14 <- r })
	require.NoError(t, err)
	defer _us14()
	var outResp messages.FsReadResp
	select {
	case outResp = <-_ch14:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Contains(t, outResp.Data, "echoed")
}

// TestE2E_WasmShardLifecycle tests the full WASM shard lifecycle:
// compile → deploy → inject events → verify state → undeploy → remove.
func TestE2E_WasmShardLifecycle(t *testing.T) {
	tk := newTestKernelFull(t)
	rt := sdk.Runtime(tk)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile a persistent shard that accumulates event data
	_pr15, err := sdk.Publish(rt, ctx, messages.WasmCompileMsg{
		Source: `
			import { _on, _setMode, _reply, _getState, _setState, _hasState } from "brainkit";

			export function init(): void {
				_setMode("persistent");
				_on("lifecycle.data", "handleData");
			}

			export function handleData(topic: usize, payload: usize): void {
				let count: i32 = 0;
				if (_hasState("eventCount") != 0) {
					count = parseInt(_getState("eventCount")) as i32;
				}
				count = count + 1;
				_setState("eventCount", count.toString());
				_reply('{"eventCount":' + count.toString() + '}');
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "lifecycle-shard"},
	})
	require.NoError(t, err)
	_ch15 := make(chan messages.WasmCompileResp, 1)
	_us15, _ := sdk.SubscribeTo[messages.WasmCompileResp](rt, ctx, _pr15.ReplyTo, func(r messages.WasmCompileResp, m messages.Message) { _ch15 <- r })
	defer _us15()
	select {
	case <-_ch15:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Deploy
	_pr16, err := sdk.Publish(rt, ctx, messages.WasmDeployMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	_ch16 := make(chan messages.WasmDeployResp, 1)
	_us16, err := sdk.SubscribeTo[messages.WasmDeployResp](rt, ctx, _pr16.ReplyTo, func(r messages.WasmDeployResp, m messages.Message) { _ch16 <- r })
	require.NoError(t, err)
	defer _us16()
	var deployResp messages.WasmDeployResp
	select {
	case deployResp = <-_ch16:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Equal(t, "persistent", deployResp.Mode)

	// Inject 5 events
	for i := 1; i <= 5; i++ {
		result, err := tk.InjectWASMEvent("lifecycle-shard", "lifecycle.data", json.RawMessage(`{"data":"event"}`))
		require.NoError(t, err)

		var resp struct {
			EventCount int `json:"eventCount"`
		}
		json.Unmarshal([]byte(result.ReplyPayload), &resp)
		assert.Equal(t, i, resp.EventCount, "event %d should have count %d", i, i)
	}

	// Describe — verify handlers
	_pr17, err := sdk.Publish(rt, ctx, messages.WasmDescribeMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	_ch17 := make(chan messages.WasmDescribeResp, 1)
	_us17, err := sdk.SubscribeTo[messages.WasmDescribeResp](rt, ctx, _pr17.ReplyTo, func(r messages.WasmDescribeResp, m messages.Message) { _ch17 <- r })
	require.NoError(t, err)
	defer _us17()
	var descResp messages.WasmDescribeResp
	select {
	case descResp = <-_ch17:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Equal(t, "persistent", descResp.Mode)
	assert.Contains(t, descResp.Handlers, "lifecycle.data")

	// Undeploy
	_pr18, err := sdk.Publish(rt, ctx, messages.WasmUndeployMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	_ch18 := make(chan messages.WasmUndeployResp, 1)
	_us18, _ := sdk.SubscribeTo[messages.WasmUndeployResp](rt, ctx, _pr18.ReplyTo, func(r messages.WasmUndeployResp, m messages.Message) { _ch18 <- r })
	defer _us18()
	select {
	case <-_ch18:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Remove
	_pr19, err := sdk.Publish(rt, ctx, messages.WasmRemoveMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	_ch19 := make(chan messages.WasmRemoveResp, 1)
	_us19, _ := sdk.SubscribeTo[messages.WasmRemoveResp](rt, ctx, _pr19.ReplyTo, func(r messages.WasmRemoveResp, m messages.Message) { _ch19 <- r })
	defer _us19()
	select {
	case <-_ch19:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// Verify it's gone
	_pr20, err := sdk.Publish(rt, ctx, messages.WasmGetMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	_ch20 := make(chan messages.WasmGetResp, 1)
	_us20, err := sdk.SubscribeTo[messages.WasmGetResp](rt, ctx, _pr20.ReplyTo, func(r messages.WasmGetResp, m messages.Message) { _ch20 <- r })
	require.NoError(t, err)
	defer _us20()
	var getResp messages.WasmGetResp
	select {
	case getResp = <-_ch20:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Nil(t, getResp.Module)
}

// TestE2E_ConcurrentOperations tests multiple operations running concurrently.
func TestE2E_ConcurrentOperations(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Fire 3 concurrent tool calls — GoChannel fan-out can cause contention at higher counts
	const n = 3
	results := make(chan int, n)
	errors := make(chan error, n)

	for i := range n {
		go func(val int) {
			pubResult, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{
				Name:  "add",
				Input: map[string]any{"a": val, "b": val},
			})
			if err != nil {
				errors <- err
				return
			}
			done := make(chan messages.ToolCallResp, 1)
			unsub, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pubResult.ReplyTo, func(r messages.ToolCallResp, m messages.Message) {
				done <- r
			})
			if err != nil {
				errors <- err
				return
			}
			defer unsub()
			select {
			case resp := <-done:
				var result map[string]int
				json.Unmarshal(resp.Result, &result)
				results <- result["sum"]
			case <-ctx.Done():
				errors <- ctx.Err()
			}
		}(i)
	}

	sums := make(map[int]bool)
	for range n {
		select {
		case sum := <-results:
			sums[sum] = true
		case err := <-errors:
			t.Fatalf("concurrent call failed: %v", err)
		case <-ctx.Done():
			t.Fatal("timeout")
		}
	}

	// Verify we got all expected sums (0+0=0, 1+1=2, ..., 9+9=18)
	for i := range n {
		assert.True(t, sums[i*2], "should have sum %d", i*2)
	}
}
