// Ported from: packages/provider/src/reranking-model/v3/reranking-model-v3-call-options.ts
package rerankingmodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// CallOptions contains the options for a reranking model call.
type CallOptions struct {
	// Documents to rerank. Either text values or JSON object values.
	Documents Documents

	// Query is the query string to rerank the documents against.
	Query string

	// TopN is an optional limit for returned documents.
	TopN *int

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// ProviderOptions are additional provider-specific options.
	ProviderOptions shared.ProviderOptions

	// Headers are additional HTTP headers to be sent with the request.
	Headers shared.Headers
}

// Documents is a sealed interface representing the documents to rerank.
// Implementations: DocumentsText, DocumentsObject.
type Documents interface {
	documentsType() string
}

// DocumentsText represents a list of text documents.
type DocumentsText struct {
	Values []string
}

func (DocumentsText) documentsType() string { return "text" }

// DocumentsObject represents a list of JSON object documents.
type DocumentsObject struct {
	Values []jsonvalue.JSONObject
}

func (DocumentsObject) documentsType() string { return "object" }
