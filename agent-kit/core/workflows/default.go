// Ported from: packages/core/src/workflows/default.ts
package workflows

import (
	"context"
	"fmt"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/events"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// DefaultExecutionEngine
// ---------------------------------------------------------------------------

// DefaultExecutionEngine is the default implementation of the ExecutionEngine.
// TS equivalent: export class DefaultExecutionEngine extends ExecutionEngine
type DefaultExecutionEngine struct {
	BaseExecutionEngine

	// retryCounts tracks the retry count for each step.
	retryCounts map[string]int
}

// NewDefaultExecutionEngine creates a new DefaultExecutionEngine.
func NewDefaultExecutionEngine(mastra Mastra, options *ExecutionEngineOptions) *DefaultExecutionEngine {
	return &DefaultExecutionEngine{
		BaseExecutionEngine: NewBaseExecutionEngine(mastra, options),
		retryCounts:         make(map[string]int),
	}
}

// GetOrGenerateRetryCount gets or generates the retry count for a step.
// TS equivalent: getOrGenerateRetryCount(stepId)
func (e *DefaultExecutionEngine) GetOrGenerateRetryCount(stepID string) int {
	if count, ok := e.retryCounts[stepID]; ok {
		next := count + 1
		e.retryCounts[stepID] = next
		return next
	}
	e.retryCounts[stepID] = 0
	return 0
}

// ---------------------------------------------------------------------------
// Execution Engine Hooks (overridable by subclasses)
// ---------------------------------------------------------------------------

// IsNestedWorkflowStep checks if a step is a nested workflow.
// Override in subclasses to detect platform-specific workflow types.
func (e *DefaultExecutionEngine) IsNestedWorkflowStep(_ *Step) bool {
	return false
}

// ExecuteSleepDuration sleeps for the given duration in milliseconds.
// Override to use platform-specific sleep primitives.
func (e *DefaultExecutionEngine) ExecuteSleepDuration(duration int64, _ string, _ string) error {
	if duration <= 0 {
		return nil
	}
	time.Sleep(time.Duration(duration) * time.Millisecond)
	return nil
}

// ExecuteSleepUntilDate sleeps until a specific date.
// Override to use platform-specific sleep primitives.
func (e *DefaultExecutionEngine) ExecuteSleepUntilDate(date time.Time, _ string, _ string) error {
	d := time.Until(date)
	if d <= 0 {
		return nil
	}
	time.Sleep(d)
	return nil
}

// WrapDurableOperation wraps an operation for durability.
// Override to add platform-specific durability (e.g., Inngest memoization).
func (e *DefaultExecutionEngine) WrapDurableOperation(_ string, fn func() (any, error)) (any, error) {
	return fn()
}

// GetEngineContext returns engine-specific context for step execution.
// Override to provide platform-specific engine primitives.
func (e *DefaultExecutionEngine) GetEngineContext() map[string]any {
	return map[string]any{}
}

// EvaluateCondition evaluates a single condition for conditional execution.
// Override to add platform-specific durability.
func (e *DefaultExecutionEngine) EvaluateCondition(
	conditionFn ConditionFunction,
	index int,
	condContext *ExecuteFunctionParams,
	_ string,
) (*int, error) {
	result, err := conditionFn(condContext)
	if err != nil {
		return nil, err
	}
	if result {
		return &index, nil
	}
	return nil, nil
}

// OnStepExecutionStart handles step execution start - emit events and return start timestamp.
// Override to add platform-specific durability.
func (e *DefaultExecutionEngine) OnStepExecutionStart(params StepExecutionStartParams) (int64, error) {
	startedAt := time.Now().UnixMilli()
	if params.PubSub != nil {
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.ExecutionContext.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.ExecutionContext.RunID,
			Data: map[string]any{
				"type": "workflow-step-start",
				"payload": map[string]any{
					"id":         params.Step.ID,
					"stepCallId": params.StepCallID,
				},
			},
		})
	}
	return startedAt, nil
}

// ExecuteWorkflowStep executes a nested workflow step.
// Default: returns nil to use standard execution.
func (e *DefaultExecutionEngine) ExecuteWorkflowStep(_ any) (*StepResult, error) {
	return nil, nil
}

