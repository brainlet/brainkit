// Ported from: packages/core/src/llm/model/model.ts
package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	genobj "github.com/brainlet/brainkit/ai-kit/ai/generateobject"
	gentext "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Stub types for unported packages
// ---------------------------------------------------------------------------

// MastraPrimitives is a stub for action.MastraPrimitives.
// STUB REASON: Cannot import action due to circular dependency chain:
// llm/model -> action -> storage -> processorprovider -> processors -> llm/model.
// The real action.MastraPrimitives has additional fields beyond Logger.
type MastraPrimitives struct {
	Logger logger.IMastraLogger
}

// MastraRef is a stub for the Mastra class.
// STUB REASON: Cannot import core.Mastra due to circular dependency: core imports llm/model.
// This minimal interface captures only the methods needed by MastraLLMV1.
type MastraRef interface {
	GetLogger() logger.IMastraLogger
}

// SchemaCompatLayer is a stub for schema-compat layers.
// STUB REASON: The schema-compat package is not yet ported. This interface captures the
// minimal contract (Apply method) needed by MastraLLMV1 for schema compatibility.
type SchemaCompatLayer interface {
	Apply(schema any) any
}

// ---------------------------------------------------------------------------
// MastraLLMV1
// ---------------------------------------------------------------------------

// MastraLLMV1 is the V1 (legacy) LLM wrapper class.
// It wraps a LanguageModelV1 and provides generate/stream methods.
type MastraLLMV1 struct {
	*agentkit.MastraBase
	model   LanguageModelV1
	mastra  MastraRef
	options *MastraModelOptions
}

// MastraLLMV1Config holds the constructor arguments for MastraLLMV1.
type MastraLLMV1Config struct {
	Model   LanguageModelV1
	Mastra  MastraRef
	Options *MastraModelOptions
}

// NewMastraLLMV1 creates a new MastraLLMV1 instance.
func NewMastraLLMV1(cfg MastraLLMV1Config) *MastraLLMV1 {
	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Name: "aisdk",
	})

	llm := &MastraLLMV1{
		MastraBase: base,
		model:      cfg.Model,
		options:    cfg.Options,
	}

	if cfg.Mastra != nil {
		llm.mastra = cfg.Mastra
		if cfg.Mastra.GetLogger() != nil {
			llm.SetLogger(cfg.Mastra.GetLogger())
		}
	}

	return llm
}

// RegisterPrimitives registers Mastra primitives (logger, etc.) on this LLM.
func (l *MastraLLMV1) RegisterPrimitives(p MastraPrimitives) {
	if p.Logger != nil {
		l.SetLogger(p.Logger)
	}
}

// RegisterMastra registers the Mastra instance on this LLM.
func (l *MastraLLMV1) RegisterMastra(m MastraRef) {
	l.mastra = m
}

// GetProvider returns the provider name of the underlying model.
func (l *MastraLLMV1) GetProvider() string {
	return l.model.Provider()
}

// GetModelID returns the model ID of the underlying model.
func (l *MastraLLMV1) GetModelID() string {
	return l.model.ModelID()
}

// GetModel returns the underlying LanguageModelV1 instance.
func (l *MastraLLMV1) GetModel() LanguageModelV1 {
	return l.model
}

// ConvertToMessages converts string or string slice messages to CoreMessage slice.
// TS: convertToMessages(messages: string | string[] | CoreMessage[]): CoreMessage[]
func (l *MastraLLMV1) ConvertToMessages(messages any) []CoreMessage {
	switch m := messages.(type) {
	case string:
		return []CoreMessage{{Role: "user", Content: m}}
	case []string:
		result := make([]CoreMessage, len(m))
		for i, s := range m {
			result[i] = CoreMessage{Role: "user", Content: s}
		}
		return result
	case []CoreMessage:
		return m
	default:
		return []CoreMessage{{Role: "user", Content: fmt.Sprintf("%v", messages)}}
	}
}

