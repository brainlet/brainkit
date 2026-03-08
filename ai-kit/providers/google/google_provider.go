// Ported from: packages/google/src/google-provider.ts
package google

import (
	"fmt"
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleProviderSettings configures a Google Generative AI provider instance.
type GoogleProviderSettings struct {
	// BaseURL is the URL prefix for API calls.
	// Default: "https://generativelanguage.googleapis.com/v1beta"
	BaseURL string

	// APIKey is the API key sent using the "x-goog-api-key" header.
	// Defaults to the GOOGLE_GENERATIVE_AI_API_KEY environment variable.
	APIKey *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional function to generate a unique ID for each request.
	GenerateID providerutils.IdGenerator

	// Name is a custom provider name. Default: "google.generative-ai"
	Name string
}

// GoogleProvider implements provider.Provider for the Google Generative AI API.
type GoogleProvider struct {
	settings     GoogleProviderSettings
	baseURL      string
	providerName string
	getHeaders   func() map[string]string
}

// NewGoogleProvider creates a new Google Generative AI provider instance.
func NewGoogleProvider(settings GoogleProviderSettings) *GoogleProvider {
	baseURL := "https://generativelanguage.googleapis.com/v1beta"
	if settings.BaseURL != "" {
		baseURL = providerutils.WithoutTrailingSlash(settings.BaseURL)
	}

	providerName := "google.generative-ai"
	if settings.Name != "" {
		providerName = settings.Name
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "GOOGLE_GENERATIVE_AI_API_KEY",
			Description:             "Google Generative AI",
		})
		if err != nil {
			// If API key is not found, return headers without it.
			// The error will surface when the API call is made.
			h := make(map[string]string)
			for k, v := range settings.Headers {
				h[k] = v
			}
			return providerutils.WithUserAgentSuffix(h, fmt.Sprintf("ai-sdk/google/%s", VERSION))
		}
		h := map[string]string{
			"x-goog-api-key": apiKey,
		}
		for k, v := range settings.Headers {
			h[k] = v
		}
		return providerutils.WithUserAgentSuffix(h, fmt.Sprintf("ai-sdk/google/%s", VERSION))
	}

	return &GoogleProvider{
		settings:     settings,
		baseURL:      baseURL,
		providerName: providerName,
		getHeaders:   getHeaders,
	}
}

// SpecificationVersion returns the provider interface version.
func (p *GoogleProvider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel returns a language model with the given model ID.
func (p *GoogleProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createChatModel(modelID), nil
}

// ChatModel creates a chat model (convenience alias for LanguageModel).
func (p *GoogleProvider) ChatModel(modelID string) *GoogleLanguageModel {
	return p.createChatModel(modelID)
}

// GenerativeAI creates a language model (deprecated: use ChatModel instead).
func (p *GoogleProvider) GenerativeAI(modelID string) *GoogleLanguageModel {
	return p.createChatModel(modelID)
}

// EmbeddingModel returns an embedding model with the given model ID.
func (p *GoogleProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.createEmbeddingModel(modelID), nil
}

// Embedding creates an embedding model (convenience alias for EmbeddingModel).
func (p *GoogleProvider) Embedding(modelID string) *GoogleEmbeddingModel {
	return p.createEmbeddingModel(modelID)
}

// TextEmbedding creates an embedding model (deprecated: use Embedding instead).
func (p *GoogleProvider) TextEmbedding(modelID string) *GoogleEmbeddingModel {
	return p.createEmbeddingModel(modelID)
}

// TextEmbeddingModel creates an embedding model (deprecated: use EmbeddingModel instead).
func (p *GoogleProvider) TextEmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.EmbeddingModel(modelID)
}

// ImageModel returns an image model with the given model ID.
func (p *GoogleProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return p.createImageModel(modelID, GoogleImageSettings{}), nil
}

// Image creates an image model with optional settings.
func (p *GoogleProvider) Image(modelID string, settings GoogleImageSettings) *GoogleImageModel {
	return p.createImageModel(modelID, settings)
}

// VideoModel returns a video model with the given model ID.
func (p *GoogleProvider) VideoModel(modelID string) (videomodel.VideoModel, error) {
	return p.createVideoModel(modelID), nil
}

// Video creates a video model (convenience alias for VideoModel).
func (p *GoogleProvider) Video(modelID string) *GoogleVideoModel {
	return p.createVideoModel(modelID)
}

// Tools returns the Google provider tools.
func (p *GoogleProvider) Tools() interface{} {
	return GoogleTools
}

// TranscriptionModel is not supported.
func (p *GoogleProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported.
func (p *GoogleProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported.
func (p *GoogleProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}

// --- internal model creation ---

func (p *GoogleProvider) createChatModel(modelID string) *GoogleLanguageModel {
	genID := p.settings.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}

	baseURL := p.baseURL

	return NewGoogleLanguageModel(modelID, GoogleLanguageModelConfig{
		Provider:   p.providerName,
		BaseURL:    baseURL,
		Headers:    p.getHeaders,
		GenerateID: genID,
		SupportedUrls: func() map[string][]*regexp.Regexp {
			return map[string][]*regexp.Regexp{
				"*": {
					// Google Generative Language "files" endpoint
					regexp.MustCompile(fmt.Sprintf(`^%s/files/.*$`, regexp.QuoteMeta(baseURL))),
					// YouTube URLs (public or unlisted videos)
					regexp.MustCompile(`^https://(?:www\.)?youtube\.com/watch\?v=[\w-]+(?:&[\w=&.-]*)?$`),
					regexp.MustCompile(`^https://youtu\.be/[\w-]+(?:\?[\w=&.-]*)?$`),
				},
			}
		},
		Fetch: p.settings.Fetch,
	})
}

func (p *GoogleProvider) createEmbeddingModel(modelID string) *GoogleEmbeddingModel {
	return NewGoogleEmbeddingModel(modelID, GoogleEmbeddingModelConfig{
		Provider: p.providerName,
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.settings.Fetch,
	})
}

func (p *GoogleProvider) createImageModel(modelID string, settings GoogleImageSettings) *GoogleImageModel {
	return NewGoogleImageModel(modelID, settings, GoogleImageModelConfig{
		Provider: p.providerName,
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.settings.Fetch,
	})
}

func (p *GoogleProvider) createVideoModel(modelID string) *GoogleVideoModel {
	genID := p.settings.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}

	return NewGoogleVideoModel(modelID, GoogleVideoModelConfig{
		Provider:   p.providerName,
		BaseURL:    p.baseURL,
		Headers:    p.getHeaders,
		Fetch:      p.settings.Fetch,
		GenerateID: genID,
	})
}
