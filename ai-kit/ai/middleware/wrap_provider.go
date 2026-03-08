// Ported from: packages/ai/src/middleware/wrap-provider.ts
package middleware

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"

	provider "github.com/brainlet/brainkit/ai-kit/provider"
)

// WrapProviderOptions holds the options for WrapProvider.
type WrapProviderOptions struct {
	Provider                provider.Provider
	LanguageModelMiddleware []mw.LanguageModelMiddleware
	ImageModelMiddleware    []mw.ImageModelMiddleware // optional
}

// WrapProvider wraps a Provider instance with middleware functionality.
// This applies the language model middleware to all language models from the
// provider, and optionally applies image model middleware to all image models.
func WrapProvider(opts WrapProviderOptions) provider.Provider {
	return &wrappedProvider{
		underlying:             opts.Provider,
		languageModelMW:        opts.LanguageModelMiddleware,
		imageModelMW:           opts.ImageModelMiddleware,
	}
}

type wrappedProvider struct {
	underlying      provider.Provider
	languageModelMW []mw.LanguageModelMiddleware
	imageModelMW    []mw.ImageModelMiddleware
}

func (w *wrappedProvider) SpecificationVersion() string { return "v3" }

func (w *wrappedProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	model, err := w.underlying.LanguageModel(modelID)
	if err != nil {
		return nil, err
	}
	return WrapLanguageModel(WrapLanguageModelOptions{
		Model:      model,
		Middleware: w.languageModelMW,
	}), nil
}

func (w *wrappedProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return w.underlying.EmbeddingModel(modelID)
}

func (w *wrappedProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	model, err := w.underlying.ImageModel(modelID)
	if err != nil {
		return nil, err
	}
	if len(w.imageModelMW) > 0 {
		model = WrapImageModel(WrapImageModelOptions{
			Model:      model,
			Middleware: w.imageModelMW,
		})
	}
	return model, nil
}

func (w *wrappedProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return w.underlying.TranscriptionModel(modelID)
}

func (w *wrappedProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return w.underlying.SpeechModel(modelID)
}

func (w *wrappedProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return w.underlying.RerankingModel(modelID)
}
