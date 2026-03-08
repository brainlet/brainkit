// Ported from: packages/core/src/workflows/handlers/step.ts
package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Execute Step Params
// ---------------------------------------------------------------------------

// ExecuteStepParams holds parameters for executing a single step.
// TS equivalent: export interface ExecuteStepParams extends ObservabilityContext
type ExecuteStepParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	Step                *wf.Step
	StepResults         map[string]wf.StepResult
	ExecutionContext    *wf.ExecutionContext
	Restart             *wf.RestartExecutionParams
	TimeTravel          *wf.TimeTravelExecutionParams
	Resume              *ResumeParams
	PrevOutput          any
	PubSub              events.PubSub
	AbortCtx            context.Context
	AbortCancel         context.CancelFunc
	RequestContext      *requestcontext.RequestContext
	SkipEmits           bool
	OutputWriter        wf.OutputWriter
	DisableScorers      bool
	SerializedStepGraph []wf.SerializedStepFlowEntry
	IterationCount      int
	PerStep             bool
	Observability       *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// ExecuteStep
// ---------------------------------------------------------------------------

// ExecuteStep executes a single step with validation, retry logic, and event emission.
// TS equivalent: export async function executeStep(engine, params)
func ExecuteStep(engine DefaultEngine, params ExecuteStepParams) (*wf.StepExecutionResult, error) {
	step := params.Step
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	stepCallID := fmt.Sprintf("%d", time.Now().UnixNano())

	// Validate step input
	validateInputs := true
	if opts := engine.GetOptions(); opts != nil {
		validateInputs = opts.ValidateInputs
	}

	inputResult := wf.ValidateStepInput(params.PrevOutput, step, validateInputs)
	inputData := inputResult.InputData
	inputValidationError := inputResult.ValidationError

	// Validate request context
	var requestContextData map[string]any
	if params.RequestContext != nil {
		requestContextData = params.RequestContext.Entries()
	}
	rcResult := wf.ValidateStepRequestContext(requestContextData, step, validateInputs)

	// Combine validation errors
	validationError := inputValidationError
	if validationError == nil {
		validationError = rcResult.ValidationError
	}

	// Handle time travel resume data validation
	var resumeDataToUse any
	if params.TimeTravel != nil {
		if sr, ok := params.TimeTravel.StepResults[step.ID]; ok {
			if sr.Status == wf.StepStatusSuspended {
				ttResumeResult := wf.ValidateStepResumeData(params.TimeTravel.ResumeData, step)
				if ttResumeResult.ResumeData != nil && ttResumeResult.ValidationError == nil {
					resumeDataToUse = ttResumeResult.ResumeData
				} else if ttResumeResult.ResumeData != nil && ttResumeResult.ValidationError != nil {
					log := engine.GetLogger()
					if log != nil {
						log.Warn(fmt.Sprintf("Time travel resume data validation failed for step %s: %v", step.ID, ttResumeResult.ValidationError))
					}
				}
			}
		}
	}
	if resumeDataToUse == nil && params.Resume != nil && len(params.Resume.Steps) > 0 && params.Resume.Steps[0] == step.ID {
		resumeDataToUse = params.Resume.ResumePayload
	}

	// Extract suspend data if this step was previously suspended
	var suspendDataToUse any
	if sr, ok := stepResults[step.ID]; ok && sr.Status == wf.StepStatusSuspended {
		suspendDataToUse = sr.SuspendPayload
	}

	// Filter out internal workflow metadata
	if suspendDataToUse != nil {
		if m, ok := suspendDataToUse.(map[string]any); ok {
			if _, hasMeta := m["__workflow_meta"]; hasMeta {
				filtered := make(map[string]any)
				for k, v := range m {
					if k != "__workflow_meta" {
						filtered[k] = v
					}
				}
				suspendDataToUse = filtered
			}
		}
	}

	var startTime *int64
	var resumeTime *int64
	now := time.Now().UnixMilli()
	if resumeDataToUse != nil {
		resumeTime = &now
	} else {
		startTime = &now
	}

	// Build step info
	stepInfo := wf.StepResult{
		Status: wf.StepStatusRunning,
	}
	if existing, ok := stepResults[step.ID]; ok {
		stepInfo = existing
		stepInfo.Status = wf.StepStatusRunning
	}
	if resumeDataToUse != nil {
		stepInfo.ResumePayload = resumeDataToUse
	} else {
		stepInfo.Payload = inputData
	}
	if startTime != nil {
		stepInfo.StartedAt = *startTime
	}
	if resumeTime != nil {
		stepInfo.ResumedAt = resumeTime
	}
	if params.IterationCount > 0 {
		stepInfo.Metadata = wf.StepMetadata{"iterationCount": params.IterationCount}
	}

	executionContext.ActiveStepsPath[step.ID] = executionContext.ExecutionPath

	// Emit step start event
	operationID := fmt.Sprintf("workflow.%s.run.%s.step.%s.running_ev", params.WorkflowID, params.RunID, step.ID)
	_, err := engine.OnStepExecutionStart(StepExecutionStartParams{
		Step:             step,
		InputData:        inputData,
		PubSub:           params.PubSub,
		ExecutionContext: executionContext,
		StepCallID:       stepCallID,
		StepInfo: map[string]any{
			"status":    string(stepInfo.Status),
			"payload":   stepInfo.Payload,
			"startedAt": stepInfo.StartedAt,
		},
		OperationID: operationID,
		SkipEmits:   params.SkipEmits,
	})
	if err != nil {
		return nil, fmt.Errorf("step execution start failed: %w", err)
	}

	// Persist running state
	err = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
		WorkflowID:          params.WorkflowID,
		RunID:               params.RunID,
		ResourceID:          params.ResourceID,
		SerializedStepGraph: params.SerializedStepGraph,
		StepResults: func() map[string]wf.StepResult {
			merged := make(map[string]wf.StepResult)
			for k, v := range stepResults {
				merged[k] = v
			}
			merged[step.ID] = stepInfo
			return merged
		}(),
		ExecutionContext: executionContext,
		WorkflowStatus:  wf.WorkflowRunStatusRunning,
		RequestContext:   params.RequestContext,
	})
	if err != nil {
		return nil, fmt.Errorf("persist step update failed: %w", err)
	}

	// Check for nested workflow step
	if engine.IsNestedWorkflowStep(step) {
		workflowResult, err := engine.ExecuteWorkflowStep(ExecuteWorkflowStepParams{
			Step:             step,
			StepResults:      stepResults,
			ExecutionContext: executionContext,
			Resume:           params.Resume,
			TimeTravel:       params.TimeTravel,
			PrevOutput:       params.PrevOutput,
			InputData:        inputData,
			PubSub:           params.PubSub,
			StartedAt:        func() int64 { if startTime != nil { return *startTime }; return now }(),
			AbortCtx:         params.AbortCtx,
			AbortCancel:      params.AbortCancel,
			RequestContext:   params.RequestContext,
			OutputWriter:     params.OutputWriter,
			PerStep:          params.PerStep,
			Observability:    params.Observability,
		})
		if err != nil {
			return nil, err
		}
		if workflowResult != nil {
			sr := wf.StepResult{
				Status:         workflowResult.Status,
				Output:         workflowResult.Output,
				Payload:        stepInfo.Payload,
				Error:          workflowResult.Error,
				StartedAt:      stepInfo.StartedAt,
				EndedAt:        workflowResult.EndedAt,
				SuspendPayload: workflowResult.SuspendPayload,
				SuspendOutput:  workflowResult.SuspendOutput,
				SuspendedAt:    workflowResult.SuspendedAt,
				ResumedAt:      stepInfo.ResumedAt,
				Metadata:       stepInfo.Metadata,
			}
			return &wf.StepExecutionResult{
				Result:         sr,
				StepResults:    map[string]wf.StepResult{step.ID: sr},
				MutableContext: engine.BuildMutableContext(executionContext),
				RequestContext: engine.SerializeRequestContext(params.RequestContext),
			}, nil
		}
	}

	// Execute the step with retry logic
	retries := 0
	if step.Retries != nil {
		retries = *step.Retries
	} else {
		retries = executionContext.RetryConfig.Attempts
	}
	delay := executionContext.RetryConfig.Delay

	stepRetryResult, err := engine.ExecuteStepWithRetry(
		fmt.Sprintf("workflow.%s.step.%s", params.WorkflowID, step.ID),
		func() (any, error) {
			if validationError != nil {
				return nil, validationError
			}

			retryCount := engine.GetOrGenerateRetryCount(step.ID)

			// Prepare time travel steps
			var timeTravelSteps []string
			if params.TimeTravel != nil && len(params.TimeTravel.Steps) > 0 {
				if params.TimeTravel.Steps[0] == step.ID {
					timeTravelSteps = params.TimeTravel.Steps[1:]
				}
			}

			var suspended *suspendResult
			var bailed *bailResult
			contextMutations := &contextMutationsData{
				SuspendedPaths:       make(map[string][]int),
				ResumeLabels:         make(map[string]wf.ResumeLabel),
				StateUpdate:          nil,
				RequestContextUpdate: nil,
			}

			// Build execute function params
			execParams := &wf.ExecuteFunctionParams{
				RunID:          params.RunID,
				ResourceID:     params.ResourceID,
				WorkflowID:     params.WorkflowID,
				Mastra:         engine.GetMastra(),
				RequestContext: params.RequestContext,
				InputData:      inputData,
				State:          executionContext.State,
				SetState: func(state any) error {
					stateResult := wf.ValidateStepStateData(state, step, validateInputs)
					if stateResult.ValidationError != nil {
						return stateResult.ValidationError
					}
					contextMutations.StateUpdate = stateResult.StateData
					return nil
				},
				RetryCount:  retryCount,
				ResumeData:  resumeDataToUse,
				SuspendData: suspendDataToUse,
				GetInitData: func() any {
					if r, ok := stepResults["input"]; ok {
						return r.Output
					}
					return nil
				},
				GetStepResult: func(s any) any {
					return wf.GetStepResult(stepResults, s)
				},
				Suspend: func(suspendPayload any, suspendOptions *wf.SuspendOptions) error {
					suspendValidation := wf.ValidateStepSuspendData(suspendPayload, step, validateInputs)
					if suspendValidation.ValidationError != nil {
						return suspendValidation.ValidationError
					}

					contextMutations.SuspendedPaths[step.ID] = executionContext.ExecutionPath
					executionContext.SuspendedPaths[step.ID] = executionContext.ExecutionPath

					if suspendOptions != nil && len(suspendOptions.ResumeLabel) > 0 {
						for _, label := range suspendOptions.ResumeLabel {
							labelData := wf.ResumeLabel{
								StepID:       step.ID,
								ForeachIndex: executionContext.ForeachIndex,
							}
							contextMutations.ResumeLabels[label] = labelData
							executionContext.ResumeLabels[label] = labelData
						}
					}

					suspended = &suspendResult{Payload: suspendValidation.SuspendData}
					return nil
				},
				Bail: func(result any) {
					bailed = &bailResult{Payload: result}
				},
				Abort: func() {
					if params.AbortCancel != nil {
						params.AbortCancel()
					}
				},
				PubSub:       params.PubSub,
				StreamFormat: executionContext.Format,
				Engine:       engine.GetEngineContext(),
				AbortCtx:     params.AbortCtx,
				OutputWriter: params.OutputWriter,
				ValidateSchemas: validateInputs,
				Observability:   params.Observability,
			}

			// Set resume info if step was previously suspended
			if sr, ok := stepResults[step.ID]; ok && sr.Status == wf.StepStatusSuspended {
				execParams.Resume = &wf.ResumeInfo{
					Steps: func() []string {
						if params.Resume != nil && len(params.Resume.Steps) > 1 {
							return params.Resume.Steps[1:]
						}
						return nil
					}(),
					ResumePayload: func() any {
						if params.Resume != nil {
							return params.Resume.ResumePayload
						}
						return nil
					}(),
					Label: func() string {
						if params.Resume != nil {
							return params.Resume.Label
						}
						return ""
					}(),
					ForEachIndex: func() *int {
						if params.Resume != nil {
							return params.Resume.ForEachIndex
						}
						return nil
					}(),
				}
			}

			// Set restart flag
			if params.Restart != nil {
				if _, ok := params.Restart.ActiveStepsPath[step.ID]; ok {
					execParams.Restart = true
				}
			}

			// Execute the step
			if step.Execute == nil {
				return nil, fmt.Errorf("step %s has no execute function", step.ID)
			}
			output, execErr := step.Execute(execParams)

			// Capture request context state if needed
			if engine.RequiresDurableContextSerialization() {
				contextMutations.RequestContextUpdate = engine.SerializeRequestContext(params.RequestContext)
			}

			isNestedWorkflow := step.Component == "WORKFLOW"
			nestedWflowStepPaused := isNestedWorkflow && params.PerStep

			if execErr != nil {
				return nil, execErr
			}

			return &stepExecutionOutput{
				Output:                 output,
				Suspended:              suspended,
				Bailed:                 bailed,
				ContextMutations:       contextMutations,
				NestedWflowStepPaused:  nestedWflowStepPaused,
				TimeTravelSteps:        timeTravelSteps,
			}, nil
		},
		RetryParams{
			Retries:    retries,
			Delay:      delay,
			WorkflowID: params.WorkflowID,
			RunID:      params.RunID,
		},
	)
	if err != nil {
		return nil, err
	}

	// Process the retry result
	var execResults wf.StepResult
	if !stepRetryResult.OK {
		execResults = wf.StepResult{
			Status:  wf.StepStatusFailed,
			Error:   stepRetryResult.Error.Error,
			EndedAt: stepRetryResult.Error.EndedAt,
		}
		if stepRetryResult.Error.Tripwire != nil {
			execResults.Tripwire = stepRetryResult.Error.Tripwire
		}
	} else {
		durableResult, ok := stepRetryResult.Result.(*stepExecutionOutput)
		if !ok {
			return nil, fmt.Errorf("unexpected step retry result type")
		}

		// Apply context mutations
		for k, v := range durableResult.ContextMutations.SuspendedPaths {
			executionContext.SuspendedPaths[k] = v
		}
		for k, v := range durableResult.ContextMutations.ResumeLabels {
			executionContext.ResumeLabels[k] = v
		}

		// Restore request context if needed
		if engine.RequiresDurableContextSerialization() && durableResult.ContextMutations.RequestContextUpdate != nil {
			if params.RequestContext != nil {
				params.RequestContext.Clear()
				for k, v := range durableResult.ContextMutations.RequestContextUpdate {
					params.RequestContext.Set(k, v)
				}
			}
		}

		// TODO: Run scorers for step
		// if step.Scorers != nil { ... }

		if durableResult.Suspended != nil {
			suspendedAt := time.Now().UnixMilli()
			execResults = wf.StepResult{
				Status:         wf.StepStatusSuspended,
				SuspendPayload: durableResult.Suspended.Payload,
				SuspendedAt:    &suspendedAt,
			}
			if durableResult.Output != nil {
				execResults.SuspendOutput = durableResult.Output
			}
		} else if durableResult.Bailed != nil {
			endedAt := time.Now().UnixMilli()
			execResults = wf.StepResult{
				Status:  "bailed",
				Output:  durableResult.Bailed.Payload,
				EndedAt: endedAt,
			}
		} else if durableResult.NestedWflowStepPaused {
			execResults = wf.StepResult{
				Status: wf.StepStatusPaused,
			}
		} else {
			endedAt := time.Now().UnixMilli()
			execResults = wf.StepResult{
				Status:  wf.StepStatusSuccess,
				Output:  durableResult.Output,
				EndedAt: endedAt,
			}
		}
	}

	// Remove from active steps
	delete(executionContext.ActiveStepsPath, step.ID)

	// Emit step result events
	if !params.SkipEmits {
		mergedResult := mergeStepResult(stepInfo, execResults)
		err := EmitStepResultEvents(EmitStepResultEventsParams{
			StepID:      step.ID,
			StepCallID:  stepCallID,
			ExecResults: mergedResult,
			PubSub:      params.PubSub,
			RunID:       params.RunID,
		})
		if err != nil {
			logError(engine, "Error emitting step result events: %v", err)
		}
	}

	// Build final step result
	finalResult := mergeStepResult(stepInfo, execResults)

	// Build state for mutable context
	state := executionContext.State
	if stepRetryResult.OK {
		if durableResult, ok := stepRetryResult.Result.(*stepExecutionOutput); ok {
			if durableResult.ContextMutations.StateUpdate != nil {
				if m, ok := durableResult.ContextMutations.StateUpdate.(map[string]any); ok {
					state = m
				}
			}
		}
	}

	return &wf.StepExecutionResult{
		Result:      finalResult,
		StepResults: map[string]wf.StepResult{step.ID: finalResult},
		MutableContext: wf.MutableContext{
			State:          state,
			SuspendedPaths: executionContext.SuspendedPaths,
			ResumeLabels:   executionContext.ResumeLabels,
		},
		RequestContext: engine.SerializeRequestContext(params.RequestContext),
	}, nil
}

