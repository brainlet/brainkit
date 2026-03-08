// Ported from: packages/anthropic/src/anthropic-provider.ts
package anthropic

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
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// AnthropicProviderSettings contains settings for the Anthropic provider.
type AnthropicProviderSettings struct {
	// BaseURL is a different URL prefix for API calls, e.g. to use proxy servers.
	// The default prefix is "https://api.anthropic.com/v1".
	BaseURL *string

	// ApiKey is the API key sent using the "x-api-key" header.
	// It defaults to the ANTHROPIC_API_KEY environment variable.
	// Only one of ApiKey or AuthToken is required.
	ApiKey *string

	// AuthToken is sent using the "Authorization: Bearer" header.
	// It defaults to the ANTHROPIC_AUTH_TOKEN environment variable.
	// Only one of ApiKey or AuthToken is required.
	AuthToken *string

	// Headers contains custom headers to include in the requests.
	Headers map[string]string

	// Fetch is a custom fetch implementation for intercepting requests or testing.
	Fetch providerutils.FetchFunction

	// GenerateID is a custom ID generation function.
	GenerateID func() string

	// Name is a custom provider name. Defaults to "anthropic.messages".
	Name *string
}

// AnthropicProvider implements the Provider interface for Anthropic.
type AnthropicProvider struct {
	baseURL      string
	providerName string
	getHeaders   func() (map[string]string, error)
	fetch        providerutils.FetchFunction
	generateID   func() string

	// Tools exposes the Anthropic provider-defined tool factories.
	Tools *anthropicToolsRegistry
}

// anthropicToolsRegistry is a type alias for the AnthropicTools variable type.
type anthropicToolsRegistry = struct {
	Bash20241022            func(opts providerutils.ProviderToolOptions[Bash20241022Input, interface{}]) providerutils.ProviderTool[Bash20241022Input, interface{}]
	Bash20250124            func(opts providerutils.ProviderToolOptions[Bash20250124Input, interface{}]) providerutils.ProviderTool[Bash20250124Input, interface{}]
	CodeExecution20250522   func(opts providerutils.ProviderToolOptions[CodeExecution20250522Input, CodeExecution20250522Output]) providerutils.ProviderTool[CodeExecution20250522Input, CodeExecution20250522Output]
	CodeExecution20250825   func(opts providerutils.ProviderToolOptions[CodeExecution20250825Input, CodeExecution20250825Output]) providerutils.ProviderTool[CodeExecution20250825Input, CodeExecution20250825Output]
	CodeExecution20260120   func(opts providerutils.ProviderToolOptions[CodeExecution20260120Input, CodeExecution20260120Output]) providerutils.ProviderTool[CodeExecution20260120Input, CodeExecution20260120Output]
	Computer20241022        func(opts providerutils.ProviderToolOptions[Computer20241022Input, interface{}]) providerutils.ProviderTool[Computer20241022Input, interface{}]
	Computer20250124        func(opts providerutils.ProviderToolOptions[Computer20250124Input, interface{}]) providerutils.ProviderTool[Computer20250124Input, interface{}]
	Computer20251124        func(opts providerutils.ProviderToolOptions[Computer20251124Input, interface{}]) providerutils.ProviderTool[Computer20251124Input, interface{}]
	Memory20250818          func(opts providerutils.ProviderToolOptions[Memory20250818Input, interface{}]) providerutils.ProviderTool[Memory20250818Input, interface{}]
	TextEditor20241022      func(opts providerutils.ProviderToolOptions[TextEditor20241022Input, interface{}]) providerutils.ProviderTool[TextEditor20241022Input, interface{}]
	TextEditor20250124      func(opts providerutils.ProviderToolOptions[TextEditor20250124Input, interface{}]) providerutils.ProviderTool[TextEditor20250124Input, interface{}]
	TextEditor20250429      func(opts providerutils.ProviderToolOptions[TextEditor20250429Input, interface{}]) providerutils.ProviderTool[TextEditor20250429Input, interface{}]
	TextEditor20250728      func(opts providerutils.ProviderToolOptions[TextEditor20250728Input, interface{}]) providerutils.ProviderTool[TextEditor20250728Input, interface{}]
	WebFetch20250910        func(opts providerutils.ProviderToolOptions[WebFetch20250910Input, WebFetch20250910Output]) providerutils.ProviderTool[WebFetch20250910Input, WebFetch20250910Output]
	WebFetch20260209        func(opts providerutils.ProviderToolOptions[WebFetch20260209Input, WebFetch20260209Output]) providerutils.ProviderTool[WebFetch20260209Input, WebFetch20260209Output]
	WebSearch20250305       func(opts providerutils.ProviderToolOptions[WebSearch20250305Input, WebSearch20250305Output]) providerutils.ProviderTool[WebSearch20250305Input, WebSearch20250305Output]
	WebSearch20260209       func(opts providerutils.ProviderToolOptions[WebSearch20260209Input, WebSearch20260209Output]) providerutils.ProviderTool[WebSearch20260209Input, WebSearch20260209Output]
	ToolSearchRegex20251119 func(opts providerutils.ProviderToolOptions[ToolSearchRegex20251119Input, ToolSearchRegex20251119Output]) providerutils.ProviderTool[ToolSearchRegex20251119Input, ToolSearchRegex20251119Output]
	ToolSearchBm2520251119  func(opts providerutils.ProviderToolOptions[ToolSearchBm2520251119Input, ToolSearchBm2520251119Output]) providerutils.ProviderTool[ToolSearchBm2520251119Input, ToolSearchBm2520251119Output]
}

