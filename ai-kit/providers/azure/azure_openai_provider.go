// Ported from: packages/azure/src/azure-openai-provider.ts
package azure

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
	"github.com/brainlet/brainkit/ai-kit/providers/openai"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// --- Provider settings ---

// AzureOpenAIProviderSettings configures the Azure OpenAI provider.
type AzureOpenAIProviderSettings struct {
	// ResourceName is the name of the Azure OpenAI resource. Either this or
	// BaseURL can be used.
	// The resource name is used in the assembled URL:
	//   https://{resourceName}.openai.azure.com/openai/v1{path}
	ResourceName *string

	// BaseURL is a different URL prefix for API calls, e.g. to use proxy servers.
	// Either this or ResourceName can be used. When a BaseURL is provided, the
	// ResourceName is ignored.
	// With a BaseURL, the resolved URL is: {baseURL}/v1{path}
	BaseURL *string

	// APIKey is the Azure OpenAI API key.
	APIKey *string

	// Headers are custom headers to include in the requests.
	Headers map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction

	// APIVersion is a custom api version to use. Defaults to "v1".
	APIVersion *string

	// UseDeploymentBasedUrls uses deployment-based URLs for specific model types.
	// Set to true to use legacy deployment format:
	//   {baseURL}/deployments/{deploymentId}{path}?api-version={apiVersion}
	// instead of:
	//   {baseURL}/v1{path}?api-version={apiVersion}
	UseDeploymentBasedUrls bool
}

// AzureOpenAIProvider implements provider.Provider for Azure OpenAI.
type AzureOpenAIProvider struct {
	settings   AzureOpenAIProviderSettings
	getHeaders func() map[string]string
	urlFn      func(options struct {
		ModelID string
		Path    string
	}) string
	Tools struct {
		CodeInterpreter  func(args *openai.CodeInterpreterArgs) map[string]interface{}
		FileSearch       func(args openai.FileSearchArgs) map[string]interface{}
		ImageGeneration  func(args *openai.ImageGenerationArgs) map[string]interface{}
		WebSearchPreview func(args *openai.WebSearchPreviewArgs) map[string]interface{}
	}
}

// NewAzureOpenAIProvider creates a new Azure OpenAI provider instance.
// Equivalent to createAzure() in the TS SDK.
func NewAzureOpenAIProvider(options ...AzureOpenAIProviderSettings) *AzureOpenAIProvider {
	var settings AzureOpenAIProviderSettings
	if len(options) > 0 {
		settings = options[0]
	}

	getHeaders := func() map[string]string {
		apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
			ApiKey:                  settings.APIKey,
			EnvironmentVariableName: "AZURE_API_KEY",
			Description:             "Azure OpenAI",
		})
		if err != nil {
			panic(err)
		}

		headers := map[string]string{
			"api-key": apiKey,
		}
		for k, v := range settings.Headers {
			headers[k] = v
		}
		return providerutils.WithUserAgentSuffix(
			headers,
			fmt.Sprintf("ai-sdk/azure/%s", VERSION),
		)
	}

	getResourceName := func() string {
		name, err := providerutils.LoadSetting(providerutils.LoadSettingOptions{
			SettingValue:            settings.ResourceName,
			SettingName:             "resourceName",
			EnvironmentVariableName: "AZURE_RESOURCE_NAME",
			Description:             "Azure OpenAI resource name",
		})
		if err != nil {
			panic(err)
		}
		return name
	}

	apiVersion := "v1"
	if settings.APIVersion != nil {
		apiVersion = *settings.APIVersion
	}

	urlFn := func(options struct {
		ModelID string
		Path    string
	}) string {
		var baseURLPrefix string
		if settings.BaseURL != nil {
			baseURLPrefix = *settings.BaseURL
		} else {
			baseURLPrefix = fmt.Sprintf("https://%s.openai.azure.com/openai", getResourceName())
		}

		var rawURL string
		if settings.UseDeploymentBasedUrls {
			// Use deployment-based format for compatibility with certain Azure OpenAI models
			rawURL = fmt.Sprintf("%s/deployments/%s%s", baseURLPrefix, options.ModelID, options.Path)
		} else {
			// Use v1 API format - no deployment ID in URL
			rawURL = fmt.Sprintf("%s/v1%s", baseURLPrefix, options.Path)
		}

		u, err := url.Parse(rawURL)
		if err != nil {
			// If URL parsing fails, return the raw URL with query param appended
			return fmt.Sprintf("%s?api-version=%s", rawURL, apiVersion)
		}
		q := u.Query()
		q.Set("api-version", apiVersion)
		u.RawQuery = q.Encode()
		return u.String()
	}

	p := &AzureOpenAIProvider{
		settings:   settings,
		getHeaders: getHeaders,
		urlFn:      urlFn,
	}
	p.Tools.CodeInterpreter = openai.NewCodeInterpreterTool
	p.Tools.FileSearch = openai.NewFileSearchTool
	p.Tools.ImageGeneration = openai.NewImageGenerationTool
	p.Tools.WebSearchPreview = openai.NewWebSearchPreviewTool

	return p
}

