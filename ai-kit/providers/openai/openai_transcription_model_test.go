// Ported from: packages/openai/src/transcription/openai-transcription-model.test.ts
package openai

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

func createTranscriptionTestModel(baseURL string) *OpenAITranscriptionModel {
	return NewOpenAITranscriptionModel("whisper-1", OpenAITranscriptionModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: "openai.transcription",
			URL: func(options struct {
				ModelID string
				Path    string
			}) string {
				return baseURL + options.Path
			},
			Headers: func() map[string]string {
				return map[string]string{
					"Authorization": "Bearer test-api-key",
					"Content-Type":  "application/json",
				}
			},
		},
		Internal: &OpenAITranscriptionModelInternal{
			CurrentDate: func() time.Time {
				return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		},
	})
}

func transcriptionFixture() map[string]any {
	return map[string]any{
		"task": "transcribe",
		"text": "Galileo was an American robotic space program that studied the planet Jupiter and its moons, as well as several other solar system bodies.",
		"language": "english",
		"duration": float64(5.0),
	}
}

func TestTranscriptionDoGenerate_TextExtraction(t *testing.T) {
	t.Run("should extract transcription text", func(t *testing.T) {
		server, _ := createFormDataTestServer(transcriptionFixture(), nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := "Galileo was an American robotic space program that studied the planet Jupiter and its moons, as well as several other solar system bodies."
		if result.Text != expected {
			t.Errorf("expected %q, got %q", expected, result.Text)
		}
	})
}

func TestTranscriptionDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should include response data with timestamp, modelId and headers", func(t *testing.T) {
		server, _ := createFormDataTestServer(transcriptionFixture(), map[string]string{
			"X-Request-Id":          "test-request-id",
			"X-Ratelimit-Remaining": "123",
		})
		defer server.Close()

		testDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		model := NewOpenAITranscriptionModel("whisper-1", OpenAITranscriptionModelConfig{
			OpenAIConfig: OpenAIConfig{
				Provider: "test-provider",
				URL: func(options struct {
					ModelID string
					Path    string
				}) string {
					return server.URL + options.Path
				},
				Headers: func() map[string]string {
					return map[string]string{}
				},
			},
			Internal: &OpenAITranscriptionModelInternal{
				CurrentDate: func() time.Time {
					return testDate
				},
			},
		})

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "whisper-1" {
			t.Errorf("expected model ID 'whisper-1', got %q", result.Response.ModelID)
		}
		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
	})
}

