// Ported from: packages/core/src/workflows/evented/step-executor.ts
package evented

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storageworkflows "github.com/brainlet/brainkit/agent-kit/core/storage/domains/workflows"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Interfaces bridging to real storage types
// ---------------------------------------------------------------------------

// Mastra is the narrow interface for the Mastra orchestrator.
// core.Mastra satisfies this.
type Mastra interface {
	GetLogger() logger.IMastraLogger
	PubSub() PubSub
	GetWorkflowsStore() WorkflowsStore
}

// PubSub is a stub for the PubSub interface.
// TODO: Replace with actual PubSub from events package when ported.
type PubSub interface {
	Publish(topic string, event any) error
	Subscribe(topic string, handler func(event any, ack func() error) error) error
	Unsubscribe(topic string, handler func(event any, ack func() error) error) error
}

// WorkflowsStore is the real workflows storage interface from
// storage/domains/workflows. No cycle exists: evented → storage is safe.
type WorkflowsStore = storageworkflows.WorkflowsStorage

// WorkflowRunState is a typed struct for the in-memory workflow run state
// used by the evented engine. It has looser field types than wf.WorkflowRunState
// (e.g. []any for ActivePaths instead of []int) because the evented engine
// stores heterogeneous path data. Converted to map[string]any for storage via
// WorkflowRunStateToMap.
type WorkflowRunState struct {
	ActivePaths         []any                        `json:"activePaths"`
	SuspendedPaths      map[string][]int             `json:"suspendedPaths"`
	ResumeLabels        map[string]any               `json:"resumeLabels"`
	WaitingPaths        map[string][]int             `json:"waitingPaths"`
	ActiveStepsPath     map[string]any               `json:"activeStepsPath"`
	SerializedStepGraph []wf.SerializedStepFlowEntry `json:"serializedStepGraph"`
	Timestamp           int64                        `json:"timestamp"`
	RunID               string                       `json:"runId"`
	Context             map[string]any               `json:"context"`
	Status              string                       `json:"status"`
	Value               any                          `json:"value"`
	RequestContext      map[string]any               `json:"requestContext"`
}

// WorkflowRun is the real workflow run record from storage/domains/workflows.
type WorkflowRun = storageworkflows.WorkflowRun

// Storage param type aliases pointing to real storage types.
type PersistSnapshotParams = storageworkflows.PersistWorkflowSnapshotArgs
type LoadSnapshotParams = storageworkflows.LoadWorkflowSnapshotArgs
type UpdateResultsParams = storageworkflows.UpdateWorkflowResultsArgs
type UpdateStateParams = storageworkflows.UpdateWorkflowStateArgs
type GetRunParams = storageworkflows.GetWorkflowRunByIDArgs

// WorkflowRunStateToMap converts the typed evented WorkflowRunState into
// the untyped map[string]any expected by storageworkflows.WorkflowRunState.
func WorkflowRunStateToMap(state WorkflowRunState) storageworkflows.WorkflowRunState {
	m := map[string]any{
		"runId":               state.RunID,
		"status":              state.Status,
		"timestamp":           state.Timestamp,
		"activePaths":         state.ActivePaths,
		"suspendedPaths":      state.SuspendedPaths,
		"resumeLabels":        state.ResumeLabels,
		"waitingPaths":        state.WaitingPaths,
		"activeStepsPath":     state.ActiveStepsPath,
		"serializedStepGraph": state.SerializedStepGraph,
		"context":             state.Context,
		"value":               state.Value,
		"requestContext":      state.RequestContext,
	}
	return m
}

// ---------------------------------------------------------------------------
// StepExecutor
// ---------------------------------------------------------------------------

// StepExecutor handles the execution of individual workflow steps
// in the evented workflow engine.
// TS equivalent: export class StepExecutor extends MastraBase
type StepExecutor struct {
	mastra Mastra
	log    logger.IMastraLogger
}

// NewStepExecutor creates a new StepExecutor.
func NewStepExecutor(mastra Mastra) *StepExecutor {
	var log logger.IMastraLogger
	if mastra != nil {
		log = mastra.GetLogger()
	}
	return &StepExecutor{
		mastra: mastra,
		log:    log,
	}
}

// RegisterMastra registers the mastra instance with this executor.
// TS equivalent: __registerMastra(mastra: Mastra)
func (se *StepExecutor) RegisterMastra(mastra Mastra) {
	se.mastra = mastra
	if mastra != nil {
		log := mastra.GetLogger()
		if log != nil {
			se.log = log
		}
	}
}

