// Ported from: packages/core/src/workflows/evented/workflow-event-processor/utils.ts
package eventprocessor

import (
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// Workflow is a stub interface for the Workflow type.
// TODO: Replace with actual Workflow type from workflows/workflow.go when ported.
type Workflow interface {
	GetID() string
	GetStepGraph() []wf.StepFlowEntry
	GetSerializedStepGraph() []wf.SerializedStepFlowEntry
	GetOptions() WorkflowOptions
}

// WorkflowOptions is a stub for workflow options.
// TODO: Replace with actual type when workflow.go is ported.
type WorkflowOptions struct {
	ValidateInputs        bool
	ShouldPersistSnapshot func(params wf.ShouldPersistSnapshotParams) bool
}

// EventedWorkflow is a stub marker interface for EventedWorkflow.
// TODO: Replace with actual EventedWorkflow from evented/workflow.go when ported.
type EventedWorkflow interface {
	Workflow
	IsEvented() bool
}

// ---------------------------------------------------------------------------
// isWorkflowStep
// ---------------------------------------------------------------------------

// isWorkflowStep checks if a step is actually a Workflow (nested workflow).
// A step is a Workflow if it is an EventedWorkflow or has Component == "WORKFLOW".
// TS equivalent: function isWorkflowStep(step: unknown): step is Workflow
func isWorkflowStep(step *wf.Step) bool {
	if step == nil {
		return false
	}
	if step.Component == "WORKFLOW" {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// GetNestedWorkflow
// ---------------------------------------------------------------------------

// GetNestedWorkflow retrieves a nested workflow from the Mastra registry by traversing
// parent workflow references and execution paths.
// TS equivalent: export function getNestedWorkflow(mastra, parentWorkflow): Workflow | null
func GetNestedWorkflow(mastra any, parentWorkflow *ParentWorkflow) Workflow {
	if parentWorkflow == nil {
		return nil
	}

	var workflow Workflow

	if parentWorkflow.ParentWorkflow != nil {
		nestedWorkflow := GetNestedWorkflow(mastra, parentWorkflow.ParentWorkflow)
		if nestedWorkflow == nil {
			return nil
		}
		workflow = nestedWorkflow
	}

	if workflow == nil {
		// TODO: Use mastra.GetWorkflow(parentWorkflow.WorkflowID) when available
		_ = mastra
		return nil
	}

	stepGraph := workflow.GetStepGraph()
	if len(parentWorkflow.ExecutionPath) == 0 {
		return nil
	}

	idx := parentWorkflow.ExecutionPath[0]
	if idx >= len(stepGraph) {
		return nil
	}

	parentStep := &stepGraph[idx]

	if (parentStep.Type == wf.StepFlowEntryTypeParallel || parentStep.Type == wf.StepFlowEntryTypeConditional) &&
		len(parentWorkflow.ExecutionPath) > 1 {
		subIdx := parentWorkflow.ExecutionPath[1]
		if subIdx >= len(parentStep.Steps) {
			return nil
		}
		nested := parentStep.Steps[subIdx]
		parentStep = &wf.StepFlowEntry{Type: wf.StepFlowEntryType(nested.Type), Step: nested.Step}
	}

	if parentStep.Type == wf.StepFlowEntryTypeStep || parentStep.Type == wf.StepFlowEntryTypeLoop {
		if isWorkflowStep(parentStep.Step) {
			// TODO: Return the step as a Workflow when proper casting is available
			return nil
		}
		return nil
	}

	if parentStep.Type == wf.StepFlowEntryTypeForeach {
		if isWorkflowStep(parentStep.Step) {
			// TODO: Return the step as a Workflow when proper casting is available
			return nil
		}
		return nil
	}

	return nil
}

// ---------------------------------------------------------------------------
// GetStep
// ---------------------------------------------------------------------------

// GetStep retrieves a Step from a workflow at the given execution path.
// TS equivalent: export function getStep(workflow, executionPath): Step | null
func GetStep(workflow Workflow, executionPath []int) *wf.Step {
	if workflow == nil || len(executionPath) == 0 {
		return nil
	}

	idx := 0
	stepGraph := workflow.GetStepGraph()
	if executionPath[0] >= len(stepGraph) {
		return nil
	}

	parentStep := &stepGraph[executionPath[0]]

	if parentStep.Type == wf.StepFlowEntryTypeParallel || parentStep.Type == wf.StepFlowEntryTypeConditional {
		if len(executionPath) > 1 && executionPath[1] < len(parentStep.Steps) {
			nested := parentStep.Steps[executionPath[1]]
			parentStep = &wf.StepFlowEntry{Type: wf.StepFlowEntryType(nested.Type), Step: nested.Step}
			idx++
		}
	} else if parentStep.Type == wf.StepFlowEntryTypeForeach {
		return parentStep.Step
	}

	if parentStep.Type != wf.StepFlowEntryTypeStep && parentStep.Type != wf.StepFlowEntryTypeLoop {
		return nil
	}

	// Check if step is an EventedWorkflow (nested workflow) - recurse into it
	if parentStep.Step != nil && parentStep.Step.Component == "WORKFLOW" {
		// Recursion into nested workflow
		// TODO: When EventedWorkflow is fully ported, cast and recurse:
		// return GetStep(nestedWorkflow, executionPath[idx+1:])
		_ = idx
		return parentStep.Step
	}

	return parentStep.Step
}

// ---------------------------------------------------------------------------
// IsExecutableStep
// ---------------------------------------------------------------------------

// IsExecutableStep checks if a step flow entry is executable (step, loop, or foreach).
// TS equivalent: export function isExecutableStep(step: StepFlowEntry)
func IsExecutableStep(step *wf.StepFlowEntry) bool {
	if step == nil {
		return false
	}
	return step.Type == wf.StepFlowEntryTypeStep ||
		step.Type == wf.StepFlowEntryTypeLoop ||
		step.Type == wf.StepFlowEntryTypeForeach
}
