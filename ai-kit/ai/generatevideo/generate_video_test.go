// Ported from: packages/ai/src/generate-video/generate-video.test.ts
package generatevideo

import (
	"context"
	"encoding/base64"
	"reflect"
	"testing"
)

// mockVideoModel is a mock implementation of VideoModel for testing.
type mockVideoModel struct {
	provider         string
	modelID          string
	maxVideosPerCall int
	doGenerate       func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

func (m *mockVideoModel) Provider() string      { return m.provider }
func (m *mockVideoModel) ModelID() string        { return m.modelID }
func (m *mockVideoModel) MaxVideosPerCall() int  { return m.maxVideosPerCall }
func (m *mockVideoModel) DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
	return m.doGenerate(ctx, opts)
}

func newMockVideoModel(maxVideosPerCall int, doGenerate func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)) *mockVideoModel {
	return &mockVideoModel{
		provider:         "mock-provider",
		modelID:          "mock-model-id",
		maxVideosPerCall: maxVideosPerCall,
		doGenerate:       doGenerate,
	}
}

var mp4Base64 = "AAAAIGZ0eXBpc29tAAACAGlzb21pc28yYXZjMW1wNDE="

func decodeMp4() []byte {
	data, _ := base64.StdEncoding.DecodeString(mp4Base64)
	return data
}

func TestGenerateVideo_Base64(t *testing.T) {
	t.Run("should return generated videos with correct mime types", func(t *testing.T) {
		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos: []VideoData{
					{Type: "base64", Data: mp4Base64, MediaType: "video/mp4"},
					{Type: "base64", Data: mp4Base64, MediaType: "video/webm"},
				},
				Warnings:         []Warning{},
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Videos) != 2 {
			t.Errorf("expected 2 videos, got %d", len(result.Videos))
		}
		if result.Videos[0].MediaType != "video/mp4" {
			t.Errorf("expected video/mp4, got %q", result.Videos[0].MediaType)
		}
		if result.Videos[1].MediaType != "video/webm" {
			t.Errorf("expected video/webm, got %q", result.Videos[1].MediaType)
		}
	})

	t.Run("should return the first video", func(t *testing.T) {
		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos: []VideoData{
					{Type: "base64", Data: mp4Base64, MediaType: "video/mp4"},
					{Type: "base64", Data: mp4Base64, MediaType: "video/webm"},
				},
				Warnings:         []Warning{},
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Video.MediaType != "video/mp4" {
			t.Errorf("expected video/mp4, got %q", result.Video.MediaType)
		}
	})
}

func TestGenerateVideo_Binary(t *testing.T) {
	t.Run("should return generated videos from binary data", func(t *testing.T) {
		binaryData := decodeMp4()

		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos: []VideoData{
					{Type: "binary", BinaryData: binaryData, MediaType: "video/mp4"},
				},
				Warnings:         []Warning{},
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Videos) != 1 {
			t.Errorf("expected 1 video, got %d", len(result.Videos))
		}
		if !reflect.DeepEqual(result.Video.Data, binaryData) {
			t.Errorf("video data mismatch")
		}
	})
}

func TestGenerateVideo_Warnings(t *testing.T) {
	t.Run("should return warnings", func(t *testing.T) {
		expectedWarnings := []Warning{
			{Type: "other", Message: "Setting is not supported"},
		}

		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos: []VideoData{
					{Type: "base64", Data: mp4Base64, MediaType: "video/mp4"},
				},
				Warnings:         expectedWarnings,
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(result.Warnings, expectedWarnings) {
			t.Errorf("expected warnings %v, got %v", expectedWarnings, result.Warnings)
		}
	})
}

func TestGenerateVideo_NoVideoError(t *testing.T) {
	t.Run("should return error when no videos are returned", func(t *testing.T) {
		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos:           []VideoData{},
				Warnings:         []Warning{},
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		_, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGenerateVideo_MultipleCalls(t *testing.T) {
	t.Run("should generate videos with multiple calls", func(t *testing.T) {
		// Dispatch based on opts.N since calls run concurrently.
		model := newMockVideoModel(2, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			videos := make([]VideoData, opts.N)
			for i := range videos {
				videos[i] = VideoData{Type: "base64", Data: mp4Base64, MediaType: "video/mp4"}
			}
			return &DoGenerateResult{
				Videos:           videos,
				Warnings:         []Warning{},
				Response:         VideoModelResponseMetadata{ModelID: "test-model-id"},
				ProviderMetadata: VideoModelProviderMetadata{},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
			N:      3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Videos) != 3 {
			t.Errorf("expected 3 videos, got %d", len(result.Videos))
		}
	})
}

func TestGenerateVideo_ProviderMetadata(t *testing.T) {
	t.Run("should return provider metadata", func(t *testing.T) {
		model := newMockVideoModel(0, func(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error) {
			return &DoGenerateResult{
				Videos: []VideoData{
					{Type: "base64", Data: mp4Base64, MediaType: "video/mp4"},
				},
				Warnings: []Warning{},
				Response: VideoModelResponseMetadata{ModelID: "test-model"},
				ProviderMetadata: VideoModelProviderMetadata{
					"testProvider": {
						"videos": []any{map[string]any{"seed": 12345, "duration": 5}},
					},
				},
			}, nil
		})

		result, err := GenerateVideo(context.Background(), GenerateVideoOptions{
			Model:  model,
			Prompt: "a cat walking on a beach",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.ProviderMetadata == nil {
			t.Fatal("expected provider metadata to be non-nil")
		}
		testProvider, ok := result.ProviderMetadata["testProvider"]
		if !ok {
			t.Fatal("expected testProvider in provider metadata")
		}
		videos, ok := testProvider["videos"]
		if !ok {
			t.Fatal("expected videos in testProvider metadata")
		}
		videosArr, ok := videos.([]any)
		if !ok || len(videosArr) != 1 {
			t.Errorf("expected 1 video metadata entry, got %v", videos)
		}
	})
}
