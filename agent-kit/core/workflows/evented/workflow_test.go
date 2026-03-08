// Ported from: packages/core/src/workflows/evented/workflow.test.ts
package evented

import (
	"testing"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// CreateStep tests
// ---------------------------------------------------------------------------

func TestCreateStep(t *testing.T) {
	t.Run("should create a step with all fields", func(t *testing.T) {
		retries := 3
		step := CreateStep(wf.StepParams{
			ID:          "my-step",
			Description: "A test step",
			Retries:     &retries,
			Metadata:    wf.StepMetadata{"priority": "high"},
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return "executed", nil
			},
		})

		if step.ID != "my-step" {
			t.Errorf("step.ID = %q, want %q", step.ID, "my-step")
		}
		if step.Description != "A test step" {
			t.Errorf("step.Description = %q, want %q", step.Description, "A test step")
		}
		if step.Retries == nil || *step.Retries != 3 {
			t.Errorf("step.Retries = %v, want 3", step.Retries)
		}
		if step.Metadata["priority"] != "high" {
			t.Errorf("step.Metadata[priority] = %v, want 'high'", step.Metadata["priority"])
		}
		if step.Execute == nil {
			t.Error("step.Execute is nil")
		}
	})

	t.Run("should create a step with minimal fields", func(t *testing.T) {
		step := CreateStep(wf.StepParams{
			ID: "minimal-step",
		})
		if step.ID != "minimal-step" {
			t.Errorf("step.ID = %q, want %q", step.ID, "minimal-step")
		}
		if step.Execute != nil {
			t.Error("step.Execute should be nil for minimal step")
		}
	})
}

// ---------------------------------------------------------------------------
// CloneStep tests
// ---------------------------------------------------------------------------

func TestCloneStep(t *testing.T) {
	t.Run("should clone a step with a new ID", func(t *testing.T) {
		retries := 2
		original := &wf.Step{
			ID:          "original",
			Description: "Original step",
			Retries:     &retries,
			Component:   "WORKFLOW",
			Metadata:    wf.StepMetadata{"key": "value"},
			Execute: func(params *wf.ExecuteFunctionParams) (any, error) {
				return "original", nil
			},
		}

		cloned := CloneStep(original, "cloned")

		if cloned.ID != "cloned" {
			t.Errorf("cloned.ID = %q, want %q", cloned.ID, "cloned")
		}
		if cloned.Description != "Original step" {
			t.Errorf("cloned.Description = %q, want %q", cloned.Description, "Original step")
		}
		if cloned.Retries == nil || *cloned.Retries != 2 {
			t.Errorf("cloned.Retries = %v, want 2", cloned.Retries)
		}
		if cloned.Component != "WORKFLOW" {
			t.Errorf("cloned.Component = %q, want %q", cloned.Component, "WORKFLOW")
		}
		if cloned.Execute == nil {
			t.Error("cloned.Execute is nil, should share the original function")
		}
	})

	t.Run("should not affect original when clone is modified", func(t *testing.T) {
		original := &wf.Step{
			ID:          "original",
			Description: "Original",
		}

		cloned := CloneStep(original, "cloned")
		cloned.Description = "Modified clone"

		if original.Description != "Original" {
			t.Errorf("original.Description changed to %q, should remain %q", original.Description, "Original")
		}
	})
}

// ---------------------------------------------------------------------------
// NewEventedWorkflow tests
// ---------------------------------------------------------------------------

func TestNewEventedWorkflow(t *testing.T) {
	t.Run("should create workflow with ID", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{
			ID: "test-workflow",
		})
		if ew.ID != "test-workflow" {
			t.Errorf("ew.ID = %q, want %q", ew.ID, "test-workflow")
		}
		if ew.EngineType != "evented" {
			t.Errorf("ew.EngineType = %q, want %q", ew.EngineType, "evented")
		}
	})

	t.Run("should initialize StepDefs to empty map when nil", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{
			ID: "test-workflow",
		})
		if ew.StepDefs == nil {
			t.Error("ew.StepDefs is nil, want empty map")
		}
		if len(ew.StepDefs) != 0 {
			t.Errorf("len(ew.StepDefs) = %d, want 0", len(ew.StepDefs))
		}
	})

	t.Run("should preserve provided steps", func(t *testing.T) {
		steps := map[string]*wf.Step{
			"step-1": {ID: "step-1"},
			"step-2": {ID: "step-2"},
		}
		ew := NewEventedWorkflow(EventedWorkflowConfig{
			ID:    "test-workflow",
			Steps: steps,
		})
		if len(ew.StepDefs) != 2 {
			t.Errorf("len(ew.StepDefs) = %d, want 2", len(ew.StepDefs))
		}
	})

	t.Run("should store options", func(t *testing.T) {
		validateInputs := true
		opts := &wf.WorkflowOptions{
			ValidateInputs: &validateInputs,
		}
		ew := NewEventedWorkflow(EventedWorkflowConfig{
			ID:      "test-workflow",
			Options: opts,
		})
		if ew.Options == nil {
			t.Fatal("ew.Options is nil")
		}
		if ew.Options.ValidateInputs == nil || *ew.Options.ValidateInputs != true {
			t.Error("ew.Options.ValidateInputs should be true")
		}
	})
}

