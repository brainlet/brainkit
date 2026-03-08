// Ported from: packages/ai/src/registry/custom-provider.test.ts
package registry

import (
	"regexp"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	aierrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// --- Mock model types for testing ---

type mockLanguageModelV4 struct {
	provider string
	modelID  string
}

func newMockLanguageModelV4() *mockLanguageModelV4 {
	return &mockLanguageModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func newMockLanguageModelV4WithID(modelID string) *mockLanguageModelV4 {
	return &mockLanguageModelV4{provider: "mock-provider", modelID: modelID}
}

func (m *mockLanguageModelV4) SpecificationVersion() string { return "v4" }
func (m *mockLanguageModelV4) Provider() string             { return m.provider }
func (m *mockLanguageModelV4) ModelID() string              { return m.modelID }
func (m *mockLanguageModelV4) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return map[string][]*regexp.Regexp{}, nil
}
func (m *mockLanguageModelV4) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	return languagemodel.GenerateResult{}, nil
}
func (m *mockLanguageModelV4) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	return languagemodel.StreamResult{}, nil
}

type mockEmbeddingModelV4 struct {
	provider string
	modelID  string
}

func newMockEmbeddingModelV4() *mockEmbeddingModelV4 {
	return &mockEmbeddingModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func (m *mockEmbeddingModelV4) SpecificationVersion() string { return "v4" }
func (m *mockEmbeddingModelV4) Provider() string             { return m.provider }
func (m *mockEmbeddingModelV4) ModelID() string              { return m.modelID }
func (m *mockEmbeddingModelV4) MaxEmbeddingsPerCall() (*int, error) {
	return nil, nil
}
func (m *mockEmbeddingModelV4) SupportsParallelCalls() (bool, error) {
	return false, nil
}
func (m *mockEmbeddingModelV4) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
	return embeddingmodel.Result{}, nil
}

type mockImageModelV4 struct {
	provider string
	modelID  string
}

func newMockImageModelV4() *mockImageModelV4 {
	return &mockImageModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func newMockImageModelV4WithID(modelID string) *mockImageModelV4 {
	return &mockImageModelV4{provider: "mock-provider", modelID: modelID}
}

func (m *mockImageModelV4) SpecificationVersion() string         { return "v4" }
func (m *mockImageModelV4) Provider() string                     { return m.provider }
func (m *mockImageModelV4) ModelID() string                      { return m.modelID }
func (m *mockImageModelV4) MaxImagesPerCall() (*int, error)      { n := 1; return &n, nil }
func (m *mockImageModelV4) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	return imagemodel.GenerateResult{}, nil
}

type mockTranscriptionModelV4 struct {
	provider string
	modelID  string
}

func newMockTranscriptionModelV4() *mockTranscriptionModelV4 {
	return &mockTranscriptionModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func (m *mockTranscriptionModelV4) SpecificationVersion() string { return "v4" }
func (m *mockTranscriptionModelV4) Provider() string             { return m.provider }
func (m *mockTranscriptionModelV4) ModelID() string              { return m.modelID }
func (m *mockTranscriptionModelV4) DoGenerate(options transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
	return transcriptionmodel.GenerateResult{}, nil
}

type mockSpeechModelV4 struct {
	provider string
	modelID  string
}

func newMockSpeechModelV4() *mockSpeechModelV4 {
	return &mockSpeechModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func (m *mockSpeechModelV4) SpecificationVersion() string { return "v4" }
func (m *mockSpeechModelV4) Provider() string             { return m.provider }
func (m *mockSpeechModelV4) ModelID() string              { return m.modelID }
func (m *mockSpeechModelV4) DoGenerate(options speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
	return speechmodel.GenerateResult{}, nil
}

type mockRerankingModelV4 struct {
	provider string
	modelID  string
}

func newMockRerankingModelV4() *mockRerankingModelV4 {
	return &mockRerankingModelV4{provider: "mock-provider", modelID: "mock-model-id"}
}

func (m *mockRerankingModelV4) SpecificationVersion() string { return "v4" }
func (m *mockRerankingModelV4) Provider() string             { return m.provider }
func (m *mockRerankingModelV4) ModelID() string              { return m.modelID }
func (m *mockRerankingModelV4) DoRerank(options rerankingmodel.CallOptions) (rerankingmodel.RerankResult, error) {
	return rerankingmodel.RerankResult{}, nil
}

// --- Mock fallback provider ---

type mockFallbackProvider struct {
	languageModelFn      func(modelID string) (languagemodel.LanguageModel, error)
	embeddingModelFn     func(modelID string) (embeddingmodel.EmbeddingModel, error)
	imageModelFn         func(modelID string) (imagemodel.ImageModel, error)
	transcriptionModelFn func(modelID string) (transcriptionmodel.TranscriptionModel, error)
	speechModelFn        func(modelID string) (speechmodel.SpeechModel, error)
	rerankingModelFn     func(modelID string) (rerankingmodel.RerankingModel, error)
	videoModelFn         func(modelID string) (videomodel.VideoModel, error)
}

func (p *mockFallbackProvider) SpecificationVersion() string { return "v4" }

func (p *mockFallbackProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	if p.languageModelFn != nil {
		return p.languageModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeLanguage,
	})
}

func (p *mockFallbackProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	if p.embeddingModelFn != nil {
		return p.embeddingModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeEmbedding,
	})
}

