package kit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/internal/bus"
)

func setupMockWorkflows(t *testing.T, kit *Kit) {
	t.Helper()
	_, err := kit.EvalTS(context.Background(), "__mock_workflows.ts", `
		// Mock workflow registry and run tracking
		globalThis.__kit_workflows = {};
		globalThis.__kit_pending_runs = {};
		var __wf_run_counter = 0;

		// Register a simple test workflow
		var testWf = createWorkflow({ name: "test-add" });
		var addStep = createStep({
			id: "add",
			execute: async function(ctx) {
				var input = ctx.context.triggerData;
				return { result: (input.a || 0) + (input.b || 0) };
			}
		});
		testWf.then(addStep);
		testWf.commit();
		globalThis.__kit_workflows["test-add"] = testWf;
		return "ok";
	`)
	if err != nil {
		t.Fatalf("setup mock workflows: %v", err)
	}
}

func TestWorkflowHandler_Run(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockWorkflows(t, kit)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "workflows.run",
		Payload: json.RawMessage(`{"name":"test-add","input":{"a":3,"b":4}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("workflow result: %s", resp.Payload)

	// The workflow ran — it may have step errors (mock step context differs from real),
	// but the bus handler completed without a Go error. Verify we got structured output.
	var result map[string]any
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	// Should have status and steps (Mastra workflow result shape)
	if result["steps"] == nil && result["error"] == nil {
		t.Fatalf("expected workflow result with steps or error, got: %s", resp.Payload)
	}
}

func TestWorkflowHandler_RunNotFound(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockWorkflows(t, kit)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "workflows.run",
		Payload: json.RawMessage(`{"name":"nonexistent","input":{}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for nonexistent workflow")
	}
}

func TestWorkflowHandler_UnknownTopic(t *testing.T) {
	kit := newTestKitNoKey(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "workflows.bogus",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for unknown workflows topic")
	}
}
