package workflows

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/infra/workflow_bus_test.go — developer scenario tests.
// These use env.Kit (shared Full kernel with tools+storage).

func testToolCallInsideStep(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "tool-in-step.ts", `
		const fetchStep = createStep({
			id: "fetch-data",
			inputSchema: z.object({ query: z.string() }),
			outputSchema: z.object({ echoed: z.string(), processed: z.string() }),
			execute: async ({ inputData }) => {
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

	resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "tool-in-step", InputData: json.RawMessage(`{"query":"hello world"}`)},
		15*time.Second,
	)
	errMsg := suite.ResponseErrorMessage(msg.Payload)
	require.Empty(t, errMsg, "tool-in-step error: %s", errMsg)
	require.Equal(t, "success", resp.Status)

	statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "tool-in-step", RunID: resp.RunID},
		5*time.Second,
	)
	steps := string(statusResp.Steps)
	assert.Contains(t, steps, "hello world", "echoed should contain original query")
	assert.Contains(t, steps, "PROCESSED-HELLO WORLD", "transform should uppercase")
}

func testToolFailureInsideStep(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "tool-failure-step.ts", `
		const fail = createStep({
			id: "fail-tool",
			inputSchema: z.object({ msg: z.string() }),
			outputSchema: z.object({ echoed: z.string() }),
			execute: async ({ inputData }) => {
				const result = await tools.call("ghost-tool-workflow", { message: inputData.msg });
				return { echoed: result.echoed };
			},
		});
		const wf = createWorkflow({
			id: "tool-failure-step",
			inputSchema: z.object({ msg: z.string() }),
			outputSchema: z.object({ echoed: z.string() }),
		}).then(fail).commit();
		kit.register("workflow", "tool-failure-step", wf);
	`)

	resp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "tool-failure-step", InputData: json.RawMessage(`{"msg":"boom"}`)},
		15*time.Second,
	)
	require.NotEmpty(t, resp.RunID)
	assert.Equal(t, "failed", resp.Status)
}

func testBusEmitFromStep(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "bus-emit-step.ts", `
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	eventCh := make(chan json.RawMessage, 1)
	unsub, err := k.SubscribeRaw(ctx, "order.processing", func(msg sdk.Message) {
		select {
		case eventCh <- msg.Payload:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "bus-emit-wf", InputData: json.RawMessage(`{"orderId":"ORD-123"}`)},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(msg.Payload))
	require.Equal(t, "success", resp.Status)

	select {
	case payload := <-eventCh:
		var event struct {
			OrderID string `json:"orderId"`
			Stage   string `json:"stage"`
		}
		require.NoError(t, json.Unmarshal(payload, &event))
		assert.Equal(t, "ORD-123", event.OrderID)
		assert.Equal(t, "started", event.Stage)
	case <-ctx.Done():
		t.Fatal("timeout waiting for order.processing bus event")
	}
}

func testConditionalBranch(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "branch-wf.ts", `
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
	premResp, premMsg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "branch-wf", InputData: json.RawMessage(`{"amount":250}`)},
		10*time.Second,
	)
	premErr := suite.ResponseErrorMessage(premMsg.Payload)
	require.Empty(t, premErr, "premium branch: %s", premErr)
	require.Equal(t, "success", premResp.Status)
	assert.NotEmpty(t, premResp.Result, "branch result must be non-empty")
	assert.Contains(t, string(premResp.Result), "premium-250", "branch result should contain step output")

	premStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "branch-wf", RunID: premResp.RunID},
		5*time.Second,
	)
	assert.Contains(t, string(premStatus.Steps), "premium-250", "should take premium branch")

	// Test standard path (amount < 100)
	stdResp, stdMsg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "branch-wf", InputData: json.RawMessage(`{"amount":50}`)},
		10*time.Second,
	)
	stdErr := suite.ResponseErrorMessage(stdMsg.Payload)
	require.Empty(t, stdErr, "standard branch: %s", stdErr)
	require.Equal(t, "success", stdResp.Status)

	stdStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "branch-wf", RunID: stdResp.RunID},
		5*time.Second,
	)
	assert.Contains(t, string(stdStatus.Steps), "standard-50", "should take standard branch")
}

func testBranchFallbackPath(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "branch-fallback.ts", `
		const classify = createStep({
			id: "classify",
			inputSchema: z.object({ tier: z.string() }),
			outputSchema: z.object({ tier: z.string() }),
			execute: async ({ inputData }) => ({ tier: inputData.tier }),
		});
		const fallback = createStep({
			id: "fallback",
			inputSchema: z.object({ tier: z.string() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData }) => ({ result: "fallback-" + inputData.tier }),
		});
		const premium = createStep({
			id: "premium",
			inputSchema: z.object({ tier: z.string() }),
			outputSchema: z.object({ result: z.string() }),
			execute: async ({ inputData }) => ({ result: "premium-" + inputData.tier }),
		});
		const wf = createWorkflow({
			id: "branch-fallback",
			inputSchema: z.object({ tier: z.string() }),
			outputSchema: z.object({ result: z.string() }),
		}).then(classify).branch([
			[async ({ inputData }) => inputData.tier === "premium", premium],
			[async () => true, fallback],
		]).commit();
		kit.register("workflow", "branch-fallback", wf);
	`)

	resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "branch-fallback", InputData: json.RawMessage(`{"tier":"unknown"}`)},
		10*time.Second,
	)
	require.Empty(t, suite.ResponseErrorMessage(msg.Payload))
	require.Equal(t, "success", resp.Status)

	statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "branch-fallback", RunID: resp.RunID},
		5*time.Second,
	)
	assert.Contains(t, string(statusResp.Steps), "fallback-unknown", "should take explicit fallback branch")
}

func testStepState(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "state-wf.ts", `
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

	resp, msg := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{Name: "state-wf", InputData: json.RawMessage(`{"name":"test-user"}`)},
		10*time.Second,
	)
	errMsg := suite.ResponseErrorMessage(msg.Payload)
	require.Empty(t, errMsg, "state workflow: %s", errMsg)
	require.Equal(t, "success", resp.Status)

	statusResp, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "state-wf", RunID: resp.RunID},
		5*time.Second,
	)
	steps := string(statusResp.Steps)
	assert.Contains(t, steps, `"counterValue":11`, "counter should be 11 (1+10)")
	assert.Contains(t, steps, `"logLength":2`, "log should have 2 entries")
}

