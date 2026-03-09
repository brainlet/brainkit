// Ported from: packages/core/src/workflows/workflow.ts
package workflows

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storagedomains "github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	storageworkflows "github.com/brainlet/brainkit/agent-kit/core/storage/domains/workflows"
	streamPkg "github.com/brainlet/brainkit/agent-kit/core/stream"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
)

// ---------------------------------------------------------------------------
// AnyWorkflow
// ---------------------------------------------------------------------------

// AnyWorkflow is the type-erased variant of Workflow.
// TS equivalent: export type AnyWorkflow = Workflow<any, any, any, any, any, any, any, any>;
type AnyWorkflow = *Workflow

// ---------------------------------------------------------------------------
// createWorkflow / cloneWorkflow (package-level functions)
// ---------------------------------------------------------------------------

// CreateWorkflow creates a new default-engine Workflow from the given config.
// TS equivalent: export function createWorkflow(params: WorkflowConfig) { return new Workflow(params); }
func CreateWorkflow(config WorkflowConfig) *Workflow {
	return NewWorkflow(config)
}

// CloneWorkflow creates a copy of an existing workflow with a new ID.
// TS equivalent: export function cloneWorkflow(workflow, opts)
func CloneWorkflow(source *Workflow, newID string) *Workflow {
	w := NewWorkflow(WorkflowConfig{
		ID:           newID,
		InputSchema:  source.InputSchema,
		OutputSchema: source.OutputSchema,
		Steps:        source.StepDefs,
		Mastra:       source.mastra,
		Options:      source.options,
	})
	w.SetStepFlow(source.StepGraph())
	w.Commit()
	return w
}

// ---------------------------------------------------------------------------
// createStep (package-level function)
// ---------------------------------------------------------------------------

// CreateStep creates a Step from explicit StepParams.
// TS equivalent: export function createStep(params: StepParams): Step
func CreateStep(params StepParams) *Step {
	return &Step{
		ID:                   params.ID,
		Description:          params.Description,
		InputSchema:          params.InputSchema,
		OutputSchema:         params.OutputSchema,
		StateSchema:          params.StateSchema,
		ResumeSchema:         params.ResumeSchema,
		SuspendSchema:        params.SuspendSchema,
		RequestContextSchema: params.RequestContextSchema,
		Scorers:              params.Scorers,
		Retries:              params.Retries,
		Metadata:             params.Metadata,
		Execute:              params.Execute,
	}
}

// ---------------------------------------------------------------------------
// Agent / Tool / Processor Step Adapters
// ---------------------------------------------------------------------------

// AgentLike is the interface for agent-like objects that can be wrapped as steps.
// TODO: import from agent package once ported.
type AgentLike interface {
	GetID() string
	GetDescription() string
	GetComponent() string
}

// AgentStepOptions holds options when wrapping an agent as a step.
// TS equivalent: AgentStepOptions<TOUTPUT>
type AgentStepOptions struct {
	StructuredOutput *StructuredOutputConfig
	Retries          *int
	Scorers          any // DynamicArgument[MastraScorers]
	Metadata         StepMetadata
}

// StructuredOutputConfig holds configuration for structured output from agents.
type StructuredOutputConfig struct {
	Schema SchemaWithValidation
}

// CreateStepFromAgent creates a Step from an agent-like object.
// The resulting step accepts {prompt: string} as input and produces {text: string} as output
// by default, or structured output if StructuredOutput is configured.
// TS equivalent: function createStepFromAgent(params, agentOrToolOptions?)
func CreateStepFromAgent(agent AgentLike, opts *AgentStepOptions) *Step {
	var retries *int
	var scorers any
	var metadata StepMetadata
	var outputSchema SchemaWithValidation
	if opts != nil {
		retries = opts.Retries
		scorers = opts.Scorers
		metadata = opts.Metadata
		if opts.StructuredOutput != nil && opts.StructuredOutput.Schema != nil {
			outputSchema = opts.StructuredOutput.Schema
		}
	}

	// In TS the execute function streams the agent and collects the text output.
	// In Go we implement a simplified version that calls the agent synchronously.
	// Full streaming support requires the Agent package to be ported.
	return &Step{
		ID:           agent.GetID(),
		Description:  agent.GetDescription(),
		InputSchema:  nil, // TODO: z.object({ prompt: z.string() }) equivalent
		OutputSchema: outputSchema,
		Retries:      retries,
		Scorers:      scorers,
		Metadata:     metadata,
		Execute: func(params *ExecuteFunctionParams) (any, error) {
			// TODO: Implement full agent streaming execution once Agent package is ported.
			// In TS, this streams the agent, captures text-delta events, and returns
			// either structured output or {text: string}.
			// For now, return an error indicating the agent execution is not yet available.
			return nil, fmt.Errorf("agent step execution not yet implemented: agent '%s' requires the Agent package to be ported", agent.GetID())
		},
		Component: agent.GetComponent(),
	}
}

// ToolLike is the interface for tool-like objects that can be wrapped as steps.
// TODO: import from tools package once ported.
type ToolLike interface {
	GetID() string
	GetDescription() string
	GetInputSchema() SchemaWithValidation
	GetOutputSchema() SchemaWithValidation
	GetResumeSchema() SchemaWithValidation
	GetSuspendSchema() SchemaWithValidation
	Execute(inputData any, ctx ToolExecutionContext) (any, error)
}

// ToolExecutionContext is the context passed to tool execution.
// TODO: import from tools package once ported.
type ToolExecutionContext struct {
	Mastra         Mastra
	RequestContext *requestcontext.RequestContext
	ResumeData     any
	Workflow       *ToolWorkflowContext
	Observability  *obstypes.ObservabilityContext
}

// ToolWorkflowContext holds workflow-specific context for tool execution.
type ToolWorkflowContext struct {
	RunID      string
	WorkflowID string
	Suspend    func(payload any, opts *SuspendOptions) error
	ResumeData any
	State      any
	SetState   func(state any) error
}

// ToolStepOptions holds options when wrapping a tool as a step.
type ToolStepOptions struct {
	Retries  *int
	Scorers  any // DynamicArgument[MastraScorers]
	Metadata StepMetadata
}

// CreateStepFromTool creates a Step from a tool-like object.
// TS equivalent: function createStepFromTool(params, toolOpts?)
func CreateStepFromTool(tool ToolLike, opts *ToolStepOptions) *Step {
	inputSchema := tool.GetInputSchema()
	outputSchema := tool.GetOutputSchema()
	if inputSchema == nil || outputSchema == nil {
		// TS throws: 'Tool must have input and output schemas defined'
		// In Go, we return a step that errors on execution.
		return &Step{
			ID:          tool.GetID(),
			Description: tool.GetDescription(),
			Execute: func(params *ExecuteFunctionParams) (any, error) {
				return nil, errors.New("tool must have input and output schemas defined")
			},
		}
	}

	var retries *int
	var scorers any
	var metadata StepMetadata
	if opts != nil {
		retries = opts.Retries
		scorers = opts.Scorers
		metadata = opts.Metadata
	}

	return &Step{
		ID:            tool.GetID(),
		Description:   tool.GetDescription(),
		InputSchema:   inputSchema,
		OutputSchema:  outputSchema,
		ResumeSchema:  tool.GetResumeSchema(),
		SuspendSchema: tool.GetSuspendSchema(),
		Retries:       retries,
		Scorers:       scorers,
		Metadata:      metadata,
		Execute: func(params *ExecuteFunctionParams) (any, error) {
			toolCtx := ToolExecutionContext{
				Mastra:         params.Mastra,
				RequestContext: params.RequestContext,
				ResumeData:     params.ResumeData,
				Workflow: &ToolWorkflowContext{
					RunID:      params.RunID,
					WorkflowID: params.WorkflowID,
					Suspend:    params.Suspend,
					ResumeData: params.ResumeData,
					State:      params.State,
					SetState:   params.SetState,
				},
				Observability: params.Observability,
			}
			return tool.Execute(params.InputData, toolCtx)
		},
		Component: "TOOL",
	}
}

// ProcessorLike is the interface for processor-like objects that can be wrapped as steps.
// TODO: import from processors package once ported.
type ProcessorLike interface {
	GetID() string
	GetName() string
	ProcessInput(ctx map[string]any) (any, error)
	ProcessInputStep(ctx map[string]any) (any, error)
	ProcessOutputStream(ctx map[string]any) (any, error)
	ProcessOutputResult(ctx map[string]any) (any, error)
	ProcessOutputStep(ctx map[string]any) (any, error)
	HasProcessInput() bool
	HasProcessInputStep() bool
	HasProcessOutputStream() bool
	HasProcessOutputResult() bool
	HasProcessOutputStep() bool
}

// CreateStepFromProcessor creates a Step from a processor-like object.
// The processor is wrapped as a workflow step with phase-based execution.
// TS equivalent: function createStepFromProcessor(processor)
func CreateStepFromProcessor(processor ProcessorLike) *Step {
	stepID := fmt.Sprintf("processor:%s", processor.GetID())
	description := processor.GetName()
	if description == "" {
		description = fmt.Sprintf("Processor %s", processor.GetID())
	}

	return &Step{
		ID:          stepID,
		Description: description,
		// InputSchema and OutputSchema would be ProcessorStepInputSchema / ProcessorStepOutputSchema
		// TODO: Set proper schemas once processor step schemas are ported.
		Execute: func(params *ExecuteFunctionParams) (any, error) {
			inputData, ok := params.InputData.(map[string]any)
			if !ok {
				return params.InputData, nil
			}

			phase, _ := inputData["phase"].(string)

			// Early return if processor doesn't implement this phase
			hasPhase := false
			switch phase {
			case "input":
				hasPhase = processor.HasProcessInput()
			case "inputStep":
				hasPhase = processor.HasProcessInputStep()
			case "outputStream":
				hasPhase = processor.HasProcessOutputStream()
			case "outputResult":
				hasPhase = processor.HasProcessOutputResult()
			case "outputStep":
				hasPhase = processor.HasProcessOutputStep()
			}
			if !hasPhase {
				return inputData, nil
			}

			// Execute the appropriate phase method
			switch phase {
			case "input":
				return processor.ProcessInput(inputData)
			case "inputStep":
				return processor.ProcessInputStep(inputData)
			case "outputStream":
				return processor.ProcessOutputStream(inputData)
			case "outputResult":
				return processor.ProcessOutputResult(inputData)
			case "outputStep":
				return processor.ProcessOutputStep(inputData)
			default:
				return inputData, nil
			}
		},
		Component: "PROCESSOR",
	}
}