// ---------------------------------------------------------------------------
// EventedWorkflow methods tests
// ---------------------------------------------------------------------------

func TestEventedWorkflow_GetID(t *testing.T) {
	t.Run("should return the workflow ID", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "my-wf"})
		if ew.GetID() != "my-wf" {
			t.Errorf("GetID() = %q, want %q", ew.GetID(), "my-wf")
		}
	})
}

func TestEventedWorkflow_SetStepFlow(t *testing.T) {
	t.Run("should set the step graph", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "test-wf"})
		steps := []wf.StepFlowEntry{
			{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
			{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s2"}},
		}
		ew.SetStepFlow(steps)
		if len(ew.StepGraph) != 2 {
			t.Errorf("len(StepGraph) = %d, want 2", len(ew.StepGraph))
		}
	})
}

func TestEventedWorkflow_Commit(t *testing.T) {
	t.Run("should set Committed to true and build execution graph", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "commit-wf"})
		ew.StepGraph = []wf.StepFlowEntry{
			{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
		}
		ew.Commit()
		if !ew.Committed {
			t.Error("ew.Committed should be true after Commit()")
		}
		if ew.ExecutionGraph.ID != "commit-wf" {
			t.Errorf("ExecutionGraph.ID = %q, want %q", ew.ExecutionGraph.ID, "commit-wf")
		}
		if len(ew.ExecutionGraph.Steps) != 1 {
			t.Errorf("len(ExecutionGraph.Steps) = %d, want 1", len(ew.ExecutionGraph.Steps))
		}
	})
}

func TestEventedWorkflow_BuildExecutionGraph(t *testing.T) {
	t.Run("should return an execution graph with correct ID", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "build-wf"})
		ew.StepGraph = []wf.StepFlowEntry{
			{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
		}
		graph := ew.BuildExecutionGraph()
		if graph.ID != "build-wf" {
			t.Errorf("graph.ID = %q, want %q", graph.ID, "build-wf")
		}
		if len(graph.Steps) != 1 {
			t.Errorf("len(graph.Steps) = %d, want 1", len(graph.Steps))
		}
	})
}

func TestEventedWorkflow_RegisterMastra(t *testing.T) {
	t.Run("should register mastra and set logger", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "reg-wf"})
		log := &mockLogger{}
		mastra := &mockMastra{log: log}
		ew.RegisterMastra(mastra)
		if ew.mastra == nil {
			t.Error("mastra not registered")
		}
		if ew.log == nil {
			t.Error("logger not set")
		}
	})

	t.Run("should propagate to execution engine if set", func(t *testing.T) {
		log := &mockLogger{}
		mastra := &mockMastra{log: log}

		// Create a mock event processor
		ep := &testEventProcessor{}
		engine := NewEventedExecutionEngine(nil, ep, nil)

		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "prop-wf"})
		ew.executionEngine = engine
		ew.RegisterMastra(mastra)

		if !ep.registered {
			t.Error("execution engine should have propagated RegisterMastra to event processor")
		}
	})
}

type testEventProcessor struct {
	registered bool
}

func (t *testEventProcessor) RegisterMastra(mastra Mastra) {
	t.registered = true
}

// ---------------------------------------------------------------------------
// CloneWorkflow tests
// ---------------------------------------------------------------------------

