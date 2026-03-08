// Ported from: packages/ai/src/generate-image/generate-image.test.ts
package generateimage

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"
)

// mockImageModel is a mock implementation of ImageModel for testing.
type mockImageModel struct {
	provider         string
	modelID          string
	maxImagesPerCall int
	doGenerate       func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

func (m *mockImageModel) Provider() string      { return m.provider }
func (m *mockImageModel) ModelID() string        { return m.modelID }
func (m *mockImageModel) MaxImagesPerCall() int  { return m.maxImagesPerCall }
func (m *mockImageModel) DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
	return m.doGenerate(ctx, opts)
}

func newMockImageModel(maxImagesPerCall int, doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)) *mockImageModel {
	return &mockImageModel{
		provider:         "mock-provider",
		modelID:          "mock-model-id",
		maxImagesPerCall: maxImagesPerCall,
		doGenerate:       doGenerate,
	}
}

var pngBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAACklEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg=="

func decodePNG() []byte {
	data, _ := base64.StdEncoding.DecodeString(pngBase64)
	return data
}

func TestGenerateImage_SingleImage(t *testing.T) {
	t.Run("should generate a single image", func(t *testing.T) {
		pngData := decodePNG()

		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Images:   [][]byte{pngData},
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{
					ModelID: "mock-model-id",
				},
			}, nil
		})

		result, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Images) != 1 {
			t.Errorf("expected 1 image, got %d", len(result.Images))
		}
		if result.Image.MediaType != "image/png" {
			t.Errorf("expected media type image/png, got %q", result.Image.MediaType)
		}
	})
}

func TestGenerateImage_MultipleImages(t *testing.T) {
	t.Run("should generate multiple images with multiple calls", func(t *testing.T) {
		pngData := decodePNG()

		// Dispatch based on opts.N since calls run concurrently.
		model := newMockImageModel(2, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			images := make([][]byte, opts.N)
			for i := range images {
				images[i] = pngData
			}
			return &DoGenerateResult{
				Images:   images,
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{ModelID: "mock-model-id"},
			}, nil
		})

		result, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
			N:      3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Images) != 3 {
			t.Errorf("expected 3 images, got %d", len(result.Images))
		}
	})
}

func TestGenerateImage_Warnings(t *testing.T) {
	t.Run("should return warnings", func(t *testing.T) {
		pngData := decodePNG()
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
		}

		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Images:   [][]byte{pngData},
				Warnings: expectedWarnings,
				Response: ImageModelResponseMetadata{ModelID: "mock-model-id"},
			}, nil
		})

		result, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})
}

func TestGenerateImage_NoImageError(t *testing.T) {
	t.Run("should return error when no images generated", func(t *testing.T) {
		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Images:   [][]byte{},
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{ModelID: "mock-model-id"},
			}, nil
		})

		_, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGenerateImage_ProviderOptions(t *testing.T) {
	t.Run("should pass provider options to model", func(t *testing.T) {
		pngData := decodePNG()

		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			expected := map[string]map[string]any{
				"openai": {"style": "vivid"},
			}
			if !reflect.DeepEqual(opts.ProviderOptions, expected) {
				t.Errorf("expected provider options %v, got %v", expected, opts.ProviderOptions)
			}
			return &DoGenerateResult{
				Images:   [][]byte{pngData},
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{ModelID: "mock-model-id"},
			}, nil
		})

		_, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
			ProviderOptions: map[string]map[string]any{
				"openai": {"style": "vivid"},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGenerateImage_SizeAndAspectRatio(t *testing.T) {
	t.Run("should pass size and aspect ratio to model", func(t *testing.T) {
		pngData := decodePNG()

		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			if opts.Size != "1024x1024" {
				t.Errorf("expected size 1024x1024, got %q", opts.Size)
			}
			if opts.AspectRatio != "1:1" {
				t.Errorf("expected aspect ratio 1:1, got %q", opts.AspectRatio)
			}
			return &DoGenerateResult{
				Images:   [][]byte{pngData},
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{ModelID: "mock-model-id"},
			}, nil
		})

		_, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:       model,
			Prompt:      "sunny day at the beach",
			Size:        "1024x1024",
			AspectRatio: "1:1",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGenerateImage_ResponseMetadata(t *testing.T) {
	t.Run("should return response metadata", func(t *testing.T) {
		pngData := decodePNG()
		testHeaders := map[string]string{"x-test": "value"}

		model := newMockImageModel(1, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Images:   [][]byte{pngData},
				Warnings: []Warning{},
				Response: ImageModelResponseMetadata{
					ModelID: "test-model",
					Headers: testHeaders,
				},
			}, nil
		})

		result, err := GenerateImage(context.Background(), GenerateImageOptions{
			Model:  model,
			Prompt: "sunny day at the beach",
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
