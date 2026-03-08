// Ported from: packages/core/src/workflows/evented/workflow-event-processor/index.ts
package eventprocessor

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
	"github.com/brainlet/brainkit/agent-kit/core/workflows/evented"
)

// ---------------------------------------------------------------------------
// Extended Mastra interface for the event processor
// ---------------------------------------------------------------------------

// ProcessorMastra extends the evented.Mastra interface with workflow registry
// methods needed by WorkflowEventProcessor.
// TS equivalent: Mastra (from mastra.ts)
type ProcessorMastra interface {
	evented.Mastra
	GetWorkflow(id string) Workflow
	HasInternalWorkflow(id string) bool
	GetInternalWorkflow(id string) Workflow
}

// ---------------------------------------------------------------------------
// WorkflowEventProcessor
// ---------------------------------------------------------------------------

// WorkflowEventProcessor processes workflow events dispatched through the
// pub/sub system. It is the central coordinator that drives evented workflow
// execution by reacting to workflow.start, workflow.resume, workflow.step.run,
// workflow.step.end, workflow.suspend, workflow.fail, workflow.end, and
// workflow.cancel events.
//
// TS equivalent: export class WorkflowEventProcessor extends EventProcessor
type WorkflowEventProcessor struct {
	mastra       ProcessorMastra
	stepExecutor *evented.StepExecutor

	// abortControllers maps runId -> cancel func for active workflow runs
	abortControllers sync.Map // map[string]context.CancelFunc — not needed directly, using sync.Map for thread safety

	// parentChildRelationships maps child runId -> parent runId for tracking nested workflows
	parentChildRelationships sync.Map // map[string]string
}

// NewWorkflowEventProcessor creates a new WorkflowEventProcessor.
// TS equivalent: constructor({ mastra }: { mastra: Mastra })
func NewWorkflowEventProcessor(mastra ProcessorMastra) *WorkflowEventProcessor {
	return &WorkflowEventProcessor{
		mastra:       mastra,
		stepExecutor: evented.NewStepExecutor(mastra),
	}
}

// RegisterMastra registers the mastra instance with this processor and its step executor.
// TS equivalent: __registerMastra(mastra: Mastra)
func (p *WorkflowEventProcessor) RegisterMastra(mastra ProcessorMastra) {
	p.mastra = mastra
	p.stepExecutor.RegisterMastra(mastra)
}

// ---------------------------------------------------------------------------
// Abort controller management
// ---------------------------------------------------------------------------

// getOrCreateAbortController returns an existing abort marker for the run or
// creates one. In Go we track presence via sync.Map; actual cancellation is
// done through the workflow's context system.
// TS equivalent: private getOrCreateAbortController(runId: string): AbortController
func (p *WorkflowEventProcessor) getOrCreateAbortController(runID string) {
	p.abortControllers.LoadOrStore(runID, true)
}

// isAborted checks if a run has been aborted.
func (p *WorkflowEventProcessor) isAborted(runID string) bool {
	val, ok := p.abortControllers.Load(runID)
	if !ok {
		return false
	}
	if aborted, ok := val.(string); ok && aborted == "aborted" {
		return true
	}
	return false
}

// cancelRunAndChildren cancels a workflow run and all its nested child workflows.
// TS equivalent: private cancelRunAndChildren(runId: string): void
func (p *WorkflowEventProcessor) cancelRunAndChildren(runID string) {
	// Mark this run as aborted
	p.abortControllers.Store(runID, "aborted")

	// Find and cancel all child workflows
	p.parentChildRelationships.Range(func(key, value any) bool {
		childRunID := key.(string)
		parentRunID := value.(string)
		if parentRunID == runID {
			p.cancelRunAndChildren(childRunID)
		}
		return true
	})
}

// cleanupRun cleans up abort controller and relationships when a workflow completes.
// Also cleans up any orphaned child entries that reference this run as parent.
// TS equivalent: private cleanupRun(runId: string): void
func (p *WorkflowEventProcessor) cleanupRun(runID string) {
	p.abortControllers.Delete(runID)
	p.parentChildRelationships.Delete(runID)

	// Clean up any orphaned child entries pointing to this run as their parent
	p.parentChildRelationships.Range(func(key, value any) bool {
		parentRunID := value.(string)
		if parentRunID == runID {
			p.parentChildRelationships.Delete(key)
		}
		return true
	})
}

// ---------------------------------------------------------------------------
// errorWorkflow
// ---------------------------------------------------------------------------

// errorWorkflow publishes a workflow.fail event.
// TS equivalent: private async errorWorkflow(args, e: Error)
func (p *WorkflowEventProcessor) errorWorkflow(args *ProcessorArgs, e error) error {
	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available to publish workflow error: %w", e)
	}

	errJSON := mastraerror.GetErrorFromUnknown(e).ToJSON()

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.fail",
		"runId": args.RunID,
		"data": map[string]any{
			"workflowId":     args.WorkflowID,
			"runId":          args.RunID,
			"executionPath":  []int{},
			"resumeSteps":    args.ResumeSteps,
			"stepResults":    args.StepResults,
			"prevResult":     map[string]any{"status": "failed", "error": errJSON},
			"requestContext": args.RequestContext,
			"resumeData":     args.ResumeData,
			"activeSteps":    map[string]bool{},
			"parentWorkflow": args.ParentWorkflow,
		},
	})
}

// ---------------------------------------------------------------------------
// processWorkflowCancel
// ---------------------------------------------------------------------------

// processWorkflowCancel handles a workflow.cancel event.
// TS equivalent: protected async processWorkflowCancel({ workflowId, runId }: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowCancel(args *ProcessorArgs) error {
	// Cancel this workflow and all nested child workflows
	p.cancelRunAndChildren(args.RunID)

	storage := p.mastra.GetStorage()
	var currentState *evented.WorkflowRunState
	if storage != nil {
		store, err := storage.GetStore("workflows")
		if err == nil && store != nil {
			currentState, _ = store.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
				WorkflowName: args.WorkflowID,
				RunID:        args.RunID,
			})
		}
	}

	if currentState == nil {
		log := p.mastra.GetLogger()
		if log != nil {
			log.Warn(fmt.Sprintf("Canceling workflow without loaded state: workflowId=%s runId=%s", args.WorkflowID, args.RunID))
		}
	}

	ctx := map[string]any{}
	rc := map[string]any{}
	if currentState != nil {
		if currentState.Context != nil {
			ctx = currentState.Context
		}
		if currentState.RequestContext != nil {
			rc = currentState.RequestContext
		}
	}

	return p.endWorkflow(&ProcessorArgs{
		Workflow:       args.Workflow,
		WorkflowID:     args.WorkflowID,
		RunID:          args.RunID,
		StepResults:    ctx,
		PrevResult:     map[string]any{"status": "canceled"},
		RequestContext: rc,
		ExecutionPath:  []int{},
		ActiveSteps:    map[string]bool{},
		ResumeSteps:    []string{},
	}, "canceled")
}

// ---------------------------------------------------------------------------
// processWorkflowStart
// ---------------------------------------------------------------------------

// processWorkflowStart handles workflow.start and workflow.resume events.
// TS equivalent: protected async processWorkflowStart(args: ProcessorArgs & { initialState?: Record<string, any> })
func (p *WorkflowEventProcessor) processWorkflowStart(args *ProcessorArgs) error {
	initialState := args.InitialState
	if initialState == nil {
		initialState = args.State
	}
	if initialState == nil {
		initialState = map[string]any{}
	}

	// Create abort controller for this workflow run
	p.getOrCreateAbortController(args.RunID)

	// Track parent-child relationship if this is a nested workflow
	if args.ParentWorkflow != nil && args.ParentWorkflow.RunID != "" {
		p.parentChildRelationships.Store(args.RunID, args.ParentWorkflow.RunID)
	}

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	// Preserve resourceId from existing snapshot if present
	storage := p.mastra.GetStorage()
	var resourceID string
	if storage != nil {
		store, err := storage.GetStore("workflows")
		if err == nil && store != nil {
			existingRun, _ := store.GetWorkflowRunByID(evented.GetRunParams{
				RunID:        args.RunID,
				WorkflowName: args.Workflow.GetID(),
			})
			if existingRun != nil {
				resourceID = existingRun.ResourceID
			}

			// Check shouldPersistSnapshot option - default to true if not specified
			shouldPersist := true
			opts := args.Workflow.GetOptions()
			if opts.ShouldPersistSnapshot != nil {
				shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
					StepResults:    anyMapToStepResults(args.StepResults),
					WorkflowStatus: "running",
				})
			}

			if shouldPersist {
				context := args.StepResults
				if context == nil {
					context = map[string]any{}
					if args.PrevResult != nil && args.PrevResult["status"] == "success" {
						context["input"] = args.PrevResult["output"]
					}
				}
				context["__state"] = initialState

				_ = store.PersistWorkflowSnapshot(evented.PersistSnapshotParams{
					WorkflowName: args.Workflow.GetID(),
					RunID:        args.RunID,
					ResourceID:   resourceID,
					Snapshot: evented.WorkflowRunState{
						ActivePaths:         []any{},
						SuspendedPaths:      map[string][]int{},
						ResumeLabels:        map[string]any{},
						WaitingPaths:        map[string][]int{},
						ActiveStepsPath:     map[string]any{},
						SerializedStepGraph: args.Workflow.GetSerializedStepGraph(),
						Timestamp:           time.Now().UnixMilli(),
						RunID:               args.RunID,
						Context:             context,
						Status:              "running",
						Value:               initialState,
					},
				})
			}
		}
	}

	executionPath := args.ExecutionPath
	if len(executionPath) == 0 {
		executionPath = []int{0}
	}

	stepResults := args.StepResults
	if stepResults == nil {
		stepResults = map[string]any{}
		if args.PrevResult != nil && args.PrevResult["status"] == "success" {
			stepResults["input"] = args.PrevResult["output"]
		}
	}
	stepResults["__state"] = initialState

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.step.run",
		"runId": args.RunID,
		"data": map[string]any{
			"parentWorkflow": args.ParentWorkflow,
			"workflowId":     args.WorkflowID,
			"runId":          args.RunID,
			"executionPath":  executionPath,
			"resumeSteps":    args.ResumeSteps,
			"stepResults":    stepResults,
			"prevResult":     args.PrevResult,
			"timeTravel":     args.TimeTravel,
			"requestContext": args.RequestContext,
			"resumeData":     args.ResumeData,
			"activeSteps":    map[string]bool{},
			"perStep":        args.PerStep,
			"state":          initialState,
			"outputOptions":  args.OutputOptions,
			"forEachIndex":   args.ForEachIndex,
		},
	})
}

