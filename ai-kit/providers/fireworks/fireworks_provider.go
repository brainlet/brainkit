// Ported from: packages/fireworks/src/fireworks-provider.ts
package fireworks

import (
	"encoding/json"
	"fmt"
	"net/http"

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

// FireworksErrorData represents the Fireworks error response structure.
type FireworksErrorData struct {
	Error string `json:"error"`
}

// fireworksErrorSchema is the schema for validating Fireworks error responses.
var fireworksErrorSchema = &providerutils.Schema[FireworksErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[FireworksErrorData], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[FireworksErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errData FireworksErrorData
		if err := json.Unmarshal(data, &errData); err != nil {
			return &providerutils.ValidationResult[FireworksErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[FireworksErrorData]{
			Success: true,
			Value:   errData,
		}, nil
	},
}

// fireworksErrorStructure defines how Fireworks API errors are parsed and converted to messages.
var fireworksErrorStructure = openaicompatible.ProviderErrorStructure[FireworksErrorData]{
	ErrorSchema:    fireworksErrorSchema,
	ErrorToMessage: func(data FireworksErrorData) string { return data.Error },
	IsRetryable:    func(resp *http.Response, err *FireworksErrorData) bool { return false },
}

// FireworksProviderSettings configures a Fireworks provider instance.
type FireworksProviderSettings struct {
	// APIKey is the Fireworks API key. Default value is taken from the
	// FIREWORKS_API_KEY environment variable.
	APIKey *string

	// BaseURL is the base URL for the API calls.
	BaseURL *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation. You can use it as a
	// middleware to intercept requests, or to provide a custom fetch
	// implementation for e.g. testing.
	Fetch providerutils.FetchFunction
}

const defaultBaseURL = "https://api.fireworks.ai/inference/v1"

// Provider implements provider.Provider for Fireworks.
type Provider struct {
	settings FireworksProviderSettings
	baseURL  string
	headers  func() map[string]string
	fetch    providerutils.FetchFunction
}

// NewProvider creates a new Fireworks provider instance.
func NewProvider(settings FireworksProviderSettings) *Provider {
	base := defaultBaseURL
	if settings.BaseURL != nil {
		base = *settings.BaseURL
	}
	baseURL := providerutils.WithoutTrailingSlash(base)

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "FIREWORKS_API_KEY",
			Description:             "Fireworks API key",
		})
		if err != nil {
			// If the key cannot be loaded, return headers without Authorization.
			// The error will surface when the actual API call is made.
			h := make(map[string]string)
			for k, v := range settings.Headers {
				h[k] = v
			}
			return providerutils.WithUserAgentSuffix(h, fmt.Sprintf("ai-sdk/fireworks/%s", VERSION))
		}

		h := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		for k, v := range settings.Headers {
			h[k] = v
		}
		return providerutils.WithUserAgentSuffix(h, fmt.Sprintf("ai-sdk/fireworks/%s", VERSION))
	}

	return &Provider{
		settings: settings,
		baseURL:  baseURL,
		headers:  getHeaders,
		fetch:    settings.Fetch,
	}
}

// CreateFireworks creates a new Fireworks provider with the given settings.
func CreateFireworks(settings ...FireworksProviderSettings) *Provider {
	var s FireworksProviderSettings
	if len(settings) > 0 {
		s = settings[0]
	}
	return NewProvider(s)
}

// getCommonModelConfig returns the shared configuration for a given model type.
func (p *Provider) getCommonModelConfig(modelType string) openaicompatible.CommonModelConfig {
	baseURL := p.baseURL
	return openaicompatible.CommonModelConfig{
		Provider: fmt.Sprintf("fireworks.%s", modelType),
		URL: func(path string) string {
			return fmt.Sprintf("%s%s", baseURL, path)
		},
		Headers: p.headers,
		Fetch:   p.fetch,
	}
}

// SpecificationVersion returns the provider interface version.
func (p *Provider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel returns a chat language model with the given model ID.
func (p *Provider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.ChatModel(modelID), nil
}

// ChatModel creates a chat language model for the given model ID.
// The model applies a transformRequestBody that maps Fireworks-specific
// thinking options (budgetTokens -> budget_tokens, reasoningHistory -> reasoning_history).
func (p *Provider) ChatModel(modelID string) *openaicompatible.ChatLanguageModel {
	cfg := p.getCommonModelConfig("chat")
	return openaicompatible.NewChatLanguageModel(modelID, openaicompatible.ChatConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
		ErrorStructure: &openaicompatible.ProviderErrorStructure[openaicompatible.ErrorData]{
			ErrorSchema:    fireworksErrorDataToOpenAIErrorSchema(),
			ErrorToMessage: func(data openaicompatible.ErrorData) string { return data.Error.Message },
		},
		TransformRequestBody: func(args map[string]any) map[string]any {
			// Extract thinking and reasoningHistory from args, transform them
			// to Fireworks API format, and pass remaining args through.
			result := make(map[string]any)
			for k, v := range args {
				if k == "thinking" || k == "reasoningHistory" {
					continue
				}
				result[k] = v
			}

			if thinking, ok := args["thinking"]; ok && thinking != nil {
				if thinkingMap, ok := thinking.(map[string]interface{}); ok {
					transformed := map[string]interface{}{}
					if t, ok := thinkingMap["type"]; ok {
						transformed["type"] = t
					}
					if bt, ok := thinkingMap["budgetTokens"]; ok && bt != nil {
						transformed["budget_tokens"] = bt
					}
					result["thinking"] = transformed
				}
			}

			if reasoningHistory, ok := args["reasoningHistory"]; ok && reasoningHistory != nil {
				result["reasoning_history"] = reasoningHistory
			}

			return result
		},
	})
}

