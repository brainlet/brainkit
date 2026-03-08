// Ported from: packages/core/src/workflows/evented/workflow-event-processor/index.ts (types section)
package eventprocessor

import (
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// ProcessorArgs - shared arguments for event processing functions
// ---------------------------------------------------------------------------

// ProcessorArgs holds the arguments passed to workflow event processing functions.
// TS equivalent: export type ProcessorArgs
type ProcessorArgs struct {
	ActiveSteps     map[string]bool
	Workflow        Workflow
	WorkflowID      string
	RunID           string
	ExecutionPath   []int
	StepResults     map[string]any
	ResumeSteps     []string
	PrevResult      map[string]any
	RequestContext  map[string]any
	TimeTravel      *wf.TimeTravelExecutionParams
	ResumeData      any
	ParentWorkflow  *ParentWorkflow
	ParentContext   *ParentContext
	RetryCount      int
	PerStep         bool
	State           map[string]any
	OutputOptions   *OutputOptions
	ForEachIndex    *int
	NestedRunID     string // runId of nested workflow when reporting back to parent
	InitialState    map[string]any
}

// ParentWorkflow holds data about the parent workflow for nested workflows.
// TS equivalent: export type ParentWorkflow
type ParentWorkflow struct {
	WorkflowID      string
	RunID           string
	ExecutionPath   []int
	Resume          bool
	StepResults     map[string]any
	ParentWorkflow  *ParentWorkflow
	StepID          string
}

// ParentContext holds context from a parent workflow for nested step completion.
type ParentContext struct {
	WorkflowID string
	Input      any
}

// OutputOptions holds output configuration.
type OutputOptions struct {
	IncludeState        bool `json:"includeState,omitempty"`
	IncludeResumeLabels bool `json:"includeResumeLabels,omitempty"`
}
