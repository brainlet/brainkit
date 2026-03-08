// Ported from: packages/provider/src/embedding-model/v3/embedding-model-v3-call-options.ts
package embeddingmodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// CallOptions contains the options for an embedding model call.
type CallOptions struct {
	// Values is the list of text values to generate embeddings for.
	Values []string

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// ProviderOptions are additional provider-specific options.
	ProviderOptions shared.ProviderOptions

	// Headers are additional HTTP headers to be sent with the request.
	// Only applicable for HTTP-based providers.
	Headers shared.Headers
}