// SpecificationVersion returns "v3".
func (p *AzureOpenAIProvider) SpecificationVersion() string { return "v3" }

// LanguageModel creates an Azure OpenAI responses API model for text generation.
// This is the default language model method per the TS SDK.
func (p *AzureOpenAIProvider) LanguageModel(deploymentID string) (languagemodel.LanguageModel, error) {
	return p.Responses(deploymentID), nil
}

// Chat creates an Azure OpenAI chat model for text generation.
func (p *AzureOpenAIProvider) Chat(deploymentID string) *openai.OpenAIChatLanguageModel {
	return openai.NewOpenAIChatLanguageModel(deploymentID, openai.OpenAIChatConfig{
		Provider: "azure.chat",
		URL:      p.urlFn,
		Headers:  p.getHeaders,
		Fetch:    p.settings.Fetch,
	})
}

// Completion creates an Azure OpenAI completion model for text generation.
func (p *AzureOpenAIProvider) Completion(deploymentID string) *openai.OpenAICompletionLanguageModel {
	return openai.NewOpenAICompletionLanguageModel(deploymentID, openai.OpenAICompletionConfig{
		Provider: "azure.completion",
		URL:      p.urlFn,
		Headers:  p.getHeaders,
		Fetch:    p.settings.Fetch,
	})
}

// Responses creates an Azure OpenAI responses API model for text generation.
func (p *AzureOpenAIProvider) Responses(deploymentID string) *openai.OpenAIResponsesLanguageModel {
	return openai.NewOpenAIResponsesLanguageModel(deploymentID, openai.OpenAIConfig{
		Provider: "azure.responses",
		URL:      p.urlFn,
		Headers:  p.getHeaders,
		Fetch:    p.settings.Fetch,
		FileIDPrefixes: []string{"assistant-"},
	})
}

// Embedding creates an Azure OpenAI model for text embeddings.
func (p *AzureOpenAIProvider) Embedding(deploymentID string) *openai.OpenAIEmbeddingModel {
	return openai.NewOpenAIEmbeddingModel(deploymentID, openai.OpenAIConfig{
		Provider: "azure.embeddings",
		Headers:  p.getHeaders,
		URL:      p.urlFn,
		Fetch:    p.settings.Fetch,
	})
}

// EmbeddingModel creates an Azure OpenAI model for text embeddings.
// Implements provider.Provider.
func (p *AzureOpenAIProvider) EmbeddingModel(deploymentID string) (embeddingmodel.EmbeddingModel, error) {
	return p.Embedding(deploymentID), nil
}

// Image creates an Azure OpenAI DALL-E model for image generation.
func (p *AzureOpenAIProvider) Image(deploymentID string) *openai.OpenAIImageModel {
	return openai.NewOpenAIImageModel(deploymentID, openai.OpenAIImageModelConfig{
		OpenAIConfig: openai.OpenAIConfig{
			Provider: "azure.image",
			URL:      p.urlFn,
			Headers:  p.getHeaders,
			Fetch:    p.settings.Fetch,
		},
	})
}

// ImageModel creates an Azure OpenAI DALL-E model for image generation.
// Implements provider.Provider.
func (p *AzureOpenAIProvider) ImageModel(deploymentID string) (imagemodel.ImageModel, error) {
	return p.Image(deploymentID), nil
}

// Transcription creates an Azure OpenAI model for audio transcription.
func (p *AzureOpenAIProvider) Transcription(deploymentID string) *openai.OpenAITranscriptionModel {
	return openai.NewOpenAITranscriptionModel(deploymentID, openai.OpenAITranscriptionModelConfig{
		OpenAIConfig: openai.OpenAIConfig{
			Provider: "azure.transcription",
			URL:      p.urlFn,
			Headers:  p.getHeaders,
			Fetch:    p.settings.Fetch,
		},
	})
}

// TranscriptionModel creates an Azure OpenAI model for audio transcription.
// Implements provider.Provider.
func (p *AzureOpenAIProvider) TranscriptionModel(deploymentID string) (transcriptionmodel.TranscriptionModel, error) {
	return p.Transcription(deploymentID), nil
}

// Speech creates an Azure OpenAI model for speech generation.
func (p *AzureOpenAIProvider) Speech(deploymentID string) *openai.OpenAISpeechModel {
	return openai.NewOpenAISpeechModel(deploymentID, openai.OpenAISpeechModelConfig{
		OpenAIConfig: openai.OpenAIConfig{
			Provider: "azure.speech",
			URL:      p.urlFn,
			Headers:  p.getHeaders,
			Fetch:    p.settings.Fetch,
		},
	})
}

// SpeechModel creates an Azure OpenAI model for speech generation.
// Implements provider.Provider.
func (p *AzureOpenAIProvider) SpeechModel(deploymentID string) (speechmodel.SpeechModel, error) {
	return p.Speech(deploymentID), nil
}

// RerankingModel is not supported by Azure OpenAI.
func (p *AzureOpenAIProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}
