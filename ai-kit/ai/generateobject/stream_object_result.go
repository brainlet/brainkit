// Ported from: packages/ai/src/generate-object/stream-object-result.ts
package generateobject

// ObjectStreamPartType represents the type of an object stream part.
type ObjectStreamPartType string

const (
	ObjectStreamPartTypeObject    ObjectStreamPartType = "object"
	ObjectStreamPartTypeTextDelta ObjectStreamPartType = "text-delta"
	ObjectStreamPartTypeError     ObjectStreamPartType = "error"
	ObjectStreamPartTypeFinish    ObjectStreamPartType = "finish"
)

// ObjectStreamPart represents a part of an object stream.
type ObjectStreamPart struct {
	// Type is the type of this stream part.
	Type ObjectStreamPartType

	// Object is the partial or complete object (when Type is "object").
	Object any

	// TextDelta is the text delta (when Type is "text-delta").
	TextDelta string

	// Error is the error (when Type is "error").
	Error error

	// FinishReason is the reason why the generation finished (when Type is "finish").
	FinishReason FinishReason

	// Usage is the token usage (when Type is "finish").
	Usage LanguageModelUsage

	// Response is the response metadata (when Type is "finish").
	Response LanguageModelResponseMetadata

	// ProviderMetadata is provider-specific metadata (when Type is "finish").
	ProviderMetadata ProviderMetadata
}

// StreamObjectResult represents the result of a streamObject call.
// It contains channels for streaming partial objects and additional information.
type StreamObjectResult struct {
	// PartialObjectStream is a channel that emits partial objects as they are generated.
	PartialObjectStream <-chan any

	// TextStream is a channel that emits text chunks of the JSON representation.
	TextStream <-chan string

	// FullStream is a channel that emits all stream parts including partial objects, errors, and finish events.
	FullStream <-chan ObjectStreamPart

	// Object is the final complete object, available after the stream is done.
	Object any

	// FinishReason is the reason why the generation finished.
	FinishReason FinishReason

	// Usage is the token usage of the generated response.
	Usage LanguageModelUsage

	// Warnings from the model provider.
	Warnings []CallWarning

	// Request is additional request information.
	Request LanguageModelRequestMetadata

	// Response is additional response information.
	Response LanguageModelResponseMetadata

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
}