// ExecuteStepWithRetry executes a step with retry logic.
// Default engine: internal retry loop.
type defaultStepRetryResult struct {
	OK     bool
	Result any
	Error  *defaultStepRetryError
}

type defaultStepRetryError struct {
	Status   string
	Error    error
	EndedAt  int64
	Tripwire *StepTripwireInfo
}

func (e *DefaultExecutionEngine) ExecuteStepWithRetry(
	stepID string,
	runStep func() (any, error),
	retries int,
	delay int,
) (*defaultStepRetryResult, error) {
	for i := 0; i < retries+1; i++ {
		if i > 0 && delay > 0 {
			time.Sleep(time.Duration(delay) * time.Millisecond)
		}
		result, err := runStep()
		if err == nil {
			return &defaultStepRetryResult{OK: true, Result: result}, nil
		}
		if i == retries {
			// Retries exhausted
			if e.log != nil {
				e.log.Error(fmt.Sprintf("Error executing step %s: %v", stepID, err))
			}
			return &defaultStepRetryResult{
				OK: false,
				Error: &defaultStepRetryError{
					Status:  "failed",
					Error:   err,
					EndedAt: time.Now().UnixMilli(),
				},
			}, nil
		}
	}
	return &defaultStepRetryResult{
		OK:    false,
		Error: &defaultStepRetryError{Status: "failed", Error: fmt.Errorf("unknown error"), EndedAt: time.Now().UnixMilli()},
	}, nil
}

// RequiresDurableContextSerialization returns whether this engine requires context serialization.
// Default engine passes by reference (no serialization needed).
func (e *DefaultExecutionEngine) RequiresDurableContextSerialization() bool {
	return false
}

// SerializeRequestContext serializes a RequestContext to a plain map.
func (e *DefaultExecutionEngine) SerializeRequestContext(rc *requestcontext.RequestContext) map[string]any {
	if rc == nil {
		return map[string]any{}
	}
	return rc.Entries()
}

// BuildMutableContext builds MutableContext from current execution state.
func (e *DefaultExecutionEngine) BuildMutableContext(ec *ExecutionContext) MutableContext {
	return MutableContext{
		State:          ec.State,
		SuspendedPaths: ec.SuspendedPaths,
		ResumeLabels:   ec.ResumeLabels,
	}
}

// ApplyMutableContext applies mutable context changes back to the execution context.
func (e *DefaultExecutionEngine) ApplyMutableContext(ec *ExecutionContext, mc MutableContext) {
	for k, v := range mc.State {
		ec.State[k] = v
	}
	for k, v := range mc.SuspendedPaths {
		ec.SuspendedPaths[k] = v
	}
	for k, v := range mc.ResumeLabels {
		ec.ResumeLabels[k] = v
	}
}

// GetStepOutput returns the output of a step from step results.
func (e *DefaultExecutionEngine) GetStepOutput(stepResults map[string]any, step *StepFlowEntry) any {
	if step == nil {
		if input, ok := stepResults["input"]; ok {
			return input
		}
		return nil
	}

	switch step.Type {
	case StepFlowEntryTypeStep:
		if step.Step != nil {
			if sr, ok := stepResults[step.Step.ID]; ok {
				if m, ok := sr.(map[string]any); ok {
					return m["output"]
				}
				if r, ok := sr.(StepResult); ok {
					return r.Output
				}
			}
		}
	case StepFlowEntryTypeSleep, StepFlowEntryTypeSleepUntil:
		if sr, ok := stepResults[step.ID]; ok {
			if m, ok := sr.(map[string]any); ok {
				return m["output"]
			}
			if r, ok := sr.(StepResult); ok {
				return r.Output
			}
		}
	case StepFlowEntryTypeParallel, StepFlowEntryTypeConditional:
		result := make(map[string]any)
		for _, entry := range step.Steps {
			if entry.Step != nil {
				if sr, ok := stepResults[entry.Step.ID]; ok {
					if m, ok := sr.(map[string]any); ok {
						result[entry.Step.ID] = m["output"]
					} else if r, ok := sr.(StepResult); ok {
						result[entry.Step.ID] = r.Output
					}
				}
			}
		}
		return result
	case StepFlowEntryTypeLoop, StepFlowEntryTypeForeach:
		if step.Step != nil {
			if sr, ok := stepResults[step.Step.ID]; ok {
				if m, ok := sr.(map[string]any); ok {
					return m["output"]
				}
				if r, ok := sr.(StepResult); ok {
					return r.Output
				}
			}
		}
	}

	return nil
}

