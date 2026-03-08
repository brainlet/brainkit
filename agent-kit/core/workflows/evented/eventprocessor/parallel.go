// Ported from: packages/core/src/workflows/evented/workflow-event-processor/parallel.ts
package eventprocessor

import (
	"fmt"
	"sync"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// ProcessWorkflowParallel
// ---------------------------------------------------------------------------

// ProcessWorkflowParallel handles parallel step execution by publishing
// workflow.step.run events for each active step in the parallel block.
// TS equivalent: export async function processWorkflowParallel(args, {pubsub, step})
func ProcessWorkflowParallel(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	step *wf.StepFlowEntry,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeParallel {
		return fmt.Errorf("expected parallel step, got %v", step)
	}

	// Get current state from stepResults or passed state
	currentState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Mark active steps
	for i, nestedStep := range step.Steps {
		if nestedStep.Type == string(wf.StepFlowEntryTypeStep) && nestedStep.Step != nil {
			args.ActiveSteps[nestedStep.Step.ID] = true
			if args.PerStep {
				_ = i
				break
			}
		}
	}

	// Publish step.run for each active step in parallel
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for idx, nestedStep := range step.Steps {
		if nestedStep.Step == nil || !args.ActiveSteps[nestedStep.Step.ID] {
			continue
		}

		wg.Add(1)
		go func(stepIdx int) {
			defer wg.Done()

			err := pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.run",
				"runId": args.RunID,
				"data": map[string]any{
					"workflowId":      args.WorkflowID,
					"runId":           args.RunID,
					"executionPath":   append(args.ExecutionPath, stepIdx),
					"resumeSteps":     args.ResumeSteps,
					"stepResults":     args.StepResults,
					"prevResult":      args.PrevResult,
					"resumeData":      args.ResumeData,
					"timeTravel":      args.TimeTravel,
					"parentWorkflow":  args.ParentWorkflow,
					"activeSteps":     args.ActiveSteps,
					"requestContext":  args.RequestContext,
					"perStep":         args.PerStep,
					"state":           currentState,
					"outputOptions":   args.OutputOptions,
				},
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(idx)
	}

	wg.Wait()
	return firstErr
}

// ---------------------------------------------------------------------------
// ProcessWorkflowConditional
// ---------------------------------------------------------------------------

// ProcessWorkflowConditional handles conditional step execution by evaluating
// conditions and publishing workflow.step.run events for matching branches.
// TS equivalent: export async function processWorkflowConditional(args, {pubsub, stepExecutor, step})
func ProcessWorkflowConditional(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	stepExecutor *evented.StepExecutor,
	step *wf.StepFlowEntry,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeConditional {
		return fmt.Errorf("expected conditional step, got %v", step)
	}

	// Get current state from stepResults or passed state
	currentState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Evaluate conditions
	var inputForCondition any
	if args.PrevResult != nil && args.PrevResult["status"] == "success" {
		inputForCondition = args.PrevResult["output"]
	}

	// Convert step results from map[string]any to map[string]wf.StepResult
	// TODO: proper type conversion when types align
	stepResultsTyped := make(map[string]wf.StepResult)
	_ = stepResultsTyped

	idxs, err := stepExecutor.EvaluateConditions(evented.EvaluateConditionsParams{
		WorkflowID: args.WorkflowID,
		Step:       step,
		RunID:      args.RunID,
		Input:      inputForCondition,
		ResumeData: args.ResumeData,
		State:      currentState,
	})
	if err != nil {
		return err
	}

	// Build truthy indices map
	truthyIdxs := make(map[int]bool)
	for _, idx := range idxs {
		truthyIdxs[idx] = true
	}

	// If perStep, only run the first matching step
	if args.PerStep {
		for _, idx := range idxs {
			if idx < len(step.Steps) && step.Steps[idx].Step != nil {
				nestedStep := step.Steps[idx]
				args.ActiveSteps[nestedStep.Step.ID] = true

				return pubsub.Publish("workflows", map[string]any{
					"type":  "workflow.step.run",
					"runId": args.RunID,
					"data": map[string]any{
						"workflowId":      args.WorkflowID,
						"runId":           args.RunID,
						"executionPath":   append(args.ExecutionPath, idx),
						"resumeSteps":     args.ResumeSteps,
						"stepResults":     args.StepResults,
						"timeTravel":      args.TimeTravel,
						"prevResult":      args.PrevResult,
						"resumeData":      args.ResumeData,
						"parentWorkflow":  args.ParentWorkflow,
						"activeSteps":     args.ActiveSteps,
						"requestContext":  args.RequestContext,
						"perStep":         args.PerStep,
						"state":           currentState,
						"outputOptions":   args.OutputOptions,
					},
				})
			}
		}
		return nil
	}

	// Publish for all steps: run for truthy, end/skipped for falsy
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for idx, nestedStep := range step.Steps {
		wg.Add(1)
		go func(stepIdx int, ns wf.StepFlowStepEntry) {
			defer wg.Done()

			var err error
			if truthyIdxs[stepIdx] {
				if ns.Type == string(wf.StepFlowEntryTypeStep) && ns.Step != nil {
					args.ActiveSteps[ns.Step.ID] = true
				}
				err = pubsub.Publish("workflows", map[string]any{
					"type":  "workflow.step.run",
					"runId": args.RunID,
					"data": map[string]any{
						"workflowId":      args.WorkflowID,
						"runId":           args.RunID,
						"executionPath":   append(args.ExecutionPath, stepIdx),
						"resumeSteps":     args.ResumeSteps,
						"stepResults":     args.StepResults,
						"timeTravel":      args.TimeTravel,
						"prevResult":      args.PrevResult,
						"resumeData":      args.ResumeData,
						"parentWorkflow":  args.ParentWorkflow,
						"activeSteps":     args.ActiveSteps,
						"requestContext":  args.RequestContext,
						"perStep":         args.PerStep,
						"state":           currentState,
						"outputOptions":   args.OutputOptions,
					},
				})
			} else {
				err = pubsub.Publish("workflows", map[string]any{
					"type":  "workflow.step.end",
					"runId": args.RunID,
					"data": map[string]any{
						"workflowId":      args.WorkflowID,
						"runId":           args.RunID,
						"executionPath":   append(args.ExecutionPath, stepIdx),
						"resumeSteps":     args.ResumeSteps,
						"stepResults":     args.StepResults,
						"prevResult":      map[string]any{"status": "skipped"},
						"resumeData":      args.ResumeData,
						"parentWorkflow":  args.ParentWorkflow,
						"activeSteps":     args.ActiveSteps,
						"requestContext":  args.RequestContext,
						"perStep":         args.PerStep,
						"state":           currentState,
						"outputOptions":   args.OutputOptions,
					},
				})
			}

			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(idx, nestedStep)
	}

	wg.Wait()
	return firstErr
}