// createOutputWriter creates an output writer function that publishes chunks
// to the workflow event stream.
// TS equivalent: private createOutputWriter(runId: string)
func (se *StepExecutor) createOutputWriter(runID string) func(chunk any) error {
	return func(chunk any) error {
		pubsub := se.mastra.PubSub()
		if pubsub == nil {
			return nil
		}
		err := pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", runID), map[string]any{
			"type":  "watch",
			"runId": runID,
			"data":  chunk,
		})
		if err != nil {
			// Non-critical: streaming events are observational
			if se.log != nil {
				se.log.Debug(fmt.Sprintf("Failed to publish workflow watch event: runId=%s, error=%v", runID, err))
			}
		}
		return nil
	}
}

// ExecuteParams holds parameters for step execution.
type ExecuteParams struct {
	WorkflowID      string
	Step            *wf.Step
	RunID           string
	Input           any
	ResumeData      any
	StepResults     map[string]wf.StepResult
	State           map[string]any
	RequestContext  *requestcontext.RequestContext
	RetryCount      int
	ForeachIdx      *int
	ValidateInputs  bool
	AbortCtx        context.Context
	AbortCancel     context.CancelFunc
	PerStep         bool
}

// Execute executes a single step.
// TS equivalent: async execute(params: {...}): Promise<StepResult>
func (se *StepExecutor) Execute(params ExecuteParams) wf.StepResult {
	step := params.Step
	stepResults := params.StepResults
	retryCount := params.RetryCount

	// Use provided abort context or create a new one
	abortCtx := params.AbortCtx
	abortCancel := params.AbortCancel
	if abortCtx == nil {
		abortCtx, abortCancel = context.WithCancel(context.Background())
		_ = abortCancel
	}

	startedAt := time.Now().UnixMilli()

	// Determine input data
	var inputData any
	if params.ForeachIdx != nil {
		// For foreach, extract the specific item from the input array
		if arr, ok := params.Input.([]any); ok && *params.ForeachIdx < len(arr) {
			inputData = arr[*params.ForeachIdx]
		}
	} else {
		inputData = params.Input
	}

	// Validate step input
	validateInputs := params.ValidateInputs
	validationResult := wf.ValidateStepInput(inputData, step, validateInputs)
	validationError := validationResult.ValidationError

	// Build step info
	stepInfo := map[string]any{
		"startedAt": startedAt,
		"payload":   inputData,
	}

	// Merge existing step result info
	if existing, ok := stepResults[step.ID]; ok {
		existingMap := map[string]any{
			"status":         existing.Status,
			"output":         existing.Output,
			"payload":        existing.Payload,
			"resumePayload":  existing.ResumePayload,
			"suspendPayload": existing.SuspendPayload,
			"suspendOutput":  existing.SuspendOutput,
			"startedAt":      existing.StartedAt,
			"endedAt":        existing.EndedAt,
			"suspendedAt":    existing.SuspendedAt,
			"resumedAt":      existing.ResumedAt,
		}
		for k, v := range existingMap {
			if _, exists := stepInfo[k]; !exists {
				stepInfo[k] = v
			}
		}
	}

	// For foreach, use full input as payload
	if params.ForeachIdx != nil {
		stepInfo["payload"] = params.Input
	}
	if stepInfo["payload"] == nil {
		stepInfo["payload"] = map[string]any{}
	}

	// Handle resume data
	if params.ResumeData != nil {
		stepInfo["resumePayload"] = params.ResumeData
		stepInfo["resumedAt"] = time.Now().UnixMilli()
		// Strip __workflow_meta from suspendPayload when step is resumed
		if sp, ok := stepInfo["suspendPayload"].(map[string]any); ok {
			if _, hasMeta := sp["__workflow_meta"]; hasMeta {
				cleaned := make(map[string]any)
				for k, v := range sp {
					if k != "__workflow_meta" {
						cleaned[k] = v
					}
				}
				stepInfo["suspendPayload"] = cleaned
			}
		}
	}

	// Extract suspend data if this step was previously suspended
	var suspendDataToUse any
	if existing, ok := stepResults[step.ID]; ok && existing.Status == "suspended" {
		suspendDataToUse = existing.SuspendPayload
	}
	// Filter out internal workflow metadata
	if sdMap, ok := suspendDataToUse.(map[string]any); ok {
		if _, hasMeta := sdMap["__workflow_meta"]; hasMeta {
			cleaned := make(map[string]any)
			for k, v := range sdMap {
				if k != "__workflow_meta" {
					cleaned[k] = v
				}
			}
			suspendDataToUse = cleaned
		}
	}

	// Track state updates
	var stateUpdate map[string]any

	if validationError != nil {
		endedAt := time.Now().UnixMilli()
		return wf.StepResult{
			Status:  "failed",
			Error:   validationError,
			Payload: stepInfo["payload"],
			StartedAt: startedAt,
			EndedAt: endedAt,
		}
	}

	// Build execution context
	execCtx := &wf.ExecuteFunctionParams{
		WorkflowID:     params.WorkflowID,
		RunID:          params.RunID,
		Mastra:         nil, // TODO: pass mastra
		RequestContext: params.RequestContext,
		InputData:      inputData,
		State:          params.State,
		SetState: func(state any) error {
			newState, ok := state.(map[string]any)
			if !ok {
				return nil
			}
			if stateUpdate == nil {
				stateUpdate = make(map[string]any)
				for k, v := range params.State {
					stateUpdate[k] = v
				}
			}
			for k, v := range newState {
				stateUpdate[k] = v
			}
			return nil
		},
		RetryCount:  retryCount,
		ResumeData:  params.ResumeData,
		SuspendData: suspendDataToUse,
		GetInitData: func() any {
			if v, ok := stepResults["input"]; ok {
				return v
			}
			return nil
		},
		GetStepResult: func(step any) any {
			return wf.GetStepResult(stepResults, step)
		},
		AbortCtx: abortCtx,
	}

	// Track suspend and bail
	var suspended *suspendedData
	var bailed *bailedData

	execCtx.Suspend = func(suspendPayload any, opts *wf.SuspendOptions) error {
		// Build resume labels if provided
		resumeLabels := map[string]any{}
		if opts != nil && len(opts.ResumeLabel) > 0 {
			for _, label := range opts.ResumeLabel {
				entry := map[string]any{
					"stepId": step.ID,
				}
				if params.ForeachIdx != nil {
					entry["foreachIndex"] = *params.ForeachIdx
				}
				resumeLabels[label] = entry
			}
		}

		workflowMeta := map[string]any{
			"runId": params.RunID,
			"path":  []string{step.ID},
		}
		if params.ForeachIdx != nil {
			workflowMeta["foreachIndex"] = *params.ForeachIdx
		}
		if len(resumeLabels) > 0 {
			workflowMeta["resumeLabels"] = resumeLabels
		}

		payload := map[string]any{
			"__workflow_meta": workflowMeta,
		}
		if spMap, ok := suspendPayload.(map[string]any); ok {
			for k, v := range spMap {
				payload[k] = v
			}
		}

		suspended = &suspendedData{payload: payload}
		return nil
	}

	execCtx.Bail = func(result any) {
		bailed = &bailedData{payload: result}
	}

	execCtx.Abort = func() {
		if abortCancel != nil {
			abortCancel()
		}
	}

	// Execute the step
	var stepOutput any
	var execErr error

	if step.Execute != nil {
		stepOutput, execErr = step.Execute(execCtx)
	}

	if execErr != nil {
		endedAt := time.Now().UnixMilli()

		if se.log != nil {
			se.log.Error(fmt.Sprintf("Error executing step %s: %v", step.ID, execErr))
		}

		result := wf.StepResult{
			Status:    "failed",
			Error:     execErr,
			Payload:   stepInfo["payload"],
			StartedAt: startedAt,
			EndedAt:   endedAt,
		}

		// Preserve TripWire data
		if tw, ok := execErr.(*TripWire); ok {
			result.Tripwire = &wf.StepTripwireInfo{
				Reason:      tw.Message,
				ProcessorID: tw.ProcessorID,
			}
			if tw.Options != nil {
				result.Tripwire.Retry = tw.Options.Retry
				if m, ok := tw.Options.Metadata.(map[string]any); ok {
					result.Tripwire.Metadata = m
				}
			}
		}

		return result
	}

	isNestedWorkflowStep := step.Component == "WORKFLOW"
	nestedWflowStepPaused := isNestedWorkflowStep && params.PerStep

	endedAt := time.Now().UnixMilli()

	// Use stateUpdate if setState was called, otherwise use original state
	finalState := stateUpdate
	if finalState == nil {
		finalState = params.State
	}

	if suspended != nil {
		result := wf.StepResult{
			Status:      "suspended",
			SuspendedAt: &endedAt,
			Payload:     stepInfo["payload"],
			StartedAt:   startedAt,
			Metadata:    wf.StepMetadata{"__state": finalState},
		}
		if stepOutput != nil {
			result.SuspendOutput = stepOutput
		}
		result.SuspendPayload = suspended.payload
		return result
	}

	if bailed != nil {
		return wf.StepResult{
			Status:    "bailed",
			EndedAt:   endedAt,
			Output:    bailed.payload,
			Payload:   stepInfo["payload"],
			StartedAt: startedAt,
			Metadata:  wf.StepMetadata{"__state": finalState},
		}
	}

	if nestedWflowStepPaused {
		return wf.StepResult{
			Status:    "paused",
			Payload:   stepInfo["payload"],
			StartedAt: startedAt,
			Metadata:  wf.StepMetadata{"__state": finalState},
		}
	}

	return wf.StepResult{
		Status:    "success",
		EndedAt:   endedAt,
		Output:    stepOutput,
		Payload:   stepInfo["payload"],
		StartedAt: startedAt,
		Metadata:  wf.StepMetadata{"__state": finalState},
	}
}

