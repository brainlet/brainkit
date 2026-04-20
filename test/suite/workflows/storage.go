package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/workflow"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "modernc.org/sqlite"
)

// Tests migrated from test/infra/workflow_bus_test.go — persistence + storage paths.
// These create their own kernels with explicit SQLite storage configuration.

func testStorageUpgrade(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "test.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
		Modules: []brainkit.Module{workflow.New()},
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

	resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "persist-test", InputData: json.RawMessage(`{"x":1}`)},
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
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test-status", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
		Modules: []brainkit.Module{workflow.New()},
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

	startResp, startMsg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "status-test", InputData: json.RawMessage(`{"x":5}`)},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(startMsg.Payload))
	require.Equal(t, "success", startResp.Status)

	statusResp, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "status-test", RunID: startResp.RunID},
		5*time.Second,
	)
	statusErr := suite.ResponseErrorMessage(statusMsg.Payload)
	require.Empty(t, statusErr, "status of completed run should work: %s", statusErr)
	assert.Equal(t, "success", statusResp.Status)
}

func testRuns(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test-runs", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
		Modules: []brainkit.Module{workflow.New()},
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

	r1, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":0}`)},
		10*time.Second,
	)
	require.Equal(t, "success", r1.Status)

	r2, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "runs-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", r2.Status)

	allResp, allMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k,
		sdk.WorkflowRunsMsg{Name: "runs-test"},
		5*time.Second,
	)
	allErr := suite.ResponseErrorMessage(allMsg.Payload)
	require.Empty(t, allErr, "list all runs: %s", allErr)
	assert.GreaterOrEqual(t, allResp.Total, 2, "should have at least 2 runs total")

	suspResp, suspMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k,
		sdk.WorkflowRunsMsg{Name: "runs-test", Status: "suspended"},
		5*time.Second,
	)
	suspErr := suite.ResponseErrorMessage(suspMsg.Payload)
	require.Empty(t, suspErr, "list suspended: %s", suspErr)
	assert.Equal(t, 1, suspResp.Total, "should have exactly 1 suspended run")
}

func testStartAsyncEventShape(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test-async", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
		Modules: []brainkit.Module{workflow.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	wfDeploy(t, k, "async-wf.ts", `
		const step = createStep({
			id: "compute",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
			execute: async ({ inputData }) => {
				await new Promise(r => setTimeout(r, 100));
				return { result: inputData.n * 10 };
			},
		});
		const wf = createWorkflow({
			id: "async-test",
			inputSchema: z.object({ n: z.number() }),
			outputSchema: z.object({ result: z.number() }),
		}).then(step).commit();
		kit.register("workflow", "async-test", wf);
	`)

	asyncResp, asyncMsg := wfPublishAndWait[sdk.WorkflowStartAsyncMsg, sdk.WorkflowStartAsyncResp](
		t, k,
		sdk.WorkflowStartAsyncMsg{Name: "async-test", InputData: json.RawMessage(`{"n":7}`)},
		5*time.Second,
	)
	asyncErr := suite.ResponseErrorMessage(asyncMsg.Payload)
	require.Empty(t, asyncErr, "startAsync: %s", asyncErr)
	require.NotEmpty(t, asyncResp.RunID)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	completionTopic := "workflow.completed." + asyncResp.RunID
	ch := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, completionTopic, func(msg sdk.Message) {
		select {
		case ch <- msg.Payload:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		var event map[string]any
		require.NoError(t, json.Unmarshal(payload, &event))
		assert.Equal(t, asyncResp.RunID, event["runId"])
		assert.Equal(t, "async-test", event["name"])
		assert.Equal(t, "success", event["status"])
		_, hasSteps := event["steps"]
		assert.True(t, hasSteps, "completion event should carry a steps field")
		_, hasError := event["error"]
		assert.False(t, hasError, "success completion event should not carry an error field")
	case <-ctx.Done():
		t.Fatal("timeout waiting for workflow.completed event")
	}
}

func testCrashRecoverySuspended(t *testing.T, _ *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, "memory", "crash-suspend")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	statusResp, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k2,
		sdk.WorkflowStatusMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	statusErr := suite.ResponseErrorMessage(statusMsg.Payload)
	require.Empty(t, statusErr, "status on k2: %s", statusErr)
	assert.Equal(t, "suspended", statusResp.Status)
}

func testResumeAfterRestart(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "resume-after-restart")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	resumeResp, resumeMsg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
		t, k2,
		sdk.WorkflowResumeMsg{
			Name:       fixture.WorkflowName,
			RunID:      fixture.RunID,
			Step:       "gate",
			ResumeData: json.RawMessage(`{"approved":true}`),
		},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(resumeMsg.Payload))
	assert.Equal(t, "success", resumeResp.Status)
}