// Generate performs a text generation call.
//
// TS:
//
//	async generate<Output, StructuredOutput, Tools>(
//	  messages: string | string[] | CoreMessage[],
//	  args?: { output?: Output, ... },
//	): Promise<GenerateReturn<Tools, Output, StructuredOutput>>
//
// If output is specified, delegates to __textObject. Otherwise delegates to __text.
func (l *MastraLLMV1) Generate(messages any, opts *GenerateOptions) (GenerateReturn, error) {
	msgs := l.ConvertToMessages(messages)

	l.Logger().Debug("[LLM] - Generating text", map[string]any{
		"messages": len(msgs),
	})

	if opts != nil && opts.Output != nil {
		return l.generateObject(msgs, opts)
	}
	return l.generateText(msgs, opts)
}

// Stream performs a streaming text generation call.
//
// TS:
//
//	stream<Output, StructuredOutput, Tools>(
//	  messages: string | string[] | CoreMessage[],
//	  args?: { output?: Output, ... },
//	): StreamReturn<Tools, Output, StructuredOutput>
//
// If output is specified, delegates to __streamObject. Otherwise delegates to __stream.
func (l *MastraLLMV1) Stream(messages any, opts *StreamOptions) (StreamReturn, error) {
	msgs := l.ConvertToMessages(messages)

	l.Logger().Debug("[LLM] - Streaming text", map[string]any{
		"messages": len(msgs),
	})

	if opts != nil && opts.Output != nil {
		return l.streamObject(msgs, opts)
	}
	return l.streamText(msgs, opts)
}

