// Ported from: packages/ai/src/model/as-provider-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsProviderV4_V4Provider_ReturnsSameProvider(t *testing.T) {
	lm := testutil.NewMockLanguageModelV4(testutil.MockLanguageModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model",
	})

	originalProvider := testutil.NewMockProviderV4(testutil.MockProviderV4Options{
		LanguageModels: map[string]languagemodel.LanguageModel{
			"test-model": lm,
		},
	})

	result := AsProviderV4(originalProvider)

	if result != originalProvider {
		t.Error("expected same v4 provider to be returned unchanged")
	}
}

func TestAsProviderV4_V3Provider_ConvertsToV4(t *testing.T) {
	lm := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model",
	})

	v3Provider := testutil.NewMockProviderV3(testutil.MockProviderV3Options{
		LanguageModels: map[string]languagemodel.LanguageModel{
			"test-model": lm,
		},
	})

	result := AsProviderV4(v3Provider)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsProviderV4_V3Provider_WrapsLanguageModelsToV4(t *testing.T) {
	lm := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model",
	})

	v3Provider := testutil.NewMockProviderV3(testutil.MockProviderV3Options{
		LanguageModels: map[string]languagemodel.LanguageModel{
			"test-model": lm,
		},
	})

	result := AsProviderV4(v3Provider)

	model, err := result.LanguageModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", model.SpecificationVersion())
	}
	if model.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", model.Provider())
	}
	if model.ModelID() != "test-model" {
		t.Errorf("expected modelId test-model, got %s", model.ModelID())
	}
}

func TestAsProviderV4_V3Provider_WrapsEmbeddingModelsToV4(t *testing.T) {
	em := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-embedding",
	})

	v3Provider := testutil.NewMockProviderV3(testutil.MockProviderV3Options{
		EmbeddingModels: map[string]embeddingmodel.EmbeddingModel{
			"test-embedding": em,
		},
	})

	result := AsProviderV4(v3Provider)

	model, err := result.EmbeddingModel("test-embedding")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", model.SpecificationVersion())
	}
	if model.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", model.Provider())
	}
}

func TestAsProviderV4_V3Provider_WrapsImageModelsToV4(t *testing.T) {
	im := testutil.NewMockImageModelV3(testutil.MockImageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-image",
	})

	v3Provider := testutil.NewMockProviderV3(testutil.MockProviderV3Options{
		ImageModels: map[string]imagemodel.ImageModel{
			"test-image": im,
		},
	})

	result := AsProviderV4(v3Provider)

	model, err := result.ImageModel("test-image")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", model.SpecificationVersion())
	}
	if model.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", model.Provider())
	}
}

func TestAsProviderV4_V3Provider_WrapsRerankingModelsToV4(t *testing.T) {
	rm := testutil.NewMockRerankingModelV3(testutil.MockRerankingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-reranking",
	})

	v3Provider := testutil.NewMockProviderV3(testutil.MockProviderV3Options{
		RerankingModels: map[string]rerankingmodel.RerankingModel{
			"test-reranking": rm,
		},
	})

	result := AsProviderV4(v3Provider)

	model, err := result.RerankingModel("test-reranking")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if model.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", model.SpecificationVersion())
	}
	if model.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", model.Provider())
	}
}
