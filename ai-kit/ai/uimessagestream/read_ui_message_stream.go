// Ported from: packages/ai/src/ui-message-stream/read-ui-message-stream.ts
package uimessagestream

import (
	"context"
	"errors"
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// ReadUIMessageStreamOptions configures ReadUIMessageStream.
type ReadUIMessageStreamOptions struct {
	// Message is the last assistant message to use as a starting point
	// when the conversation is resumed. Otherwise nil.
	Message *UIMessage

	// Stream is the channel of UIMessageChunk values to read.
	Stream <-chan UIMessageChunk

	// OnError is called when an error occurs.
	OnError func(err error)

	// TerminateOnError controls whether the stream terminates if an error occurs.
	TerminateOnError bool
}

// ReadUIMessageStream transforms a channel of UIMessageChunk values into a
// Stream of UIMessage values. Each stream value is a different state of the same
// message as it is being completed.
//
// This is the Go equivalent of the TypeScript readUIMessageStream which returns
// an AsyncIterableStream<UIMessage>.
func ReadUIMessageStream(opts ReadUIMessageStreamOptions) *util.Stream[UIMessage] {
	messageID := ""
	if opts.Message != nil {
		messageID = opts.Message.ID
	}

	state := CreateStreamingUIMessageState(messageID, opts.Message)
	hasErrored := false

	handleError := func(err error) {
		if opts.OnError != nil {
			opts.OnError(err)
		}
		if !hasErrored && opts.TerminateOnError {
			hasErrored = true
		}
	}

	return util.NewStream[UIMessage](context.Background(), func(w *util.StreamWriter[UIMessage]) {
		processed := ProcessUIMessageStream(ProcessUIMessageStreamOptions{
			Input: opts.Stream,
			RunUpdateMessageJob: func(job func(state *StreamingUIMessageState, write func()) error) error {
				return job(state, func() {
					msg := state.Message.DeepCopy()
					w.Enqueue(msg)
				})
			},
			OnError: handleError,
		})

		for range processed {
			// Consume the processed stream to drive the state machine.
			// Actual UIMessage emissions happen via the write callback above.
		}

		if hasErrored {
			w.Error(errors.New(fmt.Sprintf("stream error")))
		}
	})
}
