// Ported from: packages/google/src/google-generative-ai-video-model.test.ts
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

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

var videoPrompt = "A futuristic city with flying cars"

type videoRequestCapture struct {
	Body    []byte
	Headers http.Header
	URL     string
}

func (rc *videoRequestCapture) BodyJSON() map[string]any {
	var result map[string]any
	json.Unmarshal(rc.Body, &result)
	return result
}

// createVideoTestServer creates a mock server that handles predictLongRunning
// and operation polling with configurable responses.
func createVideoTestServer(opts videoTestServerOpts) (*httptest.Server, *videoRequestCapture) {
	capture := &videoRequestCapture{}
	pollCount := 0

	videos := opts.Videos
	if videos == nil {
		videos = []map[string]any{
			{"video": map[string]any{"uri": "https://generativelanguage.googleapis.com/files/video-123.mp4"}},
		}
	}

	operationName := opts.OperationName
	if operationName == "" {
		operationName = "operations/test-operation-id"
	}

	pollsUntilDone := opts.PollsUntilDone
	if pollsUntilDone == 0 {
		pollsUntilDone = 1
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.Contains(r.URL.Path, ":predictLongRunning") {
			bodyBytes, _ := io.ReadAll(r.Body)
			capture.Body = bodyBytes
			capture.Headers = r.Header
			capture.URL = r.URL.String()

			resp := map[string]any{
				"name": operationName,
				"done": false,
			}

			if opts.NoOperationName {
				resp = map[string]any{"done": false}
			}

			json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.Contains(r.URL.Path, "operations/") {
			pollCount++

			if pollCount < pollsUntilDone {
				json.NewEncoder(w).Encode(map[string]any{
					"name": operationName,
					"done": false,
				})
				return
			}

			resp := map[string]any{
				"name": operationName,
				"done": true,
			}

			if opts.OperationError != nil {
				resp["error"] = opts.OperationError
			} else {
				resp["response"] = map[string]any{
					"generateVideoResponse": map[string]any{
						"generatedSamples": videos,
					},
				}
			}

			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`"Not found"`))
	}))

	return server, capture
}

type videoTestServerOpts struct {
	Videos          []map[string]any
	OperationName   string
	PollsUntilDone  int
	OperationError  map[string]any
	NoOperationName bool
}

func createTestVideoModel(baseURL string) *GoogleVideoModel {
	return NewGoogleVideoModel("veo-3.1-generate-preview", GoogleVideoModelConfig{
		Provider: "google.generative-ai",
		BaseURL:  baseURL,
		Headers: func() map[string]string {
			return map[string]string{
				"x-goog-api-key": "test-api-key",
			}
		},
	})
}

func defaultVideoOptions() videomodel.CallOptions {
	prompt := videoPrompt
	return videomodel.CallOptions{
		Prompt: &prompt,
		N:      1,
		ProviderOptions: shared.ProviderOptions{
			"google": map[string]any{"pollIntervalMs": float64(10)},
		},
		Ctx: context.Background(),
	}
}

func TestGoogleVideoModel_Constructor(t *testing.T) {
	t.Run("should expose correct provider and model information", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)

		if model.Provider() != "google.generative-ai" {
			t.Errorf("expected provider 'google.generative-ai', got %q", model.Provider())
		}
		if model.ModelID() != "veo-3.1-generate-preview" {
			t.Errorf("expected modelID 'veo-3.1-generate-preview', got %q", model.ModelID())
		}
		if model.SpecificationVersion() != "v3" {
			t.Errorf("expected specificationVersion 'v3', got %q", model.SpecificationVersion())
		}
		max, err := model.MaxVideosPerCall()
		if err != nil {
			t.Fatal(err)
		}
		if max == nil || *max != 4 {
			t.Errorf("expected maxVideosPerCall 4, got %v", max)
		}
	})

	t.Run("should support different model IDs", func(t *testing.T) {
		model := NewGoogleVideoModel("veo-3.1-generate", GoogleVideoModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  "https://generativelanguage.googleapis.com/v1beta",
			Headers:  func() map[string]string { return map[string]string{} },
		})

		if model.ModelID() != "veo-3.1-generate" {
			t.Errorf("expected modelID 'veo-3.1-generate', got %q", model.ModelID())
		}
	})
}