// ---------------------------------------------------------------------------
// endWorkflow
// ---------------------------------------------------------------------------

// endWorkflow ends a workflow run by updating storage and publishing finish events.
// TS equivalent: protected async endWorkflow(args: ProcessorArgs, status)
func (p *WorkflowEventProcessor) endWorkflow(args *ProcessorArgs, status string) error {
	if status == "" {
		status = "success"
	}

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	finalStatus := status
	if args.PerStep && status == "success" {
		finalStatus = "paused"
	}

	// Check shouldPersistSnapshot option
	shouldPersist := true
	if args.Workflow != nil {
		opts := args.Workflow.GetOptions()
		if opts.ShouldPersistSnapshot != nil {
			shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
				StepResults:    anyMapToStepResults(args.StepResults),
				WorkflowStatus: wf.WorkflowRunStatus(finalStatus),
			})
		}
	}

	if shouldPersist {
		storage := p.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				_ = store.UpdateWorkflowState(evented.UpdateStateParams{
					WorkflowName: args.WorkflowID,
					RunID:        args.RunID,
					Opts: map[string]any{
						"status": finalStatus,
						"result": args.PrevResult,
					},
				})
			}
		}
	}

	if args.PerStep {
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type":    "workflow-paused",
				"payload": map[string]any{},
			},
		})
	}

	_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
		"type":  "watch",
		"runId": args.RunID,
		"data": map[string]any{
			"type": "workflow-finish",
			"payload": map[string]any{
				"runId": args.RunID,
			},
		},
	})

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.end",
		"runId": args.RunID,
		"data":  argsToData(args),
	})
}

// ---------------------------------------------------------------------------
// processWorkflowEnd
// ---------------------------------------------------------------------------

// processWorkflowEnd handles a workflow.end event.
// TS equivalent: protected async processWorkflowEnd(args: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowEnd(args *ProcessorArgs) error {
	finalState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Clean up abort controller and parent-child tracking
	p.cleanupRun(args.RunID)

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	// Handle nested workflow
	if args.ParentWorkflow != nil {
		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.end",
			"runId": args.ParentWorkflow.RunID,
			"data": map[string]any{
				"workflowId":     args.ParentWorkflow.WorkflowID,
				"runId":          args.ParentWorkflow.RunID,
				"executionPath":  args.ParentWorkflow.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"stepResults":    args.ParentWorkflow.StepResults,
				"prevResult":     args.PrevResult,
				"resumeData":     args.ResumeData,
				"activeSteps":    args.ActiveSteps,
				"parentWorkflow": args.ParentWorkflow.ParentWorkflow,
				"parentContext":  args.ParentWorkflow,
				"requestContext": args.RequestContext,
				"timeTravel":     args.TimeTravel,
				"perStep":        args.PerStep,
				"state":          finalState,
				"nestedRunId":    args.RunID,
			},
		})
	}

	data := argsToData(args)
	data["state"] = finalState

	return pubsub.Publish("workflows-finish", map[string]any{
		"type":  "workflow.end",
		"runId": args.RunID,
		"data":  data,
	})
}

// ---------------------------------------------------------------------------
// processWorkflowSuspend
// ---------------------------------------------------------------------------

// processWorkflowSuspend handles a workflow.suspend event.
// TS equivalent: protected async processWorkflowSuspend(args: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowSuspend(args *ProcessorArgs) error {
	finalState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	// Handle nested workflow
	if args.ParentWorkflow != nil {
		// Build suspend payload with nested workflow meta
		prevResult := copyMap(args.PrevResult)
		suspendPayload := getMapField(prevResult, "suspendPayload")
		if suspendPayload == nil {
			suspendPayload = map[string]any{}
		}
		workflowMeta := getMapField(suspendPayload, "__workflow_meta")
		if workflowMeta == nil {
			workflowMeta = map[string]any{}
		}

		// Build path
		existingPath, _ := workflowMeta["path"].([]any)
		var path []any
		if args.ParentWorkflow.StepID != "" {
			path = append([]any{args.ParentWorkflow.StepID}, existingPath...)
		} else {
			path = existingPath
		}
		workflowMeta["runId"] = args.RunID
		workflowMeta["path"] = path
		suspendPayload["__workflow_meta"] = workflowMeta
		prevResult["suspendPayload"] = suspendPayload

		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.end",
			"runId": args.ParentWorkflow.RunID,
			"data": map[string]any{
				"workflowId":     args.ParentWorkflow.WorkflowID,
				"runId":          args.ParentWorkflow.RunID,
				"executionPath":  args.ParentWorkflow.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"stepResults":    args.ParentWorkflow.StepResults,
				"prevResult":     prevResult,
				"timeTravel":     args.TimeTravel,
				"resumeData":     args.ResumeData,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"parentWorkflow": args.ParentWorkflow.ParentWorkflow,
				"parentContext":  args.ParentWorkflow,
				"state":          finalState,
				"outputOptions":  args.OutputOptions,
				"nestedRunId":    args.RunID,
			},
		})
	}

	data := argsToData(args)
	data["state"] = finalState

	return pubsub.Publish("workflows-finish", map[string]any{
		"type":  "workflow.suspend",
		"runId": args.RunID,
		"data":  data,
	})
}

// ---------------------------------------------------------------------------
// processWorkflowFail
// ---------------------------------------------------------------------------

// processWorkflowFail handles a workflow.fail event.
// TS equivalent: protected async processWorkflowFail(args: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowFail(args *ProcessorArgs) error {
	finalState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	// Clean up abort controller and parent-child tracking
	p.cleanupRun(args.RunID)

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	// Check shouldPersistSnapshot option
	shouldPersist := true
	if args.Workflow != nil {
		opts := args.Workflow.GetOptions()
		if opts.ShouldPersistSnapshot != nil {
			shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
				StepResults:    anyMapToStepResults(args.StepResults),
				WorkflowStatus: "failed",
			})
		}
	}

	if shouldPersist {
		storage := p.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				var errVal any
				if args.PrevResult != nil {
					errVal = args.PrevResult["error"]
				}
				_ = store.UpdateWorkflowState(evented.UpdateStateParams{
					WorkflowName: args.WorkflowID,
					RunID:        args.RunID,
					Opts: map[string]any{
						"status": "failed",
						"error":  errVal,
					},
				})
			}
		}
	}

	// Handle nested workflow
	if args.ParentWorkflow != nil {
		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.end",
			"runId": args.ParentWorkflow.RunID,
			"data": map[string]any{
				"workflowId":     args.ParentWorkflow.WorkflowID,
				"runId":          args.ParentWorkflow.RunID,
				"executionPath":  args.ParentWorkflow.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"stepResults":    args.ParentWorkflow.StepResults,
				"prevResult":     args.PrevResult,
				"timeTravel":     args.TimeTravel,
				"resumeData":     args.ResumeData,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"parentWorkflow": args.ParentWorkflow.ParentWorkflow,
				"parentContext":  args.ParentWorkflow,
				"state":          finalState,
				"outputOptions":  args.OutputOptions,
				"nestedRunId":    args.RunID,
			},
		})
	}

	data := argsToData(args)
	data["state"] = finalState

	return pubsub.Publish("workflows-finish", map[string]any{
		"type":  "workflow.fail",
		"runId": args.RunID,
		"data":  data,
	})
}

// ---------------------------------------------------------------------------
// processWorkflowStepRun
// ---------------------------------------------------------------------------