// MapVariable is a type-safe mapping helper for referencing step outputs.
// In Go we don't have TS overloads, so this is a pass-through.
// TS equivalent: export function mapVariable(config: any): any
func MapVariable(config map[string]any) map[string]any {
	return config
}

// CloneStep creates a copy of a Step with a new ID.
// TS equivalent: export function cloneStep(step, opts)
func CloneStep(step *Step, newID string) *Step {
	return &Step{
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
// Workflow
// ---------------------------------------------------------------------------

// Workflow is the main workflow class that manages step graphs and execution runs.
// It implements the Step interface (a workflow can be used as a step in another workflow).
// TS equivalent: export class Workflow extends MastraBase implements Step
type Workflow struct {
	ID                   string
	Description          string
	InputSchema          SchemaWithValidation
	OutputSchema         SchemaWithValidation
	StateSchema          SchemaWithValidation
	RequestContextSchema SchemaWithValidation
	StepDefs             []*Step
	Steps                map[string]*StepWithComponent
	EngineType           WorkflowEngineType
	Type                 WorkflowType
	Committed            bool
	RetryConfig          RetryConfig

	// stepFlow is the runtime step flow graph.
	stepFlow []StepFlowEntry

	// serializedStepFlow is the serialized step flow graph for persistence.
	serializedStepFlow []SerializedStepFlowEntry

	executionEngine ExecutionEngine
	executionGraph  ExecutionGraph
	options         *WorkflowOptions

	mastra Mastra
	runs   sync.Map // map[string]*Run
	log    logger.IMastraLogger
}

// NewWorkflow creates a new Workflow from the given config.
// TS equivalent: constructor(config: WorkflowConfig)
func NewWorkflow(config WorkflowConfig) *Workflow {
	w := &Workflow{
		ID:                   config.ID,
		Description:          config.Description,
		InputSchema:          config.InputSchema,
		OutputSchema:         config.OutputSchema,
		StateSchema:          config.StateSchema,
		RequestContextSchema: config.RequestContextSchema,
		StepDefs:             config.Steps,
		Steps:                make(map[string]*StepWithComponent),
		EngineType:           "default",
		Type:                 config.Type,
		mastra:               config.Mastra,
		stepFlow:             []StepFlowEntry{},
		serializedStepFlow:   []SerializedStepFlowEntry{},
	}

	if config.Type == "" {
		w.Type = WorkflowTypeDefault
	}

	// Resolve retry config
	if config.RetryConfig != nil {
		w.RetryConfig = *config.RetryConfig
	}

	// Resolve options with defaults
	opts := config.Options
	if opts == nil {
		opts = &WorkflowOptions{}
	}
	w.options = opts
	if w.options.ValidateInputs == nil {
		t := true
		w.options.ValidateInputs = &t
	}
	if w.options.ShouldPersistSnapshot == nil {
		w.options.ShouldPersistSnapshot = func(_ ShouldPersistSnapshotParams) bool { return true }
	}

	// Set up execution engine
	if config.ExecutionEngine != nil {
		w.executionEngine = config.ExecutionEngine
	} else {
		w.executionEngine = NewDefaultExecutionEngine(w.mastra, &ExecutionEngineOptions{
			TracingPolicy:         opts.TracingPolicy,
			ValidateInputs:        w.options.ValidateInputs != nil && *w.options.ValidateInputs,
			ShouldPersistSnapshot: w.options.ShouldPersistSnapshot,
			OnFinish:              w.options.OnFinish,
			OnError:               w.options.OnError,
		})
	}

	// Build initial execution graph
	w.executionGraph = w.BuildExecutionGraph()

	return w
}

// GetID returns the workflow ID (satisfies Step-like interface).
func (w *Workflow) GetID() string {
	return w.ID
}

// GetMastra returns the mastra instance.
func (w *Workflow) GetMastra() Mastra {
	return w.mastra
}

// GetOptions returns the workflow options.
func (w *Workflow) GetOptions() *WorkflowOptions {
	return w.options
}

// RegisterMastra registers the mastra instance with the workflow and its engine.
// TS equivalent: __registerMastra(mastra: Mastra)
func (w *Workflow) RegisterMastra(mastra Mastra) {
	w.mastra = mastra
	if mastra != nil {
		w.log = mastra.GetLogger()
	}
	if w.executionEngine != nil {
		w.executionEngine.RegisterMastra(mastra)
	}
}

// SetLogger sets the workflow's logger.
// This method exists so that the mastra orchestrator can update the logger
// independently of RegisterMastra (e.g. when the global logger changes).
func (w *Workflow) SetLogger(l logger.IMastraLogger) {
	w.log = l
}

// IsCommitted reports whether the workflow has been committed.
func (w *Workflow) IsCommitted() bool {
	return w.Committed
}

// GetEngineType returns the workflow engine type string.
func (w *Workflow) GetEngineType() string {
	return w.EngineType
}

// Primitives holds primitives passed during workflow registration.
// This mirrors the mastra-side WorkflowPrimitives struct and lets the
// workflows package accept registration data without importing mastra.
type Primitives struct {
	Logger  logger.IMastraLogger
	Storage any // *storage.MastraCompositeStore — kept as any to avoid circular import
}

// RegisterPrimitives applies registration primitives to the workflow.
// Called by the mastra orchestrator after RegisterMastra to propagate
// logger and storage references.
func (w *Workflow) RegisterPrimitives(p Primitives) {
	if p.Logger != nil {
		w.log = p.Logger
	}
}

// SetStepFlow sets the step flow entries.
// TS equivalent: setStepFlow(stepFlow)
func (w *Workflow) SetStepFlow(steps []StepFlowEntry) {
	w.stepFlow = steps
}

// StepGraph returns the current step flow graph.
// TS equivalent: get stepGraph()
func (w *Workflow) StepGraph() []StepFlowEntry {
	return w.stepFlow
}

// SerializedStepGraphEntries returns the serialized step graph.
// TS equivalent: get serializedStepGraph()
func (w *Workflow) SerializedStepGraphEntries() []SerializedStepFlowEntry {
	return w.serializedStepFlow
}

// ---------------------------------------------------------------------------
// Workflow builder methods (Then, Sleep, SleepUntil, Parallel, Branch, etc.)
// ---------------------------------------------------------------------------

// Then adds a step to the workflow.
// TS equivalent: then(step)
func (w *Workflow) Then(step *Step) *Workflow {
	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type: StepFlowEntryTypeStep,
		Step: step,
	})

	canSuspend := step.SuspendSchema != nil || step.ResumeSchema != nil
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeStep,
		Step: &SerializedStep{
			ID:          step.ID,
			Description: step.Description,
			Metadata:    step.Metadata,
			Component:   step.Component,
			CanSuspend:  canSuspend,
		},
	})

	w.Steps[step.ID] = &StepWithComponent{Step: *step, Component: step.Component}
	return w
}

// Sleep adds a sleep step to the workflow with a fixed duration (milliseconds).
// TS equivalent: sleep(duration)
func (w *Workflow) Sleep(durationMs int64) *Workflow {
	id := w.generateStepID("sleep", "sleep")

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:     StepFlowEntryTypeSleep,
		ID:       id,
		Duration: &durationMs,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type:     StepFlowEntryTypeSleep,
		ID:       id,
		Duration: &durationMs,
	})
	w.Steps[id] = &StepWithComponent{
		Step: Step{
			ID: id,
			Execute: func(params *ExecuteFunctionParams) (any, error) {
				return map[string]any{}, nil
			},
		},
	}
	return w
}

// SleepFn adds a dynamic sleep step whose duration is computed at runtime.
// TS equivalent: sleep(fn: ExecuteFunction)
func (w *Workflow) SleepFn(fn ExecuteFunction) *Workflow {
	id := w.generateStepID("sleep", "sleep")

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type: StepFlowEntryTypeSleep,
		ID:   id,
		Fn:   fn,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeSleep,
		ID:   id,
		Fn:   "dynamic",
	})
	w.Steps[id] = &StepWithComponent{
		Step: Step{
			ID: id,
			Execute: func(params *ExecuteFunctionParams) (any, error) {
				return map[string]any{}, nil
			},
		},
	}
	return w
}

// SleepUntil adds a sleepUntil step that waits until a specific date.
// TS equivalent: sleepUntil(date)
func (w *Workflow) SleepUntil(date *int64) *Workflow {
	id := w.generateStepID("sleep", "sleep-until")

	// Convert int64 ms to time.Time for the step flow entry
	// TODO: Use proper time.Time conversion when types align
	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type: StepFlowEntryTypeSleepUntil,
		ID:   id,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeSleepUntil,
		ID:   id,
	})
	w.Steps[id] = &StepWithComponent{
		Step: Step{
			ID: id,
			Execute: func(params *ExecuteFunctionParams) (any, error) {
				return map[string]any{}, nil
			},
		},
	}
	return w
}

// Parallel adds a parallel step block containing multiple steps that execute concurrently.
// TS equivalent: parallel(steps)
func (w *Workflow) Parallel(steps []*Step) *Workflow {
	stepEntries := make([]StepFlowStepEntry, len(steps))
	serializedEntries := make([]SerializedStepFlowStepEntry, len(steps))

	for i, step := range steps {
		stepEntries[i] = StepFlowStepEntry{Type: "step", Step: step}
		canSuspend := step.SuspendSchema != nil || step.ResumeSchema != nil
		serializedEntries[i] = SerializedStepFlowStepEntry{
			Type: "step",
			Step: &SerializedStep{
				ID:          step.ID,
				Description: step.Description,
				Metadata:    step.Metadata,
				Component:   step.Component,
				CanSuspend:  canSuspend,
			},
		}
		w.Steps[step.ID] = &StepWithComponent{Step: *step, Component: step.Component}
	}

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:  StepFlowEntryTypeParallel,
		Steps: stepEntries,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type:  StepFlowEntryTypeParallel,
		Steps: serializedEntries,
	})
	return w
}

// BranchEntry is a pair of (condition, step) for conditional branching.
type BranchEntry struct {
	Condition ConditionFunction
	Step      *Step
}

