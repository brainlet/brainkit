// Ported from: packages/core/src/workflows/handlers/entry.ts
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
// Execute Entry Params
// ---------------------------------------------------------------------------

// ExecuteEntryParams holds parameters for executing a single entry in the step graph.
// TS equivalent: export interface ExecuteEntryParams extends ObservabilityContext
type ExecuteEntryParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	Entry               wf.StepFlowEntry
	PrevStep            wf.StepFlowEntry
	SerializedStepGraph []wf.SerializedStepFlowEntry
	StepResults         map[string]wf.StepResult
	Restart             *wf.RestartExecutionParams
	TimeTravel          *wf.TimeTravelExecutionParams
	Resume              *ResumeParams
	ExecutionContext    *wf.ExecutionContext
	PubSub              events.PubSub
	AbortCtx            context.Context
	AbortCancel         context.CancelFunc
	RequestContext      *requestcontext.RequestContext
	OutputWriter        wf.OutputWriter
	DisableScorers      bool
	PerStep             bool
	Observability       *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// PersistStepUpdate
// ---------------------------------------------------------------------------

// PersistStepUpdate persists a step update to storage.
// TS equivalent: export async function persistStepUpdate(engine, params)
func PersistStepUpdate(engine DefaultEngine, params PersistStepUpdateParams) error {
	shouldPersist := false
	if opts := engine.GetOptions(); opts != nil && opts.ShouldPersistSnapshot != nil {
		shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
			StepResults:    params.StepResults,
			WorkflowStatus: params.WorkflowStatus,
		})
	}

	if !shouldPersist {
		return nil
	}

	mastra := engine.GetMastra()
	if mastra == nil {
		return nil
	}

	storage := mastra.GetStorage()
	if storage == nil {
		return nil
	}

	store, err := storage.GetStore("workflows")
	if err != nil || store == nil {
		return nil
	}

	var requestContextObj map[string]any
	if params.RequestContext != nil {
		requestContextObj = params.RequestContext.Entries()
	}

	// Build context from step results
	contextMap := make(map[string]any)
	for k, v := range params.StepResults {
		contextMap[k] = v
	}

	snapshot := wf.WorkflowRunState{
		RunID:               params.RunID,
		Status:              params.WorkflowStatus,
		Value:               convertStateToStringMap(params.ExecutionContext.State),
		Context:             contextMap,
		ActivePaths:         params.ExecutionContext.ExecutionPath,
		StepExecutionPath:   params.ExecutionContext.StepExecutionPath,
		ActiveStepsPath:     params.ExecutionContext.ActiveStepsPath,
		SerializedStepGraph: params.SerializedStepGraph,
		SuspendedPaths:      params.ExecutionContext.SuspendedPaths,
		WaitingPaths:        map[string][]int{},
		ResumeLabels:        params.ExecutionContext.ResumeLabels,
		RequestContext:      requestContextObj,
		Timestamp:           time.Now().UnixMilli(),
	}

	if params.Result != nil {
		snapshot.Result = params.Result
	}

	return store.PersistWorkflowSnapshot(wf.PersistWorkflowSnapshotParams{
		WorkflowName: params.WorkflowID,
		RunID:        params.RunID,
		ResourceID:   params.ResourceID,
		Snapshot:     snapshot,
	})
}

// ---------------------------------------------------------------------------
// BuildResumedBlockResult
// ---------------------------------------------------------------------------

