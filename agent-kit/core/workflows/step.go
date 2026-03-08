// Ported from: packages/core/src/workflows/step.ts
package workflows

import (
	"context"

	"github.com/brainlet/brainkit/agent-kit/core/events"
	"github.com/brainlet/brainkit/agent-kit/core/observability/types"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Suspend Options
// ---------------------------------------------------------------------------

// SuspendOptions holds options passed to the suspend function.
type SuspendOptions struct {
	ResumeLabel []string
	Extra       map[string]any
}

// ---------------------------------------------------------------------------
// Execute Function Params
// ---------------------------------------------------------------------------

// ExecuteFunctionParams holds the parameters passed to step execute functions.
// TS equivalent: ExecuteFunctionParams<TState, TStepInput, TStepOutput, TResume, TSuspend, EngineType, TRequestContext>
type ExecuteFunctionParams struct {
	RunID           string
	ResourceID      string
	WorkflowID      string
	Mastra          Mastra
	RequestContext  *requestcontext.RequestContext
	InputData       any
	State           any
	SetState        func(state any) error
	ResumeData      any
	SuspendData     any
	RetryCount      int
	GetInitData     func() any
	GetStepResult   func(step any) any
	Suspend         func(suspendPayload any, suspendOptions *SuspendOptions) error
	Bail            func(result any)
	Abort           func()
	Resume          *ResumeInfo
	Restart         bool
	PubSub          events.PubSub
	StreamFormat    string // "legacy", "vnext", or ""
	Engine          any
	AbortCtx        context.Context
	Writer          any // ToolStream equivalent
	OutputWriter    OutputWriter
	ValidateSchemas bool
	Observability   *types.ObservabilityContext
}

// ---------------------------------------------------------------------------
// Condition Function Params
// ---------------------------------------------------------------------------

// ConditionFunctionParams holds the parameters passed to condition functions.
// Same as ExecuteFunctionParams but without SetState and Suspend.
// TS equivalent: ConditionFunctionParams<TState, TStepInput, TStepOutput, TResumeSchema, TSuspendSchema, EngineType>
type ConditionFunctionParams = ExecuteFunctionParams

// ---------------------------------------------------------------------------
// Function types
// ---------------------------------------------------------------------------

// ExecuteFunction is the signature for step execution functions.
// TS equivalent: ExecuteFunction<TState, TStepInput, TStepOutput, TResumeSchema, TSuspendSchema, EngineType>
type ExecuteFunction func(params *ExecuteFunctionParams) (any, error)

// ConditionFunction is the signature for condition functions used in branching.
// TS equivalent: ConditionFunction<TState, TStepInput, TStepOutput, TResumeSchema, TSuspendSchema, EngineType>
type ConditionFunction func(params *ConditionFunctionParams) (bool, error)

// LoopConditionFunction is the signature for loop condition functions.
// TS equivalent: LoopConditionFunction<TState, TStepInput, TStepOutput, TResumeSchema, TSuspendSchema, EngineType>
type LoopConditionFunction func(params *ConditionFunctionParams, iterationCount int) (bool, error)

// ---------------------------------------------------------------------------
// Step
// ---------------------------------------------------------------------------

// Step defines a single executable step in a workflow.
// TS equivalent: interface Step<TStepId, TState, TInput, TOutput, TResume, TSuspend, TEngineType, TRequestContext>
type Step struct {
	ID                  string               `json:"id"`
	Description         string               `json:"description,omitempty"`
	InputSchema         SchemaWithValidation  `json:"-"`
	OutputSchema        SchemaWithValidation  `json:"-"`
	ResumeSchema        SchemaWithValidation  `json:"-"`
	SuspendSchema       SchemaWithValidation  `json:"-"`
	StateSchema         SchemaWithValidation  `json:"-"`
	RequestContextSchema SchemaWithValidation `json:"-"`
	Execute             ExecuteFunction       `json:"-"`
	Scorers             any                   `json:"-"` // DynamicArgument[MastraScorers]
	Retries             *int                  `json:"retries,omitempty"`
	Component           string                `json:"component,omitempty"`
	Metadata            StepMetadata          `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// StepParams
// ---------------------------------------------------------------------------

// StepParams holds the parameters for creating a Step via CreateStep.
// TS equivalent: StepParams<TStepId, TStateSchema, TInputSchema, TOutputSchema, TResumeSchema, TSuspendSchema, TRequestContextSchema>
type StepParams struct {
	ID                   string
	Description          string
	InputSchema          SchemaWithValidation
	OutputSchema         SchemaWithValidation
	ResumeSchema         SchemaWithValidation
	SuspendSchema        SchemaWithValidation
	StateSchema          SchemaWithValidation
	RequestContextSchema SchemaWithValidation
	Retries              *int
	Scorers              any // DynamicArgument[MastraScorers]
	Metadata             StepMetadata
	Execute              ExecuteFunction
}

// ---------------------------------------------------------------------------
// GetStepResult
// ---------------------------------------------------------------------------

// GetStepResult retrieves a step result output from the step results map.
// Returns nil if the step is not found or has not succeeded.
// TS equivalent: export const getStepResult = (stepResults, step) => { ... }
func GetStepResult(stepResults map[string]StepResult, step any) any {
	var stepID string
	switch v := step.(type) {
	case string:
		stepID = v
	case *Step:
		if v == nil {
			return nil
		}
		stepID = v.ID
	case Step:
		stepID = v.ID
	default:
		return nil
	}

	result, ok := stepResults[stepID]
	if !ok {
		return nil
	}

	if result.Status == StepStatusSuccess {
		return result.Output
	}

	return nil
}
