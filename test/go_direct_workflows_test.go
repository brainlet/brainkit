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

const testWorkflowCode = `
	const wf = createWorkflow({
		id: "test-workflow",
		inputSchema: z.object({ value: z.string() }),
		outputSchema: z.object({ result: z.string() }),
	});

	const step1 = createStep({
		id: "step1",
		inputSchema: z.object({ value: z.string() }),
		outputSchema: z.object({ processed: z.string() }),
		execute: async ({ inputData }) => {
			return { processed: inputData.value + "-step1" };
		},
	});

	const step2 = createStep({
		id: "step2",
		inputSchema: z.object({ processed: z.string() }),
		outputSchema: z.object({ result: z.string() }),
		execute: async ({ inputData }) => {
			return { result: inputData.processed + "-step2" };
		},
	});

	wf.then(step1).then(step2).commit();
`

func TestGoDirect_Workflows(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			tk := newTestKernelWithStorage(t)
			rt := sdk.Runtime(tk)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Deploy the test workflow
			_, err := sdk.PublishAwait[messages.KitDeployMsg, messages.KitDeployResp](rt, ctx, messages.KitDeployMsg{
				Source: "test-workflow.ts",
				Code:   testWorkflowCode,
			})
			require.NoError(t, err, "workflow deploy must succeed")

			t.Run("Run", func(t *testing.T) {
				resp, err := sdk.PublishAwait[messages.WorkflowRunMsg, messages.WorkflowRunResp](rt, ctx, messages.WorkflowRunMsg{
					Name:  "test-workflow",
					Input: map[string]any{"value": "hello"},
				})
				require.NoError(t, err)
				assert.NotNil(t, resp.Result, "should return workflow result")

				// Parse the result
				var result map[string]any
				json.Unmarshal(resp.Result, &result)
				t.Logf("Workflow result: %v", result)
			})

			// Cleanup
			sdk.PublishAwait[messages.KitTeardownMsg, messages.KitTeardownResp](rt, ctx, messages.KitTeardownMsg{Source: "test-workflow.ts"})
		})
	}
}
