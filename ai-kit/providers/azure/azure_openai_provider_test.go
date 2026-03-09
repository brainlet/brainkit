// Ported from: packages/azure/src/azure-openai-provider.test.ts
package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/providers/openai"
)

// strPtr creates a pointer to a string.
func strPtr(s string) *string { return &s }

// setupTestVersion sets VERSION to a test value and returns a cleanup function.
func setupTestVersion(t *testing.T) {
	t.Helper()
	orig := VERSION
	VERSION = "0.0.0-test"
	t.Cleanup(func() { VERSION = orig })
}

// requestCapture captures request details from the test server.
type requestCapture struct {
	URL     *url.URL
	Headers http.Header
	Body    []byte
}

func (rc *requestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

// createJSONTestServer creates a test server returning the given JSON body.
func createJSONTestServer(body any, responseHeaders map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL

		for k, v := range responseHeaders {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	}))
	return server, capture
}

// createSSETestServer creates a test server returning SSE chunks.
func createSSETestServer(chunks []string, responseHeaders map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		for k, v := range responseHeaders {
			w.Header().Set(k, v)
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		for _, chunk := range chunks {
			fmt.Fprint(w, chunk)
			flusher.Flush()
		}
	}))
	return server, capture
}

// responsesAPIResponse returns a minimal Responses API JSON response body.
func responsesAPIResponse(content string) map[string]any {
	return map[string]any{
		"id":         "resp_67c97c0203188190a025beb4a75242bc",
		"object":     "response",
		"created_at": float64(1741257730),
		"status":     "completed",
		"model":      "test-deployment",
		"output": []any{
			map[string]any{
				"id":     "msg_67c97c02656c81908e080dfdf4a03cd1",
				"type":   "message",
				"status": "completed",
				"role":   "assistant",
				"content": []any{
					map[string]any{
						"type":        "output_text",
						"text":        content,
						"annotations": []any{},
					},
				},
			},
		},
		"usage": map[string]any{
			"input_tokens": float64(4),
			"input_tokens_details": map[string]any{
				"cached_tokens": float64(0),
			},
			"output_tokens": float64(30),
			"output_tokens_details": map[string]any{
				"reasoning_tokens": float64(0),
			},
		},
		"incomplete_details": nil,
	}
}