// buildResumedBlockResult checks whether all relevant branch steps are now
// complete and builds the appropriate block-level result.
// TS equivalent: function buildResumedBlockResult(entrySteps, stepResults, executionContext, opts?)
func buildResumedBlockResult(
	entrySteps []wf.StepFlowStepEntry,
	stepResults map[string]wf.StepResult,
	executionContext *wf.ExecutionContext,
	onlyExecutedSteps bool,
) wf.StepResult {
	stepsToCheck := entrySteps
	if onlyExecutedSteps {
		filtered := make([]wf.StepFlowStepEntry, 0)
		for _, s := range entrySteps {
			if s.Step != nil {
				if _, exists := stepResults[s.Step.ID]; exists {
					filtered = append(filtered, s)
				}
			}
		}
		stepsToCheck = filtered
	}

	allComplete := true
	for _, s := range stepsToCheck {
		if s.Step != nil {
			r, ok := stepResults[s.Step.ID]
			if !ok || r.Status != wf.StepStatusSuccess {
				allComplete = false
				break
			}
		}
	}

	if allComplete {
		output := make(map[string]any)
		for _, s := range entrySteps {
			if s.Step != nil {
				if r, ok := stepResults[s.Step.ID]; ok && r.Status == wf.StepStatusSuccess {
					output[s.Step.ID] = r.Output
				}
			}
		}
		return wf.StepResult{
			Status: wf.StepStatusSuccess,
			Output: output,
		}
	}

	// Find a still-suspended step
	var suspendData any
	for _, s := range entrySteps {
		if s.Step != nil {
			if r, ok := stepResults[s.Step.ID]; ok && r.Status == wf.StepStatusSuspended {
				suspendData = r.SuspendPayload
				break
			}
		}
	}

	now := time.Now().UnixMilli()
	result := wf.StepResult{
		Status:         wf.StepStatusSuspended,
		Payload:        suspendData,
		SuspendPayload: suspendData,
		SuspendedAt:    &now,
	}

	// Update suspended paths
	for i, s := range entrySteps {
		if s.Step != nil {
			if r, ok := stepResults[s.Step.ID]; ok && r.Status == wf.StepStatusSuspended {
				executionContext.SuspendedPaths[s.Step.ID] = append(
					append([]int{}, executionContext.ExecutionPath...),
					i,
				)
			}
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// ExecuteEntry
// ---------------------------------------------------------------------------

// ExecuteEntry executes a single entry in the step graph.
// TS equivalent: export async function executeEntry(engine, params)
func ExecuteEntry(engine DefaultEngine, params ExecuteEntryParams) (*wf.EntryExecutionResult, error) {
	entry := params.Entry
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	prevOutput := engine.GetStepOutput(stepResultsToAnyMap(stepResults), &params.PrevStep)
	var execResults *wf.StepResult
	var entryRequestContext map[string]any

	switch entry.Type {
	case wf.StepFlowEntryTypeStep:
		if entry.Step == nil {
			return nil, fmt.Errorf("step entry has no step")
		}

		// Track step execution path
		isResumedStep := false
		if params.Resume != nil {
			for _, s := range params.Resume.Steps {
				if s == entry.Step.ID {
					isResumedStep = true
					break
				}
			}
		}
		if !isResumedStep && executionContext.StepExecutionPath != nil {
			executionContext.StepExecutionPath = append(executionContext.StepExecutionPath, entry.Step.ID)
		}

		stepExecResult, err := engine.ExecuteStepHandler(ExecuteStepParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			Step:                entry.Step,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			TimeTravel:          params.TimeTravel,
			Restart:             params.Restart,
			Resume:              params.Resume,
			PrevOutput:          prevOutput,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			SerializedStepGraph: params.SerializedStepGraph,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}

		execResults = &stepExecResult.Result
		engine.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
		for k, v := range stepExecResult.StepResults {
			stepResults[k] = v
		}
		entryRequestContext = stepExecResult.RequestContext

	case wf.StepFlowEntryTypeParallel:
		if params.Resume != nil && len(params.Resume.ResumePath) > 0 {
			// Resume-aware handling for parallel entries
			idx := params.Resume.ResumePath[0]
			params.Resume.ResumePath = params.Resume.ResumePath[1:]

			if idx < len(entry.Steps) {
				childEntry := stepFlowStepEntryToFlowEntry(entry.Steps[idx])
				childResult, err := ExecuteEntry(engine, ExecuteEntryParams{
					WorkflowID:          params.WorkflowID,
					RunID:               params.RunID,
					ResourceID:          params.ResourceID,
					Entry:               childEntry,
					PrevStep:            params.PrevStep,
					SerializedStepGraph: params.SerializedStepGraph,
					StepResults:         stepResults,
					Resume:              params.Resume,
					ExecutionContext: &wf.ExecutionContext{
						WorkflowID:        params.WorkflowID,
						RunID:             params.RunID,
						ExecutionPath:     append(append([]int{}, executionContext.ExecutionPath...), idx),
						StepExecutionPath: copyStringSlice(executionContext.StepExecutionPath),
						SuspendedPaths:    executionContext.SuspendedPaths,
						ResumeLabels:      executionContext.ResumeLabels,
						RetryConfig:       executionContext.RetryConfig,
						ActiveStepsPath:   executionContext.ActiveStepsPath,
						State:             executionContext.State,
					},
					PubSub:         params.PubSub,
					AbortCtx:       params.AbortCtx,
					AbortCancel:    params.AbortCancel,
					RequestContext: params.RequestContext,
					OutputWriter:   params.OutputWriter,
					DisableScorers: params.DisableScorers,
					PerStep:        params.PerStep,
					Observability:  params.Observability,
				})
				if err != nil {
					return nil, err
				}

				engine.ApplyMutableContext(executionContext, childResult.MutableContext)
				for k, v := range childResult.StepResults {
					stepResults[k] = v
				}

				blockResult := buildResumedBlockResult(entry.Steps, stepResults, executionContext, false)
				return &wf.EntryExecutionResult{
					Result:         blockResult,
					StepResults:    stepResults,
					MutableContext: engine.BuildMutableContext(executionContext),
					RequestContext: childResult.RequestContext,
				}, nil
			}
		}

		// Non-resume parallel execution
		parallelResult, err := engine.ExecuteParallelHandler(ExecuteParallelParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			Entry:               entry,
			PrevStep:            params.PrevStep,
			StepResults:         stepResults,
			SerializedStepGraph: params.SerializedStepGraph,
			TimeTravel:          params.TimeTravel,
			Restart:             params.Restart,
			Resume:              params.Resume,
			ExecutionContext:    executionContext,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}
		execResults = parallelResult

	case wf.StepFlowEntryTypeConditional:
		if params.Resume != nil && len(params.Resume.ResumePath) > 0 {
			// Resume-aware handling for conditional entries
			idx := params.Resume.ResumePath[0]
			params.Resume.ResumePath = params.Resume.ResumePath[1:]

			if idx < len(entry.Steps) {
				branchStep := entry.Steps[idx]

				var branchResult *wf.EntryExecutionResult
				var err error

				childExecCtx := &wf.ExecutionContext{
					WorkflowID:        params.WorkflowID,
					RunID:             params.RunID,
					ExecutionPath:     append(append([]int{}, executionContext.ExecutionPath...), idx),
					StepExecutionPath: copyStringSlice(executionContext.StepExecutionPath),
					SuspendedPaths:    executionContext.SuspendedPaths,
					ResumeLabels:      executionContext.ResumeLabels,
					RetryConfig:       executionContext.RetryConfig,
					ActiveStepsPath:   executionContext.ActiveStepsPath,
					State:             executionContext.State,
				}

				if branchStep.Step != nil {
					// Use step's stored payload as prevOutput
					resumePrevOutput := prevOutput
					if r, ok := stepResults[branchStep.Step.ID]; ok && r.Payload != nil {
						resumePrevOutput = r.Payload
					}

					stepExecResult, err2 := engine.ExecuteStepHandler(ExecuteStepParams{
						WorkflowID:          params.WorkflowID,
						RunID:               params.RunID,
						ResourceID:          params.ResourceID,
						Step:                branchStep.Step,
						PrevOutput:          resumePrevOutput,
						StepResults:         stepResults,
						SerializedStepGraph: params.SerializedStepGraph,
						Resume:              params.Resume,
						Restart:             params.Restart,
						TimeTravel:          params.TimeTravel,
						ExecutionContext:    childExecCtx,
						PubSub:              params.PubSub,
						AbortCtx:            params.AbortCtx,
						AbortCancel:         params.AbortCancel,
						RequestContext:      params.RequestContext,
						OutputWriter:        params.OutputWriter,
						DisableScorers:      params.DisableScorers,
						PerStep:             params.PerStep,
						Observability:       params.Observability,
					})
					if err2 != nil {
						return nil, err2
					}
					branchResult = &wf.EntryExecutionResult{
						Result:         stepExecResult.Result,
						StepResults:    stepExecResult.StepResults,
						MutableContext: stepExecResult.MutableContext,
						RequestContext: stepExecResult.RequestContext,
					}
				} else {
					// Should not happen for conditional, but handle gracefully
					return nil, fmt.Errorf("conditional branch at index %d has no step", idx)
				}

				if err != nil {
					return nil, err
				}

				engine.ApplyMutableContext(executionContext, branchResult.MutableContext)
				for k, v := range branchResult.StepResults {
					stepResults[k] = v
				}

				blockResult := buildResumedBlockResult(entry.Steps, stepResults, executionContext, true)
				return &wf.EntryExecutionResult{
					Result:         blockResult,
					StepResults:    stepResults,
					MutableContext: engine.BuildMutableContext(executionContext),
					RequestContext: branchResult.RequestContext,
				}, nil
			}
		}

		// Non-resume conditional execution
		condResult, err := engine.ExecuteConditionalHandler(ExecuteConditionalParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			Entry:               entry,
			PrevOutput:          prevOutput,
			StepResults:         stepResults,
			SerializedStepGraph: params.SerializedStepGraph,
			TimeTravel:          params.TimeTravel,
			Restart:             params.Restart,
			Resume:              params.Resume,
			ExecutionContext:    executionContext,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}
		execResults = condResult

	case wf.StepFlowEntryTypeLoop:
		loopResult, err := engine.ExecuteLoopHandler(ExecuteLoopParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			Entry:               entry,
			PrevStep:            params.PrevStep,
			PrevOutput:          prevOutput,
			StepResults:         stepResults,
			TimeTravel:          params.TimeTravel,
			Restart:             params.Restart,
			Resume:              params.Resume,
			ExecutionContext:    executionContext,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			SerializedStepGraph: params.SerializedStepGraph,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}
		execResults = loopResult

	case wf.StepFlowEntryTypeForeach:
		foreachResult, err := engine.ExecuteForeachHandler(ExecuteForeachParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			Entry:               entry,
			PrevStep:            params.PrevStep,
			PrevOutput:          prevOutput,
			StepResults:         stepResults,
			TimeTravel:          params.TimeTravel,
			Restart:             params.Restart,
			Resume:              params.Resume,
			ExecutionContext:    executionContext,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			SerializedStepGraph: params.SerializedStepGraph,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}
		execResults = foreachResult

	case wf.StepFlowEntryTypeSleep:
		if executionContext.StepExecutionPath != nil {
			executionContext.StepExecutionPath = append(executionContext.StepExecutionPath, entry.ID)
		}
		startedAt := time.Now().UnixMilli()

		// Emit waiting event
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type": "workflow-step-waiting",
				"payload": map[string]any{
					"id":        entry.ID,
					"payload":   prevOutput,
					"startedAt": startedAt,
					"status":    "waiting",
				},
			},
		})

		stepResults[entry.ID] = wf.StepResult{
			Status:    wf.StepStatusWaiting,
			Payload:   prevOutput,
			StartedAt: startedAt,
		}
		executionContext.ActiveStepsPath[entry.ID] = executionContext.ExecutionPath

		// Persist waiting state
		_ = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			SerializedStepGraph: params.SerializedStepGraph,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			WorkflowStatus:      wf.WorkflowRunStatusWaiting,
			RequestContext:      params.RequestContext,
		})

		// Execute sleep
		err := engine.ExecuteSleepHandler(ExecuteSleepParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			SerializedStepGraph: params.SerializedStepGraph,
			Entry: SleepEntry{
				Type:     "sleep",
				ID:       entry.ID,
				Duration: entry.Duration,
				Fn:       entry.Fn,
			},
			PrevStep:         params.PrevStep,
			PrevOutput:       prevOutput,
			StepResults:      stepResults,
			Resume:           params.Resume,
			ExecutionContext: executionContext,
			PubSub:           params.PubSub,
			AbortCtx:         params.AbortCtx,
			AbortCancel:      params.AbortCancel,
			RequestContext:   params.RequestContext,
			OutputWriter:     params.OutputWriter,
			Observability:    params.Observability,
		})
		if err != nil {
			return nil, err
		}

		delete(executionContext.ActiveStepsPath, entry.ID)

		// Persist running state
		_ = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			SerializedStepGraph: params.SerializedStepGraph,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			WorkflowStatus:      wf.WorkflowRunStatusRunning,
			RequestContext:      params.RequestContext,
		})

		endedAt := time.Now().UnixMilli()
		sr := wf.StepResult{
			Status:    wf.StepStatusSuccess,
			Output:    prevOutput,
			Payload:   prevOutput,
			StartedAt: startedAt,
			EndedAt:   endedAt,
		}
		execResults = &sr
		stepResults[entry.ID] = sr

		// Emit result events
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type": "workflow-step-result",
				"payload": map[string]any{
					"id":      entry.ID,
					"endedAt": endedAt,
					"status":  "success",
					"output":  prevOutput,
				},
			},
		})
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type":    "workflow-step-finish",
				"payload": map[string]any{"id": entry.ID, "metadata": map[string]any{}},
			},
		})

	case wf.StepFlowEntryTypeSleepUntil:
		if executionContext.StepExecutionPath != nil {
			executionContext.StepExecutionPath = append(executionContext.StepExecutionPath, entry.ID)
		}
		startedAt := time.Now().UnixMilli()

		// Emit waiting event
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type": "workflow-step-waiting",
				"payload": map[string]any{
					"id":        entry.ID,
					"payload":   prevOutput,
					"startedAt": startedAt,
					"status":    "waiting",
				},
			},
		})

		stepResults[entry.ID] = wf.StepResult{
			Status:    wf.StepStatusWaiting,
			Payload:   prevOutput,
			StartedAt: startedAt,
		}
		executionContext.ActiveStepsPath[entry.ID] = executionContext.ExecutionPath

		// Persist waiting state
		_ = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			SerializedStepGraph: params.SerializedStepGraph,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			WorkflowStatus:      wf.WorkflowRunStatusWaiting,
			RequestContext:      params.RequestContext,
		})

		// Execute sleepUntil
		err := engine.ExecuteSleepUntilHandler(ExecuteSleepUntilParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			SerializedStepGraph: params.SerializedStepGraph,
			Entry: SleepUntilEntry{
				Type: "sleepUntil",
				ID:   entry.ID,
				Date: entry.Date,
				Fn:   entry.Fn,
			},
			PrevStep:         params.PrevStep,
			PrevOutput:       prevOutput,
			StepResults:      stepResults,
			Resume:           params.Resume,
			ExecutionContext: executionContext,
			PubSub:           params.PubSub,
			AbortCtx:         params.AbortCtx,
			AbortCancel:      params.AbortCancel,
			RequestContext:   params.RequestContext,
			OutputWriter:     params.OutputWriter,
			Observability:    params.Observability,
		})
		if err != nil {
			return nil, err
		}

		delete(executionContext.ActiveStepsPath, entry.ID)

		// Persist running state
		_ = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			SerializedStepGraph: params.SerializedStepGraph,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			WorkflowStatus:      wf.WorkflowRunStatusRunning,
			RequestContext:      params.RequestContext,
		})

		endedAt := time.Now().UnixMilli()
		sr := wf.StepResult{
			Status:    wf.StepStatusSuccess,
			Output:    prevOutput,
			Payload:   prevOutput,
			StartedAt: startedAt,
			EndedAt:   endedAt,
		}
		execResults = &sr
		stepResults[entry.ID] = sr

		// Emit result events
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type": "workflow-step-result",
				"payload": map[string]any{
					"id":      entry.ID,
					"endedAt": endedAt,
					"status":  "success",
					"output":  prevOutput,
				},
			},
		})
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data: map[string]any{
				"type":    "workflow-step-finish",
				"payload": map[string]any{"id": entry.ID, "metadata": map[string]any{}},
			},
		})

	default:
		return nil, fmt.Errorf("unknown entry type: %s", entry.Type)
	}

	if execResults == nil {
		return nil, fmt.Errorf("no execution result produced for entry type %s", entry.Type)
	}

	// Store step result for step/loop/foreach entries
	if entry.Type == wf.StepFlowEntryTypeStep || entry.Type == wf.StepFlowEntryTypeLoop || entry.Type == wf.StepFlowEntryTypeForeach {
		if entry.Step != nil {
			stepResults[entry.Step.ID] = *execResults
		}
	}

	// Check for abort
	if params.AbortCtx != nil {
		select {
		case <-params.AbortCtx.Done():
			execResults.Status = "canceled"
		default:
		}
	}

	// Persist final state
	workflowStatus := wf.WorkflowRunStatus(execResults.Status)
	if execResults.Status == wf.StepStatusSuccess {
		workflowStatus = wf.WorkflowRunStatusRunning
	}
	_ = engine.PersistStepUpdateHandler(PersistStepUpdateParams{
		WorkflowID:          params.WorkflowID,
		RunID:               params.RunID,
		ResourceID:          params.ResourceID,
		SerializedStepGraph: params.SerializedStepGraph,
		StepResults:         stepResults,
		ExecutionContext:    executionContext,
		WorkflowStatus:      workflowStatus,
		RequestContext:      params.RequestContext,
	})

	// Emit canceled event
	if execResults.Status == "canceled" {
		_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
			Type:  "watch",
			RunID: params.RunID,
			Data:  map[string]any{"type": "workflow-canceled", "payload": map[string]any{}},
		})
	}

	rc := entryRequestContext
	if rc == nil {
		rc = engine.SerializeRequestContext(params.RequestContext)
	}

	return &wf.EntryExecutionResult{
		Result:         *execResults,
		StepResults:    stepResults,
		MutableContext: engine.BuildMutableContext(executionContext),
		RequestContext: rc,
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// stepResultsToAnyMap converts map[string]StepResult to map[string]any for GetStepOutput.
func stepResultsToAnyMap(m map[string]wf.StepResult) map[string]any {
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// stepFlowStepEntryToFlowEntry converts a StepFlowStepEntry to a StepFlowEntry.
func stepFlowStepEntryToFlowEntry(e wf.StepFlowStepEntry) wf.StepFlowEntry {
	return wf.StepFlowEntry{
		Type: wf.StepFlowEntryTypeStep,
		Step: e.Step,
	}
}

// copyStringSlice creates a copy of a string slice.
func copyStringSlice(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	return c
}

// convertStateToStringMap converts map[string]any to map[string]string.
func convertStateToStringMap(state map[string]any) map[string]string {
	if state == nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(state))
	for k, v := range state {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}