func testSuspendWithContextData(t *testing.T, env *suite.TestEnv) {
	k := env.Kit
	wfDeploy(t, k, "suspend-context.ts", `
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

	startResp, _ := wfPublishAndWait[sdk.WorkflowStartMsg, sdk.WorkflowStartResp](
		t, k,
		sdk.WorkflowStartMsg{
			Name:      "doc-review",
			InputData: json.RawMessage(`{"documentId":"DOC-456","content":"This is a very important document that needs careful review by a senior team member."}`),
		},
		10*time.Second,
	)
	require.Equal(t, "suspended", startResp.Status)

	suspStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "doc-review", RunID: startResp.RunID},
		5*time.Second,
	)
	assert.Equal(t, "suspended", suspStatus.Status)
	assert.Contains(t, string(suspStatus.Steps), "Document needs review", "suspend payload should be in storage")
	assert.Contains(t, string(suspStatus.Steps), "DOC-456", "documentId should be in suspend payload")

	resumeResp, resumeMsg := wfPublishAndWait[sdk.WorkflowResumeMsg, sdk.WorkflowResumeResp](
		t, k,
		sdk.WorkflowResumeMsg{
			Name:       "doc-review",
			RunID:      startResp.RunID,
			Step:       "review",
			ResumeData: json.RawMessage(`{"decision":"approved","reviewer":"alice@corp.com"}`),
		},
		10*time.Second,
	)
	resumeErr := suite.ResponseErrorMessage(resumeMsg.Payload)
	require.Empty(t, resumeErr, "resume doc-review: %s", resumeErr)
	require.Equal(t, "success", resumeResp.Status)

	finalStatus, _ := wfPublishAndWait[sdk.WorkflowStatusMsg, sdk.WorkflowStatusResp](
		t, k,
		sdk.WorkflowStatusMsg{Name: "doc-review", RunID: startResp.RunID},
		5*time.Second,
	)
	steps := string(finalStatus.Steps)
	assert.Contains(t, steps, `"status":"approved"`, "should be approved")
	assert.Contains(t, steps, `"reviewedBy":"alice@corp.com"`, "reviewer should be alice")
	assert.Contains(t, steps, `"documentId":"DOC-456"`, "documentId should flow through")
}
