// Ported from: packages/ai/src/model/resolve-model.test.ts
package model

import (
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/testutil"
)

// --- resolveLanguageModel ---

func TestResolveLanguageModel_V4Model_ReturnsSame(t *testing.T) {
	original := testutil.NewMockLanguageModelV4(testutil.MockLanguageModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveLanguageModel(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveLanguageModel_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockLanguageModelV3(testutil.MockLanguageModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveLanguageModel(v3Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveLanguageModel_V2Model_ConvertsToV4(t *testing.T) {
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
					InputTokens:  languagemodel.InputTokenUsage{Total: intPtr(0)},
					OutputTokens: languagemodel.OutputTokenUsage{Total: intPtr(0)},
				},
			}, nil
		},
	})

	resolved, err := ResolveLanguageModel(v2Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}

	// Verify DoGenerate works through the conversion chain
	response, err := resolved.DoGenerate(languagemodel.CallOptions{})
	if err != nil {
		t.Fatalf("unexpected error calling DoGenerate: %v", err)
	}
	if len(response.Content) != 1 {
		t.Errorf("expected 1 content item, got %d", len(response.Content))
	}
}

func TestResolveLanguageModel_String_NoDefaultProvider_ReturnsError(t *testing.T) {
	// Ensure no default provider is set
	oldProvider := DefaultProvider
	DefaultProvider = nil
	defer func() { DefaultProvider = oldProvider }()

	_, err := ResolveLanguageModel("test-model-id")
	if err == nil {
		t.Fatal("expected error when no default provider is set")
	}
}

