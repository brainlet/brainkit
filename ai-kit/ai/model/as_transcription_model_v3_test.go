// Ported from: packages/ai/src/model/as-transcription-model-v3.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsTranscriptionModelV3_V3Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV3(original)

	if result != original {
		t.Error("expected same v3 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsTranscriptionModelV3_V3Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-v3",
	})

	result := AsTranscriptionModelV3(original)

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

func TestAsTranscriptionModelV3_V2Model_ConvertsToV3(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV3(v2Model)

	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
	if result == v2Model {
		t.Error("expected a new wrapper, not the original v2 model")
	}
}

func TestAsTranscriptionModelV3_V2Model_PreservesProvider(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider-v2",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV3(v2Model)

	if result.Provider() != "test-provider-v2" {
		t.Errorf("expected provider test-provider-v2, got %s", result.Provider())
	}
}

func TestAsTranscriptionModelV3_V2Model_PreservesModelID(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-v2",
	})

	result := AsTranscriptionModelV3(v2Model)

	if result.ModelID() != "test-model-v2" {
		t.Errorf("expected modelId test-model-v2, got %s", result.ModelID())
	}
}

func TestAsTranscriptionModelV3_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
			return transcriptionmodel.GenerateResult{
				Text: "Hello, world!",
				Segments: []transcriptionmodel.Segment{
					{Text: "Hello, world!", StartSecond: 0.0, EndSecond: 1.5},
				},
			}, nil
		},
	})

	result := AsTranscriptionModelV3(v2Model)

	response, err := result.DoGenerate(transcriptionmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", response.Text)
	}
	if len(response.Segments) != 1 {
		t.Errorf("expected 1 segment, got %d", len(response.Segments))
	}
}
