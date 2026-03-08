// Ported from: packages/openai/src/embedding/openai-embedding-model.ts
package openai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIEmbeddingModel implements embeddingmodel.EmbeddingModel for the
// OpenAI embedding endpoint.
type OpenAIEmbeddingModel struct {
	modelID OpenAIEmbeddingModelID
	config  OpenAIConfig
}

// NewOpenAIEmbeddingModel creates a new OpenAIEmbeddingModel.
func NewOpenAIEmbeddingModel(modelID OpenAIEmbeddingModelID, config OpenAIConfig) *OpenAIEmbeddingModel {
	return &OpenAIEmbeddingModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAIEmbeddingModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAIEmbeddingModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAIEmbeddingModel) ModelID() string { return m.modelID }

// MaxEmbeddingsPerCall returns 2048.
func (m *OpenAIEmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	v := 2048
	return &v, nil
}

// SupportsParallelCalls returns true.
func (m *OpenAIEmbeddingModel) SupportsParallelCalls() (bool, error) {
	return true, nil
}

// DoEmbed generates embeddings for the given values.
func (m *OpenAIEmbeddingModel) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
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

	// Parse provider options
	providerOpts := providerOptionsToMap(options.ProviderOptions)
	openaiOptions, err := providerutils.ParseProviderOptions(
		"openai",
		providerOpts,
		openaiEmbeddingModelOptionsSchema,
	)
	if err != nil {
		return embeddingmodel.Result{}, err
	}
	if openaiOptions == nil {
		openaiOptions = &OpenAIEmbeddingModelOptions{}
	}

	body := map[string]interface{}{
		"model":           m.modelID,
		"input":           options.Values,
		"encoding_format": "float",
	}
	if openaiOptions.Dimensions != nil {
		body["dimensions"] = *openaiOptions.Dimensions
	}
	if openaiOptions.User != nil {
		body["user"] = *openaiOptions.User
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[openaiTextEmbeddingResponse]{
		URL:                       m.config.URL(struct{ ModelID string; Path string }{ModelID: m.modelID, Path: "/embeddings"}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), options.Headers),
		Body:                      body,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiTextEmbeddingResponseSchema),
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
		Warnings:   []shared.Warning{},
		Embeddings: embeddings,
		Usage:      usage,
		Response: &embeddingmodel.ResultResponse{
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
	}, nil
}
