// Ported from: packages/ai/src/model/as-speech-model-v4.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsSpeechModelV4_V4Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockSpeechModelV4(testutil.MockSpeechModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV4(original)

	if result != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
}

func TestAsSpeechModelV4_V4Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockSpeechModelV4(testutil.MockSpeechModelV4Options{
		Provider: "test-provider-v4",
		ModelID:  "test-model-v4",
	})

	result := AsSpeechModelV4(original)

	if result.Provider() != "test-provider-v4" {
		t.Errorf("expected provider test-provider-v4, got %s", result.Provider())
	}
	if result.ModelID() != "test-model-v4" {
		t.Errorf("expected modelId test-model-v4, got %s", result.ModelID())
	}
}

func TestAsSpeechModelV4_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV4(v3Model)

	if result.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", result.SpecificationVersion())
	}
	if result == v3Model {
		t.Error("expected a new wrapper, not the original v3 model")
	}
}

func TestAsSpeechModelV4_V3Model_PreservesProvider(t *testing.T) {
	v3Model := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV4(v3Model)

	if result.Provider() != "test-provider-v3" {
		t.Errorf("expected provider test-provider-v3, got %s", result.Provider())
	}
}

func TestAsSpeechModelV4_V3Model_PreservesModelID(t *testing.T) {
	v3Model := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-v3",
	})

	result := AsSpeechModelV4(v3Model)

	if result.ModelID() != "test-model-v3" {
		t.Errorf("expected modelId test-model-v3, got %s", result.ModelID())
	}
}

func TestAsSpeechModelV4_V3Model_DoGenerateCallable(t *testing.T) {
	v3Model := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
			return speechmodel.GenerateResult{
				Audio: speechmodel.AudioDataString{Value: "base64audio"},
			}, nil
		},
	})

	result := AsSpeechModelV4(v3Model)

	response, err := result.DoGenerate(speechmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audio, ok := response.Audio.(speechmodel.AudioDataString)
	if !ok {
		t.Fatal("expected AudioDataString")
	}
	if audio.Value != "base64audio" {
		t.Errorf("expected audio base64audio, got %s", audio.Value)
	}
}

func TestAsSpeechModelV4_V2Model_ConvertsThroughV3ToV4(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV4(v2Model)

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

func TestAsSpeechModelV4_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
			return speechmodel.GenerateResult{
				Audio: speechmodel.AudioDataString{Value: "base64audio"},
			}, nil
		},
	})

	result := AsSpeechModelV4(v2Model)

	response, err := result.DoGenerate(speechmodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	audio, ok := response.Audio.(speechmodel.AudioDataString)
	if !ok {
		t.Fatal("expected AudioDataString")
	}
	if audio.Value != "base64audio" {
		t.Errorf("expected audio base64audio, got %s", audio.Value)
	}
}