func testCancelAfterRestart(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "cancel-after-restart")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	cancelResp, cancelMsg := wfPublishAndWait[sdk.WorkflowCancelMsg, sdk.WorkflowCancelResp](
		t, k2,
		sdk.WorkflowCancelMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(cancelMsg.Payload))
	require.True(t, cancelResp.Cancelled)

	statusResp, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k2,
		sdk.WorkflowStatusMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(statusMsg.Payload))
	assert.Equal(t, "canceled", statusResp.Status)
}

func testRestartAfterRestart(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "restart-after-restart")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	restartResp, restartMsg := wfPublishAndWait[sdk.WorkflowRestartMsg, sdk.WorkflowRestartResp](
		t, k2,
		sdk.WorkflowRestartMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	require.Equal(t, "VALIDATION_ERROR", suite.ResponseCode(restartMsg.Payload))
	assert.Empty(t, restartResp.Status)
	assert.Contains(t, suite.ResponseErrorMessage(restartMsg.Payload), "not active")
}

func testRunsAfterRestart(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "runs-after-restart")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	allResp, allMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k2,
		sdk.WorkflowRunsMsg{Name: fixture.WorkflowName},
		5*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(allMsg.Payload))
	assert.GreaterOrEqual(t, allResp.Total, 1)

	suspendedResp, suspendedMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k2,
		sdk.WorkflowRunsMsg{Name: fixture.WorkflowName, Status: "suspended"},
		5*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(suspendedMsg.Payload))
	assert.Equal(t, 1, suspendedResp.Total)
}

func testCorruptSnapshotFailsCleanly(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "corrupt-snapshot")
	corruptWorkflowSnapshots(t, fixture.MastraDBPath)

	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	_, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k2,
		sdk.WorkflowStatusMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	errMsg := suite.ResponseErrorMessage(statusMsg.Payload)
	require.NotEmpty(t, errMsg)
	assert.True(t, suite.ResponseCode(statusMsg.Payload) != "", "corrupt snapshot should return a typed envelope")
}

func testRunsOnTransport(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "runs-on-transport.ts", `
		const step = createStep({
			id: "gate",
			inputSchema: z.object({ x: z.number() }),
			resumeSchema: z.object({ ok: z.boolean() }),
			outputSchema: z.object({ done: z.boolean() }),
			execute: async ({ inputData, resumeData, suspend }) => {
				if (inputData.x > 0 && !resumeData) return await suspend({});
				return { done: true };
			},
		});
		const wf = createWorkflow({
			id: "runs-on-transport",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ done: z.boolean() }),
		}).then(step).commit();
		kit.register("workflow", "runs-on-transport", wf);
	`)

	r1, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "runs-on-transport", InputData: json.RawMessage(`{"x":0}`)},
		10*time.Second,
	)
	require.Equal(t, "success", r1.Status)

	r2, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "runs-on-transport", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", r2.Status)

	allResp, allMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k,
		sdk.WorkflowRunsMsg{Name: "runs-on-transport"},
		5*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(allMsg.Payload))
	assert.GreaterOrEqual(t, allResp.Total, 2)

	suspendedResp, suspendedMsg := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k,
		sdk.WorkflowRunsMsg{Name: "runs-on-transport", Status: "suspended"},
		5*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(suspendedMsg.Payload))
	assert.Equal(t, 1, suspendedResp.Total)
}

func testCrashRecoverySuspendedOnTransport(t *testing.T, env *suite.TestEnv) {
	fixture := createSuspendedPersistedRun(t, env.Config.Transport, "crash-suspend-transport")
	k2 := reopenPersistedWorkflowKit(t, fixture)
	defer k2.Close()

	statusResp, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k2,
		sdk.WorkflowStatusMsg{Name: fixture.WorkflowName, RunID: fixture.RunID},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(statusMsg.Payload))
	assert.Equal(t, "suspended", statusResp.Status)
}

type workflowPersistenceFixture struct {
	Transport    brainkit.TransportConfig
	TmpDir       string
	StorePath    string
	MastraDBPath string
	WorkflowName string
	RunID        string
}

func workflowTransportForBackend(t *testing.T, backend string) brainkit.TransportConfig {
	t.Helper()
	switch backend {
	case "", "memory":
		return brainkit.Memory()
	default:
		tcfg := testutil.TransportConfigForBackend(t, backend)
		if backend != "embedded" {
			probe := testutil.MustCreateTransport(t, tcfg)
			testutil.WaitForBackendReady(t, probe)
			probe.Close()
		}
		return testutil.BrainkitTransport(tcfg)
	}
}

