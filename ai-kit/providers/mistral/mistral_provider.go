// Ported from: packages/mistral/src/mistral-provider.ts
package mistral

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

// MistralProviderSettings contains the settings for creating a Mistral provider.
type MistralProviderSettings struct {
	// BaseURL is a custom URL prefix for API calls.
	// Defaults to "https://api.mistral.ai/v1".
	BaseURL *string

	// APIKey is the API key sent via the Authorization header.
	// Defaults to the MISTRAL_API_KEY environment variable.
	APIKey *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is a custom fetch implementation.
	Fetch providerutils.FetchFunction

	// GenerateID is a custom ID generator.
	GenerateID providerutils.IdGenerator
}

// MistralProvider provides access to Mistral AI models.
type MistralProvider struct {
	baseURL    string
	getHeaders func() map[string]string
	fetch      providerutils.FetchFunction
	generateID providerutils.IdGenerator
}

// CreateMistral creates a new Mistral AI provider instance.
func CreateMistral(options *MistralProviderSettings) *MistralProvider {
	if options == nil {
		options = &MistralProviderSettings{}
	}

	baseURL := "https://api.mistral.ai/v1"
	if options.BaseURL != nil {
		baseURL = providerutils.WithoutTrailingSlash(*options.BaseURL)
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  options.APIKey,
			EnvironmentVariableName: "MISTRAL_API_KEY",
			Description:             "Mistral",
		})
		if err != nil {
			// In production, this would need proper error handling.
			// The TS version also throws at call time.
			panic(fmt.Sprintf("Failed to load Mistral API key: %v", err))
		}

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		for k, v := range options.Headers {
			headers[k] = v
		}

		return providerutils.WithUserAgentSuffix(
			headers,
			fmt.Sprintf("ai-sdk/mistral/%s", VERSION),
		)
	}

	return &MistralProvider{
		baseURL:    baseURL,
		getHeaders: getHeaders,
		fetch:      options.Fetch,
		generateID: options.GenerateID,
	}
}

// SpecificationVersion returns "v3".
func (p *MistralProvider) SpecificationVersion() string { return "v3" }

// LanguageModel returns a Mistral chat language model for the given model ID.
func (p *MistralProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.Chat(modelID), nil
}

// Chat creates a Mistral chat language model.
func (p *MistralProvider) Chat(modelID MistralChatModelId) *ChatLanguageModel {
	return NewChatLanguageModel(modelID, ChatConfig{
		Provider:   "mistral.chat",
		BaseURL:    p.baseURL,
		Headers:    p.getHeaders,
		Fetch:      p.fetch,
		GenerateID: p.generateID,
	})
}

// EmbeddingModel returns a Mistral embedding model for the given model ID.
func (p *MistralProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.Embedding(modelID), nil
}

// Embedding creates a Mistral embedding model.
func (p *MistralProvider) Embedding(modelID MistralEmbeddingModelId) *EmbeddingModel {
	return NewEmbeddingModel(modelID, EmbeddingConfig{
		Provider: "mistral.embedding",
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.fetch,
	})
}

// TextEmbedding is a deprecated alias for Embedding.
func (p *MistralProvider) TextEmbedding(modelID MistralEmbeddingModelId) *EmbeddingModel {
	return p.Embedding(modelID)
}

// TextEmbeddingModel is a deprecated alias for EmbeddingModel.
func (p *MistralProvider) TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel returns an error since Mistral does not support image models.
func (p *MistralProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}

// TranscriptionModel returns an error since Mistral does not support transcription models.
func (p *MistralProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel returns an error since Mistral does not support speech models.
func (p *MistralProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel returns an error since Mistral does not support reranking models.
func (p *MistralProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
