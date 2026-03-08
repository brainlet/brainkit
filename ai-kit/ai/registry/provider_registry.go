// Ported from: packages/ai/src/registry/provider-registry.ts
package registry

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	aierrors "github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// LanguageModelMiddleware defines middleware that can modify the behavior
// of language model operations within the registry.
//
// TODO: Import from brainlink/experiments/ai-kit/middleware once wrap-language-model.go is ported.
// Corresponds to LanguageModelMiddleware from packages/ai/src/types/language-model-middleware.ts
type LanguageModelMiddleware struct {
	// OverrideModelID overrides the model ID if desired.
	// Called with the model and returns the new model ID.
	OverrideModelID func(opts OverrideModelIDOptions) string
}

// OverrideModelIDOptions are the options passed to OverrideModelID.
type OverrideModelIDOptions struct {
	Model languagemodel.LanguageModel
}

// ImageModelMiddleware defines middleware that can modify the behavior
// of image model operations within the registry.
//
// TODO: Import from brainlink/experiments/ai-kit/middleware once wrap-image-model.go is ported.
// Corresponds to ImageModelMiddleware from packages/ai/src/types/image-model-middleware.ts
type ImageModelMiddleware struct {
	// OverrideModelID overrides the model ID if desired.
	// Called with the model and returns the new model ID.
	OverrideModelID func(opts ImageOverrideModelIDOptions) string
}

// ImageOverrideModelIDOptions are the options passed to ImageModelMiddleware.OverrideModelID.
type ImageOverrideModelIDOptions struct {
	Model imagemodel.ImageModel
}

// ProviderRegistryOptions are the options for creating a provider registry.
type ProviderRegistryOptions struct {
	// Separator is the separator used between provider ID and model ID
	// in the combined identifier. Defaults to ":".
	Separator string

	// LanguageModelMiddleware is optional middleware to be applied to all
	// language models from the registry. Can be a single middleware or a slice.
	// When multiple middlewares are provided, the first middleware will transform
	// the input first, and the last middleware will be wrapped directly around the model.
	LanguageModelMiddleware []LanguageModelMiddleware

	// ImageModelMiddleware is optional middleware to be applied to all
	// image models from the registry. Can be a single middleware or a slice.
	// When multiple middlewares are provided, the first middleware will transform
	// the input first, and the last middleware will be wrapped directly around the model.
	ImageModelMiddleware []ImageModelMiddleware
}

// CreateProviderRegistry creates a registry for the given providers with
// optional middleware functionality. This function allows you to register
// multiple providers and optionally apply middleware to all language models
// and image models from the registry.
//
// The providers map keys are provider IDs, and values are Provider instances.
// Model IDs passed to the registry methods must be in the format
// "providerID<separator>modelID" (default separator is ":").
func CreateProviderRegistry(providers map[string]Provider, opts ...ProviderRegistryOptions) *DefaultProviderRegistry {
	var options ProviderRegistryOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	separator := options.Separator
	if separator == "" {
		separator = ":"
	}

	registry := &DefaultProviderRegistry{
		providers:               make(map[string]Provider),
		separator:               separator,
		languageModelMiddleware: options.LanguageModelMiddleware,
		imageModelMiddleware:    options.ImageModelMiddleware,
	}

	for id, p := range providers {
		registry.RegisterProvider(id, p)
	}

	return registry
}

// DefaultProviderRegistry is the default implementation of a provider registry.
type DefaultProviderRegistry struct {
	providers               map[string]Provider
	separator               string
	languageModelMiddleware []LanguageModelMiddleware
	imageModelMiddleware    []ImageModelMiddleware
}

// RegisterProvider registers a provider with the given id.
func (r *DefaultProviderRegistry) RegisterProvider(id string, p Provider) {
	r.providers[id] = p
}

