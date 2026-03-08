// Ported from: packages/ai/src/types/usage.ts
package aitypes

// LanguageModelUsage represents the number of tokens used in a prompt and completion.
type LanguageModelUsage struct {
	// InputTokens is the total number of input (prompt) tokens used.
	InputTokens *int `json:"inputTokens"`

	// InputTokenDetails contains detailed information about the input tokens.
	InputTokenDetails InputTokenDetails `json:"inputTokenDetails"`

	// OutputTokens is the number of total output (completion) tokens used.
	OutputTokens *int `json:"outputTokens"`

	// OutputTokenDetails contains detailed information about the output tokens.
	OutputTokenDetails OutputTokenDetails `json:"outputTokenDetails"`

	// TotalTokens is the total number of tokens used.
	TotalTokens *int `json:"totalTokens"`

	// Deprecated: Use OutputTokenDetails.ReasoningTokens instead.
	ReasoningTokens *int `json:"reasoningTokens,omitempty"`

	// Deprecated: Use InputTokenDetails.CacheReadTokens instead.
	CachedInputTokens *int `json:"cachedInputTokens,omitempty"`

	// Raw is usage information from the provider in the shape the provider returns.
	// It can include additional information that is not part of the standard usage information.
	Raw JSONObject `json:"raw,omitempty"`
}

// InputTokenDetails contains detailed information about input tokens.
type InputTokenDetails struct {
	// NoCacheTokens is the number of non-cached input (prompt) tokens used.
	NoCacheTokens *int `json:"noCacheTokens"`

	// CacheReadTokens is the number of cached input (prompt) tokens read.
	CacheReadTokens *int `json:"cacheReadTokens"`

	// CacheWriteTokens is the number of cached input (prompt) tokens written.
	CacheWriteTokens *int `json:"cacheWriteTokens"`
}

// OutputTokenDetails contains detailed information about output tokens.
type OutputTokenDetails struct {
	// TextTokens is the number of text tokens used.
	TextTokens *int `json:"textTokens"`

	// ReasoningTokens is the number of reasoning tokens used.
	ReasoningTokens *int `json:"reasoningTokens"`
}

// EmbeddingModelUsage represents the number of tokens used in an embedding.
type EmbeddingModelUsage struct {
	// Tokens is the number of tokens used in the embedding.
	Tokens int `json:"tokens"`
}

// ImageModelUsage represents usage information for an image model call.
//
// Corresponds to ImageModelV4Usage from @ai-sdk/provider.
type ImageModelUsage struct {
	// InputTokens is the number of input (prompt) tokens used.
	InputTokens *int `json:"inputTokens"`

	// OutputTokens is the number of output tokens used, if reported by the provider.
	OutputTokens *int `json:"outputTokens"`

	// TotalTokens is the total number of tokens as reported by the provider.
	TotalTokens *int `json:"totalTokens"`
}

// LanguageModelV4Usage mirrors the V4 usage structure from @ai-sdk/provider
// used as input to AsLanguageModelUsage.
type LanguageModelV4Usage struct {
	InputTokens struct {
		Total    *int `json:"total"`
		NoCache  *int `json:"noCache"`
		CacheRead  *int `json:"cacheRead"`
		CacheWrite *int `json:"cacheWrite"`
	} `json:"inputTokens"`
	OutputTokens struct {
		Total     *int `json:"total"`
		Text      *int `json:"text"`
		Reasoning *int `json:"reasoning"`
	} `json:"outputTokens"`
	Raw JSONObject `json:"raw,omitempty"`
}

