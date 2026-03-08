// Ported from: packages/core/src/workflows/evented/workflow-event-processor/loop.ts
package eventprocessor

import (
	"fmt"
	"time"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// ProcessWorkflowLoop
// ---------------------------------------------------------------------------

// ProcessWorkflowLoop handles loop step execution by evaluating the loop condition
// and either continuing the loop or advancing to the next step.
// TS equivalent: export async function processWorkflowLoop(args, {pubsub, stepExecutor, step, stepResult})
func ProcessWorkflowLoop(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	stepExecutor *evented.StepExecutor,
	step *wf.StepFlowEntry,
	stepResult map[string]any,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeLoop {
		return fmt.Errorf("expected loop step, got %v", step)
	}

	// Get current state from stepResult, stepResults or passed state
	currentState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResult:  stepResult,
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Get iteration count from step results metadata
	prevIterationCount := 0
	if step.Step != nil {
		if sr, ok := args.StepResults[step.Step.ID]; ok {
			if srMap, ok := sr.(map[string]any); ok {
				if meta, ok := srMap["metadata"].(map[string]any); ok {
					if ic, ok := meta["iterationCount"].(float64); ok {
						prevIterationCount = int(ic)
					} else if ic, ok := meta["iterationCount"].(int); ok {
						prevIterationCount = ic
					}
				}
			}
		}
	}
	iterationCount := prevIterationCount + 1

	// Determine input for condition evaluation
	var conditionInput any
	if stepResult != nil && stepResult["status"] == "success" {
		conditionInput = stepResult["output"]
	}

	// Evaluate loop condition
	loopCondition, err := stepExecutor.EvaluateConditions(evented.EvaluateConditionsParams{
		WorkflowID: args.WorkflowID,
		Step:       step,
		RunID:      args.RunID,
		Input:      conditionInput,
		ResumeData: args.ResumeData,
		State:      currentState,
		RetryCount: args.RetryCount,
	})
	_ = iterationCount // Used for evaluateCondition internally
	if err != nil {
		return err
	}

	conditionResult := len(loopCondition) > 0

	loopType := step.LoopKind
	if loopType == "" {
		loopType = "dowhile"
	}

	if loopType == "dountil" {
		if conditionResult {
			// Condition met - end the loop, advance to next step
			return pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.end",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow":  args.ParentWorkflow,
					"workflowId":      args.WorkflowID,
					"runId":           args.RunID,
					"executionPath":   args.ExecutionPath,
					"resumeSteps":     args.ResumeSteps,
					"stepResults":     args.StepResults,
					"prevResult":      stepResult,
					"resumeData":      args.ResumeData,
					"activeSteps":     args.ActiveSteps,
					"requestContext":  args.RequestContext,
					"perStep":         args.PerStep,
					"state":           currentState,
					"outputOptions":   args.OutputOptions,
				},
			})
		}
		// Condition not met - continue loop
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"parentWorkflow":  args.ParentWorkflow,
				"workflowId":      args.WorkflowID,
				"runId":           args.RunID,
				"executionPath":   args.ExecutionPath,
				"resumeSteps":     args.ResumeSteps,
				"stepResults":     args.StepResults,
				"state":           currentState,
				"outputOptions":   args.OutputOptions,
				"prevResult":      stepResult,
				"resumeData":      args.ResumeData,
				"activeSteps":     args.ActiveSteps,
				"requestContext":  args.RequestContext,
				"retryCount":      args.RetryCount,
				"perStep":         args.PerStep,
			},
		})
	}

	// dowhile
	if conditionResult {
		// Condition still true - continue loop
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"parentWorkflow":  args.ParentWorkflow,
				"workflowId":      args.WorkflowID,
				"runId":           args.RunID,
				"executionPath":   args.ExecutionPath,
				"resumeSteps":     args.ResumeSteps,
				"stepResults":     args.StepResults,
				"prevResult":      stepResult,
				"resumeData":      args.ResumeData,
				"activeSteps":     args.ActiveSteps,
				"requestContext":  args.RequestContext,
				"retryCount":      args.RetryCount,
				"perStep":         args.PerStep,
				"state":           currentState,
				"outputOptions":   args.OutputOptions,
			},
		})
	}
	// Condition false - end the loop
	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.step.end",
		"runId": args.RunID,
		"data": map[string]any{
			"parentWorkflow":  args.ParentWorkflow,
			"workflowId":      args.WorkflowID,
			"runId":           args.RunID,
			"executionPath":   args.ExecutionPath,
			"resumeSteps":     args.ResumeSteps,
			"stepResults":     args.StepResults,
			"prevResult":      stepResult,
			"resumeData":      args.ResumeData,
			"activeSteps":     args.ActiveSteps,
			"requestContext":  args.RequestContext,
			"perStep":         args.PerStep,
			"state":           currentState,
			"outputOptions":   args.OutputOptions,
		},
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowForEach
// ---------------------------------------------------------------------------