// getProvider returns the provider with the given id, or returns an error if
// no such provider exists.
func (r *DefaultProviderRegistry) getProvider(id string, modelType aierrors.ModelType) (Provider, error) {
	p, ok := r.providers[id]
	if !ok {
		availableProviders := make([]string, 0, len(r.providers))
		for k := range r.providers {
			availableProviders = append(availableProviders, k)
		}
		return nil, NewNoSuchProviderError(NoSuchProviderErrorOptions{
			ModelID:            id,
			ModelType:          modelType,
			ProviderID:         id,
			AvailableProviders: availableProviders,
		})
	}
	return p, nil
}

// splitID splits a combined "providerID<separator>modelID" string into its
// two components. Returns an error if the separator is not found.
func (r *DefaultProviderRegistry) splitID(id string, modelType aierrors.ModelType) (string, string, error) {
	index := strings.Index(id, r.separator)
	if index == -1 {
		return "", "", aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: modelType,
			Message: fmt.Sprintf(
				"Invalid %s id for registry: %s (must be in the format \"providerId%smodelId\")",
				modelType, id, r.separator,
			),
		})
	}

	providerID := id[:index]
	modelID := id[index+len(r.separator):]
	return providerID, modelID, nil
}

// LanguageModel returns the language model for the given combined id
// (format: "providerID:modelID"). If language model middleware is configured,
// it will be applied to the model.
func (r *DefaultProviderRegistry) LanguageModel(id string) (languagemodel.LanguageModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeLanguage)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeLanguage)
	if err != nil {
		return nil, err
	}

	model, err := p.LanguageModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeLanguage,
		})
	}

	if len(r.languageModelMiddleware) > 0 {
		model = wrapLanguageModel(model, r.languageModelMiddleware)
	}

	return model, nil
}

// EmbeddingModel returns the embedding model for the given combined id
// (format: "providerID:modelID").
func (r *DefaultProviderRegistry) EmbeddingModel(id string) (embeddingmodel.EmbeddingModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeEmbedding)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeEmbedding)
	if err != nil {
		return nil, err
	}

	model, err := p.EmbeddingModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeEmbedding,
		})
	}

	return model, nil
}

// ImageModel returns the image model for the given combined id
// (format: "providerID:modelID"). If image model middleware is configured,
// it will be applied to the model.
func (r *DefaultProviderRegistry) ImageModel(id string) (imagemodel.ImageModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeImage)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeImage)
	if err != nil {
		return nil, err
	}

	model, err := p.ImageModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeImage,
		})
	}

	if len(r.imageModelMiddleware) > 0 {
		model = wrapImageModel(model, r.imageModelMiddleware)
	}

	return model, nil
}

// TranscriptionModel returns the transcription model for the given combined id
// (format: "providerID:modelID").
func (r *DefaultProviderRegistry) TranscriptionModel(id string) (transcriptionmodel.TranscriptionModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeTranscription)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeTranscription)
	if err != nil {
		return nil, err
	}

	model, err := p.TranscriptionModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeTranscription,
		})
	}

	return model, nil
}

// SpeechModel returns the speech model for the given combined id
// (format: "providerID:modelID").
func (r *DefaultProviderRegistry) SpeechModel(id string) (speechmodel.SpeechModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeSpeech)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeSpeech)
	if err != nil {
		return nil, err
	}

	model, err := p.SpeechModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeSpeech,
		})
	}

	return model, nil
}

// RerankingModel returns the reranking model for the given combined id
// (format: "providerID:modelID").
func (r *DefaultProviderRegistry) RerankingModel(id string) (rerankingmodel.RerankingModel, error) {
	providerID, modelID, err := r.splitID(id, aierrors.ModelTypeReranking)
	if err != nil {
		return nil, err
	}

	p, err := r.getProvider(providerID, aierrors.ModelTypeReranking)
	if err != nil {
		return nil, err
	}

	model, err := p.RerankingModel(modelID)
	if err != nil {
		return nil, err
	}

	if model == nil {
		return nil, aierrors.NewNoSuchModelError(aierrors.NoSuchModelErrorOptions{
			ModelID:   id,
			ModelType: aierrors.ModelTypeReranking,
		})
	}

	return model, nil
}

