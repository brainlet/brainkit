// Ported from: packages/groq/src/groq-transcription-model.test.ts
package groq

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// transcriptionFixture returns a standard Groq transcription API response.
func transcriptionFixture() map[string]any {
	return map[string]any{
		"task":     "transcribe",
		"language": "English",
		"duration": 2.5,
		"text":     "Hello world!",
		"segments": []any{
			map[string]any{
				"id":                0,
				"seek":              0,
				"start":            float64(0),
				"end":              2.48,
				"text":             "Hello world!",
				"tokens":           []any{float64(50365), float64(2425), float64(490), float64(264)},
				"temperature":      float64(0),
				"avg_logprob":      -0.29010406,
				"compression_ratio": 0.7777778,
				"no_speech_prob":   0.032802984,
			},
		},
		"x_groq": map[string]any{"id": "req_01jrh9nn61f24rydqq1r4b3yg5"},
	}
}

// transcriptionRequestCapture captures multipart form data from HTTP requests.
type transcriptionRequestCapture struct {
	Headers    http.Header
	FormFields map[string]string
	FormFiles  map[string][]byte
}

// createTranscriptionJSONTestServer creates a test server that responds with JSON
// and captures multipart form data from the request.
func createTranscriptionJSONTestServer(body any, respHeaders map[string]string) (*httptest.Server, *transcriptionRequestCapture) {
	capture := &transcriptionRequestCapture{
		FormFields: make(map[string]string),
		FormFiles:  make(map[string][]byte),
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capture.Headers = r.Header

		// Parse multipart form data
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "multipart/form-data") {
			_, params, err := mime.ParseMediaType(contentType)
			if err == nil {
				reader := multipart.NewReader(r.Body, params["boundary"])
				for {
					part, err := reader.NextPart()
					if err != nil {
						break
					}
					data, _ := io.ReadAll(part)
					if part.FileName() != "" {
						capture.FormFiles[part.FormName()] = data
					} else {
						capture.FormFields[part.FormName()] = string(data)
					}
					part.Close()
				}
			}
		}

		for k, v := range respHeaders {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

// createTestTranscriptionModel creates a GroqTranscriptionModel targeting a test server.
func createTestTranscriptionModel(baseURL string) *GroqTranscriptionModel {
	return NewGroqTranscriptionModel("whisper-large-v3-turbo", GroqTranscriptionModelConfig{
		GroqConfig: GroqConfig{
			Provider: "groq.transcription",
			URL: func(_ string, path string) string {
				return baseURL + path
			},
			Headers: func() map[string]string {
				return map[string]string{
					"authorization": "Bearer test-api-key",
				}
			},
		},
	})
}

// testAudioData is simple test audio bytes for transcription tests.
var testAudioData = []byte{0x52, 0x49, 0x46, 0x46} // "RIFF" header bytes

func TestGroqTranscriptionModel_PassModel(t *testing.T) {
	t.Run("should pass the model", func(t *testing.T) {
		server, capture := createTranscriptionJSONTestServer(transcriptionFixture(), nil)
		defer server.Close()

		model := createTestTranscriptionModel(server.URL)

		_, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.FormFields["model"] != "whisper-large-v3-turbo" {
			t.Errorf("expected model 'whisper-large-v3-turbo', got %q", capture.FormFields["model"])
		}
	})
}

func TestGroqTranscriptionModel_PassHeaders(t *testing.T) {
	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createTranscriptionJSONTestServer(transcriptionFixture(), nil)
		defer server.Close()

		model := NewGroqTranscriptionModel("whisper-large-v3-turbo", GroqTranscriptionModelConfig{
			GroqConfig: GroqConfig{
				Provider: "groq.transcription",
				URL: func(_ string, path string) string {
					return server.URL + path
				},
				Headers: func() map[string]string {
					return map[string]string{
						"authorization":          "Bearer test-api-key",
						"Custom-Provider-Header": "provider-header-value",
					}
				},
			},
		})

		reqHeader := "request-header-value"
		_, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
			Headers: map[string]*string{
				"Custom-Request-Header": &reqHeader,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Authorization") != "Bearer test-api-key" {
			t.Errorf("expected authorization 'Bearer test-api-key', got %q", capture.Headers.Get("Authorization"))
		}
		ct := capture.Headers.Get("Content-Type")
		if !strings.Contains(ct, "multipart/form-data") {
			t.Errorf("expected Content-Type to contain 'multipart/form-data', got %q", ct)
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q", capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q", capture.Headers.Get("Custom-Request-Header"))
		}
	})
}

