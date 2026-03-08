// Ported from: packages/provider/src/language-model/v3/language-model-v3-call-options.ts
package languagemodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// CallOptions contains the options for a language model call.
type CallOptions struct {
	// Prompt is a standardized prompt type.
	//
	// Note: This is NOT the user-facing prompt. The AI SDK methods will map the
	// user-facing prompt types such as chat or instruction prompts to this format.
	Prompt Prompt

	// MaxOutputTokens is the maximum number of tokens to generate.
	MaxOutputTokens *int

	// Temperature setting. The range depends on the provider and model.
	Temperature *float64

	// StopSequences are stop sequences. If set, the model will stop generating
	// text when one of the stop sequences is generated.
	StopSequences []string

	// TopP is the nucleus sampling parameter.
	TopP *float64

	// TopK samples from only the top K options for each subsequent token.
	TopK *int

	// PresencePenalty affects the likelihood of the model to repeat
	// information that is already in the prompt.
	PresencePenalty *float64

	// FrequencyPenalty affects the likelihood of the model to repeatedly
	// use the same words or phrases.
	FrequencyPenalty *float64

	// ResponseFormat specifies whether output should be text or JSON.
	// Default is text.
	ResponseFormat ResponseFormat

	// Seed is the seed (integer) to use for random sampling. If set and supported
	// by the model, calls will generate deterministic results.
	Seed *int

	// Tools are the tools that are available for the model.
	Tools []Tool

	// ToolChoice specifies how the tool should be selected. Defaults to 'auto'.
	ToolChoice ToolChoice

	// IncludeRawChunks includes raw chunks in the stream. Only applicable for streaming calls.
	IncludeRawChunks *bool

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Headers are additional HTTP headers to be sent with the request.
	// Only applicable for HTTP-based providers.
	Headers map[string]*string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions shared.ProviderOptions
}

// ResponseFormat is a sealed interface for specifying response format.
// Implementations: ResponseFormatText, ResponseFormatJSON.
type ResponseFormat interface {
	responseFormatType() string
}

// ResponseFormatText specifies text output.
type ResponseFormatText struct{}

func (ResponseFormatText) responseFormatType() string { return "text" }

// ResponseFormatJSON specifies JSON output, optionally with a schema.
type ResponseFormatJSON struct {
	// Schema is an optional JSON schema that the generated output should conform to.
	Schema map[string]any

	// Name of the output that should be generated.
	// Used by some providers for additional LLM guidance.
	Name *string

	// Description of the output that should be generated.
	// Used by some providers for additional LLM guidance.
	Description *string
}

func (ResponseFormatJSON) responseFormatType() string { return "json" }
