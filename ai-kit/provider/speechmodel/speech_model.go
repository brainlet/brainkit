// Ported from: packages/provider/src/speech-model/v3/speech-model-v3.ts
package speechmodel

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// AudioData represents generated audio that can be a string (base64) or bytes.
type AudioData interface {
	audioData()
}

// AudioDataString represents base64 encoded audio data.
type AudioDataString struct {
	Value string
}

func (AudioDataString) audioData() {}

// AudioDataBytes represents binary audio data.
type AudioDataBytes struct {
	Data []byte
}

func (AudioDataBytes) audioData() {}

// GenerateResult is the result of a speech model doGenerate call.
type GenerateResult struct {
	// Audio is generated audio as base64 string or binary data.
	Audio AudioData

	// Warnings for the call, e.g. unsupported settings.
	Warnings []shared.Warning

	// Request contains optional request information for telemetry and debugging.
	Request *GenerateResultRequest

	// Response contains response information for telemetry and debugging.
	Response GenerateResultResponse

	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata map[string]jsonvalue.JSONObject
}

// GenerateResultRequest contains request information.
type GenerateResultRequest struct {
	Body any
}

// GenerateResultResponse contains response information.
type GenerateResultResponse struct {
	// Timestamp for the start of the generated response.
	Timestamp time.Time

	// ModelID is the response model ID.
	ModelID string

	// Headers are the response headers.
	Headers shared.Headers

	// Body is the response body.
	Body any
}

// SpeechModel is the specification for a speech model (version 3).
type SpeechModel interface {
	// SpecificationVersion returns the speech model interface version.
	// Must return "v3".
	SpecificationVersion() string

	// Provider returns the name of the provider for logging purposes.
	Provider() string

	// ModelID returns the provider-specific model ID for logging purposes.
	ModelID() string

	// DoGenerate generates speech audio from text.
	DoGenerate(options CallOptions) (GenerateResult, error)
}
