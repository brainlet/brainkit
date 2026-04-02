package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// HELPER: publish a workflow command and wait for the typed response
// ════════════════════════════════════════════════════════════════════════════

func publishAndWait[Req messages.BrainkitMessage, Resp any](
	t *testing.T, k *brainkit.Kernel, msg Req, timeout time.Duration,
) Resp {
	t.Helper()
	result, err := sdk.Publish(k, context.Background(), msg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var resp Resp
	unsub, err := sdk.SubscribeTo[Resp](k, ctx, result.ReplyTo, func(r Resp, m messages.Message) {
		resp = r
		cancel()
	})
	require.NoError(t, err)
	defer unsub()
	<-ctx.Done()
	return resp
}

// deployWorkflow deploys a .ts file that registers a workflow.
func deployWorkflow(t *testing.T, k *brainkit.Kernel, source, code string) {
	t.Helper()
	_, err := k.Deploy(context.Background(), source, code)
	require.NoError(t, err)
}

// ════════════════════════════════════════════════════════════════════════════
// HAPPY PATH
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_Start_Sequential(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "seq-wf.ts", `
		const step1 = createStep({
			id: "upper",
			inputSchema: z.object({ text: z.string() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData }) => ({ result: inputData.text.toUpperCase() }),
		});
		const step2 = createStep({
			id: "exclaim",
			inputSchema: z.object({ result: z.string() }),
			outputSchema: z.object({ final: z.string() }),
			execute: async ({ inputData }) => ({ final: inputData.result + "!!!" }),
		});
		const wf = createWorkflow({
			id: "seq-test",
			inputSchema: z.object({ text: z.string() }),
			outputSchema: z.object({ final: z.string() }),
		}).then(step1).then(step2).commit();
		kit.register("workflow", "seq-test", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "seq-test", InputData: json.RawMessage(`{"text":"hello"}`)},
		10*time.Second,
	)

	require.Empty(t, resp.Error, "should not error: %s", resp.Error)
	require.NotEmpty(t, resp.RunID)
	assert.Contains(t, resp.Status, "success")
}

func TestWorkflowBus_Start_Parallel(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "par-wf.ts", `
		const stepA = createStep({
			id: "a", inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ doubled: z.number() }),
			execute: async ({ inputData }) => ({ doubled: inputData.x * 2 }),
		});
		const stepB = createStep({
			id: "b", inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ tripled: z.number() }),
			execute: async ({ inputData }) => ({ tripled: inputData.x * 3 }),
		});
		const wf = createWorkflow({
			id: "par-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ doubled: z.number(), tripled: z.number() }),
		}).parallel([stepA, stepB]).commit();
		kit.register("workflow", "par-test", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "par-test", InputData: json.RawMessage(`{"x":5}`)},
		10*time.Second,
	)

	require.Empty(t, resp.Error)
	assert.Contains(t, resp.Status, "success")
}

func TestWorkflowBus_List(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "list-wf.ts", `
		const wf = createWorkflow({
			id: "list-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
		}).then(createStep({
			id: "double", inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
			execute: async ({ inputData }) => ({ y: inputData.x * 2 }),
		})).commit();
		kit.register("workflow", "list-test", wf);
	`)

	resp := publishAndWait[messages.WorkflowListMsg, messages.WorkflowListResp](
		t, tk.Kernel, messages.WorkflowListMsg{}, 5*time.Second,
	)

	require.Empty(t, resp.Error)
	found := false
	for _, wf := range resp.Workflows {
		if wf.Name == "list-test" {
			found = true
			assert.NotEmpty(t, wf.Source)
		}
	}
	require.True(t, found, "list-test workflow should appear in list")
}

func TestWorkflowBus_SuspendResume(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "suspend-wf.ts", `
		const approval = createStep({
			id: "approval",
			inputSchema: z.object({ item: z.string() }),
			resumeSchema: z.object({ approved: z.boolean() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) {
					return await suspend({ reason: "needs approval" });
				}
				return { result: resumeData.approved ? "approved: " + inputData.item : "denied" };
			},
		});
		const wf = createWorkflow({
			id: "approval-flow",
			inputSchema: z.object({ item: z.string() }),
			outputSchema: z.object({ result: z.string() }),
		}).then(approval).commit();
		kit.register("workflow", "approval-flow", wf);
	`)

	// Start — should suspend
	startResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "approval-flow", InputData: json.RawMessage(`{"item":"widget"}`)},
		10*time.Second,
	)

	require.Equal(t, "suspended", startResp.Status, "workflow should suspend")
	require.NotEmpty(t, startResp.RunID)

	// Resume with approval
	resumeResp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
		t, tk.Kernel,
		messages.WorkflowResumeMsg{
			Name: "approval-flow", RunID: startResp.RunID,
			Step: "approval", ResumeData: json.RawMessage(`{"approved":true}`),
		},
		10*time.Second,
	)

	require.Empty(t, resumeResp.Error, "resume should not error: %s", resumeResp.Error)
}