// SpecificationVersion returns the provider specification version.
func (p *AnthropicProvider) SpecificationVersion() string {
	return "v3"
}

// LanguageModel creates a language model for text generation.
func (p *AnthropicProvider) LanguageModel(modelID string) (languagemodel.LanguageModel, error) {
	return p.createChatModel(modelID), nil
}

// Chat creates a language model (alias for LanguageModel).
func (p *AnthropicProvider) Chat(modelID string) (languagemodel.LanguageModel, error) {
	return p.createChatModel(modelID), nil
}

// Messages creates a language model (alias for LanguageModel).
func (p *AnthropicProvider) Messages(modelID string) (languagemodel.LanguageModel, error) {
	return p.createChatModel(modelID), nil
}

// EmbeddingModel returns a NoSuchModelError since Anthropic does not support embedding models.
func (p *AnthropicProvider) EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeEmbedding,
	})
}

// ImageModel returns a NoSuchModelError since Anthropic does not support image models.
func (p *AnthropicProvider) ImageModel(modelID string) (imagemodel.ImageModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeImage,
	})
}

// TranscriptionModel returns a NoSuchModelError since Anthropic does not support transcription models.
func (p *AnthropicProvider) TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeTranscription,
	})
}

// SpeechModel returns a NoSuchModelError since Anthropic does not support speech models.
func (p *AnthropicProvider) SpeechModel(modelID string) (speechmodel.SpeechModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeSpeech,
	})
}

// RerankingModel returns a NoSuchModelError since Anthropic does not support reranking models.
func (p *AnthropicProvider) RerankingModel(modelID string) (rerankingmodel.RerankingModel, error) {
	return nil, errors.NewNoSuchModelError(errors.NoSuchModelErrorOptions{
		ModelID:   modelID,
		ModelType: errors.ModelTypeReranking,
	})
}

// createChatModel creates a new AnthropicMessagesLanguageModel instance.
func (p *AnthropicProvider) createChatModel(modelID string) *AnthropicMessagesLanguageModel {
	return NewAnthropicMessagesLanguageModel(modelID, AnthropicMessagesConfig{
		Provider: p.providerName,
		BaseURL:  p.baseURL,
		Headers: func() map[string]string {
			headers, err := p.getHeaders()
			if err != nil {
				// In Go, the TS version would throw from loadApiKey.
				// Here we return empty headers; the error will surface at request time.
				return map[string]string{}
			}
			return headers
		},
		Fetch:      p.fetch,
		GenerateID: p.generateID,
		SupportedURLs: func() map[string][]*regexp.Regexp {
			return map[string][]*regexp.Regexp{
				"image/*":         {regexp.MustCompile(`^https?://.*$`)},
				"application/pdf": {regexp.MustCompile(`^https?://.*$`)},
			}
		},
	})
}

// CreateAnthropic creates an Anthropic provider instance.
func CreateAnthropic(options ...AnthropicProviderSettings) (*AnthropicProvider, error) {
	var opts AnthropicProviderSettings
	if len(options) > 0 {
		opts = options[0]
	}

	// Resolve base URL
	baseURLSetting := providerutils.LoadOptionalSetting(providerutils.LoadOptionalSettingOptions{
		SettingValue:            opts.BaseURL,
		EnvironmentVariableName: "ANTHROPIC_BASE_URL",
	})

	baseURL := "https://api.anthropic.com/v1"
	if baseURLSetting != nil {
		baseURL = providerutils.WithoutTrailingSlash(*baseURLSetting)
	}

	// Resolve provider name
	providerName := "anthropic.messages"
	if opts.Name != nil {
		providerName = *opts.Name
	}

	// Validate that both apiKey and authToken are not provided simultaneously
	if opts.ApiKey != nil && opts.AuthToken != nil {
		return nil, errors.NewInvalidArgumentError(
			"apiKey/authToken",
			"Both apiKey and authToken were provided. Please use only one authentication method.",
			nil,
		)
	}

	// Resolve auth token from env if not provided
	authToken := providerutils.LoadOptionalSetting(providerutils.LoadOptionalSettingOptions{
		SettingValue:            opts.AuthToken,
		EnvironmentVariableName: "ANTHROPIC_AUTH_TOKEN",
	})

	getHeaders := func() (map[string]string, error) {
		authHeaders := map[string]string{}

		if authToken != nil {
			authHeaders["Authorization"] = fmt.Sprintf("Bearer %s", *authToken)
		} else {
			apiKey, err := providerutils.LoadApiKey(providerutils.LoadApiKeyOptions{
				ApiKey:                  opts.ApiKey,
				EnvironmentVariableName: "ANTHROPIC_API_KEY",
				Description:             "Anthropic",
			})
			if err != nil {
				return nil, err
			}
			authHeaders["x-api-key"] = apiKey
		}

		// Merge headers: anthropic-version, auth headers, custom headers
		merged := map[string]string{
			"anthropic-version": "2023-06-01",
		}
		for k, v := range authHeaders {
			merged[k] = v
		}
		for k, v := range opts.Headers {
			merged[k] = v
		}

		return providerutils.WithUserAgentSuffix(
			merged,
			fmt.Sprintf("ai-sdk/anthropic/%s", VERSION),
		), nil
	}

	// Resolve generateID
	generateID := opts.GenerateID
	if generateID == nil {
		generateID = providerutils.GenerateId
	}

	return &AnthropicProvider{
		baseURL:      baseURL,
		providerName: providerName,
		getHeaders:   getHeaders,
		fetch:        opts.Fetch,
		generateID:   generateID,
		Tools:        &AnthropicTools,
	}, nil
}
