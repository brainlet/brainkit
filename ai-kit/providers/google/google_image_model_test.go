// Ported from: packages/google/src/google-generative-ai-image-model.test.ts
package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

type imageRequestCapture struct {
	Body    []byte
	Headers http.Header
}

func (rc *imageRequestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

func createImageTestServer(responseBody map[string]any, headers map[string]string) (*httptest.Server, *imageRequestCapture) {
	capture := &imageRequestCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Body = bodyBytes
		capture.Headers = r.Header
		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody)
	}))
	return server, capture
}

func imagenFixture() map[string]any {
	return map[string]any{
		"predictions": []any{
			map[string]any{"bytesBase64Encoded": "base64-image-1"},
			map[string]any{"bytesBase64Encoded": "base64-image-2"},
		},
	}
}

func createTestImagenModel(baseURL string) *GoogleImageModel {
	return NewGoogleImageModel("imagen-3.0-generate-002", GoogleImageSettings{}, GoogleImageModelConfig{
		Provider: "google.generative-ai",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return map[string]string{"api-key": "test-api-key"}
		},
	})
}

func TestGoogleImageModelImagen(t *testing.T) {
	prompt := "A cute baby sea otter"

	t.Run("should pass headers", func(t *testing.T) {
		server, capture := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := NewGoogleImageModel("imagen-3.0-generate-002", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{"Custom-Provider-Header": "provider-header-value"}
			},
		})
		hdrVal := "request-header-value"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:  &prompt,
			N:       2,
			Headers: map[string]*string{"Custom-Request-Header": &hdrVal},
			Ctx:     context.Background(),
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
	})

	t.Run("should respect maxImagesPerCall setting", func(t *testing.T) {
		maxImages := 2
		model := NewGoogleImageModel("imagen-3.0-generate-002", GoogleImageSettings{
			MaxImagesPerCall: &maxImages,
		}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://api.example.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{} },
		})
		result, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatal(err)
		}
		if *result != 2 {
			t.Errorf("expected 2, got %d", *result)
		}
	})

	t.Run("should use default maxImagesPerCall when not specified", func(t *testing.T) {
		model := NewGoogleImageModel("imagen-3.0-generate-002", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://api.example.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{} },
		})
		result, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatal(err)
		}
		if *result != 4 {
			t.Errorf("expected 4 (Imagen default), got %d", *result)
		}
	})

	t.Run("should extract the generated images", func(t *testing.T) {
		server, _ := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      2,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		images := result.Images.(imagemodel.ImageDataStrings)
		if len(images.Values) != 2 {
			t.Fatalf("expected 2 images, got %d", len(images.Values))
		}
		if images.Values[0] != "base64-image-1" {
			t.Errorf("expected 'base64-image-1', got %q", images.Values[0])
		}
		if images.Values[1] != "base64-image-2" {
			t.Errorf("expected 'base64-image-2', got %q", images.Values[1])
		}
	})

	t.Run("sends aspect ratio in the request", func(t *testing.T) {
		server, capture := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		testPrompt := "test prompt"
		ar := "16:9"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:      &testPrompt,
			N:           1,
			AspectRatio: &ar,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["aspectRatio"] != "16:9" {
			t.Errorf("expected aspectRatio '16:9', got %v", params["aspectRatio"])
		}
	})

	t.Run("should combine aspectRatio and provider options", func(t *testing.T) {
		server, capture := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		testPrompt := "test prompt"
		ar := "1:1"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:      &testPrompt,
			N:           1,
			AspectRatio: &ar,
			ProviderOptions: shared.ProviderOptions{
				"google": map[string]any{"personGeneration": "dont_allow"},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["personGeneration"] != "dont_allow" {
			t.Errorf("expected personGeneration 'dont_allow', got %v", params["personGeneration"])
		}
		if params["aspectRatio"] != "1:1" {
			t.Errorf("expected aspectRatio '1:1', got %v", params["aspectRatio"])
		}
	})

	t.Run("should return warnings for unsupported settings", func(t *testing.T) {
		server, _ := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		size := "1024x1024"
		ar := "1:1"
		seed := 123
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Size:   &size,
			AspectRatio: &ar,
			Seed:   &seed,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Warnings) != 2 {
			t.Fatalf("expected 2 warnings, got %d", len(result.Warnings))
		}
		w0, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if w0.Feature != "size" {
			t.Errorf("expected feature 'size', got %q", w0.Feature)
		}
		w1, ok := result.Warnings[1].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[1])
		}
		if w1.Feature != "seed" {
			t.Errorf("expected feature 'seed', got %q", w1.Feature)
		}
	})

	t.Run("should include response data with timestamp and modelId", func(t *testing.T) {
		server, _ := createImageTestServer(imagenFixture(), map[string]string{
			"request-id":              "test-request-id",
			"x-goog-quota-remaining": "123",
		})
		defer server.Close()

		testDate := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
		model := NewGoogleImageModel("imagen-3.0-generate-002", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{"api-key": "test-api-key"}
			},
			Internal: &GoogleImageModelInternal{
				CurrentDate: func() time.Time { return testDate },
			},
		})
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.ModelID != "imagen-3.0-generate-002" {
			t.Errorf("expected modelID 'imagen-3.0-generate-002', got %q", result.Response.ModelID)
		}
	})

	t.Run("should throw error when files are provided (Imagen)", func(t *testing.T) {
		server, _ := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		editPrompt := "Edit this image"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &editPrompt,
			Files: []imagemodel.File{
				imagemodel.FileData{
					Data:      imagemodel.ImageFileDataString{Value: "base64-source-image"},
					MediaType: "image/png",
				},
			},
			N:   1,
			Ctx: context.Background(),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "does not support image editing with Imagen models") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("should throw error when mask is provided (Imagen)", func(t *testing.T) {
		server, _ := createImageTestServer(imagenFixture(), nil)
		defer server.Close()

		model := createTestImagenModel(server.URL)
		editPrompt := "Edit this image"
		mask := imagemodel.FileData{
			Data:      imagemodel.ImageFileDataString{Value: "base64-mask-image"},
			MediaType: "image/png",
		}
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &editPrompt,
			Mask:   mask,
			N:      1,
			Ctx:    context.Background(),
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "does not support image editing with masks") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestGoogleImageModelGemini(t *testing.T) {
	t.Run("should use default maxImagesPerCall of 10 for Gemini models", func(t *testing.T) {
		model := NewGoogleImageModel("gemini-2.0-flash", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://api.example.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{} },
		})
		result, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatal(err)
		}
		if *result != 10 {
			t.Errorf("expected 10 (Gemini default), got %d", *result)
		}
	})

	t.Run("should throw error when mask is provided (Gemini)", func(t *testing.T) {
		// Gemini model test uses language model endpoint, so we need to mock generateContent
		geminiResponse := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     "base64-gemini-image",
								},
							},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
			},
		}
		server, _ := createImageTestServer(geminiResponse, nil)
		defer server.Close()

		model := NewGoogleImageModel("gemini-2.0-flash", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers:  func() map[string]string { return map[string]string{"api-key": "test-api-key"} },
		})
		editPrompt := "Edit this"
		mask := imagemodel.FileData{
			Data:      imagemodel.ImageFileDataString{Value: "base64-mask"},
			MediaType: "image/png",
		}
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &editPrompt,
			Mask:   mask,
			N:      1,
			Ctx:    context.Background(),
		})
		if err == nil {
			t.Fatal("expected error for mask")
		}
		if !strings.Contains(err.Error(), "do not support mask-based image editing") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("should throw error when n>1 (Gemini)", func(t *testing.T) {
		model := NewGoogleImageModel("gemini-2.0-flash", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://api.example.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{"api-key": "test-api-key"} },
		})
		p := "Generate"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &p,
			N:      2,
			Ctx:    context.Background(),
		})
		if err == nil {
			t.Fatal("expected error for n>1")
		}
		if !strings.Contains(err.Error(), "do not support generating a set number of images") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("should return size warning for Gemini models", func(t *testing.T) {
		geminiResponse := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     "base64-gemini-image",
								},
							},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
			},
		}
		server, _ := createImageTestServer(geminiResponse, nil)
		defer server.Close()

		model := NewGoogleImageModel("gemini-2.0-flash", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers:  func() map[string]string { return map[string]string{"api-key": "test-api-key"} },
		})
		p := "Generate an image"
		size := "1024x1024"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &p,
			N:      1,
			Size:   &size,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		w, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if w.Feature != "size" {
			t.Errorf("expected feature 'size', got %q", w.Feature)
		}
	})

	t.Run("should extract images from Gemini response", func(t *testing.T) {
		geminiResponse := map[string]any{
			"candidates": []any{
				map[string]any{
					"content": map[string]any{
						"parts": []any{
							map[string]any{
								"inlineData": map[string]any{
									"mimeType": "image/png",
									"data":     "base64-gemini-image",
								},
							},
						},
					},
					"finishReason": "STOP",
				},
			},
			"usageMetadata": map[string]any{
				"promptTokenCount":     float64(10),
				"candidatesTokenCount": float64(20),
			},
		}
		server, _ := createImageTestServer(geminiResponse, nil)
		defer server.Close()

		model := NewGoogleImageModel("gemini-2.0-flash", GoogleImageSettings{}, GoogleImageModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers:  func() map[string]string { return map[string]string{"api-key": "test-api-key"} },
		})
		p := "Generate an image"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &p,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatal(err)
		}
		images := result.Images.(imagemodel.ImageDataStrings)
		if len(images.Values) != 1 {
			t.Fatalf("expected 1 image, got %d", len(images.Values))
		}
		if images.Values[0] != "base64-gemini-image" {
			t.Errorf("expected 'base64-gemini-image', got %q", images.Values[0])
		}
	})
}
