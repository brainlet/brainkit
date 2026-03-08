// Ported from: packages/core/src/workflows/evented/workflow.ts
package evented

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	streamPkg "github.com/brainlet/brainkit/agent-kit/core/stream"
	wf "github.com/brainlet/brainkit/agent-kit/core/workflows"
)

// ---------------------------------------------------------------------------
// EventedEngineType - marker type for evented workflows
// ---------------------------------------------------------------------------

// EventedEngineType is the engine type marker for evented workflows.
// In TS this is: export type EventedEngineType = {};
type EventedEngineType struct{}

// ---------------------------------------------------------------------------
// cloneWorkflow / cloneStep
// ---------------------------------------------------------------------------

// CloneWorkflow creates a new workflow with a different ID but the same
// step graph, schemas, and configuration.
// TS equivalent: export function cloneWorkflow(workflow, opts)
func CloneWorkflow(source *EventedWorkflow, newID string) *EventedWorkflow {
	cloned := NewEventedWorkflow(EventedWorkflowConfig{
		ID:           newID,
		InputSchema:  source.InputSchema,
		OutputSchema: source.OutputSchema,
		Steps:        source.StepDefs,
		Options:      source.Options,
	})
	cloned.StepGraph = source.StepGraph
	cloned.Committed = true
	return cloned
}

// CloneStep creates a new step with a different ID but the same configuration.
// TS equivalent: export function cloneStep(step, opts)
func CloneStep(step *wf.Step, newID string) *wf.Step {
	return &wf.Step{
		ID:            newID,
		Description:   step.Description,
		InputSchema:   step.InputSchema,
		OutputSchema:  step.OutputSchema,
		SuspendSchema: step.SuspendSchema,
		ResumeSchema:  step.ResumeSchema,
		StateSchema:   step.StateSchema,
		Execute:       step.Execute,
		Retries:       step.Retries,
		Scorers:       step.Scorers,
		Metadata:      step.Metadata,
		Component:     step.Component,
	}
}

// ---------------------------------------------------------------------------
// CreateStep
// ---------------------------------------------------------------------------

// CreateStep creates a step from explicit parameters.
// In Go, we only support the StepParams variant (no Agent/Tool/Processor overloads).
// TS equivalent: export function createStep(params): Step
func CreateStep(params wf.StepParams) *wf.Step {
	step := &wf.Step{
		ID:            params.ID,
		Description:   params.Description,
		InputSchema:   params.InputSchema,
		OutputSchema:  params.OutputSchema,
		StateSchema:   params.StateSchema,
		ResumeSchema:  params.ResumeSchema,
		SuspendSchema: params.SuspendSchema,
		Scorers:       params.Scorers,
		Retries:       params.Retries,
		Metadata:      params.Metadata,
		Execute:       params.Execute,
	}
	return step
}

// ---------------------------------------------------------------------------
// EventedWorkflow
// ---------------------------------------------------------------------------

// EventedWorkflowConfig holds configuration for creating an EventedWorkflow.
type EventedWorkflowConfig struct {
	ID           string
	InputSchema  wf.SchemaWithValidation
	OutputSchema wf.SchemaWithValidation
	StateSchema  wf.SchemaWithValidation
	Steps        map[string]*wf.Step
	Options      *wf.WorkflowOptions
	Mastra       Mastra
}

// WorkflowOptions holds workflow-level options.
// TODO: Merge with wf.WorkflowOptions when types align.
type WorkflowOptions = wf.WorkflowOptions

// EventedWorkflow is a workflow that uses the evented execution engine.
// It extends the base Workflow concept with evented-specific behavior.
// TS equivalent: export class EventedWorkflow extends Workflow
type EventedWorkflow struct {
	ID                  string
	InputSchema         wf.SchemaWithValidation
	OutputSchema        wf.SchemaWithValidation
	StateSchema         wf.SchemaWithValidation
	StepDefs            map[string]*wf.Step
	StepGraph           []wf.StepFlowEntry
	SerializedStepGraph []wf.SerializedStepFlowEntry
	ExecutionGraph      wf.ExecutionGraph
	RetryConfig         wf.RetryConfig
	Options             *wf.WorkflowOptions
	EngineType          string
	Committed           bool

	mastra          Mastra
	executionEngine *EventedExecutionEngine
	runs            sync.Map // map[string]*EventedRun
	log             logger.IMastraLogger
}