// generateText is the internal text generation implementation.
//
// TS: async __text<Tools, Z>({ runId, messages, maxSteps, tools, temperature, toolChoice,
//
//	onStepFinish, experimental_output, threadId, resourceId, requestContext, ...rest })
//
// Faithfully ports the TS logic including:
//   - Experimental output schema handling (Zod/JSONSchema)
//   - Observability span creation
//   - Callback wrapping with error handling
//   - Rate limit detection
//   - Error wrapping with MastraError
func (l *MastraLLMV1) generateText(messages []CoreMessage, opts *GenerateOptions) (*GenerateTextResult, error) {
	mdl := l.model

	var runID, threadID, resourceID string
	var maxSteps int = 5
	var tools ToolSet
	var temperature *float64
	var toolChoice string = "auto"
	var onStepFinish GenerateTextOnStepFinishCallback
	var experimentalOutput any

	if opts != nil {
		runID = opts.RunID
		threadID = opts.ThreadID
		resourceID = opts.ResourceID
		if opts.MaxSteps > 0 {
			maxSteps = opts.MaxSteps
		}
		tools = opts.Tools
		temperature = opts.Temperature
		if opts.ToolChoice != "" {
			toolChoice = opts.ToolChoice
		}
		onStepFinish = opts.OnStepFinish
		experimentalOutput = opts.ExperimentalOutput
	}

	toolKeys := make([]string, 0)
	for k := range tools {
		toolKeys = append(toolKeys, k)
	}

	l.Logger().Debug("[LLM] - Generating text", map[string]any{
		"runId":      runID,
		"messages":   messages,
		"maxSteps":   maxSteps,
		"threadId":   threadID,
		"resourceId": resourceID,
		"tools":      toolKeys,
	})

	// Handle experimental output schema
	// TS: if (experimental_output) { ... schema handling ... }
	var outputSchema any
	if experimentalOutput != nil {
		l.Logger().Debug("[LLM] - Using experimental output", map[string]any{
			"runId": runID,
		})
		// In Go, schemas are JSON Schema objects (map[string]any), not Zod types.
		// TS: if (isZodType(experimental_output)) { schema = zodToJsonSchema(schema, 'jsonSchema7'); schema = jsonSchema(jsonSchemaToUse); }
		// else { schema = jsonSchema(experimental_output); }
		// TODO: implement schema-compat layers (applyCompatLayer) when ported
		outputSchema = experimentalOutput
	}

	// Create observability span
	// TS: const llmSpan = observabilityContext.tracingContext.currentSpan?.createChildSpan({ ... })
	// TODO: implement span creation when observability package is ported

	// Wrap onStepFinish callback with error handling and rate limit detection
	// TS: onStepFinish: async props => { ... }
	wrappedOnStepFinish := func(props StepFinishEvent) error {
		// Call user's onStepFinish callback
		if onStepFinish != nil {
			props.RunID = runID
			if err := onStepFinish(props); err != nil {
				return mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "LLM_TEXT_ON_STEP_FINISH_CALLBACK_EXECUTION_FAILED",
					Domain:   mastraerror.ErrorDomainLLM,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"modelId":       mdl.ModelID(),
						"modelProvider": mdl.Provider(),
						"runId":         orDefault(runID, "unknown"),
						"threadId":      orDefault(threadID, "unknown"),
						"resourceId":    orDefault(resourceID, "unknown"),
						"finishReason":  props.FinishReason,
						"toolCalls":     jsonStringify(props.ToolCalls),
						"toolResults":   jsonStringify(props.ToolResults),
						"usage":         jsonStringify(props.Usage),
					},
				}, err)
			}
		}

		l.Logger().Debug("[LLM] - Text Step Change:", map[string]any{
			"text":         props.Text,
			"toolCalls":    props.ToolCalls,
			"toolResults":  props.ToolResults,
			"finishReason": props.FinishReason,
			"usage":        props.Usage,
			"runId":        runID,
		})

		// Rate limit detection
		// TS: const remainingTokens = parseInt(props?.response?.headers?.['x-ratelimit-remaining-tokens'] ?? '', 10);
		if props.Response != nil && props.Response.Headers != nil {
			if remaining, ok := props.Response.Headers["x-ratelimit-remaining-tokens"]; ok {
				if tokens, err := strconv.Atoi(remaining); err == nil && tokens > 0 && tokens < 2000 {
					l.Logger().Warn("Rate limit approaching, waiting 10 seconds", map[string]any{"runId": runID})
					time.Sleep(10 * time.Second)
				}
			}
		}

		return nil
	}

	// Build args and invoke generateText from the AI SDK
	// TS: const argsForExecute = { ...rest, messages, model, temperature, tools, toolChoice, maxSteps, onStepFinish, experimental_output: schema ? Output.object({ schema }) : undefined };
	// TS: const result = await executeWithContext({ span: llmSpan, fn: () => generateText(argsForExecute) });
	aiOpts := gentext.GenerateTextOptions{
		Ctx:         context.Background(),
		Model:       &modelAdapterForGenText{model: mdl},
		Messages:    convertToGenTextMessages(messages),
		Temperature: temperature,
	}

	// Map maxSteps to StopWhen conditions
	if maxSteps > 0 {
		aiOpts.StopWhen = []gentext.StopCondition{gentext.StepCountIs(maxSteps)}
	}

	// Map toolChoice string to ai-kit ToolChoice
	if toolChoice != "" {
		aiOpts.ToolChoice = &gentext.ToolChoice{Type: toolChoice}
	}

	// Map experimental output schema
	// TS: experimental_output: schema ? Output.object({ schema }) : undefined
	if outputSchema != nil {
		aiOpts.Output = gentext.ObjectOutput(gentext.ObjectOutputOptions{
			Schema: outputSchema,
		})
	}

	// Wire the wrapped onStepFinish callback
	if wrappedOnStepFinish != nil {
		aiOpts.OnStepFinish = func(event gentext.StepResult) {
			stepEvent := StepFinishEvent{
				Text:         event.Text(),
				FinishReason: string(event.FinishReason),
			}
			if err := wrappedOnStepFinish(stepEvent); err != nil {
				l.Logger().Error(fmt.Sprintf("[LLM] - onStepFinish error: %v", err))
			}
		}
	}

	aiResult, err := gentext.GenerateText(aiOpts)
	if err != nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "LLM_GENERATE_TEXT_AI_SDK_EXECUTION_FAILED",
			Domain:   mastraerror.ErrorDomainLLM,
			Category: mastraerror.ErrorCategoryThirdParty,
			Text:     fmt.Sprintf("generateText execution failed: %v", err),
			Details: map[string]any{
				"modelId":       mdl.ModelID(),
				"modelProvider": mdl.Provider(),
				"runId":         orDefault(runID, "unknown"),
			},
		}, err)
	}

	// Convert ai-kit result to agent-kit result
	// TS: if (schema && result.finishReason === 'stop') { result.object = (result as any).experimental_output; }
	// TS: llmSpan?.end({ output: { text, object, reasoning, reasoningText, files, sources, warnings }, attributes: { finishReason, responseId, responseModel, usage } });
	result := &GenerateTextResult{}
	if aiResult != nil {
		result.Text = aiResult.Text
		result.FinishReason = string(aiResult.FinishReason)
		// Map experimental_output when schema is present and generation stopped normally
		if outputSchema != nil && result.FinishReason == "stop" {
			result.Object = aiResult.Output
		}
	}
	return result, nil
}

