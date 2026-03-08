// Ported from: packages/ai/src/model/as-image-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsImageModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockImageModelV4(testutil.MockImageModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsImageModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockImageModelV4(testutil.MockImageModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsImageModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsImageModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsImageModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsImageModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsImageModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsImageModelV4_V3Model_DoGenerateCallable(t *testing.T) {
	v3Model := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
			return imagemodel.GenerateResult{
				Images: imagemodel.ImageDataStrings{Values: []string{"base64data"}},
			}, nil
		},
	})

	result := AsImageModelV4(v3Model)

	response, err := result.DoGenerate(imagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	imgs, ok := response.Images.(imagemodel.ImageDataStrings)
	if !ok {
		t.Fatal("expected ImageDataStrings")
	}
	if len(imgs.Values) != 1 {
		t.Errorf("expected 1 image, got %d", len(imgs.Values))
	}
}

func TestAsImageModelV4_V2Model_ConvertsThroughV3ToV4(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV4(v2Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", result.ModelID())
	}
}

func TestAsImageModelV4_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
			return imagemodel.GenerateResult{
				Images: imagemodel.ImageDataStrings{Values: []string{"base64data"}},
			}, nil
		},
	})

	result := AsImageModelV4(v2Model)

	response, err := result.DoGenerate(imagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	imgs, ok := response.Images.(imagemodel.ImageDataStrings)
	if !ok {
		t.Fatal("expected ImageDataStrings")
	}
	if len(imgs.Values) != 1 {
		t.Errorf("expected 1 image, got %d", len(imgs.Values))
	}
}
