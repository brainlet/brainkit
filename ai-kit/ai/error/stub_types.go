// Ported from: packages/ai/src/error/ (stub types for dependencies not yet ported)
// TODO: import from correct packages once they exist
package aierror

import "time"

// --- From @ai-sdk/provider ---

// AISDKError is the base error type for the AI SDK.
// TODO: import from brainlink/experiments/ai-kit/provider
type AISDKError struct {
	// Name is the error name/classification.
	Name string
	// Message is the human-readable error message.
	Message string
	// Cause is the underlying cause of the error, if any.
	Cause error
}

func (e *AISDKError) Error() string {
	return e.Message
}

func (e *AISDKError) Unwrap() error {
	return e.Cause
}

// GetErrorMessage extracts a human-readable message from an unknown error value.
// TODO: import from brainlink/experiments/ai-kit/provider
func GetErrorMessage(err interface{}) string {
	if err == nil {
		return "unknown error"
	}
	if s, ok := err.(string); ok {
		return s
	}
	if e, ok := err.(error); ok {
		return e.Error()
	}
	return "unknown error"
}

// --- From packages/ai/src/types ---

// LanguageModelResponseMetadata holds response metadata from a language model call.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModelResponseMetadata struct {
	// ID for the generated response.
	ID string
	// Timestamp for the start of the generated response.
	Timestamp time.Time
	// ModelID is the ID of the response model that was used.
	ModelID string
	// Headers are the response headers (available only for HTTP-based providers).
	Headers map[string]string
}

// InputTokenDetails contains detailed info about input tokens.
// TODO: import from brainlink/experiments/ai-kit/types
type InputTokenDetails struct {
	NoCacheTokens   *int
	CacheReadTokens *int
	CacheWriteTokens *int
}

// OutputTokenDetails contains detailed info about output tokens.
// TODO: import from brainlink/experiments/ai-kit/types
type OutputTokenDetails struct {
	TextTokens      *int
	ReasoningTokens *int
}

// LanguageModelUsage represents the number of tokens used in a prompt and completion.
// TODO: import from brainlink/experiments/ai-kit/types
type LanguageModelUsage struct {
	InputTokens       *int
	InputTokenDetails  InputTokenDetails
	OutputTokens      *int
	OutputTokenDetails OutputTokenDetails
	TotalTokens       *int
	// Deprecated: Use OutputTokenDetails.ReasoningTokens instead.
	ReasoningTokens *int
	// Deprecated: Use InputTokenDetails.CacheReadTokens instead.
	CachedInputTokens *int
	Raw               map[string]interface{}
}

// FinishReason represents why a language model finished generating a response.
// TODO: import from brainlink/experiments/ai-kit/types
type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonContentFilter FinishReason = "content-filter"
	FinishReasonToolCalls     FinishReason = "tool-calls"
	FinishReasonError         FinishReason = "error"
	FinishReasonOther         FinishReason = "other"
)

// ImageModelResponseMetadata holds response metadata from an image model call.
// TODO: import from brainlink/experiments/ai-kit/types
type ImageModelResponseMetadata struct {
	Timestamp time.Time
	ModelID   string
	Headers   map[string]string
}

// SpeechModelResponseMetadata holds response metadata from a speech model call.
// TODO: import from brainlink/experiments/ai-kit/types
type SpeechModelResponseMetadata struct {
	Timestamp time.Time
	ModelID   string
	Headers   map[string]string
	Body      interface{}
}

// TranscriptionModelResponseMetadata holds response metadata from a transcription model call.
// TODO: import from brainlink/experiments/ai-kit/types
type TranscriptionModelResponseMetadata struct {
	Timestamp time.Time
	ModelID   string
	Headers   map[string]string
}

// VideoModelResponseMetadata holds response metadata from a video model call.
// TODO: import from brainlink/experiments/ai-kit/types
type VideoModelResponseMetadata struct {
	Timestamp        time.Time
	ModelID          string
	Headers          map[string]string
	ProviderMetadata map[string]map[string]interface{}
}

// SingleRequestTextStreamPart represents a chunk in a text generation stream.
// This is a union type in TypeScript; in Go we use an interface with a discriminator.
// TODO: import from brainlink/experiments/ai-kit/generate_text
type SingleRequestTextStreamPart struct {
	Type             string
	ID               string
	Delta            string
	ProviderMetadata map[string]map[string]interface{}
}
