// Ported from: packages/core/src/voice/aisdk/transcription.ts
package aisdk

import (
	"encoding/base64"
	"errors"
	"io"

	"github.com/brainlet/brainkit/agent-kit/core/voice"
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// ---------------------------------------------------------------------------
// AI SDK Transcription types — imported from ai-kit
// ---------------------------------------------------------------------------

// TranscriptionModel is the AI SDK TranscriptionModel interface (V3 spec).
// Imported from brainlink/experiments/ai-kit/provider/transcriptionmodel.
type TranscriptionModel = tm.TranscriptionModel

// TranscriptionResult wraps the ai-kit GenerateResult for Mastra compatibility.
type TranscriptionResult struct {
	Text string
}

// TranscribeOptions holds options for transcription.
type TranscribeOptions struct {
	Model           TranscriptionModel
	Audio           []byte
	ProviderOptions map[string]any
	Headers         map[string]string
	// AbortSignal is omitted; use context.Context at call sites instead.
}

// TranscribeFunc is the function signature for experimental_transcribe.
// Users must provide an implementation that calls the actual AI SDK.
// TODO: wire to real AI SDK implementation once ported.
type TranscribeFunc func(opts TranscribeOptions) (*TranscriptionResult, error)

// ---------------------------------------------------------------------------
// AISDKTranscription
// ---------------------------------------------------------------------------

// AISDKTranscription wraps an AI SDK TranscriptionModel as a MastraVoice provider.
type AISDKTranscription struct {
	*voice.MastraVoiceBase
	model TranscriptionModel

	// Transcribe is the function used to transcribe audio to text.
	// Must be set before calling Listen.
	// TODO: replace with direct AI SDK call once ported.
	Transcribe TranscribeFunc
}

// NewAISDKTranscription creates a new AISDKTranscription provider.
func NewAISDKTranscription(model TranscriptionModel) *AISDKTranscription {
	return &AISDKTranscription{
		MastraVoiceBase: voice.NewMastraVoiceBase(&voice.VoiceConfig{
			Name: "ai-sdk-transcription",
		}),
		model: model,
	}
}

// Speak is not supported by transcription models; it always returns an error.
func (t *AISDKTranscription) Speak(_ string, _ *voice.SpeakOptions) (io.Reader, error) {
	return nil, errors.New("AI SDK transcription models do not support text-to-speech. Use AISDKSpeech instead.")
}

// SpeakStream is not supported by transcription models; it always returns an error.
func (t *AISDKTranscription) SpeakStream(_ io.Reader, _ *voice.SpeakOptions) (io.Reader, error) {
	return nil, errors.New("AI SDK transcription models do not support text-to-speech. Use AISDKSpeech instead.")
}

// GetSpeakers returns an empty list since transcription models cannot speak.
func (t *AISDKTranscription) GetSpeakers() ([]voice.SpeakerInfo, error) {
	return []voice.SpeakerInfo{}, nil
}

// GetListener returns enabled since transcription models can listen.
func (t *AISDKTranscription) GetListener() (*voice.ListenerInfo, error) {
	return &voice.ListenerInfo{Enabled: true}, nil
}

// ListenOptions holds provider-specific options for the Listen method.
type ListenOptions struct {
	ProviderOptions map[string]any
	Headers         map[string]string
}

// Listen transcribes audio to text.
// For enhanced metadata (segments, language, duration), use the AI SDK's
// transcribe() function directly.
func (t *AISDKTranscription) Listen(audioStream io.Reader, options any) (string, error) {
	if t.Transcribe == nil {
		return "", errors.New("AISDKTranscription.Transcribe function not set; wire AI SDK implementation")
	}

	audioBuffer, err := convertToBuffer(audioStream)
	if err != nil {
		return "", err
	}

	result, err := t.Transcribe(TranscribeOptions{
		Model: t.model,
		Audio: audioBuffer,
	})
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

// convertToBuffer converts various audio input types to a byte slice.
// Supports io.Reader (streaming), []byte, and base64-encoded strings.
func convertToBuffer(audio any) ([]byte, error) {
	switch v := audio.(type) {
	case []byte:
		return v, nil
	case string:
		// Treat strings as base64-encoded audio data
		return base64.StdEncoding.DecodeString(v)
	case io.Reader:
		return io.ReadAll(v)
	default:
		return nil, errors.New("unsupported audio input type")
	}
}
