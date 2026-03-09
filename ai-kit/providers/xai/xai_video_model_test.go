// Ported from: packages/xai/src/xai-video-model.test.ts
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

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// createVideoTestServer creates a test server for video model tests.
// It handles /videos/generations, /videos/edits, and /videos/{requestId} endpoints.
// The statusResponse is used to configure the polling response.
func createVideoTestServer(
	createResponse map[string]any,
	statusResponse map[string]any,
	headers map[string]string,
) (*httptest.Server, *videoRequestCapture) {
	capture := &videoRequestCapture{}

	mux := http.NewServeMux()

	// Video generation endpoint
	mux.HandleFunc("/videos/generations", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Bodies = append(capture.Bodies, bodyBytes)
		capture.AllHeaders = append(capture.AllHeaders, r.Header.Clone())
		capture.Methods = append(capture.Methods, r.Method)
		capture.URLs = append(capture.URLs, r.URL.Path)

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createResponse)
	})

	// Video editing endpoint
	mux.HandleFunc("/videos/edits", func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		capture.Bodies = append(capture.Bodies, bodyBytes)
		capture.AllHeaders = append(capture.AllHeaders, r.Header.Clone())
		capture.Methods = append(capture.Methods, r.Method)
		capture.URLs = append(capture.URLs, r.URL.Path)

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(createResponse)
	})

	// Video status polling endpoint
	mux.HandleFunc("/videos/", func(w http.ResponseWriter, r *http.Request) {
		// Skip if this is /videos/generations or /videos/edits
		if strings.HasSuffix(r.URL.Path, "/generations") || strings.HasSuffix(r.URL.Path, "/edits") {
			return
		}
		capture.Bodies = append(capture.Bodies, nil)
		capture.AllHeaders = append(capture.AllHeaders, r.Header.Clone())
		capture.Methods = append(capture.Methods, r.Method)
		capture.URLs = append(capture.URLs, r.URL.Path)

		for k, v := range headers {
			w.Header().Set(k, v)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statusResponse)
	})

	server := httptest.NewServer(mux)
	return server, capture
}

// videoRequestCapture captures multiple HTTP request details for video tests.
type videoRequestCapture struct {
	Bodies     [][]byte
	AllHeaders []http.Header
	Methods    []string
	URLs       []string
}

func (rc *videoRequestCapture) BodyJSON(index int) map[string]any {
	if index >= len(rc.Bodies) || rc.Bodies[index] == nil {
		return nil
	}
	var result map[string]any
	json.Unmarshal(rc.Bodies[index], &result)
	return result
}