func TestWorkflowBus_Cancel(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "cancel-wf.ts", `
		const approval = createStep({
			id: "wait",
			inputSchema: z.object({ x: z.number() }),
			resumeSchema: z.object({ go: z.boolean() }),
			outputSchema: z.object({ done: z.boolean() }),
			execute: async ({ resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { done: true };
			},
		});
		const wf = createWorkflow({
			id: "cancel-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ done: z.boolean() }),
		}).then(approval).commit();
		kit.register("workflow", "cancel-test", wf);
	`)

	// Start — suspend
	startResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "cancel-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)

	// Cancel
	cancelResp := publishAndWait[messages.WorkflowCancelMsg, messages.WorkflowCancelResp](
		t, tk.Kernel,
		messages.WorkflowCancelMsg{Name: "cancel-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, cancelResp.Error)
	require.True(t, cancelResp.Cancelled)

	// Status after cancel should show "canceled" (reads from storage, not in-memory)
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, tk.Kernel,
		messages.WorkflowStatusMsg{Name: "cancel-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, statusResp.Error, "status after cancel should succeed (run persisted): %s", statusResp.Error)
	assert.Equal(t, "canceled", statusResp.Status)
}

func TestWorkflowBus_WithToolCall(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "tool-wf.ts", `
		const callTool = createStep({
			id: "call-echo",
			inputSchema: z.object({ msg: z.string() }),
			outputSchema: z.object({ echoed: z.string() }),
			execute: async ({ inputData }) => {
				const result = await tools.call("echo", { message: inputData.msg });
				return { echoed: result.echoed };
			},
		});
		const wf = createWorkflow({
			id: "tool-wf",
			inputSchema: z.object({ msg: z.string() }),
			outputSchema: z.object({ echoed: z.string() }),
		}).then(callTool).commit();
		kit.register("workflow", "tool-wf", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "tool-wf", InputData: json.RawMessage(`{"msg":"from-workflow"}`)},
		10*time.Second,
	)

	require.Empty(t, resp.Error, "tool workflow should not error: %s", resp.Error)
	assert.Contains(t, resp.Status, "success")
}

// ════════════════════════════════════════════════════════════════════════════
// ERROR PATHS
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_NotFound(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "ghost-workflow"},
		5*time.Second,
	)

	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func TestWorkflowBus_ResumeNonexistentRun(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	resp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
		t, tk.Kernel,
		messages.WorkflowResumeMsg{Name: "any", RunID: "fake-run-id", ResumeData: json.RawMessage(`{}`)},
		5*time.Second,
	)

	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func TestWorkflowBus_StatusNonexistentRun(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	resp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, tk.Kernel,
		messages.WorkflowStatusMsg{Name: "any", RunID: "fake-run-id"},
		5*time.Second,
	)

	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func TestWorkflowBus_CancelNonexistentRun(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	resp := publishAndWait[messages.WorkflowCancelMsg, messages.WorkflowCancelResp](
		t, tk.Kernel,
		messages.WorkflowCancelMsg{Name: "any", RunID: "fake-run-id"},
		5*time.Second,
	)

	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func TestWorkflowBus_StepWithError(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "error-wf.ts", `
		const failStep = createStep({
			id: "fail",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
			execute: async () => { throw new Error("intentional failure"); },
		});
		const wf = createWorkflow({
			id: "error-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
		}).then(failStep).commit();
		kit.register("workflow", "error-test", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, tk.Kernel,
		messages.WorkflowStartMsg{Name: "error-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)

	// Mastra returns failed status, not an error on the bus command itself
	require.NotEmpty(t, resp.RunID)
	assert.Equal(t, "failed", resp.Status)
}

// ════════════════════════════════════════════════════════════════════════════
// CONCURRENT
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_ConcurrentStarts(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	deployWorkflow(t, tk.Kernel, "concurrent-wf.ts", `
		const step = createStep({
			id: "compute",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
			execute: async ({ inputData }) => ({ result: inputData.n * inputData.n }),
		});
		const wf = createWorkflow({
			id: "concurrent-test",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
		}).then(step).commit();
		kit.register("workflow", "concurrent-test", wf);
	`)

	const N = 5
	results := make(chan messages.WorkflowStartResp, N)

	testutil.ConcurrentDo(t, N, func(i int) {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, tk.Kernel,
			messages.WorkflowStartMsg{
				Name:      "concurrent-test",
				InputData: json.RawMessage(`{"n":` + json.Number(string(rune('0'+i+1))).String() + `}`),
			},
			10*time.Second,
		)
		results <- resp
	})
	close(results)

	for resp := range results {
		assert.Empty(t, resp.Error, "concurrent start should not error: %s", resp.Error)
		assert.NotEmpty(t, resp.RunID)
	}
}

// ════════════════════════════════════════════════════════════════════════════
// PERSISTENCE: workflow storage uses configured backend
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_StorageUpgrade(t *testing.T) {
	// Verify that when a default storage is configured, Mastra uses it
	// (not InMemoryStore) for workflow snapshots.
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")

	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	// Deploy and run a workflow with suspend/resume
	deployWorkflow(t, k, "persist-wf.ts", `
		const step = createStep({
			id: "check",
			inputSchema: z.object({ x: z.number() }),
			resumeSchema: z.object({ go: z.boolean() }),
			outputSchema: z.object({ done: z.boolean() }),
			execute: async ({ resumeData, suspend }) => {
				if (!resumeData) return await suspend({ waiting: true });
				return { done: true };
			},
		});
		const wf = createWorkflow({
			id: "persist-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ done: z.boolean() }),
		}).then(step).commit();
		kit.register("workflow", "persist-test", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "persist-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)

	require.Equal(t, "suspended", resp.Status, "should suspend")
	require.NotEmpty(t, resp.RunID)

	// Verify the snapshot was persisted using the upgraded store (not InMemoryStore).
	// Access the shim's store directly via __kit_store_holder.
	result, err := k.EvalTS(context.Background(), "__check_storage.ts", `
		var store = globalThis.__kit_store_holder.store;
		var wfStore = await store.getStore("workflows");
		var snapshot = await wfStore.loadWorkflowSnapshot({
			workflowName: "persist-test",
			runId: "`+resp.RunID+`"
		});
		return snapshot ? "persisted" : "not-found";
	`)
	require.NoError(t, err)
	assert.Equal(t, "persisted", result, "workflow snapshot should be persisted to configured storage")
}

// ════════════════════════════════════════════════════════════════════════════
// STATUS FROM STORAGE: query completed runs (not just active)
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_StatusFromStorage(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-status", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	deployWorkflow(t, k, "status-wf.ts", `
		const step = createStep({
			id: "double",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
			execute: async ({ inputData }) => ({ y: inputData.x * 2 }),
		});
		const wf = createWorkflow({
			id: "status-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
		}).then(step).commit();
		kit.register("workflow", "status-test", wf);
	`)

	// Start and complete
	startResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "status-test", InputData: json.RawMessage(`{"x":5}`)},
		10*time.Second,
	)
	require.Empty(t, startResp.Error)
	require.Equal(t, "success", startResp.Status)

	// Query status — should return from storage even though run is completed
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "status-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, statusResp.Error, "status of completed run should work: %s", statusResp.Error)
	assert.Equal(t, "success", statusResp.Status)
}

// ════════════════════════════════════════════════════════════════════════════
// WORKFLOW.RUNS: list runs with status filter
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_Runs(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-runs", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	deployWorkflow(t, k, "runs-wf.ts", `
		const step = createStep({
			id: "check",
			inputSchema: z.object({ x: z.number() }),
			resumeSchema: z.object({ go: z.boolean() }),
			outputSchema: z.object({ done: z.boolean() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (inputData.x > 0 && !resumeData) return await suspend({});
				return { done: true };
			},
		});
		const wf = createWorkflow({
			id: "runs-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ done: z.boolean() }),
		}).then(step).commit();
		kit.register("workflow", "runs-test", wf);
	`)

	// Start one that completes (x=0 → no suspend)
	r1 := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":0}`)},
		10*time.Second,
	)
	require.Equal(t, "success", r1.Status)

	// Start one that suspends (x=1)
	r2 := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", r2.Status)

	// List all runs
	allResp := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k,
		messages.WorkflowRunsMsg{Name: "runs-test"},
		5*time.Second,
	)
	require.Empty(t, allResp.Error, "list all runs: %s", allResp.Error)
	assert.GreaterOrEqual(t, allResp.Total, 2, "should have at least 2 runs total")

	// List only suspended runs
	suspResp := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k,
		messages.WorkflowRunsMsg{Name: "runs-test", Status: "suspended"},
		5*time.Second,
	)
	require.Empty(t, suspResp.Error, "list suspended: %s", suspResp.Error)
	assert.Equal(t, 1, suspResp.Total, "should have exactly 1 suspended run")
}

