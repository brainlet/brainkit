// Ported from: packages/mistral/src/mistral-embedding-model.ts
package mistral

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// EmbeddingConfig holds the configuration for a Mistral embedding model.
type EmbeddingConfig struct {
	// Provider is the provider identifier (e.g. "mistral.embedding").
	Provider string

	// BaseURL is the base URL for the Mistral API.
	BaseURL string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction
}

// EmbeddingModel implements embeddingmodel.EmbeddingModel for the Mistral API.
type EmbeddingModel struct {
	modelID MistralEmbeddingModelId
	config  EmbeddingConfig
}

// NewEmbeddingModel creates a new EmbeddingModel.
func NewEmbeddingModel(modelID MistralEmbeddingModelId, config EmbeddingConfig) *EmbeddingModel {
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

// MaxEmbeddingsPerCall returns 32, the maximum embeddings per call for Mistral.
func (m *EmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	v := 32
	return &v, nil
}

// SupportsParallelCalls returns false for Mistral embedding models.
func (m *EmbeddingModel) SupportsParallelCalls() (bool, error) {
	return false, nil
}

// DoEmbed generates embeddings for the given values.
func (m *EmbeddingModel) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
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

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[mistralEmbeddingResponse]{
		URL:                       fmt.Sprintf("%s/embeddings", m.config.BaseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), options.Headers),
		Body:                      body,
		FailedResponseHandler:     mistralFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(mistralEmbeddingResponseSchema),
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

// --- Response schemas ---

type mistralEmbeddingResponse struct {
	Data  []mistralEmbeddingDataItem `json:"data"`
	Usage *mistralEmbeddingUsage     `json:"usage,omitempty"`
}

type mistralEmbeddingDataItem struct {
	Embedding []float64 `json:"embedding"`
}

type mistralEmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
}

var mistralEmbeddingResponseSchema = &providerutils.Schema[mistralEmbeddingResponse]{}

// Verify EmbeddingModel implements the EmbeddingModel interface.
var _ embeddingmodel.EmbeddingModel = (*EmbeddingModel)(nil)
