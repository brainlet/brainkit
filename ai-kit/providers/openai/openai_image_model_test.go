// Ported from: packages/openai/src/image/openai-image-model.test.ts
package openai

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func createImageTestModel(baseURL string) *OpenAIImageModel {
	return NewOpenAIImageModel("dall-e-3", OpenAIImageModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: "openai.image",
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
		Internal: &OpenAIImageModelInternal{
			CurrentDate: func() time.Time {
				return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		},
	})
}

func createGPTImageTestModel(baseURL string) *OpenAIImageModel {
	return NewOpenAIImageModel("gpt-image-1", OpenAIImageModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: "openai.image",
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
		Internal: &OpenAIImageModelInternal{
			CurrentDate: func() time.Time {
				return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		},
	})
}

func imageFixture() map[string]any {
	return map[string]any{
		"created": float64(1711115037),
		"data": []any{
			map[string]any{
				"b64_json":       "base64-image-data",
				"revised_prompt": "A beautiful sunset",
			},
		},
	}
}

func imageFixtureWithUsage() map[string]any {
	return map[string]any{
		"created": float64(1711115037),
		"data": []any{
			map[string]any{
				"b64_json": "base64-image-data-1",
			},
		},
		"usage": map[string]any{
			"input_tokens":  float64(100),
			"output_tokens": float64(200),
			"total_tokens":  float64(300),
			"input_tokens_details": map[string]any{
				"image_tokens": float64(50),
				"text_tokens":  float64(50),
			},
		},
	}
}

func multiImageFixture() map[string]any {
	return map[string]any{
		"created": float64(1711115037),
		"data": []any{
			map[string]any{
				"b64_json": "base64-image-1",
			},
			map[string]any{
				"b64_json": "base64-image-2",
			},
		},
	}
}

func TestImageDoGenerate_StandardGeneration(t *testing.T) {
	t.Run("should extract image data", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "A beautiful sunset"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imgStrings, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imgStrings.Values) != 1 {
			t.Fatalf("expected 1 image, got %d", len(imgStrings.Values))
		}
		if imgStrings.Values[0] != "base64-image-data" {
			t.Errorf("expected 'base64-image-data', got %q", imgStrings.Values[0])
		}
	})

	t.Run("should extract multiple images", func(t *testing.T) {
		server, _ := createJSONTestServer(multiImageFixture(), nil)
		defer server.Close()
		model := createGPTImageTestModel(server.URL)

		prompt := "Cats"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      2,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imgStrings, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imgStrings.Values) != 2 {
			t.Fatalf("expected 2 images, got %d", len(imgStrings.Values))
		}
	})
}

func TestImageDoGenerate_RequestBody(t *testing.T) {
	t.Run("should pass model, prompt, n, and response_format", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

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
		if body["model"] != "dall-e-3" {
			t.Errorf("expected model 'dall-e-3', got %v", body["model"])
		}
		if body["prompt"] != "A beautiful sunset" {
			t.Errorf("expected prompt, got %v", body["prompt"])
		}
		if body["n"] != float64(1) {
			t.Errorf("expected n 1, got %v", body["n"])
		}
		if body["size"] != "1024x1024" {
			t.Errorf("expected size '1024x1024', got %v", body["size"])
		}
		// dall-e-3 does not have default response format
		if body["response_format"] != "b64_json" {
			t.Errorf("expected response_format 'b64_json', got %v", body["response_format"])
		}
	})

	t.Run("should not include response_format for gpt-image models", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createGPTImageTestModel(server.URL)

		prompt := "Test"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["response_format"]; ok {
			t.Error("expected response_format to be absent for gpt-image model")
		}
	})

	t.Run("should pass provider-specific options", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			ProviderOptions: shared.ProviderOptions{
				"openai": map[string]any{
					"quality": "hd",
					"style":   "natural",
				},
			},
			Ctx: context.Background(),
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
	t.Run("should warn about unsupported aspectRatio", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
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

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "aspectRatio" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for aspectRatio")
		}
	})

	t.Run("should warn about unsupported seed", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
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

		found := false
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "seed" {
				found = true
			}
		}
		if !found {
			t.Error("expected unsupported warning for seed")
		}
	})
}

