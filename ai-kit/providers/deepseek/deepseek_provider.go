// Ported from: packages/deepseek/src/deepseek-provider.ts
package deepseek

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

// ProviderSettings configures a DeepSeek provider instance.
type ProviderSettings struct {
	// APIKey is the DeepSeek API key.
	APIKey *string

	// BaseURL is the base URL for the API calls.
	BaseURL *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is a custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// Provider implements the provider interface for DeepSeek APIs.
type Provider struct {
	settings ProviderSettings

	baseURL string
	headers func() map[string]string
}

// NewProvider creates a new DeepSeek provider instance with the given settings.
func NewProvider(settings ProviderSettings) *Provider {
	baseURL := "https://api.deepseek.com"
	if settings.BaseURL != nil {
		baseURL = *settings.BaseURL
	}
	baseURL = providerutils.WithoutTrailingSlash(baseURL)

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "DEEPSEEK_API_KEY",
			Description:             "DeepSeek",
		})
		if err != nil {
			// If API key is not available, proceed without it.
			// The error will surface when an actual API call is made.
			apiKey = ""
		}

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		for k, v := range settings.Headers {
			headers[k] = v
		}

		return providerutils.WithUserAgentSuffix(
			headers,
			fmt.Sprintf("ai-sdk/deepseek/%s", VERSION),
		)
	}

	return &Provider{
		settings: settings,
		baseURL:  baseURL,
		headers:  getHeaders,
	}
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string {
	return "v3"
}

// createLanguageModel creates a DeepSeek chat language model.
func (p *Provider) createLanguageModel(modelID DeepSeekChatModelId) *ChatLanguageModel {
	baseURL := p.baseURL
	return NewChatLanguageModel(modelID, ChatConfig{
		Provider: "deepseek.chat",
		URL: func(modelID string, path string) string {
			return fmt.Sprintf("%s%s", baseURL, path)
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	})
}

// LanguageModel returns a language model with the given model ID.
func (p *Provider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createLanguageModel(modelID), nil
}

// ChatModel creates a chat language model for the given model ID.
// Convenience method matching the TS chat property.
func (p *Provider) ChatModel(modelID DeepSeekChatModelId) *ChatLanguageModel {
	return p.createLanguageModel(modelID)
}

// EmbeddingModel is not supported by the DeepSeek provider.
// Returns a NoSuchModelError.
func (p *Provider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeEmbedding,
	})
}

// TextEmbeddingModel is a deprecated alias for EmbeddingModel.
// Deprecated: Use EmbeddingModel instead.
func (p *Provider) TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel is not supported by the DeepSeek provider.
// Returns a NoSuchModelError.
func (p *Provider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}

// TranscriptionModel is not supported by the DeepSeek provider.
// Returns a NoSuchModelError.
func (p *Provider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by the DeepSeek provider.
// Returns a NoSuchModelError.
func (p *Provider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by the DeepSeek provider.
// Returns a NoSuchModelError.
func (p *Provider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}

// CreateDeepSeek creates a new DeepSeek provider with the given settings.
func CreateDeepSeek(settings ...ProviderSettings) *Provider {
	s := ProviderSettings{}
	if len(settings) > 0 {
		s = settings[0]
	}
	return NewProvider(s)
}
