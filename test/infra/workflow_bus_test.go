package infra

import (
	"context"
	"encoding/json"
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

	// Status after cancel should fail (run removed from registry)
	statusResp := publishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, tk.Kernel,
		messages.WorkflowStatusMsg{Name: "cancel-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.NotEmpty(t, statusResp.Error, "status after cancel should fail")
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