func TestImageDoGenerate_Usage(t *testing.T) {
	t.Run("should extract usage when present", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixtureWithUsage(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Usage == nil {
			t.Fatal("expected non-nil usage")
		}
		if result.Usage.InputTokens == nil || *result.Usage.InputTokens != 100 {
			t.Errorf("expected input tokens 100, got %v", result.Usage.InputTokens)
		}
		if result.Usage.OutputTokens == nil || *result.Usage.OutputTokens != 200 {
			t.Errorf("expected output tokens 200, got %v", result.Usage.OutputTokens)
		}
		if result.Usage.TotalTokens == nil || *result.Usage.TotalTokens != 300 {
			t.Errorf("expected total tokens 300, got %v", result.Usage.TotalTokens)
		}
	})
}

func TestImageDoGenerate_ResponseHeaders(t *testing.T) {
	t.Run("should extract response headers", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixture(), map[string]string{
			"X-Image-Header": "image-value",
		})
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.Headers["X-Image-Header"] != "image-value" {
			t.Errorf("expected X-Image-Header 'image-value', got %q", result.Response.Headers["X-Image-Header"])
		}
	})
}

func TestImageDoGenerate_ResponseMetadata(t *testing.T) {
	t.Run("should extract response metadata", func(t *testing.T) {
		server, _ := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "dall-e-3" {
			t.Errorf("expected model ID 'dall-e-3', got %q", result.Response.ModelID)
		}
	})
}

func TestImageModel_MaxImagesPerCall(t *testing.T) {
	tests := []struct {
		modelID  string
		expected int
	}{
		{"dall-e-3", 1},
		{"dall-e-2", 10},
		{"gpt-image-1", 10},
		{"unknown-model", 1},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			model := NewOpenAIImageModel(tt.modelID, OpenAIImageModelConfig{
				OpenAIConfig: OpenAIConfig{Provider: "openai.image"},
			})

			max, err := model.MaxImagesPerCall()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if max == nil || *max != tt.expected {
				t.Errorf("expected %d, got %v", tt.expected, max)
			}
		})
	}
}

func TestImageDoGenerate_CustomHeaders(t *testing.T) {
	t.Run("should pass custom headers", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		headerVal := "custom-image"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Headers: map[string]*string{
				"X-Custom-Image": &headerVal,
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.Headers.Get("X-Custom-Image") != "custom-image" {
			t.Errorf("expected custom header, got %q", capture.Headers.Get("X-Custom-Image"))
		}
	})
}

func TestImageDoGenerate_NullRevisedPrompt(t *testing.T) {
	t.Run("should handle null revised_prompt responses", func(t *testing.T) {
		fixture := map[string]any{
			"created": float64(1733837122),
			"data": []any{
				map[string]any{
					"b64_json": "base64-image-1",
				},
			},
		}
		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createGPTImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		imgStrings, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imgStrings.Values) != 1 || imgStrings.Values[0] != "base64-image-1" {
			t.Errorf("expected image data 'base64-image-1', got %v", imgStrings.Values)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestImageDoGenerate_ChatGPTImageLatest(t *testing.T) {
	t.Run("should not include response_format for chatgpt-image-latest", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := NewOpenAIImageModel("chatgpt-image-latest", OpenAIImageModelConfig{
			OpenAIConfig: OpenAIConfig{
				Provider: "openai.image",
				URL: func(options struct {
					ModelID string
					Path    string
				}) string {
					return server.URL + options.Path
				},
				Headers: func() map[string]string {
					return map[string]string{
						"Authorization": "Bearer test-api-key",
						"Content-Type":  "application/json",
					}
				},
			},
			Internal: &OpenAIImageModelInternal{
				CurrentDate: func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				},
			},
		})

		prompt := "Test"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["response_format"]; ok {
			t.Error("expected response_format to be absent for chatgpt-image-latest")
		}
		if body["model"] != "chatgpt-image-latest" {
			t.Errorf("expected model 'chatgpt-image-latest', got %v", body["model"])
		}
	})
}

func TestImageDoGenerate_DateSuffixedGPTImage(t *testing.T) {
	t.Run("should not include response_format for date-suffixed gpt-image model IDs", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := NewOpenAIImageModel("gpt-image-1.5-2025-12-16", OpenAIImageModelConfig{
			OpenAIConfig: OpenAIConfig{
				Provider: "openai.image",
				URL: func(options struct {
					ModelID string
					Path    string
				}) string {
					return server.URL + options.Path
				},
				Headers: func() map[string]string {
					return map[string]string{
						"Authorization": "Bearer test-api-key",
						"Content-Type":  "application/json",
					}
				},
			},
			Internal: &OpenAIImageModelInternal{
				CurrentDate: func() time.Time {
					return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				},
			},
		})

		prompt := "Test"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if _, ok := body["response_format"]; ok {
			t.Error("expected response_format to be absent for date-suffixed gpt-image model")
		}
	})
}

