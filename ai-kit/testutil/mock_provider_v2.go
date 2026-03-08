// Ported from: packages/ai/src/test/mock-provider-v2.ts
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

// MockProviderV2 is a test double for a Provider with no explicit specificationVersion.
// Corresponds to ProviderV2 in TS which has textEmbeddingModel instead of embeddingModel.
type MockProviderV2 struct {
	languageModels      map[string]lm.LanguageModel
	embeddingModels     map[string]em.EmbeddingModel
	imageModels         map[string]im.ImageModel
	transcriptionModels map[string]tm.TranscriptionModel
	speechModels        map[string]sm.SpeechModel
}

// MockProviderV2Options configures a MockProviderV2.
type MockProviderV2Options struct {
	LanguageModels      map[string]lm.LanguageModel
	EmbeddingModels     map[string]em.EmbeddingModel
	ImageModels         map[string]im.ImageModel
	TranscriptionModels map[string]tm.TranscriptionModel
	SpeechModels        map[string]sm.SpeechModel
}

// NewMockProviderV2 creates a new MockProviderV2 with the given options.
func NewMockProviderV2(opts ...MockProviderV2Options) *MockProviderV2 {
	var o MockProviderV2Options
	if len(opts) > 0 {
		o = opts[0]
	}

	return &MockProviderV2{
		languageModels:      o.LanguageModels,
		embeddingModels:     o.EmbeddingModels,
		imageModels:         o.ImageModels,
		transcriptionModels: o.TranscriptionModels,
		speechModels:        o.SpeechModels,
	}
}

func (p *MockProviderV2) SpecificationVersion() string { return "v2" }

func (p *MockProviderV2) LanguageModel(modelID string) (lm.LanguageModel, error) {
	if m, ok := p.languageModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeLanguage,
	})
}

func (p *MockProviderV2) EmbeddingModel(modelID string) (em.EmbeddingModel, error) {
	if m, ok := p.embeddingModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeEmbedding,
	})
}

func (p *MockProviderV2) ImageModel(modelID string) (im.ImageModel, error) {
	if m, ok := p.imageModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeImage,
	})
}

func (p *MockProviderV2) TranscriptionModel(modelID string) (tm.TranscriptionModel, error) {
	if m, ok := p.transcriptionModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeTranscription,
	})
}

func (p *MockProviderV2) SpeechModel(modelID string) (sm.SpeechModel, error) {
	if m, ok := p.speechModels[modelID]; ok {
		return m, nil
	}
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeSpeech,
	})
}

func (p *MockProviderV2) RerankingModel(modelID string) (rm.RerankingModel, error) {
	return nil, providerErrors.NewNoSuchModelError(providerErrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: providerErrors.ModelTypeReranking,
	})
}