func createVideoModel(serverURL string) *XaiVideoModel {
	return NewXaiVideoModel("grok-imagine-video", XaiVideoModelConfig{
		Provider: "xai.video",
		BaseURL:  serverURL,
		Headers:  func() map[string]string { return map[string]string{"api-key": "test-key"} },
		CurrentDate: func() time.Time {
			return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})
}

func defaultVideoOptions(prompt string) videomodel.CallOptions {
	return videomodel.CallOptions{
		Prompt: &prompt,
		N:      1,
		ProviderOptions: shared.ProviderOptions{
			"xai": map[string]interface{}{
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		},
		Ctx: context.Background(),
	}
}

var createVideoResponse = map[string]any{
	"request_id": "req-123",
}

var doneStatusResponse = map[string]any{
	"status": "done",
	"video": map[string]any{
		"url":                "https://vidgen.x.ai/output/video-001.mp4",
		"duration":           float64(5),
		"respect_moderation": true,
	},
	"model": "grok-imagine-video",
}

func TestXaiVideoModel_Constructor(t *testing.T) {
	t.Run("should expose correct provider and model information", func(t *testing.T) {
		model := NewXaiVideoModel("grok-imagine-video", XaiVideoModelConfig{
			Provider: "xai.video",
			BaseURL:  "https://api.x.ai/v1",
			Headers:  func() map[string]string { return map[string]string{} },
		})

		if model.Provider() != "xai.video" {
			t.Errorf("expected provider 'xai.video', got %q", model.Provider())
		}
		if model.ModelID() != "grok-imagine-video" {
			t.Errorf("expected model ID 'grok-imagine-video', got %q", model.ModelID())
		}
		if model.SpecificationVersion() != "v3" {
			t.Errorf("expected specificationVersion 'v3', got %q", model.SpecificationVersion())
		}
		max, err := model.MaxVideosPerCall()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if max == nil || *max != 1 {
			t.Errorf("expected maxVideosPerCall 1, got %v", max)
		}
	})
}

func TestXaiVideoModel_RequestBody(t *testing.T) {
	t.Run("should send correct request body with model and prompt", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "A chicken flying into the sunset"
		_, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(capture.Methods) < 1 {
			t.Fatal("expected at least 1 request")
		}
		if capture.Methods[0] != "POST" {
			t.Errorf("expected POST, got %s", capture.Methods[0])
		}
		if capture.URLs[0] != "/videos/generations" {
			t.Errorf("expected /videos/generations, got %s", capture.URLs[0])
		}

		body := capture.BodyJSON(0)
		if body["model"] != "grok-imagine-video" {
			t.Errorf("expected model 'grok-imagine-video', got %v", body["model"])
		}
		if body["prompt"] != "A chicken flying into the sunset" {
			t.Errorf("expected prompt, got %v", body["prompt"])
		}
	})
}

func TestXaiVideoModel_PollingURL(t *testing.T) {
	t.Run("should poll the correct status URL", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "A chicken flying into the sunset"
		_, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(capture.Methods) < 2 {
			t.Fatal("expected at least 2 requests (create + poll)")
		}
		if capture.Methods[1] != "GET" {
			t.Errorf("expected GET for poll, got %s", capture.Methods[1])
		}
		if capture.URLs[1] != "/videos/req-123" {
			t.Errorf("expected /videos/req-123, got %s", capture.URLs[1])
		}
	})
}

func TestXaiVideoModel_Duration(t *testing.T) {
	t.Run("should send duration in request body", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "A chicken flying into the sunset"
		opts := defaultVideoOptions(prompt)
		dur := float64(10)
		opts.Duration = &dur
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if body["duration"] != float64(10) {
			t.Errorf("expected duration 10, got %v", body["duration"])
		}
	})
}

func TestXaiVideoModel_AspectRatio(t *testing.T) {
	t.Run("should send aspect_ratio in request body", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "A chicken flying into the sunset"
		opts := defaultVideoOptions(prompt)
		ar := "9:16"
		opts.AspectRatio = &ar
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if body["aspect_ratio"] != "9:16" {
			t.Errorf("expected aspect_ratio '9:16', got %v", body["aspect_ratio"])
		}
	})
}

func TestXaiVideoModel_Resolution(t *testing.T) {
	t.Run("should map SDK resolution 1280x720 to 720p", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "1280x720"
		opts.Resolution = &res
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if body["resolution"] != "720p" {
			t.Errorf("expected resolution '720p', got %v", body["resolution"])
		}
	})

	t.Run("should map SDK resolution 854x480 to 480p", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "854x480"
		opts.Resolution = &res
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if body["resolution"] != "480p" {
			t.Errorf("expected resolution '480p', got %v", body["resolution"])
		}
	})

	t.Run("should prefer provider option resolution over SDK resolution", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "1280x720"
		opts.Resolution = &res
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"resolution":     "480p",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if body["resolution"] != "480p" {
			t.Errorf("expected resolution '480p', got %v", body["resolution"])
		}
	})

	t.Run("should warn for unrecognized resolution format", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "1920x1080"
		opts.Resolution = &res
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "resolution" {
				resWarning = true
			}
		}
		if !resWarning {
			t.Error("expected unsupported warning for resolution")
		}
	})
}