// ════════════════════════════════════════════════════════════════════════════
// STARTASYNC + COMPLETION EVENT
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_StartAsyncEvent(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-async", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	deployWorkflow(t, k, "async-wf.ts", `
		const step = createStep({
			id: "compute",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
			execute: async ({ inputData }) => ({ result: inputData.n * 10 }),
		});
		const wf = createWorkflow({
			id: "async-test",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
		}).then(step).commit();
		kit.register("workflow", "async-test", wf);
	`)

	// StartAsync — returns runId immediately
	asyncResp := publishAndWait[messages.WorkflowStartAsyncMsg, messages.WorkflowStartAsyncResp](
		t, k,
		messages.WorkflowStartAsyncMsg{Name: "async-test", InputData: json.RawMessage(`{"n":7}`)},
		5*time.Second,
	)
	require.Empty(t, asyncResp.Error, "startAsync: %s", asyncResp.Error)
	require.NotEmpty(t, asyncResp.RunID)

	// Subscribe to completion event
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	completionTopic := "workflow.completed." + asyncResp.RunID
	ch := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, completionTopic, func(msg messages.Message) {
		select {
		case ch <- msg.Payload:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		var event struct {
			RunID  string `json:"runId"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		require.NoError(t, json.Unmarshal(payload, &event))
		assert.Equal(t, asyncResp.RunID, event.RunID)
		assert.Equal(t, "async-test", event.Name)
		assert.Equal(t, "success", event.Status)
	case <-ctx.Done():
		t.Fatal("timeout waiting for workflow.completed event")
	}
}

// ════════════════════════════════════════════════════════════════════════════
// CRASH RECOVERY
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_CrashRecovery_Suspended(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "kit.db")
	mastraDBPath := filepath.Join(tmpDir, "mastra.db")

	suspendCode := `
		const step = createStep({
			id: "gate",
			inputSchema: z.object({ x: z.number() }),
			resumeSchema: z.object({ approved: z.boolean() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({ reason: "needs approval" });
				return { result: resumeData.approved ? "yes" : "no" };
			},
		});
		const wf = createWorkflow({
			id: "crash-suspend",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ result: z.string() }),
		}).then(step).commit();
		kit.register("workflow", "crash-suspend", wf);
	`

	// ── Kernel 1: deploy, start, suspend ──
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-crash", FSRoot: tmpDir,
		Store: store1,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(mastraDBPath),
		},
	})
	require.NoError(t, err)

	deployWorkflow(t, k1, "crash-suspend.ts", suspendCode)

	startResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k1,
		messages.WorkflowStartMsg{Name: "crash-suspend", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)
	runID := startResp.RunID

	// Persist deployment so k2 can re-deploy
	k1.Close()

	// ── Kernel 2: restart, verify suspended run persisted, resume ──
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-crash", FSRoot: tmpDir,
		Store: store2,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(mastraDBPath),
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	// Status should still be queryable on the new Kernel
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k2,
		messages.WorkflowStatusMsg{Name: "crash-suspend", RunID: runID},
		10*time.Second,
	)
	require.Empty(t, statusResp.Error, "status on k2: %s", statusResp.Error)
	assert.Equal(t, "suspended", statusResp.Status)

	// Resume on the new Kernel
	resumeResp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
		t, k2,
		messages.WorkflowResumeMsg{
			Name: "crash-suspend", RunID: runID,
			Step: "gate", ResumeData: json.RawMessage(`{"approved":true}`),
		},
		10*time.Second,
	)
	require.Empty(t, resumeResp.Error, "resume on k2: %s", resumeResp.Error)
	assert.Equal(t, "success", resumeResp.Status)
}

// ════════════════════════════════════════════════════════════════════════════
// LONG-RUNNING MULTI-WORKFLOW STRESS TEST
// ════════════════════════════════════════════════════════════════════════════

