package workflows

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/infra/workflow_bus_test.go — happy path + error paths.
// These use env.Kit (shared Full kernel with tools+storage).

func testStartSequential(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "seq-wf.ts", `
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

	resp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "seq-test", InputData: json.RawMessage(`{"text":"hello"}`)},
		10*time.Second,
	)
	require.Empty(t, resp.Error, "should not error: %s", resp.Error)
	require.NotEmpty(t, resp.RunID)
	assert.Contains(t, resp.Status, "success")
}

func testStartParallel(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "par-wf.ts", `
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

	resp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "par-test", InputData: json.RawMessage(`{"x":5}`)},
		10*time.Second,
	)
	require.Empty(t, resp.Error)
	assert.Contains(t, resp.Status, "success")
}

func testList(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "list-wf.ts", `
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

	resp := wfPublishAndWait[sdk.WorkflowListMsg, sdk.WorkflowListResp](
		t, k, sdk.WorkflowListMsg{}, 5*time.Second,
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

func testSuspendResume(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "suspend-wf.ts", `
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

	startResp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "approval-flow", InputData: json.RawMessage(`{"item":"widget"}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)
	require.NotEmpty(t, startResp.RunID)

	resumeResp := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
		t, k,
		sdk.WorkflowResumeMsg{
			Name: "approval-flow", RunID: startResp.RunID,
			Step: "approval", ResumeData: json.RawMessage(`{"approved":true}`),
		},
		10*time.Second,
	)
	require.Empty(t, resumeResp.Error, "resume should not error: %s", resumeResp.Error)
}

func testCancel(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "cancel-wf.ts", `
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

	startResp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "cancel-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)

	cancelResp := wfPublishAndWait[sdk.WorkflowCancelMsg, sdk.WorkflowCancelResp](
		t, k,
		sdk.WorkflowCancelMsg{Name: "cancel-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, cancelResp.Error)
	require.True(t, cancelResp.Cancelled)

	statusResp := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "cancel-test", RunID: startResp.RunID},
		5*time.Second,
	)
	require.Empty(t, statusResp.Error, "status after cancel should succeed: %s", statusResp.Error)
	assert.Equal(t, "canceled", statusResp.Status)
}

func testWithToolCall(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "tool-wf.ts", `
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

	resp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "tool-wf", InputData: json.RawMessage(`{"msg":"from-workflow"}`)},
		10*time.Second,
	)
	require.Empty(t, resp.Error, "tool workflow should not error: %s", resp.Error)
	assert.Contains(t, resp.Status, "success")
}

func testNotFound(t *testing.T, env *suite.TestEnv) {
	resp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, env.Kit,
		sdk.WorkflowStartMsg{Name: "ghost-workflow"},
		5*time.Second,
	)
	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func testResumeNonexistentRun(t *testing.T, env *suite.TestEnv) {
	resp := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
		t, env.Kit,
		sdk.WorkflowResumeMsg{Name: "any", RunID: "fake-run-id", ResumeData: json.RawMessage(`{}`)},
		5*time.Second,
	)
	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func testStatusNonexistentRun(t *testing.T, env *suite.TestEnv) {
	resp := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, env.Kit,
		sdk.WorkflowStatusMsg{Name: "any", RunID: "fake-run-id"},
		5*time.Second,
	)
	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func testCancelNonexistentRun(t *testing.T, env *suite.TestEnv) {
	resp := wfPublishAndWait[sdk.WorkflowCancelMsg, sdk.WorkflowCancelResp](
		t, env.Kit,
		sdk.WorkflowCancelMsg{Name: "any", RunID: "fake-run-id"},
		5*time.Second,
	)
	require.NotEmpty(t, resp.Error)
	assert.Contains(t, resp.Error, "not found")
}

func testStepWithError(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "error-wf.ts", `
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

	resp := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "error-test", InputData: json.RawMessage(`{"x":1}`)},
		10*time.Second,
	)
	require.NotEmpty(t, resp.RunID)
	assert.Equal(t, "failed", resp.Status)
}
