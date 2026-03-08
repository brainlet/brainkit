// Ported from: packages/ai/src/test/mock-provider-v4.ts
package testutil

import (
	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	providerErrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	rm "github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	sm "github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	tm "github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// MockProviderV4 is a test double for a Provider with specificationVersion "v4".
type MockProviderV4 struct {
	languageModels      map[string]lm.LanguageModel
	embeddingModels     map[string]em.EmbeddingModel
	imageModels         map[string]im.ImageModel
	transcriptionModels map[string]tm.TranscriptionModel
	speechModels        map[string]sm.SpeechModel
	rerankingModels     map[string]rm.RerankingModel
}

// MockProviderV4Options configures a MockProviderV4.
type MockProviderV4Options struct {
	LanguageModels      map[string]lm.LanguageModel
	EmbeddingModels     map[string]em.EmbeddingModel
	ImageModels         map[string]im.ImageModel
	TranscriptionModels map[string]tm.TranscriptionModel
	SpeechModels        map[string]sm.SpeechModel
	RerankingModels     map[string]rm.RerankingModel
}

// NewMockProviderV4 creates a new MockProviderV4 with the given options.
func NewMockProviderV4(opts ...MockProviderV4Options) *MockProviderV4 {
	var o MockProviderV4Options
	if len(opts) > 0 {
		o = opts[0]
	}

	return &MockProviderV4{
		languageModels:      o.LanguageModels,
		embeddingModels:     o.EmbeddingModels,
		imageModels:         o.ImageModels,
		transcriptionModels: o.TranscriptionModels,
		speechModels:        o.SpeechModels,
		rerankingModels:     o.RerankingModels,
	}
}

func (p *MockProviderV4) SpecificationVersion() string { return "v4" }

func (p *MockProviderV4) LanguageModel(modelID string) (lm.LanguageModel, error) {
	if m, ok := p.languageModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeLanguage,
	})
}

func (p *MockProviderV4) EmbeddingModel(modelID string) (em.EmbeddingModel, error) {
	if m, ok := p.embeddingModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeEmbedding,
	})
}

func (p *MockProviderV4) ImageModel(modelID string) (im.ImageModel, error) {
	if m, ok := p.imageModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeImage,
	})
}

func (p *MockProviderV4) TranscriptionModel(modelID string) (tm.TranscriptionModel, error) {
	if m, ok := p.transcriptionModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeTranscription,
	})
}

func (p *MockProviderV4) SpeechModel(modelID string) (sm.SpeechModel, error) {
	if m, ok := p.speechModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeSpeech,
	})
}

func (p *MockProviderV4) RerankingModel(modelID string) (rm.RerankingModel, error) {
	if m, ok := p.rerankingModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeReranking,
	})
}