// processWorkflowStepRun handles a workflow.step.run event - the core step execution logic.
// TS equivalent: protected async processWorkflowStepRun(args: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowStepRun(args *ProcessorArgs) error {
	currentState := evented.ResolveCurrentState(evented.ResolveStateParams{
		StepResults: args.StepResults,
		State:       args.State,
	})

	if args.Workflow == nil {
		return p.errorWorkflow(args, fmt.Errorf("workflow is nil in processWorkflowStepRun"))
	}

	stepGraph := args.Workflow.GetStepGraph()

	if len(args.ExecutionPath) == 0 {
		return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_WORKFLOW",
			Text:     fmt.Sprintf("Execution path is empty: %v", args.ExecutionPath),
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategorySystem,
		}))
	}

	idx0 := args.ExecutionPath[0]
	if idx0 >= len(stepGraph) {
		// Past the last step, end the workflow successfully
		return p.endWorkflow(&ProcessorArgs{
			Workflow:       args.Workflow,
			ParentWorkflow: args.ParentWorkflow,
			WorkflowID:     args.WorkflowID,
			RunID:          args.RunID,
			ExecutionPath:  args.ExecutionPath,
			ResumeSteps:    args.ResumeSteps,
			StepResults:    args.StepResults,
			PrevResult:     args.PrevResult,
			ActiveSteps:    args.ActiveSteps,
			RequestContext: args.RequestContext,
			State:          currentState,
			OutputOptions:  args.OutputOptions,
		}, "success")
	}

	step := &stepGraph[idx0]

	if step == nil {
		return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_WORKFLOW",
			Text:     fmt.Sprintf("Step not found in step graph: %v", args.ExecutionPath),
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategorySystem,
		}))
	}

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	// Handle parallel/conditional with sub-index
	if (step.Type == wf.StepFlowEntryTypeParallel || step.Type == wf.StepFlowEntryTypeConditional) && len(args.ExecutionPath) > 1 {
		subIdx := args.ExecutionPath[1]
		if subIdx < len(step.Steps) {
			nested := step.Steps[subIdx]
			step = &wf.StepFlowEntry{Type: wf.StepFlowEntryType(nested.Type), Step: nested.Step}
		}
	} else if step.Type == wf.StepFlowEntryTypeParallel {
		return ProcessWorkflowParallel(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				TimeTravel: args.TimeTravel, PrevResult: args.PrevResult,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, PerStep: args.PerStep,
				State: currentState, OutputOptions: args.OutputOptions,
			}, pubsub, step,
		)
	} else if step.Type == wf.StepFlowEntryTypeConditional {
		return ProcessWorkflowConditional(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				TimeTravel: args.TimeTravel, PrevResult: args.PrevResult,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, PerStep: args.PerStep,
				State: currentState, OutputOptions: args.OutputOptions,
			}, pubsub, p.stepExecutor, step,
		)
	} else if step.Type == wf.StepFlowEntryTypeSleep {
		return ProcessWorkflowSleep(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				TimeTravel: args.TimeTravel, PrevResult: args.PrevResult,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, PerStep: args.PerStep,
				State: currentState, OutputOptions: args.OutputOptions,
			}, pubsub, p.stepExecutor, step,
		)
	} else if step.Type == wf.StepFlowEntryTypeSleepUntil {
		return ProcessWorkflowSleepUntil(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				TimeTravel: args.TimeTravel, PrevResult: args.PrevResult,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, PerStep: args.PerStep,
				State: currentState, OutputOptions: args.OutputOptions,
			}, pubsub, p.stepExecutor, step,
		)
	} else if step.Type == wf.StepFlowEntryTypeForeach && len(args.ExecutionPath) == 1 {
		return ProcessWorkflowForEach(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				TimeTravel: args.TimeTravel, PrevResult: args.PrevResult,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, PerStep: args.PerStep,
				State: currentState, OutputOptions: args.OutputOptions,
				ForEachIndex: args.ForEachIndex,
			}, pubsub, p.mastra, step,
		)
	}

	if !IsExecutableStep(step) {
		return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_WORKFLOW",
			Text:     fmt.Sprintf("Step is not executable: %s -- %v", step.Type, args.ExecutionPath),
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategorySystem,
		}))
	}

	if args.ActiveSteps == nil {
		args.ActiveSteps = map[string]bool{}
	}
	args.ActiveSteps[step.Step.ID] = true

	// Run nested workflow
	if step.Step.Component == "WORKFLOW" {
		return p.processNestedWorkflowStepRun(args, step, currentState, pubsub)
	}

	// Emit step start watch event
	if step.Type == wf.StepFlowEntryTypeStep {
		var prevOutput any
		if args.PrevResult != nil && args.PrevResult["status"] == "success" {
			prevOutput = args.PrevResult["output"]
		}
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-start",
				"payload": map[string]any{
					"id":        step.Step.ID,
					"startedAt": time.Now().UnixMilli(),
					"payload":   prevOutput,
					"status":    "running",
				},
			},
		})
	}

	// Build request context
	rc := requestcontext.NewRequestContext()
	if args.RequestContext != nil {
		for key, value := range args.RequestContext {
			rc.Set(key, value)
		}
	}

	// Validate time travel resume data
	var timeTravelResumeData any
	if args.TimeTravel != nil {
		if sr, ok := args.TimeTravel.StepResults[step.Step.ID]; ok && sr.Status == "suspended" {
			timeTravelResumeData = args.TimeTravel.ResumeData
		}
	}
	ttValidation := wf.ValidateStepResumeData(timeTravelResumeData, step.Step)

	var resumeDataToUse any
	if ttValidation.ResumeData != nil && ttValidation.ValidationError == nil {
		resumeDataToUse = ttValidation.ResumeData
	} else if ttValidation.ResumeData != nil && ttValidation.ValidationError != nil {
		log := p.mastra.GetLogger()
		if log != nil {
			log.Warn(fmt.Sprintf("Time travel resume data validation failed: stepId=%s error=%v", step.Step.ID, ttValidation.ValidationError))
		}
	} else if len(args.ResumeSteps) > 0 && args.ResumeSteps[0] == step.Step.ID {
		resumeDataToUse = args.ResumeData
	}

	// Convert step results for executor
	stepResultsTyped := make(map[string]wf.StepResult)
	for k, v := range args.StepResults {
		if k == "__state" {
			continue
		}
		if sr, ok := v.(wf.StepResult); ok {
			stepResultsTyped[k] = sr
		} else if m, ok := v.(map[string]any); ok {
			sr := wf.StepResult{}
			if s, ok := m["status"].(string); ok {
				sr.Status = wf.WorkflowStepStatus(s)
			}
			sr.Output = m["output"]
			sr.Payload = m["payload"]
			sr.ResumePayload = m["resumePayload"]
			sr.SuspendPayload = m["suspendPayload"]
			sr.SuspendOutput = m["suspendOutput"]
			if sa, ok := m["startedAt"].(int64); ok {
				sr.StartedAt = sa
			} else if sa, ok := m["startedAt"].(float64); ok {
				sr.StartedAt = int64(sa)
			}
			if ea, ok := m["endedAt"].(int64); ok {
				sr.EndedAt = ea
			} else if ea, ok := m["endedAt"].(float64); ok {
				sr.EndedAt = int64(ea)
			}
			stepResultsTyped[k] = sr
		}
	}

	// Determine foreachIdx
	var foreachIdx *int
	if step.Type == wf.StepFlowEntryTypeForeach && len(args.ExecutionPath) > 1 {
		idx := args.ExecutionPath[1]
		foreachIdx = &idx
	}

	// Get input
	var input any
	if args.PrevResult != nil {
		input = args.PrevResult["output"]
	}

	// Execute the step
	stepResult := p.stepExecutor.Execute(evented.ExecuteParams{
		WorkflowID:     args.WorkflowID,
		Step:           step.Step,
		RunID:          args.RunID,
		StepResults:    stepResultsTyped,
		State:          currentState,
		RequestContext: rc,
		Input:          input,
		ResumeData:     resumeDataToUse,
		RetryCount:     args.RetryCount,
		ForeachIdx:     foreachIdx,
		ValidateInputs: args.Workflow.GetOptions().ValidateInputs,
		PerStep:        args.PerStep,
	})

	// Update request context from rc
	args.RequestContext = rc.Entries()

	// Handle bailed status
	if stepResult.Status == "bailed" {
		stepResult.Status = "success"
		return p.endWorkflow(&ProcessorArgs{
			Workflow: args.Workflow, ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
			WorkflowID: args.WorkflowID, RunID: args.RunID, ExecutionPath: args.ExecutionPath,
			ResumeSteps: args.ResumeSteps,
			StepResults: mergeStepResultAny(args.StepResults, step.Step.ID, stepResultToMap(stepResult)),
			PrevResult:  stepResultToMap(stepResult),
			ActiveSteps: args.ActiveSteps, RequestContext: args.RequestContext,
			PerStep: args.PerStep, State: currentState, OutputOptions: args.OutputOptions,
		}, "success")
	}

	// Handle failed step
	if stepResult.Status == "failed" {
		retries := 0
		if step.Step.Retries != nil && *step.Step.Retries > 0 {
			retries = *step.Step.Retries
		}
		// TODO: also check workflow.retryConfig.attempts

		if args.RetryCount >= retries {
			// No more retries - pass failure to step.end
			return pubsub.Publish("workflows", map[string]any{
				"type":  "workflow.step.end",
				"runId": args.RunID,
				"data": map[string]any{
					"parentWorkflow": args.ParentWorkflow,
					"workflowId":     args.WorkflowID,
					"runId":          args.RunID,
					"executionPath":  args.ExecutionPath,
					"resumeSteps":    args.ResumeSteps,
					"stepResults":    args.StepResults,
					"prevResult":     stepResultToMap(stepResult),
					"activeSteps":    args.ActiveSteps,
					"requestContext": args.RequestContext,
					"state":          currentState,
					"outputOptions":  args.OutputOptions,
				},
			})
		}
		// Retry the step
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"parentWorkflow": args.ParentWorkflow,
				"workflowId":     args.WorkflowID,
				"runId":          args.RunID,
				"executionPath":  args.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"stepResults":    args.StepResults,
				"timeTravel":     args.TimeTravel,
				"prevResult":     args.PrevResult,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"retryCount":     args.RetryCount + 1,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	// Handle loop step
	if step.Type == wf.StepFlowEntryTypeLoop {
		return ProcessWorkflowLoop(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID,
				PrevResult: stepResultToMap(stepResult), RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, StepResults: args.StepResults,
				ActiveSteps: args.ActiveSteps, ResumeSteps: args.ResumeSteps,
				ResumeData: args.ResumeData, ParentWorkflow: args.ParentWorkflow,
				RequestContext: args.RequestContext, RetryCount: args.RetryCount + 1,
			},
			pubsub, p.stepExecutor, step, stepResultToMap(stepResult),
		)
	}

	// Normal step completion - extract updated state from step result
	updatedState := currentState
	if stepResult.Metadata != nil {
		if st, ok := stepResult.Metadata["__state"]; ok {
			if stMap, ok := st.(map[string]any); ok {
				updatedState = stMap
			}
		}
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
			"timeTravel":     args.TimeTravel,
			"stepResults":    mergeStepResultAny(args.StepResults, step.Step.ID, stepResultToMap(stepResult)),
			"prevResult":     stepResultToMap(stepResult),
			"activeSteps":    args.ActiveSteps,
			"requestContext": args.RequestContext,
			"perStep":        args.PerStep,
			"state":          updatedState,
			"outputOptions":  args.OutputOptions,
			"forEachIndex":   args.ForEachIndex,
		},
	})
}

