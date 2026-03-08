// Ported from: packages/core/src/workflows/evented/workflow-event-processor/sleep.ts
package eventprocessor

import (
	"fmt"
	"time"

	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// ProcessWorkflowWaitForEvent
// ---------------------------------------------------------------------------

// ProcessWorkflowWaitForEvent handles a user-emitted event that unblocks
// a workflow waiting on that event name.
// TS equivalent: export async function processWorkflowWaitForEvent(workflowData, {pubsub, eventName, currentState})
func ProcessWorkflowWaitForEvent(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	eventName string,
	currentState *evented.WorkflowRunState,
) error {
	if currentState == nil {
		return nil
	}

	executionPath, ok := currentState.WaitingPaths[eventName]
	if !ok {
		return nil
	}

	currentStep := GetStep(args.Workflow, executionPath)
	stepID := "input"
	if currentStep != nil {
		stepID = currentStep.ID
	}

	var prevOutput any
	if ctx, ok := currentState.Context[stepID].(map[string]any); ok {
		prevOutput = ctx["payload"]
	}

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.step.run",
		"runId": args.RunID,
		"data": map[string]any{
			"workflowId":      args.WorkflowID,
			"runId":           args.RunID,
			"executionPath":   executionPath,
			"resumeSteps":     []string{},
			"resumeData":      args.ResumeData,
			"parentWorkflow":  args.ParentWorkflow,
			"stepResults":     currentState.Context,
			"prevResult":      map[string]any{"status": "success", "output": prevOutput},
			"activeSteps":     []any{},
			"requestContext":  currentState.RequestContext,
			"perStep":         args.PerStep,
		},
	})
}

// ---------------------------------------------------------------------------
// ProcessWorkflowSleep
// ---------------------------------------------------------------------------

// ProcessWorkflowSleep handles a sleep step by resolving the duration,
// emitting watch events, then scheduling the next step after the delay.
// TS equivalent: export async function processWorkflowSleep(args, {pubsub, stepExecutor, step})
func ProcessWorkflowSleep(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	stepExecutor *evented.StepExecutor,
	step *wf.StepFlowEntry,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeSleep {
		return fmt.Errorf("expected sleep step, got %v", step)
	}

	startedAt := time.Now().UnixMilli()

	// Emit workflow-step-waiting watch event
	var prevOutput any
	if args.PrevResult != nil && args.PrevResult["status"] == "success" {
		prevOutput = args.PrevResult["output"]
	}

	_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
		"type":  "watch",
		"runId": args.RunID,
		"data": map[string]any{
			"type": "workflow-step-waiting",
			"payload": map[string]any{
				"id":        step.ID,
				"status":    "waiting",
				"payload":   prevOutput,
				"startedAt": startedAt,
			},
		},
	})

	// Resolve sleep duration
	duration := stepExecutor.ResolveSleep(evented.ResolveSleepParams{
		WorkflowID: args.WorkflowID,
		Step:       step,
		RunID:      args.RunID,
		Input:      prevOutput,
		ResumeData: args.ResumeData,
	})

	if duration < 0 {
		duration = 0
	}

	// Schedule the next step after the sleep duration
	go func() {
		time.Sleep(time.Duration(duration) * time.Millisecond)

		// Emit step result
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-result",
				"payload": map[string]any{
					"id":        step.ID,
					"status":    "success",
					"payload":   prevOutput,
					"output":    prevOutput,
					"startedAt": startedAt,
					"endedAt":   time.Now().UnixMilli(),
				},
			},
		})

		// Emit step finish
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-finish",
				"payload": map[string]any{
					"id":       step.ID,
					"metadata": map[string]any{},
				},
			},
		})

		// Advance to next step
		nextPath := advanceExecutionPath(args.ExecutionPath)
		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":      args.WorkflowID,
				"runId":           args.RunID,
				"executionPath":   nextPath,
				"resumeSteps":     args.ResumeSteps,
				"timeTravel":      args.TimeTravel,
				"stepResults":     args.StepResults,
				"prevResult":      args.PrevResult,
				"resumeData":      args.ResumeData,
				"parentWorkflow":  args.ParentWorkflow,
				"activeSteps":     args.ActiveSteps,
				"requestContext":  args.RequestContext,
				"perStep":         args.PerStep,
			},
		})
	}()

	return nil
}

// ---------------------------------------------------------------------------
// ProcessWorkflowSleepUntil
// ---------------------------------------------------------------------------

// ProcessWorkflowSleepUntil handles a sleepUntil step by resolving the target date,
// computing the duration, and scheduling the next step.
// TS equivalent: export async function processWorkflowSleepUntil(args, {pubsub, stepExecutor, step})
func ProcessWorkflowSleepUntil(
	args *ProcessorArgs,
	pubsub evented.PubSub,
	stepExecutor *evented.StepExecutor,
	step *wf.StepFlowEntry,
) error {
	if step == nil || step.Type != wf.StepFlowEntryTypeSleepUntil {
		return fmt.Errorf("expected sleepUntil step, got %v", step)
	}

	startedAt := time.Now().UnixMilli()

	var prevOutput any
	if args.PrevResult != nil && args.PrevResult["status"] == "success" {
		prevOutput = args.PrevResult["output"]
	}

	// Resolve sleep-until duration
	duration := stepExecutor.ResolveSleepUntil(evented.ResolveSleepUntilParams{
		WorkflowID: args.WorkflowID,
		Step:       step,
		RunID:      args.RunID,
		Input:      prevOutput,
		ResumeData: args.ResumeData,
	})

	// Emit workflow-step-waiting watch event
	_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
		"type":  "watch",
		"runId": args.RunID,
		"data": map[string]any{
			"type": "workflow-step-waiting",
			"payload": map[string]any{
				"id":        step.ID,
				"status":    "waiting",
				"payload":   prevOutput,
				"startedAt": startedAt,
			},
		},
	})

	if duration < 0 {
		duration = 0
	}

	// Schedule the next step after the sleep-until duration
	go func() {
		time.Sleep(time.Duration(duration) * time.Millisecond)

		// Emit step result
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-result",
				"payload": map[string]any{
					"id":        step.ID,
					"status":    "success",
					"payload":   prevOutput,
					"output":    prevOutput,
					"startedAt": startedAt,
					"endedAt":   time.Now().UnixMilli(),
				},
			},
		})

		// Emit step finish
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-finish",
				"payload": map[string]any{
					"id":       step.ID,
					"metadata": map[string]any{},
				},
			},
		})

		// Advance to next step
		nextPath := advanceExecutionPath(args.ExecutionPath)
		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":      args.WorkflowID,
				"runId":           args.RunID,
				"executionPath":   nextPath,
				"resumeSteps":     args.ResumeSteps,
				"timeTravel":      args.TimeTravel,
				"stepResults":     args.StepResults,
				"prevResult":      args.PrevResult,
				"resumeData":      args.ResumeData,
				"parentWorkflow":  args.ParentWorkflow,
				"activeSteps":     args.ActiveSteps,
				"requestContext":  args.RequestContext,
				"perStep":         args.PerStep,
			},
		})
	}()

	return nil
}

// advanceExecutionPath increments the last element of the execution path.
// TS equivalent: executionPath.slice(0, -1).concat([executionPath[executionPath.length - 1]! + 1])
func advanceExecutionPath(path []int) []int {
	if len(path) == 0 {
		return []int{1}
	}
	result := make([]int, len(path))
	copy(result, path)
	result[len(result)-1]++
	return result
}