func TestTranscriptionDoGenerate_MinimalResponse(t *testing.T) {
	t.Run("should work when no words, language, or duration are returned", func(t *testing.T) {
		fixture := map[string]any{
			"task": "transcribe",
			"text": "Hello from the Vercel AI SDK!",
		}
		server, _ := createFormDataTestServer(fixture, nil)
		defer server.Close()

		testDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		model := NewOpenAITranscriptionModel("whisper-1", OpenAITranscriptionModelConfig{
			OpenAIConfig: OpenAIConfig{
				Provider: "test-provider",
				URL: func(options struct {
					ModelID string
					Path    string
				}) string {
					return server.URL + options.Path
				},
				Headers: func() map[string]string {
					return map[string]string{}
				},
			},
			Internal: &OpenAITranscriptionModelInternal{
				CurrentDate: func() time.Time {
					return testDate
				},
			},
		})

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Text != "Hello from the Vercel AI SDK!" {
			t.Errorf("expected 'Hello from the Vercel AI SDK!', got %q", result.Text)
		}
		if result.Language != nil {
			t.Errorf("expected nil language, got %v", result.Language)
		}
		if result.DurationInSeconds != nil {
			t.Errorf("expected nil duration, got %v", result.DurationInSeconds)
		}
		if len(result.Segments) != 0 {
			t.Errorf("expected 0 segments, got %d", len(result.Segments))
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestTranscriptionDoGenerate_Segments(t *testing.T) {
	t.Run("should parse segments when provided in response", func(t *testing.T) {
		fixture := map[string]any{
			"task": "transcribe",
			"text": "Hello world. How are you?",
			"segments": []any{
				map[string]any{
					"id":                float64(0),
					"seek":              float64(0),
					"start":             float64(0.0),
					"end":               float64(2.5),
					"text":              "Hello world.",
					"tokens":            []any{float64(1234), float64(5678)},
					"temperature":       float64(0.0),
					"avg_logprob":       float64(-0.5),
					"compression_ratio": float64(1.2),
					"no_speech_prob":    float64(0.1),
				},
				map[string]any{
					"id":                float64(1),
					"seek":              float64(250),
					"start":             float64(2.5),
					"end":               float64(5.0),
					"text":              " How are you?",
					"tokens":            []any{float64(9012), float64(3456)},
					"temperature":       float64(0.0),
					"avg_logprob":       float64(-0.6),
					"compression_ratio": float64(1.1),
					"no_speech_prob":    float64(0.05),
				},
			},
			"language": "english",
			"duration": float64(5.0),
		}
		server, _ := createFormDataTestServer(fixture, nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Segments) != 2 {
			t.Fatalf("expected 2 segments, got %d", len(result.Segments))
		}

		if result.Segments[0].Text != "Hello world." {
			t.Errorf("expected first segment text 'Hello world.', got %q", result.Segments[0].Text)
		}
		if result.Segments[0].StartSecond != 0.0 {
			t.Errorf("expected first segment start 0.0, got %v", result.Segments[0].StartSecond)
		}
		if result.Segments[0].EndSecond != 2.5 {
			t.Errorf("expected first segment end 2.5, got %v", result.Segments[0].EndSecond)
		}

		if result.Segments[1].Text != " How are you?" {
			t.Errorf("expected second segment text ' How are you?', got %q", result.Segments[1].Text)
		}
		if result.Segments[1].StartSecond != 2.5 {
			t.Errorf("expected second segment start 2.5, got %v", result.Segments[1].StartSecond)
		}
		if result.Segments[1].EndSecond != 5.0 {
			t.Errorf("expected second segment end 5.0, got %v", result.Segments[1].EndSecond)
		}

		if result.Text != "Hello world. How are you?" {
			t.Errorf("expected text 'Hello world. How are you?', got %q", result.Text)
		}
		if result.DurationInSeconds == nil || *result.DurationInSeconds != 5.0 {
			t.Errorf("expected duration 5.0, got %v", result.DurationInSeconds)
		}
	})

	t.Run("should fallback to words when segments are not available", func(t *testing.T) {
		fixture := map[string]any{
			"task": "transcribe",
			"text": "Hello world",
			"words": []any{
				map[string]any{
					"word":  "Hello",
					"start": float64(0.0),
					"end":   float64(1.0),
				},
				map[string]any{
					"word":  "world",
					"start": float64(1.0),
					"end":   float64(2.0),
				},
			},
			"language": "english",
			"duration": float64(2.0),
		}
		server, _ := createFormDataTestServer(fixture, nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Segments) != 2 {
			t.Fatalf("expected 2 segments, got %d", len(result.Segments))
		}
		if result.Segments[0].Text != "Hello" {
			t.Errorf("expected first segment 'Hello', got %q", result.Segments[0].Text)
		}
		if result.Segments[0].StartSecond != 0.0 {
			t.Errorf("expected start 0.0, got %v", result.Segments[0].StartSecond)
		}
		if result.Segments[0].EndSecond != 1.0 {
			t.Errorf("expected end 1.0, got %v", result.Segments[0].EndSecond)
		}
		if result.Segments[1].Text != "world" {
			t.Errorf("expected second segment 'world', got %q", result.Segments[1].Text)
		}
	})

	t.Run("should handle empty segments array", func(t *testing.T) {
		fixture := map[string]any{
			"task":     "transcribe",
			"text":     "Hello world",
			"segments": []any{},
			"language": "english",
			"duration": float64(2.0),
		}
		server, _ := createFormDataTestServer(fixture, nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Segments) != 0 {
			t.Errorf("expected 0 segments, got %d", len(result.Segments))
		}
		if result.Text != "Hello world" {
			t.Errorf("expected 'Hello world', got %q", result.Text)
		}
	})

	t.Run("should handle segments with missing optional fields", func(t *testing.T) {
		fixture := map[string]any{
			"task": "transcribe",
			"text": "Test",
			"segments": []any{
				map[string]any{
					"id":                float64(0),
					"seek":              float64(0),
					"start":             float64(0.0),
					"end":               float64(1.0),
					"text":              "Test",
					"tokens":            []any{float64(1234)},
					"temperature":       float64(0.0),
					"avg_logprob":       float64(-0.5),
					"compression_ratio": float64(1.0),
					"no_speech_prob":    float64(0.1),
				},
			},
		}
		server, _ := createFormDataTestServer(fixture, nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Segments) != 1 {
			t.Fatalf("expected 1 segment, got %d", len(result.Segments))
		}
		if result.Segments[0].Text != "Test" {
			t.Errorf("expected segment text 'Test', got %q", result.Segments[0].Text)
		}
		if result.Language != nil {
			t.Errorf("expected nil language, got %v", result.Language)
		}
		if result.DurationInSeconds != nil {
			t.Errorf("expected nil duration, got %v", result.DurationInSeconds)
		}
	})
}

func TestTranscriptionDoGenerate_Language(t *testing.T) {
	t.Run("should map language name to ISO-639-1 code", func(t *testing.T) {
		server, _ := createFormDataTestServer(transcriptionFixture(), nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Ctx:       context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Language == nil || *result.Language != "en" {
			t.Errorf("expected language 'en', got %v", result.Language)
		}
	})
}

func TestTranscriptionDoGenerate_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createFormDataTestServer(transcriptionFixture(), nil)
		defer server.Close()
		model := createTranscriptionTestModel(server.URL)

		headerVal := "request-header-value"
		_, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: make([]byte, 100)},
			MediaType: "audio/wav",
			Headers: map[string]*string{
				"Custom-Request-Header": &headerVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header, got %q", capture.Headers.Get("Custom-Request-Header"))
		}
	})
}