type suspendedData struct {
	payload any
}

type bailedData struct {
	payload any
}

// ---------------------------------------------------------------------------
// EvaluateConditions
// ---------------------------------------------------------------------------

// EvaluateConditionsParams holds parameters for evaluating conditions.
type EvaluateConditionsParams struct {
	WorkflowID      string
	Step            *wf.StepFlowEntry
	RunID           string
	Input           any
	ResumeData      any
	StepResults     map[string]wf.StepResult
	State           map[string]any
	RequestContext  *requestcontext.RequestContext
	RetryCount      int
	AbortCtx        context.Context
	AbortCancel     context.CancelFunc
}

// EvaluateConditions evaluates all conditions on a conditional step and returns
// the indices of conditions that evaluated to true.
// TS equivalent: async evaluateConditions(params): Promise<number[]>
func (se *StepExecutor) EvaluateConditions(params EvaluateConditionsParams) ([]int, error) {
	if params.Step == nil || params.Step.Conditions == nil {
		return nil, nil
	}

	abortCtx := params.AbortCtx
	if abortCtx == nil {
		var cancel context.CancelFunc
		abortCtx, cancel = context.WithCancel(context.Background())
		defer cancel()
	}

	results := make([]int, 0)
	for i, condition := range params.Step.Conditions {
		ok, err := se.evaluateCondition(evaluateConditionParams{
			WorkflowID:     params.WorkflowID,
			Condition:      condition,
			RunID:          params.RunID,
			InputData:      params.Input,
			ResumeData:     params.ResumeData,
			StepResults:    params.StepResults,
			State:          params.State,
			RequestContext: params.RequestContext,
			RetryCount:     params.RetryCount,
			AbortCtx:       abortCtx,
			IterationCount: 0,
		})
		if err != nil {
			if se.log != nil {
				se.log.Error(fmt.Sprintf("error evaluating condition: %v", err))
			}
			continue
		}
		if ok {
			results = append(results, i)
		}
	}

	return results, nil
}

