// Ported from: packages/ai/src/util/simulate-readable-stream.ts
package util

import (
	"context"
	"time"
)

// SimulateReadableStreamOptions configures the simulated stream.
type SimulateReadableStreamOptions[T any] struct {
	// Chunks are the values to emit.
	Chunks []T
	// InitialDelayInMs is the delay before emitting the first value.
	// Use nil to skip the delay entirely. Use IntPtr(0) for a zero-duration delay.
	// Default if not set: IntPtr(0).
	InitialDelayInMs *int
	// ChunkDelayInMs is the delay between each subsequent value.
	// Use nil to skip the delay entirely. Use IntPtr(0) for a zero-duration delay.
	// Default if not set: IntPtr(0).
	ChunkDelayInMs *int
	// DelayFunc is an optional custom delay function for testing.
	// If nil, time.Sleep is used. Receives nil when delay should be skipped.
	DelayFunc func(ms *int)
	// InitialDelaySet indicates whether InitialDelayInMs was explicitly set
	// (to distinguish nil = "skip" from "not set" = "use default 0").
	InitialDelaySet bool
	// ChunkDelaySet indicates whether ChunkDelayInMs was explicitly set.
	ChunkDelaySet bool
}

// IntPtr returns a pointer to an int value. Helper for SimulateReadableStreamOptions.
func IntPtr(v int) *int {
	return &v
}

// SimulateReadableStream creates a Stream that emits the provided values with
// optional delays between each value.
func SimulateReadableStream[T any](ctx context.Context, opts SimulateReadableStreamOptions[T]) *Stream[T] {
	delayFn := opts.DelayFunc
	if delayFn == nil {
		delayFn = func(ms *int) {
			if ms == nil {
				return
			}
			time.Sleep(time.Duration(*ms) * time.Millisecond)
		}
	}

	// Resolve delays: use explicit value if set, otherwise default to IntPtr(0).
	initialDelay := IntPtr(0)
	if opts.InitialDelaySet {
		initialDelay = opts.InitialDelayInMs
	} else if opts.InitialDelayInMs != nil {
		initialDelay = opts.InitialDelayInMs
	}

	chunkDelay := IntPtr(0)
	if opts.ChunkDelaySet {
		chunkDelay = opts.ChunkDelayInMs
	} else if opts.ChunkDelayInMs != nil {
		chunkDelay = opts.ChunkDelayInMs
	}

	return NewStream[T](ctx, func(w *StreamWriter[T]) {
		for i, chunk := range opts.Chunks {
			if i == 0 {
				delayFn(initialDelay)
			} else {
				delayFn(chunkDelay)
			}
			if !w.Enqueue(chunk) {
				return
			}
		}
	})
}