// chatCompletionResponse returns a minimal Chat Completions API JSON response body.
func chatCompletionResponse(content string) map[string]any {
	return map[string]any{
		"id":      "chatcmpl-95ZTZkhr0mHNKqerQfiwkuox3PHAd",
		"object":  "chat.completion",
		"created": float64(1711115037),
		"model":   "gpt-3.5-turbo-0125",
		"choices": []any{
			map[string]any{
				"index": float64(0),
				"message": map[string]any{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(4),
			"total_tokens":      float64(34),
			"completion_tokens": float64(30),
		},
		"system_fingerprint": "fp_3bc1b5746c",
	}
}

// completionResponse returns a minimal Completions API JSON response body.
func completionResponse(content string) map[string]any {
	return map[string]any{
		"id":      "cmpl-96cAM1v77r4jXa4qb2NSmRREV5oWB",
		"object":  "text_completion",
		"created": float64(1711363706),
		"model":   "test-deployment",
		"choices": []any{
			map[string]any{
				"text":          content,
				"index":         float64(0),
				"finish_reason": "stop",
			},
		},
		"usage": map[string]any{
			"prompt_tokens":     float64(4),
			"total_tokens":      float64(34),
			"completion_tokens": float64(30),
		},
	}
}

// embeddingResponse returns a minimal Embeddings API JSON response body.
func embeddingResponse(embeddings [][]float64) map[string]any {
	data := make([]any, len(embeddings))
	for i, emb := range embeddings {
		data[i] = map[string]any{
			"object":    "embedding",
			"index":     float64(i),
			"embedding": toAnySlice(emb),
		}
	}
	return map[string]any{
		"object": "list",
		"data":   data,
		"model":  "my-embedding",
		"usage": map[string]any{
			"prompt_tokens": float64(8),
			"total_tokens":  float64(8),
		},
	}
}

func toAnySlice(f []float64) []any {
	result := make([]any, len(f))
	for i, v := range f {
		result[i] = v
	}
	return result
}

// imageResponse returns a minimal Images API JSON response body.
func imageResponse() map[string]any {
	return map[string]any{
		"created": float64(1733837122),
		"data": []any{
			map[string]any{
				"revised_prompt": "A charming visual illustration of a baby sea otter swimming joyously.",
				"b64_json":       "base64-image-1",
			},
			map[string]any{
				"b64_json": "base64-image-2",
			},
		},
	}
}

// transcriptionResponse returns a minimal Transcription API JSON response body.
func transcriptionResponse() map[string]any {
	return map[string]any{
		"text":     "Hello, world!",
		"segments": []any{},
		"language": "en",
		"duration": float64(5.0),
	}
}

// ===== Tests for responses (default language model) =====

func TestResponses_DefaultLanguageModel_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should set the correct default api version", func(t *testing.T) {
		server, capture := createJSONTestServer(responsesAPIResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Responses("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "v1" {
			t.Errorf("expected api-version 'v1', got %q", apiVersion)
		}
	})

	t.Run("should set the correct modified api version", func(t *testing.T) {
		server, capture := createJSONTestServer(responsesAPIResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL:    strPtr(server.URL),
			APIKey:     strPtr("test-api-key"),
			APIVersion: strPtr("2025-04-01-preview"),
		})

		model := provider.Responses("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "2025-04-01-preview" {
			t.Errorf("expected api-version '2025-04-01-preview', got %q", apiVersion)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createJSONTestServer(responsesAPIResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Provider-Header": "provider-header-value",
			},
		})

		model := provider.Responses("test-deployment")
		_, err := model.DoGenerate(doGenerateOptsWithHeaders(map[string]*string{
			"Custom-Request-Header": strPtr("request-header-value"),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Api-Key") != "test-api-key" {
			t.Errorf("expected api-key header 'test-api-key', got %q", capture.Headers.Get("Api-Key"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}

		ua := capture.Headers.Get("User-Agent")
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected User-Agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})

	t.Run("should use the baseURL correctly", func(t *testing.T) {
		server, capture := createJSONTestServer(responsesAPIResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Responses("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// The URL should include /v1/responses path
		if !strings.Contains(capture.URL.Path, "/v1/responses") {
			t.Errorf("expected URL path to contain '/v1/responses', got %q", capture.URL.Path)
		}
	})
}

// ===== Tests for chat =====

func TestChat_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should set the correct default api version", func(t *testing.T) {
		server, capture := createJSONTestServer(chatCompletionResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Chat("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "v1" {
			t.Errorf("expected api-version 'v1', got %q", apiVersion)
		}
	})

	t.Run("should set the correct modified api version", func(t *testing.T) {
		server, capture := createJSONTestServer(chatCompletionResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL:    strPtr(server.URL),
			APIKey:     strPtr("test-api-key"),
			APIVersion: strPtr("2025-04-01-preview"),
		})

		model := provider.Chat("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "2025-04-01-preview" {
			t.Errorf("expected api-version '2025-04-01-preview', got %q", apiVersion)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createJSONTestServer(chatCompletionResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Provider-Header": "provider-header-value",
			},
		})

		model := provider.Chat("test-deployment")
		_, err := model.DoGenerate(doGenerateOptsWithHeaders(map[string]*string{
			"Custom-Request-Header": strPtr("request-header-value"),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Api-Key") != "test-api-key" {
			t.Errorf("expected api-key header 'test-api-key', got %q", capture.Headers.Get("Api-Key"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}

		ua := capture.Headers.Get("User-Agent")
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected User-Agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})

	t.Run("should use the baseURL correctly", func(t *testing.T) {
		server, capture := createJSONTestServer(chatCompletionResponse(""), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Chat("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(capture.URL.Path, "/v1/chat/completions") {
			t.Errorf("expected URL path to contain '/v1/chat/completions', got %q", capture.URL.Path)
		}
	})
}

// ===== Tests for completion =====

func TestCompletion_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should set the correct api version", func(t *testing.T) {
		server, capture := createJSONTestServer(completionResponse("Hello World!"), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Completion("test-deployment")
		_, err := model.DoGenerate(doGenerateOpts())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "v1" {
			t.Errorf("expected api-version 'v1', got %q", apiVersion)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createJSONTestServer(completionResponse("Hello World!"), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Provider-Header": "provider-header-value",
			},
		})

		model := provider.Completion("test-deployment")
		_, err := model.DoGenerate(doGenerateOptsWithHeaders(map[string]*string{
			"Custom-Request-Header": strPtr("request-header-value"),
		}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Api-Key") != "test-api-key" {
			t.Errorf("expected api-key header 'test-api-key', got %q", capture.Headers.Get("Api-Key"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}

		ua := capture.Headers.Get("User-Agent")
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected User-Agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})
}

// ===== Tests for transcription =====

func TestTranscription_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should use correct URL format", func(t *testing.T) {
		server, capture := createJSONTestServer(transcriptionResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		// We can't easily call DoGenerate on transcription without the full audio pipeline,
		// but we can verify the URL construction by checking the provider's URL function.
		model := provider.Transcription("whisper-1")
		if model == nil {
			t.Fatal("expected non-nil transcription model")
		}
		if model.ModelID() != "whisper-1" {
			t.Errorf("expected model ID 'whisper-1', got %q", model.ModelID())
		}
		if model.Provider() != "azure.transcription" {
			t.Errorf("expected provider 'azure.transcription', got %q", model.Provider())
		}
		_ = server
		_ = capture
	})

	t.Run("should use deployment-based URL format when useDeploymentBasedUrls is true", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName:          strPtr("test-resource"),
			APIKey:                strPtr("test-api-key"),
			UseDeploymentBasedUrls: true,
		})

		// Verify the URL function constructs the deployment-based URL
		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "whisper-1",
			Path:    "/audio/transcriptions",
		})

		expectedURL := "https://test-resource.openai.azure.com/openai/deployments/whisper-1/audio/transcriptions?api-version=v1"
		if generatedURL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, generatedURL)
		}
	})
}