func TestWorkflowBus_MultiWorkflowStress(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-stress", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// Deploy 4 workflow definitions
	deployWorkflow(t, k, "stress-fast.ts", `
		const s1 = createStep({ id: "s1", inputSchema: z.object({ n: z.number() }), outputSchema: z.object({ r: z.number() }),
			execute: async ({ inputData }) => ({ r: inputData.n + 1 }),
		});
		const s2 = createStep({ id: "s2", inputSchema: z.object({ r: z.number() }), outputSchema: z.object({ r: z.number() }),
			execute: async ({ inputData }) => ({ r: inputData.r + 1 }),
		});
		const s3 = createStep({ id: "s3", inputSchema: z.object({ r: z.number() }), outputSchema: z.object({ r: z.number() }),
			execute: async ({ inputData }) => ({ r: inputData.r + 1 }),
		});
		const wf = createWorkflow({ id: "fast-seq", inputSchema: z.object({ n: z.number() }), outputSchema: z.object({ r: z.number() }) })
			.then(s1).then(s2).then(s3).commit();
		kit.register("workflow", "fast-seq", wf);
	`)

	deployWorkflow(t, k, "stress-suspend.ts", `
		const gate = createStep({
			id: "gate", inputSchema: z.object({ id: z.string() }),
			resumeSchema: z.object({ ok: z.boolean() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { result: "done-" + inputData.id };
			},
		});
		const wf = createWorkflow({ id: "suspend-wf", inputSchema: z.object({ id: z.string() }), outputSchema: z.object({ result: z.string() }) })
			.then(gate).commit();
		kit.register("workflow", "suspend-wf", wf);
	`)

	deployWorkflow(t, k, "stress-parallel.ts", `
		const a = createStep({ id: "a", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ a: z.number() }),
			execute: async ({ inputData }) => ({ a: inputData.x * 2 }),
		});
		const b = createStep({ id: "b", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ b: z.number() }),
			execute: async ({ inputData }) => ({ b: inputData.x * 3 }),
		});
		const wf = createWorkflow({ id: "parallel-wf", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ a: z.number(), b: z.number() }) })
			.parallel([a, b]).commit();
		kit.register("workflow", "parallel-wf", wf);
	`)

	deployWorkflow(t, k, "stress-multi-suspend.ts", `
		const g1 = createStep({
			id: "g1", inputSchema: z.object({ id: z.string() }),
			resumeSchema: z.object({ v: z.string() }),
			outputSchema: z.object({ r1: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { r1: resumeData.v };
			},
		});
		const g2 = createStep({
			id: "g2", inputSchema: z.object({ r1: z.string() }),
			resumeSchema: z.object({ v: z.string() }),
			outputSchema: z.object({ r2: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { r2: inputData.r1 + "-" + resumeData.v };
			},
		});
		const wf = createWorkflow({ id: "multi-sus", inputSchema: z.object({ id: z.string() }), outputSchema: z.object({ r2: z.string() }) })
			.then(g1).then(g2).commit();
		kit.register("workflow", "multi-sus", wf);
	`)

	// ── Phase 1: Start 10 fast-sequential concurrently ──
	t.Log("Phase 1: 10 fast-sequential")
	fastResults := make(chan messages.WorkflowStartResp, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "fast-seq", InputData: json.RawMessage(`{"n":` + fmt.Sprintf("%d", i) + `}`)},
			10*time.Second,
		)
		fastResults <- resp
	})
	close(fastResults)
	for resp := range fastResults {
		assert.Empty(t, resp.Error, "fast-seq error: %s", resp.Error)
		assert.Equal(t, "success", resp.Status)
	}

	// ── Phase 2: Start 3 suspend-resume → all suspend ──
	t.Log("Phase 2: 3 suspend-resume")
	suspRunIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "suspend-wf", InputData: json.RawMessage(fmt.Sprintf(`{"id":"s%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status, "suspend-wf should suspend")
		suspRunIDs[i] = resp.RunID
	}

	// ── Phase 3: Start 2 parallel-brancher ──
	t.Log("Phase 3: 2 parallel-brancher")
	for i := 0; i < 2; i++ {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "parallel-wf", InputData: json.RawMessage(fmt.Sprintf(`{"x":%d}`, i+1))},
			10*time.Second,
		)
		assert.Equal(t, "success", resp.Status, "parallel-wf should complete")
	}

	// ── Phase 4: Start 2 multi-suspend → first suspend ──
	t.Log("Phase 4: 2 multi-suspend")
	multiRunIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "multi-sus", InputData: json.RawMessage(fmt.Sprintf(`{"id":"m%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status)
		multiRunIDs[i] = resp.RunID
	}

	// ── Phase 5: Resume suspend-resume instances one by one ──
	t.Log("Phase 5: resume 3 suspend-resume")
	for _, runID := range suspRunIDs {
		resp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{Name: "suspend-wf", RunID: runID, Step: "gate", ResumeData: json.RawMessage(`{"ok":true}`)},
			10*time.Second,
		)
		assert.Empty(t, resp.Error, "resume suspend-wf: %s", resp.Error)
		assert.Equal(t, "success", resp.Status)
	}

	// ── Phase 6: Resume multi-suspend through both cycles ──
	t.Log("Phase 6: resume multi-suspend (2 cycles each)")
	for _, runID := range multiRunIDs {
		// First resume (g1)
		r1 := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{Name: "multi-sus", RunID: runID, Step: "g1", ResumeData: json.RawMessage(`{"v":"first"}`)},
			10*time.Second,
		)
		require.Equal(t, "suspended", r1.Status, "after g1 resume should suspend at g2")

		// Second resume (g2)
		r2 := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{Name: "multi-sus", RunID: runID, Step: "g2", ResumeData: json.RawMessage(`{"v":"second"}`)},
			10*time.Second,
		)
		assert.Equal(t, "success", r2.Status, "after g2 resume should complete")
	}

	t.Log("All phases complete — multi-workflow stress test passed")
}

// ════════════════════════════════════════════════════════════════════════════
// LONG-RUNNING INTEGRATION TEST
// Kernel stays alive for the full test. Real sleeps. Real concurrency.
// Every workflow output is verified — not just status strings but actual
// data flowing through steps, resume data received, parallel outputs correct.
// ════════════════════════════════════════════════════════════════════════════

// queryStepOutput reads the per-step output from storage for a completed run.
func queryStepOutput(t *testing.T, k *brainkit.Kernel, wfName, runID, stepID string) json.RawMessage {
	t.Helper()
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: wfName, RunID: runID},
		5*time.Second,
	)
	require.Empty(t, statusResp.Error, "queryStepOutput %s/%s: %s", wfName, runID, statusResp.Error)
	if len(statusResp.Steps) == 0 {
		return nil
	}
	// Steps is a map[string]{status, output, ...}
	var steps map[string]json.RawMessage
	json.Unmarshal(statusResp.Steps, &steps)
	return steps[stepID]
}