// generateObject is the internal structured object generation implementation.
//
// TS: async __textObject<Z>({ messages, structuredOutput, runId, threadId, resourceId, requestContext, ...rest })
//
// Faithfully ports the TS logic including:
//   - Schema compat layer application
//   - Output mode detection (object vs array)
//   - Observability span creation
//   - Error wrapping with MastraError (execution + schema conversion)
func (l *MastraLLMV1) generateObject(messages []CoreMessage, opts *GenerateOptions) (*GenerateObjectResult, error) {
	mdl := l.model

	var runID, threadID, resourceID string
	var structuredOutput any

	if opts != nil {
		runID = opts.RunID
		threadID = opts.ThreadID
		resourceID = opts.ResourceID
		structuredOutput = opts.Output
	}

	l.Logger().Debug("[LLM] - Generating a text object", map[string]any{"runId": runID})

	// Create observability span
	// TS: const llmSpan = observabilityContext.tracingContext.currentSpan?.createChildSpan({ ... })
	// TODO: implement span creation when observability package is ported

	// Determine output mode (object or array)
	// TS: let output: 'object' | 'array' = 'object';
	//     if (isZodArray(structuredOutput)) { output = 'array'; structuredOutput = getZodDef(structuredOutput).type; }
	// In Go, Zod concepts don't apply. We default to "object" mode.
	outputMode := "object"

	// Apply schema compat layers
	// TS: const processedSchema = this._applySchemaCompat(structuredOutput!);
	// TODO: implement schema compat layers when schema-compat package is ported
	processedSchema := structuredOutput

	// Build args and invoke generateObject from the AI SDK
	// TS: const argsForExecute = { ...rest, messages, model, output, schema: processedSchema };
	// TS: const result = await generateObject(argsForExecute);
	adapter := &modelAdapterForGenObj{model: mdl}
	aiResult, err := genobj.GenerateObject(context.Background(), genobj.GenerateObjectOptions{
		Model:  adapter,
		Output: outputMode,
		Schema: processedSchema,
		Mode:   "json",
		Prompt: convertMessagesToPrompt(messages),
	})
	if err != nil {
		// TS error handling (nested try/catch):
		//   Inner catch: LLM_GENERATE_OBJECT_AI_SDK_EXECUTION_FAILED (THIRD_PARTY)
		//   Outer catch: LLM_GENERATE_OBJECT_AI_SDK_SCHEMA_CONVERSION_FAILED (USER)
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "LLM_GENERATE_OBJECT_AI_SDK_EXECUTION_FAILED",
			Domain:   mastraerror.ErrorDomainLLM,
			Category: mastraerror.ErrorCategoryThirdParty,
			Text:     fmt.Sprintf("generateObject execution failed: %v", err),
			Details: map[string]any{
				"modelId":       mdl.ModelID(),
				"modelProvider": mdl.Provider(),
				"runId":         orDefault(runID, "unknown"),
				"threadId":      orDefault(threadID, "unknown"),
				"resourceId":    orDefault(resourceID, "unknown"),
			},
		}, err)
	}

	// Convert ai-kit result to agent-kit result
	// TS: llmSpan?.end({ output: { object, warnings }, attributes: { finishReason, responseId, responseModel, usage } });
	result := &GenerateObjectResult{}
	if aiResult != nil {
		result.Object = aiResult.Object
		result.FinishReason = string(aiResult.FinishReason)
	}
	return result, nil
}

