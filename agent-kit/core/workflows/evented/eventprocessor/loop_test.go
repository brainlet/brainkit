// Ported from: packages/core/src/workflows/evented/workflow-event-processor/loop.test.ts
package eventprocessor

import (
	"testing"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// mergeStepResults tests
// ---------------------------------------------------------------------------

func TestMergeStepResults(t *testing.T) {
	t.Run("should merge step result into empty map", func(t *testing.T) {
		original := map[string]any{}
		merged := mergeStepResults(original, "step-1", map[string]any{"status": "success"})
		if merged["step-1"] == nil {
			t.Fatal("merged[step-1] is nil")
		}
		result, ok := merged["step-1"].(map[string]any)
		if !ok {
			t.Fatal("merged[step-1] is not map[string]any")
		}
		if result["status"] != "success" {
			t.Errorf("result[status] = %v, want success", result["status"])
		}
	})

	t.Run("should preserve existing entries", func(t *testing.T) {
		original := map[string]any{
			"step-0": map[string]any{"status": "success"},
		}
		merged := mergeStepResults(original, "step-1", map[string]any{"status": "failed"})
		if merged["step-0"] == nil {
			t.Error("merged[step-0] should be preserved")
		}
		if merged["step-1"] == nil {
			t.Error("merged[step-1] should be added")
		}
	})

	t.Run("should overwrite existing entry with same key", func(t *testing.T) {
		original := map[string]any{
			"step-1": map[string]any{"status": "running"},
		}
		merged := mergeStepResults(original, "step-1", map[string]any{"status": "success"})
		result, ok := merged["step-1"].(map[string]any)
		if !ok {
			t.Fatal("merged[step-1] is not map[string]any")
		}
		if result["status"] != "success" {
			t.Errorf("result[status] = %v, want success", result["status"])
		}
	})

	t.Run("should not modify original map", func(t *testing.T) {
		original := map[string]any{
			"step-0": map[string]any{"status": "success"},
		}
		_ = mergeStepResults(original, "step-1", map[string]any{"status": "failed"})
		if _, exists := original["step-1"]; exists {
			t.Error("original map should not be modified")
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowLoop tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowLoop(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)
		err := ProcessWorkflowLoop(
			&ProcessorArgs{RunID: "run-1"},
			pubsub,
			se,
			nil,
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
		err := ProcessWorkflowLoop(
			&ProcessorArgs{RunID: "run-1"},
			pubsub,
			se,
			step,
			nil,
		)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("dowhile - should continue when condition is true", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoWhile,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil // condition true => keep looping
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		stepResult := map[string]any{"status": "success", "output": "result"}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
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
		// Should publish workflow.step.run (continue loop)
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run", evt["type"])
		}
	})

	t.Run("dowhile - should end loop when condition is false", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoWhile,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return false, nil // condition false => stop looping
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		stepResult := map[string]any{"status": "success", "output": "result"}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
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
		// Should publish workflow.step.end (end loop)
		if evt["type"] != "workflow.step.end" {
			t.Errorf("event type = %v, want workflow.step.end", evt["type"])
		}
	})

	t.Run("dountil - should end loop when condition is true", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoUntil,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil // condition true => stop (do-until)
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		stepResult := map[string]any{"status": "success", "output": "result"}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
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
		// dountil: condition true => end loop
		if evt["type"] != "workflow.step.end" {
			t.Errorf("event type = %v, want workflow.step.end", evt["type"])
		}
	})

	t.Run("dountil - should continue loop when condition is false", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoUntil,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return false, nil // condition false => continue (do-until)
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		stepResult := map[string]any{"status": "success", "output": "result"}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
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
		// dountil: condition false => continue loop
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run", evt["type"])
		}
	})

	t.Run("should default to dowhile when loopKind is empty", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeLoop,
			// LoopKind is empty - should default to dowhile
			Step: &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					return true, nil // condition true => for dowhile, keep looping
				},
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			StepResults:   map[string]any{},
			ActiveSteps:   map[string]bool{},
		}

		stepResult := map[string]any{"status": "success", "output": "result"}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
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
		// default dowhile: condition true => continue
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run (dowhile default with true condition)", evt["type"])
		}
	})

	t.Run("should use stepResult output as condition input", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		var capturedInput any
		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoWhile,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					capturedInput = params.InputData
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
		}

		stepResult := map[string]any{
			"status": "success",
			"output": map[string]any{"counter": 5},
		}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		inputMap, ok := capturedInput.(map[string]any)
		if !ok {
			t.Fatalf("capturedInput is not map[string]any, got %T", capturedInput)
		}
		if inputMap["counter"] != 5 {
			t.Errorf("capturedInput[counter] = %v, want 5", inputMap["counter"])
		}
	})

	t.Run("should pass nil input when stepResult has non-success status", func(t *testing.T) {
		pubsub := newMockPubSub()
		se := evented.NewStepExecutor(nil)

		var capturedInput any
		step := &wf.StepFlowEntry{
			Type:     wf.StepFlowEntryTypeLoop,
			LoopKind: wf.LoopTypeDoWhile,
			Step:     &wf.Step{ID: "loop-step"},
			Conditions: []wf.ConditionFunction{
				func(params *wf.ExecuteFunctionParams) (bool, error) {
					capturedInput = params.InputData
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
		}

		stepResult := map[string]any{
			"status": "failed",
			"output": "some output",
		}

		err := ProcessWorkflowLoop(args, pubsub, se, step, stepResult)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedInput != nil {
			t.Errorf("capturedInput = %v, want nil for non-success status", capturedInput)
		}
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowForEach tests
// ---------------------------------------------------------------------------

func TestProcessWorkflowForEach(t *testing.T) {
	t.Run("should return error for nil step", func(t *testing.T) {
		pubsub := newMockPubSub()
		err := ProcessWorkflowForEach(
			&ProcessorArgs{RunID: "run-1"},
			pubsub,
			nil,
			nil,
		)
		if err == nil {
			t.Fatal("expected error for nil step")
		}
	})

	t.Run("should return error for wrong step type", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		err := ProcessWorkflowForEach(
			&ProcessorArgs{RunID: "run-1"},
			pubsub,
			nil,
			step,
		)
		if err == nil {
			t.Fatal("expected error for wrong step type")
		}
	})

	t.Run("should kick off iterations with concurrency 1 by default", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeForeach,
			Step: &wf.Step{ID: "foreach-step"},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			PrevResult: map[string]any{
				"status": "success",
				"output": []any{"a", "b", "c"},
			},
			StepResults: map[string]any{},
			ActiveSteps: map[string]bool{},
		}

		err := ProcessWorkflowForEach(args, pubsub, nil, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		// With concurrency 1, should kick off 1 iteration
		if len(published) != 1 {
			t.Fatalf("expected 1 published event (concurrency=1), got %d", len(published))
		}

		evt, ok := published[0].Event.(map[string]any)
		if !ok {
			t.Fatal("event is not map[string]any")
		}
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run", evt["type"])
		}
		data, ok := evt["data"].(map[string]any)
		if !ok {
			t.Fatal("event data is not map[string]any")
		}
		execPath, ok := data["executionPath"].([]int)
		if !ok {
			t.Fatal("executionPath is not []int")
		}
		if len(execPath) != 2 || execPath[0] != 0 || execPath[1] != 0 {
			t.Errorf("executionPath = %v, want [0, 0]", execPath)
		}
	})

	t.Run("should kick off multiple iterations with higher concurrency", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeForeach,
			Step: &wf.Step{ID: "foreach-step"},
			ForeachOpts: &wf.ForeachOpts{
				Concurrency: 3,
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			PrevResult: map[string]any{
				"status": "success",
				"output": []any{"a", "b", "c", "d", "e"},
			},
			StepResults: map[string]any{},
			ActiveSteps: map[string]bool{},
		}

		err := ProcessWorkflowForEach(args, pubsub, nil, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		// With concurrency 3, should kick off 3 iterations
		if len(published) != 3 {
			t.Fatalf("expected 3 published events (concurrency=3), got %d", len(published))
		}
	})

	t.Run("should cap concurrency at target length", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeForeach,
			Step: &wf.Step{ID: "foreach-step"},
			ForeachOpts: &wf.ForeachOpts{
				Concurrency: 10,
			},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{0},
			PrevResult: map[string]any{
				"status": "success",
				"output": []any{"a", "b"},
			},
			StepResults: map[string]any{},
			ActiveSteps: map[string]bool{},
		}

		err := ProcessWorkflowForEach(args, pubsub, nil, step)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		published := pubsub.getPublished()
		// Concurrency is 10 but only 2 items, should kick off 2
		if len(published) != 2 {
			t.Fatalf("expected 2 published events, got %d", len(published))
		}
	})

	t.Run("should advance to next step when all iterations complete", func(t *testing.T) {
		pubsub := newMockPubSub()
		step := &wf.StepFlowEntry{
			Type: wf.StepFlowEntryTypeForeach,
			Step: &wf.Step{ID: "foreach-step"},
		}

		args := &ProcessorArgs{
			WorkflowID:    "wf-1",
			RunID:         "run-1",
			ExecutionPath: []int{2},
			PrevResult: map[string]any{
				"status": "success",
				"output": []any{"a", "b"},
			},
			StepResults: map[string]any{
				"foreach-step": map[string]any{
					"status": "success",
					"output": []any{
						map[string]any{"status": "success", "output": "result-a"},
						map[string]any{"status": "success", "output": "result-b"},
					},
				},
			},
			ActiveSteps: map[string]bool{},
		}

		err := ProcessWorkflowForEach(args, pubsub, nil, step)
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
		if evt["type"] != "workflow.step.run" {
			t.Errorf("event type = %v, want workflow.step.run (advance)", evt["type"])
		}
		data, ok := evt["data"].(map[string]any)
		if !ok {
			t.Fatal("event data is not map[string]any")
		}
		// Should advance from [2] to [3]
		execPath, ok := data["executionPath"].([]int)
		if !ok {
			t.Fatal("executionPath is not []int")
		}
		if len(execPath) != 1 || execPath[0] != 3 {
			t.Errorf("executionPath = %v, want [3]", execPath)
		}
	})
}
