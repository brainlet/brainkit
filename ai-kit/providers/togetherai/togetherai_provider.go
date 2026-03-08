// Ported from: packages/togetherai/src/togetherai-provider.ts
package togetherai

import (
	"fmt"
	"log"
	"os"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providers/openaicompatible"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ProviderSettings configures a Together AI provider instance.
type ProviderSettings struct {
	// APIKey is the Together AI API key.
	APIKey *string

	// BaseURL is the base URL for the API calls.
	BaseURL *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	// Can be used as middleware to intercept requests, or for testing.
	Fetch providerutils.FetchFunction
}

// Provider implements the provider.Provider interface for Together AI.
type Provider struct {
	baseURL    string
	getHeaders func() map[string]string
	fetch      providerutils.FetchFunction
}

// NewProvider creates a new Together AI provider instance.
func NewProvider(settings ProviderSettings) *Provider {
	baseURL := "https://api.together.xyz/v1"
	if settings.BaseURL != nil {
		baseURL = providerutils.WithoutTrailingSlash(*settings.BaseURL)
	}

	apiKey := loadAPIKey(settings.APIKey)

	// Build static headers
	staticHeaders := make(map[string]string)
	staticHeaders["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
	for k, v := range settings.Headers {
		staticHeaders[k] = v
	}

	getHeaders := func() map[string]string {
		return providerutils.WithUserAgentSuffix(
			staticHeaders,
			fmt.Sprintf("ai-sdk/togetherai/%s", VERSION),
		)
	}

	return &Provider{
		baseURL:    baseURL,
		getHeaders: getHeaders,
		fetch:      settings.Fetch,
	}
}

// loadAPIKey loads the API key from the settings or environment variables.
// Supports the deprecated TOGETHER_AI_API_KEY env var with a warning.
func loadAPIKey(apiKey *string) string {
	if apiKey != nil {
		return *apiKey
	}

	// Check the primary env var
	if key := os.Getenv("TOGETHER_API_KEY"); key != "" {
		return key
	}

	// Check the deprecated env var
	if key := os.Getenv("TOGETHER_AI_API_KEY"); key != "" {
		log.Println("TOGETHER_AI_API_KEY is deprecated and will be removed in a future release. Please use TOGETHER_API_KEY instead.")
		return key
	}

	// Fall back to LoadApiKey for the canonical error message
	key, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
		EnvironmentVariableName: "TOGETHER_API_KEY",
		Description:             "TogetherAI",
	})
	if err != nil {
		// Return empty string; the API call will fail with auth error
		return ""
	}
	return key
}

// getCommonModelConfig returns shared model configuration for a given model type.
func (p *Provider) getCommonModelConfig(modelType string) openaicompatible.CommonModelConfig {
	return openaicompatible.CommonModelConfig{
		Provider: fmt.Sprintf("togetherai.%s", modelType),
		URL: func(path string) string {
			return fmt.Sprintf("%s%s", p.baseURL, path)
		},
		Headers: p.getHeaders,
		Fetch:   p.fetch,
	}
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string { return "v3" }

// LanguageModel returns a chat language model with the given model ID.
func (p *Provider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.ChatModel(modelID), nil
}

// ChatModel creates a chat language model for the given model ID.
func (p *Provider) ChatModel(modelID TogetherAIChatModelID) *openaicompatible.ChatLanguageModel {
	cfg := p.getCommonModelConfig("chat")
	return openaicompatible.NewChatLanguageModel(modelID, openaicompatible.ChatConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// CompletionModel creates a completion language model for the given model ID.
func (p *Provider) CompletionModel(modelID TogetherAICompletionModelID) *openaicompatible.CompletionLanguageModel {
	cfg := p.getCommonModelConfig("completion")
	return openaicompatible.NewCompletionLanguageModel(modelID, openaicompatible.CompletionConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// EmbeddingModel returns an embedding model with the given model ID.
func (p *Provider) EmbeddingModel(modelID TogetherAIEmbeddingModelID) (embeddingmodel.EmbeddingModel, error) {
	return p.createEmbeddingModel(modelID), nil
}

// createEmbeddingModel creates an embedding model for the given model ID.
func (p *Provider) createEmbeddingModel(modelID TogetherAIEmbeddingModelID) *openaicompatible.EmbeddingModel {
	cfg := p.getCommonModelConfig("embedding")
	return openaicompatible.NewEmbeddingModel(modelID, openaicompatible.EmbeddingConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// TextEmbeddingModel is a deprecated alias for EmbeddingModel.
// Deprecated: Use EmbeddingModel instead.
func (p *Provider) TextEmbeddingModel(modelID TogetherAIEmbeddingModelID) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel returns a Together AI image model with the given model ID.
func (p *Provider) ImageModel(modelID TogetherAIImageModelID) (imagemodel.ImageModel, error) {
	return p.createImageModel(modelID), nil
}

// Image creates a Together AI image model with the given model ID.
func (p *Provider) Image(modelID TogetherAIImageModelID) *TogetherAIImageModel {
	return p.createImageModel(modelID)
}

// createImageModel creates an image model for the given model ID.
func (p *Provider) createImageModel(modelID TogetherAIImageModelID) *TogetherAIImageModel {
	return NewTogetherAIImageModel(modelID, TogetherAIImageModelConfig{
		Provider: fmt.Sprintf("togetherai.%s", "image"),
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.fetch,
	})
}

// RerankingModel returns a Together AI reranking model with the given model ID.
func (p *Provider) RerankingModel(modelID TogetherAIRerankingModelID) (rerankingmodel.RerankingModel, error) {
	return p.createRerankingModel(modelID), nil
}

// Reranking creates a Together AI reranking model with the given model ID.
func (p *Provider) Reranking(modelID TogetherAIRerankingModelID) *TogetherAIRerankingModel {
	return p.createRerankingModel(modelID)
}

// createRerankingModel creates a reranking model for the given model ID.
func (p *Provider) createRerankingModel(modelID TogetherAIRerankingModelID) *TogetherAIRerankingModel {
	return NewTogetherAIRerankingModel(modelID, TogetherAIRerankingConfig{
		Provider: fmt.Sprintf("togetherai.%s", "reranking"),
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.fetch,
	})
}

// TranscriptionModel is not supported by Together AI.
// Returns a NoSuchModelError.
func (p *Provider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by Together AI.
// Returns a NoSuchModelError.
func (p *Provider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}
