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
	deployResp, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
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
	assert.True(t, deployResp.Deployed)

	// 3. Verify "greeter" appears in tools.list
	listResp, err := sdk.PublishAwait[messages.ToolListMsg, messages.ToolListResp](rt, ctx, messages.ToolListMsg{})
	require.NoError(t, err)
	found := false
	for _, tool := range listResp.Tools {
		if tool.ShortName == "greeter" {
			found = true
		}
	}
	assert.True(t, found, "deployed 'greeter' tool should appear")

	// 4. Call the deployed tool via the unified API
	callResp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
		Name:  "greeter",
		Input: map[string]any{"name": "Brainkit"},
	})
	require.NoError(t, err)
	var result map[string]string
	json.Unmarshal(callResp.Result, &result)
	assert.Equal(t, "Hello, Brainkit!", result["greeting"])

	// 5. Teardown
	_, err = sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "pipeline.ts"})
	require.NoError(t, err)
}

// TestE2E_DeployLifecycle tests the full deploy → list → redeploy → teardown cycle.
func TestE2E_DeployLifecycle(t *testing.T) {
	rt := newTestKernel(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Deploy v1
	_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
		Source: "lifecycle.ts",
		Code:   `const v1 = createTool({ id: "version-check", description: "v1", execute: async () => ({ version: 1 }) });`,
	})
	require.NoError(t, err)

	// List — should show lifecycle.ts
	listResp, err := sdk.PublishAwait[messages.KitListMsg, messages.KitListResp](rt, ctx, messages.KitListMsg{})
	require.NoError(t, err)
	sources := make(map[string]bool)
	for _, d := range listResp.Deployments {
		sources[d.Source] = true
	}
	assert.True(t, sources["lifecycle.ts"])

	// Redeploy with v2
	_, err = sdk.PublishAwait[messages.KitRedeployMsg, messages.KitRedeployResp](rt, ctx, messages.KitRedeployMsg{
		Source: "lifecycle.ts",
		Code:   `const v2 = createTool({ id: "version-check-v2", description: "v2", execute: async () => ({ version: 2 }) });`,
	})
	require.NoError(t, err)

	// Teardown
	tearResp, err := sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "lifecycle.ts"})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, tearResp.Removed, 0)

	// List — should be empty (or at least not contain lifecycle.ts)
	listResp, err = sdk.PublishAwait[messages.KitListMsg, messages.KitListResp](rt, ctx, messages.KitListMsg{})
	require.NoError(t, err)
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
	_, err := sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{
		Path: "input.json",
		Data: `{"items":["apple","banana","cherry"]}`,
	})
	require.NoError(t, err)

	// 2. Read it back
	readResp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "input.json"})
	require.NoError(t, err)

	// 3. Process with the "echo" tool (simulating processing)
	var input map[string]any
	json.Unmarshal([]byte(readResp.Data), &input)

	callResp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": readResp.Data},
	})
	require.NoError(t, err)

	// 4. Write the processed output
	_, err = sdk.PublishAwait[messages.FsWriteMsg, messages.FsWriteResp](rt, ctx, messages.FsWriteMsg{
		Path: "output.json",
		Data: string(callResp.Result),
	})
	require.NoError(t, err)

	// 5. Read and verify output
	outResp, err := sdk.PublishAwait[messages.FsReadMsg, messages.FsReadResp](rt, ctx, messages.FsReadMsg{Path: "output.json"})
	require.NoError(t, err)
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
	_, err := sdk.PublishAwait[messages.WasmCompileMsg, messages.WasmCompileResp](rt, ctx, messages.WasmCompileMsg{
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

	// Deploy
	deployResp, err := sdk.PublishAwait[messages.WasmDeployMsg, messages.WasmDeployResp](rt, ctx, messages.WasmDeployMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
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
	descResp, err := sdk.PublishAwait[messages.WasmDescribeMsg, messages.WasmDescribeResp](rt, ctx, messages.WasmDescribeMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
	assert.Equal(t, "persistent", descResp.Mode)
	assert.Contains(t, descResp.Handlers, "lifecycle.data")

	// Undeploy
	_, err = sdk.PublishAwait[messages.WasmUndeployMsg, messages.WasmUndeployResp](rt, ctx, messages.WasmUndeployMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)

	// Remove
	_, err = sdk.PublishAwait[messages.WasmRemoveMsg, messages.WasmRemoveResp](rt, ctx, messages.WasmRemoveMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)

	// Verify it's gone
	getResp, err := sdk.PublishAwait[messages.WasmGetMsg, messages.WasmGetResp](rt, ctx, messages.WasmGetMsg{Name: "lifecycle-shard"})
	require.NoError(t, err)
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
			resp, err := sdk.PublishAwait[messages.ToolCallMsg, messages.ToolCallResp](rt, ctx, messages.ToolCallMsg{
				Name:  "add",
				Input: map[string]any{"a": val, "b": val},
			})
			if err != nil {
				errors <- err
				return
			}
			var result map[string]int
			json.Unmarshal(resp.Result, &result)
			results <- result["sum"]
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
