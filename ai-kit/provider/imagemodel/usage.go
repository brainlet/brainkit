// Ported from: packages/provider/src/image-model/v3/image-model-v3-usage.ts
package imagemodel

// Usage contains usage information for an image model call.
type Usage struct {
	// InputTokens is the number of input (prompt) tokens used.
	InputTokens *int

	// OutputTokens is the number of output tokens used, if reported by the provider.
	OutputTokens *int

	// TotalTokens is the total number of tokens as reported by the provider.
	TotalTokens *int
}
