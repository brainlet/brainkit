// Ported from: packages/ai/src/generate-speech/generate-speech.test.ts
package generatespeech

import (
	"context"
	"reflect"
	"testing"
)

// mockSpeechModel is a mock implementation of SpeechModel for testing.
type mockSpeechModel struct {
	provider   string
	modelID    string
	doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

func (m *mockSpeechModel) Provider() string { return m.provider }
func (m *mockSpeechModel) ModelID() string  { return m.modelID }
func (m *mockSpeechModel) DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
	return m.doGenerate(ctx, opts)
}

func newMockSpeechModel(doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)) *mockSpeechModel {
	return &mockSpeechModel{
		provider:   "mock-provider",
		modelID:    "mock-model-id",
		doGenerate: doGenerate,
	}
}

var sampleAudio = []byte{1, 2, 3, 4}
var sampleText = "This is a sample text to convert to speech."

func TestGenerateSpeech_SendArgs(t *testing.T) {
	t.Run("should send args to doGenerate", func(t *testing.T) {
		var capturedOpts DoGenerateOptions

		model := newMockSpeechModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			capturedOpts = opts
			return &DoGenerateResult{
				Audio:            sampleAudio,
				Warnings:         []Warning{},
				Response:         SpeechModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: map[string]map[string]any{},
			}, nil
		})

		_, err := GenerateSpeech(context.Background(), GenerateSpeechOptions{
			Model: model,
			Text:  sampleText,
			Voice: "test-voice",
			Headers: map[string]string{
				"custom-request-header": "request-header-value",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedOpts.Text != sampleText {
			t.Errorf("expected text %q, got %q", sampleText, capturedOpts.Text)
		}
		if capturedOpts.Voice != "test-voice" {
			t.Errorf("expected voice test-voice, got %q", capturedOpts.Voice)
		}
		if capturedOpts.Headers["custom-request-header"] != "request-header-value" {
			t.Errorf("expected custom header, got %v", capturedOpts.Headers)
		}
	})
}

func TestGenerateSpeech_Warnings(t *testing.T) {
	t.Run("should return warnings", func(t *testing.T) {
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
		}

		model := newMockSpeechModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Audio:    sampleAudio,
				Warnings: expectedWarnings,
				Response: SpeechModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: map[string]map[string]any{
					"test-provider": {"test-key": "test-value"},
				},
			}, nil
		})

		result, err := GenerateSpeech(context.Background(), GenerateSpeechOptions{
			Model: model,
			Text:  sampleText,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})
}

func TestGenerateSpeech_AudioData(t *testing.T) {
	t.Run("should return the audio data", func(t *testing.T) {
		model := newMockSpeechModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Audio:            sampleAudio,
				Warnings:         []Warning{},
				Response:         SpeechModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: map[string]map[string]any{},
			}, nil
		})

		result, err := GenerateSpeech(context.Background(), GenerateSpeechOptions{
			Model: model,
			Text:  sampleText,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Audio.Data, sampleAudio) {
			t.Errorf("expected audio data %v, got %v", sampleAudio, result.Audio.Data)
		}
	})
}

func TestGenerateSpeech_NoAudioError(t *testing.T) {
	t.Run("should return error when no audio is returned", func(t *testing.T) {
		model := newMockSpeechModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Audio:            []byte{},
				Warnings:         []Warning{},
				Response:         SpeechModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: map[string]map[string]any{},
			}, nil
		})

		_, err := GenerateSpeech(context.Background(), GenerateSpeechOptions{
			Model: model,
			Text:  sampleText,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGenerateSpeech_ResponseMetadata(t *testing.T) {
	t.Run("should return response metadata", func(t *testing.T) {
		testHeaders := map[string]string{"x-test": "value"}

		model := newMockSpeechModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Audio:    sampleAudio,
				Warnings: []Warning{},
				Response: SpeechModelResponseMetadata{
					ModelID: "test-model",
					Headers: testHeaders,
				},
				ProviderMetadata: map[string]map[string]any{},
			}, nil
		})

		result, err := GenerateSpeech(context.Background(), GenerateSpeechOptions{
			Model: model,
			Text:  sampleText,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Responses) != 1 {
			t.Fatalf("expected 1 response, got %d", len(result.Responses))
		}
		if result.Responses[0].ModelID != "test-model" {
			t.Errorf("expected modelId test-model, got %q", result.Responses[0].ModelID)
		}
		if !reflect.DeepEqual(result.Responses[0].Headers, testHeaders) {
			t.Errorf("expected headers %v, got %v", testHeaders, result.Responses[0].Headers)
		}
	})
}