func TestXaiVideoModel_ImageToVideo(t *testing.T) {
	t.Run("should send image object from URL-based image input", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.Image = videomodel.VideoFileURL{
			URL: "https://example.com/image.png",
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		img, ok := body["image"].(map[string]interface{})
		if !ok {
			t.Fatal("expected image in body")
		}
		if img["url"] != "https://example.com/image.png" {
			t.Errorf("expected image URL 'https://example.com/image.png', got %v", img["url"])
		}
	})

	t.Run("should send image object with data URI from file data bytes", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.Image = videomodel.VideoFileData{
			MediaType: "image/png",
			Data:      videomodel.VideoFileDataBytes{Data: []byte{137, 80, 78, 71}}, // PNG magic bytes
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		img, ok := body["image"].(map[string]interface{})
		if !ok {
			t.Fatal("expected image in body")
		}
		imgURL, ok := img["url"].(string)
		if !ok {
			t.Fatal("expected url in image")
		}
		if !strings.HasPrefix(imgURL, "data:image/png;base64,") {
			t.Errorf("expected data URI prefix, got %q", imgURL)
		}
	})

	t.Run("should send image object with data URI from base64 string", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.Image = videomodel.VideoFileData{
			MediaType: "image/jpeg",
			Data:      videomodel.VideoFileDataString{Value: "aGVsbG8="},
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		img, ok := body["image"].(map[string]interface{})
		if !ok {
			t.Fatal("expected image in body")
		}
		if img["url"] != "data:image/jpeg;base64,aGVsbG8=" {
			t.Errorf("expected data URI, got %v", img["url"])
		}
	})
}

func TestXaiVideoModel_VideoEditing(t *testing.T) {
	t.Run("should send video object to /videos/edits for video editing", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"videoUrl":       "https://example.com/source-video.mp4",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capture.URLs[0] != "/videos/edits" {
			t.Errorf("expected /videos/edits, got %s", capture.URLs[0])
		}

		body := capture.BodyJSON(0)
		vid, ok := body["video"].(map[string]interface{})
		if !ok {
			t.Fatal("expected video in body")
		}
		if vid["url"] != "https://example.com/source-video.mp4" {
			t.Errorf("expected video URL, got %v", vid["url"])
		}
	})

	t.Run("should warn about duration in edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		dur := float64(10)
		opts.Duration = &dur
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"videoUrl":       "https://example.com/source-video.mp4",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var durWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "duration" {
				durWarning = true
			}
		}
		if !durWarning {
			t.Error("expected unsupported warning for duration in edit mode")
		}
	})

	t.Run("should warn about aspectRatio in edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		ar := "16:9"
		opts.AspectRatio = &ar
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"videoUrl":       "https://example.com/source-video.mp4",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var arWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "aspectRatio" {
				arWarning = true
			}
		}
		if !arWarning {
			t.Error("expected unsupported warning for aspectRatio in edit mode")
		}
	})

	t.Run("should warn about resolution in edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "1280x720"
		opts.Resolution = &res
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"videoUrl":       "https://example.com/source-video.mp4",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var resWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "resolution" {
				resWarning = true
			}
		}
		if !resWarning {
			t.Error("expected unsupported warning for resolution in edit mode")
		}
	})

	t.Run("should not warn about duration outside edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		dur := float64(10)
		opts.Duration = &dur
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "duration" {
				t.Error("unexpected warning for duration outside edit mode")
			}
		}
	})

	t.Run("should not warn about aspectRatio outside edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		ar := "16:9"
		opts.AspectRatio = &ar
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "aspectRatio" {
				t.Error("unexpected warning for aspectRatio outside edit mode")
			}
		}
	})

	t.Run("should not warn about resolution outside edit mode", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		res := "1280x720"
		opts.Resolution = &res
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "resolution" {
				t.Error("unexpected warning for resolution outside edit mode")
			}
		}
	})

	t.Run("should omit duration, aspect_ratio, and resolution from body in edit mode", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		dur := float64(10)
		opts.Duration = &dur
		ar := "16:9"
		opts.AspectRatio = &ar
		res := "1280x720"
		opts.Resolution = &res
		opts.ProviderOptions = shared.ProviderOptions{
			"xai": map[string]interface{}{
				"videoUrl":       "https://example.com/source-video.mp4",
				"pollIntervalMs": float64(10),
				"pollTimeoutMs":  float64(5000),
			},
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		body := capture.BodyJSON(0)
		if _, ok := body["duration"]; ok {
			t.Error("expected no duration in body for edit mode")
		}
		if _, ok := body["aspect_ratio"]; ok {
			t.Error("expected no aspect_ratio in body for edit mode")
		}
		if _, ok := body["resolution"]; ok {
			t.Error("expected no resolution in body for edit mode")
		}
	})
}