// FormatResultError formats an error for the workflow result.
func (e *DefaultExecutionEngine) FormatResultError(err error, lastOutput StepResult) *mastraerror.SerializedError {
	errToUse := err
	if errToUse == nil && lastOutput.Error != nil {
		errToUse = lastOutput.Error
	}
	if errToUse == nil {
		errToUse = fmt.Errorf("unknown workflow error")
	}
	se := mastraerror.Serialize(errToUse)
	return &se
}

// ---------------------------------------------------------------------------
// fmtReturnValue
// ---------------------------------------------------------------------------

// fmtReturnValue formats the workflow return value.
// TS equivalent: protected async fmtReturnValue<TOutput>(...)
func (e *DefaultExecutionEngine) fmtReturnValue(
	_ events.PubSub,
	stepResults map[string]StepResult,
	lastOutput StepResult,
	err error,
	stepExecutionPath []string,
) *FormattedWorkflowResult {
	// Strip nestedRunId from metadata
	cleanStepResults := make(map[string]StepResult)
	for stepID, sr := range stepResults {
		if sr.Metadata != nil {
			cleanMeta := make(StepMetadata)
			for mk, mv := range sr.Metadata {
				if mk != "nestedRunId" {
					cleanMeta[mk] = mv
				}
			}
			cleanSR := sr
			if len(cleanMeta) > 0 {
				cleanSR.Metadata = cleanMeta
			} else {
				cleanSR.Metadata = nil
			}
			cleanStepResults[stepID] = cleanSR
		} else {
			cleanStepResults[stepID] = sr
		}
	}

	inputResult := cleanStepResults["input"]
	base := &FormattedWorkflowResult{
		Status: lastOutput.Status,
		Steps:  cleanStepResults,
		Input:  &inputResult,
	}

	if stepExecutionPath != nil {
		base.StepExecutionPath = stepExecutionPath
	}

	switch lastOutput.Status {
	case StepStatusSuccess:
		base.Result = lastOutput.Output
	case StepStatusFailed:
		if lastOutput.Tripwire != nil {
			base.Status = "tripwire"
			base.Tripwire = lastOutput.Tripwire
		} else {
			serialized := e.FormatResultError(err, lastOutput)
			base.Error = serialized
		}
	case StepStatusSuspended:
		suspendPayload := make(map[string]any)
		suspended := make([][]string, 0)
		for stepID, sr := range stepResults {
			if sr.Status == StepStatusSuspended {
				if sp, ok := sr.SuspendPayload.(map[string]any); ok {
					filtered := make(map[string]any)
					for k, v := range sp {
						if k != "__workflow_meta" {
							filtered[k] = v
						}
					}
					suspendPayload[stepID] = filtered
				} else {
					suspendPayload[stepID] = sr.SuspendPayload
				}
				suspended = append(suspended, []string{stepID})
			}
		}
		base.Suspended = suspended
		base.SuspendPayload = suspendPayload
	}

	return base
}

// ---------------------------------------------------------------------------
// Execute (main entry point)
// ---------------------------------------------------------------------------

