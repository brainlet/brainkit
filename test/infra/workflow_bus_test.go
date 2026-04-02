package infra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/require"
)

func TestWorkflowBusStart(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	k := tk.Kernel

	// Deploy a workflow
	_, err := k.Deploy(context.Background(), "wf.ts", `
		const step1 = createStep({
			id: "greet",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ greeting: z.string() }),
			execute: async ({ inputData }) => {
				return { greeting: "Hello " + inputData.name };
			},
		});
		const wf = createWorkflow({
			id: "greet-workflow",
			inputSchema: z.object({ name: z.string() }),
			outputSchema: z.object({ greeting: z.string() }),
		}).then(step1).commit();
		kit.register("workflow", "greet-workflow", wf);
	`)
	require.NoError(t, err)

	// Start via bus
	result, err := sdk.Publish(k, context.Background(), messages.WorkflowStartMsg{
		Name:      "greet-workflow",
		InputData: json.RawMessage(`{"name":"World"}`),
	})
	require.NoError(t, err)

	// Wait for response
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var resp messages.WorkflowStartResp
	unsub, err := sdk.SubscribeTo[messages.WorkflowStartResp](k, ctx, result.ReplyTo, func(r messages.WorkflowStartResp, msg messages.Message) {
		resp = r
		cancel()
	})
	require.NoError(t, err)
	defer unsub()
	<-ctx.Done()

	require.Empty(t, resp.Error, "workflow should not return error")
	require.NotEmpty(t, resp.RunID)
}

func TestWorkflowBusNotFound(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	k := tk.Kernel

	result, err := sdk.Publish(k, context.Background(), messages.WorkflowStartMsg{
		Name: "nonexistent",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var resp messages.WorkflowStartResp
	unsub, err := sdk.SubscribeTo[messages.WorkflowStartResp](k, ctx, result.ReplyTo, func(r messages.WorkflowStartResp, msg messages.Message) {
		resp = r
		cancel()
	})
	require.NoError(t, err)
	defer unsub()
	<-ctx.Done()

	require.NotEmpty(t, resp.Error)
	require.Contains(t, resp.Error, "not found")
}

func TestWorkflowBusList(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	k := tk.Kernel

	// Deploy a workflow
	_, err := k.Deploy(context.Background(), "list-wf.ts", `
		const wf = createWorkflow({
			id: "list-test",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
		}).then(createStep({
			id: "double",
			inputSchema: z.object({ x: z.number() }),
			outputSchema: z.object({ y: z.number() }),
			execute: async ({ inputData }) => ({ y: inputData.x * 2 }),
		})).commit();
		kit.register("workflow", "list-test", wf);
	`)
	require.NoError(t, err)

	result, err := sdk.Publish(k, context.Background(), messages.WorkflowListMsg{})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var resp messages.WorkflowListResp
	unsub, err := sdk.SubscribeTo[messages.WorkflowListResp](k, ctx, result.ReplyTo, func(r messages.WorkflowListResp, msg messages.Message) {
		resp = r
		cancel()
	})
	require.NoError(t, err)
	defer unsub()
	<-ctx.Done()

	require.Empty(t, resp.Error)
	found := false
	for _, wf := range resp.Workflows {
		if wf.Name == "list-test" {
			found = true
		}
	}
	require.True(t, found, "list-test workflow should appear in list")
}

func TestWorkflowBusSuspendResume(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	k := tk.Kernel

	_, err := k.Deploy(context.Background(), "suspend-wf.ts", `
		const approvalStep = createStep({
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
		}).then(approvalStep).commit();
		kit.register("workflow", "approval-flow", wf);
	`)
	require.NoError(t, err)

	// Start — should suspend
	startResult, err := sdk.Publish(k, context.Background(), messages.WorkflowStartMsg{
		Name:      "approval-flow",
		InputData: json.RawMessage(`{"item":"widget"}`),
	})
	require.NoError(t, err)

	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel1()
	var startResp messages.WorkflowStartResp
	unsub1, _ := sdk.SubscribeTo[messages.WorkflowStartResp](k, ctx1, startResult.ReplyTo, func(r messages.WorkflowStartResp, msg messages.Message) {
		startResp = r
		cancel1()
	})
	defer unsub1()
	<-ctx1.Done()

	require.Equal(t, "suspended", startResp.Status, "workflow should suspend")
	require.NotEmpty(t, startResp.RunID)

	// Resume with approval
	resumeResult, err := sdk.Publish(k, context.Background(), messages.WorkflowResumeMsg{
		Name:       "approval-flow",
		RunID:      startResp.RunID,
		Step:       "approval",
		ResumeData: json.RawMessage(`{"approved":true}`),
	})
	require.NoError(t, err)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	var resumeResp messages.WorkflowResumeResp
	unsub2, _ := sdk.SubscribeTo[messages.WorkflowResumeResp](k, ctx2, resumeResult.ReplyTo, func(r messages.WorkflowResumeResp, msg messages.Message) {
		resumeResp = r
		cancel2()
	})
	defer unsub2()
	<-ctx2.Done()

	require.Empty(t, resumeResp.Error, "resume should not return error: %s", resumeResp.Error)
}
