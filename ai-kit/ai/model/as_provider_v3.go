// Ported from: packages/ai/src/model/as-provider-v3.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// ProviderV2 represents a v2 provider interface.
// In the TS SDK, v2 providers have textEmbeddingModel instead of embeddingModel
// and lack specificationVersion. In Go, we represent this as an interface
// with the v2-specific method names.
type ProviderV2 interface {
	LanguageModel(modelID string) (languagemodel.LanguageModel, error)
	TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error)
	ImageModel(modelID string) (imagemodel.ImageModel, error)
	TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error)
	SpeechModel(modelID string) (speechmodel.SpeechModel, error)
}

// ProviderV3 represents a v3 provider interface.
type ProviderV3 interface {
	SpecificationVersion() string
	LanguageModel(modelID string) (languagemodel.LanguageModel, error)
	EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error)
	ImageModel(modelID string) (imagemodel.ImageModel, error)
	TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error)
	SpeechModel(modelID string) (speechmodel.SpeechModel, error)
	RerankingModel(modelID string) (rerankingmodel.RerankingModel, error)
}

// AsProviderV3 converts a v2 or v3 provider to a v3 provider.
// If the provider is already v3, it is returned unchanged.
// If the provider is v2, it wraps each model factory to convert models to v3.
func AsProviderV3(p interface{}) ProviderV3 {
	// Check if already v3
	if v3p, ok := p.(ProviderV3); ok {
		if v3p.SpecificationVersion() == "v3" {
			return v3p
		}
	}

	// v2 provider
	v2p, ok := p.(ProviderV2)
	if !ok {
		// If we can't identify the provider version, return as-is if it implements v3
		if v3p, ok := p.(ProviderV3); ok {
			return v3p
		}
		return nil
	}

	return &providerV2ToV3Wrapper{inner: v2p}
}

// providerV2ToV3Wrapper wraps a v2 provider to provide a v3 interface.
type providerV2ToV3Wrapper struct {
	inner ProviderV2
}

func (w *providerV2ToV3Wrapper) SpecificationVersion() string { return "v3" }

func (w *providerV2ToV3Wrapper) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	m, err := w.inner.LanguageModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsLanguageModelV3(m), nil
}

func (w *providerV2ToV3Wrapper) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	m, err := w.inner.TextEmbeddingModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsEmbeddingModelV3(m), nil
}

func (w *providerV2ToV3Wrapper) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	m, err := w.inner.ImageModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsImageModelV3(m), nil
}

func (w *providerV2ToV3Wrapper) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	m, err := w.inner.TranscriptionModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsTranscriptionModelV3(m), nil
}

func (w *providerV2ToV3Wrapper) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	m, err := w.inner.SpeechModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsSpeechModelV3(m), nil
}

func (w *providerV2ToV3Wrapper) RerankingModel(_ string) (rerankingmodel.RerankingModel, error) {
	// v2 providers don't have reranking models
	return nil, nil
}