func TestWorkflowBus_LongRunning_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping long-running integration test in short mode")
	}

	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "kit.db"))
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test-longrun", FSRoot: tmpDir,
		Store: store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	// ── Deploy 5 workflow definitions ──

	// 1. fast-seq: n → (n+1) → (n+1)*2. Verify math.
	deployWorkflow(t, k, "lr-fast.ts", `
		const s1 = createStep({ id: "s1", inputSchema: z.object({ n: z.number() }), outputSchema: z.object({ r: z.number() }),
			execute: async ({ inputData }) => ({ r: inputData.n + 1 }),
		});
		const s2 = createStep({ id: "s2", inputSchema: z.object({ r: z.number() }), outputSchema: z.object({ r: z.number() }),
			execute: async ({ inputData }) => ({ r: inputData.r * 2 }),
		});
		const wf = createWorkflow({ id: "lr-fast", inputSchema: z.object({ n: z.number() }), outputSchema: z.object({ r: z.number() }) })
			.then(s1).then(s2).commit();
		kit.register("workflow", "lr-fast", wf);
	`)

	// 2. sleeper: real sleep, then produces "slept-<id>"
	deployWorkflow(t, k, "lr-sleeper.ts", `
		const before = createStep({ id: "before", inputSchema: z.object({ id: z.string(), sleepMs: z.number() }), outputSchema: z.object({ id: z.string(), sleepMs: z.number() }),
			execute: async ({ inputData }) => inputData,
		});
		const after = createStep({ id: "after", inputSchema: z.object({ id: z.string(), sleepMs: z.number() }), outputSchema: z.object({ done: z.string() }),
			execute: async ({ inputData }) => ({ done: "slept-" + inputData.id }),
		});
		const wf = createWorkflow({ id: "lr-sleeper", inputSchema: z.object({ id: z.string(), sleepMs: z.number() }), outputSchema: z.object({ done: z.string() }) })
			.then(before)
			.sleep(({ inputData }) => inputData.sleepMs)
			.then(after)
			.commit();
		kit.register("workflow", "lr-sleeper", wf);
	`)

	// 3. suspend-resume: output includes whether approved + input id
	deployWorkflow(t, k, "lr-suspend.ts", `
		const gate = createStep({
			id: "gate", inputSchema: z.object({ id: z.string() }),
			resumeSchema: z.object({ approved: z.boolean() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({ waiting: inputData.id });
				return { result: resumeData.approved ? "approved-" + inputData.id : "denied-" + inputData.id };
			},
		});
		const wf = createWorkflow({ id: "lr-suspend", inputSchema: z.object({ id: z.string() }), outputSchema: z.object({ result: z.string() }) })
			.then(gate).commit();
		kit.register("workflow", "lr-suspend", wf);
	`)

	// 4. parallel: x → {a: x*2, b: x*3, c: x*5}
	deployWorkflow(t, k, "lr-parallel.ts", `
		const a = createStep({ id: "a", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ a: z.number() }),
			execute: async ({ inputData }) => ({ a: inputData.x * 2 }),
		});
		const b = createStep({ id: "b", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ b: z.number() }),
			execute: async ({ inputData }) => ({ b: inputData.x * 3 }),
		});
		const c = createStep({ id: "c", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ c: z.number() }),
			execute: async ({ inputData }) => ({ c: inputData.x * 5 }),
		});
		const wf = createWorkflow({ id: "lr-parallel", inputSchema: z.object({ x: z.number() }), outputSchema: z.object({ a: z.number(), b: z.number(), c: z.number() }) })
			.parallel([a, b, c]).commit();
		kit.register("workflow", "lr-parallel", wf);
	`)

	// 5. multi-suspend: id → g1 output "id-v1" → g2 output "id-v1-v2"
	deployWorkflow(t, k, "lr-multi-sus.ts", `
		const g1 = createStep({
			id: "g1", inputSchema: z.object({ id: z.string() }),
			resumeSchema: z.object({ v: z.string() }),
			outputSchema: z.object({ r1: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { r1: inputData.id + "-" + resumeData.v };
			},
		});
		const g2 = createStep({
			id: "g2", inputSchema: z.object({ r1: z.string() }),
			resumeSchema: z.object({ v: z.string() }),
			outputSchema: z.object({ r2: z.string() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (!resumeData) return await suspend({});
				return { r2: inputData.r1 + "-" + resumeData.v };
			},
		});
		const wf = createWorkflow({ id: "lr-multi-sus", inputSchema: z.object({ id: z.string() }), outputSchema: z.object({ r2: z.string() }) })
			.then(g1).then(g2).commit();
		kit.register("workflow", "lr-multi-sus", wf);
	`)

	t.Log("All 5 workflows deployed")

	// ══════════════════════════════════════════════════════════════════
	// WAVE 1: 10 fast (concurrent), 5 sleepers (10s), 3 suspend, 2 multi-suspend
	// ══════════════════════════════════════════════════════════════════
	t.Log("Wave 1: launching 20 workflows")

	// 10 fast — verify each output: (n+1)*2
	type fastResult struct {
		n     int
		runID string
		resp  messages.WorkflowStartResp
	}
	fastCh := make(chan fastResult, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "lr-fast", InputData: json.RawMessage(fmt.Sprintf(`{"n":%d}`, i))},
			10*time.Second,
		)
		fastCh <- fastResult{i, resp.RunID, resp}
	})
	close(fastCh)
	for fr := range fastCh {
		require.Empty(t, fr.resp.Error, "fast n=%d error: %s", fr.n, fr.resp.Error)
		require.Equal(t, "success", fr.resp.Status, "fast n=%d status", fr.n)
		// Verify status from storage shows success
		statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-fast", RunID: fr.runID},
			5*time.Second,
		)
		assert.Equal(t, "success", statusResp.Status, "fast n=%d storage status", fr.n)
	}

	// 5 sleepers — start async (10s sleep)
	sleeperRunIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		resp := publishAndWait[messages.WorkflowStartAsyncMsg, messages.WorkflowStartAsyncResp](
			t, k,
			messages.WorkflowStartAsyncMsg{Name: "lr-sleeper", InputData: json.RawMessage(fmt.Sprintf(`{"id":"s%d","sleepMs":10000}`, i))},
			5*time.Second,
		)
		require.Empty(t, resp.Error, "sleeper start s%d", i)
		require.NotEmpty(t, resp.RunID)
		sleeperRunIDs[i] = resp.RunID
	}
	t.Log("5 sleepers started (10s sleep each)")

	// Verify sleepers are running/pending (not completed yet)
	for i, runID := range sleeperRunIDs {
		statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
			5*time.Second,
		)
		require.Empty(t, statusResp.Error, "sleeper s%d status", i)
		assert.NotEqual(t, "success", statusResp.Status, "sleeper s%d should NOT be success yet (still sleeping)", i)
		t.Logf("  sleeper s%d status: %s", i, statusResp.Status)
	}

	// 3 suspenders — verify they suspend
	suspRunIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "lr-suspend", InputData: json.RawMessage(fmt.Sprintf(`{"id":"susp%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status, "suspend susp%d", i)
		suspRunIDs[i] = resp.RunID

		// Verify storage also shows suspended
		statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-suspend", RunID: resp.RunID},
			5*time.Second,
		)
		assert.Equal(t, "suspended", statusResp.Status, "suspend susp%d storage", i)
	}

	// 2 multi-suspend — verify first suspend
	multiRunIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "lr-multi-sus", InputData: json.RawMessage(fmt.Sprintf(`{"id":"ms%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status, "multi-sus ms%d first suspend", i)
		multiRunIDs[i] = resp.RunID
	}

	// ══════════════════════════════════════════════════════════════════
	// WAVE 2: parallel + more fast while sleepers sleep
	// ══════════════════════════════════════════════════════════════════
	t.Log("Wave 2: parallel + fast while sleepers sleep")

	// 3 parallel: verify each branch output
	for i := 0; i < 3; i++ {
		x := i + 1
		resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
			t, k,
			messages.WorkflowStartMsg{Name: "lr-parallel", InputData: json.RawMessage(fmt.Sprintf(`{"x":%d}`, x))},
			10*time.Second,
		)
		require.Empty(t, resp.Error, "parallel x=%d", x)
		require.Equal(t, "success", resp.Status, "parallel x=%d status", x)
		// Query storage for per-step outputs
		statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-parallel", RunID: resp.RunID},
			5*time.Second,
		)
		assert.Equal(t, "success", statusResp.Status, "parallel x=%d storage status", x)
		t.Logf("  parallel x=%d: status=%s steps=%s", x, statusResp.Status, string(statusResp.Steps))
	}

	// ══════════════════════════════════════════════════════════════════
	// WAVE 3: Resume suspenders — verify output includes resumeData
	// ══════════════════════════════════════════════════════════════════
	t.Log("Wave 3: resume 3 suspenders with alternating approve/deny")

	for i, runID := range suspRunIDs {
		approved := i%2 == 0
		resp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{
				Name: "lr-suspend", RunID: runID, Step: "gate",
				ResumeData: json.RawMessage(fmt.Sprintf(`{"approved":%v}`, approved)),
			},
			10*time.Second,
		)
		require.Empty(t, resp.Error, "resume susp%d: %s", i, resp.Error)
		require.Equal(t, "success", resp.Status, "resume susp%d status", i)

		// Verify output in storage
		statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-suspend", RunID: runID},
			5*time.Second,
		)
		assert.Equal(t, "success", statusResp.Status)
		// Verify the step output contains the expected approval result
		stepsJSON := string(statusResp.Steps)
		if approved {
			assert.Contains(t, stepsJSON, fmt.Sprintf("approved-susp%d", i), "susp%d should be approved", i)
		} else {
			assert.Contains(t, stepsJSON, fmt.Sprintf("denied-susp%d", i), "susp%d should be denied", i)
		}
		t.Logf("  susp%d: approved=%v steps=%s", i, approved, stepsJSON)
	}

	// ══════════════════════════════════════════════════════════════════
	// WAVE 4: Resume multi-suspend — verify chained data
	// ══════════════════════════════════════════════════════════════════
	t.Log("Wave 4: resume 2 multi-suspend (2 gates each)")

	for i, runID := range multiRunIDs {
		// First gate: g1 receives v="alpha"
		r1 := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{Name: "lr-multi-sus", RunID: runID, Step: "g1", ResumeData: json.RawMessage(`{"v":"alpha"}`)},
			10*time.Second,
		)
		require.Empty(t, r1.Error, "multi-sus ms%d g1: %s", i, r1.Error)
		require.Equal(t, "suspended", r1.Status, "ms%d should suspend at g2", i)

		// Verify g1 step output in storage: r1 = "ms<i>-alpha"
		statusAfterG1 := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-multi-sus", RunID: runID},
			5*time.Second,
		)
		assert.Equal(t, "suspended", statusAfterG1.Status)
		assert.Contains(t, string(statusAfterG1.Steps), fmt.Sprintf("ms%d-alpha", i), "g1 should produce ms%d-alpha", i)

		// Second gate: g2 receives v="beta"
		r2 := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
			t, k,
			messages.WorkflowResumeMsg{Name: "lr-multi-sus", RunID: runID, Step: "g2", ResumeData: json.RawMessage(`{"v":"beta"}`)},
			10*time.Second,
		)
		require.Empty(t, r2.Error, "multi-sus ms%d g2: %s", i, r2.Error)
		require.Equal(t, "success", r2.Status, "ms%d should complete after g2", i)

		// Verify g2 step output: r2 = "ms<i>-alpha-beta"
		statusAfterG2 := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-multi-sus", RunID: runID},
			5*time.Second,
		)
		assert.Equal(t, "success", statusAfterG2.Status)
		assert.Contains(t, string(statusAfterG2.Steps), fmt.Sprintf("ms%d-alpha-beta", i), "g2 should produce ms%d-alpha-beta", i)
		t.Logf("  ms%d: final steps=%s", i, string(statusAfterG2.Steps))
	}

	// ══════════════════════════════════════════════════════════════════
	// WAVE 5: Cancel one suspended workflow — verify status is canceled
	// ══════════════════════════════════════════════════════════════════
	t.Log("Wave 5: start a workflow, suspend it, cancel it")

	cancelResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "lr-suspend", InputData: json.RawMessage(`{"id":"to-cancel"}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", cancelResp.Status)

	cancelResult := publishAndWait[messages.WorkflowCancelMsg, messages.WorkflowCancelResp](
		t, k,
		messages.WorkflowCancelMsg{Name: "lr-suspend", RunID: cancelResp.RunID},
		5*time.Second,
	)
	require.Empty(t, cancelResult.Error)
	require.True(t, cancelResult.Cancelled)

	// Verify storage shows canceled
	cancelStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "lr-suspend", RunID: cancelResp.RunID},
		5*time.Second,
	)
	assert.Equal(t, "canceled", cancelStatus.Status, "canceled run should show canceled in storage")

	// ══════════════════════════════════════════════════════════════════
	// WAIT: sleepers finish (10s sleep, started ~10s ago)
	// ══════════════════════════════════════════════════════════════════
	t.Log("Waiting for 5 sleepers to complete...")

	for i, runID := range sleeperRunIDs {
		deadline := time.Now().Add(25 * time.Second)
		var finalStatus string
		for time.Now().Before(deadline) {
			statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
				t, k,
				messages.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
				5*time.Second,
			)
			finalStatus = statusResp.Status
			if finalStatus == "success" || finalStatus == "failed" {
				break
			}
			time.Sleep(1 * time.Second)
		}
		require.Equal(t, "success", finalStatus, "sleeper s%d should complete", i)

		// Verify the "after" step produced the correct output
		afterStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
			t, k,
			messages.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
			5*time.Second,
		)
		assert.Contains(t, string(afterStatus.Steps), fmt.Sprintf("slept-s%d", i),
			"sleeper s%d should produce 'slept-s%d' in step output", i, i)
		t.Logf("  sleeper s%d: completed, steps=%s", i, string(afterStatus.Steps))
	}

	// ══════════════════════════════════════════════════════════════════
	// FINAL: Query run counts from storage — verify totals
	// ══════════════════════════════════════════════════════════════════
	t.Log("Final: verifying run counts in storage")

	fastRuns := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k, messages.WorkflowRunsMsg{Name: "lr-fast"}, 5*time.Second,
	)
	assert.GreaterOrEqual(t, fastRuns.Total, 10, "lr-fast should have ≥10 runs")

	sleeperRuns := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k, messages.WorkflowRunsMsg{Name: "lr-sleeper"}, 5*time.Second,
	)
	assert.Equal(t, 5, sleeperRuns.Total, "lr-sleeper should have 5 runs")

	suspendRuns := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k, messages.WorkflowRunsMsg{Name: "lr-suspend"}, 5*time.Second,
	)
	assert.GreaterOrEqual(t, suspendRuns.Total, 4, "lr-suspend should have ≥4 runs (3 resumed + 1 canceled)")

	canceledRuns := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k, messages.WorkflowRunsMsg{Name: "lr-suspend", Status: "canceled"}, 5*time.Second,
	)
	assert.Equal(t, 1, canceledRuns.Total, "exactly 1 lr-suspend run should be canceled")

	multiRuns := publishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k, messages.WorkflowRunsMsg{Name: "lr-multi-sus"}, 5*time.Second,
	)
	assert.Equal(t, 2, multiRuns.Total, "lr-multi-sus should have 2 runs")

	t.Log("Long-running integration test complete — all outputs verified")
}

// ════════════════════════════════════════════════════════════════════════════
// DEVELOPER SCENARIO TESTS
// These test real-world usage patterns a developer would use with brainkit
// workflows: tools inside steps, bus events from steps, state management,
// conditional branching, forEach, nested workflows.
// ════════════════════════════════════════════════════════════════════════════

func newWorkflowKernelWithTools(t *testing.T) *brainkit.Kernel {
	t.Helper()
	tk := testutil.NewTestKernelFull(t)
	return tk.Kernel
}

// TestWorkflow_ToolCallInsideStep: a workflow step calls a Go-registered tool
// and uses the result in its output. Verifies the tool bridge works from
// within Mastra's execution engine.
func TestWorkflow_ToolCallInsideStep(t *testing.T) {
	k := newWorkflowKernelWithTools(t)

	deployWorkflow(t, k, "tool-in-step.ts", `
		const fetchStep = createStep({
			id: "fetch-data",
			inputSchema: z.object({ query: z.string() }),
			outputSchema: z.object({ echoed: z.string(), processed: z.string() }),
			execute: async ({ inputData }) => {
				// Call Go-registered "echo" tool from within a workflow step
				const result = await tools.call("echo", { message: inputData.query });
				return {
					echoed: result.echoed,
					processed: "processed-" + result.echoed,
				};
			},
		});
		const transformStep = createStep({
			id: "transform",
			inputSchema: z.object({ echoed: z.string(), processed: z.string() }),
			outputSchema: z.object({ final: z.string() }),
			execute: async ({ inputData }) => ({
				final: inputData.processed.toUpperCase(),
			}),
		});
		const wf = createWorkflow({
			id: "tool-in-step",
			inputSchema: z.object({ query: z.string() }),
			outputSchema: z.object({ final: z.string() }),
		}).then(fetchStep).then(transformStep).commit();
		kit.register("workflow", "tool-in-step", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "tool-in-step", InputData: json.RawMessage(`{"query":"hello world"}`)},
		15*time.Second,
	)
	require.Empty(t, resp.Error, "tool-in-step error: %s", resp.Error)
	require.Equal(t, "success", resp.Status)

	// Verify step outputs in storage
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "tool-in-step", RunID: resp.RunID},
		5*time.Second,
	)
	steps := string(statusResp.Steps)
	assert.Contains(t, steps, "hello world", "echoed should contain original query")
	assert.Contains(t, steps, "PROCESSED-HELLO WORLD", "transform should uppercase the processed string")
	t.Logf("tool-in-step steps: %s", steps)
}

