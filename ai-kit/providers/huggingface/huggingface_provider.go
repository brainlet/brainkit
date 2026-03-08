// Ported from: packages/huggingface/src/huggingface-provider.ts
package huggingface

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ProviderSettings configures a HuggingFace provider instance.
type ProviderSettings struct {
	// APIKey is the HuggingFace API key.
	APIKey *string

	// BaseURL is the base URL for the API calls.
	BaseURL *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional ID generator function.
	GenerateID providerutils.IdGenerator
}

// Provider implements provider.Provider for HuggingFace.
type Provider struct {
	settings ProviderSettings
	baseURL  string
	headers  func() map[string]string
}

// NewProvider creates a new HuggingFace provider instance.
func NewProvider(settings ProviderSettings) *Provider {
	baseURL := "https://router.huggingface.co/v1"
	if settings.BaseURL != nil {
		baseURL = providerutils.WithoutTrailingSlash(*settings.BaseURL)
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "HUGGINGFACE_API_KEY",
			Description:             "Hugging Face",
		})

		headers := make(map[string]string)
		if err == nil && apiKey != "" {
			headers["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
		}

		for k, v := range settings.Headers {
			headers[k] = v
		}

		return headers
	}

	return &Provider{
		settings: settings,
		baseURL:  baseURL,
		headers:  getHeaders,
	}
}

// createResponsesModel creates a HuggingFace responses language model.
func (p *Provider) createResponsesModel(modelID ResponsesModelID) *ResponsesLanguageModel {
	generateID := p.settings.GenerateID
	if generateID == nil {
		generateID = providerutils.GenerateId
	}

	return NewResponsesLanguageModel(modelID, Config{
		Provider: "huggingface.responses",
		URL: func(opts URLOptions) string {
			return fmt.Sprintf("%s%s", p.baseURL, opts.Path)
		},
		Headers:    p.headers,
		Fetch:      p.settings.Fetch,
		GenerateID: generateID,
	})
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel returns a language model with the given model ID.
func (p *Provider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createResponsesModel(modelID), nil
}

// Responses creates a HuggingFace responses model for text generation.
func (p *Provider) Responses(modelID ResponsesModelID) *ResponsesLanguageModel {
	return p.createResponsesModel(modelID)
}

// EmbeddingModel is not supported by HuggingFace Responses API.
// Returns a NoSuchModelError.
func (p *Provider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeEmbedding,
		Message:   "Hugging Face Responses API does not support text embeddings. Use the Hugging Face Inference API directly for embeddings.",
	})
}

// TextEmbeddingModel is a deprecated alias for EmbeddingModel.
// Deprecated: Use EmbeddingModel instead.
func (p *Provider) TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel is not supported by HuggingFace Responses API.
// Returns a NoSuchModelError.
func (p *Provider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
		Message:   "Hugging Face Responses API does not support image generation. Use the Hugging Face Inference API directly for image models.",
	})
}

// TranscriptionModel is not supported by HuggingFace Responses API.
// Returns a NoSuchModelError.
func (p *Provider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by HuggingFace Responses API.
// Returns a NoSuchModelError.
func (p *Provider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by HuggingFace Responses API.
// Returns a NoSuchModelError.
func (p *Provider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