// CompletionModel creates a completion language model for the given model ID.
func (p *Provider) CompletionModel(modelID string) *openaicompatible.CompletionLanguageModel {
	cfg := p.getCommonModelConfig("completion")
	return openaicompatible.NewCompletionLanguageModel(modelID, openaicompatible.CompletionConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
		ErrorStructure: &openaicompatible.ProviderErrorStructure[openaicompatible.ErrorData]{
			ErrorSchema:    fireworksErrorDataToOpenAIErrorSchema(),
			ErrorToMessage: func(data openaicompatible.ErrorData) string { return data.Error.Message },
		},
	})
}

// EmbeddingModel returns an embedding model with the given model ID.
func (p *Provider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return p.createEmbeddingModel(modelID), nil
}

// createEmbeddingModel creates an embedding model for the given model ID.
func (p *Provider) createEmbeddingModel(modelID string) *openaicompatible.EmbeddingModel {
	cfg := p.getCommonModelConfig("embedding")
	return openaicompatible.NewEmbeddingModel(modelID, openaicompatible.EmbeddingConfig{
		Provider: cfg.Provider,
		URL:      cfg.URL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
		ErrorStructure: &openaicompatible.ProviderErrorStructure[openaicompatible.ErrorData]{
			ErrorSchema:    fireworksErrorDataToOpenAIErrorSchema(),
			ErrorToMessage: func(data openaicompatible.ErrorData) string { return data.Error.Message },
		},
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

// Image creates an image model with the given model ID.
func (p *Provider) Image(modelID string) *FireworksImageModel {
	return p.createImageModel(modelID)
}

// createImageModel creates a Fireworks-specific image model for the given model ID.
func (p *Provider) createImageModel(modelID string) *FireworksImageModel {
	cfg := p.getCommonModelConfig("image")
	return NewFireworksImageModel(modelID, FireworksImageModelConfig{
		Provider: cfg.Provider,
		BaseURL:  p.baseURL,
		Headers:  cfg.Headers,
		Fetch:    cfg.Fetch,
	})
}

// TranscriptionModel is not supported by the Fireworks provider.
// Returns a NoSuchModelError.
func (p *Provider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by the Fireworks provider.
// Returns a NoSuchModelError.
func (p *Provider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by the Fireworks provider.
// Returns a NoSuchModelError.
func (p *Provider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}

// fireworksErrorDataToOpenAIErrorSchema creates a schema that wraps the Fireworks
// error format ({ error: string }) into the OpenAI error format
// ({ error: { message: string } }) that the openaicompatible package expects.
func fireworksErrorDataToOpenAIErrorSchema() *providerutils.Schema[openaicompatible.ErrorData] {
	return &providerutils.Schema[openaicompatible.ErrorData]{
		Validate: func(value interface{}) (*providerutils.ValidationResult[openaicompatible.ErrorData], error) {
			data, err := json.Marshal(value)
			if err != nil {
				return &providerutils.ValidationResult[openaicompatible.ErrorData]{
					Success: false,
					Error:   err,
				}, nil
			}

			// Try to parse as Fireworks error format first: { "error": "message" }
			var fireworksErr FireworksErrorData
			if err := json.Unmarshal(data, &fireworksErr); err == nil && fireworksErr.Error != "" {
				return &providerutils.ValidationResult[openaicompatible.ErrorData]{
					Success: true,
					Value: openaicompatible.ErrorData{
						Error: openaicompatible.ErrorDataError{
							Message: fireworksErr.Error,
						},
					},
				}, nil
			}

			// Fallback: try parsing as standard OpenAI error format
			var openAIErr openaicompatible.ErrorData
			if err := json.Unmarshal(data, &openAIErr); err != nil {
				return &providerutils.ValidationResult[openaicompatible.ErrorData]{
					Success: false,
					Error:   err,
				}, nil
			}
			return &providerutils.ValidationResult[openaicompatible.ErrorData]{
				Success: true,
				Value:   openAIErr,
			}, nil
		},
	}
}