// NewEventedWorkflow creates a new EventedWorkflow.
func NewEventedWorkflow(config EventedWorkflowConfig) *EventedWorkflow {
	ew := &EventedWorkflow{
		ID:           config.ID,
		InputSchema:  config.InputSchema,
		OutputSchema: config.OutputSchema,
		StateSchema:  config.StateSchema,
		StepDefs:     config.Steps,
		Options:      config.Options,
		EngineType:   "evented",
		mastra:       config.Mastra,
	}
	if config.Steps == nil {
		ew.StepDefs = make(map[string]*wf.Step)
	}
	return ew
}

// GetID returns the workflow ID.
func (ew *EventedWorkflow) GetID() string {
	return ew.ID
}

// GetStepGraph returns the step graph.
func (ew *EventedWorkflow) GetStepGraph() []wf.StepFlowEntry {
	return ew.StepGraph
}

// GetSerializedStepGraph returns the serialized step graph.
func (ew *EventedWorkflow) GetSerializedStepGraph() []wf.SerializedStepFlowEntry {
	return ew.SerializedStepGraph
}

// RegisterMastra registers the mastra instance.
// TS equivalent: __registerMastra(mastra: Mastra)
func (ew *EventedWorkflow) RegisterMastra(mastra Mastra) {
	ew.mastra = mastra
	if mastra != nil {
		ew.log = mastra.GetLogger()
	}
	if ew.executionEngine != nil {
		ew.executionEngine.RegisterMastra(mastra)
	}
}

// SetStepFlow sets the step flow entries.
func (ew *EventedWorkflow) SetStepFlow(steps []wf.StepFlowEntry) {
	ew.StepGraph = steps
}

// Commit finalizes the workflow configuration and builds the execution graph.
func (ew *EventedWorkflow) Commit() {
	ew.ExecutionGraph = wf.ExecutionGraph{
		ID:    ew.ID,
		Steps: ew.StepGraph,
	}
	ew.Committed = true
}

// BuildExecutionGraph builds and returns the execution graph.
func (ew *EventedWorkflow) BuildExecutionGraph() wf.ExecutionGraph {
	return wf.ExecutionGraph{
		ID:    ew.ID,
		Steps: ew.StepGraph,
	}
}

// CreateRun creates a new workflow run.
// TS equivalent: async createRun(options?)
func (ew *EventedWorkflow) CreateRun(opts *RunOptions) (*EventedRun, error) {
	runID := ""
	if opts != nil && opts.RunID != "" {
		runID = opts.RunID
	}
	if runID == "" {
		runID = generateUUID()
	}

	// Check for existing run
	if existing, ok := ew.runs.Load(runID); ok {
		return existing.(*EventedRun), nil
	}

	// Check concurrent updates support if storage is available
	if ew.mastra != nil {
		storage := ew.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				// TS: supportsConcurrentUpdates check
				// In Go we check via a type assertion for an optional interface method
				type concurrentChecker interface {
					SupportsConcurrentUpdates() bool
				}
				if checker, ok := store.(concurrentChecker); ok {
					if !checker.SupportsConcurrentUpdates() {
						return nil, errors.New("atomic storage operations are not supported for this workflow store; please use a different storage or the default workflow engine")
					}
				}
			}
		}
	}

	resourceID := ""
	if opts != nil {
		resourceID = opts.ResourceID
	}

	run := NewEventedRun(EventedRunConfig{
		WorkflowID:          ew.ID,
		RunID:               runID,
		ResourceID:          resourceID,
		ExecutionEngine:     ew.executionEngine,
		ExecutionGraph:      ew.ExecutionGraph,
		SerializedStepGraph: ew.SerializedStepGraph,
		Mastra:              ew.mastra,
		RetryConfig:         &ew.RetryConfig,
		Cleanup: func() {
			ew.runs.Delete(runID)
		},
		ValidateInputs: ew.Options != nil && ew.Options.ValidateInputs != nil && *ew.Options.ValidateInputs,
		InputSchema:    ew.InputSchema,
		StateSchema:    ew.StateSchema,
	})

	ew.runs.Store(runID, run)

	// Persist initial snapshot to storage if configured
	if ew.mastra != nil {
		storage := ew.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				// Check if snapshot persistence is desired
				shouldPersist := true
				if ew.Options != nil && ew.Options.ShouldPersistSnapshot != nil {
					shouldPersist = ew.Options.ShouldPersistSnapshot(wf.ShouldPersistSnapshotParams{
						WorkflowStatus: run.WorkflowRunStatus,
						StepResults:    map[string]wf.StepResult{},
					})
				}

				if shouldPersist {
					// Check if the run already exists in storage
					existingRun, getErr := store.GetWorkflowRunByID(GetRunParams{
						RunID:        runID,
						WorkflowName: ew.ID,
					})

					// Sync status from storage to in-memory run
					if getErr == nil && existingRun != nil {
						// Run exists in storage, don't re-persist
					} else {
						// Run doesn't exist in storage, persist initial snapshot
						_ = store.PersistWorkflowSnapshot(PersistSnapshotParams{
							WorkflowName: ew.ID,
							RunID:        runID,
							ResourceID:   resourceID,
							Snapshot: WorkflowRunState{
								RunID:               runID,
								Status:              "pending",
								Value:               map[string]any{},
								Context:             map[string]any{},
								ActivePaths:         []any{},
								SerializedStepGraph: ew.SerializedStepGraph,
								ActiveStepsPath:     map[string]any{},
								SuspendedPaths:      map[string][]int{},
								ResumeLabels:        map[string]any{},
								WaitingPaths:        map[string][]int{},
								Timestamp:           time.Now().UnixMilli(),
							},
						})
					}
				}
			}
		}
	}

	return run, nil
}