func TestXaiVideoModel_Headers(t *testing.T) {
	t.Run("should pass headers to requests", func(t *testing.T) {
		server, capture := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := NewXaiVideoModel("grok-imagine-video", XaiVideoModelConfig{
			Provider: "xai.video",
			BaseURL:  server.URL,
			Headers: func() map[string]string {
				return map[string]string{
					"Authorization": "Bearer custom-token",
					"X-Custom":      "value",
				}
			},
			CurrentDate: func() time.Time {
				return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		})

		prompt := "test"
		opts := defaultVideoOptions(prompt)
		reqHeader := "request-value"
		opts.Headers = map[string]*string{
			"X-Request-Header": &reqHeader,
		}
		_, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(capture.AllHeaders) < 2 {
			t.Fatal("expected at least 2 requests")
		}

		// Create request headers
		if capture.AllHeaders[0].Get("Authorization") != "Bearer custom-token" {
			t.Errorf("expected Authorization header on create request")
		}
		if capture.AllHeaders[0].Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header on create request")
		}

		// Poll request headers
		if capture.AllHeaders[1].Get("Authorization") != "Bearer custom-token" {
			t.Errorf("expected Authorization header on poll request")
		}
		if capture.AllHeaders[1].Get("X-Custom") != "value" {
			t.Errorf("expected X-Custom header on poll request")
		}
	})
}

func TestXaiVideoModel_VideoResult(t *testing.T) {
	t.Run("should return video with correct URL and media type", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Videos) != 1 {
			t.Fatalf("expected 1 video, got %d", len(result.Videos))
		}

		urlVideo, ok := result.Videos[0].(videomodel.VideoDataURL)
		if !ok {
			t.Fatalf("expected VideoDataURL, got %T", result.Videos[0])
		}
		if urlVideo.URL != "https://vidgen.x.ai/output/video-001.mp4" {
			t.Errorf("expected video URL, got %q", urlVideo.URL)
		}
		if urlVideo.MediaType != "video/mp4" {
			t.Errorf("expected media type 'video/mp4', got %q", urlVideo.MediaType)
		}
	})
}

func TestXaiVideoModel_DoneWithoutStatus(t *testing.T) {
	t.Run("should handle done response without status field", func(t *testing.T) {
		doneWithoutStatus := map[string]any{
			"video": map[string]any{
				"url":                "https://vidgen.x.ai/output/video-001.mp4",
				"duration":           float64(5),
				"respect_moderation": true,
			},
			"model": "grok-imagine-video",
		}
		server, _ := createVideoTestServer(createVideoResponse, doneWithoutStatus, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Videos) != 1 {
			t.Fatalf("expected 1 video, got %d", len(result.Videos))
		}

		urlVideo, ok := result.Videos[0].(videomodel.VideoDataURL)
		if !ok {
			t.Fatalf("expected VideoDataURL, got %T", result.Videos[0])
		}
		if urlVideo.URL != "https://vidgen.x.ai/output/video-001.mp4" {
			t.Errorf("expected video URL, got %q", urlVideo.URL)
		}
	})
}

func TestXaiVideoModel_Warnings(t *testing.T) {
	t.Run("should return empty warnings for supported features", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should warn about unsupported fps", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		fps := 30
		opts.FPS = &fps
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var fpsWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "fps" {
				fpsWarning = true
			}
		}
		if !fpsWarning {
			t.Error("expected unsupported warning for fps")
		}
	})

	t.Run("should warn about unsupported seed", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		seed := 42
		opts.Seed = &seed
		result, err := model.DoGenerate(opts)
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

	t.Run("should warn when n > 1", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.N = 3
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var nWarning bool
		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "n" {
				nWarning = true
			}
		}
		if !nWarning {
			t.Error("expected unsupported warning for n")
		}
	})

	t.Run("should not warn when n is 1", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := defaultVideoOptions(prompt)
		opts.N = 1
		result, err := model.DoGenerate(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for _, w := range result.Warnings {
			if uw, ok := w.(shared.UnsupportedWarning); ok && uw.Feature == "n" {
				t.Error("unexpected warning for n=1")
			}
		}
	})
}

