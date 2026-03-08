// Ported from: packages/ai/src/generate-speech/generate-speech.ts
package generatespeech

import (
	"context"
	"fmt"
)

// SpeechModel is the interface for speech generation models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type SpeechModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoGenerate performs the speech generation operation.
	DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

// DoGenerateOptions are the options passed to SpeechModel.DoGenerate.
type DoGenerateOptions struct {
	Text            string
	Voice           string
	OutputFormat    string
	Instructions    string
	Speed           *float64
	Language        string
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// DoGenerateResult is the result from SpeechModel.DoGenerate.
type DoGenerateResult struct {
	Audio            []byte
	Warnings         []Warning
	Response         SpeechModelResponseMetadata
	ProviderMetadata map[string]map[string]any
}

// GenerateSpeechOptions are the options for the GenerateSpeech function.
type GenerateSpeechOptions struct {
	// Model is the speech model to use.
	Model SpeechModel

	// Text is the text to convert to speech.
	Text string

	// Voice is the voice to use for speech generation.
	Voice string

	// OutputFormat is the desired output format for the audio (e.g., "mp3", "wav").
	OutputFormat string

	// Instructions for the speech generation (e.g., "Speak in a slow and steady tone").
	Instructions string

	// Speed of the speech generation.
	Speed *float64

	// Language for speech generation (ISO 639-1 code or "auto").
	Language string

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any
}

// GenerateSpeech generates speech audio using a speech model.
func GenerateSpeech(ctx context.Context, opts GenerateSpeechOptions) (*SpeechResult, error) {
	model := opts.Model

	providerOptions := opts.ProviderOptions
	if providerOptions == nil {
		providerOptions = map[string]map[string]any{}
	}

	result, err := model.DoGenerate(ctx, DoGenerateOptions{
		Text:            opts.Text,
		Voice:           opts.Voice,
		OutputFormat:    opts.OutputFormat,
		Instructions:    opts.Instructions,
		Speed:           opts.Speed,
		Language:        opts.Language,
		Headers:         opts.Headers,
		ProviderOptions: providerOptions,
	})
	if err != nil {
		return nil, err
	}

	if result.Audio == nil || len(result.Audio) == 0 {
		return nil, fmt.Errorf("no speech audio generated")
	}

	mediaType := detectAudioMediaType(result.Audio)
	if mediaType == "" {
		mediaType = "audio/mp3"
	}

	audioFile, err := NewGeneratedAudioFile(result.Audio, mediaType)
	if err != nil {
		return nil, err
	}

	providerMeta := result.ProviderMetadata
	if providerMeta == nil {
		providerMeta = map[string]map[string]any{}
	}

	return &SpeechResult{
		Audio:            *audioFile,
		Warnings:         result.Warnings,
		Responses:        []SpeechModelResponseMetadata{result.Response},
		ProviderMetadata: providerMeta,
	}, nil
}

// detectAudioMediaType attempts to detect the media type from audio data bytes.
func detectAudioMediaType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// MP3: FF FB or FF F3 or FF F2 or ID3
	if data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mpeg"
	}
	if len(data) >= 3 && data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
		return "audio/mpeg"
	}
	// WAV: 52 49 46 46 ... 57 41 56 45
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav"
	}
	// OGG: 4F 67 67 53
	if data[0] == 0x4F && data[1] == 0x67 && data[2] == 0x67 && data[3] == 0x53 {
		return "audio/ogg"
	}
	// FLAC: 66 4C 61 43
	if data[0] == 0x66 && data[1] == 0x4C && data[2] == 0x61 && data[3] == 0x43 {
		return "audio/flac"
	}
	return ""
}