// --- Middleware wrapping helpers ---
// TODO: Move to brainlink/experiments/ai-kit/middleware/ once wrap-language-model.go and
// wrap-image-model.go are fully ported from:
//   packages/ai/src/middleware/wrap-language-model.ts
//   packages/ai/src/middleware/wrap-image-model.ts

// wrappedLanguageModel wraps a LanguageModel and overrides ModelID().
// Faithfully ports the `doWrap` function from wrap-language-model.ts, limited
// to the overrideModelId middleware hook used by the registry.
type wrappedLanguageModel struct {
	inner     languagemodel.LanguageModel
	modelIDFn func() string
}

func (w *wrappedLanguageModel) SpecificationVersion() string { return w.inner.SpecificationVersion() }
func (w *wrappedLanguageModel) Provider() string             { return w.inner.Provider() }
func (w *wrappedLanguageModel) ModelID() string              { return w.modelIDFn() }
func (w *wrappedLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return w.inner.SupportedUrls()
}
func (w *wrappedLanguageModel) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
func (w *wrappedLanguageModel) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	return w.inner.DoStream(options)
}

// wrapLanguageModel applies a slice of LanguageModelMiddleware to a language model.
// The middlewares are applied in reverse order (last middleware wraps directly
// around the model), matching the TypeScript behavior.
func wrapLanguageModel(model languagemodel.LanguageModel, middlewares []LanguageModelMiddleware) languagemodel.LanguageModel {
	// Apply in reverse order: last middleware wraps directly around the model.
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		model = doWrapLanguageModel(model, mw)
	}
	return model
}

func doWrapLanguageModel(model languagemodel.LanguageModel, mw LanguageModelMiddleware) languagemodel.LanguageModel {
	modelIDFn := model.ModelID
	if mw.OverrideModelID != nil {
		overridden := mw.OverrideModelID(OverrideModelIDOptions{Model: model})
		modelIDFn = func() string { return overridden }
	}

	return &wrappedLanguageModel{
		inner:     model,
		modelIDFn: modelIDFn,
	}
}

// wrappedImageModel wraps an ImageModel and overrides ModelID().
// Faithfully ports the `doWrap` function from wrap-image-model.ts, limited
// to the overrideModelId middleware hook used by the registry.
type wrappedImageModel struct {
	inner     imagemodel.ImageModel
	modelIDFn func() string
}

func (w *wrappedImageModel) SpecificationVersion() string { return w.inner.SpecificationVersion() }
func (w *wrappedImageModel) Provider() string             { return w.inner.Provider() }
func (w *wrappedImageModel) ModelID() string              { return w.modelIDFn() }
func (w *wrappedImageModel) MaxImagesPerCall() (*int, error) {
	return w.inner.MaxImagesPerCall()
}
func (w *wrappedImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}

// wrapImageModel applies a slice of ImageModelMiddleware to an image model.
// The middlewares are applied in reverse order, matching the TypeScript behavior.
func wrapImageModel(model imagemodel.ImageModel, middlewares []ImageModelMiddleware) imagemodel.ImageModel {
	for i := len(middlewares) - 1; i >= 0; i-- {
		mw := middlewares[i]
		model = doWrapImageModel(model, mw)
	}
	return model
}

func doWrapImageModel(model imagemodel.ImageModel, mw ImageModelMiddleware) imagemodel.ImageModel {
	modelIDFn := model.ModelID
	if mw.OverrideModelID != nil {
		overridden := mw.OverrideModelID(ImageOverrideModelIDOptions{Model: model})
		modelIDFn = func() string { return overridden }
	}

	return &wrappedImageModel{
		inner:     model,
		modelIDFn: modelIDFn,
	}
}