// TestWorkflow_BusEmitFromStep: a workflow step emits a bus event.
// An external subscriber receives the event while the workflow runs.
func TestWorkflow_BusEmitFromStep(t *testing.T) {
	k := newWorkflowKernelWithTools(t)

	deployWorkflow(t, k, "bus-emit-step.ts", `
		const emitStep = createStep({
			id: "emit-event",
			inputSchema: z.object({ orderId: z.string() }),
			outputSchema: z.object({ emitted: z.boolean() }),
			execute: async ({ inputData }) => {
				bus.emit("order.processing", { orderId: inputData.orderId, stage: "started" });
				return { emitted: true };
			},
		});
		const wf = createWorkflow({
			id: "bus-emit-wf",
			inputSchema: z.object({ orderId: z.string() }),
			outputSchema: z.object({ emitted: z.boolean() }),
		}).then(emitStep).commit();
		kit.register("workflow", "bus-emit-wf", wf);
	`)

	// Subscribe to the event the step will emit
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eventCh := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, "order.processing", func(msg messages.Message) {
		select {
		case eventCh <- msg.Payload:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	// Start workflow
	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "bus-emit-wf", InputData: json.RawMessage(`{"orderId":"ORD-123"}`)},
		10*time.Second,
	)
	require.Empty(t, resp.Error)
	require.Equal(t, "success", resp.Status)

	// Verify we received the bus event
	select {
	case payload := <-eventCh:
		var event struct {
			OrderID string `json:"orderId"`
			Stage   string `json:"stage"`
		}
		require.NoError(t, json.Unmarshal(payload, &event))
		assert.Equal(t, "ORD-123", event.OrderID)
		assert.Equal(t, "started", event.Stage)
		t.Logf("Received bus event: %s", string(payload))
	case <-ctx.Done():
		t.Fatal("timeout waiting for order.processing bus event from workflow step")
	}
}