// RunOptions holds options for creating a workflow run.
type RunOptions struct {
	RunID          string
	ResourceID     string
	DisableScorers bool
}

// ---------------------------------------------------------------------------
// EventedRun
// ---------------------------------------------------------------------------

// EventedRunConfig holds configuration for creating an EventedRun.
type EventedRunConfig struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	ExecutionEngine     *EventedExecutionEngine
	ExecutionGraph      wf.ExecutionGraph
	SerializedStepGraph []wf.SerializedStepFlowEntry
	Mastra              Mastra
	RetryConfig         *wf.RetryConfig
	Cleanup             func()
	ValidateInputs      bool
	InputSchema         wf.SchemaWithValidation
	StateSchema         wf.SchemaWithValidation
}

// EventedRun represents a single execution of an EventedWorkflow.
// TS equivalent: export class EventedRun extends Run
type EventedRun struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	SerializedStepGraph []wf.SerializedStepFlowEntry
	WorkflowRunStatus   wf.WorkflowRunStatus

	executionEngine *EventedExecutionEngine
	executionGraph  wf.ExecutionGraph
	mastra          Mastra
	retryConfig     *wf.RetryConfig
	cleanup         func()
	validateInputs  bool
	inputSchema     wf.SchemaWithValidation
	stateSchema     wf.SchemaWithValidation
	abortCtx        context.Context
	abortCancel     context.CancelFunc
	cancelOnce      sync.Once // guards workflow.cancel publish to prevent duplicates

	// Streaming fields
	closeStreamAction func()
	streamOutput      *streamPkg.WorkflowRunOutput
	executionResults  map[string]any
}

// NewEventedRun creates a new EventedRun.
func NewEventedRun(config EventedRunConfig) *EventedRun {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventedRun{
		WorkflowID:          config.WorkflowID,
		RunID:               config.RunID,
		ResourceID:          config.ResourceID,
		SerializedStepGraph: config.SerializedStepGraph,
		WorkflowRunStatus:   wf.WorkflowRunStatusPending,
		executionEngine:     config.ExecutionEngine,
		executionGraph:      config.ExecutionGraph,
		mastra:              config.Mastra,
		retryConfig:         config.RetryConfig,
		cleanup:             config.Cleanup,
		validateInputs:      config.ValidateInputs,
		inputSchema:         config.InputSchema,
		stateSchema:         config.StateSchema,
		abortCtx:            ctx,
		abortCancel:         cancel,
	}
}

// setupAbortHandler sets up a context done handler that publishes
// workflow.cancel event when the abort context is canceled.
// TS equivalent: private setupAbortHandler()
func (r *EventedRun) setupAbortHandler() {
	go func() {
		<-r.abortCtx.Done()
		r.publishCancelEvent()
	}()
}

