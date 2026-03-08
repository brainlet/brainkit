// Ported from: packages/openai/src/openai-provider.ts
package openai

import (
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIProviderSettings contains settings for creating an OpenAI provider.
type OpenAIProviderSettings struct {
	// BaseURL for the OpenAI API calls.
	BaseURL *string

	// APIKey for authenticating requests.
	APIKey *string

	// Organization is the OpenAI Organization.
	Organization *string

	// Project is the OpenAI project.
	Project *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Name overrides the "openai" default provider name for 3rd party providers.
	Name *string

	// Fetch is a custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// OpenAIProvider provides access to OpenAI models.
type OpenAIProvider struct {
	settings OpenAIProviderSettings
	baseURL  string
	name     string
	headers  func() map[string]string

	// Tools provides OpenAI-specific tool constructors.
	Tools OpenAITools
}

// SpecificationVersion returns "v3".
func (p *OpenAIProvider) SpecificationVersion() string { return "v3" }

// LanguageModel creates a language model (defaults to Responses API).
func (p *OpenAIProvider) LanguageModel(modelID string) languagemodel.LanguageModel {
	return p.Responses(modelID)
}

// Chat creates an OpenAI chat model for text generation.
func (p *OpenAIProvider) Chat(modelID string) *OpenAIChatLanguageModel {
	return NewOpenAIChatLanguageModel(modelID, OpenAIChatConfig{
		Provider: fmt.Sprintf("%s.chat", p.name),
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return p.baseURL + options.Path
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	})
}

// Responses creates an OpenAI Responses API model for text generation.
func (p *OpenAIProvider) Responses(modelID string) *OpenAIResponsesLanguageModel {
	return NewOpenAIResponsesLanguageModel(modelID, OpenAIConfig{
		Provider: fmt.Sprintf("%s.responses", p.name),
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return p.baseURL + options.Path
		},
		Headers:        p.headers,
		Fetch:          p.settings.Fetch,
		FileIDPrefixes: []string{"file-"},
	})
}

// Completion creates an OpenAI completion model for text generation.
func (p *OpenAIProvider) Completion(modelID string) *OpenAICompletionLanguageModel {
	return NewOpenAICompletionLanguageModel(modelID, OpenAICompletionConfig{
		Provider: fmt.Sprintf("%s.completion", p.name),
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return p.baseURL + options.Path
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	})
}

// Embedding creates an OpenAI embedding model.
func (p *OpenAIProvider) Embedding(modelID string) *OpenAIEmbeddingModel {
	return NewOpenAIEmbeddingModel(modelID, OpenAIConfig{
		Provider: fmt.Sprintf("%s.embedding", p.name),
		URL: func(options struct {
			ModelID string
			Path    string
		}) string {
			return p.baseURL + options.Path
		},
		Headers: p.headers,
		Fetch:   p.settings.Fetch,
	})
}

// Image creates an OpenAI image model.
func (p *OpenAIProvider) Image(modelID string) *OpenAIImageModel {
	return NewOpenAIImageModel(modelID, OpenAIImageModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: fmt.Sprintf("%s.image", p.name),
			URL: func(options struct {
				ModelID string
				Path    string
			}) string {
				return p.baseURL + options.Path
			},
			Headers: p.headers,
			Fetch:   p.settings.Fetch,
		},
	})
}

// Transcription creates an OpenAI transcription model.
func (p *OpenAIProvider) Transcription(modelID string) *OpenAITranscriptionModel {
	return NewOpenAITranscriptionModel(modelID, OpenAITranscriptionModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: fmt.Sprintf("%s.transcription", p.name),
			URL: func(options struct {
				ModelID string
				Path    string
			}) string {
				return p.baseURL + options.Path
			},
			Headers: p.headers,
			Fetch:   p.settings.Fetch,
		},
	})
}

// Speech creates an OpenAI speech model.
func (p *OpenAIProvider) Speech(modelID string) *OpenAISpeechModel {
	return NewOpenAISpeechModel(modelID, OpenAISpeechModelConfig{
		OpenAIConfig: OpenAIConfig{
			Provider: fmt.Sprintf("%s.speech", p.name),
			URL: func(options struct {
				ModelID string
				Path    string
			}) string {
				return p.baseURL + options.Path
			},
			Headers: p.headers,
			Fetch:   p.settings.Fetch,
		},
	})
}

// CreateOpenAI creates a new OpenAI provider instance.
func CreateOpenAI(options *OpenAIProviderSettings) *OpenAIProvider {
	if options == nil {
		options = &OpenAIProviderSettings{}
	}

	baseURLSetting := providerutils.LoadOptionalSetting(providerutils.LoadOptionalSettingOptions{
		SettingValue:            options.BaseURL,
		EnvironmentVariableName: "OPENAI_BASE_URL",
	})
	baseURL := "https://api.openai.com/v1"
	if baseURLSetting != nil {
		baseURL = strings.TrimRight(*baseURLSetting, "/")
	}

	providerName := "openai"
	if options.Name != nil {
		providerName = *options.Name
	}

	getHeaders := func() map[string]string {
		apiKey, _ := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  options.APIKey,
			EnvironmentVariableName: "OPENAI_API_KEY",
			Description:             "OpenAI",
		})

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}

		if options.Organization != nil && *options.Organization != "" {
			headers["OpenAI-Organization"] = *options.Organization
		}
		if options.Project != nil && *options.Project != "" {
			headers["OpenAI-Project"] = *options.Project
		}

		for k, v := range options.Headers {
			headers[k] = v
		}

		return providerutils.WithUserAgentSuffix(headers, fmt.Sprintf("ai-sdk/openai/%s", VERSION))
	}

	return &OpenAIProvider{
		settings: *options,
		baseURL:  baseURL,
		name:     providerName,
		headers:  getHeaders,
		Tools:    NewOpenAITools(),
	}
}

// Note: VERSION is declared in version.go
