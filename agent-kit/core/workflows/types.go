// Ported from: packages/core/src/workflows/types.ts
package workflows

import (
	"context"
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/evals"
	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// Mastra is the top-level orchestrator for the framework.
// Defined here (not imported from core/mastra) to break circular dependency.
// core.Mastra satisfies this interface.
//
// NOTE: GetStorage() still uses a local Storage interface because the real
// *storage.MastraCompositeStore.GetStore takes DomainName (not string) and
// returns domains.StorageDomain (not WorkflowsStore). This structural mismatch
// requires a deeper refactor to resolve.
type Mastra interface {
	GetLogger() logger.IMastraLogger
	GetStorage() Storage
	GenerateID(ctx *GenerateIDOpts) string
	PubSub() events.PubSub
}

// Storage is the persistence layer interface.
// TODO: import from storage package once ported.
type Storage interface {
	GetStore(name string) (WorkflowsStore, error)
}

// WorkflowsStore handles workflow persistence.
// TODO: import from storage package once ported.
type WorkflowsStore interface {
	PersistWorkflowSnapshot(params PersistWorkflowSnapshotParams) error
	GetWorkflowRunByID(params GetWorkflowRunByIDParams) (*WorkflowRunRecord, error)
	UpdateWorkflowState(params UpdateWorkflowStateParams) error
	DeleteWorkflowRunByID(params DeleteWorkflowRunByIDParams) error
	ListWorkflowRuns(params ListWorkflowRunsParams) (*WorkflowRunList, error)
}

// PersistWorkflowSnapshotParams holds parameters for persisting a workflow snapshot.
type PersistWorkflowSnapshotParams struct {
	WorkflowName string
	RunID        string
	ResourceID   string
	Snapshot     WorkflowRunState
}

// GetWorkflowRunByIDParams holds parameters for getting a workflow run.
type GetWorkflowRunByIDParams struct {
	RunID        string
	WorkflowName string
}

// UpdateWorkflowStateParams holds parameters for updating workflow state.
type UpdateWorkflowStateParams struct {
	WorkflowName string
	RunID        string
	Opts         map[string]any
}

// DeleteWorkflowRunByIDParams holds parameters for deleting a workflow run.
type DeleteWorkflowRunByIDParams struct {
	RunID        string
	WorkflowName string
}

// ListWorkflowRunsParams holds parameters for listing workflow runs.
type ListWorkflowRunsParams struct {
	WorkflowName string
	Status       string
}

// WorkflowRunRecord represents a persisted workflow run.
type WorkflowRunRecord struct {
	RunID        string
	WorkflowName string
	ResourceID   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Snapshot     *WorkflowRunState
}

// WorkflowRunList represents a list of workflow runs.
type WorkflowRunList struct {
	Runs  []WorkflowRunRecord
	Total int
}

// GenerateIDOpts is an alias for types.IdGeneratorContext.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type GenerateIDOpts = aktypes.IdGeneratorContext

// SchemaWithValidation is a generic schema type that supports validation.
// TODO: import from stream/base/schema package once ported.
type SchemaWithValidation interface {
	Validate(data any) error
	SafeParse(data any) (*ParseResult, error)
}

// ParseResult represents the result of a schema validation.
type ParseResult struct {
	Success bool
	Data    any
	Error   error
}

// DynamicArgument can be a static value or a function that resolves at runtime.
// TODO: import from types package once ported.
type DynamicArgument[T any] interface{}

// MastraScorers is imported from the evals package.
type MastraScorers = evals.MastraScorers

// ---------------------------------------------------------------------------
// OutputWriter
// ---------------------------------------------------------------------------

// OutputWriter writes chunks to an output stream.
// TS equivalent: export type OutputWriter<TChunk = any> = (chunk: TChunk) => Promise<void>;
type OutputWriter func(chunk any) error

// ---------------------------------------------------------------------------
// WorkflowRunStartOptions
// ---------------------------------------------------------------------------

// WorkflowRunStartOptions are options for Run.Start() beyond inputData/initialState/requestContext.
type WorkflowRunStartOptions struct {
	OutputWriter   OutputWriter
	TracingOptions *obstypes.TracingOptions
	OutputOptions  *OutputOptions
	PerStep        bool
	// ObservabilityContext fields
	Observability *obstypes.ObservabilityContext
}

// OutputOptions controls what extra data is included in the workflow result.
type OutputOptions struct {
	IncludeState        bool
	IncludeResumeLabels bool
}

// ---------------------------------------------------------------------------
// WorkflowEngineType / WorkflowType
// ---------------------------------------------------------------------------

// WorkflowEngineType identifies the execution engine type.
type WorkflowEngineType = string

// WorkflowType determines how the workflow is categorized in the UI.
type WorkflowType string

const (
	// WorkflowTypeDefault is a standard workflow.
	WorkflowTypeDefault WorkflowType = "default"
	// WorkflowTypeProcessor is a workflow used as a processor for agent input/output processing.
	WorkflowTypeProcessor WorkflowType = "processor"
)

// ---------------------------------------------------------------------------
// Restart / TimeTravel Execution Params
// ---------------------------------------------------------------------------

// RestartExecutionParams holds parameters for restarting workflow execution.
type RestartExecutionParams struct {
	ActivePaths      []int                        `json:"activePaths"`
	ActiveStepsPath  map[string][]int             `json:"activeStepsPath"`
	StepResults      map[string]StepResult        `json:"stepResults"`
	State            map[string]any               `json:"state,omitempty"`
	StepExecutionPath []string                    `json:"stepExecutionPath,omitempty"`
}

// TimeTravelExecutionParams holds parameters for time-traveling to a specific step.
type TimeTravelExecutionParams struct {
	ExecutionPath     []int                                `json:"executionPath"`
	InputData         any                                  `json:"inputData,omitempty"`
	StepResults       map[string]StepResult                `json:"stepResults"`
	NestedStepResults map[string]map[string]StepResult     `json:"nestedStepResults,omitempty"`
	Steps             []string                             `json:"steps"`
	State             map[string]any                       `json:"state,omitempty"`
	ResumeData        any                                  `json:"resumeData,omitempty"`
	StepExecutionPath []string                             `json:"stepExecutionPath,omitempty"`
}

// ---------------------------------------------------------------------------
// Step Metadata
// ---------------------------------------------------------------------------

// StepMetadata holds arbitrary metadata associated with a step.
type StepMetadata map[string]any

// ---------------------------------------------------------------------------
// Step Result Types
// ---------------------------------------------------------------------------

// WorkflowStepStatus represents the status of a step result.
type WorkflowStepStatus string

const (
	StepStatusSuccess   WorkflowStepStatus = "success"
	StepStatusFailed    WorkflowStepStatus = "failed"
	StepStatusSuspended WorkflowStepStatus = "suspended"
	StepStatusRunning   WorkflowStepStatus = "running"
	StepStatusWaiting   WorkflowStepStatus = "waiting"
	StepStatusPaused    WorkflowStepStatus = "paused"
)

// StepTripwireInfo holds tripwire data attached to a failed step when triggered by a processor.
type StepTripwireInfo struct {
	Reason      string         `json:"reason"`
	Retry       *bool          `json:"retry,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	ProcessorID string         `json:"processorId,omitempty"`
}

// StepResult represents the result of a step execution.
// In TypeScript this is a discriminated union across 6 status variants.
// In Go we use a single struct with the status field as the discriminator.
type StepResult struct {
	Status         WorkflowStepStatus `json:"status"`
	Output         any                `json:"output,omitempty"`
	Payload        any                `json:"payload,omitempty"`
	ResumePayload  any                `json:"resumePayload,omitempty"`
	SuspendPayload any                `json:"suspendPayload,omitempty"`
	SuspendOutput  any                `json:"suspendOutput,omitempty"`
	Error          error              `json:"error,omitempty"`
	StartedAt      int64              `json:"startedAt"`
	EndedAt        int64              `json:"endedAt,omitempty"`
	SuspendedAt    *int64             `json:"suspendedAt,omitempty"`
	ResumedAt      *int64             `json:"resumedAt,omitempty"`
	Metadata       StepMetadata       `json:"metadata,omitempty"`
	// Tripwire holds tripwire data when step failed due to processor rejection.
	Tripwire *StepTripwireInfo `json:"tripwire,omitempty"`
}

// SerializedStepFailure is a step failure where the error is serialized.
// Used when loading workflow runs from storage.
type SerializedStepFailure struct {
	Status         WorkflowStepStatus       `json:"status"`
	Error          *mastraerror.SerializedError `json:"error,omitempty"`
	Payload        any                      `json:"payload,omitempty"`
	ResumePayload  any                      `json:"resumePayload,omitempty"`
	SuspendPayload any                      `json:"suspendPayload,omitempty"`
	SuspendOutput  any                      `json:"suspendOutput,omitempty"`
	StartedAt      int64                    `json:"startedAt"`
	EndedAt        int64                    `json:"endedAt,omitempty"`
	SuspendedAt    *int64                   `json:"suspendedAt,omitempty"`
	ResumedAt      *int64                   `json:"resumedAt,omitempty"`
	Metadata       StepMetadata             `json:"metadata,omitempty"`
	Tripwire       *StepTripwireInfo        `json:"tripwire,omitempty"`
}

// TimeTravelContext maps step IDs to their time-travel state.
type TimeTravelContext map[string]TimeTravelEntry

// TimeTravelEntry is a single entry in a TimeTravelContext.
type TimeTravelEntry struct {
	Status         WorkflowRunStatus `json:"status"`
	Payload        any               `json:"payload,omitempty"`
	Output         any               `json:"output,omitempty"`
	ResumePayload  any               `json:"resumePayload,omitempty"`
	SuspendPayload any               `json:"suspendPayload,omitempty"`
	SuspendOutput  any               `json:"suspendOutput,omitempty"`
	StartedAt      *int64            `json:"startedAt,omitempty"`
	EndedAt        *int64            `json:"endedAt,omitempty"`
	SuspendedAt    *int64            `json:"suspendedAt,omitempty"`
	ResumedAt      *int64            `json:"resumedAt,omitempty"`
	Metadata       StepMetadata      `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Workflow Run Status
// ---------------------------------------------------------------------------

// WorkflowRunStatus represents the overall status of a workflow run.
type WorkflowRunStatus string

const (
	WorkflowRunStatusRunning   WorkflowRunStatus = "running"
	WorkflowRunStatusSuccess   WorkflowRunStatus = "success"
	WorkflowRunStatusFailed    WorkflowRunStatus = "failed"
	WorkflowRunStatusTripwire  WorkflowRunStatus = "tripwire"
	WorkflowRunStatusSuspended WorkflowRunStatus = "suspended"
	WorkflowRunStatusWaiting   WorkflowRunStatus = "waiting"
	WorkflowRunStatusPending   WorkflowRunStatus = "pending"
	WorkflowRunStatusCanceled  WorkflowRunStatus = "canceled"
	WorkflowRunStatusBailed    WorkflowRunStatus = "bailed"
	WorkflowRunStatusPaused    WorkflowRunStatus = "paused"
)

// ---------------------------------------------------------------------------
// Workflow State
// ---------------------------------------------------------------------------

// WorkflowState is the unified workflow state that combines metadata with processed execution state.
type WorkflowState struct {
	// Metadata
	RunID        string    `json:"runId"`
	WorkflowName string    `json:"workflowName"`
	ResourceID   string    `json:"resourceId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`

	// IsFromInMemory indicates whether this result came from in-memory storage rather than persistent storage.
	// When true, the data is approximate:
	// - CreatedAt/UpdatedAt are set to current time
	// - Steps is empty (step data only available from persisted snapshots)
	IsFromInMemory bool `json:"isFromInMemory,omitempty"`

	// Execution State
	Status            WorkflowRunStatus               `json:"status"`
	InitialState      map[string]any                   `json:"initialState,omitempty"`
	StepExecutionPath []string                         `json:"stepExecutionPath,omitempty"`
	ActiveStepsPath   map[string][]int                 `json:"activeStepsPath,omitempty"`
	SerializedStepGraph []SerializedStepFlowEntry      `json:"serializedStepGraph,omitempty"`
	Steps             map[string]WorkflowStateStep      `json:"steps,omitempty"`
	Result            map[string]any                   `json:"result,omitempty"`
	Payload           map[string]any                   `json:"payload,omitempty"`
	Error             *mastraerror.SerializedError      `json:"error,omitempty"`
}

// WorkflowStateStep represents a step within WorkflowState.
type WorkflowStateStep struct {
	Status         WorkflowRunStatus               `json:"status"`
	Output         map[string]any                   `json:"output,omitempty"`
	Payload        map[string]any                   `json:"payload,omitempty"`
	ResumePayload  map[string]any                   `json:"resumePayload,omitempty"`
	Error          *mastraerror.SerializedError      `json:"error,omitempty"`
	StartedAt      int64                            `json:"startedAt"`
	EndedAt        int64                            `json:"endedAt"`
	SuspendedAt    *int64                           `json:"suspendedAt,omitempty"`
	ResumedAt      *int64                           `json:"resumedAt,omitempty"`
}

// WorkflowStateField defines valid field names for filtering WorkflowState responses.
type WorkflowStateField string

const (
	WorkflowStateFieldResult            WorkflowStateField = "result"
	WorkflowStateFieldError             WorkflowStateField = "error"
	WorkflowStateFieldPayload           WorkflowStateField = "payload"
	WorkflowStateFieldSteps             WorkflowStateField = "steps"
	WorkflowStateFieldActiveStepsPath   WorkflowStateField = "activeStepsPath"
	WorkflowStateFieldSerializedStepGraph WorkflowStateField = "serializedStepGraph"
)

// ---------------------------------------------------------------------------
// Workflow Run State (snapshot)
// ---------------------------------------------------------------------------

// WorkflowRunState represents the internal snapshot of a workflow run.
type WorkflowRunState struct {
	RunID               string                               `json:"runId"`
	Status              WorkflowRunStatus                    `json:"status"`
	Result              map[string]any                       `json:"result,omitempty"`
	Error               *mastraerror.SerializedError          `json:"error,omitempty"`
	RequestContext      map[string]any                       `json:"requestContext,omitempty"`
	Value               map[string]string                    `json:"value"`
	Context             map[string]any                       `json:"context"`
	SerializedStepGraph []SerializedStepFlowEntry            `json:"serializedStepGraph"`
	ActivePaths         []int                                `json:"activePaths"`
	ActiveStepsPath     map[string][]int                     `json:"activeStepsPath"`
	SuspendedPaths      map[string][]int                     `json:"suspendedPaths"`
	ResumeLabels        map[string]ResumeLabel               `json:"resumeLabels"`
	WaitingPaths        map[string][]int                     `json:"waitingPaths"`
	Timestamp           int64                                `json:"timestamp"`
	Tripwire            *StepTripwireInfo                    `json:"tripwire,omitempty"`
	StepExecutionPath   []string                             `json:"stepExecutionPath,omitempty"`
}

// ResumeLabel maps a label to a step and optional foreach index.
type ResumeLabel struct {
	StepID       string `json:"stepId"`
	ForeachIndex *int   `json:"foreachIndex,omitempty"`
}

// ---------------------------------------------------------------------------
// Workflow Finish / Error Callback Types
// ---------------------------------------------------------------------------

// WorkflowFinishCallbackResult is the result object passed to the onFinish callback when a workflow completes.
type WorkflowFinishCallbackResult struct {
	Status            WorkflowRunStatus            `json:"status"`
	Result            any                          `json:"result,omitempty"`
	Error             *mastraerror.SerializedError  `json:"error,omitempty"`
	Steps             map[string]StepResult        `json:"steps"`
	Tripwire          *StepTripwireInfo            `json:"tripwire,omitempty"`
	RunID             string                       `json:"runId"`
	WorkflowID        string                       `json:"workflowId"`
	ResourceID        string                       `json:"resourceId,omitempty"`
	GetInitData       func() any                   `json:"-"`
	Mastra            Mastra                       `json:"-"`
	RequestContext    *requestcontext.RequestContext `json:"-"`
	Logger            logger.IMastraLogger          `json:"-"`
	State             map[string]any               `json:"state"`
	StepExecutionPath []string                     `json:"stepExecutionPath,omitempty"`
}

// WorkflowErrorCallbackInfo is the error info object passed to the onError callback when a workflow fails.
type WorkflowErrorCallbackInfo struct {
	Status            WorkflowRunStatus             `json:"status"` // "failed" or "tripwire"
	Error             *mastraerror.SerializedError   `json:"error,omitempty"`
	Steps             map[string]StepResult         `json:"steps"`
	Tripwire          *StepTripwireInfo             `json:"tripwire,omitempty"`
	RunID             string                        `json:"runId"`
	WorkflowID        string                        `json:"workflowId"`
	ResourceID        string                        `json:"resourceId,omitempty"`
	GetInitData       func() any                    `json:"-"`
	Mastra            Mastra                        `json:"-"`
	RequestContext    *requestcontext.RequestContext  `json:"-"`
	Logger            logger.IMastraLogger           `json:"-"`
	State             map[string]any                `json:"state"`
	StepExecutionPath []string                      `json:"stepExecutionPath,omitempty"`
}

// ---------------------------------------------------------------------------
// Workflow Options
// ---------------------------------------------------------------------------

// WorkflowOptions are optional configuration for a workflow.
type WorkflowOptions struct {
	TracingPolicy        *obstypes.TracingPolicy
	ValidateInputs      *bool
	ShouldPersistSnapshot func(params ShouldPersistSnapshotParams) bool
	OnFinish             func(result WorkflowFinishCallbackResult) error
	OnError              func(errorInfo WorkflowErrorCallbackInfo) error
}

// ShouldPersistSnapshotParams are parameters for the ShouldPersistSnapshot callback.
type ShouldPersistSnapshotParams struct {
	StepResults    map[string]StepResult
	WorkflowStatus WorkflowRunStatus
}

// ---------------------------------------------------------------------------
// Workflow Info (serialization)
// ---------------------------------------------------------------------------

// WorkflowInfo contains serialized workflow metadata.
type WorkflowInfo struct {
	Steps                map[string]SerializedStep `json:"steps"`
	AllSteps             map[string]SerializedStep `json:"allSteps"`
	Name                 string                    `json:"name,omitempty"`
	Description          string                    `json:"description,omitempty"`
	StepGraph            []SerializedStepFlowEntry `json:"stepGraph"`
	InputSchema          string                    `json:"inputSchema,omitempty"`
	OutputSchema         string                    `json:"outputSchema,omitempty"`
	StateSchema          string                    `json:"stateSchema,omitempty"`
	RequestContextSchema string                    `json:"requestContextSchema,omitempty"`
	Options              *WorkflowOptions          `json:"options,omitempty"`
	StepCount            *int                      `json:"stepCount,omitempty"`
	IsProcessorWorkflow  bool                      `json:"isProcessorWorkflow,omitempty"`
}

// ---------------------------------------------------------------------------
// Step Flow Entry (runtime graph)
// ---------------------------------------------------------------------------

// StepFlowEntryType identifies the kind of flow entry.
type StepFlowEntryType string

const (
	StepFlowEntryTypeStep        StepFlowEntryType = "step"
	StepFlowEntryTypeSleep       StepFlowEntryType = "sleep"
	StepFlowEntryTypeSleepUntil  StepFlowEntryType = "sleepUntil"
	StepFlowEntryTypeParallel    StepFlowEntryType = "parallel"
	StepFlowEntryTypeConditional StepFlowEntryType = "conditional"
	StepFlowEntryTypeLoop        StepFlowEntryType = "loop"
	StepFlowEntryTypeForeach     StepFlowEntryType = "foreach"
)

// LoopType is the kind of loop ("dowhile" or "dountil").
type LoopType string

const (
	LoopTypeDoWhile LoopType = "dowhile"
	LoopTypeDoUntil LoopType = "dountil"
)

// StepFlowEntry represents a single entry in the workflow execution graph.
// In TypeScript this is a discriminated union; in Go we use a struct
// where the Type field determines which fields are populated.
type StepFlowEntry struct {
	Type StepFlowEntryType `json:"type"`

	// For type == "step"
	Step *Step `json:"step,omitempty"`

	// For type == "sleep"
	ID       string                `json:"id,omitempty"`
	Duration *int64                `json:"duration,omitempty"` // milliseconds
	Fn       ExecuteFunction       `json:"-"`                  // dynamic sleep function

	// For type == "sleepUntil"
	Date *time.Time `json:"date,omitempty"`

	// For type == "parallel" or "conditional"
	Steps []StepFlowStepEntry `json:"steps,omitempty"`

	// For type == "conditional"
	Conditions             []ConditionFunction         `json:"-"`
	SerializedConditions   []SerializedCondition        `json:"serializedConditions,omitempty"`

	// For type == "loop"
	LoopCondition          LoopConditionFunction       `json:"-"`
	SerializedCondition    *SerializedCondition         `json:"serializedCondition,omitempty"`
	LoopKind               LoopType                    `json:"loopType,omitempty"`

	// For type == "foreach"
	ForeachOpts *ForeachOpts `json:"opts,omitempty"`
}

// StepFlowStepEntry is a step entry within a parallel or conditional flow entry.
type StepFlowStepEntry struct {
	Type string `json:"type"` // always "step"
	Step *Step  `json:"step"`
}

// SerializedCondition is the serialized form of a condition function.
type SerializedCondition struct {
	ID string `json:"id"`
	Fn string `json:"fn"`
}

// ForeachOpts holds options for a foreach step.
type ForeachOpts struct {
	Concurrency int `json:"concurrency"`
}

// ---------------------------------------------------------------------------
// Serialized Step (for persistence)
// ---------------------------------------------------------------------------

// SerializedStep is the serialized representation of a Step for persistence.
type SerializedStep struct {
	ID                 string                    `json:"id"`
	Description        string                    `json:"description,omitempty"`
	Metadata           StepMetadata              `json:"metadata,omitempty"`
	Component          string                    `json:"component,omitempty"`
	SerializedStepFlow []SerializedStepFlowEntry `json:"serializedStepFlow,omitempty"`
	MapConfig          string                    `json:"mapConfig,omitempty"`
	CanSuspend         bool                      `json:"canSuspend,omitempty"`
}

// SerializedStepFlowEntry is the serialized form of a StepFlowEntry.
type SerializedStepFlowEntry struct {
	Type StepFlowEntryType `json:"type"`

	// For type == "step"
	Step *SerializedStep `json:"step,omitempty"`

	// For type == "sleep"
	ID       string `json:"id,omitempty"`
	Duration *int64 `json:"duration,omitempty"`
	Fn       string `json:"fn,omitempty"`

	// For type == "sleepUntil"
	Date *time.Time `json:"date,omitempty"`

	// For type == "parallel" or "conditional"
	Steps []SerializedStepFlowStepEntry `json:"steps,omitempty"`

	// For type == "conditional"
	SerializedConditions []SerializedCondition `json:"serializedConditions,omitempty"`

	// For type == "loop"
	SerializedCondition *SerializedCondition `json:"serializedCondition,omitempty"`
	LoopKind            LoopType             `json:"loopType,omitempty"`

	// For type == "foreach"
	ForeachOpts *ForeachOpts `json:"opts,omitempty"`
}

// SerializedStepFlowStepEntry is a step entry within a serialized parallel/conditional.
type SerializedStepFlowStepEntry struct {
	Type string         `json:"type"` // always "step"
	Step *SerializedStep `json:"step"`
}

// ---------------------------------------------------------------------------
// Step With Component (extended step for workflow internals)
// ---------------------------------------------------------------------------

// StepWithComponent extends a Step with an optional component identifier and nested steps.
type StepWithComponent struct {
	Step
	Component string                      `json:"component,omitempty"`
	Steps     map[string]*StepWithComponent `json:"steps,omitempty"`
}

// ---------------------------------------------------------------------------
// Workflow Config
// ---------------------------------------------------------------------------

// WorkflowConfig holds the configuration for creating a new Workflow.
type WorkflowConfig struct {
	Mastra              Mastra
	ID                  string
	Description         string
	InputSchema         SchemaWithValidation
	OutputSchema        SchemaWithValidation
	StateSchema         SchemaWithValidation
	RequestContextSchema SchemaWithValidation
	ExecutionEngine     ExecutionEngine
	Steps               []*Step
	RetryConfig         *RetryConfig
	Options             *WorkflowOptions
	Type                WorkflowType
}

// RetryConfig holds retry settings for a workflow or step.
type RetryConfig struct {
	Attempts int `json:"attempts,omitempty"`
	Delay    int `json:"delay,omitempty"` // milliseconds
}

// ---------------------------------------------------------------------------
// Execution Context
// ---------------------------------------------------------------------------

// ExecutionContext is the context passed through workflow execution.
type ExecutionContext struct {
	WorkflowID        string              `json:"workflowId"`
	RunID             string              `json:"runId"`
	ExecutionPath     []int               `json:"executionPath"`
	StepExecutionPath []string            `json:"stepExecutionPath,omitempty"`
	ActiveStepsPath   map[string][]int    `json:"activeStepsPath"`
	ForeachIndex      *int                `json:"foreachIndex,omitempty"`
	SuspendedPaths    map[string][]int    `json:"suspendedPaths"`
	ResumeLabels      map[string]ResumeLabel `json:"resumeLabels"`
	WaitingPaths      map[string][]int    `json:"waitingPaths,omitempty"`
	RetryConfig       RetryConfig         `json:"retryConfig"`
	Format            string              `json:"format,omitempty"` // "legacy", "vnext", or ""
	State             map[string]any      `json:"state"`
	TracingIDs        *TracingIDs         `json:"tracingIds,omitempty"`
}

// TracingIDs holds trace IDs for creating child spans in durable execution.
type TracingIDs struct {
	TraceID        string `json:"traceId"`
	WorkflowSpanID string `json:"workflowSpanId"`
}

// ---------------------------------------------------------------------------
// Mutable Context
// ---------------------------------------------------------------------------

// MutableContext is a subset of ExecutionContext containing only the fields
// that can be modified by step execution (via setState, suspend, etc.)
type MutableContext struct {
	State          map[string]any             `json:"state"`
	SuspendedPaths map[string][]int           `json:"suspendedPaths"`
	ResumeLabels   map[string]ResumeLabel     `json:"resumeLabels"`
}

// ---------------------------------------------------------------------------
// Step Execution Result
// ---------------------------------------------------------------------------

// StepExecutionResult is the result returned from step execution methods.
type StepExecutionResult struct {
	Result         StepResult             `json:"result"`
	StepResults    map[string]StepResult  `json:"stepResults"`
	MutableContext MutableContext         `json:"mutableContext"`
	RequestContext map[string]any         `json:"requestContext"`
}

// EntryExecutionResult is the result returned from entry execution methods.
type EntryExecutionResult struct {
	Result         StepResult             `json:"result"`
	StepResults    map[string]StepResult  `json:"stepResults"`
	MutableContext MutableContext         `json:"mutableContext"`
	RequestContext map[string]any         `json:"requestContext"`
}

// ---------------------------------------------------------------------------
// Step Execution Core Result
// ---------------------------------------------------------------------------

// StepExecutionCoreResult holds the core result from step execution logic.
type StepExecutionCoreResult struct {
	Status         string `json:"status"` // "success", "failed", "suspended", "bailed"
	Output         any    `json:"output,omitempty"`
	Error          string `json:"error,omitempty"`
	SuspendPayload any    `json:"suspendPayload,omitempty"`
	SuspendOutput  any    `json:"suspendOutput,omitempty"`
	EndedAt        *int64 `json:"endedAt,omitempty"`
	SuspendedAt    *int64 `json:"suspendedAt,omitempty"`
}

// ---------------------------------------------------------------------------
// Sleep Params
// ---------------------------------------------------------------------------

// SleepDurationParams holds parameters for executing a sleep duration.
type SleepDurationParams struct {
	Duration int64  // milliseconds
	SleepID  string
}

// SleepUntilDateParams holds parameters for executing a sleep until date.
type SleepUntilDateParams struct {
	Date         time.Time
	SleepUntilID string
}

// ---------------------------------------------------------------------------
// Condition Eval Params
// ---------------------------------------------------------------------------

// ConditionEvalParams holds parameters for evaluating a condition.
type ConditionEvalParams struct {
	ConditionFn ConditionFunction
	Index       int
	WorkflowID  string
	RunID       string
	Context     *ExecuteFunctionParams
	EvalSpan    obstypes.AnySpan
}

// ---------------------------------------------------------------------------
// Persistence Wrap Params
// ---------------------------------------------------------------------------

// PersistenceWrapParams holds parameters for persistence wrapping.
type PersistenceWrapParams struct {
	WorkflowID    string
	RunID         string
	ExecutionPath []int
	PersistFn     func(ctx context.Context) error
}

// ---------------------------------------------------------------------------
// Durable Operation Wrap Params
// ---------------------------------------------------------------------------

// DurableOperationWrapParams holds parameters for wrapping a durable operation.
type DurableOperationWrapParams struct {
	OperationID string
	OperationFn func(ctx context.Context) (any, error)
}

// ---------------------------------------------------------------------------
// Formatted Workflow Result
// ---------------------------------------------------------------------------

// FormattedWorkflowResult is the base type for formatted workflow results returned by fmtReturnValue.
type FormattedWorkflowResult struct {
	Status            WorkflowStepStatus           `json:"status"`
	Steps             map[string]StepResult        `json:"steps"`
	Input             *StepResult                  `json:"input,omitempty"`
	Result            any                          `json:"result,omitempty"`
	Error             *mastraerror.SerializedError  `json:"error,omitempty"`
	Suspended         [][]string                   `json:"suspended,omitempty"`
	SuspendPayload    any                          `json:"suspendPayload,omitempty"`
	Tripwire          *StepTripwireInfo            `json:"tripwire,omitempty"`
	StepExecutionPath []string                     `json:"stepExecutionPath,omitempty"`
}

// ---------------------------------------------------------------------------
// Stream Event Types
// ---------------------------------------------------------------------------

// StreamEventType identifies the kind of stream event.
type StreamEventType string

const (
	StreamEventTypeStepSuspended StreamEventType = "step-suspended"
	StreamEventTypeStepWaiting   StreamEventType = "step-waiting"
	StreamEventTypeStepResult    StreamEventType = "step-result"
)

// StreamEvent represents an event emitted during workflow streaming.
type StreamEvent struct {
	Type    StreamEventType `json:"type"`
	Payload any             `json:"payload,omitempty"`
	ID      string          `json:"id,omitempty"`
}

// ---------------------------------------------------------------------------
// Step Execution Start Params
// ---------------------------------------------------------------------------

// StepExecutionStartParams holds parameters for the step execution start hook.
type StepExecutionStartParams struct {
	WorkflowID       string
	RunID            string
	Step             *Step
	InputData        any
	PubSub           events.PubSub
	ExecutionContext *ExecutionContext
	StepCallID       string
	StepInfo         map[string]any
}

// ---------------------------------------------------------------------------
// Regular Step Execution Params
// ---------------------------------------------------------------------------

// RegularStepExecutionParams holds parameters for executing a regular (non-workflow) step.
type RegularStepExecutionParams struct {
	Step                *Step
	StepResults         map[string]StepResult
	ExecutionContext    *ExecutionContext
	Resume              *ResumeInfo
	Restart             *RestartExecutionParams
	TimeTravel          *TimeTravelExecutionParams
	PrevOutput          any
	InputData           any
	PubSub              events.PubSub
	AbortCtx            context.Context
	RequestContext      *requestcontext.RequestContext
	WritableStream      OutputWriter
	StartedAt           int64
	ResumeDataToUse     any
	StepSpan            obstypes.AnySpan
	ValidationError     error
	StepCallID          string
	SerializedStepGraph []SerializedStepFlowEntry
	ResourceID          string
	DisableScorers      bool
	Observability       *obstypes.ObservabilityContext
}

// ResumeInfo holds resume data for step execution.
type ResumeInfo struct {
	Steps         []string `json:"steps"`
	ResumePayload any      `json:"resumePayload"`
	Label         string   `json:"label,omitempty"`
	ForEachIndex  *int     `json:"forEachIndex,omitempty"`
}

// ---------------------------------------------------------------------------
// Workflow Result
// ---------------------------------------------------------------------------

// WorkflowResult represents the final result of a workflow execution.
// In TypeScript this is a discriminated union across several status variants.
// In Go we use a single struct with the Status field as the discriminator.
type WorkflowResult struct {
	Status            WorkflowRunStatus     `json:"status"`
	State             map[string]any        `json:"state,omitempty"`
	StepExecutionPath []string              `json:"stepExecutionPath,omitempty"`
	ResumeLabels      map[string]ResumeLabel `json:"resumeLabels,omitempty"`
	Result            any                   `json:"result,omitempty"`
	Input             any                   `json:"input,omitempty"`
	Steps             map[string]StepResult `json:"steps"`
	Error             error                 `json:"error,omitempty"`
	SuspendPayload    any                   `json:"suspendPayload,omitempty"`
	Suspended         [][]string            `json:"suspended,omitempty"`
	Tripwire          *StepTripwireInfo     `json:"tripwire,omitempty"`
}
