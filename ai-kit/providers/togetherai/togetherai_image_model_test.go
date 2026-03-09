package togetherai

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
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

const testPrompt = "A cute baby sea otter"

// successImageResponse is the default successful response body.
var successImageResponse = map[string]interface{}{
	"id":     "test-id",
	"data":   []map[string]interface{}{{"index": 0, "b64_json": "test-base64-content"}},
	"model":  "stabilityai/stable-diffusion-xl",
	"object": "list",
}

// createTestImageModel creates a TogetherAIImageModel pointing at the given test server URL.
func createTestImageModel(serverURL string, opts ...func(*TogetherAIImageModelConfig)) *TogetherAIImageModel {
	config := TogetherAIImageModelConfig{
		Provider: "togetherai",
		BaseURL:  serverURL,
		Headers:  func() map[string]string { return map[string]string{"api-key": "test-key"} },
		Fetch:    nil, // use default http client, which will hit the httptest server
	}
	for _, opt := range opts {
		opt(&config)
	}
	return NewTogetherAIImageModel("stabilityai/stable-diffusion-xl", config)
}

// newImageTestServer creates an httptest server that returns the given JSON response.
func newImageTestServer(t *testing.T, response interface{}, statusCode int, extraHeaders map[string]string) (*httptest.Server, *[]*http.Request) {
	t.Helper()
	var requests []*http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and store the request body so we can inspect it
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		// Re-create the body for later reading
		r.Body = io.NopCloser(strings.NewReader(string(body)))
		requests = append(requests, r)

		for k, v := range extraHeaders {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

// getRequestBody reads and parses the JSON body from a stored request.
func getRequestBody(t *testing.T, requests []*http.Request, index int) map[string]interface{} {
	t.Helper()
	if index >= len(requests) {
		t.Fatalf("expected at least %d request(s), got %d", index+1, len(requests))
	}
	body, err := io.ReadAll(requests[index].Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to parse request body JSON: %v", err)
	}
	return result
}

func strPtr(s string) *string { return &s }
func intPtr(v int) *int       { return &v }

func TestImageModel_Constructor(t *testing.T) {
	model := createTestImageModel("https://api.example.com")

	t.Run("should expose correct provider", func(t *testing.T) {
		if model.Provider() != "togetherai" {
			t.Errorf("expected provider 'togetherai', got %q", model.Provider())
		}
	})

	t.Run("should expose correct model ID", func(t *testing.T) {
		if model.ModelID() != "stabilityai/stable-diffusion-xl" {
			t.Errorf("expected modelId 'stabilityai/stable-diffusion-xl', got %q", model.ModelID())
		}
	})

	t.Run("should expose correct specification version", func(t *testing.T) {
		if model.SpecificationVersion() != "v3" {
			t.Errorf("expected specificationVersion 'v3', got %q", model.SpecificationVersion())
		}
	})

	t.Run("should expose correct maxImagesPerCall", func(t *testing.T) {
		max, err := model.MaxImagesPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 1 {
			t.Errorf("expected maxImagesPerCall 1, got %v", max)
		}
	})
}

func TestImageModel_DoGenerate(t *testing.T) {
	t.Run("should pass the correct parameters including size and seed", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:      &prompt,
			N:           1,
			Size:        strPtr("1024x1024"),
			Seed:        intPtr(42),
			AspectRatio: nil,
			ProviderOptions: shared.ProviderOptions{
				"togetherai": {"additional_param": "value"},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)

		if body["model"] != "stabilityai/stable-diffusion-xl" {
			t.Errorf("expected model 'stabilityai/stable-diffusion-xl', got %v", body["model"])
		}
		if body["prompt"] != testPrompt {
			t.Errorf("expected prompt %q, got %v", testPrompt, body["prompt"])
		}
		if body["seed"] != float64(42) {
			t.Errorf("expected seed 42, got %v", body["seed"])
		}
		if body["width"] != float64(1024) {
			t.Errorf("expected width 1024, got %v", body["width"])
		}
		if body["height"] != float64(1024) {
			t.Errorf("expected height 1024, got %v", body["height"])
		}
		if body["response_format"] != "base64" {
			t.Errorf("expected response_format 'base64', got %v", body["response_format"])
		}
		// n=1 should not be included in body
		if _, ok := body["n"]; ok {
			t.Errorf("expected n not to be included for n=1, but it was: %v", body["n"])
		}
	})

	t.Run("should include n parameter when requesting multiple images", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               3,
			Size:            strPtr("1024x1024"),
			Seed:            intPtr(42),
			AspectRatio:     nil,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)

		if body["n"] != float64(3) {
			t.Errorf("expected n=3, got %v", body["n"])
		}
	})

	t.Run("should call the correct url", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			Size:            strPtr("1024x1024"),
			Seed:            intPtr(42),
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := (*requests)[0]
		if req.Method != "POST" {
			t.Errorf("expected POST method, got %s", req.Method)
		}
		expectedPath := "/images/generations"
		if req.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, req.URL.Path)
		}
	})

	t.Run("should pass headers", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL, func(cfg *TogetherAIImageModelConfig) {
			cfg.Headers = func() map[string]string {
				return map[string]string{
					"Custom-Provider-Header": "provider-header-value",
				}
			}
		})
		prompt := testPrompt

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Headers: map[string]*string{
				"Custom-Request-Header": strPtr("request-header-value"),
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		req := (*requests)[0]
		if req.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got %q", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("Custom-Provider-Header") != "provider-header-value" {
			t.Errorf("expected Custom-Provider-Header 'provider-header-value', got %q", req.Header.Get("Custom-Provider-Header"))
		}
		if req.Header.Get("Custom-Request-Header") != "request-header-value" {
			t.Errorf("expected Custom-Request-Header 'request-header-value', got %q", req.Header.Get("Custom-Request-Header"))
		}
	})

	t.Run("should handle API errors", func(t *testing.T) {
		errorResponse := map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Bad Request",
			},
		}
		server, _ := newImageTestServer(t, errorResponse, 400, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "Bad Request") {
			t.Errorf("expected error to contain 'Bad Request', got %q", err.Error())
		}
	})

	t.Run("should return aspectRatio warning when size is provided", func(t *testing.T) {
		server, _ := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			Size:            strPtr("1024x1024"),
			AspectRatio:     strPtr("1:1"),
			Seed:            intPtr(123),
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) == 0 {
			t.Fatal("expected at least one warning")
		}
		unsupported, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if unsupported.Feature != "aspectRatio" {
			t.Errorf("expected feature 'aspectRatio', got %q", unsupported.Feature)
		}
		if unsupported.Details == nil || !strings.Contains(*unsupported.Details, "aspectRatio") {
			t.Errorf("expected details to mention aspectRatio, got %v", unsupported.Details)
		}
	})

	t.Run("should respect the abort signal", func(t *testing.T) {
		server, _ := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             ctx,
		})
		if err == nil {
			t.Fatal("expected error from cancelled context, got nil")
		}
	})

	t.Run("should include timestamp, headers and modelId in response", func(t *testing.T) {
		testDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		server, _ := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL, func(cfg *TogetherAIImageModelConfig) {
			cfg.CurrentDateFunc = func() time.Time { return testDate }
		})
		prompt := testPrompt

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.ModelID != "stabilityai/stable-diffusion-xl" {
			t.Errorf("expected modelId 'stabilityai/stable-diffusion-xl', got %q", result.Response.ModelID)
		}
		if result.Response.Headers == nil {
			t.Error("expected response headers to be non-nil")
		}
	})

	t.Run("should include response headers from API call", func(t *testing.T) {
		server, _ := newImageTestServer(t, successImageResponse, 200, map[string]string{
			"X-Request-Id": "test-request-id",
		})
		model := createTestImageModel(server.URL)
		prompt := testPrompt

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt:          &prompt,
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.Headers["X-Request-Id"] != "test-request-id" {
			t.Errorf("expected X-Request-Id 'test-request-id', got %q", result.Response.Headers["X-Request-Id"])
		}
	})
}

