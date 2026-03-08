// Ported from: packages/core/src/stream/aisdk/v4/input.ts
package v4

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/stream/base"
)

// ---------------------------------------------------------------------------
// AISDKV4InputStream
// ---------------------------------------------------------------------------

// AISDKV4InputStream is the AI SDK v4 input stream transformer.
// It reads LanguageModelV1StreamPart chunks from the provider stream,
// converts them to Mastra ChunkType chunks, and enqueues them to the output.
//
// In TS this extends MastraModelInput (which extends MastraBase).
// In Go it implements the base.MastraModelInput interface.
type AISDKV4InputStream struct {
	Component logger.RegisteredLogger
	Name      string
}

// AISDKV4InputStreamOptions configures an AISDKV4InputStream.
type AISDKV4InputStreamOptions struct {
	Component logger.RegisteredLogger
	Name      string
}

// NewAISDKV4InputStream creates a new AISDKV4InputStream.
func NewAISDKV4InputStream(opts AISDKV4InputStreamOptions) *AISDKV4InputStream {
	return &AISDKV4InputStream{
		Component: opts.Component,
		Name:      opts.Name,
	}
}

// Transform reads from the raw provider stream and converts each
// LanguageModelV1StreamPart to a Mastra ChunkType, enqueuing it
// to the controller channel.
//
// This mirrors the TS async transform() method which iterates the
// ReadableStream<LanguageModelV1StreamPart> and calls
// convertFullStreamChunkToMastra for each chunk.
//
// Note: The base.TransformParams uses stream.LanguageModelV2StreamPart
// (the generic provider stream part). For v4 we need V1 stream parts,
// so we accept V4TransformParams with the v4-specific channel type.
func (s *AISDKV4InputStream) Transform(params base.TransformParams) error {
	return s.TransformV4(V4TransformParams{
		RunID:      params.RunID,
		Controller: params.Controller,
	})
}

// V4TransformParams are the parameters for the v4-specific Transform.
type V4TransformParams struct {
	RunID      string
	Stream     <-chan LanguageModelV1StreamPart
	Controller chan<- stream.ChunkType
}

// TransformV4 is the v4-specific transform that reads LanguageModelV1StreamPart
// chunks and converts them to Mastra ChunkType chunks.
func (s *AISDKV4InputStream) TransformV4(params V4TransformParams) error {
	ctx := TransformContext{RunID: params.RunID}

	for chunk := range params.Stream {
		transformed := ConvertFullStreamChunkToMastra(chunk, ctx)
		if transformed != nil {
			params.Controller <- *transformed
		}
	}

	return nil
}