func newPersistentWorkflowKit(t *testing.T, transportCfg brainkit.TransportConfig, tmpDir, storePath, mastraDBPath string) *brainkit.Kit {
	t.Helper()
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := brainkit.New(brainkit.Config{
		Transport: transportCfg,
		Namespace: "test",
		CallerID:  "test-workflow-storage",
		FSRoot:    tmpDir,
		Store:     store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(mastraDBPath),
		},
		Modules: []brainkit.Module{workflow.New()},
	})
	require.NoError(t, err)
	return k
}

func createSuspendedPersistedRun(t *testing.T, backend, workflowName string) workflowPersistenceFixture {
	t.Helper()
	tmpDir := t.TempDir()
	fixture := workflowPersistenceFixture{
		Transport:    workflowTransportForBackend(t, backend),
		TmpDir:       tmpDir,
		StorePath:    filepath.Join(tmpDir, "kit.db"),
		MastraDBPath: filepath.Join(tmpDir, "mastra.db"),
		WorkflowName: workflowName,
	}

	k1 := newPersistentWorkflowKit(t, fixture.Transport, fixture.TmpDir, fixture.StorePath, fixture.MastraDBPath)
	wfDeploy(t, k1, workflowName+".ts", fmt.Sprintf(`
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
			id: %q,
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ result: z.string() }),
		}).then(step).commit();
		kit.register("workflow", %q, wf);
	`, workflowName, workflowName))

	startResp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k1,
		sdk.WorkflowStartMsg{Name: workflowName, InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)
	fixture.RunID = startResp.RunID
	require.NoError(t, k1.Close())
	return fixture
}

func createCompletedPersistedRun(t *testing.T, backend, workflowName string) workflowPersistenceFixture {
	t.Helper()
	tmpDir := t.TempDir()
	fixture := workflowPersistenceFixture{
		Transport:    workflowTransportForBackend(t, backend),
		TmpDir:       tmpDir,
		StorePath:    filepath.Join(tmpDir, "kit.db"),
		MastraDBPath: filepath.Join(tmpDir, "mastra.db"),
		WorkflowName: workflowName,
	}

	k1 := newPersistentWorkflowKit(t, fixture.Transport, fixture.TmpDir, fixture.StorePath, fixture.MastraDBPath)
	wfDeploy(t, k1, workflowName+".ts", fmt.Sprintf(`
		const step = createStep({
			id: "done",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
			execute: async ({ inputData }) => ({ y: inputData.x + 1 }),
		});
		const wf = createWorkflow({
			id: %q,
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
		}).then(step).commit();
		kit.register("workflow", %q, wf);
	`, workflowName, workflowName))

	startResp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k1,
		sdk.WorkflowStartMsg{Name: workflowName, InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "success", startResp.Status)
	fixture.RunID = startResp.RunID
	require.NoError(t, k1.Close())
	return fixture
}

func reopenPersistedWorkflowKit(t *testing.T, fixture workflowPersistenceFixture) *brainkit.Kit {
	t.Helper()
	return newPersistentWorkflowKit(t, fixture.Transport, fixture.TmpDir, fixture.StorePath, fixture.MastraDBPath)
}

func corruptWorkflowSnapshots(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND lower(name) LIKE '%workflow%snapshot%'`)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}
	require.NotEmpty(t, tables, "expected at least one workflow snapshot table")

	mutated := false
	for _, table := range tables {
		colRows, err := db.Query(fmt.Sprintf(`PRAGMA table_info("%s")`, table))
		require.NoError(t, err)

		var candidateCols []string
		for colRows.Next() {
			var cid, notNull, pk int
			var name, colType string
			var dflt sql.NullString
			require.NoError(t, colRows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk))
			switch name {
			case "snapshot", "state", "data", "payload", "value":
				candidateCols = append(candidateCols, name)
			}
		}
		colRows.Close()

		for _, col := range candidateCols {
			if _, err := db.Exec(fmt.Sprintf(`UPDATE "%s" SET "%s" = '{'`, table, col)); err == nil {
				mutated = true
				break
			}
		}
		if mutated {
			break
		}
		if _, err := db.Exec(fmt.Sprintf(`DELETE FROM "%s"`, table)); err == nil {
			mutated = true
			break
		}
	}
	require.True(t, mutated, "expected to corrupt or delete persisted workflow snapshot data")
}