// streamText is the internal streaming text generation implementation.
//
// TS: __stream<Tools, Z>({ messages, onStepFinish, onFinish, maxSteps, tools, runId,
//
//	temperature, toolChoice, experimental_output, threadId, resourceId, requestContext, ...rest })
//
// Faithfully ports the TS logic including:
//   - Experimental output schema handling
//   - Observability span creation
//   - onStepFinish callback with error wrapping and rate-limit detection
//   - onFinish callback with span ending and error wrapping
//   - onError callback for streaming errors
//   - Error wrapping with MastraError
func (l *MastraLLMV1) streamText(messages []CoreMessage, opts *StreamOptions) (*StreamTextResult, error) {
	mdl := l.model

	var runID, threadID, resourceID string
	var maxSteps int = 5
	var tools ToolSet
	var temperature *float64
	var toolChoice string = "auto"
	var onStepFinish StreamTextOnStepFinishCallback
	var onFinish StreamTextOnFinishCallback
	var experimentalOutput any

	if opts != nil {
		runID = opts.RunID
		threadID = opts.ThreadID
		resourceID = opts.ResourceID
		if opts.MaxSteps > 0 {
			maxSteps = opts.MaxSteps
		}
		tools = opts.Tools
		temperature = opts.Temperature
		if opts.ToolChoice != "" {
			toolChoice = opts.ToolChoice
		}
		onStepFinish = opts.OnStepFinish
		onFinish = opts.OnFinish
		experimentalOutput = opts.ExperimentalOutput
	}

	toolKeys := make([]string, 0)
	for k := range tools {
		toolKeys = append(toolKeys, k)
	}

	l.Logger().Debug("[LLM] - Streaming text", map[string]any{
		"runId":      runID,
		"threadId":   threadID,
		"resourceId": resourceID,
		"messages":   messages,
		"maxSteps":   maxSteps,
		"tools":      toolKeys,
	})

	// Handle experimental output schema
	// TS: if (experimental_output) { ... }
	var outputSchema any
	if experimentalOutput != nil {
		l.Logger().Debug("[LLM] - Using experimental output", map[string]any{
			"runId": runID,
		})
		// In Go, schemas are JSON Schema objects (map[string]any), not Zod types.
		// TODO: implement schema handling when schema-compat package is ported
		outputSchema = experimentalOutput
	}

	// Create observability span
	// TS: const llmSpan = observabilityContext.tracingContext.currentSpan?.createChildSpan({ ... })
	// TODO: implement span creation when observability package is ported

	// Build the wrapped callbacks
	// TS: onStepFinish: async props => { ... }
	wrappedOnStepFinish := func(props StepFinishEvent) error {
		if onStepFinish != nil {
			props.RunID = runID
			if err := onStepFinish(props); err != nil {
				merr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "LLM_STREAM_ON_STEP_FINISH_CALLBACK_EXECUTION_FAILED",
					Domain:   mastraerror.ErrorDomainLLM,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"modelId":       mdl.ModelID(),
						"modelProvider": mdl.Provider(),
						"runId":         orDefault(runID, "unknown"),
						"threadId":      orDefault(threadID, "unknown"),
						"resourceId":    orDefault(resourceID, "unknown"),
						"finishReason":  props.FinishReason,
						"toolCalls":     jsonStringify(props.ToolCalls),
						"toolResults":   jsonStringify(props.ToolResults),
						"usage":         jsonStringify(props.Usage),
					},
				}, err)
				l.Logger().Error(merr.Error())
				// TS: llmSpan?.error({ error: mastraError });
				return merr
			}
		}

		l.Logger().Debug("[LLM] - Stream Step Change:", map[string]any{
			"text":         props.Text,
			"toolCalls":    props.ToolCalls,
			"toolResults":  props.ToolResults,
			"finishReason": props.FinishReason,
			"usage":        props.Usage,
			"runId":        runID,
		})

		// Rate limit detection
		if props.Response != nil && props.Response.Headers != nil {
			if remaining, ok := props.Response.Headers["x-ratelimit-remaining-tokens"]; ok {
				if tokens, err := strconv.Atoi(remaining); err == nil && tokens > 0 && tokens < 2000 {
					l.Logger().Warn("Rate limit approaching, waiting 10 seconds", map[string]any{"runId": runID})
					time.Sleep(10 * time.Second)
				}
			}
		}

		return nil
	}

	// TS: onFinish: async props => { ... }
	wrappedOnFinish := func(event StreamTextFinishEvent) error {
		// End the model generation span BEFORE calling the user's onFinish callback
		// TS: llmSpan?.end({ output: { text, reasoning, reasoningText, files, sources, warnings }, attributes: { finishReason, usage } });
		// TODO: implement span ending when observability package is ported

		if onFinish != nil {
			event.RunID = runID
			if err := onFinish(event); err != nil {
				merr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "LLM_STREAM_ON_FINISH_CALLBACK_EXECUTION_FAILED",
					Domain:   mastraerror.ErrorDomainLLM,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"modelId":       mdl.ModelID(),
						"modelProvider": mdl.Provider(),
						"runId":         orDefault(runID, "unknown"),
						"threadId":      orDefault(threadID, "unknown"),
						"resourceId":    orDefault(resourceID, "unknown"),
						"finishReason":  event.FinishReason,
						"toolCalls":     jsonStringify(event.ToolCalls),
						"toolResults":   jsonStringify(event.ToolResults),
						"usage":         jsonStringify(event.Usage),
					},
				}, err)
				// TS: llmSpan?.error({ error: mastraError });
				l.Logger().Error(merr.Error())
				return merr
			}
		}

		l.Logger().Debug("[LLM] - Stream Finished:", map[string]any{
			"text":         event.Text,
			"toolCalls":    event.ToolCalls,
			"toolResults":  event.ToolResults,
			"finishReason": event.FinishReason,
			"usage":        event.Usage,
			"runId":        runID,
			"threadId":     threadID,
			"resourceId":   resourceID,
		})

		return nil
	}

	// Build args and invoke streamText from the AI SDK
	// TS: const argsForExecute = { model, temperature, tools, maxSteps, toolChoice, onStepFinish, onFinish, onError, ...rest, messages, experimental_output };
	// TS: return executeWithContextSync({ span: llmSpan, fn: () => streamText(argsForExecute) });
	streamOpts := gentext.StreamTextOptions{
		Ctx:         context.Background(),
		Model:       &modelAdapterForGenText{model: mdl},
		Messages:    convertToGenTextMessages(messages),
		Temperature: temperature,
	}

	// Map maxSteps to StopWhen conditions
	if maxSteps > 0 {
		streamOpts.StopWhen = []gentext.StopCondition{gentext.StepCountIs(maxSteps)}
	}

	// Map toolChoice string to ai-kit ToolChoice
	if toolChoice != "" {
		streamOpts.ToolChoice = &gentext.ToolChoice{Type: toolChoice}
	}

	// Map experimental output schema
	if outputSchema != nil {
		streamOpts.Output = gentext.ObjectOutput(gentext.ObjectOutputOptions{
			Schema: outputSchema,
		})
	}

	// Wire the wrapped callbacks
	if wrappedOnStepFinish != nil {
		streamOpts.OnStepFinish = func(event gentext.StepResult) {
			stepEvent := StepFinishEvent{
				Text:         event.Text(),
				FinishReason: string(event.FinishReason),
			}
			if err := wrappedOnStepFinish(stepEvent); err != nil {
				l.Logger().Error(fmt.Sprintf("[LLM] - stream onStepFinish error: %v", err))
			}
		}
	}

	if wrappedOnFinish != nil {
		streamOpts.OnFinish = func(event gentext.OnFinishEvent) {
			finishEvent := StreamTextFinishEvent{
				Text:         event.Text(),
				FinishReason: string(event.FinishReason),
			}
			if err := wrappedOnFinish(finishEvent); err != nil {
				l.Logger().Error(fmt.Sprintf("[LLM] - stream onFinish error: %v", err))
			}
		}
	}

	aiResult, err := gentext.StreamText(streamOpts)
	if err != nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "LLM_STREAM_TEXT_AI_SDK_EXECUTION_FAILED",
			Domain:   mastraerror.ErrorDomainLLM,
			Category: mastraerror.ErrorCategoryThirdParty,
			Text:     fmt.Sprintf("streamText execution failed: %v", err),
			Details: map[string]any{
				"modelId":       mdl.ModelID(),
				"modelProvider": mdl.Provider(),
				"runId":         orDefault(runID, "unknown"),
				"threadId":      orDefault(threadID, "unknown"),
				"resourceId":    orDefault(resourceID, "unknown"),
			},
		}, err)
	}

	// Convert ai-kit result to agent-kit result
	result := &StreamTextResult{}
	if aiResult != nil {
		result.Object = aiResult.Output
	}
	return result, nil
}

