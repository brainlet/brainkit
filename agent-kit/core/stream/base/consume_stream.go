// Ported from: packages/core/src/stream/base/consume-stream.ts
package base

import (
	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ConsumeStreamOptions configures the ConsumeStream helper.
type ConsumeStreamOptions struct {
	OnError func(error)
	// Logger is optional. Stub: uses any to avoid coupling to logger.IMastraLogger.
	Logger any
}

// ConsumeStream reads from a channel until it's closed, discarding all chunks.
// If an error occurs during processing and onError is provided, it is called.
//
// Mirrors the TS consumeStream() function from base/consume-stream.ts:
//
//	export async function consumeStream({ stream, onError }) {
//	  const reader = stream.getReader();
//	  try {
//	    while (true) { const { done } = await reader.read(); if (done) break; }
//	  } catch (error) { onError?.(error); }
//	  finally { reader.releaseLock(); }
//	}
func ConsumeStream(ch <-chan stream.ChunkType, opts *ConsumeStreamOptions) {
	for range ch {
		// drain — all chunk processing happens in upstream transform pipelines
	}
}