// ---------------------------------------------------------------------------
// processNestedWorkflowStepRun
// ---------------------------------------------------------------------------

// processNestedWorkflowStepRun handles nested workflow execution within processWorkflowStepRun.
// TS equivalent: the nested workflow handling section within processWorkflowStepRun
func (p *WorkflowEventProcessor) processNestedWorkflowStepRun(
	args *ProcessorArgs,
	step *wf.StepFlowEntry,
	currentState map[string]any,
	pubsub evented.PubSub,
) error {
	storage := p.mastra.GetStorage()
	var workflowsStore evented.WorkflowsStore
	if storage != nil {
		store, err := storage.GetStore("workflows")
		if err == nil {
			workflowsStore = store
		}
	}

	parentWf := &ParentWorkflow{
		StepID:         step.Step.ID,
		WorkflowID:     args.WorkflowID,
		RunID:          args.RunID,
		ExecutionPath:  args.ExecutionPath,
		Resume:         false,
		StepResults:    args.StepResults,
		ParentWorkflow: args.ParentWorkflow,
	}

	// Handle resume with only nested workflow ID specified (auto-detect suspended inner step)
	if len(args.ResumeSteps) == 1 && args.ResumeSteps[0] == step.Step.ID {
		stepData, _ := args.StepResults[step.Step.ID].(map[string]any)
		var nestedRunID string
		if stepData != nil {
			if sp, ok := stepData["suspendPayload"].(map[string]any); ok {
				if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
					nestedRunID, _ = meta["runId"].(string)
				}
			}
		}
		if nestedRunID == "" {
			return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_WORKFLOW",
				Text:     fmt.Sprintf("Nested workflow run id not found for auto-detection: %v", args.StepResults),
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategorySystem,
			}))
		}

		var snapshot *evented.WorkflowRunState
		if workflowsStore != nil {
			snapshot, _ = workflowsStore.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
				WorkflowName: step.Step.ID,
				RunID:        nestedRunID,
			})
		}

		// Auto-detect the suspended step within the nested workflow
		var suspendedStepID string
		if snapshot != nil {
			for k := range snapshot.SuspendedPaths {
				suspendedStepID = k
				break
			}
		}
		if suspendedStepID == "" {
			return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_WORKFLOW",
				Text:     fmt.Sprintf("No suspended step found in nested workflow: %s", step.Step.ID),
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategorySystem,
			}))
		}

		var nestedExecPath any
		var nestedStepResults map[string]any
		if snapshot != nil {
			nestedExecPath = snapshot.SuspendedPaths[suspendedStepID]
			nestedStepResults = snapshot.Context
		}

		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.resume",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     step.Step.ID,
				"parentWorkflow": parentWf,
				"executionPath":  nestedExecPath,
				"runId":          nestedRunID,
				"resumeSteps":    []string{suspendedStepID},
				"stepResults":    nestedStepResults,
				"prevResult":     args.PrevResult,
				"resumeData":     args.ResumeData,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"perStep":        args.PerStep,
				"initialState":   currentState,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	// Handle resume with explicit nested path
	if len(args.ResumeSteps) > 1 {
		stepData, _ := args.StepResults[step.Step.ID].(map[string]any)
		var nestedRunID string
		if stepData != nil {
			if sp, ok := stepData["suspendPayload"].(map[string]any); ok {
				if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
					nestedRunID, _ = meta["runId"].(string)
				}
			}
		}
		if nestedRunID == "" {
			return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
				ID:       "MASTRA_WORKFLOW",
				Text:     fmt.Sprintf("Nested workflow run id not found: %v", args.StepResults),
				Domain:   mastraerror.ErrorDomainMastraWorkflow,
				Category: mastraerror.ErrorCategorySystem,
			}))
		}

		var snapshot *evented.WorkflowRunState
		if workflowsStore != nil {
			snapshot, _ = workflowsStore.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
				WorkflowName: step.Step.ID,
				RunID:        nestedRunID,
			})
		}

		var nestedStepResults map[string]any
		if snapshot != nil {
			nestedStepResults = snapshot.Context
		}
		nestedSteps := args.ResumeSteps[1:]

		var execPath any
		if snapshot != nil && len(nestedSteps) > 0 {
			execPath = snapshot.SuspendedPaths[nestedSteps[0]]
		}

		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.resume",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     step.Step.ID,
				"parentWorkflow": parentWf,
				"executionPath":  execPath,
				"runId":          nestedRunID,
				"resumeSteps":    nestedSteps,
				"stepResults":    nestedStepResults,
				"prevResult":     args.PrevResult,
				"resumeData":     args.ResumeData,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"perStep":        args.PerStep,
				"initialState":   currentState,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	// Handle time travel into nested workflow
	if args.TimeTravel != nil && len(args.TimeTravel.Steps) > 1 && args.TimeTravel.Steps[0] == step.Step.ID {
		var snapshot *evented.WorkflowRunState
		if workflowsStore != nil {
			snapshot, _ = workflowsStore.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
				WorkflowName: step.Step.ID,
				RunID:        args.RunID,
			})
		}
		if snapshot == nil {
			snapshot = &evented.WorkflowRunState{Context: map[string]any{}}
		}

		// TODO: Full time travel params creation for nested workflow
		// This requires casting step.Step to a Workflow and calling buildExecutionGraph()
		// For now, publish a basic start event
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.start",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     step.Step.ID,
				"parentWorkflow": parentWf,
				"executionPath":  []int{0},
				"runId":          uuid.New().String(),
				"stepResults":    snapshot.Context,
				"prevResult":     args.PrevResult,
				"timeTravel":     args.TimeTravel,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"perStep":        args.PerStep,
				"initialState":   currentState,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	// Fresh start of nested workflow
	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.start",
		"runId": args.RunID,
		"data": map[string]any{
			"workflowId":     step.Step.ID,
			"parentWorkflow": parentWf,
			"executionPath":  []int{0},
			"runId":          uuid.New().String(),
			"resumeSteps":    args.ResumeSteps,
			"prevResult":     args.PrevResult,
			"resumeData":     args.ResumeData,
			"activeSteps":    args.ActiveSteps,
			"requestContext": args.RequestContext,
			"perStep":        args.PerStep,
			"initialState":   currentState,
			"state":          currentState,
			"outputOptions":  args.OutputOptions,
		},
	})
}

// ---------------------------------------------------------------------------
// processWorkflowStepEnd
// ---------------------------------------------------------------------------

