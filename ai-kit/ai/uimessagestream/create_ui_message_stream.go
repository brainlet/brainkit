// Ported from: packages/ai/src/ui-message-stream/create-ui-message-stream.ts
package uimessagestream

import (
	"fmt"
	"sync"
)

// defaultGetErrorMessage returns the error message from an error.
func defaultGetErrorMessage(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

// CreateUIMessageStreamOptions configures CreateUIMessageStream.
type CreateUIMessageStreamOptions struct {
	// Execute is called with a writer to write UI message chunks to the stream.
	Execute func(writer *UIMessageStreamWriter) error

	// OnError extracts an error message from an error. Defaults to err.Error().
	OnError func(err error) string

	// OriginalMessages is the original messages. If provided, persistence mode
	// is assumed and a message ID is provided for the response message.
	OriginalMessages []UIMessage

	// OnStepFinish is called when each step finishes during multi-step agent runs.
	OnStepFinish UIMessageStreamOnStepFinishCallback

	// OnFinish is called when the stream finishes.
	OnFinish UIMessageStreamOnFinishCallback

	// GenerateId generates a unique ID. Defaults to a simple counter-based generator.
	GenerateId IdGenerator
}

// CreateUIMessageStream creates a UI message stream that can be used to send
// messages to the client.
//
// It returns a channel of UIMessageChunk values that represents the stream.
func CreateUIMessageStream(opts CreateUIMessageStreamOptions) <-chan UIMessageChunk {
	onError := opts.OnError
	if onError == nil {
		onError = defaultGetErrorMessage
	}

	generateId := opts.GenerateId
	if generateId == nil {
		counter := 0
		generateId = func() string {
			counter++
			return fmt.Sprintf("msg-%d", counter)
		}
	}

	ch := make(chan UIMessageChunk, 64)

	var mu sync.Mutex
	closed := false

	safeEnqueue := func(data UIMessageChunk) {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		// Use a non-blocking send with recovery for closed channels
		defer func() {
			recover() // suppress send on closed channel
		}()
		ch <- data
	}

	var wg sync.WaitGroup

	// Build the writer
	writer := &UIMessageStreamWriter{
		write: func(chunk UIMessageChunk) {
			safeEnqueue(chunk)
		},
		merge: func(stream <-chan UIMessageChunk) {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for value := range stream {
					safeEnqueue(value)
				}
			}()
		},
		OnError: onError,
	}

	// Execute the user's function
	var executeErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					executeErr = err
				} else {
					executeErr = fmt.Errorf("%v", r)
				}
			}
		}()
		executeErr = opts.Execute(writer)
	}()

	if executeErr != nil {
		safeEnqueue(UIMessageChunk{
			Type:      "error",
			ErrorText: onError(executeErr),
		})
	}

	// Wait for all merged streams to finish, then close the channel
	go func() {
		wg.Wait()
		mu.Lock()
		closed = true
		mu.Unlock()
		close(ch)
	}()

	// Run through handleUIMessageStreamFinish for callback support
	return HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
		Stream:           ch,
		MessageID:        generateId(),
		OriginalMessages: opts.OriginalMessages,
		OnStepFinish:     opts.OnStepFinish,
		OnFinish:         opts.OnFinish,
		OnError: func(err error) {
			// Default error handler for the finish handler
			if onError != nil {
				onError(err)
			}
		},
	})
}