type evaluateConditionParams struct {
	WorkflowID     string
	Condition      wf.ConditionFunction
	RunID          string
	InputData      any
	ResumeData     any
	StepResults    map[string]wf.StepResult
	State          map[string]any
	RequestContext *requestcontext.RequestContext
	AbortCtx       context.Context
	RetryCount     int
	IterationCount int
}

// evaluateCondition evaluates a single condition function.
func (se *StepExecutor) evaluateCondition(params evaluateConditionParams) (bool, error) {
	if params.Condition == nil {
		return false, nil
	}

	ctx := &wf.ExecuteFunctionParams{
		WorkflowID:     params.WorkflowID,
		RunID:          params.RunID,
		RequestContext: params.RequestContext,
		InputData:      params.InputData,
		State:          params.State,
		RetryCount:     params.RetryCount,
		ResumeData:     params.ResumeData,
		GetInitData: func() any {
			if v, ok := params.StepResults["input"]; ok {
				return v
			}
			return nil
		},
		GetStepResult: func(step any) any {
			return wf.GetStepResult(params.StepResults, step)
		},
		AbortCtx: params.AbortCtx,
	}

	return params.Condition(ctx)
}

// ---------------------------------------------------------------------------
// ResolveSleep
// ---------------------------------------------------------------------------

// ResolveSleepParams holds parameters for resolving sleep duration.
type ResolveSleepParams struct {
	WorkflowID     string
	Step           *wf.StepFlowEntry
	RunID          string
	Input          any
	ResumeData     any
	StepResults    map[string]wf.StepResult
	State          map[string]any
	RequestContext *requestcontext.RequestContext
	RetryCount     int
	AbortCtx       context.Context
	AbortCancel    context.CancelFunc
}

