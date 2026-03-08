// Ported from: packages/xai/src/xai-provider.ts
package xai

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiProviderSettings configures the xAI provider.
type XaiProviderSettings struct {
	// BaseURL is the base URL for the xAI API calls.
	BaseURL string
	// APIKey is the API key for authenticating requests.
	APIKey string
	// Headers are custom headers to include in requests.
	Headers map[string]string
	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction
}

// XaiProvider is the xAI provider for creating language, image, and video models.
type XaiProvider struct {
	specificationVersion string
	baseURL              string
	getHeaders           func() map[string]string
	fetch                providerutils.FetchFunction
	// Tools exposes the server-side agentic tools for use with the responses API.
	Tools XaiTools
}

// XaiTools holds the tool factory functions for xAI.
type XaiTools struct{}

// WebSearch creates a web search provider tool.
func (XaiTools) WebSearch(opts ...providerutils.ProviderToolOptions[WebSearchInput, WebSearchOutput]) providerutils.ProviderTool[WebSearchInput, WebSearchOutput] {
	return WebSearch(opts...)
}

// XSearch creates an X search provider tool.
func (XaiTools) XSearch(opts ...providerutils.ProviderToolOptions[XSearchInput, XSearchOutput]) providerutils.ProviderTool[XSearchInput, XSearchOutput] {
	return XSearch(opts...)
}

// CodeExecution creates a code execution provider tool.
func (XaiTools) CodeExecution(opts ...providerutils.ProviderToolOptions[CodeExecutionInput, CodeExecutionOutput]) providerutils.ProviderTool[CodeExecutionInput, CodeExecutionOutput] {
	return CodeExecution(opts...)
}

// ViewImage creates a view image provider tool.
func (XaiTools) ViewImage(opts ...providerutils.ProviderToolOptions[ViewImageInput, ViewImageOutput]) providerutils.ProviderTool[ViewImageInput, ViewImageOutput] {
	return ViewImage(opts...)
}

// ViewXVideo creates a view X video provider tool.
func (XaiTools) ViewXVideo(opts ...providerutils.ProviderToolOptions[ViewXVideoInput, ViewXVideoOutput]) providerutils.ProviderTool[ViewXVideoInput, ViewXVideoOutput] {
	return ViewXVideo(opts...)
}

// FileSearch creates a file search provider tool.
func (XaiTools) FileSearch(opts providerutils.ProviderToolOptions[FileSearchInput, FileSearchOutput]) providerutils.ProviderTool[FileSearchInput, FileSearchOutput] {
	return FileSearch(opts)
}

// McpServer creates an MCP server provider tool.
func (XaiTools) McpServer(opts providerutils.ProviderToolOptions[McpServerInput, McpServerOutput]) providerutils.ProviderTool[McpServerInput, McpServerOutput] {
	return McpServer(opts)
}

// CreateXai creates a new xAI provider with the given settings.
func CreateXai(options ...XaiProviderSettings) *XaiProvider {
	var opts XaiProviderSettings
	if len(options) > 0 {
		opts = options[0]
	}

	baseURL := providerutils.WithoutTrailingSlash(opts.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	getHeaders := func() map[string]string {
		apiKey := opts.APIKey
		if apiKey == "" {
			var err error
			apiKey, err = providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
				EnvironmentVariableName: "XAI_API_KEY",
				Description:             "xAI API key",
			})
			if err != nil {
				// If we can't load the key, the API calls will fail with auth errors
				apiKey = ""
			}
		}

		headers := map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", apiKey),
		}
		for k, v := range opts.Headers {
			headers[k] = v
		}
		return providerutils.WithUserAgentSuffix(headers, fmt.Sprintf("ai-sdk/xai/%s", VERSION))
	}

	return &XaiProvider{
		specificationVersion: "v3",
		baseURL:              baseURL,
		getHeaders:           getHeaders,
		fetch:                opts.Fetch,
		Tools:                XaiTools{},
	}
}

// SpecificationVersion returns the provider interface version.
func (p *XaiProvider) SpecificationVersion() string {
	return p.specificationVersion
}

// LanguageModel creates an xAI chat language model.
func (p *XaiProvider) LanguageModel(modelId string) (languagemodel.LanguageModel, error) {
	return p.Chat(modelId), nil
}

// Chat creates an xAI chat language model.
func (p *XaiProvider) Chat(modelId XaiChatModelId) languagemodel.LanguageModel {
	return NewXaiChatLanguageModel(modelId, XaiChatConfig{
		Provider:   "xai.chat",
		BaseURL:    p.baseURL,
		Headers:    p.getHeaders,
		GenerateID: providerutils.GenerateId,
		Fetch:      p.fetch,
	})
}

// Responses creates an xAI responses language model.
func (p *XaiProvider) Responses(modelId XaiResponsesModelId) languagemodel.LanguageModel {
	return NewXaiResponsesLanguageModel(modelId, XaiResponsesConfig{
		Provider:   "xai.responses",
		BaseURL:    p.baseURL,
		Headers:    p.getHeaders,
		GenerateID: providerutils.GenerateId,
		Fetch:      p.fetch,
	})
}

// ImageModel creates an xAI image model.
func (p *XaiProvider) ImageModel(modelId XaiImageModelId) (imagemodel.ImageModel, error) {
	return p.Image(modelId), nil
}

// Image creates an xAI image model.
func (p *XaiProvider) Image(modelId XaiImageModelId) imagemodel.ImageModel {
	return NewXaiImageModel(modelId, XaiImageModelConfig{
		Provider: "xai.image",
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.fetch,
	})
}

// VideoModel creates an xAI video model.
func (p *XaiProvider) VideoModel(modelId XaiVideoModelId) videomodel.VideoModel {
	return p.Video(modelId)
}

// Video creates an xAI video model.
func (p *XaiProvider) Video(modelId XaiVideoModelId) videomodel.VideoModel {
	return NewXaiVideoModel(modelId, XaiVideoModelConfig{
		Provider: "xai.video",
		BaseURL:  p.baseURL,
		Headers:  p.getHeaders,
		Fetch:    p.fetch,
	})
}

// EmbeddingModel returns NoSuchModelError (not supported by xAI).
func (p *XaiProvider) EmbeddingModel(modelId string) (interface{}, error) {
	return nil, &errors.NoSuchModelError{
		ModelID:   modelId,
		ModelType: "embeddingModel",
	}
}
