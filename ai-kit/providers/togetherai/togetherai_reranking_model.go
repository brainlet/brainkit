// Ported from: packages/togetherai/src/reranking/togetherai-reranking-model.ts
package togetherai

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// TogetherAIRerankingConfig holds the configuration for the Together AI reranking model.
type TogetherAIRerankingConfig struct {
	Provider string
	BaseURL  string
	Headers  func() map[string]string
	Fetch    providerutils.FetchFunction
}

// TogetherAIRerankingModel implements rerankingmodel.RerankingModel for Together AI.
type TogetherAIRerankingModel struct {
	modelID TogetherAIRerankingModelID
	config  TogetherAIRerankingConfig
}

// NewTogetherAIRerankingModel creates a new TogetherAIRerankingModel.
func NewTogetherAIRerankingModel(modelID TogetherAIRerankingModelID, config TogetherAIRerankingConfig) *TogetherAIRerankingModel {
	return &TogetherAIRerankingModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *TogetherAIRerankingModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *TogetherAIRerankingModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *TogetherAIRerankingModel) ModelID() string { return m.modelID }

// DoRerank reranks documents using the Together AI rerank API.
// see https://docs.together.ai/reference/rerank-1
func (m *TogetherAIRerankingModel) DoRerank(options rerankingmodel.CallOptions) (rerankingmodel.RerankResult, error) {
	// Parse provider-specific options
	providerOptsMap := toInterfaceMap(options.ProviderOptions)
	rerankingOptions, err := providerutils.ParseProviderOptions(
		"togetherai",
		providerOptsMap,
		togetheraiRerankingModelOptionsSchema,
	)
	if err != nil {
		return rerankingmodel.RerankResult{}, err
	}

	// Extract document values for the API
	var documentValues interface{}
	switch docs := options.Documents.(type) {
	case rerankingmodel.DocumentsText:
		documentValues = docs.Values
	case rerankingmodel.DocumentsObject:
		documentValues = docs.Values
	default:
		return rerankingmodel.RerankResult{}, fmt.Errorf("unsupported documents type: %T", options.Documents)
	}

	// Build the request body
	returnDocuments := false
	body := TogetherAIRerankingInput{
		Model:           m.modelID,
		Documents:       documentValues,
		Query:           options.Query,
		TopN:            options.TopN,
		ReturnDocuments: &returnDocuments,
	}

	if rerankingOptions != nil && len(rerankingOptions.RankFields) > 0 {
		body.RankFields = rerankingOptions.RankFields
	}

	// Build the failed response handler
	failedResponseHandler := providerutils.CreateJsonErrorResponseHandler(
		togetheraiErrorSchema,
		func(data togetheraiErrorData) string {
			return data.Error.Message
		},
		nil,
	)
	wrappedFailedHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
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

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[togetheraiRerankingResponse]{
		URL:                       fmt.Sprintf("%s/rerank", m.config.BaseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), options.Headers),
		Body:                      body,
		FailedResponseHandler:     wrappedFailedHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(togetheraiRerankingResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return rerankingmodel.RerankResult{}, err
	}

	response := result.Value
	ranking := make([]rerankingmodel.RankedDocument, len(response.Results))
	for i, r := range response.Results {
		ranking[i] = rerankingmodel.RankedDocument{
			Index:          r.Index,
			RelevanceScore: r.RelevanceScore,
		}
	}

	rerankResult := rerankingmodel.RerankResult{
		Ranking: ranking,
		Response: &rerankingmodel.RerankResultResponse{
			ID:      response.ID,
			ModelID: response.Model,
			Headers: result.ResponseHeaders,
			Body:    result.RawValue,
		},
	}

	return rerankResult, nil
}

// toInterfaceMap converts shared.ProviderOptions (map[string]map[string]any) to
// map[string]interface{} for use with providerutils.ParseProviderOptions.
func toInterfaceMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}
