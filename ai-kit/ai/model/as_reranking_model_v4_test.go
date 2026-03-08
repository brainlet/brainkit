// Ported from: packages/ai/src/model/as-reranking-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsRerankingModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockRerankingModelV4(testutil.MockRerankingModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsRerankingModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsRerankingModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockRerankingModelV4(testutil.MockRerankingModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsRerankingModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsRerankingModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockRerankingModelV3(testutil.MockRerankingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsRerankingModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsRerankingModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockRerankingModelV3(testutil.MockRerankingModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsRerankingModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsRerankingModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockRerankingModelV3(testutil.MockRerankingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsRerankingModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsRerankingModelV4_V3Model_DoRerankCallable(t *testing.T) {
	v3Model := testutil.NewMockRerankingModelV3(testutil.MockRerankingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoRerank: func(_ rerankingmodel.CallOptions) (rerankingmodel.RerankResult, error) {
			return rerankingmodel.RerankResult{
				Ranking: []rerankingmodel.RankedDocument{
					{Index: 1, RelevanceScore: 0.95},
					{Index: 0, RelevanceScore: 0.42},
				},
			}, nil
		},
	})

	result := AsRerankingModelV4(v3Model)

	response, err := result.DoRerank(rerankingmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Ranking) != 2 {
		t.Errorf("expected 2 ranked documents, got %d", len(response.Ranking))
	}
	if response.Ranking[0].RelevanceScore != 0.95 {
		t.Errorf("expected first score 0.95, got %f", response.Ranking[0].RelevanceScore)
	}
}
