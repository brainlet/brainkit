// Ported from: packages/ai/src/model/as-transcription-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsTranscriptionModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockTranscriptionModelV4(testutil.MockTranscriptionModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsTranscriptionModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockTranscriptionModelV4(testutil.MockTranscriptionModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsTranscriptionModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsTranscriptionModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsTranscriptionModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsTranscriptionModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsTranscriptionModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsTranscriptionModelV4_V3Model_DoGenerateCallable(t *testing.T) {
	v3Model := testutil.NewMockTranscriptionModelV3(testutil.MockTranscriptionModelV3Options{
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

	result := AsTranscriptionModelV4(v3Model)

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

func TestAsTranscriptionModelV4_V2Model_ConvertsThroughV3ToV4(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsTranscriptionModelV4(v2Model)

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

func TestAsTranscriptionModelV4_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockTranscriptionModelV2(testutil.MockTranscriptionModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
			return transcriptionmodel.GenerateResult{
				Text: "Hello, world!",
			}, nil
		},
	})

	result := AsTranscriptionModelV4(v2Model)

	response, err := result.DoGenerate(transcriptionmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if response.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %s", response.Text)
	}
}