func TestImageDoGenerate_IncludeResponseFormatDallE3(t *testing.T) {
	t.Run("should include response_format for dall-e-3", func(t *testing.T) {
		server, capture := createJSONTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON()
		if body["response_format"] != "b64_json" {
			t.Errorf("expected response_format 'b64_json', got %v", body["response_format"])
		}
	})
}

func TestImageDoGenerate_ProviderMetadata(t *testing.T) {
	t.Run("should return image meta data in providerMetadata", func(t *testing.T) {
		fixture := map[string]any{
			"created": float64(1770935200),
			"data": []any{
				map[string]any{
					"b64_json":       "base64-image-data",
					"revised_prompt": "A stunning sunset over mountains",
				},
			},
		}
		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProviderMetadata == nil {
			t.Fatal("expected non-nil providerMetadata")
		}
		entry, ok := result.ProviderMetadata["openai"]
		if !ok {
			t.Fatal("expected openai key in providerMetadata")
		}
		if len(entry.Images) != 1 {
			t.Fatalf("expected 1 image in metadata, got %d", len(entry.Images))
		}
		imgMeta, ok := entry.Images[0].(map[string]interface{})
		if !ok {
			t.Fatalf("expected map[string]interface{}, got %T", entry.Images[0])
		}
		if imgMeta["created"] != 1770935200 {
			t.Errorf("expected created 1770935200, got %v (%T)", imgMeta["created"], imgMeta["created"])
		}
		if imgMeta["revisedPrompt"] != "A stunning sunset over mountains" {
			t.Errorf("expected revisedPrompt, got %v", imgMeta["revisedPrompt"])
		}
	})
}

func TestImageDoGenerate_TokenDetailsDistribution(t *testing.T) {
	t.Run("should distribute input token details evenly across images", func(t *testing.T) {
		fixture := map[string]any{
			"created": float64(1711115037),
			"data": []any{
				map[string]any{"b64_json": "base64-image-1"},
				map[string]any{"b64_json": "base64-image-2"},
			},
			"usage": map[string]any{
				"input_tokens":  float64(100),
				"output_tokens": float64(200),
				"total_tokens":  float64(300),
				"input_tokens_details": map[string]any{
					"image_tokens": float64(10),
					"text_tokens":  float64(20),
				},
			},
		}
		server, _ := createJSONTestServer(fixture, nil)
		defer server.Close()
		model := createGPTImageTestModel(server.URL)

		prompt := "Test"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      2,
			Ctx:    context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		entry := result.ProviderMetadata["openai"]
		if len(entry.Images) != 2 {
			t.Fatalf("expected 2 images in metadata, got %d", len(entry.Images))
		}

		// First image gets base (10/2=5), second gets remainder (10-5=5)
		img1 := entry.Images[0].(map[string]interface{})
		img2 := entry.Images[1].(map[string]interface{})
		if img1["imageTokens"] != 5 {
			t.Errorf("expected first image imageTokens 5, got %v", img1["imageTokens"])
		}
		if img2["imageTokens"] != 5 {
			t.Errorf("expected second image imageTokens 5, got %v", img2["imageTokens"])
		}
	})
}