// ---------------------------------------------------------------------------
// Emit Step Result Events
// ---------------------------------------------------------------------------

// EmitStepResultEventsParams holds parameters for emitting step result events.
type EmitStepResultEventsParams struct {
	StepID      string
	StepCallID  string
	ExecResults wf.StepResult
	PubSub      events.PubSub
	RunID       string
}

// EmitStepResultEvents emits step result events (suspended, result, finish).
// TS equivalent: export async function emitStepResultEvents(params)
func EmitStepResultEvents(params EmitStepResultEventsParams) error {
	payloadBase := map[string]any{
		"id": params.StepID,
	}
	if params.StepCallID != "" {
		payloadBase["stepCallId"] = params.StepCallID
	}

	eventChannel := fmt.Sprintf("workflow.events.v2.%s", params.RunID)

	if params.ExecResults.Status == wf.StepStatusSuspended {
		payload := mergePayload(payloadBase, stepResultToMap(params.ExecResults))
		return params.PubSub.Publish(eventChannel, events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type":    "workflow-step-suspended",
				"payload": payload,
			},
		})
	}

	// Emit result
	payload := mergePayload(payloadBase, stepResultToMap(params.ExecResults))
	err := params.PubSub.Publish(eventChannel, events.PublishEvent{
		Type:  "watch",
		RunID: params.RunID,
		Data: map[string]any{
			"type":    "workflow-step-result",
			"payload": payload,
		},
	})
	if err != nil {
		return err
	}

	// Emit finish
	return params.PubSub.Publish(eventChannel, events.PublishEvent{
		Type:  "watch",
		RunID: params.RunID,
		Data: map[string]any{
			"type":    "workflow-step-finish",
			"payload": mergePayload(payloadBase, map[string]any{"metadata": map[string]any{}}),
		},
	})
}