func TestXaiVideoModel_ResponseMetadata(t *testing.T) {
	t.Run("should include timestamp, headers, and modelId in response", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Response.ModelID != "grok-imagine-video" {
			t.Errorf("expected modelId 'grok-imagine-video', got %q", result.Response.ModelID)
		}
		expectedDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		if !result.Response.Timestamp.Equal(expectedDate) {
			t.Errorf("expected timestamp %v, got %v", expectedDate, result.Response.Timestamp)
		}
	})
}

func TestXaiVideoModel_ProviderMetadata(t *testing.T) {
	t.Run("should include requestId, videoUrl, and duration", func(t *testing.T) {
		server, _ := createVideoTestServer(createVideoResponse, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		result, err := model.DoGenerate(defaultVideoOptions(prompt))
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
		if xaiMeta["requestId"] != "req-123" {
			t.Errorf("expected requestId 'req-123', got %v", xaiMeta["requestId"])
		}
		if xaiMeta["videoUrl"] != "https://vidgen.x.ai/output/video-001.mp4" {
			t.Errorf("expected videoUrl, got %v", xaiMeta["videoUrl"])
		}
		if xaiMeta["duration"] != float64(5) {
			t.Errorf("expected duration 5, got %v", xaiMeta["duration"])
		}
	})
}

func TestXaiVideoModel_ErrorHandling(t *testing.T) {
	t.Run("should error when status is expired", func(t *testing.T) {
		expiredStatus := map[string]any{
			"status": "expired",
			"model":  "grok-imagine-video",
		}
		server, _ := createVideoTestServer(createVideoResponse, expiredStatus, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		_, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err == nil {
			t.Fatal("expected error for expired status")
		}
		if !strings.Contains(err.Error(), "expired") {
			t.Errorf("expected error to contain 'expired', got %q", err.Error())
		}
	})

	t.Run("should error when no request_id is returned", func(t *testing.T) {
		emptyCreate := map[string]any{}
		server, _ := createVideoTestServer(emptyCreate, doneStatusResponse, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		_, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err == nil {
			t.Fatal("expected error for missing request_id")
		}
		if !strings.Contains(err.Error(), "No request_id") {
			t.Errorf("expected error to contain 'No request_id', got %q", err.Error())
		}
	})

	t.Run("should error when video URL is missing on done status", func(t *testing.T) {
		doneNoVideo := map[string]any{
			"status": "done",
			"video":  nil,
			"model":  "grok-imagine-video",
		}
		server, _ := createVideoTestServer(createVideoResponse, doneNoVideo, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		_, err := model.DoGenerate(defaultVideoOptions(prompt))
		if err == nil {
			t.Fatal("expected error for missing video URL")
		}
		if !strings.Contains(err.Error(), "no video URL") && !strings.Contains(err.Error(), "Video generation completed") {
			t.Errorf("expected error about missing video URL, got %q", err.Error())
		}
	})

	t.Run("should error on timeout", func(t *testing.T) {
		pendingStatus := map[string]any{
			"status": "pending",
			"model":  "grok-imagine-video",
		}
		server, _ := createVideoTestServer(createVideoResponse, pendingStatus, nil)
		defer server.Close()

		model := createVideoModel(server.URL)
		prompt := "test"
		opts := videomodel.CallOptions{
			Prompt: &prompt,
			N:      1,
			ProviderOptions: shared.ProviderOptions{
				"xai": map[string]interface{}{
					"pollIntervalMs": float64(10),
					"pollTimeoutMs":  float64(50),
				},
			},
			Ctx: context.Background(),
		}
		_, err := model.DoGenerate(opts)
		if err == nil {
			t.Fatal("expected timeout error")
		}
		if !strings.Contains(err.Error(), "timed out") && !strings.Contains(err.Error(), "Timeout") {
			t.Errorf("expected timeout error, got %q", err.Error())
		}
	})
}

// unused import suppression
var _ = fmt.Sprintf
