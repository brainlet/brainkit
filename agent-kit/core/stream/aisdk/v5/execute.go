// Ported from: packages/core/src/stream/aisdk/v5/execute.ts
package v5

import (
	"context"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/stream/aisdk/v5/compat"
	"github.com/brainlet/brainkit/agent-kit/core/stream/base"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraLanguageModel mirrors the TS MastraLanguageModel from llm/model/shared.types.
// Stub: real llm.MastraLanguageModel has additional provider-specific methods and
// wraps the AI SDK provider model; simplified here for execute.go's needs.
type MastraLanguageModel struct {
	ModelID              string
	Provider             string
	SpecificationVersion string // "v2" or "v3"
	// DoStream calls the model's streaming method.
	DoStream func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error)
	// DoGenerate calls the model's non-streaming method.
	DoGenerate func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error)
}

// DoStreamOptions are the options passed to DoStream/DoGenerate.
// Stub: parallel-stubs architecture — real type lives in llm/model with different shape.
type DoStreamOptions struct {
	Prompt           any                    // LanguageModelV2Prompt
	Tools            []compat.PreparedTool  // prepared tools
	ToolChoice       *compat.PreparedToolChoice // prepared tool choice
	ProviderOptions  map[string]any
	AbortSignal      context.Context
	IncludeRawChunks bool
	ResponseFormat   *base.ResponseFormat
	Headers          map[string]string
	// Extra model settings (merged from modelSettings minus maxRetries/headers)
	Extra map[string]any
}

// SharedProviderOptions mirrors the TS SharedProviderOptions.
// Stub: type alias — real type has same shape (map[string]any).
type SharedProviderOptions = map[string]any

// LoopModelSettings mirrors the TS LoopOptions['modelSettings'].
// Stub: real loop type has additional fields; simplified for execute.go's needs.
type LoopModelSettings struct {
	MaxRetries *int
	Headers    map[string]string
	// Extra catches any other model settings.
	Extra map[string]any
}

// StructuredOutputOptions mirrors the TS StructuredOutputOptions from agent/types.
// Stub: parallel-stubs architecture — real agent type has different field set.
type StructuredOutputOptions struct {
	Schema              base.OutputSchema
	Model               any    // MastraLanguageModel for processor mode, nil for direct
	JSONPromptInjection bool
	ErrorStrategy       string // "strict" | "warn" | "fallback"
	Fallback            any
}

// ModelMethodType mirrors the TS ModelMethodType from llm/model/model.loop.types.
// Stub: simple string enum — real type has same shape but lives in llm/model.
type ModelMethodType string

const (
	MethodStream   ModelMethodType = "stream"
	MethodGenerate ModelMethodType = "generate"
)

// APICallError mirrors the TS APICallError from @internal/ai-sdk-v5.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type APICallError struct {
	Err         error
	IsRetryable bool
}

func (e *APICallError) Error() string { return e.Err.Error() }
func (e *APICallError) Unwrap() error { return e.Err }

