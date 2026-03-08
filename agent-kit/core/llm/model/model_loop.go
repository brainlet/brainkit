// Ported from: packages/core/src/llm/model/model.loop.ts
package model

import (
	"fmt"
	"strconv"
	"time"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ---------------------------------------------------------------------------
// Stub types for unported packages referenced by MastraLLMVNext
// ---------------------------------------------------------------------------

// ModelManagerModelConfig is a stub for stream/types.ModelManagerModelConfig.
// STUB REASON: The real stream.ModelManagerModelConfig has extra fields (MaxRetries int,
// ID string, Headers map[string]string) beyond just Model. This stub only captures
// the Model field. Structural mismatch prevents direct replacement.
type ModelManagerModelConfig struct {
	// Model is the resolved language model.
	Model MastraLanguageModel
}

// MastraModelOutput is a stub for stream/base/output.MastraModelOutput.
// STUB REASON: The real stream/base/output.MastraModelOutput is a complex struct with
// sync.Mutex, channels, callbacks, and 30+ fields. Using `= any` here as a simplified
// placeholder. Cannot replace without propagating full struct dependencies.
type MastraModelOutput = any

// OutputSchema is a stub for stream/base/schema.OutputSchema.
// STUB REASON: The real stream/base/schema.OutputSchema is also `= any` (represents
// Zod-like schema which has no Go equivalent). Importing would add a dependency for
// no type-safety gain. Keep local alias.
type OutputSchema = any

// Schema is a stub for AI SDK Schema type.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type Schema = any

// ModelMessage is a stub for AI SDK ModelMessage type.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type ModelMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

// SpanType is re-exported from observability/types.
type SpanType = obstypes.SpanType

// SpanTypeModelGeneration is re-exported from observability/types.
const SpanTypeModelGeneration = obstypes.SpanTypeModelGeneration

// ---------------------------------------------------------------------------
// MastraLLMVNext
// ---------------------------------------------------------------------------

// MastraLLMVNext is the V-Next (modern) LLM wrapper that supports the model loop.
// It extends MastraBase and wraps one or more ModelManagerModelConfig entries for
// retry/fallback behavior.
//
// TS: export class MastraLLMVNext extends MastraBase { ... }
type MastraLLMVNext struct {
	*agentkit.MastraBase

	models     []ModelManagerModelConfig
	mastra     MastraRef
	options    *MastraModelOptions
	firstModel ModelManagerModelConfig
}

// MastraLLMVNextConfig holds the constructor arguments for MastraLLMVNext.
type MastraLLMVNextConfig struct {
	Mastra  MastraRef
	Models  []ModelManagerModelConfig
	Options *MastraModelOptions
}

// NewMastraLLMVNext creates a new MastraLLMVNext instance.
// Returns an error if the models list is empty.
func NewMastraLLMVNext(cfg MastraLLMVNextConfig) (*MastraLLMVNext, error) {
	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Name: "aisdk",
	})

	llm := &MastraLLMVNext{
		MastraBase: base,
		options:    cfg.Options,
	}

	if cfg.Mastra != nil {
		llm.mastra = cfg.Mastra
		if cfg.Mastra.GetLogger() != nil {
			llm.SetLogger(cfg.Mastra.GetLogger())
		}
	}

	if len(cfg.Models) == 0 {
		err := mastraerror.NewMastraBaseError(mastraerror.ErrorDefinition{
			ID:       "LLM_LOOP_MODELS_EMPTY",
			Domain:   mastraerror.ErrorDomainLLM,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "models list must not be empty",
		})
		llm.Logger().Error(err.Error())
		return nil, err
	}

	llm.models = cfg.Models
	llm.firstModel = cfg.Models[0]

	return llm, nil
}

// RegisterPrimitives registers Mastra primitives (logger, etc.) on this LLM.
// TS: __registerPrimitives(p: MastraPrimitives)
func (l *MastraLLMVNext) RegisterPrimitives(p MastraPrimitives) {
	if p.Logger != nil {
		l.SetLogger(p.Logger)
	}
}

// RegisterMastra registers the Mastra instance on this LLM.
// TS: __registerMastra(p: Mastra)
func (l *MastraLLMVNext) RegisterMastra(m MastraRef) {
	l.mastra = m
}

// GetProvider returns the provider name of the first model.
func (l *MastraLLMVNext) GetProvider() string {
	return l.firstModel.Model.Provider()
}

// GetModelID returns the model ID of the first model.
func (l *MastraLLMVNext) GetModelID() string {
	return l.firstModel.Model.ModelID()
}