// Execute executes a workflow run with the provided execution graph and input.
// TS equivalent: async execute<TState, TInput, TOutput>(params)
func (e *DefaultExecutionEngine) Execute(params ExecuteParams) (any, error) {
	graph := params.Graph
	steps := graph.Steps

	// Clear retry counts
	e.retryCounts = make(map[string]int)

	if len(steps) == 0 {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "WORKFLOW_EXECUTE_EMPTY_GRAPH",
			Text:     "Workflow must have at least one step",
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategoryUser,
		})
	}

	startIdx := 0
	timeTravel := params.TimeTravel
	restart := params.Restart
	resume := params.Resume

	if timeTravel != nil && len(timeTravel.ExecutionPath) > 0 {
		startIdx = timeTravel.ExecutionPath[0]
		timeTravel.ExecutionPath = timeTravel.ExecutionPath[1:]
	} else if restart != nil && len(restart.ActivePaths) > 0 {
		startIdx = restart.ActivePaths[0]
		restart.ActivePaths = restart.ActivePaths[1:]
	} else if resume != nil && len(resume.ResumePath) > 0 {
		startIdx = resume.ResumePath[0]
		resume.ResumePath = resume.ResumePath[1:]
	}

	// Initialize step results
	stepResults := make(map[string]StepResult)
	if timeTravel != nil {
		stepResults = timeTravel.StepResults
	} else if restart != nil {
		stepResults = restart.StepResults
	} else if resume != nil {
		stepResults = resume.StepResults
	} else {
		stepResults["input"] = StepResult{
			Status: StepStatusSuccess,
			Output: params.Input,
		}
	}

	stepExecutionPath := make([]string, 0)
	if timeTravel != nil && timeTravel.StepExecutionPath != nil {
		stepExecutionPath = timeTravel.StepExecutionPath
	} else if restart != nil && restart.StepExecutionPath != nil {
		stepExecutionPath = restart.StepExecutionPath
	} else if resume != nil && resume.StepExecutionPath != nil {
		stepExecutionPath = resume.StepExecutionPath
	}

	var lastOutput *EntryExecutionResult
	var lastState map[string]any
	if timeTravel != nil {
		lastState = timeTravel.State
	} else if restart != nil {
		lastState = restart.State
	} else if params.InitialState != nil {
		if m, ok := params.InitialState.(map[string]any); ok {
			lastState = m
		}
	}
	if lastState == nil {
		lastState = make(map[string]any)
	}
	var lastExecutionContext *ExecutionContext
	attempts := 0
	delay := 0
	if params.RetryConfig != nil {
		attempts = params.RetryConfig.Attempts
		delay = params.RetryConfig.Delay
	}

	for i := startIdx; i < len(steps); i++ {
		entry := steps[i]

		executionContext := &ExecutionContext{
			WorkflowID:        params.WorkflowID,
			RunID:             params.RunID,
			ExecutionPath:     []int{i},
			StepExecutionPath: stepExecutionPath,
			ActiveStepsPath:   make(map[string][]int),
			SuspendedPaths:    make(map[string][]int),
			ResumeLabels:      make(map[string]ResumeLabel),
			RetryConfig:       RetryConfig{Attempts: attempts, Delay: delay},
			Format:            params.Format,
			State:             lastState,
			TracingIDs:        params.TracingIDs,
		}
		lastExecutionContext = executionContext

		entryResult, err := e.executeEntry(
			params.WorkflowID,
			params.RunID,
			params.ResourceID,
			entry,
			func() StepFlowEntry {
				if i > 0 {
					return steps[i-1]
				}
				return StepFlowEntry{}
			}(),
			params.SerializedStepGraph,
			stepResults,
			resume,
			timeTravel,
			restart,
			executionContext,
			params.PubSub,
			params.AbortCtx,
			params.AbortCancel,
			params.RequestContext,
			params.OutputWriter,
			params.DisableScorers,
			params.PerStep,
		)
		if err != nil {
			return nil, err
		}

		lastOutput = entryResult
		e.ApplyMutableContext(executionContext, lastOutput.MutableContext)
		lastState = lastOutput.MutableContext.State

		// If step result is not success, stop
		if lastOutput.Result.Status != StepStatusSuccess {
			if lastOutput.Result.Status == "bailed" {
				lastOutput.Result.Status = StepStatusSuccess
			}

			result := e.fmtReturnValue(params.PubSub, stepResults, lastOutput.Result, nil, stepExecutionPath)

			if lastOutput.Result.Status != StepStatusPaused {
				_ = e.InvokeLifecycleCallbacks(LifecycleCallbackParams{
					Status:            WorkflowRunStatus(result.Status),
					Result:            result.Result,
					Steps:             result.Steps,
					Tripwire:          result.Tripwire,
					RunID:             params.RunID,
					WorkflowID:        params.WorkflowID,
					ResourceID:        params.ResourceID,
					Input:             params.Input,
					RequestContext:    params.RequestContext,
					State:             lastState,
					StepExecutionPath: stepExecutionPath,
				})
			}

			finalResult := map[string]any{
				"status": string(result.Status),
				"steps":  result.Steps,
			}
			if result.Result != nil {
				finalResult["result"] = result.Result
			}
			if result.Error != nil {
				finalResult["error"] = result.Error
			}
			if result.Suspended != nil {
				finalResult["suspended"] = result.Suspended
			}
			if result.SuspendPayload != nil {
				finalResult["suspendPayload"] = result.SuspendPayload
			}
			if result.Tripwire != nil {
				finalResult["tripwire"] = result.Tripwire
			}
			if result.StepExecutionPath != nil {
				finalResult["stepExecutionPath"] = result.StepExecutionPath
			}

			if params.OutputOptions != nil && params.OutputOptions.IncludeState {
				finalResult["state"] = lastState
			}
			if lastOutput.Result.Status == StepStatusSuspended && params.OutputOptions != nil && params.OutputOptions.IncludeResumeLabels {
				finalResult["resumeLabels"] = lastOutput.MutableContext.ResumeLabels
			}

			return finalResult, nil
		}

		if params.PerStep {
			result := e.fmtReturnValue(params.PubSub, stepResults, lastOutput.Result, nil, stepExecutionPath)
			finalResult := map[string]any{
				"status": "paused",
				"steps":  result.Steps,
			}
			if result.StepExecutionPath != nil {
				finalResult["stepExecutionPath"] = result.StepExecutionPath
			}
			if params.OutputOptions != nil && params.OutputOptions.IncludeState {
				finalResult["state"] = lastState
			}
			return finalResult, nil
		}
	}

	if lastOutput == nil {
		return nil, fmt.Errorf("no steps executed")
	}

	// All steps successful
	result := e.fmtReturnValue(params.PubSub, stepResults, lastOutput.Result, nil, stepExecutionPath)

	_ = e.InvokeLifecycleCallbacks(LifecycleCallbackParams{
		Status:            WorkflowRunStatus(result.Status),
		Result:            result.Result,
		Steps:             result.Steps,
		Tripwire:          result.Tripwire,
		RunID:             params.RunID,
		WorkflowID:        params.WorkflowID,
		ResourceID:        params.ResourceID,
		Input:             params.Input,
		RequestContext:    params.RequestContext,
		State:             lastState,
		StepExecutionPath: stepExecutionPath,
	})

	finalResult := map[string]any{
		"status": string(result.Status),
		"steps":  result.Steps,
	}
	if result.Result != nil {
		finalResult["result"] = result.Result
	}
	if params.OutputOptions != nil && params.OutputOptions.IncludeState {
		finalResult["state"] = lastState
	}

	_ = lastExecutionContext // suppress unused warning

	return finalResult, nil
}

