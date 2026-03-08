// Ported from: packages/groq/src/groq-provider.ts
package groq

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ProviderSettings configures a Groq provider instance.
type ProviderSettings struct {
	// BaseURL is the base URL for the Groq API calls.
	BaseURL string

	// APIKey is the API key for authenticating requests.
	APIKey string

	// Headers are optional custom headers to include in requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// Provider implements the Groq provider.
type Provider struct {
	settings ProviderSettings
	baseURL  string
	headers  func() map[string]string

	// Tools holds the tools provided by Groq.
	Tools struct {
		BrowserSearch func(opts providerutils.ProviderToolOptions[struct{}, interface{}]) providerutils.ProviderTool[struct{}, interface{}]
	}
}

// NewProvider creates a new Groq provider instance.
func NewProvider(settings ProviderSettings) *Provider {
	baseURL := providerutils.WithoutTrailingSlash(settings.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	// Build static headers
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
			fmt.Sprintf("ai-sdk/groq/%s", VERSION),
		)
	}

	p := &Provider{
		settings: settings,
		baseURL:  baseURL,
		headers:  getHeaders,
	}
	p.Tools.BrowserSearch = BrowserSearch

	return p
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel returns a chat language model with the given model ID.
func (p *Provider) LanguageModel(modelID GroqChatModelId) (languagemodel.LanguageModel, error) {
	return p.ChatModel(modelID), nil
}

// ChatModel creates a chat language model for the given model ID.
func (p *Provider) ChatModel(modelID GroqChatModelId) *GroqChatLanguageModel {
	return NewGroqChatLanguageModel(modelID, GroqChatConfig{
		Provider: "groq.chat",
		URL: func(_ string, path string) string {
			return fmt.Sprintf("%s%s", p.baseURL, path)
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	})
}

// TranscriptionModel creates a transcription model for the given model ID.
func (p *Provider) TranscriptionModel(modelID GroqTranscriptionModelId) (transcriptionmodel.TranscriptionModel, error) {
	return p.createTranscriptionModel(modelID), nil
}

// createTranscriptionModel creates a transcription model for the given model ID.
func (p *Provider) createTranscriptionModel(modelID GroqTranscriptionModelId) *GroqTranscriptionModel {
	return NewGroqTranscriptionModel(modelID, GroqTranscriptionModelConfig{
		GroqConfig: GroqConfig{
			Provider: "groq.transcription",
			URL: func(_ string, path string) string {
				return fmt.Sprintf("%s%s", p.baseURL, path)
			},
			Headers: p.headers,
			Fetch:   p.settings.Fetch,
		},
	})
}

// EmbeddingModel is not supported by Groq. Returns a NoSuchModelError.
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

// ImageModel is not supported by Groq. Returns a NoSuchModelError.
func (p *Provider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}