// ProcessWorkflowForEach handles foreach step execution with concurrency control,
// suspend/resume support, and iteration tracking.
// TS equivalent: export async function processWorkflowForEach(args, {pubsub, mastra, step})
func ProcessWorkflowForEach(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	mastra evented.Mastra,
	step *wf.StepFlowEntry,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeForeach {
		return fmt.Errorf("expected foreach step, got %v", step)
	}

	// Get current state
	currentState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Get current result for this foreach step
	var currentResult map[string]any
	if step.Step != nil {
		if sr, ok := args.StepResults[step.Step.ID]; ok {
			if srMap, ok := sr.(map[string]any); ok {
				currentResult = srMap
			}
		}
	}

	// Determine how many iterations have been completed
	var currentOutput []any
	if currentResult != nil {
		if out, ok := currentResult["output"].([]any); ok {
			currentOutput = out
		}
	}
	idx := len(currentOutput)

	// Get target length from prevResult
	targetLen := 0
	if args.PrevResult != nil && args.PrevResult["status"] == "success" {
		if out, ok := args.PrevResult["output"].([]any); ok {
			targetLen = len(out)
		}
	}

	// Handle resume with forEachIndex
	if args.ForEachIndex != nil && len(args.ResumeSteps) > 0 && idx > 0 {
		foreachIdx := *args.ForEachIndex

		// Validate index bounds
		if currentOutput == nil || foreachIdx < 0 || foreachIdx >= len(currentOutput) {
			return pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.fail",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow":  args.ParentWorkflow,
					"workflowId":      args.WorkflowID,
					"runId":           args.RunID,
					"executionPath":   args.ExecutionPath,
					"resumeSteps":     args.ResumeSteps,
					"stepResults":     args.StepResults,
					"prevResult":      map[string]any{"status": "failed", "error": fmt.Sprintf("Invalid forEachIndex %d", foreachIdx)},
					"activeSteps":     args.ActiveSteps,
					"requestContext":  args.RequestContext,
					"state":           currentState,
					"outputOptions":   args.OutputOptions,
				},
			})
		}

		// Check if target iteration is suspended
		iterationResult := currentOutput[foreachIdx]
		isSuspended := false
		if irMap, ok := iterationResult.(map[string]any); ok {
			isSuspended = irMap["status"] == "suspended"
		}
		if iterationResult == nil || isSuspended {
			isNestedWorkflow := step.Step != nil && step.Step.Component == "WORKFLOW"
			iterationPrevResult := args.PrevResult
			if isNestedWorkflow && args.PrevResult["status"] == "success" {
				if arr, ok := args.PrevResult["output"].([]any); ok && foreachIdx < len(arr) {
					iterationPrevResult = map[string]any{"status": "success", "output": arr[foreachIdx]}
				}
			}

			return pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.run",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow":  args.ParentWorkflow,
					"workflowId":      args.WorkflowID,
					"runId":           args.RunID,
					"executionPath":   []int{args.ExecutionPath[0], foreachIdx},
					"resumeSteps":     args.ResumeSteps,
					"timeTravel":      args.TimeTravel,
					"stepResults":     args.StepResults,
					"prevResult":      iterationPrevResult,
					"resumeData":      args.ResumeData,
					"activeSteps":     args.ActiveSteps,
					"requestContext":  args.RequestContext,
					"perStep":         args.PerStep,
					"state":           currentState,
					"outputOptions":   args.OutputOptions,
				},
			})
		}

		// Target iteration is already complete - check for remaining pending iterations
		pendingCount := 0
		for _, r := range currentOutput {
			if r == nil {
				pendingCount++
			} else if rMap, ok := r.(map[string]any); ok && rMap["status"] == "suspended" {
				pendingCount++
			}
		}

		if pendingCount > 0 {
			// Re-suspend with collected resume labels
			suspendMeta := map[string]any{
				"foreachIndex": foreachIdx,
			}

			return pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.end",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow": args.ParentWorkflow,
					"workflowId":     args.WorkflowID,
					"runId":          args.RunID,
					"executionPath":  args.ExecutionPath,
					"resumeSteps":    args.ResumeSteps,
					"stepResults": mergeStepResults(args.StepResults, step.Step.ID, map[string]any{
						"status":         "suspended",
						"output":         currentOutput,
						"suspendedAt":    time.Now().UnixMilli(),
						"suspendPayload": map[string]any{"__workflow_meta": suspendMeta},
					}),
					"prevResult": map[string]any{
						"status":         "suspended",
						"output":         currentOutput,
						"suspendPayload": map[string]any{"__workflow_meta": suspendMeta},
						"payload":        currentResult["payload"],
						"startedAt":      currentResult["startedAt"],
						"suspendedAt":    time.Now().UnixMilli(),
					},
					"activeSteps":    args.ActiveSteps,
					"requestContext": args.RequestContext,
					"state":          currentState,
					"outputOptions":  args.OutputOptions,
				},
			})
		}

		return nil
	}

	// Handle bulk resume (resumeData provided but no forEachIndex)
	if args.ResumeData != nil && args.ForEachIndex == nil && currentOutput != nil && len(currentOutput) > 0 {
		suspendedIndices := make([]int, 0)
		for i, r := range currentOutput {
			if rMap, ok := r.(map[string]any); ok && rMap["status"] == "suspended" {
				suspendedIndices = append(suspendedIndices, i)
			}
		}

		if len(suspendedIndices) > 0 {
			concurrency := 1
			if step.ForeachOpts != nil && step.ForeachOpts.Concurrency > 0 {
				concurrency = step.ForeachOpts.Concurrency
			}
			indicesToResume := suspendedIndices
			if len(indicesToResume) > concurrency {
				indicesToResume = indicesToResume[:concurrency]
			}

			// Mark suspended iterations as pending.
			// Reset suspended iterations to "pending" state before re-running them.
			//
			// Why PendingMarker instead of null?
			// The storage merge logic treats null as "keep existing value" to prevent
			// completed results from being overwritten by concurrent iterations that
			// haven't finished yet. But when resuming, we need to force-reset the
			// suspended result to null so the iteration can run fresh.
			//
			// PendingMarker ({ __mastra_pending__: true }) tells the storage layer
			// "force this to null, don't preserve the existing suspended result."
			// See inmemory.ts updateWorkflowResults for the merge logic.
			// TS: const workflowsStore = await mastra.getStorage()?.getStore('workflows');
			// TS: const updatedOutput = [...currentResult.output];
			// TS: for (const suspIdx of indicesToResume) { updatedOutput[suspIdx] = createPendingMarker(); }
			// TS: await workflowsStore?.updateWorkflowResults({ workflowName, runId, stepId, result: {...currentResult, output: updatedOutput}, requestContext });
			if mastra != nil {
				storage := mastra.GetStorage()
				if storage != nil {
					store, storeErr := storage.GetStore("workflows")
					if storeErr == nil && store != nil && step.Step != nil {
						updatedOutput := make([]any, len(currentOutput))
						copy(updatedOutput, currentOutput)
						for _, suspIdx := range indicesToResume {
							updatedOutput[suspIdx] = evented.CreatePendingMarker()
						}
						updatedResult := make(map[string]any)
						for k, v := range currentResult {
							updatedResult[k] = v
						}
						updatedResult["output"] = updatedOutput
						_, _ = store.UpdateWorkflowResults(evented.UpdateResultsParams{
							WorkflowName:   args.WorkflowID,
							RunID:          args.RunID,
							StepID:         step.Step.ID,
							Result:         updatedResult,
							RequestContext: args.RequestContext,
						})
					}
				}
			}

			isNestedWorkflow := step.Step != nil && step.Step.Component == "WORKFLOW"

			for _, suspIdx := range indicesToResume {
				iterationPrevResult := args.PrevResult
				if isNestedWorkflow && args.PrevResult["status"] == "success" {
					if arr, ok := args.PrevResult["output"].([]any); ok && suspIdx < len(arr) {
						iterationPrevResult = map[string]any{"status": "success", "output": arr[suspIdx]}
					}
				}

				_ = pubsub.Publish("workflows", map[string]any{
					"type":  "workflow.step.run",
					"runId": args.RunID,
					"data": map[string]any{
						"parentWorkflow":  args.ParentWorkflow,
						"workflowId":      args.WorkflowID,
						"runId":           args.RunID,
						"executionPath":   []int{args.ExecutionPath[0], suspIdx},
						"resumeSteps":     args.ResumeSteps,
						"timeTravel":      args.TimeTravel,
						"stepResults":     args.StepResults,
						"prevResult":      iterationPrevResult,
						"resumeData":      args.ResumeData,
						"activeSteps":     args.ActiveSteps,
						"requestContext":  args.RequestContext,
						"perStep":         args.PerStep,
						"state":           currentState,
						"outputOptions":   args.OutputOptions,
					},
				})
			}
			return nil
		}
	}

	// Check if all iterations are complete
	nonNullCount := 0
	for _, r := range currentOutput {
		if r != nil {
			nonNullCount++
		}
	}

	if idx >= targetLen && nonNullCount >= targetLen {
		// All iterations complete - advance to next step
		nextPath := advanceExecutionPath(args.ExecutionPath[:1])
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"parentWorkflow":  args.ParentWorkflow,
				"workflowId":      args.WorkflowID,
				"runId":           args.RunID,
				"executionPath":   nextPath,
				"resumeSteps":     args.ResumeSteps,
				"stepResults":     args.StepResults,
				"timeTravel":      args.TimeTravel,
				"prevResult":      currentResult,
				"resumeData":      nil,
				"activeSteps":     args.ActiveSteps,
				"requestContext":  args.RequestContext,
				"perStep":         args.PerStep,
				"state":           currentState,
				"outputOptions":   args.OutputOptions,
			},
		})
	} else if idx >= targetLen {
		// Wait for concurrent runs to fill null values
		return nil
	}

	isNestedWorkflow := step.Step != nil && step.Step.Component == "WORKFLOW"

	// First iteration - kick off up to concurrency
	if len(args.ExecutionPath) == 1 && idx == 0 {
		concurrency := 1
		if step.ForeachOpts != nil && step.ForeachOpts.Concurrency > 0 {
			concurrency = step.ForeachOpts.Concurrency
		}
		if concurrency > targetLen {
			concurrency = targetLen
		}

		// Initialize with null results.
		// On the first iteration, create a dummy result array of length `concurrency` filled with nils
		// and persist it to storage so concurrent iterations have a slot to write into.
		// TS: const dummyResult = Array.from({ length: concurrency }, () => null);
		// TS: await workflowsStore?.updateWorkflowResults({
		//   workflowName, runId, stepId: step.step.id,
		//   result: { status: 'success', output: dummyResult, startedAt: Date.now(), payload: prevResult?.output },
		//   requestContext,
		// });
		if mastra != nil && step.Step != nil {
			storage := mastra.GetStorage()
			if storage != nil {
				store, storeErr := storage.GetStore("workflows")
				if storeErr == nil && store != nil {
					dummyResult := make([]any, concurrency) // filled with nil values
					var payload any
					if args.PrevResult != nil {
						payload = args.PrevResult["output"]
					}
					_, _ = store.UpdateWorkflowResults(evented.UpdateResultsParams{
						WorkflowName: args.WorkflowID,
						RunID:        args.RunID,
						StepID:       step.Step.ID,
						Result: map[string]any{
							"status":    "success",
							"output":    dummyResult,
							"startedAt": time.Now().UnixMilli(),
							"payload":   payload,
						},
						RequestContext: args.RequestContext,
					})
				}
			}
		}

		for i := 0; i < concurrency; i++ {
			iterationPrevResult := args.PrevResult
			if isNestedWorkflow && args.PrevResult["status"] == "success" {
				if arr, ok := args.PrevResult["output"].([]any); ok && i < len(arr) {
					iterationPrevResult = map[string]any{"status": "success", "output": arr[i]}
				}
			}

			_ = pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.run",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow":  args.ParentWorkflow,
					"workflowId":      args.WorkflowID,
					"runId":           args.RunID,
					"executionPath":   []int{args.ExecutionPath[0], i},
					"resumeSteps":     args.ResumeSteps,
					"stepResults":     args.StepResults,
					"timeTravel":      args.TimeTravel,
					"prevResult":      iterationPrevResult,
					"resumeData":      args.ResumeData,
					"activeSteps":     args.ActiveSteps,
					"requestContext":  args.RequestContext,
					"perStep":         args.PerStep,
					"state":           currentState,
					"outputOptions":   args.OutputOptions,
				},
			})
		}

		return nil
	}

	// Subsequent iteration - append null and kick off next.
	// Append a nil slot to the output array and persist the updated result to storage
	// so the next concurrent iteration has a slot to write into.
	// TS: (currentResult as any).output.push(null);
	// TS: await workflowsStore?.updateWorkflowResults({
	//   workflowName, runId, stepId: step.step.id,
	//   result: { status: 'success', output: currentResult.output, startedAt: Date.now(), payload: prevResult?.output },
	//   requestContext,
	// });
	currentOutput = append(currentOutput, nil)
	if currentResult != nil {
		currentResult["output"] = currentOutput
	}
	if mastra != nil && step.Step != nil {
		storage := mastra.GetStorage()
		if storage != nil {
			store, storeErr := storage.GetStore("workflows")
			if storeErr == nil && store != nil {
				var payload any
				if args.PrevResult != nil {
					payload = args.PrevResult["output"]
				}
				_, _ = store.UpdateWorkflowResults(evented.UpdateResultsParams{
					WorkflowName: args.WorkflowID,
					RunID:        args.RunID,
					StepID:       step.Step.ID,
					Result: map[string]any{
						"status":    "success",
						"output":    currentOutput,
						"startedAt": time.Now().UnixMilli(),
						"payload":   payload,
					},
					RequestContext: args.RequestContext,
				})
			}
		}
	}

	iterationPrevResult := args.PrevResult
	if isNestedWorkflow && args.PrevResult["status"] == "success" {
		if arr, ok := args.PrevResult["output"].([]any); ok && idx < len(arr) {
			iterationPrevResult = map[string]any{"status": "success", "output": arr[idx]}
		}
	}

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.step.run",
		"runId": args.RunID,
		"data": map[string]any{
			"parentWorkflow":  args.ParentWorkflow,
			"workflowId":      args.WorkflowID,
			"runId":           args.RunID,
			"executionPath":   []int{args.ExecutionPath[0], idx},
			"resumeSteps":     args.ResumeSteps,
			"timeTravel":      args.TimeTravel,
			"stepResults":     args.StepResults,
			"prevResult":      iterationPrevResult,
			"resumeData":      args.ResumeData,
			"activeSteps":     args.ActiveSteps,
			"requestContext":  args.RequestContext,
			"perStep":         args.PerStep,
			"state":           currentState,
			"outputOptions":   args.OutputOptions,
		},
	})
}

// mergeStepResults creates a new step results map with the given step result merged in.
func mergeStepResults(stepResults map[string]any, stepID string, result map[string]any) map[string]any {
	merged := make(map[string]any)
	for k, v := range stepResults {
		merged[k] = v
	}
	merged[stepID] = result
	return merged
}