// Branch adds a conditional branching step.
// TS equivalent: branch(steps: Array<[ConditionFunction, Step]>)
func (w *Workflow) Branch(entries []BranchEntry) *Workflow {
	stepEntries := make([]StepFlowStepEntry, len(entries))
	serializedEntries := make([]SerializedStepFlowStepEntry, len(entries))
	conditions := make([]ConditionFunction, len(entries))
	serializedConditions := make([]SerializedCondition, len(entries))

	for i, entry := range entries {
		stepEntries[i] = StepFlowStepEntry{Type: "step", Step: entry.Step}
		canSuspend := entry.Step.SuspendSchema != nil || entry.Step.ResumeSchema != nil
		serializedEntries[i] = SerializedStepFlowStepEntry{
			Type: "step",
			Step: &SerializedStep{
				ID:          entry.Step.ID,
				Description: entry.Step.Description,
				Metadata:    entry.Step.Metadata,
				Component:   entry.Step.Component,
				CanSuspend:  canSuspend,
			},
		}
		conditions[i] = entry.Condition
		serializedConditions[i] = SerializedCondition{
			ID: fmt.Sprintf("%s-condition", entry.Step.ID),
			Fn: "condition",
		}
		w.Steps[entry.Step.ID] = &StepWithComponent{Step: *entry.Step, Component: entry.Step.Component}
	}

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:               StepFlowEntryTypeConditional,
		Steps:              stepEntries,
		Conditions:         conditions,
		SerializedConditions: serializedConditions,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type:                 StepFlowEntryTypeConditional,
		Steps:                serializedEntries,
		SerializedConditions: serializedConditions,
	})
	return w
}

// DoWhile adds a do-while loop step.
// TS equivalent: dowhile(step, condition)
func (w *Workflow) DoWhile(step *Step, condition LoopConditionFunction) *Workflow {
	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:          StepFlowEntryTypeLoop,
		Step:          step,
		LoopCondition: condition,
		LoopKind:      LoopTypeDoWhile,
		SerializedCondition: &SerializedCondition{
			ID: fmt.Sprintf("%s-condition", step.ID),
			Fn: "condition",
		},
	})

	canSuspend := step.SuspendSchema != nil || step.ResumeSchema != nil
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeLoop,
		Step: &SerializedStep{
			ID:          step.ID,
			Description: step.Description,
			Metadata:    step.Metadata,
			Component:   step.Component,
			CanSuspend:  canSuspend,
		},
		SerializedCondition: &SerializedCondition{
			ID: fmt.Sprintf("%s-condition", step.ID),
			Fn: "condition",
		},
		LoopKind: LoopTypeDoWhile,
	})
	w.Steps[step.ID] = &StepWithComponent{Step: *step, Component: step.Component}
	return w
}

// DoUntil adds a do-until loop step.
// TS equivalent: dountil(step, condition)
func (w *Workflow) DoUntil(step *Step, condition LoopConditionFunction) *Workflow {
	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:          StepFlowEntryTypeLoop,
		Step:          step,
		LoopCondition: condition,
		LoopKind:      LoopTypeDoUntil,
		SerializedCondition: &SerializedCondition{
			ID: fmt.Sprintf("%s-condition", step.ID),
			Fn: "condition",
		},
	})

	canSuspend := step.SuspendSchema != nil || step.ResumeSchema != nil
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeLoop,
		Step: &SerializedStep{
			ID:          step.ID,
			Description: step.Description,
			Metadata:    step.Metadata,
			Component:   step.Component,
			CanSuspend:  canSuspend,
		},
		SerializedCondition: &SerializedCondition{
			ID: fmt.Sprintf("%s-condition", step.ID),
			Fn: "condition",
		},
		LoopKind: LoopTypeDoUntil,
	})
	w.Steps[step.ID] = &StepWithComponent{Step: *step, Component: step.Component}
	return w
}

// ForEach adds a forEach step that iterates over array output from the previous step.
// TS equivalent: foreach(step, opts?)
func (w *Workflow) ForEach(step *Step, opts *ForeachOpts) *Workflow {
	if opts == nil {
		opts = &ForeachOpts{Concurrency: 1}
	}

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type:        StepFlowEntryTypeForeach,
		Step:        step,
		ForeachOpts: opts,
	})

	canSuspend := step.SuspendSchema != nil || step.ResumeSchema != nil
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeForeach,
		Step: &SerializedStep{
			ID:          step.ID,
			Description: step.Description,
			Metadata:    step.Metadata,
			Component:   step.Component,
			CanSuspend:  canSuspend,
		},
		ForeachOpts: opts,
	})
	w.Steps[step.ID] = &StepWithComponent{Step: *step, Component: step.Component}
	return w
}

// Map adds a mapping step that transforms data between steps.
// When fn is provided, it acts as a dynamic mapping function.
// TS equivalent: map(mappingConfig | fn, stepOptions?)
func (w *Workflow) Map(fn ExecuteFunction, id string) *Workflow {
	if id == "" {
		id = w.generateStepID("mapping", "mapping")
	}

	mappingStep := CreateStep(StepParams{
		ID:      id,
		Execute: fn,
	})

	w.stepFlow = append(w.stepFlow, StepFlowEntry{
		Type: StepFlowEntryTypeStep,
		Step: mappingStep,
	})
	w.serializedStepFlow = append(w.serializedStepFlow, SerializedStepFlowEntry{
		Type: StepFlowEntryTypeStep,
		Step: &SerializedStep{
			ID:        id,
			MapConfig: "dynamic",
		},
	})
	return w
}

// BuildExecutionGraph builds and returns the execution graph for this workflow.
// TS equivalent: buildExecutionGraph()
func (w *Workflow) BuildExecutionGraph() ExecutionGraph {
	return ExecutionGraph{
		ID:    w.ID,
		Steps: w.stepFlow,
	}
}

// Commit finalizes the workflow definition and prepares it for execution.
// Must be called after all steps have been added.
// TS equivalent: commit()
func (w *Workflow) Commit() *Workflow {
	w.executionGraph = w.BuildExecutionGraph()
	w.Committed = true
	return w
}

// ---------------------------------------------------------------------------
// CreateRun
// ---------------------------------------------------------------------------

// CreateRunOptions holds options for creating a workflow run.
type CreateRunOptions struct {
	RunID          string
	ResourceID     string
	DisableScorers bool
}

// CreateRun creates a new workflow run instance.
// TS equivalent: async createRun(options?)
func (w *Workflow) CreateRun(opts *CreateRunOptions) (*Run, error) {
	if len(w.stepFlow) == 0 {
		return nil, errors.New("execution flow of workflow is not defined; add steps via .Then(), .Branch(), etc.")
	}
	if len(w.executionGraph.Steps) == 0 {
		return nil, errors.New("uncommitted step flow changes detected; call .Commit() to register the steps")
	}

	runID := ""
	resourceID := ""
	disableScorers := false
	if opts != nil {
		runID = opts.RunID
		resourceID = opts.ResourceID
		disableScorers = opts.DisableScorers
	}

	if runID == "" {
		if w.mastra != nil {
			source := aktypes.IdGeneratorSourceWorkflow
			runID = w.mastra.GenerateID(&GenerateIDOpts{
				IdType:     aktypes.IdTypeRun,
				Source:     &source,
				EntityId:   &w.ID,
				ResourceId: &resourceID,
			})
		}
		if runID == "" {
			runID = generateUUID()
		}
	}

	// Return existing run if present
	if existing, ok := w.runs.Load(runID); ok {
		return existing.(*Run), nil
	}

	validateInputs := false
	if w.options != nil && w.options.ValidateInputs != nil {
		validateInputs = *w.options.ValidateInputs
	}

	var tracingPolicy *obstypes.TracingPolicy
	if w.options != nil {
		tracingPolicy = w.options.TracingPolicy
	}

	run := NewRun(RunConfig{
		WorkflowID:          w.ID,
		RunID:               runID,
		ResourceID:          resourceID,
		StateSchema:         w.StateSchema,
		InputSchema:         w.InputSchema,
		RequestContextSchema: w.RequestContextSchema,
		ExecutionEngine:     w.executionEngine,
		ExecutionGraph:      w.executionGraph,
		Mastra:              w.mastra,
		RetryConfig:         &w.RetryConfig,
		SerializedStepGraph: w.serializedStepFlow,
		DisableScorers:      disableScorers,
		TracingPolicy:       tracingPolicy,
		WorkflowSteps:       w.Steps,
		ValidateInputs:      validateInputs,
		WorkflowEngineType:  w.EngineType,
		Cleanup: func() {
			w.runs.Delete(runID)
		},
	})

	w.runs.Store(runID, run)

	// Persist initial snapshot to storage if the shouldPersistSnapshot callback allows it
	// and the run doesn't already exist in persistent storage.
	// TS: const shouldPersistSnapshot = this.#options.shouldPersistSnapshot({...});
	// TS: if (!existsInStorage && shouldPersistSnapshot) { ... }
	shouldPersist := true
	if w.options != nil && w.options.ShouldPersistSnapshot != nil {
		shouldPersist = w.options.ShouldPersistSnapshot(ShouldPersistSnapshotParams{
			WorkflowStatus: run.WorkflowRunStatus,
			StepResults:    map[string]StepResult{},
		})
	}

	if shouldPersist && w.mastra != nil {
		// Check if the run already exists in persistent storage.
		// TS: const existingRun = await this.getWorkflowRunById(runIdToUse, { withNestedWorkflows: false });
		// TS: const existsInStorage = existingRun && !existingRun.isFromInMemory;
		withNested := false
		existingRun, _ := w.GetWorkflowRunByID(runID, &GetWorkflowRunByIDOptions{WithNestedWorkflows: &withNested})
		existsInStorage := existingRun != nil && !existingRun.IsFromInMemory

		// If a run exists in persistent storage, sync its status to the in-memory run.
		// TS: if (existsInStorage && existingRun.status) { run.workflowRunStatus = existingRun.status; }
		if existsInStorage && existingRun.Status != "" {
			run.WorkflowRunStatus = existingRun.Status
		}

		if !existsInStorage {
			if store := w.mastra.GetWorkflowsStore(); store != nil {
				// TS: await workflowsStore.persistWorkflowSnapshot({
				//   workflowName: this.id, runId, resourceId, snapshot: { ... }
				// })
				_ = store.PersistWorkflowSnapshot(context.Background(), storageworkflows.PersistWorkflowSnapshotArgs{
					WorkflowName: w.ID,
					RunID:        runID,
					ResourceID:   resourceID,
					Snapshot: WorkflowRunStateToMap(WorkflowRunState{
						RunID:               runID,
						Status:              WorkflowRunStatusPending,
						Value:               map[string]string{},
						Context:             map[string]any{},
						ActivePaths:         []int{},
						ActiveStepsPath:     map[string][]int{},
						SerializedStepGraph: w.serializedStepFlow,
						SuspendedPaths:      map[string][]int{},
						ResumeLabels:        map[string]ResumeLabel{},
						WaitingPaths:        map[string][]int{},
						Timestamp:           time.Now().UnixMilli(),
					}),
				})
			}
		}
	}

	return run, nil
}

