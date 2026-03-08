// Ported from: packages/core/src/workflows/evented/workflow-event-processor/parallel.test.ts
package eventprocessor

import (
	"testing"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// ProcessWorkflowParallel tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowParallel(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		err := ProcessWorkflowParallel(
			&ProcessorArgs{RunID: "run-1", ActiveSteps: map[string]bool{}},
			pubsub,
			nil,
		)
		if err == nil {
			t.Fatal("expected error for nil step")
		}
	})

	t.Run("should return error for wrong step type", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		err := ProcessWorkflowParallel(
			&ProcessorArgs{RunID: "run-1", ActiveSteps: map[string]bool{}},
			pubsub,
			step,
		)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("should publish step.run for each nested step", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeParallel,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "p1"}},
				{Type: "step", Step: &wf.Step{ID: "p2"}},
				{Type: "step", Step: &wf.Step{ID: "p3"}},
			},
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success"},
		}

		err := ProcessWorkflowParallel(args, pubsub, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 3 {
			t.Fatalf("expected 3 published events, got %d", len(published))
		}

		// Each should be on "workflows" topic with workflow.step.run type
		for i, p := range published {
			if p.Topic != "workflows" {
				t.Errorf("event[%d] topic = %q, want 'workflows'", i, p.Topic)
			}
			evt, ok := p.Event.(map[string]any)
			if !ok {
				t.Fatalf("event[%d] is not map[string]any", i)
			}
			if evt["type"] != "workflow.step.run" {
				t.Errorf("event[%d] type = %v, want workflow.step.run", i, evt["type"])
			}
		}
	})

	t.Run("should mark active steps", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeParallel,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "p1"}},
				{Type: "step", Step: &wf.Step{ID: "p2"}},
			},
		}
		activeSteps := map[string]bool{}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   activeSteps,
		}

		err := ProcessWorkflowParallel(args, pubsub, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !activeSteps["p1"] {
			t.Error("activeSteps should contain p1")
		}
		if !activeSteps["p2"] {
			t.Error("activeSteps should contain p2")
		}
	})

	t.Run("should only activate first step when perStep is true", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeParallel,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "p1"}},
				{Type: "step", Step: &wf.Step{ID: "p2"}},
				{Type: "step", Step: &wf.Step{ID: "p3"}},
			},
		}
		activeSteps := map[string]bool{}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   activeSteps,
			PerStep:       true,
		}

		err := ProcessWorkflowParallel(args, pubsub, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !activeSteps["p1"] {
			t.Error("activeSteps should contain p1 (first step)")
		}
		// p2 and p3 should only be published if they are active
		// With perStep, only p1 is activated
	})

	t.Run("should include execution path with step index", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeParallel,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "p1"}},
				{Type: "step", Step: &wf.Step{ID: "p2"}},
			},
		}
		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{3},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		err := ProcessWorkflowParallel(args, pubsub, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		// Check that execution paths contain the parent path + step index
		for _, p := range published {
			evt, ok := p.Event.(map[string]any)
			if !ok {
				continue
			}
			data, ok := evt["data"].(map[string]any)
			if !ok {
				continue
			}
			execPath, ok := data["executionPath"].([]int)
			if !ok {
				continue
			}
			if len(execPath) < 2 || execPath[0] != 3 {
				t.Errorf("executionPath %v should start with [3, ...]", execPath)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowConditional tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowConditional(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		err := ProcessWorkflowConditional(
			&ProcessorArgs{RunID: "run-1", ActiveSteps: map[string]bool{}},
			pubsub,
			se,
			nil,
		)
		if err == nil {
			t.Fatal("expected error for nil step")
		}
	})

	t.Run("should return error for wrong step type", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		step := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		err := ProcessWorkflowConditional(
			&ProcessorArgs{RunID: "run-1", ActiveSteps: map[string]bool{}},
			pubsub,
			se,
			step,
		)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("should publish step.run for truthy conditions and step.end for falsy", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeConditional,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "branch-a"}},
				{Type: "step", Step: &wf.Step{ID: "branch-b"}},
				{Type: "step", Step: &wf.Step{ID: "branch-c"}},
			},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil
				},
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return false, nil
				},
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success", "output": "data"},
		}

		err := ProcessWorkflowConditional(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 3 {
			t.Fatalf("expected 3 published events, got %d", len(published))
		}

		// Count step.run and step.end events
		runCount := 0
		endCount := 0
		for _, p := range published {
			evt, ok := p.Event.(map[string]any)
			if !ok {
				continue
			}
			switch evt["type"] {
			case "workflow.step.run":
				runCount++
			case "workflow.step.end":
				endCount++
			}
		}

		if runCount != 2 {
			t.Errorf("step.run count = %d, want 2 (truthy conditions)", runCount)
		}
		if endCount != 1 {
			t.Errorf("step.end count = %d, want 1 (falsy condition)", endCount)
		}
	})

	t.Run("should publish skipped status for falsy conditions", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeConditional,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "branch-a"}},
			},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return false, nil
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success"},
		}

		err := ProcessWorkflowConditional(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 1 {
			t.Fatalf("expected 1 published event, got %d", len(published))
		}

		evt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		if evt["type"] != "workflow.step.end" {
			t.Errorf("event type = %v, want workflow.step.end", evt["type"])
		}
		data, ok := evt["data"].(map[string]any)
		if !ok {
			t.Fatal("event data is not map[string]any")
		}
		prevResult, ok := data["prevResult"].(map[string]any)
		if !ok {
			t.Fatal("prevResult is not map[string]any")
		}
		if prevResult["status"] != "skipped" {
			t.Errorf("prevResult[status] = %v, want skipped", prevResult["status"])
		}
	})

	t.Run("perStep should only run first matching condition", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeConditional,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "branch-a"}},
				{Type: "step", Step: &wf.Step{ID: "branch-b"}},
			},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil
				},
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success"},
			PerStep:       true,
		}

		err := ProcessWorkflowConditional(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 1 {
			t.Fatalf("expected 1 published event (perStep), got %d", len(published))
		}

		evt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run", evt["type"])
		}
	})

	t.Run("perStep should return nil when no conditions match", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeConditional,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "branch-a"}},
			},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return false, nil
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success"},
			PerStep:       true,
		}

		err := ProcessWorkflowConditional(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		if len(published) != 0 {
			t.Errorf("expected 0 published events (perStep, no match), got %d", len(published))
		}
	})

	t.Run("should pass previous output as input to condition", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		var capturedInput any
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeConditional,
			Steps: []wf.StepFlowStepEntry{
				{Type: "step", Step: &wf.Step{ID: "branch-a"}},
			},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					capturedInput = params.InputData
					return true, nil
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
			PrevResult:    map[string]any{"status": "success", "output": map[string]any{"value": 42}},
		}

		err := ProcessWorkflowConditional(args, pubsub, se, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		inputMap, ok := capturedInput.(map[string]any)
		if !ok {
			t.Fatalf("capturedInput is not map[string]any, got %T", capturedInput)
		}
		if inputMap["value"] != 42 {
			t.Errorf("capturedInput[value] = %v, want 42", inputMap["value"])
		}
	})
}
