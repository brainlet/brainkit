// Ported from: packages/google/src/google-generative-ai-embedding-model.ts
package google

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleEmbeddingModelConfig configures a GoogleEmbeddingModel.
type GoogleEmbeddingModelConfig struct {
	Provider string
	BaseURL  string
	Headers  func() map[string]string
	Fetch    providerutils.FetchFunction
}

// GoogleEmbeddingModel implements embeddingmodel.EmbeddingModel for the Google
// Generative AI API.
type GoogleEmbeddingModel struct {
	modelID string
	config  GoogleEmbeddingModelConfig
}

// NewGoogleEmbeddingModel creates a new GoogleEmbeddingModel.
func NewGoogleEmbeddingModel(modelID string, config GoogleEmbeddingModelConfig) *GoogleEmbeddingModel {
	return &GoogleEmbeddingModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns the embedding model interface version.
func (m *GoogleEmbeddingModel) SpecificationVersion() string {
	return "v3"
}

// Provider returns the provider ID.
func (m *GoogleEmbeddingModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *GoogleEmbeddingModel) ModelID() string {
	return m.modelID
}

// MaxEmbeddingsPerCall returns the maximum embeddings per call.
func (m *GoogleEmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	v := 2048
	return &v, nil
}

// SupportsParallelCalls returns true since Google supports parallel calls.
func (m *GoogleEmbeddingModel) SupportsParallelCalls() (bool, error) {
	return true, nil
}

// DoEmbed generates embeddings for the given input values.
func (m *GoogleEmbeddingModel) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
	// Parse provider options
	googleOptions, err := providerutils.ParseProviderOptions(
		"google",
		toInterfaceMap(options.ProviderOptions),
		GoogleEmbeddingModelOptionsSchema,
	)
	if err != nil {
		return embeddingmodel.Result{}, err
	}

	maxEmbeddings := 2048
	if len(options.Values) > maxEmbeddings {
		values := make([]any, len(options.Values))
		for i, v := range options.Values {
			values[i] = v
		}
		return embeddingmodel.Result{}, errors.NewTooManyEmbeddingValuesForCallError(
			m.Provider(), m.modelID, maxEmbeddings, values,
		)
	}

	headers := m.config.Headers()
	mergedHeaders := providerutils.CombineHeaders(headers, options.Headers)

	// For single embeddings, use the single endpoint (rate limits, etc.)
	if len(options.Values) == 1 {
		body := map[string]any{
			"model": fmt.Sprintf("models/%s", m.modelID),
			"content": map[string]any{
				"parts": []map[string]any{
					{"text": options.Values[0]},
				},
			},
		}
		if googleOptions != nil && googleOptions.OutputDimensionality != nil {
			body["outputDimensionality"] = *googleOptions.OutputDimensionality
		}
		if googleOptions != nil && googleOptions.TaskType != nil {
			body["taskType"] = *googleOptions.TaskType
		}

		result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[googleSingleEmbeddingResponse]{
			URL:                       fmt.Sprintf("%s/models/%s:embedContent", m.config.BaseURL, m.modelID),
			Headers:                   mergedHeaders,
			Body:                      body,
			FailedResponseHandler:     GoogleFailedResponseHandler,
			SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleSingleEmbeddingResponse](nil),
			Ctx:                       options.Ctx,
			Fetch:                     m.config.Fetch,
		})
		if err != nil {
			return embeddingmodel.Result{}, err
		}

		return embeddingmodel.Result{
			Warnings:   []shared.Warning{},
			Embeddings: []embeddingmodel.Embedding{result.Value.Embedding.Values},
			Response: &embeddingmodel.ResultResponse{
				Headers: result.ResponseHeaders,
				Body:    result.RawValue,
			},
		}, nil
	}

	// Batch endpoint
	requests := make([]map[string]any, len(options.Values))
	for i, value := range options.Values {
		req := map[string]any{
			"model": fmt.Sprintf("models/%s", m.modelID),
			"content": map[string]any{
				"role": "user",
				"parts": []map[string]any{
					{"text": value},
				},
			},
		}
		if googleOptions != nil && googleOptions.OutputDimensionality != nil {
			req["outputDimensionality"] = *googleOptions.OutputDimensionality
		}
		if googleOptions != nil && googleOptions.TaskType != nil {
			req["taskType"] = *googleOptions.TaskType
		}
		requests[i] = req
	}

	body := map[string]any{
		"requests": requests,
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[googleBatchEmbeddingResponse]{
		URL:                       fmt.Sprintf("%s/models/%s:batchEmbedContents", m.config.BaseURL, m.modelID),
		Headers:                   mergedHeaders,
		Body:                      body,
		FailedResponseHandler:     GoogleFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleBatchEmbeddingResponse](nil),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return embeddingmodel.Result{}, err
	}

	embeddings := make([]embeddingmodel.Embedding, len(result.Value.Embeddings))
	for i, item := range result.Value.Embeddings {
		embeddings[i] = item.Values
	}

	return embeddingmodel.Result{
		Warnings:   []shared.Warning{},
		Embeddings: embeddings,
		Response: &embeddingmodel.ResultResponse{
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
	}, nil
}

// --- Response types ---

type googleSingleEmbeddingResponse struct {
	Embedding googleEmbeddingValues `json:"embedding"`
}

type googleBatchEmbeddingResponse struct {
	Embeddings []googleEmbeddingValues `json:"embeddings"`
}

type googleEmbeddingValues struct {
	Values []float64 `json:"values"`
}
