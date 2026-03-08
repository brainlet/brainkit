// Ported from: packages/core/src/workflows/handlers/control-flow.ts
package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// Execute Parallel Params
// ---------------------------------------------------------------------------

// ExecuteParallelParams holds parameters for executing parallel steps.
// TS equivalent: export interface ExecuteParallelParams extends ObservabilityContext
type ExecuteParallelParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	Entry               wf.StepFlowEntry
	PrevStep            wf.StepFlowEntry
	StepResults         map[string]wf.StepResult
	SerializedStepGraph []wf.SerializedStepFlowEntry
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
// ExecuteParallel
// ---------------------------------------------------------------------------

// ExecuteParallel executes parallel steps.
// TS equivalent: export async function executeParallel(engine, params)
func ExecuteParallel(engine DefaultEngine, params ExecuteParallelParams) (*wf.StepResult, error) {
	entry := params.Entry
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	prevOutput := engine.GetStepOutput(stepResultsToAnyMap(stepResults), &params.PrevStep)

	// Set up running status for each step
	for stepIndex, step := range entry.Steps {
		if step.Step == nil {
			continue
		}

		makeStepRunning := true
		if params.Restart != nil {
			if _, ok := params.Restart.ActiveStepsPath[step.Step.ID]; !ok {
				makeStepRunning = false
			}
		}
		if params.TimeTravel != nil && len(params.TimeTravel.ExecutionPath) > 0 {
			makeStepRunning = params.TimeTravel.Steps[0] == step.Step.ID
		}
		if !makeStepRunning {
			break
		}

		now := time.Now().UnixMilli()
		isResumedStep := params.Resume != nil && len(params.Resume.Steps) > 0 && params.Resume.Steps[0] == step.Step.ID

		sr := wf.StepResult{Status: wf.StepStatusRunning}
		if existing, ok := stepResults[step.Step.ID]; ok {
			sr = existing
			sr.Status = wf.StepStatusRunning
		}

		if isResumedStep {
			sr.ResumePayload = params.Resume.ResumePayload
			sr.ResumedAt = &now
		} else {
			sr.Payload = prevOutput
			sr.StartedAt = now
		}

		stepResults[step.Step.ID] = sr
		executionContext.ActiveStepsPath[step.Step.ID] = append(
			append([]int{}, executionContext.ExecutionPath...),
			stepIndex,
		)

		if params.PerStep {
			break
		}
	}

	// Shift time travel execution path
	if params.TimeTravel != nil && len(params.TimeTravel.ExecutionPath) > 0 {
		params.TimeTravel.ExecutionPath = params.TimeTravel.ExecutionPath[1:]
	}

	// Execute all steps in parallel
	type parallelResult struct {
		index  int
		result wf.StepResult
		err    error
	}

	var wg sync.WaitGroup
	results := make([]wf.StepResult, len(entry.Steps))
	errChan := make(chan error, len(entry.Steps))

	for i, step := range entry.Steps {
		if step.Step == nil {
			continue
		}

		currStepResult, exists := stepResults[step.Step.ID]
		if exists && currStepResult.Status != wf.StepStatusRunning {
			results[i] = currStepResult
			continue
		}
		if !exists && (params.PerStep || params.TimeTravel != nil) {
			results[i] = wf.StepResult{}
			continue
		}

		wg.Add(1)
		go func(idx int, s wf.StepFlowStepEntry) {
			defer wg.Done()

			stepExecResult, err := engine.ExecuteStepHandler(ExecuteStepParams{
				WorkflowID:          params.WorkflowID,
				RunID:               params.RunID,
				ResourceID:          params.ResourceID,
				Step:                s.Step,
				PrevOutput:          prevOutput,
				StepResults:         stepResults,
				SerializedStepGraph: params.SerializedStepGraph,
				Restart:             params.Restart,
				TimeTravel:          params.TimeTravel,
				Resume:              params.Resume,
				ExecutionContext: &wf.ExecutionContext{
					ActiveStepsPath:   executionContext.ActiveStepsPath,
					WorkflowID:        params.WorkflowID,
					RunID:             params.RunID,
					ExecutionPath:     append(append([]int{}, executionContext.ExecutionPath...), idx),
					StepExecutionPath: executionContext.StepExecutionPath,
					SuspendedPaths:    executionContext.SuspendedPaths,
					ResumeLabels:      executionContext.ResumeLabels,
					RetryConfig:       executionContext.RetryConfig,
					State:             executionContext.State,
					TracingIDs:        executionContext.TracingIDs,
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
				errChan <- err
				return
			}

			engine.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
			for k, v := range stepExecResult.StepResults {
				stepResults[k] = v
			}
			results[idx] = stepExecResult.Result
		}(i, step)
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return nil, err
	}

	// Determine overall result
	var execResults wf.StepResult

	// Check for failures
	var hasFailed *wf.StepResult
	var hasSuspended *wf.StepResult
	for i := range results {
		if results[i].Status == wf.StepStatusFailed {
			hasFailed = &results[i]
			break
		}
		if results[i].Status == wf.StepStatusSuspended {
			hasSuspended = &results[i]
		}
	}

	if hasFailed != nil {
		execResults = wf.StepResult{
			Status:   wf.StepStatusFailed,
			Error:    hasFailed.Error,
			Tripwire: hasFailed.Tripwire,
		}
	} else if hasSuspended != nil {
		execResults = wf.StepResult{
			Status:         wf.StepStatusSuspended,
			SuspendPayload: hasSuspended.SuspendPayload,
			SuspendOutput:  hasSuspended.SuspendOutput,
		}
	} else if params.AbortCtx != nil {
		select {
		case <-params.AbortCtx.Done():
			execResults = wf.StepResult{Status: "canceled"}
		default:
			// Build success output
			output := make(map[string]any)
			for i, result := range results {
				if result.Status == wf.StepStatusSuccess && i < len(entry.Steps) && entry.Steps[i].Step != nil {
					output[entry.Steps[i].Step.ID] = result.Output
				}
			}
			execResults = wf.StepResult{Status: wf.StepStatusSuccess, Output: output}
		}
	} else {
		output := make(map[string]any)
		for i, result := range results {
			if result.Status == wf.StepStatusSuccess && i < len(entry.Steps) && entry.Steps[i].Step != nil {
				output[entry.Steps[i].Step.ID] = result.Output
			}
		}
		execResults = wf.StepResult{Status: wf.StepStatusSuccess, Output: output}
	}

	return &execResults, nil
}

// ---------------------------------------------------------------------------
// Execute Conditional Params
// ---------------------------------------------------------------------------

// ExecuteConditionalParams holds parameters for executing conditional steps.
// TS equivalent: export interface ExecuteConditionalParams extends ObservabilityContext
type ExecuteConditionalParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	SerializedStepGraph []wf.SerializedStepFlowEntry
	Entry               wf.StepFlowEntry
	PrevOutput          any
	StepResults         map[string]wf.StepResult
	Resume              *ResumeParams
	Restart             *wf.RestartExecutionParams
	TimeTravel          *wf.TimeTravelExecutionParams
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
// ExecuteConditional
// ---------------------------------------------------------------------------

// ExecuteConditional executes conditional steps.
// TS equivalent: export async function executeConditional(engine, params)
func ExecuteConditional(engine DefaultEngine, params ExecuteConditionalParams) (*wf.StepResult, error) {
	entry := params.Entry
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	// Evaluate conditions
	truthyIndexes := make([]int, 0)
	for i, cond := range entry.Conditions {
		if cond == nil {
			continue
		}

		operationID := fmt.Sprintf("workflow.%s.conditional.%d", params.WorkflowID, i)

		condContext := &wf.ExecuteFunctionParams{
			RunID:          params.RunID,
			WorkflowID:     params.WorkflowID,
			Mastra:         engine.GetMastra(),
			RequestContext: params.RequestContext,
			InputData:      params.PrevOutput,
			State:          executionContext.State,
			RetryCount:     -1,
			GetInitData: func() any {
				if r, ok := stepResults["input"]; ok {
					return r.Output
				}
				return nil
			},
			GetStepResult: func(step any) any {
				return wf.GetStepResult(stepResults, step)
			},
			Bail:         func(_ any) {},
			Abort:        func() { if params.AbortCancel != nil { params.AbortCancel() } },
			PubSub:       params.PubSub,
			StreamFormat: executionContext.Format,
			Engine:       engine.GetEngineContext(),
			AbortCtx:     params.AbortCtx,
		}

		result, err := engine.EvaluateCondition(cond, i, condContext, operationID)
		if err != nil {
			logError(engine, "Error evaluating condition: %v", err)
			continue
		}
		if result != nil {
			truthyIndexes = append(truthyIndexes, *result)
		}
	}

	// Filter steps to run
	stepsToRun := make([]indexedStep, 0)
	for _, idx := range truthyIndexes {
		if idx < len(entry.Steps) {
			stepsToRun = append(stepsToRun, indexedStep{Index: idx, Step: entry.Steps[idx]})
		}
	}

	if params.PerStep || (params.TimeTravel != nil && len(params.TimeTravel.ExecutionPath) > 0) {
		filtered := make([]indexedStep, 0)
		for _, s := range stepsToRun {
			if s.Step.Step == nil {
				continue
			}
			if params.TimeTravel != nil && len(params.TimeTravel.ExecutionPath) > 0 {
				if params.TimeTravel.Steps[0] == s.Step.Step.ID {
					filtered = append(filtered, s)
					break
				}
			} else {
				if _, exists := stepResults[s.Step.Step.ID]; !exists {
					filtered = append(filtered, s)
					break
				}
			}
		}
		if len(filtered) > 0 {
			stepsToRun = filtered
		}
	}

	// Execute selected steps
	results := make([]wf.StepResult, len(stepsToRun))
	for ri, is := range stepsToRun {
		if is.Step.Step == nil {
			continue
		}

		currStepResult, exists := stepResults[is.Step.Step.ID]
		isRestartStep := false
		if params.Restart != nil {
			if _, ok := params.Restart.ActiveStepsPath[is.Step.Step.ID]; ok {
				isRestartStep = true
			}
		}

		if params.TimeTravel != nil && len(params.TimeTravel.ExecutionPath) > 0 {
			if exists && params.TimeTravel.Steps[0] != is.Step.Step.ID {
				results[ri] = currStepResult
				continue
			}
		}

		if exists && (currStepResult.Status == wf.StepStatusSuccess || currStepResult.Status == wf.StepStatusFailed) && !isRestartStep {
			results[ri] = currStepResult
			continue
		}

		stepExecResult, err := engine.ExecuteStepHandler(ExecuteStepParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			Step:                is.Step.Step,
			PrevOutput:          params.PrevOutput,
			StepResults:         stepResults,
			SerializedStepGraph: params.SerializedStepGraph,
			Resume:              params.Resume,
			Restart:             params.Restart,
			TimeTravel:          params.TimeTravel,
			ExecutionContext: &wf.ExecutionContext{
				WorkflowID:        params.WorkflowID,
				RunID:             params.RunID,
				ExecutionPath:     append(append([]int{}, executionContext.ExecutionPath...), is.Index),
				StepExecutionPath: executionContext.StepExecutionPath,
				ActiveStepsPath:   executionContext.ActiveStepsPath,
				SuspendedPaths:    executionContext.SuspendedPaths,
				ResumeLabels:      executionContext.ResumeLabels,
				RetryConfig:       executionContext.RetryConfig,
				State:             executionContext.State,
				TracingIDs:        executionContext.TracingIDs,
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

		engine.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
		for k, v := range stepExecResult.StepResults {
			stepResults[k] = v
		}
		results[ri] = stepExecResult.Result
	}

	// Determine overall result
	var execResults wf.StepResult
	var hasFailed *wf.StepResult
	var hasSuspended *wf.StepResult
	for i := range results {
		if results[i].Status == wf.StepStatusFailed {
			hasFailed = &results[i]
			break
		}
		if results[i].Status == wf.StepStatusSuspended {
			hasSuspended = &results[i]
		}
	}

	if hasFailed != nil {
		execResults = wf.StepResult{
			Status:   wf.StepStatusFailed,
			Error:    hasFailed.Error,
			Tripwire: hasFailed.Tripwire,
		}
	} else if hasSuspended != nil {
		execResults = wf.StepResult{
			Status:         wf.StepStatusSuspended,
			SuspendPayload: hasSuspended.SuspendPayload,
			SuspendOutput:  hasSuspended.SuspendOutput,
			SuspendedAt:    hasSuspended.SuspendedAt,
		}
	} else if params.AbortCtx != nil {
		select {
		case <-params.AbortCtx.Done():
			execResults = wf.StepResult{Status: "canceled"}
		default:
			output := make(map[string]any)
			for i, result := range results {
				if result.Status == wf.StepStatusSuccess && i < len(stepsToRun) && stepsToRun[i].Step.Step != nil {
					output[stepsToRun[i].Step.Step.ID] = result.Output
				}
			}
			execResults = wf.StepResult{Status: wf.StepStatusSuccess, Output: output}
		}
	} else {
		output := make(map[string]any)
		for i, result := range results {
			if result.Status == wf.StepStatusSuccess && i < len(stepsToRun) && stepsToRun[i].Step.Step != nil {
				output[stepsToRun[i].Step.Step.ID] = result.Output
			}
		}
		execResults = wf.StepResult{Status: wf.StepStatusSuccess, Output: output}
	}

	return &execResults, nil
}

// ---------------------------------------------------------------------------
// Execute Loop Params
// ---------------------------------------------------------------------------

// ExecuteLoopParams holds parameters for executing a loop.
// TS equivalent: export interface ExecuteLoopParams extends ObservabilityContext
type ExecuteLoopParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	Entry               wf.StepFlowEntry
	PrevStep            wf.StepFlowEntry
	PrevOutput          any
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
	SerializedStepGraph []wf.SerializedStepFlowEntry
	PerStep             bool
	Observability       *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// ExecuteLoop
// ---------------------------------------------------------------------------

// ExecuteLoop executes a loop step.
// TS equivalent: export async function executeLoop(engine, params)
func ExecuteLoop(engine DefaultEngine, params ExecuteLoopParams) (*wf.StepResult, error) {
	entry := params.Entry
	step := entry.Step
	condition := entry.LoopCondition
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	if step == nil || condition == nil {
		return nil, fmt.Errorf("loop entry missing step or condition")
	}

	isTrue := true
	prevIterationCount := 0
	if sr, ok := stepResults[step.ID]; ok && sr.Metadata != nil {
		if ic, ok := sr.Metadata["iterationCount"].(int); ok {
			prevIterationCount = ic
		}
	}
	iteration := 0
	if prevIterationCount > 0 {
		iteration = prevIterationCount - 1
	}

	prevPayload := params.PrevOutput
	if sr, ok := stepResults[step.ID]; ok && sr.Payload != nil {
		prevPayload = sr.Payload
	}

	result := wf.StepResult{Status: wf.StepStatusSuccess, Output: prevPayload}
	currentResume := params.Resume
	currentRestart := params.Restart
	currentTimeTravel := params.TimeTravel

	for {
		stepExecResult, err := engine.ExecuteStepHandler(ExecuteStepParams{
			WorkflowID:          params.WorkflowID,
			RunID:               params.RunID,
			ResourceID:          params.ResourceID,
			Step:                step,
			StepResults:         stepResults,
			ExecutionContext:    executionContext,
			Restart:             currentRestart,
			Resume:              currentResume,
			TimeTravel:          currentTimeTravel,
			PrevOutput:          result.Output,
			PubSub:              params.PubSub,
			AbortCtx:            params.AbortCtx,
			AbortCancel:         params.AbortCancel,
			RequestContext:      params.RequestContext,
			OutputWriter:        params.OutputWriter,
			DisableScorers:      params.DisableScorers,
			SerializedStepGraph: params.SerializedStepGraph,
			IterationCount:      iteration + 1,
			PerStep:             params.PerStep,
			Observability:       params.Observability,
		})
		if err != nil {
			return nil, err
		}

		engine.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
		for k, v := range stepExecResult.StepResults {
			stepResults[k] = v
		}
		result = stepExecResult.Result

		// Clear restart and time travel for next iteration
		currentRestart = nil
		currentTimeTravel = nil
		if currentResume != nil && result.Status != wf.StepStatusSuspended {
			currentResume = nil
		}

		if result.Status != wf.StepStatusSuccess {
			return &result, nil
		}

		// Evaluate loop condition
		condContext := &wf.ExecuteFunctionParams{
			WorkflowID:     params.WorkflowID,
			RunID:          params.RunID,
			Mastra:         engine.GetMastra(),
			RequestContext: params.RequestContext,
			InputData:      result.Output,
			State:          executionContext.State,
			RetryCount:     -1,
			GetInitData: func() any {
				if r, ok := stepResults["input"]; ok {
					return r.Output
				}
				return nil
			},
			GetStepResult: func(s any) any {
				return wf.GetStepResult(stepResults, s)
			},
			Bail:         func(_ any) {},
			Abort:        func() { if params.AbortCancel != nil { params.AbortCancel() } },
			PubSub:       params.PubSub,
			StreamFormat: executionContext.Format,
			Engine:       engine.GetEngineContext(),
			AbortCtx:     params.AbortCtx,
		}

		condResult, err := condition(condContext, iteration+1)
		if err != nil {
			return nil, fmt.Errorf("loop condition evaluation failed: %w", err)
		}
		isTrue = condResult

		iteration++

		// Check loop continuation based on loop type
		if entry.LoopKind == wf.LoopTypeDoWhile && !isTrue {
			break
		}
		if entry.LoopKind == wf.LoopTypeDoUntil && isTrue {
			break
		}
	}

	return &result, nil
}

// ---------------------------------------------------------------------------
// Execute Foreach Params
// ---------------------------------------------------------------------------

// ExecuteForeachParams holds parameters for executing a foreach.
// TS equivalent: export interface ExecuteForeachParams extends ObservabilityContext
type ExecuteForeachParams struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	Entry               wf.StepFlowEntry
	PrevStep            wf.StepFlowEntry
	PrevOutput          any
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
	SerializedStepGraph []wf.SerializedStepFlowEntry
	PerStep             bool
	Observability       *obstypes.ObservabilityContext
}

// ---------------------------------------------------------------------------
// ExecuteForeach
// ---------------------------------------------------------------------------

// ExecuteForeach executes a foreach step.
// TS equivalent: export async function executeForeach(engine, params)
func ExecuteForeach(engine DefaultEngine, params ExecuteForeachParams) (*wf.StepResult, error) {
	entry := params.Entry
	step := entry.Step
	stepResults := params.StepResults
	executionContext := params.ExecutionContext

	if step == nil {
		return nil, fmt.Errorf("foreach entry missing step")
	}

	concurrency := 1
	if entry.ForeachOpts != nil && entry.ForeachOpts.Concurrency > 0 {
		concurrency = entry.ForeachOpts.Concurrency
	}

	// Convert prevOutput to a slice
	items, ok := params.PrevOutput.([]any)
	if !ok {
		return nil, fmt.Errorf("foreach expects array input, got %T", params.PrevOutput)
	}

	now := time.Now().UnixMilli()
	isResumedStep := params.Resume != nil && len(params.Resume.Steps) > 0 && params.Resume.Steps[0] == step.ID

	stepInfo := wf.StepResult{}
	if existing, ok := stepResults[step.ID]; ok {
		stepInfo = existing
	}
	if isResumedStep {
		stepInfo.ResumePayload = params.Resume.ResumePayload
		stepInfo.ResumedAt = &now
	} else {
		stepInfo.Payload = params.PrevOutput
		stepInfo.StartedAt = now
	}

	// Get previous foreach output for resume
	var prevForeachOutput []wf.StepResult
	if sr, ok := stepResults[step.ID]; ok && sr.Status == wf.StepStatusSuspended {
		if sp, ok := sr.SuspendPayload.(map[string]any); ok {
			if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
				if fo, ok := meta["foreachOutput"].([]any); ok {
					prevForeachOutput = make([]wf.StepResult, len(fo))
					for i, item := range fo {
						if sr, ok := item.(wf.StepResult); ok {
							prevForeachOutput[i] = sr
						}
					}
				}
			}
		}
	}
	if prevForeachOutput == nil {
		prevForeachOutput = make([]wf.StepResult, len(items))
	}

	// Emit start event
	_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
		Type:  "watch",
		RunID: params.RunID,
		Data: map[string]any{
			"type": "workflow-step-start",
			"payload": map[string]any{
				"id":     step.ID,
				"status": "running",
			},
		},
	})

	results := make([]any, len(items))
	foreachIndexObj := make(map[int]wf.StepResult)
	completedCount := 0

	// Process items in batches of concurrency
	for i := 0; i < len(items); i += concurrency {
		end := i + concurrency
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]

		batchResults := make([]wf.StepResult, len(batch))
		for j, item := range batch {
			k := i + j

			// Check if previous result exists
			if k < len(prevForeachOutput) {
				prev := prevForeachOutput[k]
				if prev.Status == wf.StepStatusSuccess {
					batchResults[j] = prev
					continue
				}
				if prev.Status == wf.StepStatusSuspended && params.Resume != nil {
					if params.Resume.ForEachIndex != nil && *params.Resume.ForEachIndex != k {
						batchResults[j] = prev
						continue
					}
				}
			}

			var resumeToUse *ResumeParams
			if params.Resume != nil && params.Resume.ForEachIndex != nil {
				if *params.Resume.ForEachIndex == k {
					resumeToUse = params.Resume
				}
			} else if params.Resume != nil {
				resumeToUse = params.Resume
			}

			foreachIndex := k
			stepExecResult, err := engine.ExecuteStepHandler(ExecuteStepParams{
				WorkflowID:       params.WorkflowID,
				RunID:            params.RunID,
				ResourceID:       params.ResourceID,
				Step:             step,
				StepResults:      stepResults,
				Restart:          params.Restart,
				TimeTravel:       params.TimeTravel,
				ExecutionContext: &wf.ExecutionContext{
					WorkflowID:        executionContext.WorkflowID,
					RunID:             executionContext.RunID,
					ExecutionPath:     executionContext.ExecutionPath,
					StepExecutionPath: executionContext.StepExecutionPath,
					ActiveStepsPath:   executionContext.ActiveStepsPath,
					ForeachIndex:      &foreachIndex,
					SuspendedPaths:    executionContext.SuspendedPaths,
					ResumeLabels:      executionContext.ResumeLabels,
					RetryConfig:       executionContext.RetryConfig,
					Format:            executionContext.Format,
					State:             executionContext.State,
					TracingIDs:        executionContext.TracingIDs,
				},
				Resume:              resumeToUse,
				PrevOutput:          item,
				PubSub:              params.PubSub,
				AbortCtx:            params.AbortCtx,
				AbortCancel:         params.AbortCancel,
				RequestContext:      params.RequestContext,
				SkipEmits:           true,
				OutputWriter:        params.OutputWriter,
				DisableScorers:      params.DisableScorers,
				SerializedStepGraph: params.SerializedStepGraph,
				PerStep:             params.PerStep,
				Observability:       params.Observability,
			})
			if err != nil {
				return nil, err
			}

			engine.ApplyMutableContext(executionContext, stepExecResult.MutableContext)
			for k, v := range stepExecResult.StepResults {
				stepResults[k] = v
			}
			batchResults[j] = stepExecResult.Result
		}

		// Process batch results
		for resultIndex, result := range batchResults {
			globalIndex := i + resultIndex

			if result.Status != wf.StepStatusSuccess {
				if result.Status == wf.StepStatusSuspended {
					foreachIndexObj[globalIndex] = result
				} else {
					completedCount++
					return &result, nil
				}
			} else {
				completedCount++
			}

			if result.Output != nil {
				results[globalIndex] = result.Output
			}

			if globalIndex < len(prevForeachOutput) {
				prevForeachOutput[globalIndex] = wf.StepResult{
					Status:         result.Status,
					Output:         result.Output,
					SuspendPayload: nil, // Clear suspend payload
				}
			}
		}

		// Check for suspended items in this batch
		if len(foreachIndexObj) > 0 {
			// Find first suspended index
			var firstSuspendedIdx int
			for idx := range foreachIndexObj {
				firstSuspendedIdx = idx
				break
			}

			suspendedResult := foreachIndexObj[firstSuspendedIdx]
			executionContext.SuspendedPaths[step.ID] = executionContext.ExecutionPath

			suspendedAt := time.Now().UnixMilli()
			return &wf.StepResult{
				Status:      wf.StepStatusSuspended,
				Payload:     stepInfo.Payload,
				StartedAt:   stepInfo.StartedAt,
				SuspendedAt: &suspendedAt,
				SuspendPayload: map[string]any{
					"__workflow_meta": map[string]any{
						"foreachIndex":  firstSuspendedIdx,
						"foreachOutput": prevForeachOutput,
						"resumeLabels":  executionContext.ResumeLabels,
					},
				},
				SuspendOutput: suspendedResult.SuspendOutput,
			}, nil
		}
	}

	// Emit completion events
	_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
		Type:  "watch",
		RunID: params.RunID,
		Data: map[string]any{
			"type": "workflow-step-result",
			"payload": map[string]any{
				"id":      step.ID,
				"status":  "success",
				"output":  results,
				"endedAt": time.Now().UnixMilli(),
			},
		},
	})

	_ = params.PubSub.Publish(fmt.Sprintf("workflow.events.v2.%s", params.RunID), events.PublishEvent{
		Type:  "watch",
		RunID: params.RunID,
		Data: map[string]any{
			"type":    "workflow-step-finish",
			"payload": map[string]any{"id": step.ID, "metadata": map[string]any{}},
		},
	})

	endedAt := time.Now().UnixMilli()
	return &wf.StepResult{
		Status:    wf.StepStatusSuccess,
		Output:    results,
		Payload:   stepInfo.Payload,
		StartedAt: stepInfo.StartedAt,
		EndedAt:   endedAt,
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

type indexedStep struct {
	Index int
	Step  wf.StepFlowStepEntry
}