// ---------------------------------------------------------------------------
// Internal executeEntry (delegates to handlers)
// ---------------------------------------------------------------------------

func (e *DefaultExecutionEngine) executeEntry(
	workflowID, runID, resourceID string,
	entry, prevStep StepFlowEntry,
	serializedStepGraph []SerializedStepFlowEntry,
	stepResults map[string]StepResult,
	resume *ResumeExecuteParams,
	timeTravel *TimeTravelExecutionParams,
	restart *RestartExecutionParams,
	executionContext *ExecutionContext,
	pubsub events.PubSub,
	abortCtx context.Context,
	abortCancel context.CancelFunc,
	requestContext *requestcontext.RequestContext,
	outputWriter OutputWriter,
	disableScorers bool,
	perStep bool,
) (*EntryExecutionResult, error) {
	// This is a simplified inline implementation that mirrors the handler pattern.
	// In the TS version, this delegates to executeEntryHandler.
	// Here we implement the core logic directly.

	prevOutput := e.GetStepOutput(stepResultsToAnyMap(stepResults), &prevStep)

	switch entry.Type {
	case StepFlowEntryTypeStep:
		if entry.Step == nil {
			return nil, fmt.Errorf("step entry has no step")
		}

		isResumed := false
		if resume != nil {
			for _, s := range resume.Steps {
				if s == entry.Step.ID {
					isResumed = true
					break
				}
			}
		}
		if !isResumed && executionContext.StepExecutionPath != nil {
			executionContext.StepExecutionPath = append(executionContext.StepExecutionPath, entry.Step.ID)
		}

		stepExecResult, err := e.executeStepInternal(
			workflowID, runID, resourceID, entry.Step,
			stepResults, executionContext, restart, timeTravel,
			resume, prevOutput, pubsub, abortCtx, abortCancel,
			requestContext, false, outputWriter, disableScorers,
			serializedStepGraph, 0, perStep,
		)
		if err != nil {
			return nil, err
		}

		e.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
		for k, v := range stepExecResult.StepResults {
			stepResults[k] = v
		}

		return &EntryExecutionResult{
			Result:         stepExecResult.Result,
			StepResults:    stepResults,
			MutableContext: e.BuildMutableContext(executionContext),
			RequestContext: stepExecResult.RequestContext,
		}, nil

	// For other entry types, we delegate to the specific execution methods
	// This is a simplified version - in production, this would call the
	// full handler implementations.
	default:
		return nil, fmt.Errorf("entry type %s execution not yet implemented in simplified default engine; use handler-based engine", entry.Type)
	}
}