func TestGoogleVideoModel_DoGenerate(t *testing.T) {
	t.Run("should pass the correct parameters including prompt", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		instances, ok := body["instances"].([]any)
		if !ok || len(instances) != 1 {
			t.Fatalf("expected 1 instance, got %v", body["instances"])
		}
		inst := instances[0].(map[string]any)
		if inst["prompt"] != videoPrompt {
			t.Errorf("expected prompt %q, got %v", videoPrompt, inst["prompt"])
		}

		params := body["parameters"].(map[string]any)
		if params["sampleCount"] != float64(1) {
			t.Errorf("expected sampleCount 1, got %v", params["sampleCount"])
		}
	})

	t.Run("should pass seed when provided", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		seed := 42
		opts.Seed = &seed
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["seed"] != float64(42) {
			t.Errorf("expected seed 42, got %v", params["seed"])
		}
	})

	t.Run("should pass aspect ratio when provided", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		ar := "16:9"
		opts.AspectRatio = &ar
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["aspectRatio"] != "16:9" {
			t.Errorf("expected aspectRatio '16:9', got %v", params["aspectRatio"])
		}
	})

	t.Run("should convert resolution to Google format", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		res := "1920x1080"
		opts.Resolution = &res
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["resolution"] != "1080p" {
			t.Errorf("expected resolution '1080p', got %v", params["resolution"])
		}
	})

	t.Run("should pass duration as durationSeconds", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		dur := float64(5)
		opts.Duration = &dur
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["durationSeconds"] != float64(5) {
			t.Errorf("expected durationSeconds 5, got %v", params["durationSeconds"])
		}
	})

	t.Run("should pass n as sampleCount", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://example.com/video1.mp4"}},
				{"video": map[string]any{"uri": "https://example.com/video2.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.N = 2
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["sampleCount"] != float64(2) {
			t.Errorf("expected sampleCount 2, got %v", params["sampleCount"])
		}
	})

	t.Run("should return video with correct data (URL with API key)", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://generativelanguage.googleapis.com/files/video-123.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Videos) != 1 {
			t.Fatalf("expected 1 video, got %d", len(result.Videos))
		}

		v, ok := result.Videos[0].(videomodel.VideoDataURL)
		if !ok {
			t.Fatalf("expected VideoDataURL, got %T", result.Videos[0])
		}
		expectedURL := "https://generativelanguage.googleapis.com/files/video-123.mp4?key=test-api-key"
		if v.URL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, v.URL)
		}
		if v.MediaType != "video/mp4" {
			t.Errorf("expected mediaType 'video/mp4', got %q", v.MediaType)
		}
	})

	t.Run("should append API key with & when URL already has query params", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://generativelanguage.googleapis.com/files/video-123.mp4?param=value"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		v := result.Videos[0].(videomodel.VideoDataURL)
		expectedURL := "https://generativelanguage.googleapis.com/files/video-123.mp4?param=value&key=test-api-key"
		if v.URL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, v.URL)
		}
	})

	t.Run("should return multiple videos", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://example.com/video1.mp4"}},
				{"video": map[string]any{"uri": "https://example.com/video2.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.N = 2
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Videos) != 2 {
			t.Fatalf("expected 2 videos, got %d", len(result.Videos))
		}

		v0 := result.Videos[0].(videomodel.VideoDataURL)
		if !strings.Contains(v0.URL, "video1.mp4") {
			t.Errorf("expected URL to contain 'video1.mp4', got %q", v0.URL)
		}
		v1 := result.Videos[1].(videomodel.VideoDataURL)
		if !strings.Contains(v1.URL, "video2.mp4") {
			t.Errorf("expected URL to contain 'video2.mp4', got %q", v1.URL)
		}
	})

	t.Run("should return warnings array", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}

func TestGoogleVideoModel_ResponseMetadata(t *testing.T) {
	t.Run("should include timestamp and modelId in response", func(t *testing.T) {
		testDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		server, _ := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := NewGoogleVideoModel("veo-3.1-generate-preview", GoogleVideoModelConfig{
			Provider: "google.generative-ai",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{"x-goog-api-key": "test-api-key"}
			},
			Internal: &GoogleVideoModelInternal{
				CurrentDate: func() time.Time { return testDate },
			},
		})

		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		if !result.Response.Timestamp.Equal(testDate) {
			t.Errorf("expected timestamp %v, got %v", testDate, result.Response.Timestamp)
		}
		if result.Response.ModelID != "veo-3.1-generate-preview" {
			t.Errorf("expected modelId 'veo-3.1-generate-preview', got %q", result.Response.ModelID)
		}
		if result.Response.Headers == nil {
			t.Error("expected response headers to be defined")
		}
	})
}