// processWorkflowStepEnd handles a workflow.step.end event.
// TS equivalent: protected async processWorkflowStepEnd(args: ProcessorArgs)
func (p *WorkflowEventProcessor) processWorkflowStepEnd(args *ProcessorArgs) error {
	// Extract state
	var currentState map[string]any
	if args.ParentContext != nil {
		// For nested workflow completion, prefer the passed state
		currentState = args.State
		if currentState == nil {
			currentState = getMapField(args.PrevResult, "__state")
		}
		if currentState == nil {
			currentState = getMapField(args.StepResults, "__state")
		}
		if currentState == nil {
			currentState = map[string]any{}
		}
	} else {
		currentState = getMapField(args.PrevResult, "__state")
		if currentState == nil {
			currentState = getMapField(args.StepResults, "__state")
		}
		if currentState == nil {
			currentState = args.State
		}
		if currentState == nil {
			currentState = map[string]any{}
		}
	}

	// Clean __state from prevResult
	prevResult := args.PrevResult
	if prevResult != nil {
		cleaned := make(map[string]any)
		for k, v := range prevResult {
			if k != "__state" {
				cleaned[k] = v
			}
		}
		prevResult = cleaned
	}

	if args.Workflow == nil {
		return p.errorWorkflow(args, fmt.Errorf("workflow is nil in processWorkflowStepEnd"))
	}

	stepGraph := args.Workflow.GetStepGraph()
	if len(args.ExecutionPath) == 0 || args.ExecutionPath[0] >= len(stepGraph) {
		return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_WORKFLOW",
			Text:     fmt.Sprintf("Step not found: %v", args.ExecutionPath),
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategorySystem,
		}))
	}

	step := &stepGraph[args.ExecutionPath[0]]

	if (step.Type == wf.StepFlowEntryTypeParallel || step.Type == wf.StepFlowEntryTypeConditional) && len(args.ExecutionPath) > 1 {
		subIdx := args.ExecutionPath[1]
		if subIdx < len(step.Steps) {
			nested := step.Steps[subIdx]
			step = &wf.StepFlowEntry{Type: wf.StepFlowEntryType(nested.Type), Step: nested.Step}
		}
	}

	pubsub := p.mastra.PubSub()
	if pubsub == nil {
		return fmt.Errorf("no pubsub available")
	}

	storage := p.mastra.GetStorage()
	var workflowsStore evented.WorkflowsStore
	if storage != nil {
		store, err := storage.GetStore("workflows")
		if err == nil {
			workflowsStore = store
		}
	}

	// Handle foreach step end
	if step.Type == wf.StepFlowEntryTypeForeach {
		return p.processWorkflowStepEndForeach(args, step, currentState, prevResult, pubsub, workflowsStore)
	}

	// Handle executable step end
	if IsExecutableStep(step) {
		// Clear from activeSteps
		if args.ActiveSteps != nil {
			delete(args.ActiveSteps, step.Step.ID)
		}

		// Handle nested workflow completion
		if args.ParentContext != nil {
			input, _ := args.ParentContext.Input.(map[string]any)
			var inputOutput any
			if input != nil {
				inputOutput = input["output"]
			}
			if inputOutput == nil {
				inputOutput = map[string]any{}
			}

			updatedResult := copyMap(prevResult)
			updatedResult["payload"] = inputOutput
			if args.NestedRunID != "" {
				metadata := getMapField(updatedResult, "metadata")
				if metadata == nil {
					metadata = map[string]any{}
				}
				metadata["nestedRunId"] = args.NestedRunID
				updatedResult["metadata"] = metadata
			}
			prevResult = updatedResult
			args.StepResults[step.Step.ID] = prevResult
		}

		if workflowsStore != nil {
			newStepResults, err := workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
				WorkflowName:   args.Workflow.GetID(),
				RunID:          args.RunID,
				StepID:         step.Step.ID,
				Result:         prevResult,
				RequestContext: args.RequestContext,
			})
			if err == nil && newStepResults != nil {
				args.StepResults = newStepResults
			} else if newStepResults == nil {
				return nil
			}
		}
	}

	// Update stepResults with current state
	args.StepResults["__state"] = currentState

	// Handle failed
	prevStatus, _ := prevResult["status"].(string)
	if prevStatus == "" || prevStatus == "failed" {
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.fail",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     args.WorkflowID,
				"runId":          args.RunID,
				"executionPath":  args.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"parentWorkflow": args.ParentWorkflow,
				"stepResults":    args.StepResults,
				"timeTravel":     args.TimeTravel,
				"prevResult":     prevResult,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	// Handle suspended
	if prevStatus == "suspended" {
		suspendedPaths := map[string][]int{}
		suspendedStep := GetStep(args.Workflow, args.ExecutionPath)
		if suspendedStep != nil {
			suspendedPaths[suspendedStep.ID] = args.ExecutionPath
		}

		// Extract resume labels from suspend payload metadata
		resumeLabels := map[string]any{}
		if sp, ok := prevResult["suspendPayload"].(map[string]any); ok {
			if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
				if rl, ok := meta["resumeLabels"].(map[string]any); ok {
					resumeLabels = rl
				}
			}
		}

		// Check shouldPersistSnapshot option
		shouldPersist := true
		opts := args.Workflow.GetOptions()
		if opts.ShouldPersistSnapshot != nil {
			shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
				StepResults:    anyMapToStepResults(args.StepResults),
				WorkflowStatus: "suspended",
			})
		}

		if shouldPersist && workflowsStore != nil {
			// Persist state to snapshot context before suspending
			_, _ = workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
				WorkflowName:   args.Workflow.GetID(),
				RunID:          args.RunID,
				StepID:         "__state",
				Result:         currentState,
				RequestContext: args.RequestContext,
			})

			_ = workflowsStore.UpdateWorkflowState(evented.UpdateStateParams{
				WorkflowName: args.WorkflowID,
				RunID:        args.RunID,
				Opts: map[string]any{
					"status":         "suspended",
					"result":         prevResult,
					"suspendedPaths": suspendedPaths,
					"resumeLabels":   resumeLabels,
				},
			})
		}

		_ = pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.suspend",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     args.WorkflowID,
				"runId":          args.RunID,
				"executionPath":  args.ExecutionPath,
				"resumeSteps":    args.ResumeSteps,
				"parentWorkflow": args.ParentWorkflow,
				"stepResults":    args.StepResults,
				"prevResult":     prevResult,
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"timeTravel":     args.TimeTravel,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})

		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-suspended",
				"payload": mergeMap(map[string]any{
					"id":             step.Step.ID,
					"suspendedAt":    time.Now().UnixMilli(),
					"suspendPayload": prevResult["suspendPayload"],
				}, prevResult),
			},
		})

		return nil
	}

	// Emit step result and finish watch events
	if step.Type == wf.StepFlowEntryTypeStep {
		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type": "workflow-step-result",
				"payload": mergeMap(map[string]any{
					"id": step.Step.ID,
				}, prevResult),
			},
		})

		if prevStatus == "success" {
			_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
				"type":  "watch",
				"runId": args.RunID,
				"data": map[string]any{
					"type": "workflow-step-finish",
					"payload": map[string]any{
						"id":       step.Step.ID,
						"metadata": map[string]any{},
					},
				},
			})
		}
	}

	// Determine next action
	topStep := &stepGraph[args.ExecutionPath[0]]

	if args.PerStep {
		if args.ParentWorkflow != nil && args.ExecutionPath[0] < len(stepGraph)-1 {
			// Nested workflow, not at last step - pause
			pausedResult := copyMap(prevResult)
			pausedResult["status"] = "paused"
			delete(pausedResult, "endedAt")
			delete(pausedResult, "output")
			return p.endWorkflow(&ProcessorArgs{
				Workflow: args.Workflow, ParentWorkflow: args.ParentWorkflow,
				WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: args.ExecutionPath, ResumeSteps: args.ResumeSteps,
				StepResults: args.StepResults, PrevResult: pausedResult,
				ActiveSteps: args.ActiveSteps, RequestContext: args.RequestContext,
				PerStep: args.PerStep,
			}, "success")
		}
		return p.endWorkflow(&ProcessorArgs{
			Workflow: args.Workflow, ParentWorkflow: args.ParentWorkflow,
			WorkflowID: args.WorkflowID, RunID: args.RunID,
			ExecutionPath: args.ExecutionPath, ResumeSteps: args.ResumeSteps,
			StepResults: args.StepResults, PrevResult: prevResult,
			ActiveSteps: args.ActiveSteps, RequestContext: args.RequestContext,
			PerStep: args.PerStep,
		}, "success")
	}

	if (topStep.Type == wf.StepFlowEntryTypeParallel || topStep.Type == wf.StepFlowEntryTypeConditional) && len(args.ExecutionPath) > 1 {
		// Check if all parallel/conditional branches are complete
		skippedCount := 0
		allResults := map[string]any{}
		for _, ns := range topStep.Steps {
			fe := &wf.StepFlowEntry{Type: wf.StepFlowEntryType(ns.Type), Step: ns.Step}
			if IsExecutableStep(fe) && ns.Step != nil {
				res := args.StepResults[ns.Step.ID]
				if resMap, ok := res.(map[string]any); ok {
					if resMap["status"] == "success" {
						allResults[ns.Step.ID] = resMap["output"]
					} else if resMap["status"] == "skipped" {
						skippedCount++
					}
				}
			}
		}

		if len(allResults)+skippedCount < len(topStep.Steps) {
			// Not all branches complete yet
			return nil
		}

		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.end",
			"runId": args.RunID,
			"data": map[string]any{
				"parentWorkflow": args.ParentWorkflow,
				"workflowId":     args.WorkflowID,
				"runId":          args.RunID,
				"executionPath":  args.ExecutionPath[:len(args.ExecutionPath)-1],
				"resumeSteps":    args.ResumeSteps,
				"stepResults":    args.StepResults,
				"prevResult":     map[string]any{"status": "success", "output": allResults},
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"timeTravel":     args.TimeTravel,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
			},
		})
	}

	if topStep.Type == wf.StepFlowEntryTypeForeach {
		// Get the original array from the foreach step's stored payload
		var originalArray any
		if topStep.Step != nil {
			if foreachResult, ok := args.StepResults[topStep.Step.ID].(map[string]any); ok {
				originalArray = foreachResult["payload"]
			}
		}
		return pubsub.Publish("workflows", map[string]any{
			"type":  "workflow.step.run",
			"runId": args.RunID,
			"data": map[string]any{
				"workflowId":     args.WorkflowID,
				"runId":          args.RunID,
				"executionPath":  args.ExecutionPath[:len(args.ExecutionPath)-1],
				"resumeSteps":    args.ResumeSteps,
				"parentWorkflow": args.ParentWorkflow,
				"stepResults":    args.StepResults,
				"prevResult":     mergeMap(copyMap(prevResult), map[string]any{"output": originalArray}),
				"activeSteps":    args.ActiveSteps,
				"requestContext": args.RequestContext,
				"timeTravel":     args.TimeTravel,
				"state":          currentState,
				"outputOptions":  args.OutputOptions,
				"forEachIndex":   args.ForEachIndex,
			},
		})
	}

	if args.ExecutionPath[0] >= len(stepGraph)-1 {
		// Last step - end workflow
		return p.endWorkflow(&ProcessorArgs{
			Workflow: args.Workflow, ParentWorkflow: args.ParentWorkflow,
			WorkflowID: args.WorkflowID, RunID: args.RunID,
			ExecutionPath: args.ExecutionPath, ResumeSteps: args.ResumeSteps,
			StepResults: args.StepResults, PrevResult: prevResult,
			ActiveSteps: args.ActiveSteps, RequestContext: args.RequestContext,
			State: currentState, OutputOptions: args.OutputOptions,
		}, "success")
	}

	// Advance to next step
	nextPath := advanceExecutionPath(args.ExecutionPath)
	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.step.run",
		"runId": args.RunID,
		"data": map[string]any{
			"workflowId":     args.WorkflowID,
			"runId":          args.RunID,
			"executionPath":  nextPath,
			"resumeSteps":    args.ResumeSteps,
			"parentWorkflow": args.ParentWorkflow,
			"stepResults":    args.StepResults,
			"prevResult":     prevResult,
			"activeSteps":    args.ActiveSteps,
			"requestContext": args.RequestContext,
			"timeTravel":     args.TimeTravel,
			"state":          currentState,
			"outputOptions":  args.OutputOptions,
		},
	})
}

