// Ported from: packages/provider/src/speech-model/v3/speech-model-v3-call-options.ts
package speechmodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
)

// ProviderOptions is provider-specific options for speech models.
type ProviderOptions = map[string]jsonvalue.JSONObject

// CallOptions contains the options for a speech model call.
type CallOptions struct {
	// Text to convert to speech.
	Text string

	// Voice to use for speech synthesis. Provider-specific.
	Voice *string

	// OutputFormat is the desired output format for the audio e.g. "mp3", "wav", etc.
	OutputFormat *string

	// Instructions for the speech generation e.g. "Speak in a slow and steady tone".
	Instructions *string

	// Speed of the speech generation.
	Speed *float64

	// Language for speech generation. ISO 639-1 language code (e.g. "en", "es", "fr")
	// or "auto" for automatic language detection.
	Language *string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions ProviderOptions

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Headers are additional HTTP headers to be sent with the request.
	Headers map[string]*string
}