func TestGoogleVideoModel_ProviderMetadata(t *testing.T) {
	t.Run("should include video metadata", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://example.com/video.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		googleMeta, ok := result.ProviderMetadata["google"]
		if !ok {
			t.Fatal("expected 'google' in providerMetadata")
		}
		videos, ok := googleMeta["videos"]
		if !ok {
			t.Fatal("expected 'videos' in google metadata")
		}
		videoSlice, ok := videos.([]map[string]any)
		if !ok {
			t.Fatalf("expected []map[string]any, got %T", videos)
		}
		if len(videoSlice) != 1 {
			t.Fatalf("expected 1 video metadata, got %d", len(videoSlice))
		}
		if videoSlice[0]["uri"] != "https://example.com/video.mp4" {
			t.Errorf("expected uri 'https://example.com/video.mp4', got %v", videoSlice[0]["uri"])
		}
	})
}

func TestGoogleVideoModel_ImageToVideo(t *testing.T) {
	t.Run("should send image as inlineData", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.Image = videomodel.VideoFileData{
			Data:      videomodel.VideoFileDataString{Value: "base64-image-data"},
			MediaType: "image/png",
		}

		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		instances := body["instances"].([]any)
		inst := instances[0].(map[string]any)
		image, ok := inst["image"].(map[string]any)
		if !ok {
			t.Fatalf("expected image in instance, got %v", inst)
		}
		inlineData, ok := image["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData in image, got %v", image)
		}
		if inlineData["mimeType"] != "image/png" {
			t.Errorf("expected mimeType 'image/png', got %v", inlineData["mimeType"])
		}
		if inlineData["data"] != "base64-image-data" {
			t.Errorf("expected data 'base64-image-data', got %v", inlineData["data"])
		}
	})

	t.Run("should warn when URL-based image is provided", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.Image = videomodel.VideoFileURL{
			URL: "https://example.com/image.png",
		}

		result, err := model.DoGenerate(opts)
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
		if w.Feature != "URL-based image input" {
			t.Errorf("expected feature 'URL-based image input', got %q", w.Feature)
		}
	})
}

func TestGoogleVideoModel_ProviderOptions(t *testing.T) {
	t.Run("should pass personGeneration option", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.ProviderOptions = shared.ProviderOptions{
			"google": map[string]any{
				"pollIntervalMs":   float64(10),
				"personGeneration": "allow_adult",
			},
		}

		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["personGeneration"] != "allow_adult" {
			t.Errorf("expected personGeneration 'allow_adult', got %v", params["personGeneration"])
		}
	})

	t.Run("should pass negativePrompt option", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.ProviderOptions = shared.ProviderOptions{
			"google": map[string]any{
				"pollIntervalMs": float64(10),
				"negativePrompt": "blurry, low quality",
			},
		}

		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		params := body["parameters"].(map[string]any)
		if params["negativePrompt"] != "blurry, low quality" {
			t.Errorf("expected negativePrompt 'blurry, low quality', got %v", params["negativePrompt"])
		}
	})

	t.Run("should pass referenceImages option", func(t *testing.T) {
		server, capture := createVideoTestServer(videoTestServerOpts{})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		refData := "reference-image-data"
		gcsURI := "gs://bucket/reference.png"
		opts.ProviderOptions = shared.ProviderOptions{
			"google": map[string]any{
				"pollIntervalMs": float64(10),
				"referenceImages": []map[string]any{
					{"bytesBase64Encoded": refData},
					{"gcsUri": gcsURI},
				},
			},
		}

		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		body := capture.BodyJSON()
		instances := body["instances"].([]any)
		inst := instances[0].(map[string]any)
		refImages, ok := inst["referenceImages"].([]any)
		if !ok {
			t.Fatalf("expected referenceImages in instance, got %v", inst)
		}
		if len(refImages) != 2 {
			t.Fatalf("expected 2 reference images, got %d", len(refImages))
		}

		// First reference image should have inlineData
		ref0 := refImages[0].(map[string]any)
		inlineData, ok := ref0["inlineData"].(map[string]any)
		if !ok {
			t.Fatalf("expected inlineData in first ref image, got %v", ref0)
		}
		if inlineData["mimeType"] != "image/png" {
			t.Errorf("expected mimeType 'image/png', got %v", inlineData["mimeType"])
		}
		if inlineData["data"] != "reference-image-data" {
			t.Errorf("expected data 'reference-image-data', got %v", inlineData["data"])
		}

		// Second reference image should have gcsUri
		ref1 := refImages[1].(map[string]any)
		if ref1["gcsUri"] != "gs://bucket/reference.png" {
			t.Errorf("expected gcsUri 'gs://bucket/reference.png', got %v", ref1["gcsUri"])
		}
	})
}