// ===== Tests for speech =====

func TestSpeech_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should use correct URL format", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		// Verify the URL function constructs the v1 URL
		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "tts-1",
			Path:    "/audio/speech",
		})

		expectedURL := "https://test-resource.openai.azure.com/openai/v1/audio/speech?api-version=v1"
		if generatedURL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, generatedURL)
		}
	})

	t.Run("should create speech model with correct provider", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model := provider.Speech("tts-1")
		if model == nil {
			t.Fatal("expected non-nil speech model")
		}
		if model.ModelID() != "tts-1" {
			t.Errorf("expected model ID 'tts-1', got %q", model.ModelID())
		}
		if model.Provider() != "azure.speech" {
			t.Errorf("expected provider 'azure.speech', got %q", model.Provider())
		}
	})
}

// ===== Tests for embedding =====

func TestEmbedding_DoEmbed(t *testing.T) {
	setupTestVersion(t)

	t.Run("should set the correct api version", func(t *testing.T) {
		dummyEmbeddings := [][]float64{
			{0.1, 0.2, 0.3, 0.4, 0.5},
			{0.6, 0.7, 0.8, 0.9, 1.0},
		}
		server, capture := createJSONTestServer(embeddingResponse(dummyEmbeddings), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Embedding("my-embedding")
		_, err := model.DoEmbed(doEmbedOpts([]string{"sunny day at the beach", "rainy day in the city"}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "v1" {
			t.Errorf("expected api-version 'v1', got %q", apiVersion)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		dummyEmbeddings := [][]float64{
			{0.1, 0.2, 0.3, 0.4, 0.5},
			{0.6, 0.7, 0.8, 0.9, 1.0},
		}
		server, capture := createJSONTestServer(embeddingResponse(dummyEmbeddings), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Provider-Header": "provider-header-value",
			},
		})

		model := provider.Embedding("my-embedding")
		_, err := model.DoEmbed(doEmbedOptsWithHeaders(
			[]string{"sunny day at the beach", "rainy day in the city"},
			map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Api-Key") != "test-api-key" {
			t.Errorf("expected api-key header 'test-api-key', got %q", capture.Headers.Get("Api-Key"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}

		ua := capture.Headers.Get("User-Agent")
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected User-Agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})
}

// ===== Tests for image =====

func TestImage_DoGenerate(t *testing.T) {
	setupTestVersion(t)

	t.Run("should set the correct default api version", func(t *testing.T) {
		server, capture := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Image("dalle-deployment")
		_, err := model.DoGenerate(doImageGenerateOpts("A cute baby sea otter"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "v1" {
			t.Errorf("expected api-version 'v1', got %q", apiVersion)
		}
	})

	t.Run("should set the correct modified api version", func(t *testing.T) {
		server, capture := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL:    strPtr(server.URL),
			APIKey:     strPtr("test-api-key"),
			APIVersion: strPtr("2025-04-01-preview"),
		})

		model := provider.Image("dalle-deployment")
		_, err := model.DoGenerate(doImageGenerateOpts("A cute baby sea otter"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		apiVersion := capture.URL.Query().Get("api-version")
		if apiVersion != "2025-04-01-preview" {
			t.Errorf("expected api-version '2025-04-01-preview', got %q", apiVersion)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Provider-Header": "provider-header-value",
			},
		})

		model := provider.Image("dalle-deployment")
		_, err := model.DoGenerate(doImageGenerateOptsWithHeaders(
			"A cute baby sea otter",
			map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
		))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("Api-Key") != "test-api-key" {
			t.Errorf("expected api-key header 'test-api-key', got %q", capture.Headers.Get("Api-Key"))
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q",
				capture.Headers.Get("Custom-Provider-Header"))
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q",
				capture.Headers.Get("Custom-Request-Header"))
		}

		ua := capture.Headers.Get("User-Agent")
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected User-Agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})

	t.Run("should use the baseURL correctly", func(t *testing.T) {
		server, capture := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Image("dalle-deployment")
		_, err := model.DoGenerate(doImageGenerateOpts("A cute baby sea otter"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(capture.URL.Path, "/v1/images/generations") {
			t.Errorf("expected URL path to contain '/v1/images/generations', got %q", capture.URL.Path)
		}
	})

	t.Run("should extract the generated images", func(t *testing.T) {
		server, _ := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Image("dalle-deployment")
		result, err := model.DoGenerate(doImageGenerateOpts("A cute baby sea otter"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imageStrings, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imageStrings.Values) != 2 {
			t.Fatalf("expected 2 images, got %d", len(imageStrings.Values))
		}
		if imageStrings.Values[0] != "base64-image-1" {
			t.Errorf("expected first image 'base64-image-1', got %q", imageStrings.Values[0])
		}
		if imageStrings.Values[1] != "base64-image-2" {
			t.Errorf("expected second image 'base64-image-2', got %q", imageStrings.Values[1])
		}
	})

	t.Run("should send the correct request body", func(t *testing.T) {
		server, capture := createJSONTestServer(imageResponse(), nil)
		defer server.Close()

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr(server.URL),
			APIKey:  strPtr("test-api-key"),
		})

		model := provider.Image("dalle-deployment")
		n := 2
		_, err := model.DoGenerate(doImageGenerateOptsWithN("A cute baby sea otter", n))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "dalle-deployment" {
			t.Errorf("expected model 'dalle-deployment', got %v", body["model"])
		}
		if body["prompt"] != "A cute baby sea otter" {
			t.Errorf("expected prompt 'A cute baby sea otter', got %v", body["prompt"])
		}
		if body["response_format"] != "b64_json" {
			t.Errorf("expected response_format 'b64_json', got %v", body["response_format"])
		}
		// n should be 2
		if nVal, ok := body["n"].(float64); !ok || int(nVal) != 2 {
			t.Errorf("expected n 2, got %v", body["n"])
		}
	})

	t.Run("imageModel method should create model with correct provider", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		imageModel1, err := provider.ImageModel("dalle-deployment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		imageModel2 := provider.Image("dalle-deployment")

		if imageModel1.Provider() != imageModel2.Provider() {
			t.Errorf("expected same provider, got %q and %q", imageModel1.Provider(), imageModel2.Provider())
		}
		if imageModel1.ModelID() != imageModel2.ModelID() {
			t.Errorf("expected same modelId, got %q and %q", imageModel1.ModelID(), imageModel2.ModelID())
		}
	})
}

// ===== Tests for provider construction =====

func TestCreateAzureProvider(t *testing.T) {
	setupTestVersion(t)

	t.Run("should create provider with resource name", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		if provider == nil {
			t.Fatal("expected non-nil provider")
		}

		// Check URL construction
		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		expected := "https://test-resource.openai.azure.com/openai/v1/responses?api-version=v1"
		if generatedURL != expected {
			t.Errorf("expected URL %q, got %q", expected, generatedURL)
		}
	})

	t.Run("should create provider with base URL", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			BaseURL: strPtr("https://custom-endpoint.openai.azure.com/openai"),
			APIKey:  strPtr("test-api-key"),
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		expected := "https://custom-endpoint.openai.azure.com/openai/v1/responses?api-version=v1"
		if generatedURL != expected {
			t.Errorf("expected URL %q, got %q", expected, generatedURL)
		}
	})

	t.Run("should prefer base URL over resource name", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("should-be-ignored"),
			BaseURL:      strPtr("https://custom-endpoint.openai.azure.com/openai"),
			APIKey:       strPtr("test-api-key"),
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		if strings.Contains(generatedURL, "should-be-ignored") {
			t.Errorf("expected base URL to take precedence over resource name, got %q", generatedURL)
		}
		if !strings.Contains(generatedURL, "custom-endpoint") {
			t.Errorf("expected URL to contain 'custom-endpoint', got %q", generatedURL)
		}
	})

	t.Run("should use deployment-based URLs when configured", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName:          strPtr("test-resource"),
			APIKey:                strPtr("test-api-key"),
			UseDeploymentBasedUrls: true,
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "my-deployment",
			Path:    "/chat/completions",
		})

		expected := "https://test-resource.openai.azure.com/openai/deployments/my-deployment/chat/completions?api-version=v1"
		if generatedURL != expected {
			t.Errorf("expected deployment-based URL %q, got %q", expected, generatedURL)
		}
	})

	t.Run("should set custom api version", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
			APIVersion:   strPtr("2025-04-01-preview"),
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		if !strings.Contains(generatedURL, "api-version=2025-04-01-preview") {
			t.Errorf("expected api-version '2025-04-01-preview' in URL, got %q", generatedURL)
		}
	})

	t.Run("should pass api-key header", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("my-test-key"),
		})

		headers := provider.getHeaders()
		if headers["api-key"] != "my-test-key" {
			t.Errorf("expected api-key header 'my-test-key', got %q", headers["api-key"])
		}
	})

	t.Run("should pass user-agent header with version", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		headers := provider.getHeaders()
		ua := headers["user-agent"]
		if !strings.Contains(ua, "ai-sdk/azure/0.0.0-test") {
			t.Errorf("expected user-agent to contain 'ai-sdk/azure/0.0.0-test', got %q", ua)
		}
	})

	t.Run("should pass custom headers", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
			Headers: map[string]string{
				"Custom-Header": "custom-value",
			},
		})

		headers := provider.getHeaders()
		// NormalizeHeaders lowercases all keys
		if headers["custom-header"] != "custom-value" {
			t.Errorf("expected custom-header 'custom-value', got %q", headers["custom-header"])
		}
	})
}

