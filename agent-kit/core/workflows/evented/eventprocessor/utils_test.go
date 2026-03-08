// Ported from: packages/core/src/workflows/evented/workflow-event-processor/utils.test.ts
package eventprocessor

import (
	"testing"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Mock workflow for tests
// ---------------------------------------------------------------------------

type mockWorkflow struct {
	id                  string
	stepGraph           []wf.StepFlowEntry
	serializedStepGraph []wf.SerializedStepFlowEntry
	options             WorkflowOptions
}

func (m *mockWorkflow) GetID() string {
	return m.id
}

func (m *mockWorkflow) GetStepGraph() []wf.StepFlowEntry {
	return m.stepGraph
}

func (m *mockWorkflow) GetSerializedStepGraph() []wf.SerializedStepFlowEntry {
	return m.serializedStepGraph
}

func (m *mockWorkflow) GetOptions() WorkflowOptions {
	return m.options
}

// ---------------------------------------------------------------------------
// isWorkflowStep tests
// ---------------------------------------------------------------------------

func TestIsWorkflowStep(t *testing.T) {
	t.Run("should return false for nil step", func(t *testing.T) {
		if isWorkflowStep(nil) {
			t.Error("isWorkflowStep(nil) = true, want false")
		}
	})

	t.Run("should return false for regular step", func(t *testing.T) {
		step := &wf.Step{ID: "regular-step"}
		if isWorkflowStep(step) {
			t.Error("isWorkflowStep(regular) = true, want false")
		}
	})

	t.Run("should return true for WORKFLOW component", func(t *testing.T) {
		step := &wf.Step{ID: "nested-wf", Component: "WORKFLOW"}
		if !isWorkflowStep(step) {
			t.Error("isWorkflowStep(WORKFLOW) = false, want true")
		}
	})

	t.Run("should return false for other components", func(t *testing.T) {
		step := &wf.Step{ID: "step", Component: "OTHER"}
		if isWorkflowStep(step) {
			t.Error("isWorkflowStep(OTHER) = true, want false")
		}
	})
}

// ---------------------------------------------------------------------------
// IsExecutableStep tests
// ---------------------------------------------------------------------------

func TestIsExecutableStep(t *testing.T) {
	t.Run("should return false for nil", func(t *testing.T) {
		if IsExecutableStep(nil) {
			t.Error("IsExecutableStep(nil) = true, want false")
		}
	})

	t.Run("should return true for step type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeStep}
		if !IsExecutableStep(entry) {
			t.Error("IsExecutableStep(step) = false, want true")
		}
	})

	t.Run("should return true for loop type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeLoop}
		if !IsExecutableStep(entry) {
			t.Error("IsExecutableStep(loop) = false, want true")
		}
	})

	t.Run("should return true for foreach type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeForeach}
		if !IsExecutableStep(entry) {
			t.Error("IsExecutableStep(foreach) = false, want true")
		}
	})

	t.Run("should return false for sleep type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeSleep}
		if IsExecutableStep(entry) {
			t.Error("IsExecutableStep(sleep) = true, want false")
		}
	})

	t.Run("should return false for sleepUntil type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeSleepUntil}
		if IsExecutableStep(entry) {
			t.Error("IsExecutableStep(sleepUntil) = true, want false")
		}
	})

	t.Run("should return false for parallel type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeParallel}
		if IsExecutableStep(entry) {
			t.Error("IsExecutableStep(parallel) = true, want false")
		}
	})

	t.Run("should return false for conditional type", func(t *testing.T) {
		entry := &wf.StepFlowEntry{Type: wf.StepFlowEntryTypeConditional}
		if IsExecutableStep(entry) {
			t.Error("IsExecutableStep(conditional) = true, want false")
		}
	})
}

// ---------------------------------------------------------------------------
// GetStep tests
// ---------------------------------------------------------------------------

