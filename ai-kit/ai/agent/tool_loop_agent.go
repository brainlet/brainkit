// Ported from: packages/ai/src/agent/tool-loop-agent.ts
package agent

import (
	"context"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// ToolLoopAgent is an agent that runs tools in a loop. In each step,
// it calls the LLM, and if there are tool calls, it executes the tools
// and calls the LLM again in a new step with the tool results.
//
// The loop continues until:
//   - A finish reasoning other than tool-calls is returned, or
//   - A tool that is invoked does not have an execute function, or
//   - A tool call needs approval, or
//   - A stop condition is met (default stop condition is StepCountIs(20))
type ToolLoopAgent struct {
	settings ToolLoopAgentSettings
}

// Ensure ToolLoopAgent implements Agent.
var _ Agent = (*ToolLoopAgent)(nil)

// NewToolLoopAgent creates a new ToolLoopAgent with the given settings.
func NewToolLoopAgent(settings ToolLoopAgentSettings) *ToolLoopAgent {
	return &ToolLoopAgent{settings: settings}
}

// Version returns the specification version of the agent interface.
func (a *ToolLoopAgent) Version() string {
	return "agent-v1"
}

// ID returns the id of the agent, or empty string if not set.
func (a *ToolLoopAgent) ID() string {
	return a.settings.ID
}

// Tools returns the tools that the agent can use.
func (a *ToolLoopAgent) Tools() gt.ToolSet {
	return a.settings.Tools
}

// prepareCallArgs builds the GenerateTextOptions/StreamTextOptions by merging
// settings, applying defaults, and optionally running the PrepareCall function.
type preparedCallArgs struct {
	Model               gt.LanguageModel
	Tools               gt.ToolSet
	ToolChoice          *gt.ToolChoice
	System              interface{} // string | SystemModelMessage | []SystemModelMessage
	Prompt              string
	PromptMessages      []gt.ModelMessage
	Messages            []gt.ModelMessage
	MaxOutputTokens     *int
	Temperature         *float64
	TopP                *float64
	TopK                *int
	PresencePenalty     *float64
	FrequencyPenalty    *float64
	StopSequences       []string
	Seed                *int
	MaxRetries          *int
	Headers             map[string]string
	StopWhen            []gt.StopCondition
	Output              gt.Output
	ProviderOptions     gt.ProviderOptions
	ActiveTools         []string
	PrepareStep         gt.PrepareStepFunction
	RepairToolCall      gt.ToolCallRepairFunction
	ExperimentalContext interface{}
	ExperimentalDownload interface{}
	ExperimentalTelemetry interface{}
}

func (a *ToolLoopAgent) prepareCall(prompt string, promptMessages []gt.ModelMessage, messages []gt.ModelMessage, options interface{}) (*preparedCallArgs, error) {
	s := a.settings

	// Default stop condition
	stopWhen := s.StopWhen
	if stopWhen == nil {
		stopWhen = []gt.StopCondition{gt.StepCountIs(20)}
	}

	// Build base call args
	base := PrepareCallInput{
		Prompt:                prompt,
		PromptMessages:        promptMessages,
		Messages:              messages,
		Options:               options,
		Model:                 s.Model,
		Tools:                 s.Tools,
		MaxOutputTokens:       s.MaxOutputTokens,
		Temperature:           s.Temperature,
		TopP:                  s.TopP,
		TopK:                  s.TopK,
		PresencePenalty:       s.PresencePenalty,
		FrequencyPenalty:      s.FrequencyPenalty,
		StopSequences:         s.StopSequences,
		Seed:                  s.Seed,
		Headers:               s.Headers,
		Instructions:          s.Instructions,
		StopWhen:              stopWhen,
		ExperimentalTelemetry: s.ExperimentalTelemetry,
		ActiveTools:           s.ActiveTools,
		ProviderOptions:       s.ProviderOptions,
		ExperimentalContext:   s.ExperimentalContext,
		ExperimentalDownload:  s.ExperimentalDownload,
	}

	// If PrepareCall is set, run it
	if s.PrepareCall != nil {
		prepared, err := s.PrepareCall(base)
		if err != nil {
			return nil, err
		}
		if prepared != nil {
			return &preparedCallArgs{
				Model:                 prepared.Model,
				Tools:                 prepared.Tools,
				ToolChoice:            s.ToolChoice,
				System:                prepared.Instructions,
				Prompt:                prepared.Prompt,
				PromptMessages:        prepared.PromptMessages,
				Messages:              prepared.Messages,
				MaxOutputTokens:       prepared.MaxOutputTokens,
				Temperature:           prepared.Temperature,
				TopP:                  prepared.TopP,
				TopK:                  prepared.TopK,
				PresencePenalty:       prepared.PresencePenalty,
				FrequencyPenalty:      prepared.FrequencyPenalty,
				StopSequences:         prepared.StopSequences,
				Seed:                  prepared.Seed,
				MaxRetries:            s.MaxRetries,
				Headers:               prepared.Headers,
				StopWhen:              prepared.StopWhen,
				Output:                s.Output,
				ProviderOptions:       prepared.ProviderOptions,
				ActiveTools:           prepared.ActiveTools,
				PrepareStep:           s.PrepareStep,
				RepairToolCall:        s.ExperimentalRepairToolCall,
				ExperimentalContext:   prepared.ExperimentalContext,
				ExperimentalDownload:  prepared.ExperimentalDownload,
				ExperimentalTelemetry: prepared.ExperimentalTelemetry,
			}, nil
		}
	}

	// No prepareCall or it returned nil, use base args
	return &preparedCallArgs{
		Model:                 s.Model,
		Tools:                 s.Tools,
		ToolChoice:            s.ToolChoice,
		System:                s.Instructions,
		Prompt:                prompt,
		PromptMessages:        promptMessages,
		Messages:              messages,
		MaxOutputTokens:       s.MaxOutputTokens,
		Temperature:           s.Temperature,
		TopP:                  s.TopP,
		TopK:                  s.TopK,
		PresencePenalty:       s.PresencePenalty,
		FrequencyPenalty:      s.FrequencyPenalty,
		StopSequences:         s.StopSequences,
		Seed:                  s.Seed,
		MaxRetries:            s.MaxRetries,
		Headers:               s.Headers,
		StopWhen:              stopWhen,
		Output:                s.Output,
		ProviderOptions:       s.ProviderOptions,
		ActiveTools:           s.ActiveTools,
		PrepareStep:           s.PrepareStep,
		RepairToolCall:        s.ExperimentalRepairToolCall,
		ExperimentalContext:   s.ExperimentalContext,
		ExperimentalDownload:  s.ExperimentalDownload,
		ExperimentalTelemetry: s.ExperimentalTelemetry,
	}, nil
}

// mergeCallbacks merges two callbacks of the same type.
// If both are set, the settings callback is called first, then the method callback.
// If only one is set, it is returned as-is.
func mergeOnStartCallbacks(settings, method OnStartCallback) OnStartCallback {
	if method != nil && settings != nil {
		return func(event gt.OnStartEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

func mergeOnStepStartCallbacks(settings, method OnStepStartCallback) OnStepStartCallback {
	if method != nil && settings != nil {
		return func(event gt.OnStepStartEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

func mergeOnToolCallStartCallbacks(settings, method OnToolCallStartCallback) OnToolCallStartCallback {
	if method != nil && settings != nil {
		return func(event gt.OnToolCallStartEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

func mergeOnToolCallFinishCallbacks(settings, method OnToolCallFinishCallback) OnToolCallFinishCallback {
	if method != nil && settings != nil {
		return func(event gt.OnToolCallFinishEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

func mergeOnStepFinishCallbacks(settings, method OnStepFinishCallback) OnStepFinishCallback {
	if method != nil && settings != nil {
		return func(event gt.OnStepFinishEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

func mergeOnFinishCallbacks(settings, method OnFinishCallback) OnFinishCallback {
	if method != nil && settings != nil {
		return func(event gt.OnFinishEvent) {
			settings(event)
			method(event)
		}
	}
	if method != nil {
		return method
	}
	return settings
}

// toGenerateTextOptions converts preparedCallArgs into GenerateTextOptions.
func (args *preparedCallArgs) toGenerateTextOptions(ctx context.Context, timeout *gt.TimeoutConfiguration) gt.GenerateTextOptions {
	opts := gt.GenerateTextOptions{
		Ctx:                 ctx,
		Model:               args.Model,
		Tools:               args.Tools,
		ToolChoice:          args.ToolChoice,
		System:              args.System,
		MaxOutputTokens:     args.MaxOutputTokens,
		Temperature:         args.Temperature,
		TopP:                args.TopP,
		TopK:                args.TopK,
		PresencePenalty:     args.PresencePenalty,
		FrequencyPenalty:    args.FrequencyPenalty,
		StopSequences:       args.StopSequences,
		Seed:                args.Seed,
		MaxRetries:          args.MaxRetries,
		Headers:             args.Headers,
		Timeout:             timeout,
		StopWhen:            args.StopWhen,
		Output:              args.Output,
		ProviderOptions:     args.ProviderOptions,
		ActiveTools:         args.ActiveTools,
		PrepareStep:         args.PrepareStep,
		RepairToolCall:      args.RepairToolCall,
		ExperimentalContext: args.ExperimentalContext,
	}

	// Set the prompt: either a string prompt or messages
	if args.Prompt != "" {
		opts.Prompt = args.Prompt
	} else if len(args.PromptMessages) > 0 {
		opts.Messages = args.PromptMessages
	} else if len(args.Messages) > 0 {
		opts.Messages = args.Messages
	}

	return opts
}

// toStreamTextOptions converts preparedCallArgs into StreamTextOptions.
func (args *preparedCallArgs) toStreamTextOptions(ctx context.Context, timeout *gt.TimeoutConfiguration, transforms []gt.StreamTextTransform) gt.StreamTextOptions {
	opts := gt.StreamTextOptions{
		Ctx:                 ctx,
		Model:               args.Model,
		Tools:               args.Tools,
		ToolChoice:          args.ToolChoice,
		System:              args.System,
		MaxOutputTokens:     args.MaxOutputTokens,
		Temperature:         args.Temperature,
		TopP:                args.TopP,
		TopK:                args.TopK,
		PresencePenalty:     args.PresencePenalty,
		FrequencyPenalty:    args.FrequencyPenalty,
		StopSequences:       args.StopSequences,
		Seed:                args.Seed,
		MaxRetries:          args.MaxRetries,
		Headers:             args.Headers,
		Timeout:             timeout,
		StopWhen:            args.StopWhen,
		Output:              args.Output,
		ProviderOptions:     args.ProviderOptions,
		ActiveTools:         args.ActiveTools,
		PrepareStep:         args.PrepareStep,
		RepairToolCall:      args.RepairToolCall,
		ExperimentalContext: args.ExperimentalContext,
	}

	// Set the prompt: either a string prompt or messages
	if args.Prompt != "" {
		opts.Prompt = args.Prompt
	} else if len(args.PromptMessages) > 0 {
		opts.Messages = args.PromptMessages
	} else if len(args.Messages) > 0 {
		opts.Messages = args.Messages
	}

	// Apply the first transform (Go struct has a single Transform, not a slice)
	if len(transforms) > 0 {
		opts.Transform = transforms[0]
	}

	return opts
}

// Generate generates an output from the agent (non-streaming).
func (a *ToolLoopAgent) Generate(params AgentCallParameters) (*gt.GenerateTextResult, error) {
	prepared, err := a.prepareCall(params.Prompt, params.PromptMessages, params.Messages, params.Options)
	if err != nil {
		return nil, err
	}

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	opts := prepared.toGenerateTextOptions(ctx, params.Timeout)

	// Merge callbacks from settings and method params
	if cb := mergeOnStartCallbacks(a.settings.ExperimentalOnStart, params.ExperimentalOnStart); cb != nil {
		opts.OnStart = gt.GenerateTextOnStartCallback(cb)
	}
	if cb := mergeOnStepStartCallbacks(a.settings.ExperimentalOnStepStart, params.ExperimentalOnStepStart); cb != nil {
		opts.OnStepStart = gt.GenerateTextOnStepStartCallback(cb)
	}
	if cb := mergeOnToolCallStartCallbacks(a.settings.ExperimentalOnToolCallStart, params.ExperimentalOnToolCallStart); cb != nil {
		opts.OnToolCallStart = gt.GenerateTextOnToolCallStartCallback(cb)
	}
	if cb := mergeOnToolCallFinishCallbacks(a.settings.ExperimentalOnToolCallFinish, params.ExperimentalOnToolCallFinish); cb != nil {
		opts.OnToolCallFinish = gt.GenerateTextOnToolCallFinishCallback(cb)
	}
	if cb := mergeOnStepFinishCallbacks(a.settings.OnStepFinish, params.OnStepFinish); cb != nil {
		opts.OnStepFinish = gt.GenerateTextOnStepFinishCallback(cb)
	}
	if cb := mergeOnFinishCallbacks(a.settings.OnFinish, params.OnFinish); cb != nil {
		opts.OnFinish = gt.GenerateTextOnFinishCallback(cb)
	}

	return gt.GenerateText(opts)
}

// Stream streams an output from the agent (streaming).
func (a *ToolLoopAgent) Stream(params AgentStreamParameters) (*gt.StreamTextResult, error) {
	prepared, err := a.prepareCall(params.Prompt, params.PromptMessages, params.Messages, params.Options)
	if err != nil {
		return nil, err
	}

	ctx := params.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	opts := prepared.toStreamTextOptions(ctx, params.Timeout, params.ExperimentalTransform)

	// Merge callbacks from settings and method params
	if cb := mergeOnStartCallbacks(a.settings.ExperimentalOnStart, params.ExperimentalOnStart); cb != nil {
		opts.OnStart = gt.StreamTextOnStartCallback(cb)
	}
	if cb := mergeOnStepStartCallbacks(a.settings.ExperimentalOnStepStart, params.ExperimentalOnStepStart); cb != nil {
		opts.OnStepStart = gt.StreamTextOnStepStartCallback(cb)
	}
	if cb := mergeOnToolCallStartCallbacks(a.settings.ExperimentalOnToolCallStart, params.ExperimentalOnToolCallStart); cb != nil {
		opts.OnToolCallStart = gt.StreamTextOnToolCallStartCallback(cb)
	}
	if cb := mergeOnToolCallFinishCallbacks(a.settings.ExperimentalOnToolCallFinish, params.ExperimentalOnToolCallFinish); cb != nil {
		opts.OnToolCallFinish = gt.StreamTextOnToolCallFinishCallback(cb)
	}
	if cb := mergeOnStepFinishCallbacks(a.settings.OnStepFinish, params.OnStepFinish); cb != nil {
		opts.OnStepFinish = gt.StreamTextOnStepFinishCallback(cb)
	}
	if cb := mergeOnFinishCallbacks(a.settings.OnFinish, params.OnFinish); cb != nil {
		opts.OnFinish = gt.StreamTextOnFinishCallback(cb)
	}

	return gt.StreamText(opts)
}