func TestGroqTranscriptionModel_ExtractText(t *testing.T) {
	t.Run("should extract the transcription text", func(t *testing.T) {
		server, _ := createTranscriptionJSONTestServer(transcriptionFixture(), nil)
		defer server.Close()

		model := createTestTranscriptionModel(server.URL)

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Text != "Hello world!" {
			t.Errorf("expected text 'Hello world!', got %q", result.Text)
		}
	})
}

func TestGroqTranscriptionModel_ResponseMetadata(t *testing.T) {
	t.Run("should include response data with timestamp, modelId and headers", func(t *testing.T) {
		server, _ := createTranscriptionJSONTestServer(transcriptionFixture(), map[string]string{
			"x-request-id":          "test-request-id",
			"x-ratelimit-remaining": "123",
		})
		defer server.Close()

		testDate := time.Unix(0, 0)
		model := NewGroqTranscriptionModel("whisper-large-v3-turbo", GroqTranscriptionModelConfig{
			GroqConfig: GroqConfig{
				Provider: "test-provider",
				URL: func(_ string, _ string) string {
					return server.URL + "/audio/transcriptions"
				},
				Headers: func() map[string]string {
					return map[string]string{}
				},
			},
			Internal: &GroqTranscriptionModelInternal{
				CurrentDate: func() time.Time { return testDate },
			},
		})

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.ModelID != "whisper-large-v3-turbo" {
			t.Errorf("expected modelId 'whisper-large-v3-turbo', got %q", result.Response.ModelID)
		}
		if result.Response.Headers["X-Request-Id"] != "test-request-id" {
			t.Errorf("expected X-Request-Id 'test-request-id', got %q", result.Response.Headers["X-Request-Id"])
		}
		if result.Response.Headers["X-Ratelimit-Remaining"] != "123" {
			t.Errorf("expected X-Ratelimit-Remaining '123', got %q", result.Response.Headers["X-Ratelimit-Remaining"])
		}
	})
}

func TestGroqTranscriptionModel_CustomDate(t *testing.T) {
	t.Run("should use real date when no custom date provider is specified", func(t *testing.T) {
		server, _ := createTranscriptionJSONTestServer(transcriptionFixture(), nil)
		defer server.Close()

		testDate := time.Unix(0, 0)
		model := NewGroqTranscriptionModel("whisper-large-v3-turbo", GroqTranscriptionModelConfig{
			GroqConfig: GroqConfig{
				Provider: "test-provider",
				URL: func(_ string, _ string) string {
					return server.URL + "/audio/transcriptions"
				},
				Headers: func() map[string]string {
					return map[string]string{}
				},
			},
			Internal: &GroqTranscriptionModelInternal{
				CurrentDate: func() time.Time { return testDate },
			},
		})

		result, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.Timestamp.Unix() != testDate.Unix() {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.ModelID != "whisper-large-v3-turbo" {
			t.Errorf("expected modelId 'whisper-large-v3-turbo', got %q", result.Response.ModelID)
		}
	})
}

func TestGroqTranscriptionModel_ProviderOptions(t *testing.T) {
	t.Run("should correctly pass provider options when they are an array", func(t *testing.T) {
		server, capture := createTranscriptionJSONTestServer(transcriptionFixture(), nil)
		defer server.Close()

		model := createTestTranscriptionModel(server.URL)

		_, err := model.DoGenerate(transcriptionmodel.CallOptions{
			Audio:     transcriptionmodel.AudioDataBytes{Data: testAudioData},
			MediaType: "audio/wav",
			ProviderOptions: map[string]map[string]any{
				"groq": {
					"timestampGranularities": []string{"segment"},
					"responseFormat":         "verbose_json",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.FormFields["timestamp_granularities[]"] != "segment" {
			t.Errorf("expected timestamp_granularities[] 'segment', got %q", capture.FormFields["timestamp_granularities[]"])
		}
		if capture.FormFields["response_format"] != "verbose_json" {
			t.Errorf("expected response_format 'verbose_json', got %q", capture.FormFields["response_format"])
		}
	})
}
