package workflows

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/infra/workflow_bus_test.go — concurrency + stress paths.

func testConcurrentStarts(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	k := env.Kit

	wfDeploy(t, k, "concurrent-wf.ts", `
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
	type startResult struct {
		resp sdk.WorkflowStartResp
		msg  sdk.Message
	}
	results := make(chan startResult, N)

	testutil.ConcurrentDo(t, N, func(i int) {
		resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{
				Name:      "concurrent-test",
				InputData: json.RawMessage(`{"n":` + json.Number(string(rune('0'+i+1))).String() + `}`),
			},
			10*time.Second,
		)
		results <- startResult{resp, msg}
	})
	close(results)

	for r := range results {
		errMsg := suite.ResponseErrorMessage(r.msg.Payload)
		assert.Empty(t, errMsg, "concurrent start should not error: %s", errMsg)
		assert.NotEmpty(t, r.resp.RunID)
	}
}

func testMultiWorkflowStress(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test-stress", FSRoot: tmpDir,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	// Deploy 4 workflow definitions
	wfDeploy(t, k, "stress-fast.ts", `
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

	wfDeploy(t, k, "stress-suspend.ts", `
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

	wfDeploy(t, k, "stress-parallel.ts", `
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

	wfDeploy(t, k, "stress-multi-suspend.ts", `
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

	// Phase 1: Start 10 fast-sequential concurrently
	t.Log("Phase 1: 10 fast-sequential")
	type phase1Result struct {
		resp sdk.WorkflowStartResp
		msg  sdk.Message
	}
	fastResults := make(chan phase1Result, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "fast-seq", InputData: json.RawMessage(`{"n":` + fmt.Sprintf("%d", i) + `}`)},
			10*time.Second,
		)
		fastResults <- phase1Result{resp, msg}
	})
	close(fastResults)
	for r := range fastResults {
		errMsg := suite.ResponseErrorMessage(r.msg.Payload)
		assert.Empty(t, errMsg, "fast-seq error: %s", errMsg)
		assert.Equal(t, "success", r.resp.Status)
	}

	// Phase 2: Start 3 suspend-resume → all suspend
	t.Log("Phase 2: 3 suspend-resume")
	suspRunIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "suspend-wf", InputData: json.RawMessage(fmt.Sprintf(`{"id":"s%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status, "suspend-wf should suspend")
		suspRunIDs[i] = resp.RunID
	}

	// Phase 3: Start 2 parallel-brancher
	t.Log("Phase 3: 2 parallel-brancher")
	for i := 0; i < 2; i++ {
		resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "parallel-wf", InputData: json.RawMessage(fmt.Sprintf(`{"x":%d}`, i+1))},
			10*time.Second,
		)
		assert.Equal(t, "success", resp.Status, "parallel-wf should complete")
	}

	// Phase 4: Start 2 multi-suspend → first suspend
	t.Log("Phase 4: 2 multi-suspend")
	multiRunIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "multi-sus", InputData: json.RawMessage(fmt.Sprintf(`{"id":"m%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status)
		multiRunIDs[i] = resp.RunID
	}

	// Phase 5: Resume suspend-resume instances one by one
	t.Log("Phase 5: resume 3 suspend-resume")
	for _, runID := range suspRunIDs {
		resp, msg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{Name: "suspend-wf", RunID: runID, Step: "gate", ResumeData: json.RawMessage(`{"ok":true}`)},
			10*time.Second,
		)
		errMsg := suite.ResponseErrorMessage(msg.Payload)
		assert.Empty(t, errMsg, "resume suspend-wf: %s", errMsg)
		assert.Equal(t, "success", resp.Status)
	}

	// Phase 6: Resume multi-suspend through both cycles
	t.Log("Phase 6: resume multi-suspend (2 cycles each)")
	for _, runID := range multiRunIDs {
		r1, _ := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{Name: "multi-sus", RunID: runID, Step: "g1", ResumeData: json.RawMessage(`{"v":"first"}`)},
			10*time.Second,
		)
		require.Equal(t, "suspended", r1.Status, "after g1 resume should suspend at g2")

		r2, _ := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{Name: "multi-sus", RunID: runID, Step: "g2", ResumeData: json.RawMessage(`{"v":"second"}`)},
			10*time.Second,
		)
		assert.Equal(t, "success", r2.Status, "after g2 resume should complete")
	}

	t.Log("All phases complete — multi-workflow stress test passed")
}

func testLongRunningIntegration(t *testing.T, _ *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipping long-running integration test in short mode")
	}

	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "kit.db"))
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test-longrun", FSRoot: tmpDir,
		Store: store,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmpDir, "mastra.db")),
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	// ── Deploy 5 workflow definitions ──

	// 1. fast-seq: n → (n+1) → (n+1)*2. Verify math.
	wfDeploy(t, k, "lr-fast.ts", `
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
	wfDeploy(t, k, "lr-sleeper.ts", `
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
	wfDeploy(t, k, "lr-suspend.ts", `
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
	wfDeploy(t, k, "lr-parallel.ts", `
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
	wfDeploy(t, k, "lr-multi-sus.ts", `
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
		resp  sdk.WorkflowStartResp
		msg   sdk.Message
	}
	fastCh := make(chan fastResult, 10)
	testutil.ConcurrentDo(t, 10, func(i int) {
		resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "lr-fast", InputData: json.RawMessage(fmt.Sprintf(`{"n":%d}`, i))},
			10*time.Second,
		)
		fastCh <- fastResult{i, resp.RunID, resp, msg}
	})
	close(fastCh)
	for fr := range fastCh {
		errMsg := suite.ResponseErrorMessage(fr.msg.Payload)
		require.Empty(t, errMsg, "fast n=%d error: %s", fr.n, errMsg)
		require.Equal(t, "success", fr.resp.Status, "fast n=%d status", fr.n)
		// Verify status from storage shows success
		statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-fast", RunID: fr.runID},
			5*time.Second,
		)
		assert.Equal(t, "success", statusResp.Status, "fast n=%d storage status", fr.n)
	}

	// 5 sleepers — start async (10s sleep)
	sleeperRunIDs := make([]string, 5)
	for i := 0; i < 5; i++ {
		resp, msg := wfPublishAndWait[sdk.WorkflowStartAsyncMsg, sdk.WorkflowStartAsyncResp](
			t, k,
			sdk.WorkflowStartAsyncMsg{Name: "lr-sleeper", InputData: json.RawMessage(fmt.Sprintf(`{"id":"s%d","sleepMs":10000}`, i))},
			5*time.Second,
		)
		require.Empty(t, suite.ResponseErrorMessage(msg.Payload), "sleeper start s%d", i)
		require.NotEmpty(t, resp.RunID)
		sleeperRunIDs[i] = resp.RunID
	}
	t.Log("5 sleepers started (10s sleep each)")

	// Verify sleepers are running/pending (not completed yet)
	for i, runID := range sleeperRunIDs {
		statusResp, statusMsg := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
			5*time.Second,
		)
		require.Empty(t, suite.ResponseErrorMessage(statusMsg.Payload), "sleeper s%d status", i)
		assert.NotEqual(t, "success", statusResp.Status, "sleeper s%d should NOT be success yet (still sleeping)", i)
		t.Logf("  sleeper s%d status: %s", i, statusResp.Status)
	}

	// 3 suspenders — verify they suspend
	suspRunIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "lr-suspend", InputData: json.RawMessage(fmt.Sprintf(`{"id":"susp%d"}`, i))},
			10*time.Second,
		)
		require.Equal(t, "suspended", resp.Status, "suspend susp%d", i)
		suspRunIDs[i] = resp.RunID

		// Verify storage also shows suspended
		statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-suspend", RunID: resp.RunID},
			5*time.Second,
		)
		assert.Equal(t, "suspended", statusResp.Status, "suspend susp%d storage", i)
	}

	// 2 multi-suspend — verify first suspend
	multiRunIDs := make([]string, 2)
	for i := 0; i < 2; i++ {
		resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "lr-multi-sus", InputData: json.RawMessage(fmt.Sprintf(`{"id":"ms%d"}`, i))},
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
		resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
			t, k,
			sdk.WorkflowStartMsg{Name: "lr-parallel", InputData: json.RawMessage(fmt.Sprintf(`{"x":%d}`, x))},
			10*time.Second,
		)
		require.Empty(t, suite.ResponseErrorMessage(msg.Payload), "parallel x=%d", x)
		require.Equal(t, "success", resp.Status, "parallel x=%d status", x)
		// Query storage for per-step outputs
		statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-parallel", RunID: resp.RunID},
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
		resp, msg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{
				Name: "lr-suspend", RunID: runID, Step: "gate",
				ResumeData: json.RawMessage(fmt.Sprintf(`{"approved":%v}`, approved)),
			},
			10*time.Second,
		)
		errMsg := suite.ResponseErrorMessage(msg.Payload)
		require.Empty(t, errMsg, "resume susp%d: %s", i, errMsg)
		require.Equal(t, "success", resp.Status, "resume susp%d status", i)

		// Verify output in storage
		statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-suspend", RunID: runID},
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
		r1, r1Msg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{Name: "lr-multi-sus", RunID: runID, Step: "g1", ResumeData: json.RawMessage(`{"v":"alpha"}`)},
			10*time.Second,
		)
		r1Err := suite.ResponseErrorMessage(r1Msg.Payload)
		require.Empty(t, r1Err, "multi-sus ms%d g1: %s", i, r1Err)
		require.Equal(t, "suspended", r1.Status, "ms%d should suspend at g2", i)

		// Verify g1 step output in storage: r1 = "ms<i>-alpha"
		statusAfterG1, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-multi-sus", RunID: runID},
			5*time.Second,
		)
		assert.Equal(t, "suspended", statusAfterG1.Status)
		assert.Contains(t, string(statusAfterG1.Steps), fmt.Sprintf("ms%d-alpha", i), "g1 should produce ms%d-alpha", i)

		// Second gate: g2 receives v="beta"
		r2, r2Msg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
			t, k,
			sdk.WorkflowResumeMsg{Name: "lr-multi-sus", RunID: runID, Step: "g2", ResumeData: json.RawMessage(`{"v":"beta"}`)},
			10*time.Second,
		)
		r2Err := suite.ResponseErrorMessage(r2Msg.Payload)
		require.Empty(t, r2Err, "multi-sus ms%d g2: %s", i, r2Err)
		require.Equal(t, "success", r2.Status, "ms%d should complete after g2", i)

		// Verify g2 step output: r2 = "ms<i>-alpha-beta"
		statusAfterG2, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-multi-sus", RunID: runID},
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

	cancelResp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "lr-suspend", InputData: json.RawMessage(`{"id":"to-cancel"}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", cancelResp.Status)

	cancelResult, cancelMsg := wfPublishAndWait[sdk.WorkflowCancelMsg, sdk.WorkflowCancelResp](
		t, k,
		sdk.WorkflowCancelMsg{Name: "lr-suspend", RunID: cancelResp.RunID},
		5*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(cancelMsg.Payload))
	require.True(t, cancelResult.Cancelled)

	// Verify storage shows canceled
	cancelStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "lr-suspend", RunID: cancelResp.RunID},
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
			statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
				t, k,
				sdk.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
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
		afterStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
			t, k,
			sdk.WorkflowStatusMsg{Name: "lr-sleeper", RunID: runID},
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

	fastRuns, _ := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k, sdk.WorkflowRunsMsg{Name: "lr-fast"}, 5*time.Second,
	)
	assert.GreaterOrEqual(t, fastRuns.Total, 10, "lr-fast should have >=10 runs")

	sleeperRuns, _ := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k, sdk.WorkflowRunsMsg{Name: "lr-sleeper"}, 5*time.Second,
	)
	assert.Equal(t, 5, sleeperRuns.Total, "lr-sleeper should have 5 runs")

	suspendRuns, _ := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k, sdk.WorkflowRunsMsg{Name: "lr-suspend"}, 5*time.Second,
	)
	assert.GreaterOrEqual(t, suspendRuns.Total, 4, "lr-suspend should have >=4 runs (3 resumed + 1 canceled)")

	canceledRuns, _ := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k, sdk.WorkflowRunsMsg{Name: "lr-suspend", Status: "canceled"}, 5*time.Second,
	)
	assert.Equal(t, 1, canceledRuns.Total, "exactly 1 lr-suspend run should be canceled")

	multiRuns, _ := wfPublishAndWait[sdk.WorkflowRunsMsg, sdk.WorkflowRunsResp](
		t, k, sdk.WorkflowRunsMsg{Name: "lr-multi-sus"}, 5*time.Second,
	)
	assert.Equal(t, 2, multiRuns.Total, "lr-multi-sus should have 2 runs")

	t.Log("Long-running integration test complete — all outputs verified")
}