// ---------------------------------------------------------------------------
// Execute (internal, for nested workflows)
// ---------------------------------------------------------------------------

// Execute is the internal execution method for nested workflow steps.
// Should only be called internally for nested workflow execution or from mastra server handlers.
// TS equivalent: async execute({runId, inputData, ...})
func (w *Workflow) Execute(params WorkflowExecuteParams) (any, error) {
	if params.Mastra != nil {
		w.RegisterMastra(params.Mastra)
	}

	isResume := params.Resume != nil &&
		(len(params.Resume.Steps) > 0 || params.Resume.Label != "" ||
			(len(params.Resume.Steps) == 0 && (params.RetryCount == nil || *params.RetryCount == 0)))
	isTimeTravel := params.TimeTravel != nil && len(params.TimeTravel.Steps) > 0

	var run *Run
	var err error

	if isResume && params.Resume != nil && params.Resume.RunID != "" {
		run, err = w.CreateRun(&CreateRunOptions{RunID: params.Resume.RunID})
	} else {
		run, err = w.CreateRun(&CreateRunOptions{RunID: params.RunID})
	}
	if err != nil {
		return nil, err
	}

	var result *WorkflowResult

	if isTimeTravel {
		result, err = run.TimeTravel(TimeTravelParams{
			InputData:    params.TimeTravel.InputData,
			ResumeData:   params.TimeTravel.ResumeData,
			InitialState: params.State,
			Step:         params.TimeTravel.Steps,
			OutputOptions: &OutputOptions{
				IncludeState:        true,
				IncludeResumeLabels: true,
			},
			PerStep:        params.PerStep,
			RequestContext: params.RequestContext,
		})
	} else if params.Restart {
		result, err = run.Restart(RestartParams{
			RequestContext: params.RequestContext,
		})
	} else if isResume {
		var steps []string
		if params.Resume != nil && len(params.Resume.Steps) > 0 {
			steps = params.Resume.Steps
		}
		result, err = run.Resume(ResumeParams{
			ResumeData:     params.ResumeData,
			Step:           steps,
			RequestContext: params.RequestContext,
			OutputOptions: &OutputOptions{
				IncludeState:        true,
				IncludeResumeLabels: true,
			},
			Label:   params.Resume.Label,
			PerStep: params.PerStep,
		})
	} else {
		result, err = run.Start(StartParams{
			InputData:      params.InputData,
			InitialState:   params.State,
			RequestContext: params.RequestContext,
			OutputOptions: &OutputOptions{
				IncludeState:        true,
				IncludeResumeLabels: true,
			},
			PerStep: params.PerStep,
		})
	}

	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}

	if result.Status == WorkflowRunStatusFailed {
		return nil, result.Error
	}

	if result.Status == WorkflowRunStatusSuccess {
		return result.Result, nil
	}

	return result, nil
}

// WorkflowExecuteParams holds parameters for the internal Execute method (nested workflows).
type WorkflowExecuteParams struct {
	RunID          string
	InputData      any
	ResumeData     any
	State          any
	Restart        bool
	TimeTravel     *TimeTravelExecutionParams
	Resume         *WorkflowExecuteResumeParams
	Mastra         Mastra
	PubSub         events.PubSub
	RequestContext *requestcontext.RequestContext
	RetryCount     *int
	OutputWriter   OutputWriter
	PerStep        bool
}

// WorkflowExecuteResumeParams holds resume parameters for nested workflow execution.
type WorkflowExecuteResumeParams struct {
	Steps         []string
	ResumePayload any
	RunID         string
	Label         string
	ForEachIndex  *int
}

// ---------------------------------------------------------------------------
// Workflow query helpers
// ---------------------------------------------------------------------------

// ListScorers lists all scorers across workflow steps.
// TS equivalent: async listScorers({requestContext})
func (w *Workflow) ListScorers() map[string]any {
	scorers := make(map[string]any)
	for _, step := range w.Steps {
		if step.Scorers != nil {
			if scorerMap, ok := step.Scorers.(map[string]any); ok {
				for id, scorer := range scorerMap {
					scorers[id] = scorer
				}
			}
		}
	}
	return scorers
}

// generateStepID generates a unique step ID using the mastra instance or a fallback UUID.
func (w *Workflow) generateStepID(prefix, stepType string) string {
	id := ""
	if w.mastra != nil {
		source := aktypes.IdGeneratorSourceWorkflow
		id = w.mastra.GenerateID(&GenerateIDOpts{
			IdType:   aktypes.IdTypeStep,
			Source:   &source,
			EntityId: &w.ID,
			StepType: &stepType,
		})
	}
	if id == "" {
		id = generateUUID()
	}
	return fmt.Sprintf("%s_%s", prefix, id)
}

// generateUUID generates a random UUID string.
// TS equivalent: randomUUID() from 'node:crypto'
func generateUUID() string {
	return uuid.New().String()
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

// RunConfig holds configuration for creating a Run.
type RunConfig struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	StateSchema         SchemaWithValidation
	InputSchema         SchemaWithValidation
	RequestContextSchema SchemaWithValidation
	ExecutionEngine     ExecutionEngine
	ExecutionGraph      ExecutionGraph
	Mastra              Mastra
	RetryConfig         *RetryConfig
	Cleanup             func()
	SerializedStepGraph []SerializedStepFlowEntry
	DisableScorers      bool
	TracingPolicy       *obstypes.TracingPolicy
	WorkflowSteps       map[string]*StepWithComponent
	ValidateInputs      bool
	WorkflowEngineType  WorkflowEngineType
}

// Run represents a single execution of a Workflow.
// TS equivalent: export class Run
type Run struct {
	WorkflowID          string
	RunID               string
	ResourceID          string
	SerializedStepGraph []SerializedStepFlowEntry
	WorkflowRunStatus   WorkflowRunStatus
	WorkflowEngineType  WorkflowEngineType
	WorkflowSteps       map[string]*StepWithComponent
	DisableScorers      bool
	TracingPolicy       *obstypes.TracingPolicy
	ValidateInputs      bool

	executionEngine     ExecutionEngine
	executionGraph      ExecutionGraph
	mastra              Mastra
	pubsub              events.PubSub
	retryConfig         *RetryConfig
	cleanup             func()
	stateSchema         SchemaWithValidation
	inputSchema         SchemaWithValidation
	requestContextSchema SchemaWithValidation

	abortCtx    context.Context
	abortCancel context.CancelFunc

	// Streaming fields
	closeStreamAction func()
	streamOutput      *streamPkg.WorkflowRunOutput
	executionResults  *WorkflowResult
	observerHandlers  []func()
}

// NewRun creates a new Run.
// TS equivalent: constructor(params)
func NewRun(config RunConfig) *Run {
	ctx, cancel := context.WithCancel(context.Background())

	var pubsub events.PubSub
	// TODO: Use EventEmitterPubSub when ported
	// pubsub = events.NewEventEmitterPubSub()

	return &Run{
		WorkflowID:          config.WorkflowID,
		RunID:               config.RunID,
		ResourceID:          config.ResourceID,
		SerializedStepGraph: config.SerializedStepGraph,
		WorkflowRunStatus:   WorkflowRunStatusPending,
		WorkflowEngineType:  config.WorkflowEngineType,
		WorkflowSteps:       config.WorkflowSteps,
		DisableScorers:      config.DisableScorers,
		TracingPolicy:       config.TracingPolicy,
		ValidateInputs:      config.ValidateInputs,
		executionEngine:     config.ExecutionEngine,
		executionGraph:      config.ExecutionGraph,
		mastra:              config.Mastra,
		pubsub:              pubsub,
		retryConfig:         config.RetryConfig,
		cleanup:             config.Cleanup,
		stateSchema:         config.StateSchema,
		inputSchema:         config.InputSchema,
		requestContextSchema: config.RequestContextSchema,
		abortCtx:            ctx,
		abortCancel:         cancel,
	}
}

// ---------------------------------------------------------------------------
// Run lifecycle methods
// ---------------------------------------------------------------------------

// StartParams holds parameters for Run.Start().
type StartParams struct {
	InputData      any
	InitialState   any
	RequestContext *requestcontext.RequestContext
	OutputWriter   OutputWriter
	OutputOptions  *OutputOptions
	PerStep        bool
}

// Start begins execution of the workflow run.
// TS equivalent: async start(args)
func (r *Run) Start(params StartParams) (*WorkflowResult, error) {
	return r.internalStart(params)
}

// ResumeParams holds parameters for Run.Resume().
type ResumeParams struct {
	ResumeData     any
	Step           []string // step IDs
	Label          string
	RequestContext *requestcontext.RequestContext
	OutputWriter   OutputWriter
	OutputOptions  *OutputOptions
	ForEachIndex   *int
	PerStep        bool
}

// Resume resumes a suspended workflow run.
// TS equivalent: async resume(params)
func (r *Run) Resume(params ResumeParams) (*WorkflowResult, error) {
	return r.internalResume(params)
}

// RestartParams holds parameters for Run.Restart().
type RestartParams struct {
	RequestContext *requestcontext.RequestContext
	OutputWriter   OutputWriter
}

// Restart restarts a previously active workflow run.
// TS equivalent: async restart(args)
func (r *Run) Restart(params RestartParams) (*WorkflowResult, error) {
	return r.internalRestart(params)
}

// TimeTravelParams holds parameters for Run.TimeTravel().
type TimeTravelParams struct {
	InputData      any
	ResumeData     any
	InitialState   any
	Step           []string // step IDs
	Context        TimeTravelContext
	RequestContext *requestcontext.RequestContext
	OutputWriter   OutputWriter
	OutputOptions  *OutputOptions
	PerStep        bool
}

// TimeTravel executes the workflow from a specific step with provided context.
// TS equivalent: async timeTravel(args)
func (r *Run) TimeTravel(params TimeTravelParams) (*WorkflowResult, error) {
	return r.internalTimeTravel(params)
}

