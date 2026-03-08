// Ported from: packages/core/src/stream/aisdk/v4/usage.ts
package v4

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// UsageStats mirrors ../../../observability UsageStats.
// Stub: simplified shape — real observability.UsageStats has additional metric fields.
type UsageStats struct {
	// InputTokens is the total input tokens (sum of all input details).
	InputTokens *int `json:"inputTokens,omitempty"`
	// OutputTokens is the total output tokens (sum of all output details).
	OutputTokens *int `json:"outputTokens,omitempty"`
}

// LanguageModelUsageV4 mirrors the AI SDK v4 LanguageModelUsage type definition.
// (matches the ai package's LanguageModelUsage)
type LanguageModelUsageV4 struct {
	PromptTokens     int  `json:"promptTokens"`
	CompletionTokens int  `json:"completionTokens"`
	TotalTokens      *int `json:"totalTokens,omitempty"`
}

// ConvertV4Usage converts AI SDK v4 LanguageModelUsage to our UsageStats format.
//
// Parameters:
//   - usage: The LanguageModelUsage from AI SDK v4 (may be nil)
//
// Returns normalized UsageStats.
func ConvertV4Usage(usage *LanguageModelUsageV4) UsageStats {
	if usage == nil {
		return UsageStats{}
	}

	return UsageStats{
		InputTokens:  &usage.PromptTokens,
		OutputTokens: &usage.CompletionTokens,
	}
}
