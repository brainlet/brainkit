// Ported from: packages/cerebras/src/cerebras-provider.ts
package cerebras

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

// --- Error schema and structure ---

// CerebrasErrorData represents the Cerebras API error response.
// The Cerebras API returns errors as: { "message": "...", "type": "...", "param": "...", "code": "..." }
type CerebrasErrorData struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param"`
	Code    string `json:"code"`
}

// cerebrasErrorDataToOpenAIErrorSchema creates a schema that wraps the Cerebras
// error format ({ message, type, param, code }) into the OpenAI error format
// ({ error: { message } }) that the openaicompatible package expects.
func cerebrasErrorDataToOpenAIErrorSchema() *providerutils.Schema[openaicompatible.ErrorData] {
	return &providerutils.Schema[openaicompatible.ErrorData]{
		Validate: func(value interface{}) (*providerutils.ValidationResult[openaicompatible.ErrorData], error) {
			data, err := json.Marshal(value)
			if err != nil {
				return &providerutils.ValidationResult[openaicompatible.ErrorData]{
					Success: false,
					Error:   err,
				}, nil
			}

			// Try to parse as Cerebras error format first: { "message": "...", "type": "...", ... }
			var cerebrasErr CerebrasErrorData
			if err := json.Unmarshal(data, &cerebrasErr); err == nil && cerebrasErr.Message != "" {
				return &providerutils.ValidationResult[openaicompatible.ErrorData]{
					Success: true,
					Value: openaicompatible.ErrorData{
						Error: openaicompatible.ErrorDataError{
							Message: cerebrasErr.Message,
							Type:    &cerebrasErr.Type,
							Param:   cerebrasErr.Param,
							Code:    cerebrasErr.Code,
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

var cerebrasErrorStructure = &openaicompatible.ProviderErrorStructure[openaicompatible.ErrorData]{
	ErrorSchema:    cerebrasErrorDataToOpenAIErrorSchema(),
	ErrorToMessage: func(data openaicompatible.ErrorData) string { return data.Error.Message },
	IsRetryable:    func(resp *http.Response, err *openaicompatible.ErrorData) bool { return false },
}

// --- Provider settings ---

// CerebrasProviderSettings configures the Cerebras provider.
type CerebrasProviderSettings struct {
	// APIKey is the Cerebras API key.
	APIKey *string

	// BaseURL is the base URL for the API calls.
	BaseURL *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// CerebrasProvider implements provider.Provider for Cerebras.
type CerebrasProvider struct {
	settings   CerebrasProviderSettings
	baseURL    string
	getHeaders func() map[string]string
}

// NewCerebrasProvider creates a new Cerebras provider instance.
// Equivalent to createCerebras() in the TS SDK.
func NewCerebrasProvider(options ...CerebrasProviderSettings) *CerebrasProvider {
	var settings CerebrasProviderSettings
	if len(options) > 0 {
		settings = options[0]
	}

	baseURL := "https://api.cerebras.ai/v1"
	if settings.BaseURL != nil {
		baseURL = providerutils.WithoutTrailingSlash(*settings.BaseURL)
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "CEREBRAS_API_KEY",
			Description:             "Cerebras API key",
		})
		if err != nil {
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
			fmt.Sprintf("ai-sdk/cerebras/%s", VERSION),
		)
	}

	return &CerebrasProvider{
		settings:   settings,
		baseURL:    baseURL,
		getHeaders: getHeaders,
	}
}

// SpecificationVersion returns "v3".
func (p *CerebrasProvider) SpecificationVersion() string { return "v3" }

// LanguageModel creates a Cerebras language model for text generation.
func (p *CerebrasProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createLanguageModel(modelID), nil
}

// Chat creates a Cerebras chat model for text generation.
func (p *CerebrasProvider) Chat(modelID CerebrasChatModelID) *openaicompatible.ChatLanguageModel {
	return p.createLanguageModel(modelID)
}

// createLanguageModel creates a new OpenAICompatibleChatLanguageModel for Cerebras.
func (p *CerebrasProvider) createLanguageModel(modelID CerebrasChatModelID) *openaicompatible.ChatLanguageModel {
	supportsStructured := true
	return openaicompatible.NewChatLanguageModel(modelID, openaicompatible.ChatConfig{
		Provider: "cerebras.chat",
		URL: func(path string) string {
			return fmt.Sprintf("%s%s", p.baseURL, path)
		},
		Headers:                   p.getHeaders,
		Fetch:                     p.settings.Fetch,
		ErrorStructure:            cerebrasErrorStructure,
		SupportsStructuredOutputs: &supportsStructured,
	})
}

// EmbeddingModel is not supported by Cerebras.
func (p *CerebrasProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeEmbedding,
	})
}

// ImageModel is not supported by Cerebras.
func (p *CerebrasProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}

// TranscriptionModel is not supported by Cerebras.
func (p *CerebrasProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel is not supported by Cerebras.
func (p *CerebrasProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel is not supported by Cerebras.
func (p *CerebrasProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