// Cancel cancels the workflow execution.
// TS equivalent: async cancel()
func (r *Run) Cancel() {
	if r.abortCancel != nil {
		r.abortCancel()
	}
	r.WorkflowRunStatus = WorkflowRunStatusCanceled

	// Update workflow status in storage to 'canceled'.
	// This is necessary for suspended/waiting workflows where the abort signal won't be checked.
	// TS: try { const workflowsStore = await this.mastra?.getStorage()?.getStore('workflows');
	//       await workflowsStore?.updateWorkflowState({ workflowName, runId, opts: { status: 'canceled' } });
	//     } catch { /* Storage errors should not prevent cancellation from succeeding */ }
	if r.mastra != nil {
		if store := r.mastra.GetWorkflowsStore(); store != nil {
			// Ignore errors: storage failures should not prevent cancellation from succeeding.
			// The abort signal and in-memory status are already updated.
			_, _ = store.UpdateWorkflowState(context.Background(), storageworkflows.UpdateWorkflowStateArgs{
				WorkflowName: r.WorkflowID,
				RunID:        r.RunID,
				Opts:         storagedomains.UpdateWorkflowStateOptions{Status: storagedomains.WorkflowRunStatus("canceled")},
			})
		}
	}
}

// Watch subscribes to workflow events for this run.
// Returns an unwatch function.
// TS equivalent: watch(cb)
func (r *Run) Watch(cb func(event map[string]any)) func() {
	if r.pubsub == nil {
		return func() {}
	}

	topic := fmt.Sprintf("workflow.events.v2.%s", r.RunID)
	handler := func(event events.Event, _ events.AckFunc) {
		if event.RunID == r.RunID {
			if data, ok := event.Data.(map[string]any); ok {
				cb(data)
			}
		}
	}

	_ = r.pubsub.Subscribe(topic, handler)

	return func() {
		_ = r.pubsub.Unsubscribe(topic, handler)
	}
}

// StartAsync starts the workflow execution without waiting for completion (fire-and-forget).
// TS equivalent: async startAsync(args)
func (r *Run) StartAsync(params StartParams) string {
	go func() {
		_, err := r.Start(params)
		if err != nil && r.mastra != nil {
			log := r.mastra.GetLogger()
			if log != nil {
				log.Error(fmt.Sprintf("[Workflow %s] Background execution failed: %v", r.WorkflowID, err))
			}
		}
	}()
	return r.RunID
}

// ---------------------------------------------------------------------------
// Workflow Storage Query Methods
// ---------------------------------------------------------------------------

// ListWorkflowRuns lists workflow runs from storage.
// TS equivalent: async listWorkflowRuns(args?)
func (w *Workflow) ListWorkflowRuns(params *ListWorkflowRunsParams) (*WorkflowRunList, error) {
	if w.mastra == nil {
		return &WorkflowRunList{Runs: []WorkflowRunRecord{}, Total: 0}, nil
	}
	store := w.mastra.GetWorkflowsStore()
	if store == nil {
		if w.log != nil {
			w.log.Debug("Cannot get workflow runs. Mastra storage is not initialized")
		}
		return &WorkflowRunList{Runs: []WorkflowRunRecord{}, Total: 0}, nil
	}

	p := &ListWorkflowRunsParams{WorkflowName: w.ID}
	if params != nil {
		p.Status = params.Status
	}

	return store.ListWorkflowRuns(context.Background(), p)
}

// ListActiveWorkflowRuns lists all active (running + waiting) workflow runs.
// TS equivalent: public async listActiveWorkflowRuns()
func (w *Workflow) ListActiveWorkflowRuns() (*WorkflowRunList, error) {
	runningRuns, err := w.ListWorkflowRuns(&ListWorkflowRunsParams{Status: "running"})
	if err != nil {
		return nil, err
	}
	waitingRuns, err := w.ListWorkflowRuns(&ListWorkflowRunsParams{Status: "waiting"})
	if err != nil {
		return nil, err
	}

	allRuns := append(runningRuns.Runs, waitingRuns.Runs...)
	return &WorkflowRunList{
		Runs:  allRuns,
		Total: runningRuns.Total + waitingRuns.Total,
	}, nil
}

// RestartAllActiveWorkflowRuns restarts all active workflow runs.
// TS equivalent: public async restartAllActiveWorkflowRuns()
func (w *Workflow) RestartAllActiveWorkflowRuns() error {
	if w.EngineType != "default" {
		if w.log != nil {
			w.log.Debug(fmt.Sprintf("Cannot restart active workflow runs for %s engine", w.EngineType))
		}
		return nil
	}

	activeRuns, err := w.ListActiveWorkflowRuns()
	if err != nil {
		return err
	}

	if len(activeRuns.Runs) > 0 && w.log != nil {
		w.log.Debug(fmt.Sprintf("Restarting %d active workflow run(s)", len(activeRuns.Runs)))
	}

	for _, runSnapshot := range activeRuns.Runs {
		run, err := w.CreateRun(&CreateRunOptions{RunID: runSnapshot.RunID})
		if err != nil {
			if w.log != nil {
				w.log.Error(fmt.Sprintf("Failed to restart %s workflow run %s: %v", w.ID, runSnapshot.RunID, err))
			}
			continue
		}
		_, err = run.Restart(RestartParams{})
		if err != nil {
			if w.log != nil {
				w.log.Error(fmt.Sprintf("Failed to restart %s workflow run %s: %v", w.ID, runSnapshot.RunID, err))
			}
			continue
		}
		if w.log != nil {
			w.log.Debug(fmt.Sprintf("Restarted %s workflow run %s", w.ID, runSnapshot.RunID))
		}
	}
	return nil
}

// DeleteWorkflowRunByID deletes a workflow run by its ID from storage.
// TS equivalent: async deleteWorkflowRunById(runId)
func (w *Workflow) DeleteWorkflowRunByID(runID string) error {
	if w.mastra == nil {
		return nil
	}
	store := w.mastra.GetWorkflowsStore()
	if store == nil {
		if w.log != nil {
			w.log.Debug("Cannot delete workflow run by ID. Mastra storage is not initialized")
		}
		return nil
	}

	err := store.DeleteWorkflowRunByID(context.Background(), DeleteWorkflowRunByIDParams{
		RunID:        runID,
		WorkflowName: w.ID,
	})
	if err != nil {
		return err
	}

	// Delete from in-memory runs
	w.runs.Delete(runID)
	return nil
}

// GetWorkflowRunSteps retrieves processed step results from a workflow run,
// including nested workflow step results.
// TS equivalent: protected async getWorkflowRunSteps({runId, workflowId})
func (w *Workflow) GetWorkflowRunSteps(runID, workflowID string) (map[string]StepResult, error) {
	if w.mastra == nil {
		return map[string]StepResult{}, nil
	}
	store := w.mastra.GetWorkflowsStore()
	if store == nil {
		if w.log != nil {
			w.log.Debug("Cannot get workflow run steps. Mastra storage is not initialized")
		}
		return map[string]StepResult{}, nil
	}

	run, err := store.GetWorkflowRunByID(context.Background(), GetWorkflowRunByIDParams{
		RunID:        runID,
		WorkflowName: workflowID,
	})
	if err != nil || run == nil || run.Snapshot == nil {
		return map[string]StepResult{}, nil
	}

	snapshot := SnapshotToWorkflowRunState(run.Snapshot)
	if snapshot == nil {
		return map[string]StepResult{}, nil
	}
	stepContext := snapshot.Context
	if stepContext == nil {
		return map[string]StepResult{}, nil
	}

	finalSteps := make(map[string]StepResult)

	for stepID, stepVal := range stepContext {
		if stepID == "input" {
			continue
		}

		// Try to convert the step value to a StepResult
		if sr, ok := stepVal.(StepResult); ok {
			finalSteps[stepID] = sr
		} else if m, ok := stepVal.(map[string]any); ok {
			sr := mapToStepResult(m)
			finalSteps[stepID] = sr

			// Check for nested workflow steps
			component := ""
			for _, ssfe := range snapshot.SerializedStepGraph {
				if ssfe.Step != nil && ssfe.Step.ID == stepID {
					component = ssfe.Step.Component
					break
				}
			}
			if component == "WORKFLOW" {
				nestedRunID := runID
				if md, ok := m["metadata"].(map[string]any); ok {
					if nri, ok := md["nestedRunId"].(string); ok {
						nestedRunID = nri
					}
				}
				nestedSteps, err := w.GetWorkflowRunSteps(nestedRunID, stepID)
				if err == nil && len(nestedSteps) > 0 {
					for k, v := range nestedSteps {
						finalSteps[fmt.Sprintf("%s.%s", stepID, k)] = v
					}
				}
			}
		}
	}

	return finalSteps, nil
}

// GetWorkflowRunByID retrieves a workflow run by ID with processed execution state.
// TS equivalent: async getWorkflowRunById(runId, options?)
func (w *Workflow) GetWorkflowRunByID(runID string, opts *GetWorkflowRunByIDOptions) (*WorkflowState, error) {
	withNestedWorkflows := true
	if opts != nil && opts.WithNestedWorkflows != nil {
		withNestedWorkflows = *opts.WithNestedWorkflows
	}

	if w.mastra == nil {
		return w.getInMemoryRunAsWorkflowState(runID), nil
	}

	store := w.mastra.GetWorkflowsStore()
	if store == nil {
		if w.log != nil {
			w.log.Debug("Cannot get workflow run. Mastra storage is not initialized")
		}
		return w.getInMemoryRunAsWorkflowState(runID), nil
	}

	run, err := store.GetWorkflowRunByID(context.Background(), GetWorkflowRunByIDParams{
		RunID:        runID,
		WorkflowName: w.ID,
	})
	if err != nil || run == nil {
		return w.getInMemoryRunAsWorkflowState(runID), nil
	}

	if run.Snapshot == nil {
		return w.getInMemoryRunAsWorkflowState(runID), nil
	}

	snapshot := SnapshotToWorkflowRunState(run.Snapshot)
	if snapshot == nil {
		return w.getInMemoryRunAsWorkflowState(runID), nil
	}

	// Get steps if needed
	steps := make(map[string]WorkflowStateStep)
	if withNestedWorkflows {
		rawSteps, err := w.GetWorkflowRunSteps(runID, w.ID)
		if err == nil {
			for k, v := range rawSteps {
				steps[k] = WorkflowStateStep{
					Status:    WorkflowRunStatus(v.Status),
					StartedAt: v.StartedAt,
					EndedAt:   v.EndedAt,
				}
			}
		}
	} else if snapshot.Context != nil {
		for k, v := range snapshot.Context {
			if k == "input" {
				continue
			}
			if m, ok := v.(map[string]any); ok {
				sr := mapToStepResult(m)
				steps[k] = WorkflowStateStep{
					Status:    WorkflowRunStatus(sr.Status),
					StartedAt: sr.StartedAt,
					EndedAt:   sr.EndedAt,
				}
			}
		}
	}

	var initialState map[string]any
	if len(snapshot.Value) > 0 {
		initialState = make(map[string]any)
		for k, v := range snapshot.Value {
			initialState[k] = v
		}
	}

	var payload map[string]any
	if snapshot.Context != nil {
		if input, ok := snapshot.Context["input"]; ok {
			if m, ok := input.(map[string]any); ok {
				payload = m
			}
		}
	}

	return &WorkflowState{
		RunID:               run.RunID,
		WorkflowName:        run.WorkflowName,
		ResourceID:          run.ResourceID,
		CreatedAt:           run.CreatedAt,
		UpdatedAt:           run.UpdatedAt,
		Status:              snapshot.Status,
		InitialState:        initialState,
		Result:              snapshot.Result,
		Error:               snapshot.Error,
		Payload:             payload,
		Steps:               steps,
		ActiveStepsPath:     snapshot.ActiveStepsPath,
		SerializedStepGraph: snapshot.SerializedStepGraph,
	}, nil
}