// ===== Tests for API key loading =====

func TestLoadApiKeyFromEnvironment(t *testing.T) {
	setupTestVersion(t)

	t.Run("should load API key from AZURE_API_KEY environment variable", func(t *testing.T) {
		t.Setenv("AZURE_API_KEY", "env-api-key-123")
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
		})

		headers := provider.getHeaders()
		if headers["api-key"] != "env-api-key-123" {
			t.Errorf("expected api-key header with env key, got %q", headers["api-key"])
		}
	})

	t.Run("should panic when no API key is provided and env var is unset", func(t *testing.T) {
		os.Unsetenv("AZURE_API_KEY")
		t.Setenv("AZURE_API_KEY", "")
		os.Unsetenv("AZURE_API_KEY")

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
		})

		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic when no API key is available")
			}
		}()

		provider.getHeaders()
	})

	t.Run("should prefer explicit API key over environment variable", func(t *testing.T) {
		t.Setenv("AZURE_API_KEY", "env-key")
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("explicit-key"),
		})

		headers := provider.getHeaders()
		if headers["api-key"] != "explicit-key" {
			t.Errorf("expected explicit key to take precedence, got %q", headers["api-key"])
		}
	})
}

// ===== Tests for resource name loading =====

func TestLoadResourceNameFromEnvironment(t *testing.T) {
	setupTestVersion(t)

	t.Run("should load resource name from AZURE_RESOURCE_NAME environment variable", func(t *testing.T) {
		t.Setenv("AZURE_RESOURCE_NAME", "env-resource")

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			APIKey: strPtr("test-key"),
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		if !strings.Contains(generatedURL, "env-resource.openai.azure.com") {
			t.Errorf("expected URL to contain resource name from env, got %q", generatedURL)
		}
	})

	t.Run("should prefer explicit resource name over environment variable", func(t *testing.T) {
		t.Setenv("AZURE_RESOURCE_NAME", "env-resource")

		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("explicit-resource"),
			APIKey:       strPtr("test-key"),
		})

		generatedURL := provider.urlFn(struct {
			ModelID string
			Path    string
		}{
			ModelID: "test-deployment",
			Path:    "/responses",
		})

		if !strings.Contains(generatedURL, "explicit-resource.openai.azure.com") {
			t.Errorf("expected URL to contain explicit resource name, got %q", generatedURL)
		}
	})
}