// ---------------------------------------------------------------------------
// RunScorersForStep (stub)
// ---------------------------------------------------------------------------

// RunScorersForStep runs scorers for a step.
// TODO: Implement once evals package is ported.
func RunScorersForStep(_ DefaultEngine, _ any) {
	// Stub - scorers not yet ported
}

// ---------------------------------------------------------------------------
// Internal helper types
// ---------------------------------------------------------------------------

type suspendResult struct {
	Payload any
}

type bailResult struct {
	Payload any
}

type contextMutationsData struct {
	SuspendedPaths       map[string][]int
	ResumeLabels         map[string]wf.ResumeLabel
	StateUpdate          any
	RequestContextUpdate map[string]any
}

type stepExecutionOutput struct {
	Output                any
	Suspended             *suspendResult
	Bailed                *bailResult
	ContextMutations      *contextMutationsData
	NestedWflowStepPaused bool
	TimeTravelSteps       []string
}

// mergeStepResult merges a base step result with overrides.
func mergeStepResult(base, override wf.StepResult) wf.StepResult {
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
	if override.ResumedAt != nil {
		result.ResumedAt = override.ResumedAt
	}
	if override.Metadata != nil {
		result.Metadata = override.Metadata
	}
	if override.Tripwire != nil {
		result.Tripwire = override.Tripwire
	}
	return result
}

// mergePayload merges two maps.
func mergePayload(base, extra map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range extra {
		result[k] = v
	}
	return result
}

// stepResultToMap converts a StepResult to a map for event payloads.
func stepResultToMap(sr wf.StepResult) map[string]any {
	m := map[string]any{
		"status": string(sr.Status),
	}
	if sr.Output != nil {
		m["output"] = sr.Output
	}
	if sr.Payload != nil {
		m["payload"] = sr.Payload
	}
	if sr.Error != nil {
		m["error"] = sr.Error.Error()
	}
	if sr.SuspendPayload != nil {
		m["suspendPayload"] = sr.SuspendPayload
	}
	if sr.SuspendOutput != nil {
		m["suspendOutput"] = sr.SuspendOutput
	}
	if sr.StartedAt != 0 {
		m["startedAt"] = sr.StartedAt
	}
	if sr.EndedAt != 0 {
		m["endedAt"] = sr.EndedAt
	}
	if sr.SuspendedAt != nil {
		m["suspendedAt"] = *sr.SuspendedAt
	}
	if sr.ResumedAt != nil {
		m["resumedAt"] = *sr.ResumedAt
	}
	return m
}
