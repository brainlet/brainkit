// Ported from: packages/core/src/stream/aisdk/v5/compat/consume-stream.ts
package compat

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ConsumeStreamOptions configures the ConsumeStream helper.
type ConsumeStreamOptions struct {
	OnError func(err error)
	Logger  logger.IMastraLogger
}

// ConsumeStream reads from a channel until it is closed, draining all values.
// If an error callback is provided, it is called on any recovered panic.
// Mirrors the TS consumeStream() that reads a ReadableStream to completion.
func ConsumeStream(ch <-chan any, opts *ConsumeStreamOptions) {
	defer func() {
		if r := recover(); r != nil {
			if opts != nil {
				if opts.Logger != nil {
					if err, ok := r.(error); ok {
						opts.Logger.Error("consumeStream error", err)
					}
				}
				if opts.OnError != nil {
					if err, ok := r.(error); ok {
						opts.OnError(err)
					}
				}
			}
		}
	}()

	for range ch {
		// drain
	}
}
