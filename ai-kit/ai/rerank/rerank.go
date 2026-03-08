// Ported from: packages/ai/src/rerank/rerank.ts
package rerank

import (
	"context"
	"time"
)

// Warning from the model provider for this call.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type Warning struct {
	Type    string `json:"type"`
	Feature string `json:"feature,omitempty"`
	Details string `json:"details,omitempty"`
	Message string `json:"message,omitempty"`
}

// RerankingModel is the interface for reranking models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type RerankingModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoRerank performs the reranking operation.
	DoRerank(ctx context.Context, opts DoRerankOptions) (*DoRerankResult, error)
}

// DocumentsInput represents the documents to rerank, with type detection.
type DocumentsInput struct {
	// Type is either "text" or "object".
	Type string
	// TextValues are the documents as strings (when Type is "text").
	TextValues []string
	// ObjectValues are the documents as maps (when Type is "object").
	ObjectValues []map[string]any
}

// DoRerankOptions are the options passed to RerankingModel.DoRerank.
type DoRerankOptions struct {
	Documents       DocumentsInput
	Query           string
	TopN            *int
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// ModelRanking represents a single ranking entry from the model.
type ModelRanking struct {
	Index          int     `json:"index"`
	RelevanceScore float64 `json:"relevanceScore"`
}

// DoRerankResult is the result from RerankingModel.DoRerank.
type DoRerankResult struct {
	Ranking          []ModelRanking
	ProviderMetadata ProviderMetadata
	Response         *DoRerankResponseData
	Warnings         []Warning
}

// DoRerankResponseData holds response data from the reranking provider.
type DoRerankResponseData struct {
	ID      string
	Headers map[string]string
	Body    any
	ModelID string
}

// RerankOptions are the options for the Rerank function.
type RerankOptions struct {
	// Model is the reranking model to use.
	Model RerankingModel

	// Documents are the documents that should be reranked.
	Documents []any

	// Query is the query to rerank the documents against.
	Query string

	// TopN is the number of top documents to return.
	TopN *int

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any
}

// Rerank reranks documents using a reranking model.
func Rerank(ctx context.Context, opts RerankOptions) (*RerankResult, error) {
	model := opts.Model

	if len(opts.Documents) == 0 {
		return &RerankResult{
			OriginalDocuments: []any{},
			Ranking:           []RankedDocument{},
			Response: ResponseData{
				Timestamp: time.Now(),
				ModelID:   model.ModelID(),
			},
		}, nil
	}

	// Detect the type of the documents.
	var docsInput DocumentsInput
	if _, ok := opts.Documents[0].(string); ok {
		textValues := make([]string, len(opts.Documents))
		for i, doc := range opts.Documents {
			textValues[i] = doc.(string)
		}
		docsInput = DocumentsInput{
			Type:       "text",
			TextValues: textValues,
		}
	} else {
		objectValues := make([]map[string]any, len(opts.Documents))
		for i, doc := range opts.Documents {
			if m, ok := doc.(map[string]any); ok {
				objectValues[i] = m
			}
		}
		docsInput = DocumentsInput{
			Type:         "object",
			ObjectValues: objectValues,
		}
	}

	result, err := model.DoRerank(ctx, DoRerankOptions{
		Documents:       docsInput,
		Query:           opts.Query,
		TopN:            opts.TopN,
		Headers:         opts.Headers,
		ProviderOptions: opts.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	ranking := make([]RankedDocument, len(result.Ranking))
	for i, r := range result.Ranking {
		ranking[i] = RankedDocument{
			OriginalIndex: r.Index,
			Score:         r.RelevanceScore,
			Document:      opts.Documents[r.Index],
		}
	}

	var responseTimestamp time.Time
	var responseModelID string
	var responseID string
	var responseHeaders map[string]string
	var responseBody any

	if result.Response != nil {
		responseTimestamp = time.Now()
		responseModelID = result.Response.ModelID
		if responseModelID == "" {
			responseModelID = model.ModelID()
		}
		responseID = result.Response.ID
		responseHeaders = result.Response.Headers
		responseBody = result.Response.Body
	} else {
		responseTimestamp = time.Now()
		responseModelID = model.ModelID()
	}

	return &RerankResult{
		OriginalDocuments: opts.Documents,
		Ranking:           ranking,
		ProviderMetadata:  result.ProviderMetadata,
		Response: ResponseData{
			ID:        responseID,
			Timestamp: responseTimestamp,
			ModelID:   responseModelID,
			Headers:   responseHeaders,
			Body:      responseBody,
		},
	}, nil
}
