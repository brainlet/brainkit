// Ported from: packages/ai/src/model/as-language-model-v3.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsLanguageModelV3_V3Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV3(original)

	if result != original {
		t.Error("expected same v3 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsLanguageModelV3_V3Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-v3",
	})

	result := AsLanguageModelV3(original)

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

func TestAsLanguageModelV3_V2Model_ConvertsToV3(t *testing.T) {
	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV3(v2Model)

	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
	if result == v2Model {
		t.Error("expected a new wrapper, not the original v2 model")
	}
}

func TestAsLanguageModelV3_V2Model_PreservesProvider(t *testing.T) {
	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider-v2",
		ModelID:  "test-model-id",
	})

	result := AsLanguageModelV3(v2Model)

	if result.Provider() != "test-provider-v2" {
		t.Errorf("expected provider test-provider-v2, got %s", result.Provider())
	}
}

func TestAsLanguageModelV3_V2Model_PreservesModelID(t *testing.T) {
	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-v2",
	})

	result := AsLanguageModelV3(v2Model)

	if result.ModelID() != "test-model-v2" {
		t.Errorf("expected modelId test-model-v2, got %s", result.ModelID())
	}
}

func TestAsLanguageModelV3_V2Model_DoGenerateCallable(t *testing.T) {
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

	result := AsLanguageModelV3(v2Model)

	response, err := result.DoGenerate(languagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(response.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(response.Content))
	}
}

func TestAsLanguageModelV3_V2Model_DoStreamCallable(t *testing.T) {
	ch := make(chan languagemodel.StreamPart, 3)
	ch <- languagemodel.StreamPartTextStart{ID: "1"}
	ch <- languagemodel.StreamPartTextDelta{ID: "1", Delta: "Hello"}
	ch <- languagemodel.StreamPartTextEnd{ID: "1"}
	close(ch)

	v2Model := testutil.NewMockLanguageModelV2(testutil.MockLanguageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoStream: func(_ languagemodel.CallOptions) (languagemodel.StreamResult, error) {
			return languagemodel.StreamResult{
				Stream: ch,
			}, nil
		},
	})

	result := AsLanguageModelV3(v2Model)

	streamResult, err := result.DoStream(languagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parts := drainStream(streamResult.Stream)
	if len(parts) != 3 {
		t.Errorf("expected 3 stream parts, got %d", len(parts))
	}
}

// helper to drain a stream channel
func drainStream(ch <-chan languagemodel.StreamPart) []languagemodel.StreamPart {
	var parts []languagemodel.StreamPart
	for part := range ch {
		parts = append(parts, part)
	}
	return parts
}

// helper to create int pointer
func intPtr(v int) *int {
	return &v
}