// publishCancelEvent publishes the workflow.cancel event exactly once.
func (r *EventedRun) publishCancelEvent() {
	r.cancelOnce.Do(func() {
		if r.mastra != nil {
			pubsub := r.mastra.PubSub()
			if pubsub != nil {
				_ = pubsub.Publish("workflows", map[string]any{
					"type":  "workflow.cancel",
					"runId": r.RunID,
					"data": map[string]any{
						"workflowId": r.WorkflowID,
						"runId":      r.RunID,
					},
				})
			}
		}
	})
}

// Start begins execution of the workflow run.
// TS equivalent: async start({inputData, initialState, requestContext, perStep, outputOptions})
func (r *EventedRun) Start(params StartParams) (map[string]any, error) {
	if len(r.SerializedStepGraph) == 0 {
		return nil, errors.New("execution flow of workflow is not defined; add steps via .Then(), .Branch(), etc.")
	}

	if r.executionEngine == nil {
		return nil, errors.New("execution engine not configured")
	}

	if r.executionGraph.Steps == nil {
		return nil, errors.New("uncommitted step flow changes detected; call .Commit() to register the steps")
	}

	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Persist initial running snapshot to storage
	if r.mastra != nil {
		storage := r.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				_ = store.PersistWorkflowSnapshot(PersistSnapshotParams{
					WorkflowName: r.WorkflowID,
					RunID:        r.RunID,
					ResourceID:   r.ResourceID,
					Snapshot: WorkflowRunState{
						RunID:               r.RunID,
						SerializedStepGraph: r.SerializedStepGraph,
						Status:              "running",
						Value:               map[string]any{},
						Context:             map[string]any{},
						RequestContext:      rc.All(),
						ActivePaths:         []any{},
						ActiveStepsPath:     map[string]any{},
						SuspendedPaths:      map[string][]int{},
						ResumeLabels:        map[string]any{},
						WaitingPaths:        map[string][]int{},
						Timestamp:           time.Now().UnixMilli(),
					},
				})
			}
		}
	}

	// Validate inputs
	inputData := params.InputData
	if r.validateInputs && r.inputSchema != nil {
		result, err := r.inputSchema.SafeParse(inputData)
		if err != nil {
			return nil, fmt.Errorf("invalid input data: %w", err)
		}
		if result != nil && result.Success && result.Data != nil {
			inputData = result.Data
		} else if result != nil && !result.Success && result.Error != nil {
			return nil, fmt.Errorf("invalid input data: %w", result.Error)
		}
	}

	initialState := params.InitialState
	if r.validateInputs && r.stateSchema != nil {
		result, err := r.stateSchema.SafeParse(initialState)
		if err != nil {
			return nil, fmt.Errorf("invalid initial state: %w", err)
		}
		if result != nil && result.Success && result.Data != nil {
			initialState = result.Data
		} else if result != nil && !result.Success && result.Error != nil {
			return nil, fmt.Errorf("invalid initial state: %w", result.Error)
		}
	}

	// Require pubsub
	if r.mastra == nil || r.mastra.PubSub() == nil {
		return nil, errors.New("mastra instance with pubsub is required for workflow execution")
	}

	r.setupAbortHandler()

	// Note: PubSub is NOT passed here because the EventedExecutionEngine
	// fetches it internally from mastra.PubSub(). The evented PubSub
	// interface is not compatible with events.PubSub.
	result, err := r.executionEngine.Execute(wf.ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		ResourceID:          r.ResourceID,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		Input:               inputData,
		InitialState:        initialState,
		RetryConfig:         r.retryConfig,
		RequestContext:      rc,
		AbortCtx:            r.abortCtx,
		AbortCancel:         r.abortCancel,
		PerStep:             params.PerStep,
		OutputOptions:       params.OutputOptions,
	})
	if err != nil {
		return nil, err
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	// Update run status
	if status, ok := resultMap["status"].(string); ok {
		r.WorkflowRunStatus = wf.WorkflowRunStatus(status)
	}

	if r.WorkflowRunStatus != wf.WorkflowRunStatusSuspended {
		if r.cleanup != nil {
			r.cleanup()
		}
	}

	return resultMap, nil
}

