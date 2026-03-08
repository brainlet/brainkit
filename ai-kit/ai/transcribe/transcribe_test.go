// Ported from: packages/ai/src/transcribe/transcribe.test.ts
package transcribe

import (
	"context"
	"reflect"
	"testing"
)

// mockTranscriptionModel is a mock implementation of TranscriptionModel for testing.
type mockTranscriptionModel struct {
	provider   string
	modelID    string
	doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

func (m *mockTranscriptionModel) Provider() string { return m.provider }
func (m *mockTranscriptionModel) ModelID() string  { return m.modelID }
func (m *mockTranscriptionModel) DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
	return m.doGenerate(ctx, opts)
}

func newMockTranscriptionModel(doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)) *mockTranscriptionModel {
	return &mockTranscriptionModel{
		provider:   "mock-provider",
		modelID:    "mock-model-id",
		doGenerate: doGenerate,
	}
}

var audioData = []byte{1, 2, 3, 4}

func ptrFloat64(v float64) *float64 { return &v }

var sampleSegments = []TranscriptionSegment{
	{StartSecond: 0, EndSecond: 2.5, Text: "This is a"},
	{StartSecond: 2.5, EndSecond: 4.0, Text: "sample transcript."},
}

func TestTranscribe_SendArgs(t *testing.T) {
	t.Run("should send args to doGenerate", func(t *testing.T) {
		var capturedOpts DoGenerateOptions

		model := newMockTranscriptionModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			capturedOpts = opts
			return &DoGenerateResult{
				Text:              "This is a sample transcript.",
				Segments:          sampleSegments,
				Language:          "en",
				DurationInSeconds: ptrFloat64(4.0),
				Warnings:          []Warning{},
				Response:          TranscriptionModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata:  map[string]map[string]any{},
			}, nil
		})

		_, err := Transcribe(context.Background(), TranscribeOptions{
			Model: model,
			Audio: audioData,
			Headers: map[string]string{
				"custom-request-header": "request-header-value",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(capturedOpts.Audio, audioData) {
			t.Errorf("expected audio data %v, got %v", audioData, capturedOpts.Audio)
		}
		if capturedOpts.Headers["custom-request-header"] != "request-header-value" {
			t.Errorf("expected custom header, got %v", capturedOpts.Headers)
		}
	})
}

func TestTranscribe_Warnings(t *testing.T) {
	t.Run("should return warnings", func(t *testing.T) {
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
		}

		model := newMockTranscriptionModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Text:     "This is a sample transcript.",
				Segments: sampleSegments,
				Language: "en",
				DurationInSeconds: ptrFloat64(4.0),
				Warnings: expectedWarnings,
				Response: TranscriptionModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: map[string]map[string]any{
					"test-provider": {"test-key": "test-value"},
				},
			}, nil
		})

		result, err := Transcribe(context.Background(), TranscribeOptions{
			Model: model,
			Audio: audioData,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})
}

func TestTranscribe_Transcript(t *testing.T) {
	t.Run("should return the transcript", func(t *testing.T) {
		model := newMockTranscriptionModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Text:              "This is a sample transcript.",
				Segments:          sampleSegments,
				Language:          "en",
				DurationInSeconds: ptrFloat64(4.0),
				Warnings:          []Warning{},
				Response:          TranscriptionModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata:  map[string]map[string]any{},
			}, nil
		})

		result, err := Transcribe(context.Background(), TranscribeOptions{
			Model: model,
			Audio: audioData,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Text != "This is a sample transcript." {
			t.Errorf("expected text %q, got %q", "This is a sample transcript.", result.Text)
		}
		if result.Language != "en" {
			t.Errorf("expected language en, got %q", result.Language)
		}
		if result.DurationInSeconds == nil || *result.DurationInSeconds != 4.0 {
			t.Errorf("expected duration 4.0, got %v", result.DurationInSeconds)
		}
		if !reflect.DeepEqual(result.Segments, sampleSegments) {
			t.Errorf("expected segments %v, got %v", sampleSegments, result.Segments)
		}
	})
}

func TestTranscribe_NoTranscriptError(t *testing.T) {
	t.Run("should return error when no transcript is returned", func(t *testing.T) {
		model := newMockTranscriptionModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Text:              "",
				Segments:          []TranscriptionSegment{},
				Language:          "en",
				DurationInSeconds: ptrFloat64(0),
				Warnings:          []Warning{},
				Response:          TranscriptionModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata:  map[string]map[string]any{},
			}, nil
		})

		_, err := Transcribe(context.Background(), TranscribeOptions{
			Model: model,
			Audio: audioData,
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestTranscribe_ResponseMetadata(t *testing.T) {
	t.Run("should return response metadata", func(t *testing.T) {
		testHeaders := map[string]string{"x-test": "value"}

		model := newMockTranscriptionModel(func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Text:              "This is a sample transcript.",
				Segments:          sampleSegments,
				Language:          "en",
				DurationInSeconds: ptrFloat64(4.0),
				Warnings:          []Warning{},
				Response: TranscriptionModelResponseMetadata{
					ModelID: "test-model",
					Headers: testHeaders,
				},
				ProviderMetadata: map[string]map[string]any{},
			}, nil
		})

		result, err := Transcribe(context.Background(), TranscribeOptions{
			Model: model,
			Audio: audioData,
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