func TestResolveLanguageModel_String_WithDefaultProvider(t *testing.T) {
	lm := testutil.NewMockLanguageModelV4(testutil.MockLanguageModelV4Options{
		Provider: "global-test-provider",
		ModelID:  "actual-test-model-id",
	})

	oldProvider := DefaultProvider
	DefaultProvider = testutil.NewMockProviderV4(testutil.MockProviderV4Options{
		LanguageModels: map[string]languagemodel.LanguageModel{
			"test-model-id": lm,
		},
	})
	defer func() { DefaultProvider = oldProvider }()

	resolved, err := ResolveLanguageModel("test-model-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "global-test-provider" {
		t.Errorf("expected provider global-test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "actual-test-model-id" {
		t.Errorf("expected modelId actual-test-model-id, got %s", resolved.ModelID())
	}
}

// --- resolveEmbeddingModel ---

func TestResolveEmbeddingModel_V4Model_ReturnsSame(t *testing.T) {
	original := testutil.NewMockEmbeddingModelV4(testutil.MockEmbeddingModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveEmbeddingModel(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveEmbeddingModel_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockEmbeddingModelV3(testutil.MockEmbeddingModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveEmbeddingModel(v3Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveEmbeddingModel_V2Model_ConvertsToV4(t *testing.T) {
	v2Model := testutil.NewMockEmbeddingModelV2(testutil.MockEmbeddingModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
		DoEmbed: func(_ embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
			return embeddingmodel.Result{
				Embeddings: []embeddingmodel.Embedding{{0.1, 0.2, 0.3}},
			}, nil
		},
	})

	resolved, err := ResolveEmbeddingModel(v2Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}

	result, err := resolved.DoEmbed(embeddingmodel.CallOptions{Values: []string{"hello"}})
	if err != nil {
		t.Fatalf("unexpected error calling DoEmbed: %v", err)
	}
	if len(result.Embeddings) != 1 {
		t.Errorf("expected 1 embedding, got %d", len(result.Embeddings))
	}
}

func TestResolveEmbeddingModel_String_NoDefaultProvider_ReturnsError(t *testing.T) {
	oldProvider := DefaultProvider
	DefaultProvider = nil
	defer func() { DefaultProvider = oldProvider }()

	_, err := ResolveEmbeddingModel("test-model-id")
	if err == nil {
		t.Fatal("expected error when no default provider is set")
	}
}

func TestResolveEmbeddingModel_String_WithDefaultProvider(t *testing.T) {
	em := testutil.NewMockEmbeddingModelV4(testutil.MockEmbeddingModelV4Options{
		Provider: "global-test-provider",
		ModelID:  "actual-test-model-id",
	})

	oldProvider := DefaultProvider
	DefaultProvider = testutil.NewMockProviderV4(testutil.MockProviderV4Options{
		EmbeddingModels: map[string]embeddingmodel.EmbeddingModel{
			"test-model-id": em,
		},
	})
	defer func() { DefaultProvider = oldProvider }()

	resolved, err := ResolveEmbeddingModel("test-model-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "global-test-provider" {
		t.Errorf("expected provider global-test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "actual-test-model-id" {
		t.Errorf("expected modelId actual-test-model-id, got %s", resolved.ModelID())
	}
}

// --- resolveImageModel ---

func TestResolveImageModel_V2Model_ConvertsToV4(t *testing.T) {
	v2Model := testutil.NewMockImageModelV2(testutil.MockImageModelV2Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveImageModel(v2Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
}

func TestResolveImageModel_String_WithDefaultProvider(t *testing.T) {
	im := testutil.NewMockImageModelV4(testutil.MockImageModelV4Options{
		Provider: "global-test-provider",
		ModelID:  "actual-test-model-id",
	})

	oldProvider := DefaultProvider
	DefaultProvider = testutil.NewMockProviderV4(testutil.MockProviderV4Options{
		ImageModels: map[string]imagemodel.ImageModel{
			"test-model-id": im,
		},
	})
	defer func() { DefaultProvider = oldProvider }()

	resolved, err := ResolveImageModel("test-model-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "global-test-provider" {
		t.Errorf("expected provider global-test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "actual-test-model-id" {
		t.Errorf("expected modelId actual-test-model-id, got %s", resolved.ModelID())
	}
}

// --- resolveVideoModel ---

func TestResolveVideoModel_V4Model_ReturnsSame(t *testing.T) {
	original := testutil.NewMockVideoModelV4(testutil.MockVideoModelV4Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveVideoModel(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved != original {
		t.Error("expected same v4 model to be returned unchanged")
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveVideoModel_V3Model_ConvertsToV4(t *testing.T) {
	v3Model := testutil.NewMockVideoModelV3(testutil.MockVideoModelV3Options{
		Provider: "test-provider",
		ModelID:  "test-model-id",
	})

	resolved, err := ResolveVideoModel(v3Model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "test-provider" {
		t.Errorf("expected provider test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "test-model-id" {
		t.Errorf("expected modelId test-model-id, got %s", resolved.ModelID())
	}
	if resolved.SpecificationVersion() != "v4" {
		t.Errorf("expected specificationVersion v4, got %s", resolved.SpecificationVersion())
	}
}

func TestResolveVideoModel_String_NoDefaultProvider_ReturnsError(t *testing.T) {
	oldProvider := DefaultProvider
	DefaultProvider = nil
	defer func() { DefaultProvider = oldProvider }()

	_, err := ResolveVideoModel("test-model-id")
	if err == nil {
		t.Fatal("expected error when no default provider is set")
	}
}

func TestResolveVideoModel_String_ProviderDoesNotSupportVideo_ReturnsError(t *testing.T) {
	// Use a provider that doesn't implement VideoModelProvider
	oldProvider := DefaultProvider
	DefaultProvider = testutil.NewMockProviderV4(testutil.MockProviderV4Options{})
	defer func() { DefaultProvider = oldProvider }()

	_, err := ResolveVideoModel("test-model-id")
	if err == nil {
		t.Fatal("expected error when provider does not support video models")
	}
	if !strings.Contains(err.Error(), "does not support video models") {
		t.Errorf("expected error about video models not supported, got: %s", err.Error())
	}
}

func TestResolveVideoModel_String_WithVideoModelProvider(t *testing.T) {
	vm := testutil.NewMockVideoModelV4(testutil.MockVideoModelV4Options{
		Provider: "global-test-provider",
		ModelID:  "actual-test-model-id",
	})

	oldProvider := DefaultProvider
	DefaultProvider = &mockVideoModelProvider{
		MockProviderV4: testutil.NewMockProviderV4(testutil.MockProviderV4Options{}),
		videoModels: map[string]videomodel.VideoModel{
			"test-model-id": vm,
		},
	}
	defer func() { DefaultProvider = oldProvider }()

	resolved, err := ResolveVideoModel("test-model-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.Provider() != "global-test-provider" {
		t.Errorf("expected provider global-test-provider, got %s", resolved.Provider())
	}
	if resolved.ModelID() != "actual-test-model-id" {
		t.Errorf("expected modelId actual-test-model-id, got %s", resolved.ModelID())
	}
}

// mockVideoModelProvider wraps a MockProviderV4 and adds VideoModel support.
type mockVideoModelProvider struct {
	*testutil.MockProviderV4
	videoModels map[string]videomodel.VideoModel
}

func (p *mockVideoModelProvider) VideoModel(modelID string) (videomodel.VideoModel, error) {
	if m, ok := p.videoModels[modelID]; ok {
		return m, nil
	}
	return nil, nil
}