// StartAsync starts the workflow execution without waiting for completion (fire-and-forget).
// Returns immediately with the runId. The workflow executes in the background via pubsub.
// TS equivalent: async startAsync({inputData, initialState, requestContext, perStep})
func (r *EventedRun) StartAsync(params StartParams) (string, error) {
	if len(r.SerializedStepGraph) == 0 {
		return "", errors.New("execution flow of workflow is not defined; add steps via .Then(), .Branch(), etc.")
	}

	if r.executionGraph.Steps == nil {
		return "", errors.New("uncommitted step flow changes detected; call .Commit() to register the steps")
	}

	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Persist initial running snapshot to storage
	if r.mastra != nil {
		storage := r.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				_ = store.PersistWorkflowSnapshot(PersistSnapshotParams{
					WorkflowName: r.WorkflowID,
					RunID:        r.RunID,
					ResourceID:   r.ResourceID,
					Snapshot: WorkflowRunState{
						RunID:               r.RunID,
						SerializedStepGraph: r.SerializedStepGraph,
						Status:              "running",
						Value:               map[string]any{},
						Context:             map[string]any{},
						RequestContext:      rc.All(),
						ActivePaths:         []any{},
						ActiveStepsPath:     map[string]any{},
						SuspendedPaths:      map[string][]int{},
						ResumeLabels:        map[string]any{},
						WaitingPaths:        map[string][]int{},
						Timestamp:           time.Now().UnixMilli(),
					},
				})
			}
		}
	}

	// Validate inputs
	inputData := params.InputData
	if r.validateInputs && r.inputSchema != nil {
		result, err := r.inputSchema.SafeParse(inputData)
		if err != nil {
			return "", fmt.Errorf("invalid input data: %w", err)
		}
		if result != nil && result.Success && result.Data != nil {
			inputData = result.Data
		} else if result != nil && !result.Success && result.Error != nil {
			return "", fmt.Errorf("invalid input data: %w", result.Error)
		}
	}

	initialState := params.InitialState
	if r.validateInputs && r.stateSchema != nil {
		result, err := r.stateSchema.SafeParse(initialState)
		if err != nil {
			return "", fmt.Errorf("invalid initial state: %w", err)
		}
		if result != nil && result.Success && result.Data != nil {
			initialState = result.Data
		} else if result != nil && !result.Success && result.Error != nil {
			return "", fmt.Errorf("invalid initial state: %w", result.Error)
		}
	}

	if r.mastra == nil || r.mastra.PubSub() == nil {
		return "", errors.New("mastra instance with pubsub is required for workflow execution")
	}

	// Fire-and-forget: publish the workflow start event
	pubsub := r.mastra.PubSub()
	_ = pubsub.Publish("workflows", map[string]any{
		"type":  "workflow.start",
		"runId": r.RunID,
		"data": map[string]any{
			"workflowId":   r.WorkflowID,
			"runId":        r.RunID,
			"prevResult":   map[string]any{"status": "success", "output": inputData},
			"initialState": initialState,
			"perStep":      params.PerStep,
		},
	})

	return r.RunID, nil
}