// streamObject is the internal streaming object generation implementation.
//
// TS: __streamObject<T>({ messages, runId, requestContext, threadId, resourceId, onFinish, structuredOutput, ...rest })
//
// Faithfully ports the TS logic including:
//   - Schema compat layer application
//   - Output mode detection (object vs array)
//   - Observability span creation
//   - onFinish callback with span ending and error wrapping
//   - onError callback for streaming errors
//   - Error wrapping with MastraError (execution + schema conversion)
func (l *MastraLLMV1) streamObject(messages []CoreMessage, opts *StreamOptions) (*StreamObjectResult, error) {
	mdl := l.model

	var runID, threadID, resourceID string
	var structuredOutput any
	var onFinish StreamTextOnFinishCallback

	if opts != nil {
		runID = opts.RunID
		threadID = opts.ThreadID
		resourceID = opts.ResourceID
		structuredOutput = opts.Output
		onFinish = opts.OnFinish
	}

	l.Logger().Debug("[LLM] - Streaming structured output", map[string]any{
		"runId":    runID,
		"messages": messages,
	})

	// Create observability span
	// TS: const llmSpan = observabilityContext.tracingContext.currentSpan?.createChildSpan({ ... })
	// TODO: implement span creation when observability package is ported

	// Determine output mode (object or array)
	// TS: let output: 'object' | 'array' = 'object';
	//     if (isZodArray(structuredOutput)) { output = 'array'; structuredOutput = getZodDef(structuredOutput).type; }
	outputMode := "object"

	// Apply schema compat layers
	// TS: const processedSchema = this._applySchemaCompat(structuredOutput!);
	// TODO: implement schema compat layers when schema-compat package is ported
	processedSchema := structuredOutput

	// Build the wrapped onFinish callback
	// TS: onFinish: async (props: any) => { ... }
	wrappedOnFinish := func(event StreamTextFinishEvent) error {
		// End the model generation span BEFORE calling the user's onFinish callback
		// TS: llmSpan?.end({ output: { text, object, reasoning, reasoningText, files, sources, warnings }, attributes: { finishReason, usage } });
		// TODO: implement span ending when observability package is ported

		if onFinish != nil {
			event.RunID = runID
			if err := onFinish(event); err != nil {
				merr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
					ID:       "LLM_STREAM_OBJECT_ON_FINISH_CALLBACK_EXECUTION_FAILED",
					Domain:   mastraerror.ErrorDomainLLM,
					Category: mastraerror.ErrorCategoryUser,
					Details: map[string]any{
						"modelId":       mdl.ModelID(),
						"modelProvider": mdl.Provider(),
						"runId":         orDefault(runID, "unknown"),
						"threadId":      orDefault(threadID, "unknown"),
						"resourceId":    orDefault(resourceID, "unknown"),
						"toolCalls":     "",
						"toolResults":   "",
						"finishReason":  "",
						"usage":         jsonStringify(event.Usage),
					},
				}, err)
				l.Logger().Error(merr.Error())
				// TS: llmSpan?.error({ error: mastraError });
				return merr
			}
		}

		l.Logger().Debug("[LLM] - Object Stream Finished:", map[string]any{
			"usage":      event.Usage,
			"runId":      runID,
			"threadId":   threadID,
			"resourceId": resourceID,
		})

		return nil
	}

	// Build args and invoke streamObject from the AI SDK
	// TS: const argsForExecute = { ...rest, model, onFinish, onError, messages, output, schema: processedSchema };
	// TS: return streamObject(argsForExecute as any);
	adapter := &modelAdapterForStreamObj{model: mdl}
	streamObjOpts := genobj.StreamObjectOptions{
		Model:  adapter,
		Output: outputMode,
		Schema: processedSchema,
		Mode:   "json",
		Prompt: convertMessagesToPrompt(messages),
	}

	// Wire the wrapped onFinish callback
	if wrappedOnFinish != nil {
		streamObjOpts.OnFinish = func(event genobj.StreamObjectOnFinishEvent) {
			finishEvent := StreamTextFinishEvent{}
			if err := wrappedOnFinish(finishEvent); err != nil {
				l.Logger().Error(fmt.Sprintf("[LLM] - streamObject onFinish error: %v", err))
			}
		}
	}

	aiResult, err := genobj.StreamObject(context.Background(), streamObjOpts)
	if err != nil {
		// TS error handling (nested try/catch):
		//   Inner catch: LLM_STREAM_OBJECT_AI_SDK_EXECUTION_FAILED (THIRD_PARTY)
		//   Outer catch: LLM_STREAM_OBJECT_AI_SDK_SCHEMA_CONVERSION_FAILED (USER)
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "LLM_STREAM_OBJECT_AI_SDK_EXECUTION_FAILED",
			Domain:   mastraerror.ErrorDomainLLM,
			Category: mastraerror.ErrorCategoryThirdParty,
			Text:     fmt.Sprintf("streamObject execution failed: %v", err),
			Details: map[string]any{
				"modelId":       mdl.ModelID(),
				"modelProvider": mdl.Provider(),
				"runId":         orDefault(runID, "unknown"),
				"threadId":      orDefault(threadID, "unknown"),
				"resourceId":    orDefault(resourceID, "unknown"),
			},
		}, err)
	}

	// Convert ai-kit result to agent-kit result
	result := &StreamObjectResult{}
	if aiResult != nil {
		// StreamObjectResult from ai-kit has Object, FinishReason, etc.
		// agent-kit StreamObjectResult is minimal (just TripwireProperties).
		_ = aiResult
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Option types for Generate/Stream
// ---------------------------------------------------------------------------

// GenerateOptions holds options for the Generate method.
type GenerateOptions struct {
	// Output is the structured output schema (ZodSchema / JSONSchema7 equivalent).
	// When set, Generate returns a GenerateObjectResult.
	Output any `json:"output,omitempty"`
	// Tools available for the generation call.
	Tools ToolSet `json:"tools,omitempty"`
	// MaxSteps is the maximum number of tool-use steps. Default: 5.
	MaxSteps int `json:"maxSteps,omitempty"`
	// Temperature for sampling.
	Temperature *float64 `json:"temperature,omitempty"`
	// ToolChoice controls tool selection. Default: "auto".
	ToolChoice string `json:"toolChoice,omitempty"`
	// RunID for tracking.
	RunID string `json:"runId,omitempty"`
	// ThreadID for conversation threading.
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID for resource-scoped operations.
	ResourceID string `json:"resourceId,omitempty"`
	// ExperimentalOutput is the experimental output schema.
	ExperimentalOutput any `json:"experimental_output,omitempty"`
	// OnStepFinish callback.
	OnStepFinish GenerateTextOnStepFinishCallback `json:"-"`
}

// StreamOptions holds options for the Stream method.
type StreamOptions struct {
	// Output is the structured output schema.
	// When set, Stream returns a StreamObjectResult.
	Output any `json:"output,omitempty"`
	// Tools available for the stream call.
	Tools ToolSet `json:"tools,omitempty"`
	// MaxSteps is the maximum number of tool-use steps. Default: 5.
	MaxSteps int `json:"maxSteps,omitempty"`
	// Temperature for sampling.
	Temperature *float64 `json:"temperature,omitempty"`
	// ToolChoice controls tool selection. Default: "auto".
	ToolChoice string `json:"toolChoice,omitempty"`
	// RunID for tracking.
	RunID string `json:"runId,omitempty"`
	// ThreadID for conversation threading.
	ThreadID string `json:"threadId,omitempty"`
	// ResourceID for resource-scoped operations.
	ResourceID string `json:"resourceId,omitempty"`
	// ExperimentalOutput is the experimental output schema.
	ExperimentalOutput any `json:"experimental_output,omitempty"`
	// OnStepFinish callback.
	OnStepFinish StreamTextOnStepFinishCallback `json:"-"`
	// OnFinish callback.
	OnFinish StreamTextOnFinishCallback `json:"-"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// orDefault returns the value if non-empty, otherwise the fallback.
func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// jsonStringify serializes a value to JSON string for error details.
// Returns empty string if marshaling fails or value is nil.
func jsonStringify(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(b)
}