func TestImageDoGenerate_EditWithSingleFile(t *testing.T) {
	t.Run("should send edit request with single file to /images/edits", func(t *testing.T) {
		server, capture := createFormDataTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Make the sky purple"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("fake-image-data")},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify image data returned
		imgs, ok := result.Images.(imagemodel.ImageDataStrings)
		if !ok {
			t.Fatalf("expected ImageDataStrings, got %T", result.Images)
		}
		if len(imgs.Values) != 1 {
			t.Fatalf("expected 1 image, got %d", len(imgs.Values))
		}

		// Verify the request went to /images/edits endpoint
		bodyStr := string(capture.Body)
		if len(bodyStr) == 0 {
			t.Fatal("expected non-empty request body")
		}
		// The body should be multipart form data containing the model and prompt
		if !strings.Contains(bodyStr, "dall-e-3") {
			t.Error("expected request body to contain model 'dall-e-3'")
		}
		if !strings.Contains(bodyStr, "Make the sky purple") {
			t.Error("expected request body to contain prompt")
		}
		if !strings.Contains(bodyStr, "fake-image-data") {
			t.Error("expected request body to contain image data")
		}
	})
}

func TestImageDoGenerate_EditWithMultipleFiles(t *testing.T) {
	t.Run("should send edit request with multiple files", func(t *testing.T) {
		server, capture := createFormDataTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Merge these images"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("image-data-1")},
				},
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("image-data-2")},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		bodyStr := string(capture.Body)
		if !strings.Contains(bodyStr, "image-data-1") {
			t.Error("expected request body to contain first image data")
		}
		if !strings.Contains(bodyStr, "image-data-2") {
			t.Error("expected request body to contain second image data")
		}
	})
}

func TestImageDoGenerate_EditWithMask(t *testing.T) {
	t.Run("should send edit request with mask", func(t *testing.T) {
		server, capture := createFormDataTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Fill in the masked area"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("source-image")},
				},
			},
			Mask: imagemodel.FileData{
				MediaType: "image/png",
				Data:      imagemodel.ImageFileDataBytes{Data: []byte("mask-image-data")},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		bodyStr := string(capture.Body)
		if !strings.Contains(bodyStr, "source-image") {
			t.Error("expected request body to contain source image data")
		}
		if !strings.Contains(bodyStr, "mask-image-data") {
			t.Error("expected request body to contain mask data")
		}
	})
}

func TestImageDoGenerate_EditResponseMetadata(t *testing.T) {
	t.Run("should include response metadata for edit requests", func(t *testing.T) {
		testDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
		server, _ := createFormDataTestServer(imageFixture(), map[string]string{
			"X-Request-Id": "edit-request-id",
		})
		defer server.Close()

		model := NewOpenAIImageModel("dall-e-2", OpenAIImageModelConfig{
			OpenAIConfig: OpenAIConfig{
				Provider: "openai.image",
				URL: func(options struct {
					ModelID string
					Path    string
				}) string {
					return server.URL + options.Path
				},
				Headers: func() map[string]string {
					return map[string]string{
						"Authorization": "Bearer test-api-key",
					}
				},
			},
			Internal: &OpenAIImageModelInternal{
				CurrentDate: func() time.Time {
					return testDate
				},
			},
		})

		prompt := "Edit this image"
		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("test-image")},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "dall-e-2" {
			t.Errorf("expected model ID 'dall-e-2', got %q", result.Response.ModelID)
		}
		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
	})
}

func TestImageDoGenerate_EditWithSize(t *testing.T) {
	t.Run("should pass size parameter in edit request", func(t *testing.T) {
		server, capture := createFormDataTestServer(imageFixture(), nil)
		defer server.Close()
		model := createImageTestModel(server.URL)

		prompt := "Edit this"
		size := "512x512"
		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			Size:   &size,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: []byte("test-img")},
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		bodyStr := string(capture.Body)
		if !strings.Contains(bodyStr, "512x512") {
			t.Error("expected request body to contain size '512x512'")
		}
	})
}