func TestGetStep(t *testing.T) {
	t.Run("should return nil for nil workflow", func(t *testing.T) {
		result := GetStep(nil, []int{0})
		if result != nil {
			t.Errorf("GetStep(nil, ...) = %v, want nil", result)
		}
	})

	t.Run("should return nil for empty execution path", func(t *testing.T) {
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
			},
		}
		result := GetStep(w, []int{})
		if result != nil {
			t.Errorf("GetStep(w, []) = %v, want nil", result)
		}
	})

	t.Run("should return step at simple path", func(t *testing.T) {
		step := &wf.Step{ID: "s1"}
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeStep, Step: step},
			},
		}
		result := GetStep(w, []int{0})
		if result == nil {
			t.Fatal("GetStep returned nil")
		}
		if result.ID != "s1" {
			t.Errorf("result.ID = %q, want %q", result.ID, "s1")
		}
	})

	t.Run("should return nil for out-of-bounds path", func(t *testing.T) {
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeStep, Step: &wf.Step{ID: "s1"}},
			},
		}
		result := GetStep(w, []int{5})
		if result != nil {
			t.Errorf("GetStep(w, [5]) = %v, want nil", result)
		}
	})

	t.Run("should return step within parallel block", func(t *testing.T) {
		step1 := &wf.Step{ID: "p-s1"}
		step2 := &wf.Step{ID: "p-s2"}
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{
					Type: wf.StepFlowEntryTypeParallel,
					Steps: []wf.StepFlowStepEntry{
						{Type: "step", Step: step1},
						{Type: "step", Step: step2},
					},
				},
			},
		}
		result := GetStep(w, []int{0, 1})
		if result == nil {
			t.Fatal("GetStep returned nil")
		}
		if result.ID != "p-s2" {
			t.Errorf("result.ID = %q, want %q", result.ID, "p-s2")
		}
	})

	t.Run("should return step within conditional block", func(t *testing.T) {
		step1 := &wf.Step{ID: "c-s1"}
		step2 := &wf.Step{ID: "c-s2"}
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{
					Type: wf.StepFlowEntryTypeConditional,
					Steps: []wf.StepFlowStepEntry{
						{Type: "step", Step: step1},
						{Type: "step", Step: step2},
					},
				},
			},
		}
		result := GetStep(w, []int{0, 0})
		if result == nil {
			t.Fatal("GetStep returned nil")
		}
		if result.ID != "c-s1" {
			t.Errorf("result.ID = %q, want %q", result.ID, "c-s1")
		}
	})

	t.Run("should return nil for parallel with out-of-bounds sub-index", func(t *testing.T) {
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{
					Type: wf.StepFlowEntryTypeParallel,
					Steps: []wf.StepFlowStepEntry{
						{Type: "step", Step: &wf.Step{ID: "p-s1"}},
					},
				},
			},
		}
		result := GetStep(w, []int{0, 5})
		if result != nil {
			t.Errorf("GetStep(w, [0,5]) = %v, want nil", result)
		}
	})

	t.Run("should return step for foreach type", func(t *testing.T) {
		step := &wf.Step{ID: "foreach-step"}
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeForeach, Step: step},
			},
		}
		result := GetStep(w, []int{0})
		if result == nil {
			t.Fatal("GetStep returned nil")
		}
		if result.ID != "foreach-step" {
			t.Errorf("result.ID = %q, want %q", result.ID, "foreach-step")
		}
	})

	t.Run("should return nil for non-executable step types without step", func(t *testing.T) {
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeSleep, ID: "sleep-1"},
			},
		}
		result := GetStep(w, []int{0})
		if result != nil {
			t.Errorf("GetStep for sleep step = %v, want nil", result)
		}
	})

	t.Run("should return step for loop type", func(t *testing.T) {
		step := &wf.Step{ID: "loop-step"}
		w := &mockWorkflow{
			stepGraph: []wf.StepFlowEntry{
				{Type: wf.StepFlowEntryTypeLoop, Step: step},
			},
		}
		result := GetStep(w, []int{0})
		if result == nil {
			t.Fatal("GetStep returned nil")
		}
		if result.ID != "loop-step" {
			t.Errorf("result.ID = %q, want %q", result.ID, "loop-step")
		}
	})
}

// ---------------------------------------------------------------------------
// GetNestedWorkflow tests
// ---------------------------------------------------------------------------

func TestGetNestedWorkflow(t *testing.T) {
	t.Run("should return nil for nil parentWorkflow", func(t *testing.T) {
		result := GetNestedWorkflow(nil, nil)
		if result != nil {
			t.Errorf("GetNestedWorkflow(nil, nil) = %v, want nil", result)
		}
	})

	t.Run("should return nil when mastra is nil", func(t *testing.T) {
		pw := &ParentWorkflow{
			WorkflowID:    "parent-wf",
			RunID:         "run-1",
			ExecutionPath: []int{0},
		}
		result := GetNestedWorkflow(nil, pw)
		if result != nil {
			t.Errorf("GetNestedWorkflow with nil mastra = %v, want nil", result)
		}
	})
}
