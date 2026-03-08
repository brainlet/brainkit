// Ported from: packages/core/src/voice/aisdk/speech.ts
package aisdk

import (
	"bytes"
	"errors"
	"io"

	"github.com/brainlet/brainkit/agent-kit/core/voice"
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

// ---------------------------------------------------------------------------
// AI SDK Speech types — imported from ai-kit
// ---------------------------------------------------------------------------

// SpeechModel is the AI SDK SpeechModel interface (V3 spec).
// Imported from brainlink/experiments/ai-kit/provider/speechmodel.
type SpeechModel = sm.SpeechModel

// SpeechResult wraps the ai-kit GenerateResult for Mastra compatibility.
type SpeechResult struct {
	Audio SpeechAudio
}

// SpeechAudio holds the generated audio data.
type SpeechAudio struct {
	Uint8Array []byte
}

// GenerateSpeechOptions holds options for speech generation.
type GenerateSpeechOptions struct {
	Model           SpeechModel
	Text            string
	Voice           string
	Language        string
	ProviderOptions map[string]any
	Headers         map[string]string
	// AbortSignal is omitted; use context.Context at call sites instead.
}

// GenerateSpeechFunc is the function signature for experimental_generateSpeech.
// Users must provide an implementation that calls the actual AI SDK.
type GenerateSpeechFunc func(opts GenerateSpeechOptions) (*SpeechResult, error)

// ---------------------------------------------------------------------------
// AISDKSpeech
// ---------------------------------------------------------------------------

// AISDKSpeechOptions holds optional construction parameters for AISDKSpeech.
type AISDKSpeechOptions struct {
	Voice string
}

// AISDKSpeech wraps an AI SDK SpeechModel as a MastraVoice provider.
type AISDKSpeech struct {
	*voice.MastraVoiceBase
	model        SpeechModel
	defaultVoice string

	// GenerateSpeech is the function used to generate speech from text.
	// Must be set before calling Speak.
	// TODO: replace with direct AI SDK call once ported.
	GenerateSpeech GenerateSpeechFunc
}

// NewAISDKSpeech creates a new AISDKSpeech provider.
func NewAISDKSpeech(model SpeechModel, options *AISDKSpeechOptions) *AISDKSpeech {
	var defaultVoice string
	if options != nil {
		defaultVoice = options.Voice
	}

	return &AISDKSpeech{
		MastraVoiceBase: voice.NewMastraVoiceBase(&voice.VoiceConfig{
			Name: "ai-sdk-speech",
		}),
		model:        model,
		defaultVoice: defaultVoice,
	}
}

// SpeakOptions holds provider-specific options for the Speak method.
type SpeakOptions struct {
	Speaker         string
	Language        string
	ProviderOptions map[string]any
	Headers         map[string]string
}

// Speak converts text to an audio stream using the AI SDK speech model.
func (s *AISDKSpeech) Speak(input string, options *voice.SpeakOptions) (io.Reader, error) {
	if s.GenerateSpeech == nil {
		return nil, errors.New("AISDKSpeech.GenerateSpeech function not set; wire AI SDK implementation")
	}

	voiceName := s.defaultVoice
	if options != nil && options.Speaker != "" {
		voiceName = options.Speaker
	}

	result, err := s.GenerateSpeech(GenerateSpeechOptions{
		Model: s.model,
		Text:  input,
		Voice: voiceName,
	})
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(result.Audio.Uint8Array), nil
}

// SpeakStream converts a text stream to an audio stream.
// It reads the entire stream into a string first, then delegates to Speak.
func (s *AISDKSpeech) SpeakStream(input io.Reader, options *voice.SpeakOptions) (io.Reader, error) {
	text, err := streamToText(input)
	if err != nil {
		return nil, err
	}
	return s.Speak(text, options)
}

// Listen is not supported by speech models; it always returns an error.
func (s *AISDKSpeech) Listen(_ io.Reader, _ any) (string, error) {
	return "", errors.New("AI SDK speech models do not support transcription. Use AISDKTranscription instead.")
}

// GetSpeakers returns an empty list; voice must be specified in Speak options.
func (s *AISDKSpeech) GetSpeakers() ([]voice.SpeakerInfo, error) {
	return []voice.SpeakerInfo{}, nil
}

// GetListener returns disabled since speech models cannot listen.
func (s *AISDKSpeech) GetListener() (*voice.ListenerInfo, error) {
	return &voice.ListenerInfo{Enabled: false}, nil
}

// streamToText reads an io.Reader fully and returns its content as a UTF-8 string.
func streamToText(r io.Reader) (string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