// ---------------------------------------------------------------------------
// processWorkflowStepEndForeach
// ---------------------------------------------------------------------------

// processWorkflowStepEndForeach handles foreach-specific logic in processWorkflowStepEnd.
func (p *WorkflowEventProcessor) processWorkflowStepEndForeach(
	args *ProcessorArgs,
	step *wf.StepFlowEntry,
	currentState map[string]any,
	prevResult map[string]any,
	pubsub evented.PubSub,
	workflowsStore evented.WorkflowsStore,
) error {
	var snapshot *evented.WorkflowRunState
	if workflowsStore != nil {
		snapshot, _ = workflowsStore.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
			WorkflowName: args.WorkflowID,
			RunID:        args.RunID,
		})
	}

	currentIdx := -1
	if len(args.ExecutionPath) > 1 {
		currentIdx = args.ExecutionPath[1]
	}

	var existingStepResult map[string]any
	if snapshot != nil && step.Step != nil {
		if sr, ok := snapshot.Context[step.Step.ID].(map[string]any); ok {
			existingStepResult = sr
		}
	}

	var currentOutput []any
	if existingStepResult != nil {
		if out, ok := existingStepResult["output"].([]any); ok {
			currentOutput = out
		}
	}
	var originalPayload any
	if existingStepResult != nil {
		originalPayload = existingStepResult["payload"]
	}

	newResult := prevResult
	if currentIdx >= 0 {
		prevStatus, _ := prevResult["status"].(string)

		// Check for bail
		if prevStatus == "bailed" {
			bailedResult := map[string]any{
				"status":    "success",
				"output":    prevResult["output"],
				"startedAt": existingStepResult["startedAt"],
				"endedAt":   time.Now().UnixMilli(),
				"payload":   originalPayload,
			}
			if bailedResult["startedAt"] == nil {
				bailedResult["startedAt"] = time.Now().UnixMilli()
			}

			if workflowsStore != nil {
				_, _ = workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
					WorkflowName:   args.Workflow.GetID(),
					RunID:          args.RunID,
					StepID:         step.Step.ID,
					Result:         bailedResult,
					RequestContext: args.RequestContext,
				})
			}

			return p.endWorkflow(&ProcessorArgs{
				Workflow: args.Workflow, ParentWorkflow: args.ParentWorkflow,
				WorkflowID: args.WorkflowID, RunID: args.RunID,
				ExecutionPath: []int{args.ExecutionPath[0]},
				ResumeSteps: args.ResumeSteps,
				StepResults: mergeStepResultAny(args.StepResults, step.Step.ID, bailedResult),
				PrevResult:  bailedResult,
				ActiveSteps: args.ActiveSteps, RequestContext: args.RequestContext,
				PerStep: args.PerStep, State: currentState, OutputOptions: args.OutputOptions,
			}, "success")
		}

		// Store iteration result
		var iterationResult any
		if prevStatus == "suspended" {
			iterationResult = prevResult
		} else {
			iterationResult = prevResult["output"]
		}

		if currentOutput != nil && currentIdx < len(currentOutput) {
			currentOutput[currentIdx] = iterationResult
			newResult = copyMap(existingStepResult)
			for k, v := range prevResult {
				newResult[k] = v
			}
			newResult["output"] = currentOutput
			newResult["payload"] = originalPayload
			// Preserve suspend metadata from first suspension
			if existingStepResult["suspendPayload"] != nil {
				newResult["suspendPayload"] = existingStepResult["suspendPayload"]
			}
			if existingStepResult["suspendedAt"] != nil {
				newResult["suspendedAt"] = existingStepResult["suspendedAt"]
			}
			// Update resume metadata to most recent resume
			if prevResult["resumePayload"] != nil {
				newResult["resumePayload"] = prevResult["resumePayload"]
			}
			if prevResult["resumedAt"] != nil {
				newResult["resumedAt"] = prevResult["resumedAt"]
			}
		} else {
			newResult = copyMap(prevResult)
			newResult["output"] = []any{iterationResult}
			newResult["payload"] = originalPayload
		}
	}

	var newStepResults map[string]any
	if workflowsStore != nil {
		var err error
		newStepResults, err = workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
			WorkflowName:   args.Workflow.GetID(),
			RunID:          args.RunID,
			StepID:         step.Step.ID,
			Result:         newResult,
			RequestContext: args.RequestContext,
		})
		if err != nil || newStepResults == nil {
			return nil
		}
		args.StepResults = newStepResults
	}

	// For foreach iterations, check if all iterations are complete
	if currentIdx >= 0 {
		foreachResult, _ := args.StepResults[step.Step.ID].(map[string]any)
		iterationResults, _ := foreachResult["output"].([]any)
		var targetPayload []any
		if foreachResult != nil {
			targetPayload, _ = foreachResult["payload"].([]any)
		}
		targetLen := len(targetPayload)

		// Count by status
		pendingCount := 0
		suspendedCount := 0
		completedCount := 0
		for _, r := range iterationResults {
			if r == nil {
				pendingCount++
			} else if rMap, ok := r.(map[string]any); ok && rMap["status"] == "suspended" {
				suspendedCount++
			} else {
				completedCount++
			}
		}

		// Emit per-iteration progress event
		prevStatus, _ := prevResult["status"].(string)
		iterationStatus := "failed"
		if prevStatus == "suspended" {
			iterationStatus = "suspended"
		} else if prevStatus == "success" {
			iterationStatus = "success"
		}

		progressPayload := map[string]any{
			"id":              step.Step.ID,
			"completedCount":  completedCount,
			"totalCount":      targetLen,
			"currentIndex":    currentIdx,
			"iterationStatus": iterationStatus,
		}
		if prevStatus == "success" {
			progressPayload["iterationOutput"] = prevResult["output"]
		}

		_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", args.RunID), map[string]any{
			"type":  "watch",
			"runId": args.RunID,
			"data": map[string]any{
				"type":    "workflow-step-progress",
				"payload": progressPayload,
			},
		})

		if pendingCount > 0 {
			// Still pending iterations
			return nil
		}

		iterationsStarted := len(iterationResults)
		if iterationsStarted < targetLen {
			// More iterations to start
			return ProcessWorkflowForEach(
				&ProcessorArgs{
					Workflow: args.Workflow, WorkflowID: args.WorkflowID,
					PrevResult: map[string]any{"status": "success", "output": targetPayload},
					RunID: args.RunID, ExecutionPath: []int{args.ExecutionPath[0]},
					StepResults: args.StepResults, ActiveSteps: args.ActiveSteps,
					ResumeSteps: args.ResumeSteps, TimeTravel: args.TimeTravel,
					ParentWorkflow: args.ParentWorkflow, RequestContext: args.RequestContext,
					PerStep: args.PerStep, State: currentState, OutputOptions: args.OutputOptions,
				}, pubsub, p.mastra, step,
			)
		}

		if suspendedCount > 0 {
			// Some iterations suspended - suspend the whole foreach
			return p.handleForeachSuspend(args, step, currentState, iterationResults, targetPayload, pubsub, workflowsStore)
		}

		// All iterations succeeded - advance to next step
		return ProcessWorkflowForEach(
			&ProcessorArgs{
				Workflow: args.Workflow, WorkflowID: args.WorkflowID,
				PrevResult: map[string]any{"status": "success", "output": targetPayload},
				RunID: args.RunID, ExecutionPath: []int{args.ExecutionPath[0]},
				StepResults: args.StepResults, ActiveSteps: args.ActiveSteps,
				ResumeSteps: args.ResumeSteps, TimeTravel: args.TimeTravel,
				ParentWorkflow: args.ParentWorkflow, RequestContext: args.RequestContext,
				PerStep: args.PerStep, State: currentState, OutputOptions: args.OutputOptions,
			}, pubsub, p.mastra, step,
		)
	}

	return nil
}

