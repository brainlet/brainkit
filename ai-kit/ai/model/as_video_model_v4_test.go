// Ported from: packages/ai/src/model/as-video-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsVideoModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockVideoModelV4(testutil.MockVideoModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsVideoModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsVideoModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockVideoModelV4(testutil.MockVideoModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsVideoModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsVideoModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockVideoModelV3(testutil.MockVideoModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsVideoModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsVideoModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockVideoModelV3(testutil.MockVideoModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsVideoModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsVideoModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockVideoModelV3(testutil.MockVideoModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsVideoModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsVideoModelV4_V3Model_DoGenerateCallable(t *testing.T) {
	v3Model := testutil.NewMockVideoModelV3(testutil.MockVideoModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ videomodel.CallOptions) (videomodel.GenerateResult, error) {
			return videomodel.GenerateResult{
				Videos: []videomodel.VideoData{
					videomodel.VideoDataBase64{Data: "base64video", MediaType: "video/mp4"},
				},
			}, nil
		},
	})

	result := AsVideoModelV4(v3Model)

	response, err := result.DoGenerate(videomodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Videos) != 1 {
		t.Errorf("expected 1 video, got %d", len(response.Videos))
	}
}