// ResolveSleep resolves the sleep duration for a sleep step.
// Returns duration in milliseconds.
// TS equivalent: async resolveSleep(params): Promise<number>
func (se *StepExecutor) ResolveSleep(params ResolveSleepParams) int64 {
	if params.Step == nil {
		return 0
	}

	// If a static duration is specified, use it
	if params.Step.Duration != nil && *params.Step.Duration > 0 {
		return *params.Step.Duration
	}

	// If no function, return 0
	if params.Step.Fn == nil {
		return 0
	}

	currentState := params.State
	if currentState == nil {
		// Fall back to stepResults.__state
		if params.StepResults != nil {
			if stateVal, ok := params.StepResults["__state"]; ok {
				if stateMap, ok := stateVal.Output.(map[string]any); ok {
					if inner, ok := stateMap["__state"].(map[string]any); ok {
						currentState = inner
					}
				}
			}
		}
		if currentState == nil {
			currentState = map[string]any{}
		}
	}

	ctx := &wf.ExecuteFunctionParams{
		WorkflowID:     params.WorkflowID,
		RunID:          params.RunID,
		RequestContext: params.RequestContext,
		InputData:      params.Input,
		State:          currentState,
		RetryCount:     params.RetryCount,
		ResumeData:     params.ResumeData,
		GetInitData: func() any {
			if v, ok := params.StepResults["input"]; ok {
				return v
			}
			return nil
		},
		GetStepResult: func(step any) any {
			return wf.GetStepResult(params.StepResults, step)
		},
		AbortCtx: params.AbortCtx,
	}

	durationVal, err := params.Step.Fn(ctx)
	if err != nil {
		if se.log != nil {
			se.log.Error(fmt.Sprintf("error resolving sleep duration: %v", err))
		}
		return 0
	}

	if d, ok := durationVal.(int64); ok {
		return d
	}
	if d, ok := durationVal.(float64); ok {
		return int64(d)
	}
	if d, ok := durationVal.(int); ok {
		return int64(d)
	}
	return 0
}

// ---------------------------------------------------------------------------
// ResolveSleepUntil
// ---------------------------------------------------------------------------

// ResolveSleepUntilParams holds parameters for resolving sleepUntil duration.
type ResolveSleepUntilParams struct {
	WorkflowID     string
	Step           *wf.StepFlowEntry
	RunID          string
	Input          any
	ResumeData     any
	StepResults    map[string]wf.StepResult
	State          map[string]any
	RequestContext *requestcontext.RequestContext
	RetryCount     int
	AbortCtx       context.Context
	AbortCancel    context.CancelFunc
}

// ResolveSleepUntil resolves the sleep-until duration for a sleepUntil step.
// Returns duration in milliseconds until the target date.
// TS equivalent: async resolveSleepUntil(params): Promise<number>
func (se *StepExecutor) ResolveSleepUntil(params ResolveSleepUntilParams) int64 {
	if params.Step == nil {
		return 0
	}

	// If a static date is specified, compute duration from now
	if params.Step.Date != nil {
		return params.Step.Date.UnixMilli() - time.Now().UnixMilli()
	}

	// If no function, return 0
	if params.Step.Fn == nil {
		return 0
	}

	currentState := params.State
	if currentState == nil {
		currentState = map[string]any{}
	}

	ctx := &wf.ExecuteFunctionParams{
		WorkflowID:     params.WorkflowID,
		RunID:          params.RunID,
		RequestContext: params.RequestContext,
		InputData:      params.Input,
		State:          currentState,
		RetryCount:     params.RetryCount,
		ResumeData:     params.ResumeData,
		GetInitData: func() any {
			if v, ok := params.StepResults["input"]; ok {
				return v
			}
			return nil
		},
		GetStepResult: func(step any) any {
			return wf.GetStepResult(params.StepResults, step)
		},
		AbortCtx: params.AbortCtx,
	}

	result, err := params.Step.Fn(ctx)
	if err != nil {
		if se.log != nil {
			se.log.Error(fmt.Sprintf("error resolving sleepUntil date: %v", err))
		}
		return 0
	}

	if targetDate, ok := result.(time.Time); ok {
		return targetDate.UnixMilli() - time.Now().UnixMilli()
	}
	if targetDate, ok := result.(*time.Time); ok && targetDate != nil {
		return targetDate.UnixMilli() - time.Now().UnixMilli()
	}
	return 0
}
