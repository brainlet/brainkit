// Ported from: packages/ai/src/registry/custom-provider.ts
package registry

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	aierrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// CustomProviderOptions are the options for creating a custom provider.
type CustomProviderOptions struct {
	// LanguageModels is a map of model IDs to language model instances.
	LanguageModels map[string]languagemodel.LanguageModel

	// EmbeddingModels is a map of model IDs to embedding model instances.
	EmbeddingModels map[string]embeddingmodel.EmbeddingModel

	// ImageModels is a map of model IDs to image model instances.
	ImageModels map[string]imagemodel.ImageModel

	// TranscriptionModels is a map of model IDs to transcription model instances.
	TranscriptionModels map[string]transcriptionmodel.TranscriptionModel

	// SpeechModels is a map of model IDs to speech model instances.
	SpeechModels map[string]speechmodel.SpeechModel

	// RerankingModels is a map of model IDs to reranking model instances.
	RerankingModels map[string]rerankingmodel.RerankingModel

	// VideoModels is a map of model IDs to video model instances.
	VideoModels map[string]videomodel.VideoModel

	// FallbackProvider is an optional fallback provider to use when a
	// requested model is not found in the custom provider.
	FallbackProvider Provider
}

// CustomProvider creates a custom provider with specified model maps and an
// optional fallback provider.
//
// It returns a Provider that looks up models in the provided maps first, then
// falls back to the fallback provider if one is set, and finally returns a
// NoSuchModelError if the model is not found anywhere.
func CustomProvider(opts CustomProviderOptions) Provider {
	return &customProviderImpl{
		languageModels:      opts.LanguageModels,
		embeddingModels:     opts.EmbeddingModels,
		imageModels:         opts.ImageModels,
		transcriptionModels: opts.TranscriptionModels,
		speechModels:        opts.SpeechModels,
		rerankingModels:     opts.RerankingModels,
		videoModels:         opts.VideoModels,
		fallbackProvider:    opts.FallbackProvider,
	}
}

type customProviderImpl struct {
	languageModels      map[string]languagemodel.LanguageModel
	embeddingModels     map[string]embeddingmodel.EmbeddingModel
	imageModels         map[string]imagemodel.ImageModel
	transcriptionModels map[string]transcriptionmodel.TranscriptionModel
	speechModels        map[string]speechmodel.SpeechModel
	rerankingModels     map[string]rerankingmodel.RerankingModel
	videoModels         map[string]videomodel.VideoModel
	fallbackProvider    Provider
}

func (p *customProviderImpl) SpecificationVersion() string {
	return "v4"
}

func (p *customProviderImpl) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	if p.languageModels != nil {
		if model, ok := p.languageModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.LanguageModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeLanguage,
	})
}

func (p *customProviderImpl) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	if p.embeddingModels != nil {
		if model, ok := p.embeddingModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.EmbeddingModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeEmbedding,
	})
}

func (p *customProviderImpl) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	if p.imageModels != nil {
		if model, ok := p.imageModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.ImageModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeImage,
	})
}

func (p *customProviderImpl) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	if p.transcriptionModels != nil {
		if model, ok := p.transcriptionModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.TranscriptionModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeTranscription,
	})
}

func (p *customProviderImpl) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	if p.speechModels != nil {
		if model, ok := p.speechModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.SpeechModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeSpeech,
	})
}

func (p *customProviderImpl) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	if p.rerankingModels != nil {
		if model, ok := p.rerankingModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.RerankingModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeReranking,
	})
}

func (p *customProviderImpl) VideoModel(modelID string) (videomodel.VideoModel, error) {
	if p.videoModels != nil {
		if model, ok := p.videoModels[modelID]; ok {
			return model, nil
		}
	}

	if p.fallbackProvider != nil {
		return p.fallbackProvider.VideoModel(modelID)
	}

	return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: aierrors.ModelTypeVideo,
	})
}