// handleForeachSuspend handles the case where some foreach iterations are suspended.
func (p *WorkflowEventProcessor) handleForeachSuspend(
	args *ProcessorArgs,
	step *wf.StepFlowEntry,
	currentState map[string]any,
	iterationResults []any,
	targetPayload []any,
	pubsub evented.PubSub,
	workflowsStore evented.WorkflowsStore,
) error {
	collectedResumeLabels := map[string]any{}
	suspendedPaths := map[string][]int{
		step.Step.ID: {args.ExecutionPath[0]},
	}

	for _, iterResult := range iterationResults {
		if rMap, ok := iterResult.(map[string]any); ok && rMap["status"] == "suspended" {
			if sp, ok := rMap["suspendPayload"].(map[string]any); ok {
				if meta, ok := sp["__workflow_meta"].(map[string]any); ok {
					if rl, ok := meta["resumeLabels"].(map[string]any); ok {
						for k, v := range rl {
							collectedResumeLabels[k] = v
						}
					}
				}
			}
		}
	}

	foreachSuspendResult := map[string]any{
		"status":      "suspended",
		"output":      iterationResults,
		"payload":     targetPayload,
		"suspendedAt": time.Now().UnixMilli(),
		"suspendPayload": map[string]any{
			"__workflow_meta": map[string]any{
				"path":         args.ExecutionPath,
				"resumeLabels": collectedResumeLabels,
			},
		},
	}
	if existingResult, ok := args.StepResults[step.Step.ID].(map[string]any); ok {
		foreachSuspendResult["startedAt"] = existingResult["startedAt"]
	}

	if workflowsStore != nil {
		_, _ = workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
			WorkflowName:   args.Workflow.GetID(),
			RunID:          args.RunID,
			StepID:         step.Step.ID,
			Result:         foreachSuspendResult,
			RequestContext: args.RequestContext,
		})

		// Check shouldPersistSnapshot option
		shouldPersist := true
		opts := args.Workflow.GetOptions()
		if opts.ShouldPersistSnapshot != nil {
			shouldPersist = opts.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
				StepResults:    anyMapToStepResults(args.StepResults),
				WorkflowStatus: "suspended",
			})
		}

		if shouldPersist {
			_, _ = workflowsStore.UpdateWorkflowResults(evented.UpdateResultsParams{
				WorkflowName:   args.Workflow.GetID(),
				RunID:          args.RunID,
				StepID:         "__state",
				Result:         currentState,
				RequestContext: args.RequestContext,
			})

			_ = workflowsStore.UpdateWorkflowState(evented.UpdateStateParams{
				WorkflowName: args.WorkflowID,
				RunID:        args.RunID,
				Opts: map[string]any{
					"status":         "suspended",
					"result":         foreachSuspendResult,
					"suspendedPaths": suspendedPaths,
					"resumeLabels":   collectedResumeLabels,
				},
			})
		}
	}

	return pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.suspend",
		"runId": args.RunID,
		"data": map[string]any{
			"workflowId":     args.WorkflowID,
			"runId":          args.RunID,
			"executionPath":  []int{args.ExecutionPath[0]},
			"resumeSteps":    args.ResumeSteps,
			"parentWorkflow": args.ParentWorkflow,
			"stepResults":    mergeStepResultAny(args.StepResults, step.Step.ID, foreachSuspendResult),
			"prevResult":     foreachSuspendResult,
			"activeSteps":    args.ActiveSteps,
			"requestContext": args.RequestContext,
			"timeTravel":     args.TimeTravel,
			"state":          currentState,
			"outputOptions":  args.OutputOptions,
		},
	})
}

// ---------------------------------------------------------------------------
// loadData
// ---------------------------------------------------------------------------

// LoadData loads a workflow's snapshot from storage.
// TS equivalent: async loadData({ workflowId, runId }): Promise<WorkflowRunState | null | undefined>
func (p *WorkflowEventProcessor) LoadData(workflowID, runID string) *evented.WorkflowRunState {
	storage := p.mastra.GetStorage()
	if storage == nil {
		return nil
	}
	store, err := storage.GetStore("workflows")
	if err != nil || store == nil {
		return nil
	}
	snapshot, _ := store.LoadWorkflowSnapshot(evented.LoadSnapshotParams{
		WorkflowName: workflowID,
		RunID:        runID,
	})
	return snapshot
}

// ---------------------------------------------------------------------------
// Process - main event dispatcher
// ---------------------------------------------------------------------------

// Process handles a single event from the pub/sub system.
// TS equivalent: async process(event: Event, ack?: () => Promise<void>)
func (p *WorkflowEventProcessor) Process(eventType string, eventData map[string]any, ack func() error) error {
	workflowID, _ := eventData["workflowId"].(string)
	runID, _ := eventData["runId"].(string)

	currentState := p.LoadData(workflowID, runID)

	if currentState != nil && currentState.Status == "canceled" && eventType != "workflow.end" && eventType != "workflow.cancel" {
		return nil
	}

	pubsub := p.mastra.PubSub()

	// Handle user events (workflow.user-event.*)
	if len(eventType) > 20 && eventType[:20] == "workflow.user-event." {
		eventName := eventType[20:]
		workflow := p.mastra.GetWorkflow(workflowID)
		args := dataToProcessorArgs(eventData)
		args.Workflow = workflow
		return ProcessWorkflowWaitForEvent(args, pubsub, eventName, currentState)
	}

	// Resolve workflow
	var workflow Workflow
	if p.mastra.HasInternalWorkflow(workflowID) {
		workflow = p.mastra.GetInternalWorkflow(workflowID)
	} else if pw := getParentWorkflow(eventData); pw != nil {
		workflow = GetNestedWorkflow(p.mastra, pw)
	} else {
		workflow = p.mastra.GetWorkflow(workflowID)
	}

	if workflow == nil {
		args := dataToProcessorArgs(eventData)
		return p.errorWorkflow(args, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "MASTRA_WORKFLOW",
			Text:     fmt.Sprintf("Workflow not found: %s", workflowID),
			Domain:   mastraerror.ErrorDomainMastraWorkflow,
			Category: mastraerror.ErrorCategorySystem,
		}))
	}

	// Emit workflow-start watch event for start/resume
	if eventType == "workflow.start" || eventType == "workflow.resume" {
		if pubsub != nil {
			_ = pubsub.Publish(fmt.Sprintf("workflow.events.v2.%s", runID), map[string]any{
				"type":  "watch",
				"runId": runID,
				"data": map[string]any{
					"type": "workflow-start",
					"payload": map[string]any{
						"runId": runID,
					},
				},
			})
		}
	}

	args := dataToProcessorArgs(eventData)
	args.Workflow = workflow

	var err error
	switch eventType {
	case "workflow.cancel":
		err = p.processWorkflowCancel(args)
	case "workflow.start", "workflow.resume":
		err = p.processWorkflowStart(args)
	case "workflow.end":
		err = p.processWorkflowEnd(args)
	case "workflow.step.end":
		err = p.processWorkflowStepEnd(args)
	case "workflow.step.run":
		err = p.processWorkflowStepRun(args)
	case "workflow.suspend":
		err = p.processWorkflowSuspend(args)
	case "workflow.fail":
		err = p.processWorkflowFail(args)
	}

	if ack != nil {
		if ackErr := ack(); ackErr != nil {
			log := p.mastra.GetLogger()
			if log != nil {
				log.Error(fmt.Sprintf("Error acking event: %v", ackErr))
			}
		}
	}

	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// argsToData converts ProcessorArgs to a map for event data, excluding the workflow field.
