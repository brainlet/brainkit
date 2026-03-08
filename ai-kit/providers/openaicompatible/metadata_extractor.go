// Ported from: packages/openai-compatible/src/chat/openai-compatible-metadata-extractor.ts
package openaicompatible

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// StreamExtractor processes individual chunks from a streaming response
// and builds the final metadata from the accumulated stream data.
type StreamExtractor interface {
	// ProcessChunk processes an individual chunk from the stream. Called for
	// each chunk in the response stream to accumulate metadata throughout
	// the streaming process.
	ProcessChunk(parsedChunk interface{})

	// BuildMetadata builds the metadata object after all chunks have been
	// processed. Called at the end of the stream to generate the complete
	// provider metadata.
	BuildMetadata() *shared.ProviderMetadata
}

// MetadataExtractor extracts provider-specific metadata from API responses.
// Used to standardize metadata handling across different LLM providers while
// allowing provider-specific metadata to be captured.
type MetadataExtractor interface {
	// ExtractMetadata extracts provider metadata from a complete, non-streaming
	// response.
	//
	// parsedBody is the parsed response JSON body from the provider's API.
	// Returns provider-specific metadata or nil if no metadata is available.
	// The metadata should be under a key indicating the provider id.
	ExtractMetadata(parsedBody interface{}) (*shared.ProviderMetadata, error)

	// CreateStreamExtractor creates an extractor for handling streaming responses.
	// The returned object provides methods to process individual chunks and build
	// the final metadata from the accumulated stream data.
	CreateStreamExtractor() StreamExtractor
}
