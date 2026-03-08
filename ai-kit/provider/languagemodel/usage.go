// Ported from: packages/provider/src/language-model/v3/language-model-v3-usage.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// Usage contains usage information for a language model call.
type Usage struct {
	// InputTokens contains information about the input tokens.
	InputTokens InputTokenUsage

	// OutputTokens contains information about the output tokens.
	OutputTokens OutputTokenUsage

	// Raw is the raw usage information from the provider.
	// This can include additional information not part of the standard usage.
	Raw jsonvalue.JSONObject
}

// InputTokenUsage contains information about input token usage.
type InputTokenUsage struct {
	// Total is the total number of input (prompt) tokens used.
	Total *int

	// NoCache is the number of non-cached input tokens used.
	NoCache *int

	// CacheRead is the number of cached input tokens read.
	CacheRead *int

	// CacheWrite is the number of cached input tokens written.
	CacheWrite *int
}

// OutputTokenUsage contains information about output token usage.
type OutputTokenUsage struct {
	// Total is the total number of output (completion) tokens used.
	Total *int

	// Text is the number of text tokens used.
	Text *int

	// Reasoning is the number of reasoning tokens used.
	Reasoning *int
}