func (p *mockFallbackProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	if p.imageModelFn != nil {
		return p.imageModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeImage,
	})
}

func (p *mockFallbackProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	if p.transcriptionModelFn != nil {
		return p.transcriptionModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeTranscription,
	})
}

func (p *mockFallbackProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	if p.speechModelFn != nil {
		return p.speechModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeSpeech,
	})
}

func (p *mockFallbackProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	if p.rerankingModelFn != nil {
		return p.rerankingModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeReranking,
	})
}

func (p *mockFallbackProvider) VideoModel(modelID string) (videomodel.VideoModel, error) {
	if p.videoModelFn != nil {
		return p.videoModelFn(modelID)
	}
	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeVideo,
	})
}

// --- Tests ---

func TestCustomProvider_LanguageModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockLanguageModelV4()

	provider := CustomProvider(CustomProviderOptions{
		LanguageModels: map[string]languagemodel.LanguageModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.LanguageModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_LanguageModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockLanguageModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		languageModelFn: func(modelID string) (languagemodel.LanguageModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.LanguageModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_LanguageModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.LanguageModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestCustomProvider_EmbeddingModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockEmbeddingModelV4()

	provider := CustomProvider(CustomProviderOptions{
		EmbeddingModels: map[string]embeddingmodel.EmbeddingModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.EmbeddingModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_EmbeddingModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockEmbeddingModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		embeddingModelFn: func(modelID string) (embeddingmodel.EmbeddingModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.EmbeddingModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_EmbeddingModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.EmbeddingModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestCustomProvider_ImageModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockImageModelV4()

	provider := CustomProvider(CustomProviderOptions{
		ImageModels: map[string]imagemodel.ImageModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.ImageModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_ImageModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockImageModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		imageModelFn: func(modelID string) (imagemodel.ImageModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.ImageModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_ImageModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.ImageModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestCustomProvider_TranscriptionModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockTranscriptionModelV4()

	provider := CustomProvider(CustomProviderOptions{
		TranscriptionModels: map[string]transcriptionmodel.TranscriptionModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.TranscriptionModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_TranscriptionModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockTranscriptionModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		transcriptionModelFn: func(modelID string) (transcriptionmodel.TranscriptionModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.TranscriptionModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_TranscriptionModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.TranscriptionModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestCustomProvider_SpeechModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockSpeechModelV4()

	provider := CustomProvider(CustomProviderOptions{
		SpeechModels: map[string]speechmodel.SpeechModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.SpeechModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_SpeechModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockSpeechModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		speechModelFn: func(modelID string) (speechmodel.SpeechModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.SpeechModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_SpeechModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.SpeechModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}

func TestCustomProvider_RerankingModel_ReturnsModelIfExists(t *testing.T) {
	mockModel := newMockRerankingModelV4()

	provider := CustomProvider(CustomProviderOptions{
		RerankingModels: map[string]rerankingmodel.RerankingModel{
			"test-model": mockModel,
		},
	})

	result, err := provider.RerankingModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
}

func TestCustomProvider_RerankingModel_UsesFallbackIfModelNotFound(t *testing.T) {
	mockModel := newMockRerankingModelV4()

	var calledWith string
	fallback := &mockFallbackProvider{
		rerankingModelFn: func(modelID string) (rerankingmodel.RerankingModel, error) {
			calledWith = modelID
			return mockModel, nil
		},
	}

	provider := CustomProvider(CustomProviderOptions{
		FallbackProvider: fallback,
	})

	result, err := provider.RerankingModel("test-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != mockModel {
		t.Fatalf("expected model to be the mock model")
	}
	if calledWith != "test-model" {
		t.Fatalf("expected fallback to be called with 'test-model', got %q", calledWith)
	}
}

func TestCustomProvider_RerankingModel_ThrowsNoSuchModelErrorIfNotFoundAndNoFallback(t *testing.T) {
	provider := CustomProvider(CustomProviderOptions{})

	_, err := provider.RerankingModel("test-model")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !aierrors.IsNoSuchModelError(err) {
		t.Fatalf("expected NoSuchModelError, got %T: %v", err, err)
	}
}