// GetWorkflowRunByIDOptions holds options for GetWorkflowRunByID.
type GetWorkflowRunByIDOptions struct {
	WithNestedWorkflows *bool
	Fields              []WorkflowStateField
}

// getInMemoryRunAsWorkflowState converts an in-memory Run to a WorkflowState.
// Used as a fallback when storage is not available.
// TS equivalent: #getInMemoryRunAsWorkflowState(runId)
func (w *Workflow) getInMemoryRunAsWorkflowState(runID string) *WorkflowState {
	val, ok := w.runs.Load(runID)
	if !ok {
		return nil
	}
	inMemoryRun := val.(*Run)

	now := time.Now()
	return &WorkflowState{
		RunID:          runID,
		WorkflowName:   w.ID,
		ResourceID:     inMemoryRun.ResourceID,
		CreatedAt:      now,
		UpdatedAt:      now,
		IsFromInMemory: true,
		Status:         inMemoryRun.WorkflowRunStatus,
		Steps:          map[string]WorkflowStateStep{},
	}
}

// mapToStepResult converts a map[string]any to a StepResult.
func mapToStepResult(m map[string]any) StepResult {
	sr := StepResult{}
	if s, ok := m["status"].(string); ok {
		sr.Status = WorkflowStepStatus(s)
	}
	sr.Output = m["output"]
	sr.Payload = m["payload"]
	sr.ResumePayload = m["resumePayload"]
	sr.SuspendPayload = m["suspendPayload"]
	sr.SuspendOutput = m["suspendOutput"]
	if sa, ok := m["startedAt"].(float64); ok {
		sr.StartedAt = int64(sa)
	} else if sa, ok := m["startedAt"].(int64); ok {
		sr.StartedAt = sa
	}
	if ea, ok := m["endedAt"].(float64); ok {
		sr.EndedAt = int64(ea)
	} else if ea, ok := m["endedAt"].(int64); ok {
		sr.EndedAt = ea
	}
	return sr
}

// ---------------------------------------------------------------------------
// Run Validation Methods
// ---------------------------------------------------------------------------

// validateInput validates input data against the workflow's input schema.
// TS equivalent: protected async _validateInput(inputData?)
func (r *Run) validateInput(inputData any) (any, error) {
	if !r.ValidateInputs || r.inputSchema == nil {
		return inputData, nil
	}

	result, err := r.inputSchema.SafeParse(inputData)
	if err != nil {
		return nil, fmt.Errorf("invalid input data: %w", err)
	}
	if result != nil && result.Success && result.Data != nil {
		return result.Data, nil
	}
	if result != nil && !result.Success && result.Error != nil {
		return nil, fmt.Errorf("invalid input data: %w", result.Error)
	}
	return inputData, nil
}

// validateInitialState validates the initial state against the workflow's state schema.
// TS equivalent: protected async _validateInitialState(initialState?)
func (r *Run) validateInitialState(initialState any) (any, error) {
	if !r.ValidateInputs || r.stateSchema == nil {
		return initialState, nil
	}

	result, err := r.stateSchema.SafeParse(initialState)
	if err != nil {
		return nil, fmt.Errorf("invalid initial state: %w", err)
	}
	if result != nil && result.Success && result.Data != nil {
		return result.Data, nil
	}
	if result != nil && !result.Success && result.Error != nil {
		return nil, fmt.Errorf("invalid initial state: %w", result.Error)
	}
	return initialState, nil
}

// validateRequestContext validates the request context against the workflow's request context schema.
// TS equivalent: protected async _validateRequestContext(requestContext?)
func (r *Run) validateRequestContext(requestContext *requestcontext.RequestContext) error {
	if !r.ValidateInputs || r.requestContextSchema == nil || requestContext == nil {
		return nil
	}

	contextValues := requestContext.All()
	err := r.requestContextSchema.Validate(contextValues)
	if err != nil {
		return fmt.Errorf("request context validation failed for workflow '%s': %w", r.WorkflowID, err)
	}
	return nil
}

// validateResumeData validates resume data against the suspended step's resume schema.
// TS equivalent: protected async _validateResumeData(resumeData, suspendedStep?)
func (r *Run) validateResumeData(resumeData any, suspendedStep *StepWithComponent) (any, error) {
	if suspendedStep == nil || suspendedStep.ResumeSchema == nil || !r.ValidateInputs {
		return resumeData, nil
	}

	result, err := suspendedStep.ResumeSchema.SafeParse(resumeData)
	if err != nil {
		return nil, fmt.Errorf("invalid resume data: %w", err)
	}
	if result != nil && result.Success && result.Data != nil {
		return result.Data, nil
	}
	if result != nil && !result.Success && result.Error != nil {
		return nil, fmt.Errorf("invalid resume data: %w", result.Error)
	}
	return resumeData, nil
}

// validateTimeTravelInputData validates time travel input data against the step's input schema.
// TS equivalent: protected async _validateTimetravelInputData(inputData, step)
func (r *Run) validateTimeTravelInputData(inputData any, step *StepWithComponent) (any, error) {
	if step == nil || step.InputSchema == nil || !r.ValidateInputs {
		return inputData, nil
	}

	result, err := step.InputSchema.SafeParse(inputData)
	if err != nil {
		return nil, fmt.Errorf("invalid inputData: %w", err)
	}
	if result != nil && result.Success && result.Data != nil {
		return result.Data, nil
	}
	if result != nil && !result.Success && result.Error != nil {
		return nil, fmt.Errorf("invalid inputData: %w", result.Error)
	}
	return inputData, nil
}

// ---------------------------------------------------------------------------
// Run internal _start / _resume / _restart methods
// (faithful to TS: snapshot loading from storage, validation, etc.)
// ---------------------------------------------------------------------------

// internalStart is the internal start implementation with validation.
// TS equivalent: protected async _start({inputData, initialState, requestContext, ...})
func (r *Run) internalStart(params StartParams) (*WorkflowResult, error) {
	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Validate inputs
	inputData, err := r.validateInput(params.InputData)
	if err != nil {
		return nil, err
	}
	initialState, err := r.validateInitialState(params.InitialState)
	if err != nil {
		return nil, err
	}
	if err := r.validateRequestContext(rc); err != nil {
		return nil, err
	}

	result, err := r.executionEngine.Execute(ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		ResourceID:          r.ResourceID,
		DisableScorers:      r.DisableScorers,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		Input:               inputData,
		InitialState:        initialState,
		PubSub:              r.pubsub,
		RetryConfig:         r.retryConfig,
		RequestContext:      rc,
		AbortCtx:            r.abortCtx,
		AbortCancel:         r.abortCancel,
		OutputWriter:        params.OutputWriter,
		OutputOptions:       params.OutputOptions,
		PerStep:             params.PerStep,
	})
	if err != nil {
		return nil, err
	}

	workflowResult := asWorkflowResult(result)
	if workflowResult != nil {
		r.WorkflowRunStatus = workflowResult.Status
		if workflowResult.Status != WorkflowRunStatusSuspended {
			if r.cleanup != nil {
				r.cleanup()
			}
		}
	}

	return workflowResult, nil
}

