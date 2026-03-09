// Ported from: packages/openai-compatible/src/image/openai-compatible-image-model.test.ts
package openaicompatible

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- Image model test helpers ---

func createImageModel(baseURL string) *ImageModel {
	fixedDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return NewImageModel("test-image-model", ImageModelConfig{
		Provider: "test-provider.image",
		URL: func(path string) string {
			return baseURL + path
		},
		Headers: func() map[string]string {
			return map[string]string{
				"Authorization": "Bearer test-key",
				"Content-Type":  "application/json",
			}
		},
		CurrentDateFunc: func() time.Time { return fixedDate },
	})
}

func imageGenerationFixture() map[string]any {
	return map[string]any{
		"data": []any{
			map[string]any{
				"b64_json": "base64-image-data-1",
			},
		},
	}
}

func imageGenerationMultiFixture() map[string]any {
	return map[string]any{
		"data": []any{
			map[string]any{
				"b64_json": "base64-image-data-1",
			},
			map[string]any{
				"b64_json": "base64-image-data-2",
			},
		},
	}
}

// --- Constructor tests ---

func TestImageModel_Constructor(t *testing.T) {
	t.Run("should have correct model ID", func(t *testing.T) {
		model := NewImageModel("dall-e-3", ImageModelConfig{
			Provider: "openai.image",
		})
		if model.ModelID() != "dall-e-3" {
			t.Errorf("expected model ID 'dall-e-3', got %q", model.ModelID())
		}
	})

	t.Run("should have correct provider", func(t *testing.T) {
		model := NewImageModel("dall-e-3", ImageModelConfig{
			Provider: "openai.image",
		})
		if model.Provider() != "openai.image" {
			t.Errorf("expected provider 'openai.image', got %q", model.Provider())
		}
	})

	t.Run("should return max images per call", func(t *testing.T) {
		model := NewImageModel("test", ImageModelConfig{
			Provider: "test.image",
		})
		max, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 10 {
			t.Errorf("expected max 10, got %v", max)
		}
	})
}

// --- DoGenerate tests ---

func TestImageDoGenerate_Parameters(t *testing.T) {
	t.Run("should send correct request body", func(t *testing.T) {
		server, capture := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A beautiful sunset"
		size := "1024x1024"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Size:   &size,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["model"] != "test-image-model" {
			t.Errorf("expected model 'test-image-model', got %v", body["model"])
		}
		if body["prompt"] != "A beautiful sunset" {
			t.Errorf("expected prompt 'A beautiful sunset', got %v", body["prompt"])
		}
		if body["n"] != float64(1) {
			t.Errorf("expected n 1, got %v", body["n"])
		}
		if body["size"] != "1024x1024" {
			t.Errorf("expected size '1024x1024', got %v", body["size"])
		}
		if body["response_format"] != "b64_json" {
			t.Errorf("expected response_format 'b64_json', got %v", body["response_format"])
		}
	})
}

func TestImageDoGenerate_ProviderOptions(t *testing.T) {
	t.Run("should use provider options key from provider name", func(t *testing.T) {
		server, capture := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"quality": "hd",
					"style":   "natural",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["quality"] != "hd" {
			t.Errorf("expected quality 'hd', got %v", body["quality"])
		}
		if body["style"] != "natural" {
			t.Errorf("expected style 'natural', got %v", body["style"])
		}
	})
}

func TestImageDoGenerate_Warnings(t *testing.T) {
	t.Run("should warn about aspectRatio", func(t *testing.T) {
		server, _ := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"
		aspectRatio := "16:9"

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:      &prompt,
			N:           1,
			AspectRatio: &aspectRatio,
			Ctx:         context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "aspectRatio" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for aspectRatio")
		}
	})

	t.Run("should warn about seed", func(t *testing.T) {
		server, _ := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"
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

		hasWarning := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "seed" {
				hasWarning = true
				break
			}
		}
		if !hasWarning {
			t.Error("expected unsupported warning for seed")
		}
	})
}

func TestImageDoGenerate_Headers(t *testing.T) {
	t.Run("should pass headers to request", func(t *testing.T) {
		server, capture := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			Headers: map[string]*string{
				"X-Image-Request": strPtr("image-request-value"),
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Image-Request") != "image-request-value" {
			t.Errorf("expected X-Image-Request header, got %q", capture.Headers.Get("X-Image-Request"))
		}
	})
}

func TestImageDoGenerate_Content(t *testing.T) {
	t.Run("should extract b64_json images", func(t *testing.T) {
		server, _ := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imageData, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imageData.Values) != 1 {
			t.Fatalf("expected 1 image, got %d", len(imageData.Values))
		}
		if imageData.Values[0] != "base64-image-data-1" {
			t.Errorf("expected 'base64-image-data-1', got %q", imageData.Values[0])
		}
	})

	t.Run("should extract multiple images", func(t *testing.T) {
		server, _ := createTestServer(imageGenerationMultiFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      2,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imageData, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imageData.Values) != 2 {
			t.Fatalf("expected 2 images, got %d", len(imageData.Values))
		}
	})
}

func TestImageDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should include response metadata", func(t *testing.T) {
		server, _ := createTestServer(imageGenerationFixture(), map[string]string{
			"X-Image-Response": "response-value",
		})
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "test-image-model" {
			t.Errorf("expected model ID 'test-image-model', got %q", result.Response.ModelID)
		}
	})
}

func TestImageDoGenerate_UserSetting(t *testing.T) {
	t.Run("should pass user from provider options", func(t *testing.T) {
		server, capture := createTestServer(imageGenerationFixture(), nil)
		defer server.Close()
		model := createImageModel(server.URL)
		prompt := "A sunset"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
			ProviderOptions: shared.ProviderOptions{
				"test-provider": {
					"user": "test-user-id",
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["user"] != "test-user-id" {
			t.Errorf("expected user 'test-user-id', got %v", body["user"])
		}
	})
}
