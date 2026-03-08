// Ported from: packages/ai/src/model/as-image-model-v3.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsImageModelV3_V3Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV3(original)

	if result != original {
		t.Error("expected same v3 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsImageModelV3_V3Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-v3",
	})

	result := AsImageModelV3(original)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsImageModelV3_V2Model_ConvertsToV3(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV3(v2Model)

	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
	if result == v2Model {
		t.Error("expected a new wrapper, not the original v2 model")
	}
}

func TestAsImageModelV3_V2Model_PreservesProvider(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider-v2",
		ModelID:  "test-model-id",
	})

	result := AsImageModelV3(v2Model)

	if result.Provider() != "test-provider-v2" {
		t.Errorf("expected provider test-provider-v2, got %s", result.Provider())
	}
}

func TestAsImageModelV3_V2Model_PreservesModelID(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-v2",
	})

	result := AsImageModelV3(v2Model)

	if result.ModelID() != "test-model-v2" {
		t.Errorf("expected modelId test-model-v2, got %s", result.ModelID())
	}
}

func TestAsImageModelV3_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
			return imagemodel.GenerateResult{
				Images: imagemodel.ImageDataStrings{Values: []string{"base64data"}},
			}, nil
		},
	})

	result := AsImageModelV3(v2Model)

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
