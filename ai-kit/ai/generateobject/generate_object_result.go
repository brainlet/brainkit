// Ported from: packages/ai/src/generate-object/generate-object-result.ts
package generateobject

// FinishReason represents why the generation finished.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type FinishReason = string

// LanguageModelUsage represents token usage for language model operations.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type LanguageModelUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// CallWarning is a warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type CallWarning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// LanguageModelRequestMetadata holds additional request information.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type LanguageModelRequestMetadata struct {
	Body any `json:"body,omitempty"`
}

// LanguageModelResponseMetadata holds additional response information.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type LanguageModelResponseMetadata struct {
	ID        string            `json:"id,omitempty"`
	ModelID   string            `json:"modelId,omitempty"`
	Timestamp any               `json:"timestamp,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      any               `json:"body,omitempty"`
}

// ProviderMetadata is additional provider-specific metadata.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ProviderMetadata = map[string]map[string]any

// GenerateObjectResult is the result of a generateObject call.
type GenerateObjectResult struct {
	// Object is the generated object.
	Object any

	// Reasoning is the reasoning that was used to generate the object.
	Reasoning string

	// FinishReason is the reason why the generation finished.
	FinishReason FinishReason

	// Usage is the token usage of the generated response.
	Usage LanguageModelUsage

	// Warnings from the model provider (e.g. unsupported settings).
	Warnings []CallWarning

	// Request is additional request information.
	Request LanguageModelRequestMetadata

	// Response is additional response information.
	Response LanguageModelResponseMetadata

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
}
