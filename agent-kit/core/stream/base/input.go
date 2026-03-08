// Ported from: packages/core/src/stream/base/input.ts
package base

import (
	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// Safe controller helpers
// ---------------------------------------------------------------------------

// SafeEnqueue safely sends a chunk to a channel. Returns false if the
// channel is closed or cannot accept the value.
//
// Mirrors the TS safeEnqueue() from base/input.ts:
//
//	Prefer this over checking desiredSize before enqueue, because
//	desiredSize === 0 indicates backpressure (queue full, stream still open)
//	— not closure. Guarding on desiredSize would silently drop chunks.
func SafeEnqueue(ch chan<- stream.ChunkType, chunk stream.ChunkType) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	ch <- chunk
	return true
}

// SafeClose safely closes a channel. Returns false if the channel
// was already closed.
// Mirrors the TS safeClose() from base/input.ts.
func SafeClose(ch chan<- stream.ChunkType) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	close(ch)
	return true
}

// SafeError is a no-op in Go (channels don't have an error state like
// ReadableStreamDefaultController). Kept for API parity with the TS version.
func SafeError(_ chan<- stream.ChunkType, _ error) bool {
	return false
}

// ---------------------------------------------------------------------------
// MastraModelInput — abstract base for model input stream transforms
// ---------------------------------------------------------------------------

// MastraModelInput is the abstract base for stream input transformers.
// In TS this is an abstract class extending MastraBase; in Go it's an interface.
//
// Implementations must provide Transform(). The Initialize() function
// wires up the stream creation pipeline as a default helper.
//
// TS signature:
//
//	abstract class MastraModelInput extends MastraBase {
//	  abstract transform({ runId, stream, controller }): Promise<void>;
//	  initialize({ runId, createStream, onResult }): ReadableStream<ChunkType>;
//	}
type MastraModelInput interface {
	// Transform reads from the raw provider stream and writes Mastra chunks
	// to the output channel.
	Transform(params TransformParams) error
}

// TransformParams are the parameters for MastraModelInput.Transform.
type TransformParams struct {
	RunID      string
	Stream     <-chan stream.LanguageModelV2StreamPart
	Controller chan<- stream.ChunkType
}

// Initialize wires up the stream creation pipeline for a MastraModelInput.
// It calls createStream, invokes onResult with stream metadata, then
// delegates to input.Transform() for the actual stream processing.
//
// Returns a read-only channel that delivers ChunkType events.
//
// Mirrors the TS MastraModelInput.initialize() method:
//
//	initialize({ runId, createStream, onResult }) {
//	  return new ReadableStream<ChunkType>({
//	    async start(controller) {
//	      const stream = await createStream();
//	      onResult({ warnings, request, rawResponse });
//	      await self.transform({ runId, stream: stream.stream, controller });
//	      safeClose(controller);
//	    }
//	  });
//	}
func Initialize(input MastraModelInput, runID string, createStream stream.CreateStream, onResult stream.OnResult) <-chan stream.ChunkType {
	out := make(chan stream.ChunkType, 256)

	go func() {
		defer SafeClose(out)

		result, err := createStream()
		if err != nil {
			// Cannot enqueue error to channel in Go the way TS does with controller.error;
			// the caller should handle this via the channel closing.
			return
		}

		if onResult != nil {
			onResult(stream.LanguageModelV2StreamResultMeta{
				Warnings:    result.Warnings,
				Request:     result.Request,
				RawResponse: result.RawResponse,
			})
		}

		err = input.Transform(TransformParams{
			RunID:      runID,
			Stream:     result.Stream,
			Controller: out,
		})
		if err != nil {
			return
		}
	}()

	return out
}