func TestImageModel_ImageEditing(t *testing.T) {
	t.Run("should send image_url when URL file is provided", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Make the shirt yellow"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileURL{URL: "https://example.com/input.jpg"},
			},
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)
		if body["image_url"] != "https://example.com/input.jpg" {
			t.Errorf("expected image_url 'https://example.com/input.jpg', got %v", body["image_url"])
		}
		if body["prompt"] != "Make the shirt yellow" {
			t.Errorf("expected prompt 'Make the shirt yellow', got %v", body["prompt"])
		}
		if body["response_format"] != "base64" {
			t.Errorf("expected response_format 'base64', got %v", body["response_format"])
		}
	})

	t.Run("should convert Uint8Array file to data URI", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Transform this image"
		testImageData := []byte{137, 80, 78, 71, 13, 10, 26, 10}

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataBytes{Data: testImageData},
				},
			},
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)
		imageURL, ok := body["image_url"].(string)
		if !ok {
			t.Fatalf("expected image_url to be a string, got %T", body["image_url"])
		}
		if !strings.HasPrefix(imageURL, "data:image/png;base64,") {
			t.Errorf("expected image_url to start with 'data:image/png;base64,', got %q", imageURL)
		}
		if body["prompt"] != "Transform this image" {
			t.Errorf("expected prompt 'Transform this image', got %v", body["prompt"])
		}
	})

	t.Run("should convert file with base64 string data to data URI", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Edit this"
		b64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileData{
					MediaType: "image/png",
					Data:      imagemodel.ImageFileDataString{Value: b64Data},
				},
			},
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)
		expectedURL := "data:image/png;base64," + b64Data
		if body["image_url"] != expectedURL {
			t.Errorf("expected image_url %q, got %v", expectedURL, body["image_url"])
		}
	})

	t.Run("should throw error when mask is provided", func(t *testing.T) {
		server, _ := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Inpaint this area"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileURL{URL: "https://example.com/input.jpg"},
			},
			Mask:            imagemodel.FileURL{URL: "https://example.com/mask.png"},
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err == nil {
			t.Fatal("expected error when mask is provided, got nil")
		}
		expectedMsg := "Together AI does not support mask-based image editing"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("expected error to contain %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("should warn when multiple files are provided", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Edit multiple images"

		result, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileURL{URL: "https://example.com/input1.jpg"},
				imagemodel.FileURL{URL: "https://example.com/input2.jpg"},
			},
			N:               1,
			ProviderOptions: shared.ProviderOptions{},
			Ctx:             context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check warning
		if len(result.Warnings) == 0 {
			t.Fatal("expected at least one warning for multiple files")
		}
		otherWarning, ok := result.Warnings[0].(shared.OtherWarning)
		if !ok {
			t.Fatalf("expected OtherWarning, got %T", result.Warnings[0])
		}
		if !strings.Contains(otherWarning.Message, "single input image") {
			t.Errorf("expected warning about single input image, got %q", otherWarning.Message)
		}

		// Should only use the first image
		body := getRequestBody(t, *requests, 0)
		if body["image_url"] != "https://example.com/input1.jpg" {
			t.Errorf("expected image_url 'https://example.com/input1.jpg', got %v", body["image_url"])
		}
	})

	t.Run("should pass provider options with image editing", func(t *testing.T) {
		server, requests := newImageTestServer(t, successImageResponse, 200, nil)
		model := createTestImageModel(server.URL)
		prompt := "Transform the style"

		_, err := model.DoGenerate(imagemodel.CallOptions{
			Prompt: &prompt,
			Files: []imagemodel.File{
				imagemodel.FileURL{URL: "https://example.com/input.jpg"},
			},
			N: 1,
			ProviderOptions: shared.ProviderOptions{
				"togetherai": {
					"steps":    float64(28),
					"guidance": 3.5,
				},
			},
			Ctx: context.Background(),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := getRequestBody(t, *requests, 0)
		if body["steps"] != float64(28) {
			t.Errorf("expected steps=28, got %v", body["steps"])
		}
		if body["guidance"] != 3.5 {
			t.Errorf("expected guidance=3.5, got %v", body["guidance"])
		}
		if body["image_url"] != "https://example.com/input.jpg" {
			t.Errorf("expected image_url 'https://example.com/input.jpg', got %v", body["image_url"])
		}
		if body["prompt"] != "Transform the style" {
			t.Errorf("expected prompt 'Transform the style', got %v", body["prompt"])
		}
		if body["response_format"] != "base64" {
			t.Errorf("expected response_format 'base64', got %v", body["response_format"])
		}
	})
}

// newImageTestServerWithFetch creates a test server and returns a custom Fetch function
// that routes requests to it. This is useful when the model is configured with a
// baseURL that doesn't match the test server URL (e.g., for testing URL construction).
func newImageTestServerWithFetch(t *testing.T, response interface{}, statusCode int, extraHeaders map[string]string) (providerutils.FetchFunction, *[]*http.Request) {
	t.Helper()
	server, requests := newImageTestServer(t, response, statusCode, extraHeaders)
	fetch := func(req *http.Request) (*http.Response, error) {
		// Rewrite the URL to point to our test server
		newURL := server.URL + req.URL.Path
		newReq, err := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		if err != nil {
			return nil, err
		}
		newReq.Header = req.Header
		return http.DefaultClient.Do(newReq)
	}
	return fetch, requests
}
