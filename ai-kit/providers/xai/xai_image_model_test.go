// Ported from: packages/xai/src/xai-image-model.test.ts
package xai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// createImageTestServer creates a server that returns an image generation response
// and also serves a fake image download.
func createImageTestServer(body map[string]any, headers map[string]string) (*httptest.Server, *requestCapture) {
	capture := &requestCapture{}
	var serverURL string

	mux := http.NewServeMux()

	// Image generation endpoint
	mux.HandleFunc("/images/generations", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header.Clone()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	})

	// Image editing endpoint
	mux.HandleFunc("/images/edits", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header.Clone()

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(body)
	})

	// Image download endpoint
	mux.HandleFunc("/fake-image", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47}) // PNG header bytes
	})

	server := httptest.NewServer(mux)
	serverURL = server.URL

	// Update the body to point to this test server for image downloads
	if data, ok := body["data"].([]any); ok {
		for _, item := range data {
			if m, ok := item.(map[string]any); ok {
				if _, hasURL := m["url"]; hasURL {
					m["url"] = fmt.Sprintf("%s/fake-image", serverURL)
				}
			}
		}
	}

	return server, capture
}

func createImageModel(serverURL string) *XaiImageModel {
	return NewXaiImageModel("grok-2-image", XaiImageModelConfig{
		Provider: "xai.image",
		BaseURL:  serverURL,
		Headers:  func() map[string]string { return map[string]string{"Authorization": "Bearer test-key"} },
		CurrentDate: func() time.Time {
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})
}

func xaiImageFixture(serverURL string) map[string]any {
	return map[string]any{
		"data": []any{
			map[string]any{
				"url":             fmt.Sprintf("%s/fake-image", serverURL),
				"revised_prompt":  "A beautiful sunset over the ocean",
			},
		},
	}
}

func TestXaiImageModel_Constructor(t *testing.T) {
	t.Run("should create model with correct properties", func(t *testing.T) {
		model := NewXaiImageModel("grok-2-image", XaiImageModelConfig{
			Provider: "xai.image",
			BaseURL:  "https://api.x.ai/v1",
			Headers:  func() map[string]string { return map[string]string{} },
		})

		if model.ModelID() != "grok-2-image" {
			t.Errorf("expected model ID 'grok-2-image', got %q", model.ModelID())
		}
		if model.Provider() != "xai.image" {
			t.Errorf("expected provider 'xai.image', got %q", model.Provider())
		}
	})
}

func TestXaiImageModel_MaxImagesPerCall(t *testing.T) {
	t.Run("should return 1 for maxImagesPerCall", func(t *testing.T) {
		model := NewXaiImageModel("grok-2-image", XaiImageModelConfig{
			Provider: "xai.image",
			BaseURL:  "https://api.x.ai/v1",
			Headers:  func() map[string]string { return map[string]string{} },
		})

		max, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 1 {
			t.Errorf("expected maxImagesPerCall 1, got %v", max)
		}
	})
}

func TestXaiImageModel_Generation(t *testing.T) {
	t.Run("should send correct request body for generation", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, capture := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "A beautiful sunset"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "grok-2-image" {
			t.Errorf("expected model 'grok-2-image', got %v", body["model"])
		}
		if body["prompt"] != "A beautiful sunset" {
			t.Errorf("expected prompt 'A beautiful sunset', got %v", body["prompt"])
		}
		if body["n"] != float64(1) {
			t.Errorf("expected n 1, got %v", body["n"])
		}
		if body["response_format"] != "url" {
			t.Errorf("expected response_format 'url', got %v", body["response_format"])
		}
	})
}

func TestXaiImageModel_AspectRatio(t *testing.T) {
	t.Run("should include aspect_ratio in request", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, capture := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "test"
		ar := "16:9"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:      &prompt,
			N:           1,
			AspectRatio: &ar,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["aspect_ratio"] != "16:9" {
			t.Errorf("expected aspect_ratio '16:9', got %v", body["aspect_ratio"])
		}
	})
}

func TestXaiImageModel_Warnings(t *testing.T) {
	t.Run("should warn about unsupported size parameter", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, _ := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "test"
		size := "1024x1024"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Size:   &size,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var sizeWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "size" {
				sizeWarning = true
			}
		}
		if !sizeWarning {
			t.Error("expected unsupported warning for size")
		}
	})

	t.Run("should warn about unsupported seed parameter", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, _ := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "test"
		seed := 42
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Seed:   &seed,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var seedWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "seed" {
				seedWarning = true
			}
		}
		if !seedWarning {
			t.Error("expected unsupported warning for seed")
		}
	})
}

func TestXaiImageModel_Headers(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, capture := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "test"
		customVal := "custom-value"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Custom": &customVal,
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom 'custom-value', got %q", capture.Headers.Get("X-Custom"))
		}
	})
}

func TestXaiImageModel_FileURL(t *testing.T) {
	t.Run("should handle file URL for editing", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, capture := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "edit this image"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			Files: []imagemodel.File{
				imagemodel.FileURL{
					URL: "https://example.com/image.jpg",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		img, ok := body["image"].(map[string]interface{})
		if !ok {
			t.Fatal("expected image in body")
		}
		if img["url"] != "https://example.com/image.jpg" {
			t.Errorf("expected image URL 'https://example.com/image.jpg', got %v", img["url"])
		}

		// Should hit /images/edits endpoint
		if !strings.Contains(string(capture.Body), "edit this image") {
			t.Error("expected body to contain prompt")
		}
	})
}

func TestXaiImageModel_RevisedPrompt(t *testing.T) {
	t.Run("should include revised_prompt in provider metadata", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url":            "PLACEHOLDER",
					"revised_prompt": "An enhanced beautiful sunset",
				},
			},
		}
		server, _ := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "A beautiful sunset"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProviderMetadata == nil {
			t.Fatal("expected non-nil provider metadata")
		}
		xaiMeta, ok := result.ProviderMetadata["xai"]
		if !ok {
			t.Fatal("expected xai metadata")
		}
		if len(xaiMeta.Images) == 0 {
			t.Fatal("expected at least 1 image metadata entry")
		}
		imgMeta := xaiMeta.Images[0].(map[string]interface{})
		if imgMeta["revisedPrompt"] != "An enhanced beautiful sunset" {
			t.Errorf("expected revisedPrompt 'An enhanced beautiful sunset', got %v", imgMeta["revisedPrompt"])
		}
	})
}

func TestXaiImageModel_MultipleFilesWarning(t *testing.T) {
	t.Run("should warn when multiple files provided", func(t *testing.T) {
		fixture := map[string]any{
			"data": []any{
				map[string]any{
					"url": "PLACEHOLDER",
				},
			},
		}
		server, _ := createImageTestServer(fixture, nil)
		defer server.Close()

		model := createImageModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			Files: []imagemodel.File{
				imagemodel.FileURL{URL: "https://example.com/1.jpg"},
				imagemodel.FileURL{URL: "https://example.com/2.jpg"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var multipleWarning bool
		for _, w := range result.Warnings {
			if ow, ok := w.(shared.OtherWarning); ok {
				if strings.Contains(ow.Message, "single input image") {
					multipleWarning = true
				}
			}
		}
		if !multipleWarning {
			t.Error("expected warning about single input image")
		}
	})
}
