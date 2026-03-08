// Ported from: packages/core/src/stream/base/base.ts
// and: packages/core/src/stream/base/index.ts (re-exports)
//
// NOTE: The following base/ files have been ported to their own Go files:
//   input.ts       → input.go       (SafeEnqueue, SafeClose, SafeError, MastraModelInput, TransformParams, Initialize)
//   consume-stream.ts → consume_stream.go (ConsumeStreamOptions, ConsumeStream)
//   schema.ts      → schema.go      (JSONSchema7, OutputSchema, PartialSchemaOutput, etc.)
//   output.ts      → output.go      (MastraModelOutput, FullOutput, etc.)
//   output-format-handlers.ts → output_format_handlers.go (FormatHandler, ObjectStreamTransformer, etc.)
package base

import (
	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// MastraBaseStream interface
// ---------------------------------------------------------------------------

// MastraBaseStream is the core interface for all Mastra stream types.
// Mirrors the TS interface: { fullStream, consumeStream() }.
type MastraBaseStream interface {
	// FullStream returns a channel that delivers all stream events (replaying
	// buffered chunks and delivering new ones).
	FullStream() <-chan stream.ChunkType
	// ConsumeStream reads through the full stream to drive processing.
	ConsumeStream(onError func(error))
}