// executeStepInternal is the internal step execution that mirrors handlers/step.go logic.
func (e *DefaultExecutionEngine) executeStepInternal(
	workflowID, runID, resourceID string,
	step *Step,
	stepResults map[string]StepResult,
	executionContext *ExecutionContext,
	restart *RestartExecutionParams,
	timeTravel *TimeTravelExecutionParams,
	resume *ResumeExecuteParams,
	prevOutput any,
	pubsub events.PubSub,
	abortCtx context.Context,
	abortCancel context.CancelFunc,
	rc *requestcontext.RequestContext,
	skipEmits bool,
	outputWriter OutputWriter,
	disableScorers bool,
	serializedStepGraph []SerializedStepFlowEntry,
	iterationCount int,
	perStep bool,
) (*StepExecutionResult, error) {
	validateInputs := true
	if e.Options != nil {
		validateInputs = e.Options.ValidateInputs
	}

	// Validate input
	inputResult := ValidateStepInput(prevOutput, step, validateInputs)
	inputData := inputResult.InputData
	validationError := inputResult.ValidationError

	// Determine resume data
	var resumeDataToUse any
	if resume != nil && len(resume.Steps) > 0 && resume.Steps[0] == step.ID {
		resumeDataToUse = resume.ResumePayload
	}

	// Build step info
	now := time.Now().UnixMilli()
	stepInfo := StepResult{Status: StepStatusRunning}
	if existing, ok := stepResults[step.ID]; ok {
		stepInfo = existing
		stepInfo.Status = StepStatusRunning
	}
	if resumeDataToUse != nil {
		stepInfo.ResumePayload = resumeDataToUse
		stepInfo.ResumedAt = &now
	} else {
		stepInfo.Payload = inputData
		stepInfo.StartedAt = now
	}

	executionContext.ActiveStepsPath[step.ID] = executionContext.ExecutionPath

	// Execute step with retry
	retries := 0
	if step.Retries != nil {
		retries = *step.Retries
	} else {
		retries = executionContext.RetryConfig.Attempts
	}
	delay := executionContext.RetryConfig.Delay

	retryResult, err := e.ExecuteStepWithRetry(
		fmt.Sprintf("workflow.%s.step.%s", workflowID, step.ID),
		func() (any, error) {
			if validationError != nil {
				return nil, validationError
			}

			retryCount := e.GetOrGenerateRetryCount(step.ID)

			var suspended *stepSuspendResult
			var bailed *stepBailResult

			if step.Execute == nil {
				return nil, fmt.Errorf("step %s has no execute function", step.ID)
			}

			execParams := &ExecuteFunctionParams{
				RunID:          runID,
				ResourceID:     resourceID,
				WorkflowID:     workflowID,
				Mastra:         e.mastra,
				RequestContext: rc,
				InputData:      inputData,
				State:          executionContext.State,
				SetState: func(state any) error {
					if m, ok := state.(map[string]any); ok {
						executionContext.State = m
					}
					return nil
				},
				RetryCount:  retryCount,
				ResumeData:  resumeDataToUse,
				GetInitData: func() any {
					if r, ok := stepResults["input"]; ok {
						return r.Output
					}
					return nil
				},
				GetStepResult: func(s any) any {
					return GetStepResult(stepResults, s)
				},
				Suspend: func(suspendPayload any, suspendOptions *SuspendOptions) error {
					executionContext.SuspendedPaths[step.ID] = executionContext.ExecutionPath
					if suspendOptions != nil {
						for _, label := range suspendOptions.ResumeLabel {
							executionContext.ResumeLabels[label] = ResumeLabel{
								StepID:       step.ID,
								ForeachIndex: executionContext.ForeachIndex,
							}
						}
					}
					suspended = &stepSuspendResult{Payload: suspendPayload}
					return nil
				},
				Bail: func(result any) {
					bailed = &stepBailResult{Payload: result}
				},
				Abort: func() {
					if abortCancel != nil {
						abortCancel()
					}
				},
				PubSub:          pubsub,
				StreamFormat:    executionContext.Format,
				Engine:          e.GetEngineContext(),
				AbortCtx:        abortCtx,
				OutputWriter:    outputWriter,
				ValidateSchemas: validateInputs,
			}

			output, execErr := step.Execute(execParams)
			if execErr != nil {
				return nil, execErr
			}

			return &internalStepOutput{
				Output:    output,
				Suspended: suspended,
				Bailed:    bailed,
			}, nil
		},
		retries,
		delay,
	)
	if err != nil {
		return nil, err
	}

	var execResults StepResult
	if !retryResult.OK {
		execResults = StepResult{
			Status:  StepStatusFailed,
			Error:   retryResult.Error.Error,
			EndedAt: retryResult.Error.EndedAt,
		}
	} else {
		durableResult, ok := retryResult.Result.(*internalStepOutput)
		if !ok {
			return nil, fmt.Errorf("unexpected step result type")
		}

		if durableResult.Suspended != nil {
			suspendedAt := time.Now().UnixMilli()
			execResults = StepResult{
				Status:         StepStatusSuspended,
				SuspendPayload: durableResult.Suspended.Payload,
				SuspendedAt:    &suspendedAt,
			}
			if durableResult.Output != nil {
				execResults.SuspendOutput = durableResult.Output
			}
		} else if durableResult.Bailed != nil {
			endedAt := time.Now().UnixMilli()
			execResults = StepResult{
				Status:  "bailed",
				Output:  durableResult.Bailed.Payload,
				EndedAt: endedAt,
			}
		} else {
			endedAt := time.Now().UnixMilli()
			execResults = StepResult{
				Status:  StepStatusSuccess,
				Output:  durableResult.Output,
				EndedAt: endedAt,
			}
		}
	}

	delete(executionContext.ActiveStepsPath, step.ID)

	finalResult := mergeStepResults(stepInfo, execResults)

	return &StepExecutionResult{
		Result:      finalResult,
		StepResults: map[string]StepResult{step.ID: finalResult},
		MutableContext: MutableContext{
			State:          executionContext.State,
			SuspendedPaths: executionContext.SuspendedPaths,
			ResumeLabels:   executionContext.ResumeLabels,
		},
		RequestContext: e.SerializeRequestContext(rc),
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helper types
// ---------------------------------------------------------------------------

type stepSuspendResult struct {
	Payload any
}

type stepBailResult struct {
	Payload any
}

type internalStepOutput struct {
	Output    any
	Suspended *stepSuspendResult
	Bailed    *stepBailResult
}

func mergeStepResults(base, override StepResult) StepResult {
	result := base
	result.Status = override.Status
	if override.Output != nil {
		result.Output = override.Output
	}
	if override.Error != nil {
		result.Error = override.Error
	}
	if override.SuspendPayload != nil {
		result.SuspendPayload = override.SuspendPayload
	}
	if override.SuspendOutput != nil {
		result.SuspendOutput = override.SuspendOutput
	}
	if override.EndedAt != 0 {
		result.EndedAt = override.EndedAt
	}
	if override.SuspendedAt != nil {
		result.SuspendedAt = override.SuspendedAt
	}
	if override.Tripwire != nil {
		result.Tripwire = override.Tripwire
	}
	return result
}

func stepResultsToAnyMap(m map[string]StepResult) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
