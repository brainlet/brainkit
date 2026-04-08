package workflows

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/infra/workflow_bus_test.go — persistence + storage paths.
// These create their own kernels with explicit SQLite storage configuration.

func testStorageUpgrade(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	wfDeploy(t, k, "persist-wf.ts", `
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

	resp := wfPublishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "persist-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", resp.Status, "should suspend")
	require.NotEmpty(t, resp.RunID)

	result, err := testutil.EvalTSErr(k, "__check_storage.ts", `
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

func testStatusFromStorage(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test-status", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	wfDeploy(t, k, "status-wf.ts", `
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

	startResp := wfPublishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "status-test", InputData: json.RawMessage(`{"x":5}`)},
		10*time.Second,
	)
	require.Empty(t, startResp.Error)
	require.Equal(t, "success", startResp.Status)

	statusResp := wfPublishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k,
		messages.WorkflowStatusMsg{Name: "status-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, statusResp.Error, "status of completed run should work: %s", statusResp.Error)
	assert.Equal(t, "success", statusResp.Status)
}

func testRuns(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test-runs", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	wfDeploy(t, k, "runs-wf.ts", `
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

	r1 := wfPublishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":0}`)},
		10*time.Second,
	)
	require.Equal(t, "success", r1.Status)

	r2 := wfPublishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k,
		messages.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", r2.Status)

	allResp := wfPublishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k,
		messages.WorkflowRunsMsg{Name: "runs-test"},
		5*time.Second,
	)
	require.Empty(t, allResp.Error, "list all runs: %s", allResp.Error)
	assert.GreaterOrEqual(t, allResp.Total, 2, "should have at least 2 runs total")

	suspResp := wfPublishAndWait[messages.WorkflowRunsMsg, messages.WorkflowRunsResp](
		t, k,
		messages.WorkflowRunsMsg{Name: "runs-test", Status: "suspended"},
		5*time.Second,
	)
	require.Empty(t, suspResp.Error, "list suspended: %s", suspResp.Error)
	assert.Equal(t, 1, suspResp.Total, "should have exactly 1 suspended run")
}

func testStartAsyncEvent(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test-async", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	defer k.Close()

	wfDeploy(t, k, "async-wf.ts", `
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

	asyncResp := wfPublishAndWait[messages.WorkflowStartAsyncMsg, messages.WorkflowStartAsyncResp](
		t, k,
		messages.WorkflowStartAsyncMsg{Name: "async-test", InputData: json.RawMessage(`{"n":7}`)},
		5*time.Second,
	)
	require.Empty(t, asyncResp.Error, "startAsync: %s", asyncResp.Error)
	require.NotEmpty(t, asyncResp.RunID)

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

func testCrashRecoverySuspended(t *testing.T, _ *suite.TestEnv) {
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

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test-crash", FSRoot: tmpDir,
		Store: store1,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(mastraDBPath),
		},
	})
	require.NoError(t, err)

	wfDeploy(t, k1, "crash-suspend.ts", suspendCode)

	startResp := wfPublishAndWait[messages.WorkflowStartMsg, messages.WorkflowStartResp](
		t, k1,
		messages.WorkflowStartMsg{Name: "crash-suspend", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)
	runID := startResp.RunID
	k1.Close()

	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test-crash", FSRoot: tmpDir,
		Store: store2,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(mastraDBPath),
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	statusResp := wfPublishAndWait[messages.WorkflowStatusMsg, messages.WorkflowStatusResp](
		t, k2,
		messages.WorkflowStatusMsg{Name: "crash-suspend", RunID: runID},
		10*time.Second,
	)
	require.Empty(t, statusResp.Error, "status on k2: %s", statusResp.Error)
	assert.Equal(t, "suspended", statusResp.Status)

	resumeResp := wfPublishAndWait[messages.WorkflowResumeMsg, messages.WorkflowResumeResp](
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
