// Ported from: packages/provider/src/language-model/v3/language-model-v3-finish-reason.ts
package languagemodel

// UnifiedFinishReason is why a language model finished generating a response.
type UnifiedFinishReason string

const (
	// FinishReasonStop means the model generated a stop sequence.
	FinishReasonStop UnifiedFinishReason = "stop"

	// FinishReasonLength means the model generated the maximum number of tokens.
	FinishReasonLength UnifiedFinishReason = "length"

	// FinishReasonContentFilter means a content filter violation stopped the model.
	FinishReasonContentFilter UnifiedFinishReason = "content-filter"

	// FinishReasonToolCalls means the model triggered tool calls.
	FinishReasonToolCalls UnifiedFinishReason = "tool-calls"

	// FinishReasonError means the model stopped because of an error.
	FinishReasonError UnifiedFinishReason = "error"

	// FinishReasonOther means the model stopped for other reasons.
	FinishReasonOther UnifiedFinishReason = "other"
)

// FinishReason contains both a unified finish reason and a raw finish reason
// from the provider.
type FinishReason struct {
	// Unified is the unified finish reason across different providers.
	Unified UnifiedFinishReason

	// Raw is the original finish reason from the provider.
	Raw *string
}