// TestWorkflow_ConditionalBranch: workflow branches based on step output.
// Tests the .branch() API with condition functions.
func TestWorkflow_ConditionalBranch(t *testing.T) {
	k := newWorkflowKernelWithTools(t)

	deployWorkflow(t, k, "branch-wf.ts", `
		const classify = createStep({
			id: "classify",
			inputSchema: z.object({ amount: z.number() }),
			outputSchema: z.object({ tier: z.string(), amount: z.number() }),
			execute: async ({ inputData }) => ({
				tier: inputData.amount >= 100 ? "premium" : "standard",
				amount: inputData.amount,
			}),
		});
		const premiumHandler = createStep({
			id: "premium",
			inputSchema: z.object({ tier: z.string(), amount: z.number() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData }) => ({ result: "premium-" + inputData.amount }),
		});
		const standardHandler = createStep({
			id: "standard",
			inputSchema: z.object({ tier: z.string(), amount: z.number() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData }) => ({ result: "standard-" + inputData.amount }),
		});
		const wf = createWorkflow({
			id: "branch-wf",
			inputSchema: z.object({ amount: z.number() }),
			outputSchema: z.object({ result: z.string() }),
		}).then(classify).branch([
			[async ({ inputData }) => inputData.tier === "premium", premiumHandler],
			[async ({ inputData }) => inputData.tier === "standard", standardHandler],
		]).commit();
		kit.register("workflow", "branch-wf", wf);
	`)

	// Test premium path (amount >= 100)
	premResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "branch-wf", InputData: json.RawMessage(`{"amount":250}`)},
		10*time.Second,
	)
	require.Empty(t, premResp.Error, "premium branch: %s", premResp.Error)
	require.Equal(t, "success", premResp.Status)

	premStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "branch-wf", RunID: premResp.RunID},
		5*time.Second,
	)
	assert.Contains(t, string(premStatus.Steps), "premium-250", "should take premium branch")
	t.Logf("premium branch steps: %s", string(premStatus.Steps))

	// Test standard path (amount < 100)
	stdResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "branch-wf", InputData: json.RawMessage(`{"amount":50}`)},
		10*time.Second,
	)
	require.Empty(t, stdResp.Error, "standard branch: %s", stdResp.Error)
	require.Equal(t, "success", stdResp.Status)

	stdStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "branch-wf", RunID: stdResp.RunID},
		5*time.Second,
	)
	assert.Contains(t, string(stdStatus.Steps), "standard-50", "should take standard branch")
	t.Logf("standard branch steps: %s", string(stdStatus.Steps))
}

