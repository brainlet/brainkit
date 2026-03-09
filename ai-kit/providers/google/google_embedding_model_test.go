// Ported from: packages/google/src/google-generative-ai-embedding-model.test.ts
package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

var dummyEmbeddings = [][]float64{
	{0.1, 0.2, 0.3, 0.4, 0.5},
	{0.6, 0.7, 0.8, 0.9, 1.0},
}

var embeddingTestValues = []string{"sunny day at the beach", "rainy day in the city"}

type embeddingRequestCapture struct {
	Body    []byte
	Headers http.Header
	URL     string
}

func (rc *embeddingRequestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

func createBatchEmbeddingServer(embeddings [][]float64, headers map[string]string) (*httptest.Server, *embeddingRequestCapture) {
	capture := &embeddingRequestCapture{}
	embeddingObjs := make([]map[string]any, len(embeddings))
	for i, e := range embeddings {
		embeddingObjs[i] = map[string]any{"values": e}
	}
	responseBody := map[string]any{"embeddings": embeddingObjs}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL.String()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}))
	return server, capture
}

func createSingleEmbeddingServer(embedding []float64, headers map[string]string) (*httptest.Server, *embeddingRequestCapture) {
	capture := &embeddingRequestCapture{}
	responseBody := map[string]any{
		"embedding": map[string]any{"values": embedding},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		capture.URL = r.URL.String()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}))
	return server, capture
}

func createTestEmbeddingModel(baseURL string) *GoogleEmbeddingModel {
	return NewGoogleEmbeddingModel("gemini-embedding-001", GoogleEmbeddingModelConfig{
		Provider: "google.generative-ai",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return map[string]string{
				"x-goog-api-key": "test-api-key",
			}
		},
	})
}

func TestGoogleEmbeddingModel(t *testing.T) {
	t.Run("should extract embedding", func(t *testing.T) {
		server, _ := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Embeddings) != 2 {
			t.Fatalf("expected 2 embeddings, got %d", len(result.Embeddings))
		}
		if len(result.Embeddings[0]) != 5 {
			t.Errorf("expected 5 values in first embedding, got %d", len(result.Embeddings[0]))
		}
		if result.Embeddings[0][0] != 0.1 {
			t.Errorf("expected first value 0.1, got %f", result.Embeddings[0][0])
		}
		if result.Embeddings[1][4] != 1.0 {
			t.Errorf("expected last value 1.0, got %f", result.Embeddings[1][4])
		}
	})

	t.Run("should expose the raw response", func(t *testing.T) {
		server, _ := createBatchEmbeddingServer(dummyEmbeddings, map[string]string{
			"test-header": "test-value",
		})
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Response == nil {
			t.Fatal("expected response")
		}
		if result.Response.Headers["Test-Header"] != "test-value" {
			t.Errorf("expected test-header, got headers: %v", result.Response.Headers)
		}
	})

	t.Run("should pass the model and the values", func(t *testing.T) {
		server, capture := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body := capture.BodyJSON()
		requests, ok := body["requests"].([]any)
		if !ok {
			t.Fatalf("expected requests array, got %T", body["requests"])
		}
		if len(requests) != 2 {
			t.Fatalf("expected 2 requests, got %d", len(requests))
		}
		req0, ok := requests[0].(map[string]any)
		if !ok {
			t.Fatal("unexpected request type")
		}
		if req0["model"] != "models/gemini-embedding-001" {
			t.Errorf("expected model 'models/gemini-embedding-001', got %v", req0["model"])
		}
	})

	t.Run("should pass the outputDimensionality setting", func(t *testing.T) {
		server, capture := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{"outputDimensionality": 64},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body := capture.BodyJSON()
		requests := body["requests"].([]any)
		req0 := requests[0].(map[string]any)
		if req0["outputDimensionality"] == nil {
			t.Error("expected outputDimensionality")
		}
	})

	t.Run("should pass the taskType setting", func(t *testing.T) {
		server, capture := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{"taskType": "SEMANTIC_SIMILARITY"},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body := capture.BodyJSON()
		requests := body["requests"].([]any)
		req0 := requests[0].(map[string]any)
		if req0["taskType"] != "SEMANTIC_SIMILARITY" {
			t.Errorf("expected taskType 'SEMANTIC_SIMILARITY', got %v", req0["taskType"])
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := NewGoogleEmbeddingModel("gemini-embedding-001", GoogleEmbeddingModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{
					"x-goog-api-key":        "test-api-key",
					"Custom-Provider-Header": "provider-header-value",
				}
			},
		})
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			Headers: map[string]string{
				"Custom-Request-Header": "request-header-value",
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if capture.Headers.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Error("expected provider header")
		}
		if capture.Headers.Get("Custom-Request-Header") != "request-header-value" {
			t.Error("expected request header")
		}
		if capture.Headers.Get("X-Goog-Api-Key") != "test-api-key" {
			t.Error("expected api key header")
		}
	})

	t.Run("should throw an error if too many values are provided", func(t *testing.T) {
		model := NewGoogleEmbeddingModel("gemini-embedding-001", GoogleEmbeddingModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{} },
		})
		tooManyValues := make([]string, 2049)
		for i := range tooManyValues {
			tooManyValues[i] = "test"
		}
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: tooManyValues,
			Ctx:    context.Background(),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "2048") {
			t.Errorf("expected error to mention 2048, got: %v", err)
		}
	})

	t.Run("should use the batch embeddings endpoint", func(t *testing.T) {
		server, capture := createBatchEmbeddingServer(dummyEmbeddings, nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		_, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: embeddingTestValues,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(capture.URL, "batchEmbedContents") {
			t.Errorf("expected batch endpoint, got URL: %s", capture.URL)
		}
	})

	t.Run("should use the single embeddings endpoint", func(t *testing.T) {
		server, capture := createSingleEmbeddingServer(dummyEmbeddings[0], nil)
		defer server.Close()

		model := createTestEmbeddingModel(server.URL)
		result, err := model.DoEmbed(embeddingmodel.CallOptions{
			Values: []string{embeddingTestValues[0]},
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(capture.URL, "embedContent") {
			t.Errorf("expected single endpoint, got URL: %s", capture.URL)
		}
		if len(result.Embeddings) != 1 {
			t.Fatalf("expected 1 embedding, got %d", len(result.Embeddings))
		}
		if len(result.Embeddings[0]) != 5 {
			t.Errorf("expected 5 values, got %d", len(result.Embeddings[0]))
		}
	})
}
