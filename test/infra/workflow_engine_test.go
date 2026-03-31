package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowEngine_CompileRunAndJournal(t *testing.T) {
	storePath := t.TempDir() + "/wf-test.db"
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	traceStore := tracing.NewMemoryTraceStore(100)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Store:      store,
		TraceStore: traceStore,
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Step 1: Compile AS source to WASM via wasm.compile
	// Uses @external decorators for host functions (not imports — these are wazero host modules)
	workflowAS := `
@external("brainkit", "step")
declare function step(name: string): void;

@external("brainkit", "complete")
declare function complete(result: string): void;

export function run(inputPtr: string): void {
    step("greet");
    step("finish");
    complete("done");
}
`
	compileResult, err := evalBusCommand(k, ctx, "wasm.compile",
		map[string]any{"source": workflowAS, "options": map[string]any{"name": "test-wf"}})
	require.NoError(t, err)
	assert.NotZero(t, compileResult["size"], "compiled binary should have size")

	// Step 2: Deploy as automation (compile → register workflow → ready to run)
	manifest := map[string]any{
		"name":    "test-wf",
		"version": "1.0.0",
		"type":    "automation",
		"workflow": map[string]any{
			"entry":   "workflow.ts",
			"timeout": 30,
		},
	}
	manifestJSON, _ := json.Marshal(manifest)
	deployResult, err := evalBusCommand(k, ctx, "automation.deploy",
		map[string]any{"manifest": json.RawMessage(manifestJSON), "workflowSource": workflowAS})
	require.NoError(t, err)
	assert.Equal(t, true, deployResult["deployed"])
	assert.NotEmpty(t, deployResult["workflowId"])

	// Step 3: Run the workflow
	runResult, err := evalBusCommand(k, ctx, "workflow.run",
		map[string]any{"workflowId": "test-wf", "input": "hello"})
	require.NoError(t, err)
	assert.Equal(t, "running", runResult["status"])
	runID, _ := runResult["runId"].(string)
	require.NotEmpty(t, runID)

	// Step 4: Wait for completion (poll status)
	var finalStatus string
	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		statusResult, err := evalBusCommand(k, ctx, "workflow.status",
			map[string]any{"runId": runID})
		if err != nil {
			continue
		}
		s, _ := statusResult["status"].(string)
		if s == "completed" || s == "failed" {
			finalStatus = s
			break
		}
	}
	assert.Equal(t, "completed", finalStatus, "workflow should complete")

	// Step 5: Verify journal has entries
	histResult, err := evalBusCommand(k, ctx, "workflow.history",
		map[string]any{"runId": runID})
	require.NoError(t, err)
	entriesRaw, ok := histResult["entries"]
	require.True(t, ok, "history should have entries")

	// entries is a JSON-encoded array
	var entries []map[string]any
	switch v := entriesRaw.(type) {
	case string:
		json.Unmarshal([]byte(v), &entries)
	case json.RawMessage:
		json.Unmarshal(v, &entries)
	default:
		// Try marshaling and re-parsing
		b, _ := json.Marshal(v)
		json.Unmarshal(b, &entries)
	}
	assert.GreaterOrEqual(t, len(entries), 2, "should have at least 2 journal entries (greet + finish)")

	// Step 6: Verify automation list shows the deployed automation
	listResult, err := evalBusCommand(k, ctx, "automation.list", map[string]any{})
	require.NoError(t, err)
	automations, _ := listResult["automations"].([]any)
	assert.Len(t, automations, 1)

	// Step 7: Teardown automation
	teardownResult, err := evalBusCommand(k, ctx, "automation.teardown",
		map[string]any{"name": "test-wf"})
	require.NoError(t, err)
	assert.Equal(t, true, teardownResult["removed"])

	// Step 8: Verify traces were recorded
	spans, _ := traceStore.ListTraces(tracing.TraceQuery{})
	assert.Greater(t, len(spans), 0, "should have trace spans from workflow execution")
}

// evalBusCommand calls a Kernel bus command via EvalTS (JS bridge path).
// Uses JSON.stringify on the Go side and JSON.parse on the JS side to avoid
// template literal escaping issues with newlines in payloads.
func evalBusCommand(k *brainkit.Kernel, ctx context.Context, topic string, payload map[string]any) (map[string]any, error) {
	payloadJSON, _ := json.Marshal(payload)
	// Double-encode: Go JSON → JS string literal → JSON.parse inside JS
	quotedPayload, _ := json.Marshal(string(payloadJSON))
	script := fmt.Sprintf(`return __go_brainkit_request(%q, %s);`, topic, string(quotedPayload))
	raw, err := k.EvalTS(ctx, "__test_cmd.ts", script)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, err
	}
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return result, fmt.Errorf("%s", errMsg)
	}
	return result, nil
}