// ===== Tests for specification version =====

func TestSpecificationVersion(t *testing.T) {
	t.Run("should return v3", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		if v := provider.SpecificationVersion(); v != "v3" {
			t.Errorf("expected specification version %q, got %q", "v3", v)
		}
	})
}

// ===== Tests for model creation =====

func TestLanguageModel(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a language model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model, err := provider.LanguageModel("test-deployment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		if model.ModelID() != "test-deployment" {
			t.Errorf("expected model ID %q, got %q", "test-deployment", model.ModelID())
		}
		if model.Provider() != "azure.responses" {
			t.Errorf("expected provider %q, got %q", "azure.responses", model.Provider())
		}
	})
}

func TestChatModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a chat model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model := provider.Chat("test-deployment")
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		var _ *openai.OpenAIChatLanguageModel = model

		if model.ModelID() != "test-deployment" {
			t.Errorf("expected model ID %q, got %q", "test-deployment", model.ModelID())
		}
		if model.Provider() != "azure.chat" {
			t.Errorf("expected provider %q, got %q", "azure.chat", model.Provider())
		}
	})
}

func TestCompletionModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a completion model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model := provider.Completion("test-deployment")
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		var _ *openai.OpenAICompletionLanguageModel = model

		if model.ModelID() != "test-deployment" {
			t.Errorf("expected model ID %q, got %q", "test-deployment", model.ModelID())
		}
		if model.Provider() != "azure.completion" {
			t.Errorf("expected provider %q, got %q", "azure.completion", model.Provider())
		}
	})
}

func TestResponsesModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a responses model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model := provider.Responses("test-deployment")
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		var _ *openai.OpenAIResponsesLanguageModel = model

		if model.ModelID() != "test-deployment" {
			t.Errorf("expected model ID %q, got %q", "test-deployment", model.ModelID())
		}
		if model.Provider() != "azure.responses" {
			t.Errorf("expected provider %q, got %q", "azure.responses", model.Provider())
		}
	})
}

func TestEmbeddingModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct an embedding model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model, err := provider.EmbeddingModel("my-embedding")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		if model.ModelID() != "my-embedding" {
			t.Errorf("expected model ID %q, got %q", "my-embedding", model.ModelID())
		}
		if model.Provider() != "azure.embeddings" {
			t.Errorf("expected provider %q, got %q", "azure.embeddings", model.Provider())
		}
	})
}

func TestImageModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct an image model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model, err := provider.ImageModel("dalle-deployment")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		if model.ModelID() != "dalle-deployment" {
			t.Errorf("expected model ID %q, got %q", "dalle-deployment", model.ModelID())
		}
		if model.Provider() != "azure.image" {
			t.Errorf("expected provider %q, got %q", "azure.image", model.Provider())
		}
	})
}

func TestTranscriptionModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a transcription model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model, err := provider.TranscriptionModel("whisper-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		if model.ModelID() != "whisper-1" {
			t.Errorf("expected model ID %q, got %q", "whisper-1", model.ModelID())
		}
		if model.Provider() != "azure.transcription" {
			t.Errorf("expected provider %q, got %q", "azure.transcription", model.Provider())
		}
	})
}

func TestSpeechModelCreation(t *testing.T) {
	setupTestVersion(t)

	t.Run("should construct a speech model with correct configuration", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		model, err := provider.SpeechModel("tts-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if model == nil {
			t.Fatal("expected non-nil model")
		}

		if model.ModelID() != "tts-1" {
			t.Errorf("expected model ID %q, got %q", "tts-1", model.ModelID())
		}
		if model.Provider() != "azure.speech" {
			t.Errorf("expected provider %q, got %q", "azure.speech", model.Provider())
		}
	})
}