// GetModel returns the first model's language model instance.
func (l *MastraLLMVNext) GetModel() MastraLanguageModel {
	return l.firstModel.Model
}

// ConvertToMessages converts string or string slice messages to ModelMessage slice.
// TS: convertToMessages(messages: string | string[] | ModelMessage[]): ModelMessage[]
func (l *MastraLLMVNext) ConvertToMessages(messages any) []ModelMessage {
	switch m := messages.(type) {
	case string:
		return []ModelMessage{{Role: "user", Content: m}}
	case []string:
		result := make([]ModelMessage, len(m))
		for i, s := range m {
			result[i] = ModelMessage{Role: "user", Content: s}
		}
		return result
	case []ModelMessage:
		out := make([]ModelMessage, len(m))
		for i, msg := range m {
			if s, ok := msg.Content.(string); ok && msg.Role == "" {
				out[i] = ModelMessage{Role: "user", Content: s}
			} else {
				out[i] = msg
			}
		}
		return out
	default:
		return []ModelMessage{{Role: "user", Content: fmt.Sprintf("%v", messages)}}
	}
}

// Stream performs a streaming text generation call via the model loop.
// This is the primary entry point for streaming in the V-Next architecture.
//
// TS: stream<Tools extends ToolSet, OUTPUT = undefined>(args: ModelLoopStreamArgs<Tools, OUTPUT>): MastraModelOutput<OUTPUT>
//
// TODO: implement fully when stream, loop, and observability packages are ported.
// Currently returns an error indicating the method is not yet implemented.
func (l *MastraLLMVNext) Stream(args ModelLoopStreamArgs) (MastraModelOutput, error) {
	// Resolve stop condition
	// TS: let stopWhenToUse;
	// if (maxSteps && typeof maxSteps === 'number') { stopWhenToUse = stepCountIs(maxSteps); }
	// else { stopWhenToUse = stopWhen; }
	_ = args.StopWhen
	_ = args.MaxSteps

	firstModel := l.firstModel.Model
	l.Logger().Debug("[LLM] - Streaming text", map[string]any{
		"runId":      args.RunID,
		"threadId":   args.ThreadID,
		"resourceId": args.ResourceID,
		"tools":      toolNames(args.Tools),
	})

	// Create model span for observability
	// TS: const modelSpan = observabilityContext.tracingContext.currentSpan?.createChildSpan({...})
	// TODO: implement span creation when observability package is ported
	_ = firstModel

	// Build loop options
	// TS: const loopOptions: LoopOptions<Tools, OUTPUT> = { ... }
	loopOpts := LoopOptions{
		ResumeContext:            args.ResumeContext,
		RunID:                    args.RunID,
		ToolCallID:               args.ToolCallID,
		MessageList:              args.MessageList,
		Models:                   l.models,
		Logger:                   l.Logger(),
		Tools:                    args.Tools,
		StopWhen:                 args.StopWhen,
		ToolChoice:               args.ToolChoice,
		ModelSettings:            args.ModelSettings,
		ProviderOptions:          args.ProviderOptions,
		Internal:                 args.Internal,
		StructuredOutput:         args.StructuredOutput,
		InputProcessors:          args.InputProcessors,
		OutputProcessors:         args.OutputProcessors,
		ReturnScorerData:         args.ReturnScorerData,
		RequireToolApproval:      args.RequireToolApproval,
		ToolCallConcurrency:      args.ToolCallConcurrency,
		AgentID:                  args.AgentID,
		AgentName:                args.AgentName,
		RequestContext:           args.RequestCtx,
		MethodType:               args.MethodType,
		IncludeRawChunks:         args.IncludeRawChunks,
		AutoResumeSuspendedTools: args.AutoResumeSuspendedTools,
		MaxProcessorRetries:      args.MaxProcessorRetries,
		ProcessorStates:          args.ProcessorStates,
		ActiveTools:              args.ActiveTools,
		IsTaskComplete:           args.IsTaskComplete,
		OnIterationComplete:      args.OnIterationComplete,
		Workspace:                args.Workspace,
		MaxSteps:                 args.MaxSteps,
	}

	// Wrap onStepFinish and onFinish callbacks with error handling and rate-limit detection
	// TS: options: { onStepFinish: async props => { ... }, onFinish: async props => { ... } }
	if args.Options != nil {
		loopOpts.Options = &ModelLoopStreamOptions{
			// TS: onStepFinish: async props => { ... }
			OnStepFinish: func(props any) error {
				// Call user's onStepFinish callback with error wrapping
				if args.Options.OnStepFinish != nil {
					if err := args.Options.OnStepFinish(props); err != nil {
						merr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
							ID:       "LLM_STREAM_ON_STEP_FINISH_CALLBACK_EXECUTION_FAILED",
							Domain:   mastraerror.ErrorDomainLLM,
							Category: mastraerror.ErrorCategoryUser,
							Details: map[string]any{
								"runId":      orDefault(args.RunID, "unknown"),
								"threadId":   orDefault(args.ThreadID, "unknown"),
								"resourceId": orDefault(args.ResourceID, "unknown"),
							},
						}, err)
						// TS: modelSpanTracker?.reportGenerationError({ error: mastraError });
						l.Logger().Error(merr.Error())
						return merr
					}
				}

				l.Logger().Debug("[LLM] - Stream Step Change:", map[string]any{
					"runId": args.RunID,
				})

				// Rate limit detection
				// TS: const remainingTokens = parseInt(props?.response?.headers?.['x-ratelimit-remaining-tokens'] ?? '', 10);
				if propsMap, ok := props.(map[string]any); ok {
					if resp, ok := propsMap["response"].(map[string]any); ok {
						if headers, ok := resp["headers"].(map[string]string); ok {
							if remaining, ok := headers["x-ratelimit-remaining-tokens"]; ok {
								if tokens, parseErr := strconv.Atoi(remaining); parseErr == nil && tokens > 0 && tokens < 2000 {
									l.Logger().Warn("Rate limit approaching, waiting 10 seconds", map[string]any{
										"runId": args.RunID,
									})
									time.Sleep(10 * time.Second)
								}
							}
						}
					}
				}

				return nil
			},
			// TS: onFinish: async props => { ... }
			OnFinish: func(props any) error {
				// End the model generation span BEFORE calling the user's onFinish callback
				// TS: modelSpanTracker?.endGeneration({ output: { ... }, attributes: { ... }, usage, providerMetadata });
				// TODO: implement when observability package is ported

				// Call user's onFinish callback with error wrapping
				if args.Options.OnFinish != nil {
					if err := args.Options.OnFinish(props); err != nil {
						merr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
							ID:       "LLM_STREAM_ON_FINISH_CALLBACK_EXECUTION_FAILED",
							Domain:   mastraerror.ErrorDomainLLM,
							Category: mastraerror.ErrorCategoryUser,
							Details: map[string]any{
								"runId":      orDefault(args.RunID, "unknown"),
								"threadId":   orDefault(args.ThreadID, "unknown"),
								"resourceId": orDefault(args.ResourceID, "unknown"),
							},
						}, err)
						// TS: modelSpanTracker?.reportGenerationError({ error: mastraError });
						l.Logger().Error(merr.Error())
						return merr
					}
				}

				l.Logger().Debug("[LLM] - Stream Finished:", map[string]any{
					"runId":      args.RunID,
					"threadId":   args.ThreadID,
					"resourceId": args.ResourceID,
				})

				return nil
			},
		}
	}

	// TS: return loop(loopOptions);
	// TODO: call the loop function when the loop package is ported.
	// The loop() function from ../../loop orchestrates multi-step LLM calls with
	// tool execution, retry/fallback across models, and streaming output composition.
	// Until the loop package is ported, this method cannot perform real generation.
	//
	// TS catch block wraps errors as:
	//   MastraError({ id: 'LLM_STREAM_TEXT_AI_SDK_EXECUTION_FAILED', domain: ErrorDomain.LLM, category: ErrorCategory.THIRD_PARTY })
	//   modelSpanTracker?.reportGenerationError({ error: mastraError });
	_ = loopOpts
	return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
		ID:       "LLM_STREAM_TEXT_AI_SDK_NOT_PORTED",
		Domain:   mastraerror.ErrorDomainLLM,
		Category: mastraerror.ErrorCategorySystem,
		Text:     "MastraLLMVNext.Stream requires the loop package which is not yet ported to Go",
		Details: map[string]any{
			"modelId":       firstModel.ModelID(),
			"modelProvider": firstModel.Provider(),
			"runId":         orDefault(args.RunID, "unknown"),
			"threadId":      orDefault(args.ThreadID, "unknown"),
			"resourceId":    orDefault(args.ResourceID, "unknown"),
		},
	})
}

// toolNames extracts tool names from a ToolSet for logging.
func toolNames(tools ToolSet) []string {
	if tools == nil {
		return nil
	}
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	return names
}

// ---------------------------------------------------------------------------
// Logger helper interface assertion
// ---------------------------------------------------------------------------

// Ensure MastraLLMVNext's Logger method is accessible.
// MastraBase already provides Logger() via the embedded struct.
var _ interface {
	Logger() logger.IMastraLogger
} = (*MastraLLMVNext)(nil)
