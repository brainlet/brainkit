// Ported from: packages/openai-compatible/src/openai-compatible-provider.ts
package openaicompatible

import (
	"fmt"
	"net/url"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ProviderSettings configures an OpenAI-compatible provider instance.
type ProviderSettings struct {
	// BaseURL is the base URL for the API calls.
	BaseURL string

	// Name is the provider name.
	Name string

	// APIKey is the optional API key for authenticating requests.
	// If specified, adds an Authorization header with the value "Bearer <APIKey>".
	// This is added before any headers specified in the Headers option.
	APIKey string

	// Headers are optional custom headers to include in requests.
	// These are added after any headers potentially added by the APIKey option.
	Headers map[string]string

	// QueryParams are optional custom URL query parameters to include in request URLs.
	QueryParams map[string]string

	// Fetch is an optional custom fetch implementation.
	// Can be used as middleware to intercept requests, or for testing.
	Fetch providerutils.FetchFunction

	// IncludeUsage indicates whether to include usage information in streaming responses.
	IncludeUsage *bool

	// SupportsStructuredOutputs indicates whether the provider supports
	// structured outputs in chat models.
	SupportsStructuredOutputs *bool

	// TransformRequestBody is an optional function to transform the request body
	// before sending it to the API. Useful for proxy providers that may require
	// a different request format than the official OpenAI API.
	TransformRequestBody func(map[string]any) map[string]any

	// MetadataExtractor is an optional metadata extractor to capture
	// provider-specific metadata from API responses.
	MetadataExtractor MetadataExtractor
}

// CommonModelConfig holds the shared configuration passed to all model constructors.
// Mirrors the TS CommonModelConfig interface defined inside createOpenAICompatible.
type CommonModelConfig struct {
	Provider string
	URL      func(path string) string
	Headers  func() map[string]string
	Fetch    providerutils.FetchFunction
}

// Provider implements provider.Provider for OpenAI-compatible APIs.
type Provider struct {
	settings ProviderSettings

	// internal cached values
	baseURL string
	name    string
	headers func() map[string]string
}

// NewProvider creates a new OpenAI-compatible provider instance.
func NewProvider(settings ProviderSettings) *Provider {
	baseURL := providerutils.WithoutTrailingSlash(settings.BaseURL)
	providerName := settings.Name

	// Build static headers: apiKey-based Authorization + user-supplied headers.
	staticHeaders := make(map[string]string)
	if settings.APIKey != "" {
		staticHeaders["Authorization"] = fmt.Sprintf("Bearer %s", settings.APIKey)
	}
	for k, v := range settings.Headers {
		staticHeaders[k] = v
	}

	getHeaders := func() map[string]string {
		return providerutils.WithUserAgentSuffix(
			staticHeaders,
			fmt.Sprintf("ai-sdk/openai-compatible/%s", VERSION),
		)
	}

	return &Provider{
		settings: settings,
		baseURL:  baseURL,
		name:     providerName,
		headers:  getHeaders,
	}
}

// getCommonModelConfig returns the shared configuration for a given model type.
// Mirrors the TS getCommonModelConfig function.
func (p *Provider) getCommonModelConfig(modelType string) CommonModelConfig {
	return CommonModelConfig{
		Provider: fmt.Sprintf("%s.%s", p.name, modelType),
		URL: func(path string) string {
			u, err := url.Parse(fmt.Sprintf("%s%s", p.baseURL, path))
			if err != nil {
				// Fallback to simple concatenation if URL parsing fails.
				return fmt.Sprintf("%s%s", p.baseURL, path)
			}
			if len(p.settings.QueryParams) > 0 {
				q := u.Query()
				for k, v := range p.settings.QueryParams {
					q.Set(k, v)
				}
				u.RawQuery = q.Encode()
			}
			return u.String()
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	}
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel returns a chat language model with the given model ID.
// This is the primary method for obtaining a language model (maps to the TS
// callable provider and languageModel method).
func (p *Provider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.ChatModel(modelID), nil
}

// ChatModel creates a chat language model for the given model ID.
// Convenience method matching the TS chatModel property.
func (p *Provider) ChatModel(modelID string) *ChatLanguageModel {
	cfg := p.getCommonModelConfig("chat")
	return NewChatLanguageModel(modelID, ChatConfig{
		Provider:                  cfg.Provider,
		URL:                       cfg.URL,
		Headers:                   cfg.Headers,
		Fetch:                     cfg.Fetch,
		IncludeUsage:              p.settings.IncludeUsage,
		SupportsStructuredOutputs: p.settings.SupportsStructuredOutputs,
		TransformRequestBody:      p.settings.TransformRequestBody,
		MetadataExtractor:         p.settings.MetadataExtractor,
	})
}

// CompletionModel creates a completion language model for the given model ID.
// Convenience method matching the TS completionModel property.
func (p *Provider) CompletionModel(modelID string) *CompletionLanguageModel {
	cfg := p.getCommonModelConfig("completion")
	return NewCompletionLanguageModel(modelID, CompletionConfig{
		Provider:     cfg.Provider,
		URL:          cfg.URL,
		Headers:      cfg.Headers,
		Fetch:        cfg.Fetch,
		IncludeUsage: p.settings.IncludeUsage,
	})
}

// EmbeddingModel returns an embedding model with the given model ID.
func (p *Provider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.createEmbeddingModel(modelID), nil
}

// createEmbeddingModel creates an embedding model for the given model ID.
func (p *Provider) createEmbeddingModel(modelID string) *EmbeddingModel {
	cfg := p.getCommonModelConfig("embedding")
	return NewEmbeddingModel(modelID, EmbeddingConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// TextEmbeddingModel is a deprecated alias for EmbeddingModel.
// Deprecated: Use EmbeddingModel instead.
func (p *Provider) TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel returns an image model with the given model ID.
func (p *Provider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return p.createImageModel(modelID), nil
}

// createImageModel creates an image model for the given model ID.
func (p *Provider) createImageModel(modelID string) *ImageModel {
	cfg := p.getCommonModelConfig("image")
	return NewImageModel(modelID, ImageModelConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// TranscriptionModel is not supported by OpenAI-compatible providers.
// Returns a NoSuchModelError.
func (p *Provider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by OpenAI-compatible providers.
// Returns a NoSuchModelError.
func (p *Provider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by OpenAI-compatible providers.
// Returns a NoSuchModelError.
func (p *Provider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