func TestGoogleVideoModel_ErrorHandling(t *testing.T) {
	t.Run("should throw error when no operation name is returned", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			NoOperationName: true,
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		_, err := model.DoGenerate(opts)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "No operation name returned from API") {
			t.Errorf("expected error to contain 'No operation name returned from API', got: %v", err)
		}
	})

	t.Run("should throw error when operation fails", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			OperationError: map[string]any{
				"code":    float64(400),
				"message": "Invalid request",
				"status":  "INVALID_ARGUMENT",
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		_, err := model.DoGenerate(opts)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "Invalid request") {
			t.Errorf("expected error to contain 'Invalid request', got: %v", err)
		}
	})

	t.Run("should throw error when no videos in response", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		_, err := model.DoGenerate(opts)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "No videos in response") {
			t.Errorf("expected error to contain 'No videos in response', got: %v", err)
		}
	})
}

func TestGoogleVideoModel_PollingBehavior(t *testing.T) {
	t.Run("should poll until operation is done", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			PollsUntilDone: 3,
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://example.com/final-video.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		if len(result.Videos) != 1 {
			t.Fatalf("expected 1 video, got %d", len(result.Videos))
		}
		v := result.Videos[0].(videomodel.VideoDataURL)
		if !strings.Contains(v.URL, "final-video.mp4") {
			t.Errorf("expected URL to contain 'final-video.mp4', got %q", v.URL)
		}
	})

	t.Run("should timeout after pollTimeoutMs", func(t *testing.T) {
		// Create a server that never returns done
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if strings.Contains(r.URL.Path, ":predictLongRunning") {
				json.NewEncoder(w).Encode(map[string]any{
					"name": "operations/timeout-test",
					"done": false,
				})
				return
			}

			if strings.Contains(r.URL.Path, "operations/timeout-test") {
				json.NewEncoder(w).Encode(map[string]any{
					"name": "operations/timeout-test",
					"done": false,
				})
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.ProviderOptions = shared.ProviderOptions{
			"google": map[string]any{
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(50),
			},
		}

		_, err := model.DoGenerate(opts)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "timed out") {
			t.Errorf("expected error to contain 'timed out', got: %v", err)
		}
	})

	t.Run("should respect abort signal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		callCount := 0

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			if strings.Contains(r.URL.Path, ":predictLongRunning") {
				json.NewEncoder(w).Encode(map[string]any{
					"name": "operations/abort-test",
					"done": false,
				})
				return
			}

			if strings.Contains(r.URL.Path, "operations/abort-test") {
				callCount++
				cancel() // Cancel context when polled
				json.NewEncoder(w).Encode(map[string]any{
					"name": "operations/abort-test",
					"done": false,
				})
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		opts.Ctx = ctx
		opts.ProviderOptions = shared.ProviderOptions{
			"google": map[string]any{
				"pollIntervalMs": float64(10),
			},
		}

		_, err := model.DoGenerate(opts)

		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "aborted") && !strings.Contains(err.Error(), "canceled") && !strings.Contains(err.Error(), "context") {
			t.Errorf("expected error to indicate abort/cancel, got: %v", err)
		}
	})
}

func TestGoogleVideoModel_MediaType(t *testing.T) {
	t.Run("should always return video/mp4 as media type", func(t *testing.T) {
		server, _ := createVideoTestServer(videoTestServerOpts{
			Videos: []map[string]any{
				{"video": map[string]any{"uri": "https://example.com/video.mp4"}},
			},
		})
		defer server.Close()

		model := createTestVideoModel(server.URL)
		opts := defaultVideoOptions()
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatal(err)
		}

		v := result.Videos[0].(videomodel.VideoDataURL)
		if v.MediaType != "video/mp4" {
			t.Errorf("expected mediaType 'video/mp4', got %q", v.MediaType)
		}
	})
}
