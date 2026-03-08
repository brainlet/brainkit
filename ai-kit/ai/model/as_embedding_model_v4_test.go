// Ported from: packages/ai/src/model/as-embedding-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsEmbeddingModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockEmbeddingModelV4(testutil.MockEmbeddingModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsEmbeddingModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockEmbeddingModelV4(testutil.MockEmbeddingModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsEmbeddingModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsEmbeddingModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsEmbeddingModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsEmbeddingModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsEmbeddingModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsEmbeddingModelV4_V3Model_DoEmbedCallable(t *testing.T) {
	v3Model := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoEmbed: func(_ embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
			return embeddingmodel.Result{
				Embeddings: []embeddingmodel.Embedding{{0.1, 0.2, 0.3}},
			}, nil
		},
	})

	result := AsEmbeddingModelV4(v3Model)

	response, err := result.DoEmbed(embeddingmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Embeddings) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(response.Embeddings))
	}
}

func TestAsEmbeddingModelV4_V2Model_ConvertsThroughV3ToV4(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsEmbeddingModelV4(v2Model)

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

func TestAsEmbeddingModelV4_V2Model_DoEmbedCallable(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoEmbed: func(_ embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
			return embeddingmodel.Result{
				Embeddings: []embeddingmodel.Embedding{{0.1, 0.2, 0.3}},
			}, nil
		},
	})

	result := AsEmbeddingModelV4(v2Model)

	response, err := result.DoEmbed(embeddingmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Embeddings) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(response.Embeddings))
	}
}