func TestCloneWorkflow(t *testing.T) {
	t.Run("should clone workflow with new ID", func(t *testing.T) {
		original := NewEventedWorkflow(EventedWorkflowConfig{
			ID: "original-wf",
			Steps: map[string]*wf.Step{
				"s1": {ID: "s1"},
			},
		})
		original.StepGraph = []wf.StepFlowEntry{
			{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
		}
		original.Committed = true

		cloned := CloneWorkflow(original, "cloned-wf")

		if cloned.ID != "cloned-wf" {
			t.Errorf("cloned.ID = %q, want %q", cloned.ID, "cloned-wf")
		}
		if !cloned.Committed {
			t.Error("cloned.Committed should be true")
		}
		if len(cloned.StepGraph) != 1 {
			t.Errorf("len(cloned.StepGraph) = %d, want 1", len(cloned.StepGraph))
		}
	})
}

// ---------------------------------------------------------------------------
// EventedRun tests
// ---------------------------------------------------------------------------

func TestNewEventedRun(t *testing.T) {
	t.Run("should create run with correct initial state", func(t *testing.T) {
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			ResourceID: "resource-1",
		})
		if run.WorkflowID != "wf-1" {
			t.Errorf("run.WorkflowID = %q, want %q", run.WorkflowID, "wf-1")
		}
		if run.RunID != "run-1" {
			t.Errorf("run.RunID = %q, want %q", run.RunID, "run-1")
		}
		if run.ResourceID != "resource-1" {
			t.Errorf("run.ResourceID = %q, want %q", run.ResourceID, "resource-1")
		}
		if run.WorkflowRunStatus != "pending" {
			t.Errorf("run.WorkflowRunStatus = %q, want %q", run.WorkflowRunStatus, "pending")
		}
	})

	t.Run("should have a cancellable context", func(t *testing.T) {
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		})
		if run.abortCtx == nil {
			t.Error("run.abortCtx is nil")
		}
		if run.abortCancel == nil {
			t.Error("run.abortCancel is nil")
		}
	})
}

func TestEventedRun_Cancel(t *testing.T) {
	t.Run("should cancel the context", func(t *testing.T) {
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		})

		run.Cancel()

		// Context should be cancelled
		select {
		case <-run.abortCtx.Done():
			// expected
		default:
			t.Error("context should be cancelled after Cancel()")
		}
	})

	t.Run("should publish cancel event when mastra is set", func(t *testing.T) {
		pubsub := newMockPubSub()
		mastra := &mockMastra{pubsub: pubsub}
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			Mastra:     mastra,
		})

		run.Cancel()

		if len(pubsub.published) != 1 {
			t.Fatalf("len(published) = %d, want 1", len(pubsub.published))
		}
		if pubsub.published[0].Topic != "workflows" {
			t.Errorf("topic = %q, want %q", pubsub.published[0].Topic, "workflows")
		}
		event, ok := pubsub.published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		if event["type"] != "workflow.cancel" {
			t.Errorf("event type = %v, want workflow.cancel", event["type"])
		}
	})
}

func TestEventedRun_Start(t *testing.T) {
	t.Run("should return error when no serialized step graph", func(t *testing.T) {
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
		})

		_, err := run.Start(StartParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "execution flow of workflow is not defined; add steps via .Then(), .Branch(), etc." {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("should return error when no execution engine", func(t *testing.T) {
		run := NewEventedRun(EventedRunConfig{
			WorkflowID: "wf-1",
			RunID:      "run-1",
			SerializedStepGraph: []wf.SerializedStepFlowEntry{
				{Type: wf.StepFlowEntryTypeStep},
			},
		})

		_, err := run.Start(StartParams{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "execution engine not configured" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateRun tests
// ---------------------------------------------------------------------------

func TestEventedWorkflow_CreateRun(t *testing.T) {
	t.Run("should create a run with generated ID when none provided", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "create-run-wf"})
		run, err := ew.CreateRun(&RunOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run == nil {
			t.Fatal("run is nil")
		}
		if run.RunID == "" {
			t.Error("run.RunID should not be empty")
		}
		if run.WorkflowID != "create-run-wf" {
			t.Errorf("run.WorkflowID = %q, want %q", run.WorkflowID, "create-run-wf")
		}
	})

	t.Run("should create a run with specified ID", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "create-run-wf"})
		run, err := ew.CreateRun(&RunOptions{RunID: "custom-run-id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.RunID != "custom-run-id" {
			t.Errorf("run.RunID = %q, want %q", run.RunID, "custom-run-id")
		}
	})

	t.Run("should return existing run for same ID", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "create-run-wf"})
		run1, err := ew.CreateRun(&RunOptions{RunID: "same-id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		run2, err := ew.CreateRun(&RunOptions{RunID: "same-id"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if run1 != run2 {
			t.Error("expected same run instance for same ID")
		}
	})

	t.Run("should set resourceID on run", func(t *testing.T) {
		ew := NewEventedWorkflow(EventedWorkflowConfig{ID: "create-run-wf"})
		run, err := ew.CreateRun(&RunOptions{
			RunID:      "res-run",
			ResourceID: "my-resource",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if run.ResourceID != "my-resource" {
			t.Errorf("run.ResourceID = %q, want %q", run.ResourceID, "my-resource")
		}
	})
}
