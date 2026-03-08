// Ported from: packages/ai/src/model/as-embedding-model-v3.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsEmbeddingModelV3_V3Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV3(original)

	if result != original {
		t.Error("expected same v3 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsEmbeddingModelV3_V3Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-v3",
	})

	result := AsEmbeddingModelV3(original)

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

func TestAsEmbeddingModelV3_V2Model_ConvertsToV3(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV3(v2Model)

	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
	if result == v2Model {
		t.Error("expected a new wrapper, not the original v2 model")
	}
}

func TestAsEmbeddingModelV3_V2Model_PreservesProvider(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider-v2",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV3(v2Model)

	if result.Provider() != "test-provider-v2" {
		t.Errorf("expected provider test-provider-v2, got %s", result.Provider())
	}
}

func TestAsEmbeddingModelV3_V2Model_PreservesModelID(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-v2",
	})

	result := AsEmbeddingModelV3(v2Model)

	if result.ModelID() != "test-model-v2" {
		t.Errorf("expected modelId test-model-v2, got %s", result.ModelID())
	}
}

func TestAsEmbeddingModelV3_V2Model_DoEmbedCallable(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoEmbed: func(_ embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
			return embeddingmodel.Result{
				Embeddings: []embeddingmodel.Embedding{{0.1, 0.2, 0.3}},
			}, nil
		},
	})

	result := AsEmbeddingModelV3(v2Model)

	response, err := result.DoEmbed(embeddingmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Embeddings) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(response.Embeddings))
	}
}