// IsAPICallError checks if an error is an APICallError.
func IsAPICallError(err error) (*APICallError, bool) {
	if ace, ok := err.(*APICallError); ok {
		return ace, true
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// omit helper
// ---------------------------------------------------------------------------

// omitFromMap returns a copy of m without the specified keys.
func omitFromMap(m map[string]any, keys ...string) map[string]any {
	result := make(map[string]any, len(m))
	skip := make(map[string]bool, len(keys))
	for _, k := range keys {
		skip[k] = true
	}
	for k, v := range m {
		if !skip[k] {
			result[k] = v
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// injectJsonInstructionIntoMessages stub
// ---------------------------------------------------------------------------

// injectJsonInstructionIntoMessages mirrors @ai-sdk/provider-utils-v5 injectJsonInstructionIntoMessages.
// It appends JSON schema instructions to the prompt messages.
// Stub: returns messages unchanged. Real implementation appends a system message
// with JSON schema instructions; ai-sdk provider-utils not ported to Go.
func injectJsonInstructionIntoMessages(messages any, schema *base.JSONSchema7, prefix string, suffix string) any {
	// Stub: returns messages unchanged.
	// In the real implementation, this appends a system message with the JSON schema
	// instruction to guide the model's output.
	return messages
}

// ---------------------------------------------------------------------------
// ExecutionProps
// ---------------------------------------------------------------------------

// ExecutionProps configures the execute function.
type ExecutionProps struct {
	RunID            string
	Model            *MastraLanguageModel
	ProviderOptions  SharedProviderOptions
	InputMessages    any // LanguageModelV2Prompt
	Tools            map[string]any
	ToolChoice       any // ToolChoice
	ActiveTools      []string
	Options          *ExecuteOptions
	IncludeRawChunks bool
	ModelSettings    *LoopModelSettings
	OnResult         stream.OnResult
	StructuredOutput *StructuredOutputOptions
	Headers          map[string]string
	ShouldThrowError bool
	MethodType       ModelMethodType
	GenerateID       IdGenerator
}

// ExecuteOptions are optional execution parameters.
type ExecuteOptions struct {
	AbortSignal context.Context
}

// ---------------------------------------------------------------------------
// Execute
// ---------------------------------------------------------------------------

// Execute wires an AISDKV5InputStream with model.doStream, retry logic,
// structured output mode handling, and response format injection.
//
// This mirrors the TS execute() function which:
//  1. Creates an AISDKV5InputStream
//  2. Determines target version based on model's specificationVersion
//  3. Prepares tools and tool choice via compat.PrepareToolsAndToolChoice
//  4. Handles structured output modes (direct vs processor)
//  5. Injects JSON schema instructions when appropriate
//  6. Calls model.DoStream/DoGenerate with retry logic
func Execute(props ExecutionProps) <-chan stream.ChunkType {
	v5 := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
		Component:  "LLM",
		Name:       props.Model.ModelID,
		GenerateID: props.GenerateID,
	})

	// Determine target version based on model's specificationVersion
	// V3 models (AI SDK v6) need 'provider' type, V2 models need 'provider-defined'
	targetVersion := compat.ModelSpecVersionV2
	if props.Model.SpecificationVersion == "v3" {
		targetVersion = compat.ModelSpecVersionV3
	}

	toolsAndToolChoice := compat.PrepareToolsAndToolChoice(compat.PrepareToolsAndToolChoiceParams{
		Tools:         props.Tools,
		ToolChoice:    props.ToolChoice,
		ActiveTools:   props.ActiveTools,
		TargetVersion: targetVersion,
	})

	// Determine structured output mode
	var structuredOutputMode string
	if props.StructuredOutput != nil && props.StructuredOutput.Schema != nil {
		if props.StructuredOutput.Model != nil {
			structuredOutputMode = "processor"
		} else {
			structuredOutputMode = "direct"
		}
	}

	// Get response format from schema
	var responseFormat *base.ResponseFormat
	if props.StructuredOutput != nil && props.StructuredOutput.Schema != nil {
		rf := base.GetResponseFormat(props.StructuredOutput.Schema)
		responseFormat = &rf
	}

	prompt := props.InputMessages

	// For direct mode (no model provided for structuring agent), inject JSON schema
	// instruction if opting out of native response format with jsonPromptInjection
	if structuredOutputMode == "direct" && responseFormat != nil &&
		responseFormat.Type == base.ResponseFormatJSON && props.StructuredOutput.JSONPromptInjection {
		prompt = injectJsonInstructionIntoMessages(
			props.InputMessages,
			responseFormat.Schema,
			"",
			"",
		)
	}

	// For processor mode (model provided for structuring agent), inject a custom
	// prompt to inform the main agent about the structured output schema
	if structuredOutputMode == "processor" && responseFormat != nil &&
		responseFormat.Type == base.ResponseFormatJSON && responseFormat.Schema != nil {
		prompt = injectJsonInstructionIntoMessages(
			props.InputMessages,
			responseFormat.Schema,
			"Your response will be processed by another agent to extract structured data. Please ensure your response contains comprehensive information for all the following fields that will be extracted:\n",
			"\n\nYou don't need to format your response as JSON unless the user asks you to. Just ensure your natural language response includes relevant information for each field in the schema above.",
		)
	}

	// Enable OpenAI's strict JSON schema mode to ensure schema compliance
	providerOptionsToUse := props.ProviderOptions
	if props.Model.Provider != "" && len(props.Model.Provider) >= 6 &&
		props.Model.Provider[:6] == "openai" &&
		responseFormat != nil && responseFormat.Type == base.ResponseFormatJSON &&
		(props.StructuredOutput == nil || !props.StructuredOutput.JSONPromptInjection) {
		if providerOptionsToUse == nil {
			providerOptionsToUse = make(SharedProviderOptions)
		}
		openaiOpts, _ := providerOptionsToUse["openai"].(map[string]any)
		if openaiOpts == nil {
			openaiOpts = make(map[string]any)
		}
		merged := make(map[string]any)
		merged["strictJsonSchema"] = true
		for k, v := range openaiOpts {
			merged[k] = v
		}
		providerOptionsToUse["openai"] = merged
	}

	outputStream := v5.Initialize(InitializeParams{
		RunID:    props.RunID,
		OnResult: props.OnResult,
		CreateStream: func() (*stream.LanguageModelV2StreamResult, error) {
			maxRetries := 2
			if props.ModelSettings != nil && props.ModelSettings.MaxRetries != nil {
				maxRetries = *props.ModelSettings.MaxRetries
			}

			var ctx context.Context
			if props.Options != nil && props.Options.AbortSignal != nil {
				ctx = props.Options.AbortSignal
			} else {
				ctx = context.Background()
			}

			// Retry loop
			var lastErr error
			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					// Simple backoff between retries
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(time.Duration(attempt*100) * time.Millisecond):
					}
				}

				fn := props.Model.DoStream
				if props.MethodType != MethodStream {
					fn = props.Model.DoGenerate
				}

				// Determine response format to pass
				var rf *base.ResponseFormat
				if structuredOutputMode == "direct" &&
					(props.StructuredOutput == nil || !props.StructuredOutput.JSONPromptInjection) {
					rf = responseFormat
				}

				result, err := fn(ctx, DoStreamOptions{
					Prompt:           prompt,
					Tools:            toolsAndToolChoice.Tools,
					ToolChoice:       toolsAndToolChoice.ToolChoice,
					ProviderOptions:  providerOptionsToUse,
					AbortSignal:      ctx,
					IncludeRawChunks: props.IncludeRawChunks,
					ResponseFormat:   rf,
					Headers:          props.Headers,
				})

				if err != nil {
					lastErr = err
					// Check if error is retryable
					if ace, ok := IsAPICallError(err); ok {
						if !ace.IsRetryable {
							break
						}
					}
					continue
				}

				return result, nil
			}

			// All retries exhausted
			if props.ShouldThrowError {
				return nil, lastErr
			}

			// Return error stream instead of throwing
			errCh := make(chan stream.LanguageModelV2StreamPart, 1)
			errCh <- stream.LanguageModelV2StreamPart{
				Type: "error",
				Data: map[string]any{"error": lastErr},
			}
			close(errCh)

			return &stream.LanguageModelV2StreamResult{
				Stream:   errCh,
				Warnings: nil,
			}, nil
		},
	})

	return outputStream
}

// ---------------------------------------------------------------------------
// String helpers for error messages
// ---------------------------------------------------------------------------

func init() {
	// Ensure fmt is used (it's available for error formatting if needed)
	_ = fmt.Sprintf
}