// Resume resumes a suspended workflow run.
// TS equivalent: async resume(params)
func (r *EventedRun) Resume(params ResumeParams) (map[string]any, error) {
	if r.mastra == nil {
		return nil, errors.New("cannot resume workflow: mastra instance is required")
	}

	storage := r.mastra.GetStorage()
	if storage == nil {
		return nil, errors.New("cannot resume workflow: storage is required")
	}

	store, err := storage.GetStore("workflows")
	if err != nil || store == nil {
		return nil, errors.New("cannot resume workflow: workflows store is required")
	}

	snapshot, err := store.LoadWorkflowSnapshot(LoadSnapshotParams{
		WorkflowName: r.WorkflowID,
		RunID:        r.RunID,
	})
	if err != nil || snapshot == nil {
		return nil, fmt.Errorf("cannot resume workflow: no snapshot found for runId %s", r.RunID)
	}

	if snapshot.Status != "suspended" {
		return nil, errors.New("this workflow run was not suspended")
	}

	// Resolve label to step path if provided
	var snapshotResumeLabel map[string]any
	if params.Label != "" && snapshot.ResumeLabels != nil {
		if rl, ok := snapshot.ResumeLabels[params.Label]; ok {
			if rlMap, ok := rl.(map[string]any); ok {
				snapshotResumeLabel = rlMap
			}
		}
	}

	// Validate label exists if provided
	if params.Label != "" && snapshotResumeLabel == nil {
		availableLabels := make([]string, 0, len(snapshot.ResumeLabels))
		for k := range snapshot.ResumeLabels {
			availableLabels = append(availableLabels, k)
		}
		return nil, fmt.Errorf("resume label %q not found. Available labels: %v", params.Label, availableLabels)
	}

	// Label takes precedence over step param
	stepParam := params.Step
	if snapshotResumeLabel != nil {
		if stepID, ok := snapshotResumeLabel["stepId"].(string); ok && stepID != "" {
			stepParam = []string{stepID}
		}
	}

	// Determine the step(s) to resume
	steps := stepParam
	if len(steps) == 0 {
		// Auto-detect suspended steps from suspendedPaths
		suspendedStepPaths := [][]string{}
		for stepID := range snapshot.SuspendedPaths {
			stepResult, ok := snapshot.Context[stepID]
			if ok {
				if m, ok := stepResult.(map[string]any); ok {
					if status, ok := m["status"].(string); ok && status == "suspended" {
						nestedPath := getNestedPath(m)
						if nestedPath != nil {
							suspendedStepPaths = append(suspendedStepPaths, append([]string{stepID}, nestedPath...))
						} else {
							suspendedStepPaths = append(suspendedStepPaths, []string{stepID})
						}
					}
				}
			}
		}

		if len(suspendedStepPaths) == 0 {
			return nil, errors.New("no suspended steps found in this workflow run")
		}
		if len(suspendedStepPaths) == 1 {
			steps = suspendedStepPaths[0]
		} else {
			return nil, errors.New("multiple suspended steps found. Please specify which step to resume using the 'step' parameter")
		}
	}

	// Validate the step is actually suspended
	if len(steps) > 0 {
		suspendedStepIDs := make([]string, 0, len(snapshot.SuspendedPaths))
		for k := range snapshot.SuspendedPaths {
			suspendedStepIDs = append(suspendedStepIDs, k)
		}
		isStepSuspended := false
		for _, id := range suspendedStepIDs {
			if id == steps[0] {
				isStepSuspended = true
				break
			}
		}
		if !isStepSuspended {
			return nil, fmt.Errorf("this workflow step %q was not suspended. Available suspended steps: %v", steps[0], suspendedStepIDs)
		}
	}

	// Build request context: snapshot values first, then override with new values
	rc := requestcontext.NewRequestContext()
	if snapshot.RequestContext != nil {
		for key, value := range snapshot.RequestContext {
			rc.Set(key, value)
		}
	}
	if params.RequestContext != nil {
		for key, value := range params.RequestContext.All() {
			rc.Set(key, value)
		}
	}

	var resumePath []int
	if len(steps) > 0 {
		if paths, ok := snapshot.SuspendedPaths[steps[0]]; ok {
			resumePath = paths
		}
	}

	// Extract state from snapshot
	resumeState := snapshot.Value

	if r.mastra.PubSub() == nil {
		return nil, errors.New("mastra instance with pubsub is required for workflow execution")
	}

	r.setupAbortHandler()

	// Determine forEachIndex
	var forEachIndex *int
	if params.ForEachIndex != nil {
		forEachIndex = params.ForEachIndex
	} else if snapshotResumeLabel != nil {
		if fi, ok := snapshotResumeLabel["foreachIndex"]; ok {
			if fiFloat, ok := fi.(float64); ok {
				fiInt := int(fiFloat)
				forEachIndex = &fiInt
			} else if fiInt, ok := fi.(int); ok {
				forEachIndex = &fiInt
			}
		}
	}

	// Convert snapshot context to step results
	stepResults := make(map[string]wf.StepResult)
	for k, v := range snapshot.Context {
		if m, ok := v.(map[string]any); ok {
			sr := wf.StepResult{}
			if s, ok := m["status"].(string); ok {
				sr.Status = wf.WorkflowStepStatus(s)
			}
			sr.Output = m["output"]
			sr.Payload = m["payload"]
			sr.ResumePayload = m["resumePayload"]
			sr.SuspendPayload = m["suspendPayload"]
			sr.SuspendOutput = m["suspendOutput"]
			stepResults[k] = sr
		} else if sr, ok := v.(wf.StepResult); ok {
			stepResults[k] = sr
		}
	}

	// Note: PubSub is NOT passed here because the EventedExecutionEngine
	// fetches it internally from mastra.PubSub().
	result, err := r.executionEngine.Execute(wf.ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		Input:               snapshot.Context["input"],
		InitialState:        resumeState,
		Resume: &wf.ResumeExecuteParams{
			Steps:         steps,
			StepResults:   stepResults,
			ResumePayload: params.ResumeData,
			ResumePath:    resumePath,
			ForEachIndex:  forEachIndex,
		},
		RequestContext: rc,
		AbortCtx:       r.abortCtx,
		AbortCancel:    r.abortCancel,
		PerStep:        params.PerStep,
		OutputOptions:  params.OutputOptions,
	})
	if err != nil {
		return nil, err
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}

	if status, ok := resultMap["status"].(string); ok {
		r.WorkflowRunStatus = wf.WorkflowRunStatus(status)
	}

	if r.WorkflowRunStatus != wf.WorkflowRunStatusSuspended {
		if r.closeStreamAction != nil {
			r.closeStreamAction()
		}
	}

	return resultMap, nil
}

