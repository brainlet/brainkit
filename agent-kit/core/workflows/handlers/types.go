// Ported from: packages/core/src/workflows/handlers/ (shared types)
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// DefaultEngine Interface
// ---------------------------------------------------------------------------

// DefaultEngine is the interface that handlers use to call back into the
// DefaultExecutionEngine. This avoids import cycles between the handlers
// package and the workflows package.
type DefaultEngine interface {
	wf.ExecutionEngine

	// GetMastra returns the engine's mastra instance.
	GetMastra() wf.Mastra

	// GetEngineContext returns engine-specific context for step execution.
	GetEngineContext() map[string]any

	// GetStepOutput returns the output of a step from step results.
	GetStepOutput(stepResults map[string]any, step *wf.StepFlowEntry) any

	// ExecuteSleepDuration sleeps for the given duration in milliseconds.
	ExecuteSleepDuration(duration int64, sleepID string, workflowID string) error

	// ExecuteSleepUntilDate sleeps until the given date.
	ExecuteSleepUntilDate(date time.Time, sleepUntilID string, workflowID string) error

	// WrapDurableOperation wraps an operation for durability.
	WrapDurableOperation(operationID string, fn func() (any, error)) (any, error)

	// EvaluateCondition evaluates a single condition for conditional execution.
	EvaluateCondition(conditionFn wf.ConditionFunction, index int, context *wf.ExecuteFunctionParams, operationID string) (*int, error)

	// IsNestedWorkflowStep checks if a step is a nested workflow.
	IsNestedWorkflowStep(step *wf.Step) bool

	// ExecuteWorkflowStep executes a nested workflow step. Returns nil if not handled.
	ExecuteWorkflowStep(params ExecuteWorkflowStepParams) (*wf.StepResult, error)

	// OnStepExecutionStart handles step execution start - emit events and return start timestamp.
	OnStepExecutionStart(params StepExecutionStartParams) (int64, error)

	// GetOrGenerateRetryCount gets or generates the retry count for a step.
	GetOrGenerateRetryCount(stepID string) int

	// ExecuteStepWithRetry executes a step with retry logic.
	ExecuteStepWithRetry(stepID string, runStep func() (any, error), params RetryParams) (*StepRetryResult, error)

	// RequiresDurableContextSerialization returns whether this engine requires context serialization.
	RequiresDurableContextSerialization() bool

	// SerializeRequestContext serializes a RequestContext to a map.
	SerializeRequestContext(rc *requestcontext.RequestContext) map[string]any

	// BuildMutableContext builds MutableContext from current execution state.
	BuildMutableContext(ec *wf.ExecutionContext) wf.MutableContext

	// ApplyMutableContext applies mutable context changes back to the execution context.
	ApplyMutableContext(ec *wf.ExecutionContext, mc wf.MutableContext)

	// FormatResultError formats an error for the workflow result.
	FormatResultError(err error, lastOutput wf.StepResult) any

	// ExecuteStepHandler executes a single step.
	ExecuteStepHandler(params ExecuteStepParams) (*wf.StepExecutionResult, error)

	// ExecuteEntryHandler executes a single entry.
	ExecuteEntryHandler(params ExecuteEntryParams) (*wf.EntryExecutionResult, error)

	// ExecuteParallelHandler executes parallel steps.
	ExecuteParallelHandler(params ExecuteParallelParams) (*wf.StepResult, error)

	// ExecuteConditionalHandler executes conditional steps.
	ExecuteConditionalHandler(params ExecuteConditionalParams) (*wf.StepResult, error)

	// ExecuteLoopHandler executes a loop.
	ExecuteLoopHandler(params ExecuteLoopParams) (*wf.StepResult, error)

	// ExecuteForeachHandler executes a foreach.
	ExecuteForeachHandler(params ExecuteForeachParams) (*wf.StepResult, error)

	// ExecuteSleepHandler executes a sleep.
	ExecuteSleepHandler(params ExecuteSleepParams) error

	// ExecuteSleepUntilHandler executes a sleepUntil.
	ExecuteSleepUntilHandler(params ExecuteSleepUntilParams) error

	// PersistStepUpdateHandler persists a step update.
	PersistStepUpdateHandler(params PersistStepUpdateParams) error
}

// ---------------------------------------------------------------------------
// Shared Resume Params
// ---------------------------------------------------------------------------

// ResumeParams holds resume data used across handlers.
type ResumeParams struct {
	Steps         []string
	StepResults   map[string]wf.StepResult
	ResumePayload any
	ResumePath    []int
	Label         string
	ForEachIndex  *int
}

// ---------------------------------------------------------------------------
// Step Execution Start Params
// ---------------------------------------------------------------------------

// StepExecutionStartParams holds parameters for the step execution start hook.
type StepExecutionStartParams struct {
	Step             *wf.Step
	InputData        any
	PubSub           events.PubSub
	ExecutionContext *wf.ExecutionContext
	StepCallID       string
	StepInfo         map[string]any
	OperationID      string
	SkipEmits        bool
}

// ---------------------------------------------------------------------------
// Execute Workflow Step Params
// ---------------------------------------------------------------------------

// ExecuteWorkflowStepParams holds parameters for executing a nested workflow step.
type ExecuteWorkflowStepParams struct {
	Step             *wf.Step
	StepResults      map[string]wf.StepResult
	ExecutionContext *wf.ExecutionContext
	Resume           *ResumeParams
	TimeTravel       *wf.TimeTravelExecutionParams
	PrevOutput       any
	InputData        any
	PubSub           events.PubSub
	StartedAt        int64
	AbortCtx         context.Context
	AbortCancel      context.CancelFunc
	RequestContext   *requestcontext.RequestContext
	OutputWriter     wf.OutputWriter
	StepSpan         obstypes.AnySpan
	PerStep          bool
	Observability    *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// Retry Params
// ---------------------------------------------------------------------------

// RetryParams holds parameters for step retry logic.
type RetryParams struct {
	Retries    int
	Delay      int
	StepSpan   obstypes.AnySpan
	WorkflowID string
	RunID      string
}

// StepRetryResult is the result of executeStepWithRetry.
// TS equivalent: discriminated union { ok: true, result: T } | { ok: false, error: ... }
type StepRetryResult struct {
	OK     bool
	Result any
	Error  *StepRetryError
}

// StepRetryError holds error info from a failed step retry.
type StepRetryError struct {
	Status   string // "failed"
	Error    error
	EndedAt  int64
	Tripwire *wf.StepTripwireInfo
}

// ---------------------------------------------------------------------------
// Persist Step Update Params
// ---------------------------------------------------------------------------

// PersistStepUpdateParams holds parameters for persisting a step update.
// TS equivalent: export interface PersistStepUpdateParams
type PersistStepUpdateParams struct {
	WorkflowID        string
	RunID             string
	ResourceID        string
	StepResults       map[string]wf.StepResult
	SerializedStepGraph []wf.SerializedStepFlowEntry
	ExecutionContext  *wf.ExecutionContext
	WorkflowStatus    wf.WorkflowRunStatus
	Result            map[string]any
	Error             any // SerializedError
	RequestContext    *requestcontext.RequestContext
}

// ---------------------------------------------------------------------------
// Log helpers
// ---------------------------------------------------------------------------

// getLogger safely gets a logger from the engine.
func getLogger(engine DefaultEngine) logger.IMastraLogger {
	if engine == nil {
		return nil
	}
	return engine.GetLogger()
}

// logError safely logs an error.
func logError(engine DefaultEngine, msg string, args ...any) {
	log := getLogger(engine)
	if log != nil {
		log.Error(fmt.Sprintf(msg, args...))
	}
}
