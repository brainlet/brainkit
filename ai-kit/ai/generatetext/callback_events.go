// Ported from: packages/ai/src/generate-text/callback-events.ts
package generatetext

// CallbackModelInfo contains common model information used across callback events.
// It is structurally identical to ModelInfo.
type CallbackModelInfo = ModelInfo

// OnStartEvent is passed to the onStart callback.
// Called when the generation operation begins, before any LLM calls.
type OnStartEvent struct {
	Model              CallbackModelInfo
	System             interface{} // string | SystemModelMessage | []SystemModelMessage | nil
	Prompt             interface{} // string | []ModelMessage | nil
	Messages           []ModelMessage
	Tools              ToolSet
	ToolChoice         *ToolChoice
	ActiveTools        []string
	MaxOutputTokens    *int
	Temperature        *float64
	TopP               *float64
	TopK               *int
	PresencePenalty    *float64
	FrequencyPenalty   *float64
	StopSequences      []string
	Seed               *int
	MaxRetries         int
	Timeout            *TimeoutConfiguration
	Headers            map[string]string
	ProviderOptions    ProviderOptions
	StopWhen           []StopCondition
	Output             Output
	AbortSignal        <-chan struct{}
	Include            interface{}
	FunctionID         string
	Metadata           map[string]interface{}
	ExperimentalContext interface{}
}

// OnStepStartEvent is passed to the onStepStart callback.
// Called when a step (LLM call) begins, before the provider is called.
type OnStepStartEvent struct {
	StepNumber          int
	Model               CallbackModelInfo
	System              interface{} // string | SystemModelMessage | []SystemModelMessage | nil
	Messages            []ModelMessage
	Tools               ToolSet
	ToolChoice          *LanguageModelV4ToolChoice
	ActiveTools         []string
	Steps               []StepResult
	ProviderOptions     ProviderOptions
	Timeout             *TimeoutConfiguration
	Headers             map[string]string
	StopWhen            []StopCondition
	Output              Output
	AbortSignal         <-chan struct{}
	Include             interface{}
	FunctionID          string
	Metadata            map[string]interface{}
	ExperimentalContext interface{}
}

// OnToolCallStartEvent is passed to the onToolCallStart callback.
// Called when a tool execution begins, before the tool's execute function is invoked.
type OnToolCallStartEvent struct {
	StepNumber          *int
	Model               *CallbackModelInfo
	ToolCall            ToolCall
	Messages            []ModelMessage
	AbortSignal         <-chan struct{}
	FunctionID          string
	Metadata            map[string]interface{}
	ExperimentalContext interface{}
}

// OnToolCallFinishEvent is passed to the onToolCallFinish callback.
// Called when a tool execution completes, either successfully or with an error.
type OnToolCallFinishEvent struct {
	StepNumber          *int
	Model               *CallbackModelInfo
	ToolCall            ToolCall
	Messages            []ModelMessage
	AbortSignal         <-chan struct{}
	DurationMs          float64
	FunctionID          string
	Metadata            map[string]interface{}
	ExperimentalContext interface{}

	// Success indicates whether the tool call succeeded.
	Success bool

	// Output is the tool's return value (set when Success is true).
	Output interface{}

	// Error is the error that occurred (set when Success is false).
	Error interface{}
}

// OnStepFinishEvent is passed to the onStepFinish callback.
// Called when a step (LLM call) completes. This is simply the StepResult.
type OnStepFinishEvent = StepResult

// OnFinishEvent is passed to the onFinish callback.
// Called when the entire generation completes (all steps finished).
type OnFinishEvent struct {
	StepResult

	// Steps contains results from all steps in the generation.
	Steps []StepResult

	// TotalUsage is aggregated token usage across all steps.
	TotalUsage LanguageModelUsage

	// ExperimentalContext is the final state of the user-defined context object.
	ExperimentalContext interface{}

	// FunctionID is an identifier from telemetry settings.
	FunctionID string

	// Metadata is additional metadata from telemetry settings.
	Metadata map[string]interface{}
}
