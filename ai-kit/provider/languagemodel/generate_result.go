// Ported from: packages/provider/src/language-model/v3/language-model-v3-generate-result.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// GenerateResult is the result of a language model doGenerate call.
type GenerateResult struct {
	// Content is ordered content that the model has generated.
	Content []Content

	// FinishReason is the reason the model stopped generating.
	FinishReason FinishReason

	// Usage is the usage information.
	Usage Usage

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata shared.ProviderMetadata

	// Request contains optional request information for telemetry and debugging.
	Request *GenerateResultRequest

	// Response contains optional response information for telemetry and debugging.
	Response *GenerateResultResponse

	// Warnings for the call, e.g. unsupported settings.
	Warnings []shared.Warning
}

// GenerateResultRequest contains request information for telemetry and debugging.
type GenerateResultRequest struct {
	// Body is the request HTTP body that was sent to the provider API.
	Body any
}

// GenerateResultResponse contains response information for telemetry and debugging.
type GenerateResultResponse struct {
	ResponseMetadata

	// Headers are the response headers.
	Headers shared.Headers

	// Body is the response HTTP body.
	Body any
}
