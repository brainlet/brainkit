// Ported from: packages/ai/src/agent/tool-loop-agent-settings.ts
package agent

import (
	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// OnStartCallback is called when the agent operation begins, before any LLM calls.
type OnStartCallback func(event gt.OnStartEvent)

// OnStepStartCallback is called when a step (LLM call) begins.
type OnStepStartCallback func(event gt.OnStepStartEvent)

// OnToolCallStartCallback is called before each tool execution begins.
type OnToolCallStartCallback func(event gt.OnToolCallStartEvent)

// OnToolCallFinishCallback is called after each tool execution completes.
type OnToolCallFinishCallback func(event gt.OnToolCallFinishEvent)

// OnStepFinishCallback is called when each step (LLM call) is finished.
type OnStepFinishCallback func(event gt.OnStepFinishEvent)

// OnFinishCallback is called when all steps are finished and the response is complete.
type OnFinishCallback func(event gt.OnFinishEvent)

// ToolLoopAgentSettings contains configuration options for a ToolLoopAgent.
type ToolLoopAgentSettings struct {
	// ID is the id of the agent.
	ID string

	// Instructions for the agent.
	// It can be a string, or, if you need to pass additional provider options (e.g. for caching),
	// a SystemModelMessage or a slice of SystemModelMessage.
	// In Go, use one of: string, gt.SystemModelMessage, []gt.SystemModelMessage.
	Instructions interface{}

	// Model is the language model to use.
	Model gt.LanguageModel

	// Tools are the tools that the model can call.
	Tools gt.ToolSet

	// ToolChoice is the tool choice strategy. Default: "auto".
	ToolChoice *gt.ToolChoice

	// StopWhen is the condition(s) for stopping generation when there are tool results.
	// When the slice has multiple entries, any condition being met stops the generation.
	// Default: StepCountIs(20).
	StopWhen []gt.StopCondition

	// ExperimentalTelemetry is optional telemetry configuration.
	ExperimentalTelemetry interface{} // TODO: TelemetrySettings

	// ActiveTools limits the tools available for the model to call.
	ActiveTools []string

	// Output is an optional specification for generating structured outputs.
	Output gt.Output

	// PrepareStep is an optional function to provide different settings for a step.
	PrepareStep gt.PrepareStepFunction

	// ExperimentalRepairToolCall attempts to repair a tool call that failed to parse.
	ExperimentalRepairToolCall gt.ToolCallRepairFunction

	// Callbacks

	// ExperimentalOnStart is called when the agent operation begins, before any LLM calls.
	ExperimentalOnStart OnStartCallback

	// ExperimentalOnStepStart is called when a step (LLM call) begins.
	ExperimentalOnStepStart OnStepStartCallback

	// ExperimentalOnToolCallStart is called before each tool execution begins.
	ExperimentalOnToolCallStart OnToolCallStartCallback

	// ExperimentalOnToolCallFinish is called after each tool execution completes.
	ExperimentalOnToolCallFinish OnToolCallFinishCallback

	// OnStepFinish is called when each step (LLM call) is finished.
	OnStepFinish OnStepFinishCallback

	// OnFinish is called when all steps are finished and the response is complete.
	OnFinish OnFinishCallback

	// ProviderOptions are additional provider-specific options.
	ProviderOptions gt.ProviderOptions

	// ExperimentalContext is context that is passed into tool calls.
	ExperimentalContext interface{}

	// ExperimentalDownload is a custom download function to use for URLs.
	ExperimentalDownload interface{} // TODO: DownloadFunction

	// CallSettings (embedded from CallSettings, minus AbortSignal)

	// MaxOutputTokens is the maximum number of tokens to generate.
	MaxOutputTokens *int

	// Temperature is the sampling temperature.
	Temperature *float64

	// TopP is the nucleus sampling parameter.
	TopP *float64

	// TopK is the top-K sampling parameter.
	TopK *int

	// PresencePenalty affects likelihood of repeating prompt information.
	PresencePenalty *float64

	// FrequencyPenalty affects likelihood of repeating words/phrases.
	FrequencyPenalty *float64

	// StopSequences causes generation to stop when one is generated.
	StopSequences []string

	// Seed for reproducible generation.
	Seed *int

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional HTTP headers for the request.
	Headers map[string]string

	// PrepareCall is an optional function to prepare parameters before calling
	// generateText or streamText.
	PrepareCall PrepareCallFunc
}

// PrepareCallInput contains the input to a PrepareCall function.
// It merges the AgentCallParameters (minus callbacks) with overridable settings.
type PrepareCallInput struct {
	// From AgentCallParameters
	Prompt         string
	PromptMessages []gt.ModelMessage
	Messages       []gt.ModelMessage
	Options        interface{}

	// Overridable settings from ToolLoopAgentSettings
	Model               gt.LanguageModel
	Tools               gt.ToolSet
	MaxOutputTokens     *int
	Temperature         *float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	FrequencyPenalty    *float64
	StopSequences       []string
	Seed                *int
	Headers             map[string]string
	Instructions        interface{}
	StopWhen            []gt.StopCondition
	ExperimentalTelemetry interface{}
	ActiveTools         []string
	ProviderOptions     gt.ProviderOptions
	ExperimentalContext interface{}
	ExperimentalDownload interface{}
}

// PrepareCallResult contains the output of a PrepareCall function.
// It contains the overridden settings plus prompt fields.
type PrepareCallResult struct {
	Model               gt.LanguageModel
	Tools               gt.ToolSet
	MaxOutputTokens     *int
	Temperature         *float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	FrequencyPenalty    *float64
	StopSequences       []string
	Seed                *int
	Headers             map[string]string
	Instructions        interface{}
	StopWhen            []gt.StopCondition
	ExperimentalTelemetry interface{}
	ActiveTools         []string
	ProviderOptions     gt.ProviderOptions
	ExperimentalContext interface{}
	ExperimentalDownload interface{}

	// Prompt fields from Prompt type (minus System, which comes from Instructions)
	Prompt         string
	PromptMessages []gt.ModelMessage
	Messages       []gt.ModelMessage
}

// PrepareCallFunc is a function that prepares the parameters for the
// generateText or streamText call. You can use this to have templates
// based on call options.
type PrepareCallFunc func(input PrepareCallInput) (*PrepareCallResult, error)