// Stream starts the workflow execution and returns a WorkflowRunOutput with a
// channel-based stream of WorkflowStreamEvent chunks.
// TS equivalent: stream({inputData, requestContext, closeOnSuspend, ...})
func (r *EventedRun) Stream(params StreamParams) *streamPkg.WorkflowRunOutput {
	closeOnSuspend := params.CloseOnSuspend

	if r.closeStreamAction != nil && r.streamOutput != nil {
		return r.streamOutput
	}

	r.closeStreamAction = func() {} // placeholder

	ch := make(chan streamPkg.WorkflowStreamEvent, 256)

	unwatch := r.Watch(func(event streamPkg.WorkflowStreamEvent) {
		ch <- event
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// Start execution in background
	go func() {
		result, err := r.Start(StartParams{
			InputData:      params.InputData,
			InitialState:   params.InitialState,
			RequestContext: params.RequestContext,
			PerStep:        params.PerStep,
			OutputOptions:  params.OutputOptions,
		})
		if err != nil {
			if r.streamOutput != nil {
				r.streamOutput.RejectResults(err)
			}
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
			return
		}
		r.executionResults = result
		if r.streamOutput != nil {
			r.streamOutput.UpdateResults(result)
		}
		if closeOnSuspend {
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
		} else {
			status, _ := result["status"].(string)
			if status != string(wf.WorkflowRunStatusSuspended) {
				if r.closeStreamAction != nil {
					r.closeStreamAction()
				}
			}
		}
	}()

	r.streamOutput = streamPkg.NewWorkflowRunOutput(streamPkg.WorkflowRunOutputParams{
		RunID:      r.RunID,
		WorkflowID: r.WorkflowID,
		Stream:     ch,
	})

	return r.streamOutput
}

// ResumeStream resumes a suspended workflow and returns a streaming output.
// TS equivalent: resumeStream({step, resumeData, requestContext, ...})
func (r *EventedRun) ResumeStream(params ResumeStreamParams) *streamPkg.WorkflowRunOutput {
	r.closeStreamAction = func() {} // placeholder

	ch := make(chan streamPkg.WorkflowStreamEvent, 256)

	unwatch := r.Watch(func(event streamPkg.WorkflowStreamEvent) {
		ch <- event
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// Resume execution in background
	go func() {
		result, err := r.Resume(ResumeParams{
			ResumeData:     params.ResumeData,
			Step:           params.Step,
			RequestContext: params.RequestContext,
			PerStep:        params.PerStep,
			OutputOptions:  params.OutputOptions,
		})
		if err != nil {
			if r.streamOutput != nil {
				r.streamOutput.RejectResults(err)
			}
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
			return
		}
		r.executionResults = result
		if r.streamOutput != nil {
			r.streamOutput.UpdateResults(result)
		}
		// Allow pending events to flush
		time.Sleep(time.Millisecond)
		if r.closeStreamAction != nil {
			r.closeStreamAction()
		}
	}()

	r.streamOutput = streamPkg.NewWorkflowRunOutput(streamPkg.WorkflowRunOutputParams{
		RunID:      r.RunID,
		WorkflowID: r.WorkflowID,
		Stream:     ch,
	})

	return r.streamOutput
}

// Watch subscribes to workflow events for this run.
// Returns an unwatch function.
// TS equivalent: watch(cb)
func (r *EventedRun) Watch(cb func(event streamPkg.WorkflowStreamEvent)) func() {
	if r.mastra == nil {
		return func() {}
	}

	pubsub := r.mastra.PubSub()
	if pubsub == nil {
		return func() {}
	}

	topic := fmt.Sprintf("workflow.events.v2.%s", r.RunID)

	handler := func(event any, ack func() error) error {
		if evt, ok := event.(map[string]any); ok {
			runID, _ := evt["runId"].(string)
			if runID != r.RunID {
				return nil
			}
			if data, ok := evt["data"]; ok {
				chunk := anyToStreamEvent(data, r.RunID)
				cb(chunk)
			}
		}
		if ack != nil {
			return ack()
		}
		return nil
	}

	_ = pubsub.Subscribe(topic, handler)

	return func() {
		_ = pubsub.Unsubscribe(topic, handler)
	}
}

// WatchAsync subscribes to workflow events for this run (async variant).
// TS equivalent: async watchAsync(cb)
func (r *EventedRun) WatchAsync(cb func(event streamPkg.WorkflowStreamEvent)) func() {
	return r.Watch(cb)
}

// Cancel cancels the workflow run.
// TS equivalent: async cancel()
func (r *EventedRun) Cancel() {
	// Update storage directly for immediate status update
	if r.mastra != nil {
		storage := r.mastra.GetStorage()
		if storage != nil {
			store, err := storage.GetStore("workflows")
			if err == nil && store != nil {
				_ = store.UpdateWorkflowState(UpdateStateParams{
					WorkflowName: r.WorkflowID,
					RunID:        r.RunID,
					Opts:         map[string]any{"status": "canceled"},
				})
			}
		}
	}

	// Publish workflow.cancel event. Uses sync.Once so this is safe even when
	// setupAbortHandler's goroutine also fires (after Start was called).
	r.publishCancelEvent()

	// Trigger abort signal
	if r.abortCancel != nil {
		r.abortCancel()
	}
}

// ---------------------------------------------------------------------------
// Param types
// ---------------------------------------------------------------------------

// StartParams holds parameters for starting a workflow run.
type StartParams struct {
	InputData      any
	InitialState   any
	RequestContext *requestcontext.RequestContext
	PerStep        bool
	OutputOptions  *wf.OutputOptions
}

// ResumeParams holds parameters for resuming a suspended workflow run.
type ResumeParams struct {
	ResumeData     any
	Step           []string
	Label          string
	RequestContext *requestcontext.RequestContext
	ForEachIndex   *int
	PerStep        bool
	OutputOptions  *wf.OutputOptions
}

// StreamParams holds parameters for streaming a workflow execution.
type StreamParams struct {
	InputData      any
	InitialState   any
	RequestContext *requestcontext.RequestContext
	CloseOnSuspend bool
	PerStep        bool
	OutputOptions  *wf.OutputOptions
}

// ResumeStreamParams holds parameters for streaming a resume operation.
type ResumeStreamParams struct {
	ResumeData     any
	Step           []string
	RequestContext *requestcontext.RequestContext
	PerStep        bool
	OutputOptions  *wf.OutputOptions
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// generateUUID generates a random UUID string.
// TS equivalent: randomUUID() from 'node:crypto'
func generateUUID() string {
	return uuid.New().String()
}

// getNestedPath extracts the nested workflow path from a step result map.
func getNestedPath(m map[string]any) []string {
	sp, ok := m["suspendPayload"]
	if !ok {
		return nil
	}
	spMap, ok := sp.(map[string]any)
	if !ok {
		return nil
	}
	meta, ok := spMap["__workflow_meta"]
	if !ok {
		return nil
	}
	metaMap, ok := meta.(map[string]any)
	if !ok {
		return nil
	}
	path, ok := metaMap["path"]
	if !ok {
		return nil
	}
	pathSlice, ok := path.([]any)
	if !ok {
		return nil
	}
	result := make([]string, len(pathSlice))
	for i, p := range pathSlice {
		result[i] = fmt.Sprintf("%v", p)
	}
	return result
}

// anyToStreamEvent converts an any value into a streamPkg.WorkflowStreamEvent.
func anyToStreamEvent(data any, runID string) streamPkg.WorkflowStreamEvent {
	chunk := streamPkg.WorkflowStreamEvent{}
	chunk.RunID = runID
	chunk.From = streamPkg.ChunkFromWorkflow

	if c, ok := data.(streamPkg.WorkflowStreamEvent); ok {
		return c
	}
	if m, ok := data.(map[string]any); ok {
		if t, ok := m["type"].(string); ok {
			chunk.Type = t
		}
		if payload, ok := m["payload"]; ok {
			chunk.Payload = payload
		}
		if from, ok := m["from"].(string); ok {
			chunk.From = streamPkg.ChunkFrom(from)
		}
	}
	return chunk
}