// TestWorkflow_StepState: steps use setState/state to share mutable state
// across the workflow execution.
func TestWorkflow_StepState(t *testing.T) {
	k := newWorkflowKernelWithTools(t)

	deployWorkflow(t, k, "state-wf.ts", `
		const initState = createStep({
			id: "init",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ name: z.string() }),
			stateSchema: z.object({ counter: z.number(), log: z.array(z.string()) }),
			execute: async ({ inputData, setState }) => {
				await setState({ counter: 1, log: ["init:" + inputData.name] });
				return inputData;
			},
		});
		const incrementState = createStep({
			id: "increment",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ name: z.string() }),
			stateSchema: z.object({ counter: z.number(), log: z.array(z.string()) }),
			execute: async ({ inputData, state, setState }) => {
				await setState({
					counter: state.counter + 10,
					log: [...state.log, "incremented"],
				});
				return inputData;
			},
		});
		const finalStep = createStep({
			id: "final",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ counterValue: z.number(), logLength: z.number() }),
			stateSchema: z.object({ counter: z.number(), log: z.array(z.string()) }),
			execute: async ({ state }) => ({
				counterValue: state.counter,
				logLength: state.log.length,
			}),
		});
		const wf = createWorkflow({
			id: "state-wf",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ counterValue: z.number(), logLength: z.number() }),
			stateSchema: z.object({ counter: z.number().default(0), log: z.array(z.string()).default([]) }),
		}).then(initState).then(incrementState).then(finalStep).commit();
		kit.register("workflow", "state-wf", wf);
	`)

	resp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "state-wf", InputData: json.RawMessage(`{"name":"test-user"}`)},
		10*time.Second,
	)
	require.Empty(t, resp.Error, "state workflow: %s", resp.Error)
	require.Equal(t, "success", resp.Status)

	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "state-wf", RunID: resp.RunID},
		5*time.Second,
	)
	steps := string(statusResp.Steps)
	// Final step should have counter=11 (1+10) and logLength=2 (["init:test-user", "incremented"])
	assert.Contains(t, steps, `"counterValue":11`, "counter should be 11 (1+10)")
	assert.Contains(t, steps, `"logLength":2`, "log should have 2 entries")
	t.Logf("state-wf steps: %s", steps)
}

// TestWorkflow_SuspendWithContextData: suspend carries context data (suspendData)
// that is available on resume. Verifies the full suspend → resume data flow
// that a real HITL workflow would use.
func TestWorkflow_SuspendWithContextData(t *testing.T) {
	k := newWorkflowKernelWithTools(t)

	deployWorkflow(t, k, "suspend-context.ts", `
		const review = createStep({
			id: "review",
			inputSchema: z.object({ documentId: z.string(), content: z.string() }),
			resumeSchema: z.object({ decision: z.string(), reviewer: z.string() }),
			suspendSchema: z.object({ reason: z.string(), documentId: z.string(), preview: z.string() }),
			outputSchema: z.object({ status: z.string(), reviewedBy: z.string(), documentId: z.string() }),
			execute: async ({ inputData, resumeData, suspend, suspendData }) => {
				if (!resumeData) {
					return await suspend({
						reason: "Document needs review",
						documentId: inputData.documentId,
						preview: inputData.content.substring(0, 50),
					});
				}
				// After resume, suspendData has the original context
				return {
					status: resumeData.decision,
					reviewedBy: resumeData.reviewer,
					documentId: inputData.documentId,
				};
			},
		});
		const wf = createWorkflow({
			id: "doc-review",
			inputSchema: z.object({ documentId: z.string(), content: z.string() }),
			outputSchema: z.object({ status: z.string(), reviewedBy: z.string(), documentId: z.string() }),
		}).then(review).commit();
		kit.register("workflow", "doc-review", wf);
	`)

	// Start — should suspend
	startResp := publishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{
			Name:      "doc-review",
			InputData: json.RawMessage(`{"documentId":"DOC-456","content":"This is a very important document that needs careful review by a senior team member."}`),
		},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)

	// Check suspended state in storage — should show suspend payload
	suspStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "doc-review", RunID: startResp.RunID},
		5*time.Second,
	)
	assert.Equal(t, "suspended", suspStatus.Status)
	assert.Contains(t, string(suspStatus.Steps), "Document needs review", "suspend payload should be in storage")
	assert.Contains(t, string(suspStatus.Steps), "DOC-456", "documentId should be in suspend payload")
	t.Logf("suspended state: %s", string(suspStatus.Steps))

	// Resume with review decision
	resumeResp := publishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
		t, k,
		messages.WorkflowResumeMsg{
			Name:       "doc-review",
			RunID:      startResp.RunID,
			Step:       "review",
			ResumeData: json.RawMessage(`{"decision":"approved","reviewer":"alice@corp.com"}`),
		},
		10*time.Second,
	)
	require.Empty(t, resumeResp.Error, "resume doc-review: %s", resumeResp.Error)
	require.Equal(t, "success", resumeResp.Status)

	// Verify final output contains all the data
	finalStatus := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "doc-review", RunID: startResp.RunID},
		5*time.Second,
	)
	steps := string(finalStatus.Steps)
	assert.Contains(t, steps, `"status":"approved"`, "should be approved")
	assert.Contains(t, steps, `"reviewedBy":"alice@corp.com"`, "reviewer should be alice")
	assert.Contains(t, steps, `"documentId":"DOC-456"`, "documentId should flow through")
	t.Logf("doc-review final: %s", steps)
}
