// Ported from: packages/core/src/workflows/evented/execution-engine.ts
package evented

import (
	"errors"
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// EventedExecutionEngine
// ---------------------------------------------------------------------------

// EventedExecutionEngine is an execution engine that uses pub/sub events
// to drive workflow execution. It publishes workflow start/resume events
// and subscribes to finish events to collect results.
// TS equivalent: export class EventedExecutionEngine extends ExecutionEngine
type EventedExecutionEngine struct {
	wf.BaseExecutionEngine

	eventProcessor EventProcessor
}

// EventProcessor is the interface for the workflow event processor.
// TODO: Replace with actual WorkflowEventProcessor when index.go is ported.
type EventProcessor interface {
	RegisterMastra(mastra Mastra)
}

// NewEventedExecutionEngine creates a new EventedExecutionEngine.
func NewEventedExecutionEngine(mastra Mastra, eventProcessor EventProcessor, options *wf.ExecutionEngineOptions) *EventedExecutionEngine {
	var m wf.Mastra
	if mastra != nil {
		// TODO: proper type assertion when Mastra interfaces align
		m = nil
	}
	return &EventedExecutionEngine{
		BaseExecutionEngine: wf.NewBaseExecutionEngine(m, options),
		eventProcessor:      eventProcessor,
	}
}

// RegisterMastra registers the mastra instance with this engine and its event processor.
// TS equivalent: __registerMastra(mastra: Mastra)
func (e *EventedExecutionEngine) RegisterMastra(mastra Mastra) {
	// TODO: Call base RegisterMastra when type alignment is resolved
	e.eventProcessor.RegisterMastra(mastra)
}

// Execute executes a workflow run using the evented execution model.
// It publishes workflow start/resume events to the pub/sub system, then
// waits for a workflow finish event to collect the result.
//
// TS equivalent: async execute<TState, TInput, TOutput>(params): Promise<TOutput>
func (e *EventedExecutionEngine) Execute(params wf.ExecuteParams) (any, error) {
	mastra := e.GetMastra()
	if mastra == nil {
		return nil, errors.New("mastra instance not registered")
	}

	pubsub := mastra.PubSub()
	if pubsub == nil {
		return nil, errors.New("no Pubsub adapter configured on the Mastra instance")
	}

	// Set up result channel - MUST subscribe BEFORE publishing to avoid race condition
	resultCh := make(chan any, 1)
	errCh := make(chan error, 1)

	finishCb := events.SubscribeCallback(func(evt events.Event, ack events.AckFunc) {
		if evt.RunID != params.RunID {
			return
		}

		eventType := evt.Type
		if eventType == "workflow.end" || eventType == "workflow.fail" || eventType == "workflow.suspend" {
			data, _ := evt.Data.(map[string]any)

			// Re-hydrate serialized errors back to Error instances when workflow fails
			if eventType == "workflow.fail" {
				if sr, ok := data["stepResults"].(map[string]any); ok {
					data["stepResults"] = wf.HydrateSerializedStepErrors(sr)
				}
			}

			resultCh <- data
		}
	})

	// Subscribe to finish events first
	err := pubsub.Subscribe("workflows-finish", finishCb)
	if err != nil {
		log := e.GetLogger()
		if log != nil {
			log.Error(fmt.Sprintf("Failed to subscribe to workflows-finish: %v", err))
		}
		return nil, err
	}
	defer func() {
		_ = pubsub.Unsubscribe("workflows-finish", finishCb)
	}()

	// NOW publish the start/resume event
	if params.Resume != nil {
		// Resume workflow
		resumeState := params.InitialState
		if resumeState == nil {
			resumeState = map[string]any{}
		}

		err = pubsub.Publish("workflows", events.PublishEvent{
			Type:  "workflow.resume",
			RunID: params.RunID,
			Data: map[string]any{
				"workflowId":     params.WorkflowID,
				"runId":          params.RunID,
				"executionPath":  params.Resume.ResumePath,
				"stepResults":    params.Resume.StepResults,
				"resumeSteps":    params.Resume.Steps,
				"prevResult":     map[string]any{"status": "success", "output": getResumePayload(params)},
				"resumeData":     params.Resume.ResumePayload,
				"requestContext": serializeRequestContext(params.RequestContext),
				"format":         params.Format,
				"perStep":        params.PerStep,
				"initialState":   resumeState,
				"state":          resumeState,
				"outputOptions":  serializeOutputOptions(params.OutputOptions),
				"forEachIndex":   params.Resume.ForEachIndex,
			},
		})
	} else if params.TimeTravel != nil {
		// Time travel execution
		err = pubsub.Publish("workflows", events.PublishEvent{
			Type:  "workflow.start",
			RunID: params.RunID,
			Data: map[string]any{
				"workflowId":     params.WorkflowID,
				"runId":          params.RunID,
				"executionPath":  params.TimeTravel.ExecutionPath,
				"stepResults":    params.TimeTravel.StepResults,
				"timeTravel":     params.TimeTravel,
				"prevResult":     map[string]any{"status": "success", "output": getTimeTravelPayload(params)},
				"requestContext": serializeRequestContext(params.RequestContext),
				"format":         params.Format,
				"perStep":        params.PerStep,
			},
		})
	} else {
		// Fresh start
		err = pubsub.Publish("workflows", events.PublishEvent{
			Type:  "workflow.start",
			RunID: params.RunID,
			Data: map[string]any{
				"workflowId":     params.WorkflowID,
				"runId":          params.RunID,
				"prevResult":     map[string]any{"status": "success", "output": params.Input},
				"requestContext": serializeRequestContext(params.RequestContext),
				"format":         params.Format,
				"perStep":        params.PerStep,
				"initialState":   params.InitialState,
				"outputOptions":  serializeOutputOptions(params.OutputOptions),
			},
		})
	}

	if err != nil {
		return nil, fmt.Errorf("failed to publish workflow event: %w", err)
	}

	// Wait for workflow to complete
	var resultData map[string]any
	select {
	case data := <-resultCh:
		if d, ok := data.(map[string]any); ok {
			resultData = d
		} else {
			return nil, errors.New("unexpected result data type")
		}
	case err := <-errCh:
		return nil, err
	}

	if resultData == nil {
		return nil, errors.New("nil result data from workflow")
	}

	// Extract state from resultData
	finalState := resultData["state"]
	if finalState == nil {
		if sr, ok := resultData["stepResults"].(map[string]any); ok {
			finalState = sr["__state"]
		}
	}
	if finalState == nil {
		finalState = params.InitialState
	}
	if finalState == nil {
		finalState = map[string]any{}
	}

	// Strip __state from stepResults at top level
	stepResultsRaw, _ := resultData["stepResults"].(map[string]any)
	cleanStepResults := make(map[string]any)
	if stepResultsRaw != nil {
		for k, v := range stepResultsRaw {
			if k == "__state" {
				continue
			}
			cleanStepResults[k] = wf.CleanStepResult(v)
		}
	}

	// Build the final result
	prevResult, _ := resultData["prevResult"].(map[string]any)
	prevStatus, _ := prevResult["status"].(string)

	result := map[string]any{
		"steps": cleanStepResults,
	}

	switch prevStatus {
	case "failed":
		// Check for tripwire
		var tripwireData map[string]any
		for _, sr := range cleanStepResults {
			if srMap, ok := sr.(map[string]any); ok {
				if srMap["status"] == "failed" && srMap["tripwire"] != nil {
					tripwireData, _ = srMap["tripwire"].(map[string]any)
					break
				}
			}
		}

		if tripwireData != nil {
			result["status"] = "tripwire"
			result["tripwire"] = tripwireData
		} else {
			result["status"] = "failed"
			result["error"] = prevResult["error"]
		}

	case "suspended":
		result["status"] = "suspended"
		// Build suspended steps list
		suspendedSteps := make([]any, 0)
		if stepResultsRaw != nil {
			for stepID, srRaw := range stepResultsRaw {
				if stepID == "__state" {
					continue
				}
				if srMap, ok := srRaw.(map[string]any); ok {
					if srMap["status"] == "suspended" {
						path := []any{stepID}
						if sp, ok := srMap["suspendPayload"].(map[string]any); ok {
							if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
								if existingPath, ok := meta["path"].([]any); ok {
									path = append(path, existingPath...)
								}
							}
						}
						suspendedSteps = append(suspendedSteps, path)
					}
				}
			}
		}
		result["suspended"] = suspendedSteps

	case "paused":
		result["status"] = "paused"

	default:
		if params.PerStep {
			result["status"] = "paused"
		} else {
			result["status"] = prevStatus
			result["result"] = prevResult["output"]
		}
	}

	// Include state in result only if outputOptions.includeState is true
	if params.OutputOptions != nil && params.OutputOptions.IncludeState {
		result["state"] = finalState
	}

	// Invoke lifecycle callbacks for non-paused results
	if result["status"] != "paused" {
		_ = e.InvokeLifecycleCallbacks(wf.LifecycleCallbackParams{
			Status:         wf.WorkflowRunStatus(result["status"].(string)),
			Result:         result["result"],
			Error:          result["error"],
			Steps:          rawMapToStepResults(cleanStepResults),
			Tripwire:       result["tripwire"],
			RunID:          params.RunID,
			WorkflowID:     params.WorkflowID,
			ResourceID:     params.ResourceID,
			Input:          params.Input,
			RequestContext: params.RequestContext,
			State:          finalState.(map[string]any),
		})
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func serializeRequestContext(rc *requestcontext.RequestContext) map[string]any {
	if rc == nil {
		return map[string]any{}
	}
	return rc.Entries()
}

// rawMapToStepResults converts a map[string]any (from the evented engine's raw
// event data) into map[string]wf.StepResult for the lifecycle callback params.
// Each value is expected to be a map[string]any with "status", "output", etc.
// Values that cannot be converted are given a minimal StepResult with status "completed".
func rawMapToStepResults(raw map[string]any) map[string]wf.StepResult {
	if raw == nil {
		return nil
	}
	result := make(map[string]wf.StepResult, len(raw))
	for k, v := range raw {
		switch typed := v.(type) {
		case wf.StepResult:
			result[k] = typed
		case map[string]any:
			sr := wf.StepResult{}
			if status, ok := typed["status"].(string); ok {
				sr.Status = wf.WorkflowStepStatus(status)
			}
			sr.Output = typed["output"]
			sr.Payload = typed["payload"]
			sr.ResumePayload = typed["resumePayload"]
			sr.SuspendPayload = typed["suspendPayload"]
			sr.SuspendOutput = typed["suspendOutput"]
			if errVal := typed["error"]; errVal != nil {
				if e, ok := errVal.(error); ok {
					sr.Error = e
				} else {
					sr.Error = fmt.Errorf("%v", errVal)
				}
			}
			if tw, ok := typed["tripwire"].(map[string]any); ok {
				reason, _ := tw["reason"].(string)
				sr.Tripwire = &wf.StepTripwireInfo{
					Reason:   reason,
					Metadata: tw,
				}
			}
			result[k] = sr
		default:
			// Fallback: wrap raw value as output
			result[k] = wf.StepResult{
				Status: wf.StepStatusSuccess,
				Output: v,
			}
		}
	}
	return result
}

func serializeOutputOptions(opts *wf.OutputOptions) map[string]any {
	if opts == nil {
		return nil
	}
	return map[string]any{
		"includeState":        opts.IncludeState,
		"includeResumeLabels": opts.IncludeResumeLabels,
	}
}

func getResumePayload(params wf.ExecuteParams) any {
	if params.Resume == nil {
		return nil
	}
	// In TS: const prevStep = getStep(this.mastra!.getWorkflow(params.workflowId), params.resume.resumePath);
	//         const prevResult = params.resume.stepResults[prevStep?.id ?? 'input'];
	//
	// We cannot call eventprocessor.GetStep here due to circular import
	// (eventprocessor already imports evented). The TS resolves the step from
	// the workflow graph to get its ID, then looks up that step's result.
	// Without access to the graph, we fall back to "input" which is the TS
	// default when getStep returns nil.
	if result, ok := params.Resume.StepResults["input"]; ok {
		return result.Payload
	}
	return nil
}

func getTimeTravelPayload(params wf.ExecuteParams) any {
	if params.TimeTravel == nil {
		return nil
	}
	// Same approach as getResumePayload — fall back to "input" key since we
	// cannot resolve the step ID from the workflow graph without importing
	// eventprocessor (which would create a circular import).
	if result, ok := params.TimeTravel.StepResults["input"]; ok {
		return result.Payload
	}
	return nil
}
