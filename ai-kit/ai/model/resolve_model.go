// Ported from: packages/ai/src/model/resolve-model.ts
package model

import (
	"errors"
	"fmt"

	aierror "github.com/brainlet/brainkit/ai-kit/ai/error"
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// VersionedModel is an interface for models that have a specification version,
// provider, and model ID. All versioned model types satisfy this interface.
type VersionedModel interface {
	SpecificationVersion() string
	Provider() string
	ModelID() string
}

// DefaultProvider is the global default provider. In the TS SDK, this is
// globalThis.AI_SDK_DEFAULT_PROVIDER. Set this to configure a custom
// default provider for string-based model resolution.
//
// If nil, model resolution with string IDs will fail with an error.
var DefaultProvider ProviderV4

// ResolveLanguageModel resolves a language model from either a string model ID
// or a model object. If a string is provided, the global default provider is used.
// If a model object is provided, it is converted to v4 if necessary.
func ResolveLanguageModel(model interface{}) (languagemodel.LanguageModel, error) {
	if modelID, ok := model.(string); ok {
		p, err := getGlobalProvider()
		if err != nil {
			return nil, err
		}
		return p.LanguageModel(modelID)
	}

	lm, ok := model.(languagemodel.LanguageModel)
	if !ok {
		return nil, errors.New("model must be a string or implement languagemodel.LanguageModel")
	}

	ver := lm.SpecificationVersion()
	if ver != "v4" && ver != "v3" && ver != "v2" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, lm.Provider(), lm.ModelID())
	}

	return AsLanguageModelV4(lm), nil
}

// ResolveEmbeddingModel resolves an embedding model from either a string model ID
// or a model object. If a string is provided, the global default provider is used.
// If a model object is provided, it is converted to v4 if necessary.
func ResolveEmbeddingModel(model interface{}) (embeddingmodel.EmbeddingModel, error) {
	if modelID, ok := model.(string); ok {
		p, err := getGlobalProvider()
		if err != nil {
			return nil, err
		}
		return p.EmbeddingModel(modelID)
	}

	em, ok := model.(embeddingmodel.EmbeddingModel)
	if !ok {
		return nil, errors.New("model must be a string or implement embeddingmodel.EmbeddingModel")
	}

	ver := em.SpecificationVersion()
	if ver != "v4" && ver != "v3" && ver != "v2" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, em.Provider(), em.ModelID())
	}

	return AsEmbeddingModelV4(em), nil
}

// ResolveTranscriptionModel resolves a transcription model from either a string
// model ID or a model object. Returns (nil, nil) if the provider doesn't
// support transcription models.
func ResolveTranscriptionModel(model interface{}) (transcriptionmodel.TranscriptionModel, error) {
	if modelID, ok := model.(string); ok {
		p, err := getGlobalProvider()
		if err != nil {
			return nil, err
		}
		return p.TranscriptionModel(modelID)
	}

	tm, ok := model.(transcriptionmodel.TranscriptionModel)
	if !ok {
		return nil, errors.New("model must be a string or implement transcriptionmodel.TranscriptionModel")
	}

	ver := tm.SpecificationVersion()
	if ver != "v4" && ver != "v3" && ver != "v2" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, tm.Provider(), tm.ModelID())
	}

	return AsTranscriptionModelV4(tm), nil
}

// ResolveSpeechModel resolves a speech model from either a string model ID
// or a model object. Returns (nil, nil) if the provider doesn't support
// speech models.
func ResolveSpeechModel(model interface{}) (speechmodel.SpeechModel, error) {
	if modelID, ok := model.(string); ok {
		p, err := getGlobalProvider()
		if err != nil {
			return nil, err
		}
		return p.SpeechModel(modelID)
	}

	sm, ok := model.(speechmodel.SpeechModel)
	if !ok {
		return nil, errors.New("model must be a string or implement speechmodel.SpeechModel")
	}

	ver := sm.SpecificationVersion()
	if ver != "v4" && ver != "v3" && ver != "v2" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, sm.Provider(), sm.ModelID())
	}

	return AsSpeechModelV4(sm), nil
}

// ResolveImageModel resolves an image model from either a string model ID
// or a model object. If a string is provided, the global default provider is used.
// If a model object is provided, it is converted to v4 if necessary.
func ResolveImageModel(model interface{}) (imagemodel.ImageModel, error) {
	if modelID, ok := model.(string); ok {
		p, err := getGlobalProvider()
		if err != nil {
			return nil, err
		}
		return p.ImageModel(modelID)
	}

	im, ok := model.(imagemodel.ImageModel)
	if !ok {
		return nil, errors.New("model must be a string or implement imagemodel.ImageModel")
	}

	ver := im.SpecificationVersion()
	if ver != "v4" && ver != "v3" && ver != "v2" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, im.Provider(), im.ModelID())
	}

	return AsImageModelV4(im), nil
}

// VideoModelProvider is an interface for providers that support video models.
// This is separate because video model support is experimental and not part
// of the standard ProviderV4 interface.
type VideoModelProvider interface {
	VideoModel(modelID string) (videomodel.VideoModel, error)
}

// ResolveVideoModel resolves a video model from either a string model ID
// or a model object. If a string is provided, the global default provider is used
// (which must implement VideoModelProvider).
func ResolveVideoModel(model interface{}) (videomodel.VideoModel, error) {
	if modelID, ok := model.(string); ok {
		if DefaultProvider == nil {
			return nil, errors.New("no default provider configured; set model.DefaultProvider")
		}
		vp, ok := DefaultProvider.(VideoModelProvider)
		if !ok {
			return nil, errors.New(
				"The default provider does not support video models. " +
					"Please use a Experimental_VideoModelV4 object from a provider (e.g., vertex.Video(\"model-id\")).",
			)
		}
		return vp.VideoModel(modelID)
	}

	vm, ok := model.(videomodel.VideoModel)
	if !ok {
		return nil, errors.New("model must be a string or implement videomodel.VideoModel")
	}

	ver := vm.SpecificationVersion()
	if ver != "v4" && ver != "v3" {
		return nil, aierror.NewUnsupportedModelVersionError(ver, vm.Provider(), vm.ModelID())
	}

	return AsVideoModelV4(vm), nil
}

// getGlobalProvider returns the global default provider, or an error if none is set.
func getGlobalProvider() (ProviderV4, error) {
	if DefaultProvider == nil {
		return nil, fmt.Errorf("no default provider configured; set model.DefaultProvider")
	}
	return AsProviderV4(DefaultProvider), nil
}
