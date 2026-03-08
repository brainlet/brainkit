// Ported from: packages/ai/src/model/as-provider-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// ProviderV4 represents a v4 provider interface.
type ProviderV4 interface {
	SpecificationVersion() string
	LanguageModel(modelID string) (languagemodel.LanguageModel, error)
	EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error)
	ImageModel(modelID string) (imagemodel.ImageModel, error)
	TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error)
	SpeechModel(modelID string) (speechmodel.SpeechModel, error)
	RerankingModel(modelID string) (rerankingmodel.RerankingModel, error)
}

// AsProviderV4 converts a v2, v3, or v4 provider to a v4 provider.
// If the provider is already v4, it is returned unchanged.
// If the provider is v2, it is first converted to v3, then wrapped as v4.
// If the provider is v3, it is wrapped to return v4 models.
func AsProviderV4(p interface{}) ProviderV4 {
	// Check if already v4
	if v4p, ok := p.(ProviderV4); ok {
		if v4p.SpecificationVersion() == "v4" {
			return v4p
		}
	}

	// First ensure we have at least a v3 provider
	var v3Provider ProviderV3
	if v3p, ok := p.(ProviderV3); ok && v3p.SpecificationVersion() == "v3" {
		v3Provider = v3p
	} else {
		v3Provider = AsProviderV3(p)
	}

	if v3Provider == nil {
		return nil
	}

	return &providerV3ToV4Wrapper{inner: v3Provider}
}

// providerV3ToV4Wrapper wraps a v3 provider to provide a v4 interface.
type providerV3ToV4Wrapper struct {
	inner ProviderV3
}

func (w *providerV3ToV4Wrapper) SpecificationVersion() string { return "v4" }

func (w *providerV3ToV4Wrapper) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	m, err := w.inner.LanguageModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsLanguageModelV4(m), nil
}

func (w *providerV3ToV4Wrapper) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	m, err := w.inner.EmbeddingModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsEmbeddingModelV4(m), nil
}

func (w *providerV3ToV4Wrapper) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	m, err := w.inner.ImageModel(modelID)
	if err != nil {
		return nil, err
	}
	return AsImageModelV4(m), nil
}

func (w *providerV3ToV4Wrapper) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	m, err := w.inner.TranscriptionModel(modelID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return AsTranscriptionModelV4(m), nil
}

func (w *providerV3ToV4Wrapper) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	m, err := w.inner.SpeechModel(modelID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return AsSpeechModelV4(m), nil
}

func (w *providerV3ToV4Wrapper) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	m, err := w.inner.RerankingModel(modelID)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return AsRerankingModelV4(m), nil
}
