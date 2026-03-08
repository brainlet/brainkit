// Ported from: packages/perplexity/src/perplexity-provider.ts
package perplexity

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

// PerplexityProviderSettings configures the Perplexity provider.
type PerplexityProviderSettings struct {
	// BaseURL is the base URL for the Perplexity API calls.
	BaseURL *string

	// APIKey is the API key for authenticating requests.
	APIKey *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// PerplexityProvider implements provider.Provider for Perplexity.
type PerplexityProvider struct {
	settings   PerplexityProviderSettings
	getHeaders func() map[string]string
	baseURL    string
}

// NewPerplexityProvider creates a new Perplexity provider instance.
// Equivalent to createPerplexity() in the TS SDK.
func NewPerplexityProvider(options ...PerplexityProviderSettings) *PerplexityProvider {
	var settings PerplexityProviderSettings
	if len(options) > 0 {
		settings = options[0]
	}

	baseURL := "https://api.perplexity.ai"
	if settings.BaseURL != nil {
		baseURL = providerutils.WithoutTrailingSlash(*settings.BaseURL)
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "PERPLEXITY_API_KEY",
			Description:             "Perplexity",
		})
		if err != nil {
			// In the TS SDK, loadApiKey throws synchronously.
			// Here we panic since headers are computed lazily.
			panic(err)
		}

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		for k, v := range settings.Headers {
			headers[k] = v
		}
		return providerutils.WithUserAgentSuffix(
			headers,
			fmt.Sprintf("ai-sdk/perplexity/%s", VERSION),
		)
	}

	return &PerplexityProvider{
		settings:   settings,
		getHeaders: getHeaders,
		baseURL:    baseURL,
	}
}

// SpecificationVersion returns "v3".
func (p *PerplexityProvider) SpecificationVersion() string { return "v3" }

// LanguageModel creates a Perplexity language model for text generation.
func (p *PerplexityProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createLanguageModel(modelID), nil
}

// createLanguageModel creates a new PerplexityLanguageModel.
func (p *PerplexityProvider) createLanguageModel(modelID PerplexityLanguageModelID) *PerplexityLanguageModel {
	return NewPerplexityLanguageModel(modelID, PerplexityChatConfig{
		BaseURL:    p.baseURL,
		Headers:    p.getHeaders,
		GenerateID: providerutils.GenerateId,
		Fetch:      p.settings.Fetch,
	})
}

// EmbeddingModel is not supported by Perplexity.
func (p *PerplexityProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeEmbedding,
	})
}

// ImageModel is not supported by Perplexity.
func (p *PerplexityProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}

// TranscriptionModel is not supported by Perplexity.
func (p *PerplexityProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by Perplexity.
func (p *PerplexityProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by Perplexity.
func (p *PerplexityProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
