// Ported from: packages/openai/src/speech/openai-speech-model.test.ts
package openai

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

func createSpeechTestModel(baseURL string) *OpenAISpeechModel {
	return NewOpenAISpeechModel("tts-1", OpenAISpeechModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: "openai.speech",
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
		Internal: &OpenAISpeechModelInternal{
			CurrentDate: func() time.Time {
				return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		},
	})
}

func TestSpeechDoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass model and text", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, capture := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		_, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
			Ctx:  context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "tts-1" {
			t.Errorf("expected model 'tts-1', got %v", body["model"])
		}
		if body["input"] != "Hello from the AI SDK!" {
			t.Errorf("expected input 'Hello from the AI SDK!', got %v", body["input"])
		}
	})

	t.Run("should pass voice, speed, and response_format", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, capture := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/opus",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		voice := "nova"
		speed := 1.5
		outputFormat := "opus"
		_, err := model.DoGenerate(speechmodel.CallOptions{
			Text:         "Hello from the AI SDK!",
			Voice:        &voice,
			Speed:        &speed,
			OutputFormat: &outputFormat,
			Ctx:          context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "tts-1" {
			t.Errorf("expected model 'tts-1', got %v", body["model"])
		}
		if body["input"] != "Hello from the AI SDK!" {
			t.Errorf("expected input, got %v", body["input"])
		}
		if body["voice"] != "nova" {
			t.Errorf("expected voice 'nova', got %v", body["voice"])
		}
		if body["speed"] != 1.5 {
			t.Errorf("expected speed 1.5, got %v", body["speed"])
		}
		if body["response_format"] != "opus" {
			t.Errorf("expected response_format 'opus', got %v", body["response_format"])
		}
	})
}

func TestSpeechDoGenerate_AudioOutput(t *testing.T) {
	t.Run("should return audio data", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, _ := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		result, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
			Ctx:  context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		audioBytes, ok := result.Audio.(speechmodel.AudioDataBytes)
		if !ok {
			t.Fatalf("expected AudioDataBytes, got %T", result.Audio)
		}
		if len(audioBytes.Data) != 100 {
			t.Errorf("expected 100 bytes, got %d", len(audioBytes.Data))
		}
	})
}

func TestSpeechDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should include response data with timestamp, modelId and headers", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, _ := createBinaryTestServer(audioData, map[string]string{
			"Content-Type":          "audio/mp3",
			"X-Request-Id":         "test-request-id",
			"X-Ratelimit-Remaining": "123",
		})
		defer server.Close()

		testDate := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
		model := NewOpenAISpeechModel("tts-1", OpenAISpeechModelConfig{
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
			Internal: &OpenAISpeechModelInternal{
				CurrentDate: func() time.Time {
					return testDate
				},
			},
		})

		result, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
			Ctx:  context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "tts-1" {
			t.Errorf("expected model ID 'tts-1', got %q", result.Response.ModelID)
		}
		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.Headers["X-Request-Id"] != "test-request-id" {
			t.Errorf("expected X-Request-Id 'test-request-id', got %q", result.Response.Headers["X-Request-Id"])
		}
	})
}

func TestSpeechDoGenerate_Warnings(t *testing.T) {
	t.Run("should include empty warnings by default", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, _ := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		result, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
			Ctx:  context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should warn about unsupported language", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, _ := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		lang := "en"
		result, err := model.DoGenerate(speechmodel.CallOptions{
			Text:     "Hello from the AI SDK!",
			Language: &lang,
			Ctx:      context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
	})
}

func TestSpeechDoGenerate_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, capture := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		headerVal := "request-header-value"
		_, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
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

func TestSpeechDoGenerate_DefaultVoice(t *testing.T) {
	t.Run("should use alloy as default voice", func(t *testing.T) {
		audioData := make([]byte, 100)
		server, capture := createBinaryTestServer(audioData, map[string]string{
			"Content-Type": "audio/mp3",
		})
		defer server.Close()
		model := createSpeechTestModel(server.URL)

		_, err := model.DoGenerate(speechmodel.CallOptions{
			Text: "Hello from the AI SDK!",
			Ctx:  context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["voice"] != "alloy" {
			t.Errorf("expected default voice 'alloy', got %v", body["voice"])
		}
	})
}