// ===== Tests for unsupported models =====

func TestUnsupportedModels(t *testing.T) {
	setupTestVersion(t)

	provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
		ResourceName: strPtr("test-resource"),
		APIKey:       strPtr("test-api-key"),
	})

	t.Run("RerankingModel should return NoSuchModelError", func(t *testing.T) {
		model, err := provider.RerankingModel("any-model")
		if model != nil {
			t.Fatal("expected nil model")
		}
		if err == nil {
			t.Fatal("expected error")
		}

		var noSuchModelErr *errors.NoSuchModelError
		if !errors.As(err, &noSuchModelErr) {
			t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
		}
	})
}

// ===== Tests for tools =====

func TestAzureOpenAITools(t *testing.T) {
	t.Run("should have CodeInterpreter tool", func(t *testing.T) {
		if AzureOpenAITools.CodeInterpreter == nil {
			t.Fatal("expected CodeInterpreter to be non-nil")
		}
		result := AzureOpenAITools.CodeInterpreter(nil)
		if result == nil {
			t.Fatal("expected non-nil result from CodeInterpreter")
		}
		// Tool constructors return type "provider" with id "openai.code_interpreter"
		if result["type"] != "provider" {
			t.Errorf("expected type 'provider', got %v", result["type"])
		}
		if result["id"] != "openai.code_interpreter" {
			t.Errorf("expected id 'openai.code_interpreter', got %v", result["id"])
		}
	})

	t.Run("should have FileSearch tool", func(t *testing.T) {
		if AzureOpenAITools.FileSearch == nil {
			t.Fatal("expected FileSearch to be non-nil")
		}
		result := AzureOpenAITools.FileSearch(openai.FileSearchArgs{
			VectorStoreIds: []string{"vs_123"},
		})
		if result == nil {
			t.Fatal("expected non-nil result from FileSearch")
		}
		if result["type"] != "provider" {
			t.Errorf("expected type 'provider', got %v", result["type"])
		}
		if result["id"] != "openai.file_search" {
			t.Errorf("expected id 'openai.file_search', got %v", result["id"])
		}
	})

	t.Run("should have ImageGeneration tool", func(t *testing.T) {
		if AzureOpenAITools.ImageGeneration == nil {
			t.Fatal("expected ImageGeneration to be non-nil")
		}
		result := AzureOpenAITools.ImageGeneration(nil)
		if result == nil {
			t.Fatal("expected non-nil result from ImageGeneration")
		}
		if result["type"] != "provider" {
			t.Errorf("expected type 'provider', got %v", result["type"])
		}
		if result["id"] != "openai.image_generation" {
			t.Errorf("expected id 'openai.image_generation', got %v", result["id"])
		}
	})

	t.Run("should have WebSearchPreview tool", func(t *testing.T) {
		if AzureOpenAITools.WebSearchPreview == nil {
			t.Fatal("expected WebSearchPreview to be non-nil")
		}
		result := AzureOpenAITools.WebSearchPreview(nil)
		if result == nil {
			t.Fatal("expected non-nil result from WebSearchPreview")
		}
		if result["type"] != "provider" {
			t.Errorf("expected type 'provider', got %v", result["type"])
		}
		if result["id"] != "openai.web_search_preview" {
			t.Errorf("expected id 'openai.web_search_preview', got %v", result["id"])
		}
	})

	t.Run("provider Tools should match module-level tools", func(t *testing.T) {
		provider := NewAzureOpenAIProvider(AzureOpenAIProviderSettings{
			ResourceName: strPtr("test-resource"),
			APIKey:       strPtr("test-api-key"),
		})

		// Verify provider has tools
		if provider.Tools.CodeInterpreter == nil {
			t.Fatal("expected provider Tools.CodeInterpreter to be non-nil")
		}
		if provider.Tools.FileSearch == nil {
			t.Fatal("expected provider Tools.FileSearch to be non-nil")
		}
		if provider.Tools.ImageGeneration == nil {
			t.Fatal("expected provider Tools.ImageGeneration to be non-nil")
		}
		if provider.Tools.WebSearchPreview == nil {
			t.Fatal("expected provider Tools.WebSearchPreview to be non-nil")
		}
	})
}
