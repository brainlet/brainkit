// Ported from: packages/ai/src/transcribe/transcribe.ts
package transcribe

import (
	"context"
	"fmt"
)

// TranscriptionModel is the interface for transcription models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type TranscriptionModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoGenerate performs the transcription operation.
	DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

// DoGenerateOptions are the options passed to TranscriptionModel.DoGenerate.
type DoGenerateOptions struct {
	Audio           []byte
	MediaType       string
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// DoGenerateResult is the result from TranscriptionModel.DoGenerate.
type DoGenerateResult struct {
	Text              string
	Segments          []TranscriptionSegment
	Language          string
	DurationInSeconds *float64
	Warnings          []Warning
	Response          TranscriptionModelResponseMetadata
	ProviderMetadata  map[string]map[string]any
}

// DownloadFunc is a function that downloads data from a URL.
type DownloadFunc func(ctx context.Context, url string) (data []byte, mediaType string, err error)

// TranscribeOptions are the options for the Transcribe function.
type TranscribeOptions struct {
	// Model is the transcription model to use.
	Model TranscriptionModel

	// Audio is the audio data to transcribe.
	Audio []byte

	// AudioURL is an alternative to Audio, providing a URL to download the audio from.
	AudioURL string

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any

	// Download is a custom download function for fetching audio from URLs.
	Download DownloadFunc
}

// Transcribe generates transcripts using a transcription model.
func Transcribe(ctx context.Context, opts TranscribeOptions) (*TranscriptionResult, error) {
	model := opts.Model

	providerOptions := opts.ProviderOptions
	if providerOptions == nil {
		providerOptions = map[string]map[string]any{}
	}

	audioData := opts.Audio
	if audioData == nil && opts.AudioURL != "" && opts.Download != nil {
		data, _, err := opts.Download(ctx, opts.AudioURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download audio: %w", err)
		}
		audioData = data
	}

	mediaType := detectAudioMediaType(audioData)
	if mediaType == "" {
		mediaType = "audio/wav"
	}

	result, err := model.DoGenerate(ctx, DoGenerateOptions{
		Audio:           audioData,
		MediaType:       mediaType,
		Headers:         opts.Headers,
		ProviderOptions: providerOptions,
	})
	if err != nil {
		return nil, err
	}

	if result.Text == "" {
		return nil, fmt.Errorf("no transcript generated")
	}

	providerMeta := result.ProviderMetadata
	if providerMeta == nil {
		providerMeta = map[string]map[string]any{}
	}

	return &TranscriptionResult{
		Text:              result.Text,
		Segments:          result.Segments,
		Language:          result.Language,
		DurationInSeconds: result.DurationInSeconds,
		Warnings:          result.Warnings,
		Responses:         []TranscriptionModelResponseMetadata{result.Response},
		ProviderMetadata:  providerMeta,
	}, nil
}

// detectAudioMediaType attempts to detect the media type from audio data bytes.
func detectAudioMediaType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// WAV: 52 49 46 46 ... 57 41 56 45
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x41 && data[10] == 0x56 && data[11] == 0x45 {
		return "audio/wav"
	}
	// MP3: FF FB or FF F3 or FF F2 or ID3
	if data[0] == 0xFF && (data[1]&0xE0) == 0xE0 {
		return "audio/mpeg"
	}
	if len(data) >= 3 && data[0] == 0x49 && data[1] == 0x44 && data[2] == 0x33 {
		return "audio/mpeg"
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