func argsToData(args *ProcessorArgs) map[string]any {
	data := map[string]any{
		"workflowId":     args.WorkflowID,
		"runId":          args.RunID,
		"executionPath":  args.ExecutionPath,
		"stepResults":    args.StepResults,
		"prevResult":     args.PrevResult,
		"activeSteps":    args.ActiveSteps,
		"resumeSteps":    args.ResumeSteps,
		"requestContext": args.RequestContext,
		"state":          args.State,
		"outputOptions":  args.OutputOptions,
	}
	if args.ParentWorkflow != nil {
		data["parentWorkflow"] = args.ParentWorkflow
	}
	if args.TimeTravel != nil {
		data["timeTravel"] = args.TimeTravel
	}
	if args.ResumeData != nil {
		data["resumeData"] = args.ResumeData
	}
	if args.PerStep {
		data["perStep"] = args.PerStep
	}
	if args.ForEachIndex != nil {
		data["forEachIndex"] = args.ForEachIndex
	}
	if args.NestedRunID != "" {
		data["nestedRunId"] = args.NestedRunID
	}
	return data
}

// dataToProcessorArgs converts event data map to ProcessorArgs.
func dataToProcessorArgs(data map[string]any) *ProcessorArgs {
	args := &ProcessorArgs{}
	args.WorkflowID, _ = data["workflowId"].(string)
	args.RunID, _ = data["runId"].(string)
	args.NestedRunID, _ = data["nestedRunId"].(string)

	if ep, ok := data["executionPath"].([]int); ok {
		args.ExecutionPath = ep
	} else if ep, ok := data["executionPath"].([]any); ok {
		args.ExecutionPath = anySliceToIntSlice(ep)
	}

	if sr, ok := data["stepResults"].(map[string]any); ok {
		args.StepResults = sr
	}
	if pr, ok := data["prevResult"].(map[string]any); ok {
		args.PrevResult = pr
	}
	if as, ok := data["activeSteps"].(map[string]bool); ok {
		args.ActiveSteps = as
	} else {
		args.ActiveSteps = map[string]bool{}
	}
	if rs, ok := data["resumeSteps"].([]string); ok {
		args.ResumeSteps = rs
	} else if rs, ok := data["resumeSteps"].([]any); ok {
		args.ResumeSteps = anySliceToStringSlice(rs)
	}
	if rc, ok := data["requestContext"].(map[string]any); ok {
		args.RequestContext = rc
	}
	args.ResumeData = data["resumeData"]
	if tt, ok := data["timeTravel"].(*wf.TimeTravelExecutionParams); ok {
		args.TimeTravel = tt
	}
	if pw, ok := data["parentWorkflow"].(*ParentWorkflow); ok {
		args.ParentWorkflow = pw
	}
	if pc, ok := data["parentContext"].(*ParentContext); ok {
		args.ParentContext = pc
	} else if pc, ok := data["parentContext"].(*ParentWorkflow); ok {
		// parentContext is often a ParentWorkflow in the TS code
		args.ParentContext = &ParentContext{
			WorkflowID: pc.WorkflowID,
			Input:      map[string]any{"output": map[string]any{}},
		}
	}
	if ps, ok := data["perStep"].(bool); ok {
		args.PerStep = ps
	}
	if st, ok := data["state"].(map[string]any); ok {
		args.State = st
	}
	if is, ok := data["initialState"].(map[string]any); ok {
		args.InitialState = is
	}
	if oo, ok := data["outputOptions"].(*OutputOptions); ok {
		args.OutputOptions = oo
	} else if oo, ok := data["outputOptions"].(map[string]any); ok {
		args.OutputOptions = &OutputOptions{
			IncludeState:        toBool(oo["includeState"]),
			IncludeResumeLabels: toBool(oo["includeResumeLabels"]),
		}
	}
	if fi, ok := data["forEachIndex"].(int); ok {
		args.ForEachIndex = &fi
	} else if fi, ok := data["forEachIndex"].(*int); ok {
		args.ForEachIndex = fi
	}
	if rc, ok := data["retryCount"].(int); ok {
		args.RetryCount = rc
	} else if rc, ok := data["retryCount"].(float64); ok {
		args.RetryCount = int(rc)
	}

	return args
}

// getParentWorkflow extracts the parent workflow from event data.
func getParentWorkflow(data map[string]any) *ParentWorkflow {
	pw, ok := data["parentWorkflow"].(*ParentWorkflow)
	if ok {
		return pw
	}
	return nil
}

// copyMap creates a shallow copy of a map.
func copyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// mergeMap merges src into dst (dst fields take precedence if already set by mergeMap logic;
// here we follow the TS {...prevResult} spread pattern where later keys win).
func mergeMap(dst, src map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
	return dst
}

// getMapField extracts a map[string]any field from a map.
func getMapField(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key]; ok {
		if vm, ok := v.(map[string]any); ok {
			return vm
		}
	}
	return nil
}

// stepResultToMap converts a wf.StepResult to map[string]any for event data.
func stepResultToMap(sr wf.StepResult) map[string]any {
	m := map[string]any{
		"status":    string(sr.Status),
		"startedAt": sr.StartedAt,
	}
	if sr.EndedAt != 0 {
		m["endedAt"] = sr.EndedAt
	}
	if sr.Output != nil {
		m["output"] = sr.Output
	}
	if sr.Error != nil {
		m["error"] = sr.Error
	}
	if sr.Payload != nil {
		m["payload"] = sr.Payload
	}
	if sr.ResumePayload != nil {
		m["resumePayload"] = sr.ResumePayload
	}
	if sr.SuspendPayload != nil {
		m["suspendPayload"] = sr.SuspendPayload
	}
	if sr.SuspendOutput != nil {
		m["suspendOutput"] = sr.SuspendOutput
	}
	if sr.SuspendedAt != nil {
		m["suspendedAt"] = *sr.SuspendedAt
	}
	if sr.ResumedAt != nil {
		m["resumedAt"] = *sr.ResumedAt
	}
	if sr.Metadata != nil {
		m["metadata"] = sr.Metadata
		// Extract __state for top-level use
		if st, ok := sr.Metadata["__state"]; ok {
			m["__state"] = st
		}
	}
	if sr.Tripwire != nil {
		m["tripwire"] = sr.Tripwire
	}
	return m
}

// mergeStepResultAny merges a step result into a step results map.
func mergeStepResultAny(stepResults map[string]any, stepID string, result map[string]any) map[string]any {
	merged := copyMap(stepResults)
	merged[stepID] = result
	return merged
}

// anySliceToIntSlice converts []any to []int.
func anySliceToIntSlice(s []any) []int {
	result := make([]int, 0, len(s))
	for _, v := range s {
		switch n := v.(type) {
		case int:
			result = append(result, n)
		case float64:
			result = append(result, int(n))
		case int64:
			result = append(result, int(n))
		}
	}
	return result
}

// anySliceToStringSlice converts []any to []string.
func anySliceToStringSlice(s []any) []string {
	result := make([]string, 0, len(s))
	for _, v := range s {
		if str, ok := v.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

// toBool converts an any value to bool.
func toBool(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// advanceExecutionPath increments the last element of the execution path.
// This is duplicated from sleep.go for use within the processor.
func advanceExecPath(path []int) []int {
	if len(path) == 0 {
		return []int{1}
	}
	result := make([]int, len(path))
	copy(result, path)
	result[len(result)-1]++
	return result
}

// anyMapToStepResults converts a map[string]any to map[string]wf.StepResult
// for use with ShouldPersistSnapshotParams.
func anyMapToStepResults(m map[string]any) map[string]wf.StepResult {
	if m == nil {
		return nil
	}
	result := make(map[string]wf.StepResult, len(m))
	for k, v := range m {
		if k == "__state" {
			continue
		}
		switch sr := v.(type) {
		case wf.StepResult:
			result[k] = sr
		case map[string]any:
			r := wf.StepResult{}
			if s, ok := sr["status"].(string); ok {
				r.Status = wf.WorkflowStepStatus(s)
			}
			r.Output = sr["output"]
			r.Payload = sr["payload"]
			r.ResumePayload = sr["resumePayload"]
			r.SuspendPayload = sr["suspendPayload"]
			r.SuspendOutput = sr["suspendOutput"]
			if sa, ok := sr["startedAt"].(int64); ok {
				r.StartedAt = sa
			} else if sa, ok := sr["startedAt"].(float64); ok {
				r.StartedAt = int64(sa)
			}
			if ea, ok := sr["endedAt"].(int64); ok {
				r.EndedAt = ea
			} else if ea, ok := sr["endedAt"].(float64); ok {
				r.EndedAt = int64(ea)
			}
			result[k] = r
		}
	}
	return result
}
