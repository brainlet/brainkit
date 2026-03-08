// Ported from: packages/provider/src/transcription-model/v3/transcription-model-v3-call-options.ts
package transcriptionmodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
)

// ProviderOptions is provider-specific options for transcription models.
type ProviderOptions = map[string]jsonvalue.JSONObject

// CallOptions contains the options for a transcription model call.
type CallOptions struct {
	// Audio data to transcribe. Either raw bytes or a base64 encoded string.
	Audio AudioData

	// MediaType is the IANA media type of the audio data.
	MediaType string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions ProviderOptions

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Headers are additional HTTP headers to be sent with the request.
	Headers map[string]*string
}

// AudioData is a sealed interface representing audio input data.
type AudioData interface {
	audioData()
}

// AudioDataBytes represents raw binary audio data.
type AudioDataBytes struct {
	Data []byte
}

func (AudioDataBytes) audioData() {}

// AudioDataString represents base64 encoded audio data.
type AudioDataString struct {
	Value string
}

func (AudioDataString) audioData() {}
