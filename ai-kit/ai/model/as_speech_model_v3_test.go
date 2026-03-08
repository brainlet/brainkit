// Ported from: packages/ai/src/model/as-speech-model-v3.test.ts
package model

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

func TestAsSpeechModelV3_V3Model_ReturnsSameModel(t *testing.T) {
	original := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV3(original)

	if result != original {
		t.Error("expected same v3 model to be returned unchanged")
	}
	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
}

func TestAsSpeechModelV3_V3Model_PreservesProperties(t *testing.T) {
	original := testutil.NewMockSpeechModelV3(testutil.MockSpeechModelV3Options{
		Provider: "test-provider-v3",
		ModelID:  "test-model-v3",
	})

	result := AsSpeechModelV3(original)

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

func TestAsSpeechModelV3_V2Model_ConvertsToV3(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV3(v2Model)

	if result.SpecificationVersion() != "v3" {
		t.Errorf("expected specificationVersion v3, got %s", result.SpecificationVersion())
	}
	if result == v2Model {
		t.Error("expected a new wrapper, not the original v2 model")
	}
}

func TestAsSpeechModelV3_V2Model_PreservesProvider(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider-v2",
		ModelID:  "test-model-id",
	})

	result := AsSpeechModelV3(v2Model)

	if result.Provider() != "test-provider-v2" {
		t.Errorf("expected provider test-provider-v2, got %s", result.Provider())
	}
}

func TestAsSpeechModelV3_V2Model_PreservesModelID(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-v2",
	})

	result := AsSpeechModelV3(v2Model)

	if result.ModelID() != "test-model-v2" {
		t.Errorf("expected modelId test-model-v2, got %s", result.ModelID())
	}
}

func TestAsSpeechModelV3_V2Model_DoGenerateCallable(t *testing.T) {
	v2Model := testutil.NewMockSpeechModelV2(testutil.MockSpeechModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoGenerate: func(_ speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
			return speechmodel.GenerateResult{
				Audio: speechmodel.AudioDataString{Value: "base64audio"},
			}, nil
		},
	})

	result := AsSpeechModelV3(v2Model)

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