// internalResume is the internal resume implementation with snapshot loading.
// TS equivalent: protected async _resume(params)
func (r *Run) internalResume(params ResumeParams) (*WorkflowResult, error) {
	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Load snapshot from storage
	var snapshot *WorkflowRunState
	if r.mastra != nil {
		if store := r.mastra.GetWorkflowsStore(); store != nil {
			record, err := store.GetWorkflowRunByID(context.Background(), GetWorkflowRunByIDParams{
				WorkflowName: r.WorkflowID,
				RunID:        r.RunID,
			})
			if err == nil && record != nil {
				snapshot = SnapshotToWorkflowRunState(record.Snapshot)
			}
		}
	}

	if snapshot == nil {
		return nil, fmt.Errorf("no snapshot found for this workflow run: %s %s", r.WorkflowID, r.RunID)
	}

	if snapshot.Status != WorkflowRunStatusSuspended {
		return nil, errors.New("this workflow run was not suspended")
	}

	// Resolve step from label if provided
	var snapshotResumeLabel *ResumeLabel
	if params.Label != "" && snapshot.ResumeLabels != nil {
		if rl, ok := snapshot.ResumeLabels[params.Label]; ok {
			snapshotResumeLabel = &rl
		}
	}

	// Determine the step(s) to resume
	steps := params.Step
	if snapshotResumeLabel != nil && snapshotResumeLabel.StepID != "" {
		steps = []string{snapshotResumeLabel.StepID}
	}

	if len(steps) == 0 {
		// Auto-detect suspended steps from suspendedPaths
		suspendedStepPaths := [][]string{}
		for stepID := range snapshot.SuspendedPaths {
			stepResult, ok := snapshot.Context[stepID]
			if ok {
				if m, ok := stepResult.(map[string]any); ok {
					if status, ok := m["status"].(string); ok && status == "suspended" {
						nestedPath, _ := getNestedPath(m)
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
			return nil, fmt.Errorf("multiple suspended steps found. Please specify which step to resume using the 'step' parameter")
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

	// Validate resume data
	suspendedStep := r.WorkflowSteps[steps[0]]
	resumeData, err := r.validateResumeData(params.ResumeData, suspendedStep)
	if err != nil {
		return nil, err
	}

	// Build step results from snapshot context
	stepResults := make(map[string]StepResult)
	if snapshot.Context != nil {
		for k, v := range snapshot.Context {
			if m, ok := v.(map[string]any); ok {
				stepResults[k] = mapToStepResult(m)
			} else if sr, ok := v.(StepResult); ok {
				stepResults[k] = sr
			}
		}
	}

	// Restore request context from snapshot
	if snapshot.RequestContext != nil {
		for key, value := range snapshot.RequestContext {
			if !rc.Has(key) {
				rc.Set(key, value)
			}
		}
	}

	var resumePath []int
	if len(steps) > 0 {
		if paths, ok := snapshot.SuspendedPaths[steps[0]]; ok {
			resumePath = paths
		}
	}

	result, err := r.executionEngine.Execute(ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		ResourceID:          r.ResourceID,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		Input:               snapshot.Context["input"],
		InitialState:        snapshotValueToAny(snapshot.Value),
		PubSub:              r.pubsub,
		RequestContext:      rc,
		AbortCtx:            r.abortCtx,
		AbortCancel:         r.abortCancel,
		OutputWriter:        params.OutputWriter,
		OutputOptions:       params.OutputOptions,
		PerStep:             params.PerStep,
		Resume: &ResumeExecuteParams{
			Steps:             steps,
			StepResults:       stepResults,
			ResumePayload:     resumeData,
			ResumePath:        resumePath,
			StepExecutionPath: snapshot.StepExecutionPath,
			Label:             params.Label,
			ForEachIndex:      resolveForEachIndex(params.ForEachIndex, snapshotResumeLabel),
		},
	})
	if err != nil {
		return nil, err
	}

	workflowResult := asWorkflowResult(result)
	if workflowResult != nil {
		r.WorkflowRunStatus = workflowResult.Status
	}

	return workflowResult, nil
}

// internalRestart is the internal restart implementation with snapshot loading.
// TS equivalent: protected async _restart({requestContext, outputWriter, ...})
func (r *Run) internalRestart(params RestartParams) (*WorkflowResult, error) {
	if r.WorkflowEngineType != "default" {
		return nil, fmt.Errorf("restart() is not supported on %s workflows", r.WorkflowEngineType)
	}

	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Load snapshot from storage
	var snapshot *WorkflowRunState
	if r.mastra != nil {
		if store := r.mastra.GetWorkflowsStore(); store != nil {
			record, err := store.GetWorkflowRunByID(context.Background(), GetWorkflowRunByIDParams{
				WorkflowName: r.WorkflowID,
				RunID:        r.RunID,
			})
			if err == nil && record != nil {
				snapshot = SnapshotToWorkflowRunState(record.Snapshot)
			}
		}
	}

	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found for run %s", r.RunID)
	}

	if snapshot.Status != WorkflowRunStatusRunning && snapshot.Status != WorkflowRunStatusWaiting {
		// Check for pending nested workflow
		if snapshot.Status == WorkflowRunStatusPending && snapshot.Context != nil {
			if _, hasInput := snapshot.Context["input"]; !hasInput {
				return nil, errors.New("this workflow run was not active")
			}
		} else {
			return nil, errors.New("this workflow run was not active")
		}
	}

	// Build restart data from snapshot
	restartData := &RestartExecutionParams{
		ActivePaths:       snapshot.ActivePaths,
		ActiveStepsPath:   snapshot.ActiveStepsPath,
		StepExecutionPath: snapshot.StepExecutionPath,
	}

	// Build step results from context
	stepResults := make(map[string]StepResult)
	if snapshot.Context != nil {
		for k, v := range snapshot.Context {
			if m, ok := v.(map[string]any); ok {
				stepResults[k] = mapToStepResult(m)
			} else if sr, ok := v.(StepResult); ok {
				stepResults[k] = sr
			}
		}
	}
	restartData.StepResults = stepResults

	if len(snapshot.Value) > 0 {
		state := make(map[string]any)
		for k, v := range snapshot.Value {
			state[k] = v
		}
		restartData.State = state
	}

	// Restore request context from snapshot
	if snapshot.RequestContext != nil {
		for key, value := range snapshot.RequestContext {
			if !rc.Has(key) {
				rc.Set(key, value)
			}
		}
	}

	result, err := r.executionEngine.Execute(ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		ResourceID:          r.ResourceID,
		DisableScorers:      r.DisableScorers,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		Restart:             restartData,
		PubSub:              r.pubsub,
		RetryConfig:         r.retryConfig,
		RequestContext:      rc,
		AbortCtx:            r.abortCtx,
		AbortCancel:         r.abortCancel,
		OutputWriter:        params.OutputWriter,
	})
	if err != nil {
		return nil, err
	}

	workflowResult := asWorkflowResult(result)
	if workflowResult != nil {
		r.WorkflowRunStatus = workflowResult.Status
		if workflowResult.Status != WorkflowRunStatusSuspended {
			if r.cleanup != nil {
				r.cleanup()
			}
		}
	}

	return workflowResult, nil
}

// internalTimeTravel is the internal time travel implementation with snapshot loading.
// TS equivalent: protected async _timeTravel({inputData, step, context, ...})
func (r *Run) internalTimeTravel(params TimeTravelParams) (*WorkflowResult, error) {
	if len(params.Step) == 0 {
		return nil, errors.New("step is required and must be a valid step or array of steps")
	}

	rc := params.RequestContext
	if rc == nil {
		rc = requestcontext.NewRequestContext()
	}

	// Load snapshot from storage for time travel
	var snapshot *WorkflowRunState
	if r.mastra != nil {
		if store := r.mastra.GetWorkflowsStore(); store != nil {
			record, err := store.GetWorkflowRunByID(context.Background(), GetWorkflowRunByIDParams{
				WorkflowName: r.WorkflowID,
				RunID:        r.RunID,
			})
			if err == nil && record != nil {
				snapshot = SnapshotToWorkflowRunState(record.Snapshot)
			}
		}
	}

	if snapshot == nil {
		return nil, fmt.Errorf("snapshot not found for run %s", r.RunID)
	}

	if snapshot.Status == WorkflowRunStatusRunning {
		return nil, errors.New("this workflow run is still running, cannot time travel")
	}

	// Validate time travel input data
	inputData := params.InputData
	if inputData != nil && len(params.Step) == 1 {
		if step, ok := r.WorkflowSteps[params.Step[0]]; ok {
			validated, err := r.validateTimeTravelInputData(inputData, step)
			if err != nil {
				return nil, err
			}
			inputData = validated
		}
	}

	timeTravelData, err := CreateTimeTravelExecutionParams(CreateTimeTravelParams{
		Steps:        params.Step,
		InputData:    inputData,
		ResumeData:   params.ResumeData,
		Context:      params.Context,
		Snapshot:      *snapshot,
		InitialState: nil, // Use snapshot value
		Graph:        r.executionGraph,
		PerStep:      params.PerStep,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create time travel params: %w", err)
	}

	// Restore request context from snapshot
	if snapshot.RequestContext != nil {
		for key, value := range snapshot.RequestContext {
			if !rc.Has(key) {
				rc.Set(key, value)
			}
		}
	}

	result, err := r.executionEngine.Execute(ExecuteParams{
		WorkflowID:          r.WorkflowID,
		RunID:               r.RunID,
		ResourceID:          r.ResourceID,
		DisableScorers:      r.DisableScorers,
		Graph:               r.executionGraph,
		SerializedStepGraph: r.SerializedStepGraph,
		TimeTravel:          timeTravelData,
		PubSub:              r.pubsub,
		RetryConfig:         r.retryConfig,
		RequestContext:      rc,
		AbortCtx:            r.abortCtx,
		AbortCancel:         r.abortCancel,
		OutputWriter:        params.OutputWriter,
		OutputOptions:       params.OutputOptions,
		PerStep:             params.PerStep,
	})
	if err != nil {
		return nil, err
	}

	workflowResult := asWorkflowResult(result)
	if workflowResult != nil {
		r.WorkflowRunStatus = workflowResult.Status
		if workflowResult.Status != WorkflowRunStatusSuspended {
			if r.cleanup != nil {
				r.cleanup()
			}
		}
	}

	return workflowResult, nil
}

// WatchAsync subscribes to workflow events for this run (async variant).
// TS equivalent: async watchAsync(cb)
func (r *Run) WatchAsync(cb func(event map[string]any)) func() {
	return r.Watch(cb)
}

// GetExecutionResults returns the execution results if available.
// TS equivalent: _getExecutionResults()
func (r *Run) GetExecutionResults() *WorkflowResult {
	if r.streamOutput != nil {
		result, err := r.streamOutput.AwaitResult()
		if err != nil {
			return nil
		}
		// Convert stream.WorkflowResult (map[string]any) to *WorkflowResult
		wr := asWorkflowResult(result)
		return wr
	}
	return r.executionResults
}

// ---------------------------------------------------------------------------
// Run Streaming Methods
// ---------------------------------------------------------------------------

// StreamLegacyParams holds parameters for Run.StreamLegacy().
type StreamLegacyParams struct {
	InputData      any
	RequestContext *requestcontext.RequestContext
	OnChunk        func(chunk StreamEvent) error
}

// StreamLegacyResult holds the result of Run.StreamLegacy().
type StreamLegacyResult struct {
	Stream          <-chan StreamEvent
	GetWorkflowState func() (*WorkflowResult, error)
}

// StreamLegacy starts the workflow and returns a stream of legacy StreamEvent events.
// TS equivalent: streamLegacy({inputData, requestContext, onChunk, ...})
func (r *Run) StreamLegacy(params StreamLegacyParams) *StreamLegacyResult {
	if r.closeStreamAction != nil {
		// Already streaming, return observer stream
		ch := make(chan StreamEvent, 256)
		unwatch := r.Watch(func(event map[string]any) {
			e := StreamEvent{}
			if t, ok := event["type"].(string); ok {
				e.Type = StreamEventType(t)
			}
			e.Payload = event
			ch <- e
		})
		_ = unwatch // caller should manage via close
		return &StreamLegacyResult{
			Stream: ch,
			GetWorkflowState: func() (*WorkflowResult, error) {
				return r.executionResults, nil
			},
		}
	}

	ch := make(chan StreamEvent, 256)

	unwatch := r.Watch(func(event map[string]any) {
		e := StreamEvent{}
		if t, ok := event["type"].(string); ok {
			e.Type = StreamEventType(t)
		}
		e.Payload = event
		ch <- e
		if params.OnChunk != nil {
			_ = params.OnChunk(e)
		}
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// Publish start event
	if r.pubsub != nil {
		topic := fmt.Sprintf("workflow.events.v2.%s", r.RunID)
		_ = r.pubsub.Publish(topic, events.PublishEvent{
			Type:  "watch",
			RunID: r.RunID,
			Data:  map[string]any{"type": "workflow-start", "payload": map[string]any{"runId": r.RunID}},
		})
	}

	// Start execution in background
	go func() {
		result, _ := r.internalStart(StartParams{
			InputData:      params.InputData,
			RequestContext: params.RequestContext,
		})
		r.executionResults = result
		if result == nil || result.Status != WorkflowRunStatusSuspended {
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
		}
	}()

	return &StreamLegacyResult{
		Stream: ch,
		GetWorkflowState: func() (*WorkflowResult, error) {
			return r.executionResults, nil
		},
	}
}

// StreamParams holds parameters for Run.Stream().
type StreamParams struct {
	InputData      any
	InitialState   any
	RequestContext *requestcontext.RequestContext
	CloseOnSuspend bool
	OutputOptions  *OutputOptions
	PerStep        bool
}

// Stream starts the workflow execution and returns a WorkflowRunOutput with a
// channel-based stream of WorkflowStreamEvent chunks.
// TS equivalent: stream({inputData, requestContext, closeOnSuspend, ...})
func (r *Run) Stream(params StreamParams) *streamPkg.WorkflowRunOutput {
	closeOnSuspend := params.CloseOnSuspend

	if r.closeStreamAction != nil && r.streamOutput != nil {
		return r.streamOutput
	}

	r.closeStreamAction = func() {} // placeholder

	ch := make(chan streamPkg.WorkflowStreamEvent, 256)

	unwatch := r.Watch(func(event map[string]any) {
		chunk := watchEventToStreamEvent(event, r.RunID)
		ch <- chunk
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// Start execution in background
	go func() {
		result, err := r.internalStart(StartParams{
			InputData:      params.InputData,
			InitialState:   params.InitialState,
			RequestContext: params.RequestContext,
			OutputWriter: func(chunk any) error {
				if r.pubsub != nil {
					topic := fmt.Sprintf("workflow.events.v2.%s", r.RunID)
					_ = r.pubsub.Publish(topic, events.PublishEvent{
						Type:  "watch",
						RunID: r.RunID,
						Data:  chunk,
					})
				}
				return nil
			},
			OutputOptions: params.OutputOptions,
			PerStep:       params.PerStep,
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
		if closeOnSuspend {
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
		} else if result == nil || result.Status != WorkflowRunStatusSuspended {
			if r.closeStreamAction != nil {
				r.closeStreamAction()
			}
		}
		if r.streamOutput != nil && result != nil {
			r.streamOutput.UpdateResults(streamPkg.WorkflowResult{
				"status": string(result.Status),
				"result": result.Result,
			})
		}
	}()

	r.streamOutput = streamPkg.NewWorkflowRunOutput(streamPkg.WorkflowRunOutputParams{
		RunID:      r.RunID,
		WorkflowID: r.WorkflowID,
		Stream:     ch,
	})

	return r.streamOutput
}

// ResumeStreamParams holds parameters for Run.ResumeStream().
type ResumeStreamParams struct {
	Step           []string
	ResumeData     any
	RequestContext *requestcontext.RequestContext
	ForEachIndex   *int
	OutputOptions  *OutputOptions
	PerStep        bool
}

// ResumeStream resumes a suspended workflow and returns a streaming output.
// TS equivalent: resumeStream({step, resumeData, requestContext, ...})
func (r *Run) ResumeStream(params ResumeStreamParams) *streamPkg.WorkflowRunOutput {
	r.closeStreamAction = func() {} // placeholder

	ch := make(chan streamPkg.WorkflowStreamEvent, 256)

	unwatch := r.Watch(func(event map[string]any) {
		chunk := watchEventToStreamEvent(event, r.RunID)
		ch <- chunk
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// Resume execution in background
	go func() {
		result, err := r.internalResume(ResumeParams{
			ResumeData:     params.ResumeData,
			Step:           params.Step,
			RequestContext: params.RequestContext,
			ForEachIndex:   params.ForEachIndex,
			OutputOptions:  params.OutputOptions,
			PerStep:        params.PerStep,
			OutputWriter: func(chunk any) error {
				if c, ok := chunk.(streamPkg.WorkflowStreamEvent); ok {
					ch <- c
				}
				return nil
			},
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
		if r.closeStreamAction != nil {
			r.closeStreamAction()
		}
		if r.streamOutput != nil && result != nil {
			r.streamOutput.UpdateResults(streamPkg.WorkflowResult{
				"status": string(result.Status),
				"result": result.Result,
			})
		}
	}()

	r.streamOutput = streamPkg.NewWorkflowRunOutput(streamPkg.WorkflowRunOutputParams{
		RunID:      r.RunID,
		WorkflowID: r.WorkflowID,
		Stream:     ch,
	})

	return r.streamOutput
}

// TimeTravelStreamParams holds parameters for Run.TimeTravelStream().
type TimeTravelStreamParams struct {
	InputData      any
	ResumeData     any
	InitialState   any
	Step           []string
	Context        TimeTravelContext
	RequestContext *requestcontext.RequestContext
	OutputOptions  *OutputOptions
	PerStep        bool
}

// TimeTravelStream executes a time travel to a specific step and returns a streaming output.
// TS equivalent: timeTravelStream({inputData, step, context, ...})
func (r *Run) TimeTravelStream(params TimeTravelStreamParams) *streamPkg.WorkflowRunOutput {
	r.closeStreamAction = func() {} // placeholder

	ch := make(chan streamPkg.WorkflowStreamEvent, 256)

	unwatch := r.Watch(func(event map[string]any) {
		chunk := watchEventToStreamEvent(event, r.RunID)
		ch <- chunk
	})

	r.closeStreamAction = func() {
		unwatch()
		close(ch)
	}

	// TimeTravel execution in background
	go func() {
		result, err := r.internalTimeTravel(TimeTravelParams{
			InputData:      params.InputData,
			ResumeData:     params.ResumeData,
			InitialState:   params.InitialState,
			Step:           params.Step,
			Context:        params.Context,
			RequestContext: params.RequestContext,
			OutputOptions:  params.OutputOptions,
			PerStep:        params.PerStep,
			OutputWriter: func(chunk any) error {
				if c, ok := chunk.(streamPkg.WorkflowStreamEvent); ok {
					ch <- c
				}
				return nil
			},
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
		if r.closeStreamAction != nil {
			r.closeStreamAction()
		}
		if r.streamOutput != nil && result != nil {
			r.streamOutput.UpdateResults(streamPkg.WorkflowResult{
				"status": string(result.Status),
				"result": result.Result,
			})
		}
	}()

	r.streamOutput = streamPkg.NewWorkflowRunOutput(streamPkg.WorkflowRunOutputParams{
		RunID:      r.RunID,
		WorkflowID: r.WorkflowID,
		Stream:     ch,
	})

	return r.streamOutput
}

// ObserveStreamLegacy returns a channel-based stream of legacy StreamEvent from the current run.
// TS equivalent: observeStreamLegacy()
func (r *Run) ObserveStreamLegacy() <-chan StreamEvent {
	ch := make(chan StreamEvent, 256)

	unwatch := r.Watch(func(event map[string]any) {
		e := StreamEvent{}
		if t, ok := event["type"].(string); ok {
			e.Type = StreamEventType(t)
		}
		e.Payload = event
		ch <- e
	})

	r.observerHandlers = append(r.observerHandlers, func() {
		unwatch()
		close(ch)
	})

	return ch
}

// ObserveStream returns a channel-based stream of WorkflowStreamEvent from the current run.
// TS equivalent: observeStream()
func (r *Run) ObserveStream() <-chan streamPkg.WorkflowStreamEvent {
	if r.streamOutput == nil {
		ch := make(chan streamPkg.WorkflowStreamEvent)
		close(ch)
		return ch
	}
	return r.streamOutput.FullStream()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// getNestedPath extracts the nested workflow path from a step result map.
func getNestedPath(m map[string]any) ([]string, bool) {
	sp, ok := m["suspendPayload"]
	if !ok {
		return nil, false
	}
	spMap, ok := sp.(map[string]any)
	if !ok {
		return nil, false
	}
	meta, ok := spMap["__workflow_meta"]
	if !ok {
		return nil, false
	}
	metaMap, ok := meta.(map[string]any)
	if !ok {
		return nil, false
	}
	path, ok := metaMap["path"]
	if !ok {
		return nil, false
	}
	pathSlice, ok := path.([]any)
	if !ok {
		return nil, false
	}
	result := make([]string, len(pathSlice))
	for i, p := range pathSlice {
		result[i] = fmt.Sprintf("%v", p)
	}
	return result, true
}

// watchEventToStreamEvent converts a Watch callback event (map[string]any) into
// a proper streamPkg.WorkflowStreamEvent (ChunkType struct).
func watchEventToStreamEvent(event map[string]any, runID string) streamPkg.WorkflowStreamEvent {
	chunk := streamPkg.WorkflowStreamEvent{}
	if t, ok := event["type"].(string); ok {
		chunk.Type = t
	}
	chunk.RunID = runID
	if from, ok := event["from"].(streamPkg.ChunkFrom); ok {
		chunk.From = from
	}
	if payload, ok := event["payload"]; ok {
		chunk.Payload = payload
	} else if data, ok := event["data"]; ok {
		chunk.Payload = data
	}
	return chunk
}

// resolveForEachIndex returns the ForEachIndex from params, falling back to the
// snapshotResumeLabel's ForeachIndex if available.
func resolveForEachIndex(paramsIndex *int, label *ResumeLabel) *int {
	if paramsIndex != nil {
		return paramsIndex
	}
	if label != nil && label.ForeachIndex != nil {
		return label.ForeachIndex
	}
	return nil
}

// snapshotValueToAny converts the snapshot Value (map[string]string) to map[string]any.
func snapshotValueToAny(value map[string]string) map[string]any {
	if len(value) == 0 {
		return map[string]any{}
	}
	result := make(map[string]any, len(value))
	for k, v := range value {
		result[k] = v
	}
	return result
}

// asWorkflowResult attempts to convert a generic result from ExecutionEngine.Execute
// into a *WorkflowResult.
func asWorkflowResult(result any) *WorkflowResult {
	if result == nil {
		return nil
	}
	if wr, ok := result.(*WorkflowResult); ok {
		return wr
	}
	if wr, ok := result.(WorkflowResult); ok {
		return &wr
	}
	// If result is a map, try to extract status
	if m, ok := result.(map[string]any); ok {
		wr := &WorkflowResult{}
		if status, ok := m["status"].(string); ok {
			wr.Status = WorkflowRunStatus(status)
		}
		if res, ok := m["result"]; ok {
			wr.Result = res
		}
		if steps, ok := m["steps"].(map[string]StepResult); ok {
			wr.Steps = steps
		}
		if errVal, ok := m["error"]; ok {
			if e, ok := errVal.(error); ok {
				wr.Error = e
			}
		}
		return wr
	}
	return nil
}
