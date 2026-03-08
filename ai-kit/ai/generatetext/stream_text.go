// Ported from: packages/ai/src/generate-text/stream-text.ts
package generatetext

import (
	"context"
	"fmt"
)

// StreamTextTransform is a transformation applied to the stream.
type StreamTextTransform func(options StreamTextTransformOptions) func(input <-chan TextStreamPart, output chan<- TextStreamPart)

// StreamTextTransformOptions contains options for a stream text transform.
type StreamTextTransformOptions struct {
	Tools      ToolSet
	StopStream func()
}

// StreamTextOnErrorCallback is called when an error occurs during streaming.
type StreamTextOnErrorCallback func(event StreamTextErrorEvent)

// StreamTextErrorEvent contains the error event data.
type StreamTextErrorEvent struct {
	Error interface{}
}

// StreamTextOnStepFinishCallback is called when each step finishes.
type StreamTextOnStepFinishCallback func(event OnStepFinishEvent)

// StreamTextOnChunkCallback is called for each chunk of the stream.
type StreamTextOnChunkCallback func(event StreamTextChunkEvent)

// StreamTextChunkEvent contains the chunk event data.
type StreamTextChunkEvent struct {
	Chunk TextStreamPart
}

// StreamTextOnFinishCallback is called when all steps are finished.
type StreamTextOnFinishCallback func(event OnFinishEvent)

// StreamTextOnAbortCallback is called when the stream is aborted.
type StreamTextOnAbortCallback func(event StreamTextAbortEvent)

// StreamTextAbortEvent contains the abort event data.
type StreamTextAbortEvent struct {
	Steps []StepResult
}

// StreamTextOnStartCallback is called when the streamText operation begins.
type StreamTextOnStartCallback func(event OnStartEvent)

// StreamTextOnStepStartCallback is called when a step begins.
type StreamTextOnStepStartCallback func(event OnStepStartEvent)

// StreamTextOnToolCallStartCallback is called before tool execution.
type StreamTextOnToolCallStartCallback func(event OnToolCallStartEvent)

// StreamTextOnToolCallFinishCallback is called after tool execution.
type StreamTextOnToolCallFinishCallback func(event OnToolCallFinishEvent)

// StreamTextOptions contains all options for the streamText function.
type StreamTextOptions struct {
	// Ctx is the Go context for cancellation (replaces AbortSignal).
	Ctx context.Context

	// Model is the language model to use.
	Model LanguageModel

	// Tools are the tools that the model can call.
	Tools ToolSet

	// ToolChoice is the tool choice strategy. Default: "auto".
	ToolChoice *ToolChoice

	// System is a system message for the prompt.
	System interface{} // string | SystemModelMessage | []SystemModelMessage

	// Prompt is a simple text prompt (use either Prompt or Messages, not both).
	Prompt string

	// Messages is a list of messages (use either Prompt or Messages, not both).
	Messages []ModelMessage

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

	// Timeout configuration.
	Timeout *TimeoutConfiguration

	// StopWhen is the condition(s) for stopping generation.
	StopWhen []StopCondition

	// Output is the specification for structured outputs.
	Output Output

	// ProviderOptions are additional provider-specific options.
	ProviderOptions ProviderOptions

	// ActiveTools limits which tools are available for the model to call.
	ActiveTools []string

	// PrepareStep is an optional function to provide different settings for a step.
	PrepareStep PrepareStepFunction

	// RepairToolCall attempts to repair a tool call that failed to parse.
	RepairToolCall ToolCallRepairFunction

	// ExperimentalContext is a user-defined context object.
	ExperimentalContext interface{}

	// Include controls what data is included in step results.
	Include *StreamTextIncludeSettings

	// Transform is an optional transformation applied to the stream.
	Transform StreamTextTransform

	// GenerateID is an optional ID generator (for testing).
	GenerateID IdGenerator

	// Callbacks
	OnStart          StreamTextOnStartCallback
	OnStepStart      StreamTextOnStepStartCallback
	OnToolCallStart  StreamTextOnToolCallStartCallback
	OnToolCallFinish StreamTextOnToolCallFinishCallback
	OnChunk          StreamTextOnChunkCallback
	OnError          StreamTextOnErrorCallback
	OnStepFinish     StreamTextOnStepFinishCallback
	OnFinish         StreamTextOnFinishCallback
	OnAbort          StreamTextOnAbortCallback
}

// StreamTextIncludeSettings controls what data is included in step results.
type StreamTextIncludeSettings struct {
	// RequestBody controls whether to retain the request body.
	RequestBody *bool
}

// EnrichedStreamPart is a TextStreamPart enriched with partial output information.
// Used for output parsing transforms.
type EnrichedStreamPart struct {
	TextStreamPart
	PartialOutput interface{}
}

// StreamText generates text and calls tools for a given prompt using a language model.
//
// This function streams the output. If you do not want to stream the output, use GenerateText instead.
//
// In the TypeScript SDK, this returns a StreamTextResult with lazy promise properties.
// In Go, we return a StreamTextResult that is populated as the stream is consumed.
//
// TODO: Full implementation requires porting the streaming infrastructure, telemetry,
// and the multi-step loop with stitchable streams. This is a structural port of the
// function signature, types, and core flow.
func StreamText(opts StreamTextOptions) (*StreamTextResult, error) {
	if opts.Ctx == nil {
		opts.Ctx = context.Background()
	}

	if opts.Model == nil {
		return nil, fmt.Errorf("model is required")
	}

	if opts.StopWhen == nil {
		opts.StopWhen = []StopCondition{StepCountIs(1)}
	}

	// TODO: Full implementation of the streaming pipeline:
	// 1. Standardize prompt
	// 2. Prepare call settings
	// 3. Set up telemetry
	// 4. Create stitchable stream for multi-step
	// 5. For each step:
	//    a. Prepare step (call PrepareStep if provided)
	//    b. Convert to language model prompt
	//    c. Call model.doStream
	//    d. Run tools transformation
	//    e. Process results and emit to stream
	// 6. Apply transforms
	// 7. Handle output parsing
	// 8. Wire up callbacks (onChunk, onStepFinish, onFinish, etc.)

	return &StreamTextResult{}, nil
}