// AsLanguageModelUsage converts a LanguageModelV4Usage to a LanguageModelUsage.
func AsLanguageModelUsage(usage LanguageModelV4Usage) LanguageModelUsage {
	return LanguageModelUsage{
		InputTokens: usage.InputTokens.Total,
		InputTokenDetails: InputTokenDetails{
			NoCacheTokens:    usage.InputTokens.NoCache,
			CacheReadTokens:  usage.InputTokens.CacheRead,
			CacheWriteTokens: usage.InputTokens.CacheWrite,
		},
		OutputTokens: usage.OutputTokens.Total,
		OutputTokenDetails: OutputTokenDetails{
			TextTokens:      usage.OutputTokens.Text,
			ReasoningTokens: usage.OutputTokens.Reasoning,
		},
		TotalTokens:       addTokenCounts(usage.InputTokens.Total, usage.OutputTokens.Total),
		Raw:               usage.Raw,
		ReasoningTokens:   usage.OutputTokens.Reasoning,
		CachedInputTokens: usage.InputTokens.CacheRead,
	}
}

// CreateNullLanguageModelUsage creates a LanguageModelUsage with all nil values.
func CreateNullLanguageModelUsage() LanguageModelUsage {
	return LanguageModelUsage{
		InputTokens: nil,
		InputTokenDetails: InputTokenDetails{
			NoCacheTokens:    nil,
			CacheReadTokens:  nil,
			CacheWriteTokens: nil,
		},
		OutputTokens: nil,
		OutputTokenDetails: OutputTokenDetails{
			TextTokens:      nil,
			ReasoningTokens: nil,
		},
		TotalTokens: nil,
		Raw:         nil,
	}
}

// AddLanguageModelUsage adds two LanguageModelUsage values together.
func AddLanguageModelUsage(usage1, usage2 LanguageModelUsage) LanguageModelUsage {
	return LanguageModelUsage{
		InputTokens: addTokenCounts(usage1.InputTokens, usage2.InputTokens),
		InputTokenDetails: InputTokenDetails{
			NoCacheTokens:    addTokenCounts(usage1.InputTokenDetails.NoCacheTokens, usage2.InputTokenDetails.NoCacheTokens),
			CacheReadTokens:  addTokenCounts(usage1.InputTokenDetails.CacheReadTokens, usage2.InputTokenDetails.CacheReadTokens),
			CacheWriteTokens: addTokenCounts(usage1.InputTokenDetails.CacheWriteTokens, usage2.InputTokenDetails.CacheWriteTokens),
		},
		OutputTokens: addTokenCounts(usage1.OutputTokens, usage2.OutputTokens),
		OutputTokenDetails: OutputTokenDetails{
			TextTokens:      addTokenCounts(usage1.OutputTokenDetails.TextTokens, usage2.OutputTokenDetails.TextTokens),
			ReasoningTokens: addTokenCounts(usage1.OutputTokenDetails.ReasoningTokens, usage2.OutputTokenDetails.ReasoningTokens),
		},
		TotalTokens:       addTokenCounts(usage1.TotalTokens, usage2.TotalTokens),
		ReasoningTokens:   addTokenCounts(usage1.ReasoningTokens, usage2.ReasoningTokens),
		CachedInputTokens: addTokenCounts(usage1.CachedInputTokens, usage2.CachedInputTokens),
	}
}

// addTokenCounts adds two optional token counts together.
// Returns nil if both are nil, otherwise returns the sum (treating nil as 0).
func addTokenCounts(count1, count2 *int) *int {
	if count1 == nil && count2 == nil {
		return nil
	}
	v1 := 0
	if count1 != nil {
		v1 = *count1
	}
	v2 := 0
	if count2 != nil {
		v2 = *count2
	}
	result := v1 + v2
	return &result
}

// AddImageModelUsage adds two ImageModelUsage values together.
func AddImageModelUsage(usage1, usage2 ImageModelUsage) ImageModelUsage {
	return ImageModelUsage{
		InputTokens:  addTokenCounts(usage1.InputTokens, usage2.InputTokens),
		OutputTokens: addTokenCounts(usage1.OutputTokens, usage2.OutputTokens),
		TotalTokens:  addTokenCounts(usage1.TotalTokens, usage2.TotalTokens),
	}
}
