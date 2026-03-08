// Ported from: packages/ai/src/model/as-language-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsLanguageModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockLanguageModelV4(testutil.MockLanguageModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsLanguageModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockLanguageModelV4(testutil.MockLanguageModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsLanguageModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsLanguageModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsLanguageModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsLanguageModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsLanguageModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsLanguageModelV4_V3Model_DoGenerateCallable(t *testing.T) {
	v3Model := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
			return languagemodel.GenerateResult{
				Content: []languagemodel.Content{
					languagemodel.Text{Text: "Hello"},
				},
				FinishReason: languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonStop,
				},
				Usage: languagemodel.Usage{
					InputTokens:  languagemodel.InputTokenUsage{Total: intPtr(10)},
					OutputTokens: languagemodel.OutputTokenUsage{Total: intPtr(5)},
				},
			}, nil
		},
	})

	result := AsLanguageModelV4(v3Model)

	response, err := result.DoGenerate(languagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(response.Content))
	}
	if response.FinishReason.Unified != languagemodel.FinishReasonStop {
		t.Errorf("expected finish reason stop, got %s", response.FinishReason.Unified)
	}
}

func TestAsLanguageModelV4_V2Model_ConvertsThroughV3ToV4(t *testing.T) {
	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV4(v2Model)

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

func TestAsLanguageModelV4_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
			return languagemodel.GenerateResult{
				Content: []languagemodel.Content{
					languagemodel.Text{Text: "Hello"},
				},
				FinishReason: languagemodel.FinishReason{
					Unified: languagemodel.FinishReasonStop,
				},
				Usage: languagemodel.Usage{
					InputTokens:  languagemodel.InputTokenUsage{Total: intPtr(10)},
					OutputTokens: languagemodel.OutputTokenUsage{Total: intPtr(5)},
				},
			}, nil
		},
	})

	result := AsLanguageModelV4(v2Model)

	response, err := result.DoGenerate(languagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(response.Content))
	}
}
