// Ported from: packages/openai-compatible/src/embedding/openai-compatible-embedding-model.ts
package openaicompatible

import (
	"encoding/json"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// EmbeddingConfig holds the configuration for an embedding model.
type EmbeddingConfig struct {
	// MaxEmbeddingsPerCall overrides the maximum number of embeddings per call.
	MaxEmbeddingsPerCall *int

	// SupportsParallelCalls overrides the parallelism of embedding calls.
	SupportsParallelCalls *bool

	// Provider is the provider identifier (e.g. "openai.embedding").
	Provider string

	// URL builds the full API URL from the given path.
	URL func(path string) string

	// Headers returns the HTTP headers to send with each request.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// ErrorStructure is the provider-specific error structure.
	// If nil, DefaultErrorStructure is used.
	ErrorStructure *ProviderErrorStructure[ErrorData]
}

// EmbeddingModel implements embeddingmodel.EmbeddingModel for
// OpenAI-compatible embedding endpoints.
type EmbeddingModel struct {
	modelID EmbeddingModelID
	config  EmbeddingConfig
}

// NewEmbeddingModel creates a new EmbeddingModel.
func NewEmbeddingModel(modelID string, config EmbeddingConfig) *EmbeddingModel {
	return &EmbeddingModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *EmbeddingModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *EmbeddingModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *EmbeddingModel) ModelID() string { return m.modelID }

// MaxEmbeddingsPerCall returns the maximum number of embeddings per call.
// Defaults to 2048.
func (m *EmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	if m.config.MaxEmbeddingsPerCall != nil {
		return m.config.MaxEmbeddingsPerCall, nil
	}
	v := 2048
	return &v, nil
}

// SupportsParallelCalls returns whether the model supports parallel calls.
// Defaults to true.
func (m *EmbeddingModel) SupportsParallelCalls() (bool, error) {
	if m.config.SupportsParallelCalls != nil {
		return *m.config.SupportsParallelCalls, nil
	}
	return true, nil
}

func (m *EmbeddingModel) providerOptionsName() string {
	return strings.TrimSpace(strings.SplitN(m.config.Provider, ".", 2)[0])
}

// DoEmbed generates embeddings for the given values.
func (m *EmbeddingModel) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
	var warnings []shared.Warning

	providerOpts := providerOptionsToMap(options.ProviderOptions)

	// Parse provider options - check for deprecated 'openai-compatible' key
	deprecatedOptions, err := providerutils.ParseProviderOptions(
		"openai-compatible",
		providerOpts,
		EmbeddingOptionsSchema,
	)
	if err != nil {
		return embeddingmodel.Result{}, err
	}

	if deprecatedOptions != nil {
		warnings = append(warnings, shared.OtherWarning{
			Message: "The 'openai-compatible' key in providerOptions is deprecated. Use 'openaiCompatible' instead.",
		})
	}

	// Merge options from multiple provider keys: deprecated, camelCase, and provider-specific
	compatibleOptions := mergeEmbeddingOptions(deprecatedOptions)

	camelCaseOptions, err := providerutils.ParseProviderOptions(
		"openaiCompatible",
		providerOpts,
		EmbeddingOptionsSchema,
	)
	if err != nil {
		return embeddingmodel.Result{}, err
	}
	compatibleOptions = mergeEmbeddingOptions2(compatibleOptions, camelCaseOptions)

	providerSpecificOptions, err := providerutils.ParseProviderOptions(
		m.providerOptionsName(),
		providerOpts,
		EmbeddingOptionsSchema,
	)
	if err != nil {
		return embeddingmodel.Result{}, err
	}
	compatibleOptions = mergeEmbeddingOptions2(compatibleOptions, providerSpecificOptions)

	// Check max embeddings per call
	maxPerCall, _ := m.MaxEmbeddingsPerCall()
	if maxPerCall != nil && len(options.Values) > *maxPerCall {
		valuesAsAny := make([]any, len(options.Values))
		for i, v := range options.Values {
			valuesAsAny[i] = v
		}
		return embeddingmodel.Result{}, errors.NewTooManyEmbeddingValuesForCallError(
			m.Provider(),
			m.ModelID(),
			*maxPerCall,
			valuesAsAny,
		)
	}

	body := map[string]interface{}{
		"model":           m.modelID,
		"input":           options.Values,
		"encoding_format": "float",
	}
	if compatibleOptions.Dimensions != nil {
		body["dimensions"] = *compatibleOptions.Dimensions
	}
	if compatibleOptions.User != nil {
		body["user"] = *compatibleOptions.User
	}

	errorStructure := m.config.ErrorStructure
	if errorStructure == nil {
		es := DefaultErrorStructure
		errorStructure = &es
	}

	failedResponseHandler := providerutils.CreateJsonErrorResponseHandler(
		errorStructure.ErrorSchema,
		errorStructure.ErrorToMessage,
		errorStructure.IsRetryable,
	)
	wrappedHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, handlerErr := failedResponseHandler(opts)
		if handlerErr != nil {
			return nil, handlerErr
		}
		if result == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           result.Value,
			RawValue:        result.RawValue,
			ResponseHeaders: result.ResponseHeaders,
		}, nil
	})

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[embeddingResponse]{
		URL:                       m.config.URL("/embeddings"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), options.Headers),
		Body:                      body,
		FailedResponseHandler:     wrappedHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(embeddingResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return embeddingmodel.Result{}, err
	}

	response := result.Value

	embeddings := make([]embeddingmodel.Embedding, len(response.Data))
	for i, item := range response.Data {
		embeddings[i] = item.Embedding
	}

	var usage *embeddingmodel.EmbeddingUsage
	if response.Usage != nil {
		usage = &embeddingmodel.EmbeddingUsage{
			Tokens: response.Usage.PromptTokens,
		}
	}

	return embeddingmodel.Result{
		Warnings:         warnings,
		Embeddings:       embeddings,
		Usage:            usage,
		ProviderMetadata: response.ProviderMetadata,
		Response: &embeddingmodel.ResultResponse{
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
	}, nil
}

// --- Response schemas ---

type embeddingResponse struct {
	Data             []embeddingDataItem    `json:"data"`
	Usage            *embeddingUsageRaw     `json:"usage,omitempty"`
	ProviderMetadata shared.ProviderMetadata `json:"providerMetadata,omitempty"`
}

type embeddingDataItem struct {
	Embedding []float64 `json:"embedding"`
}

type embeddingUsageRaw struct {
	PromptTokens int `json:"prompt_tokens"`
}

var embeddingResponseSchema = &providerutils.Schema[embeddingResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[embeddingResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[embeddingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp embeddingResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[embeddingResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[embeddingResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// --- Helper functions ---

func mergeEmbeddingOptions(opts *EmbeddingOptions) EmbeddingOptions {
	if opts == nil {
		return EmbeddingOptions{}
	}
	return *opts
}

func mergeEmbeddingOptions2(base EmbeddingOptions, override *EmbeddingOptions) EmbeddingOptions {
	if override == nil {
		return base
	}
	if override.Dimensions != nil {
		base.Dimensions = override.Dimensions
	}
	if override.User != nil {
		base.User = override.User
	}
	return base
}
