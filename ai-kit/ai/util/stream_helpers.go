// Ported from: packages/ai/src/util/async-iterable-stream.ts + consume-stream.ts
// This file defines a minimal Stream[T] abstraction used by the util package,
// analogous to TypeScript's ReadableStream<T> combined with AsyncIterable<T>.
package util

import (
	"context"
	"errors"
	"iter"
	"sync"
)

// Stream is a generic push-based stream that supports iteration via iter.Seq2.
// It is the Go equivalent of ReadableStream<T> + AsyncIterable<T> from the TS SDK.
type Stream[T any] struct {
	ch     chan T
	err    error
	errMu  sync.Mutex
	done   chan struct{}
	cancel context.CancelFunc
	ctx    context.Context
}

// NewStream creates a new Stream. The producer function is called in a goroutine
// and should send values via the Enqueue method and call Close when done.
// If the producer encounters an error, it should call Error.
func NewStream[T any](ctx context.Context, producer func(s *StreamWriter[T])) *Stream[T] {
	ctx, cancel := context.WithCancel(ctx)
	s := &Stream[T]{
		ch:     make(chan T, 1),
		done:   make(chan struct{}),
		cancel: cancel,
		ctx:    ctx,
	}
	w := &StreamWriter[T]{stream: s}
	go func() {
		defer close(s.done)
		defer close(s.ch)
		producer(w)
	}()
	return s
}

// StreamWriter provides methods for the producer to push values into a Stream.
type StreamWriter[T any] struct {
	stream *Stream[T]
}

// Enqueue sends a value to the stream. Returns false if the stream's context is cancelled.
func (w *StreamWriter[T]) Enqueue(value T) bool {
	select {
	case w.stream.ch <- value:
		return true
	case <-w.stream.ctx.Done():
		return false
	}
}

// Error sets an error on the stream.
func (w *StreamWriter[T]) Error(err error) {
	w.stream.errMu.Lock()
	w.stream.err = err
	w.stream.errMu.Unlock()
}

// Context returns the stream's context.
func (w *StreamWriter[T]) Context() context.Context {
	return w.stream.ctx
}

// Iter returns an iter.Seq2 that yields (value, error) pairs.
// The iterator completes when the stream is closed or cancelled.
func (s *Stream[T]) Iter() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// If the stream was already cancelled (e.g. a previous Iter broke),
		// don't yield any values.
		select {
		case <-s.ctx.Done():
			return
		default:
		}
		for {
			select {
			case val, ok := <-s.ch:
				if !ok {
					// Channel closed. Check for error.
					s.errMu.Lock()
					err := s.err
					s.errMu.Unlock()
					if err != nil {
						var zero T
						yield(zero, err)
					}
					return
				}
				if !yield(val, nil) {
					s.cancel()
					// Drain any buffered value so subsequent Iter() calls see nothing.
					select {
					case <-s.ch:
					default:
					}
					return
				}
			case <-s.ctx.Done():
				var zero T
				yield(zero, s.ctx.Err())
				return
			}
		}
	}
}

// Cancel cancels the stream, signaling the producer to stop.
func (s *Stream[T]) Cancel() {
	s.cancel()
}

// Wait blocks until the stream is fully consumed/closed.
func (s *Stream[T]) Wait() {
	<-s.done
}

// Err returns any error that occurred on the stream.
func (s *Stream[T]) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

// StreamFromSlice creates a Stream from a slice of values.
func StreamFromSlice[T any](ctx context.Context, values []T) *Stream[T] {
	return NewStream[T](ctx, func(w *StreamWriter[T]) {
		for _, v := range values {
			if !w.Enqueue(v) {
				return
			}
		}
	})
}

// CollectStream collects all values from a Stream into a slice.
func CollectStream[T any](s *Stream[T]) ([]T, error) {
	var result []T
	for val, err := range s.Iter() {
		if err != nil {
			return result, err
		}
		result = append(result, val)
	}
	return result, nil
}

// ConsumeStream reads a stream to completion, discarding all values.
// If onError is provided, it is called with any error that occurs.
func ConsumeStream[T any](s *Stream[T], onError func(error)) {
	for _, err := range s.Iter() {
		if err != nil && onError != nil {
			onError(err)
			return
		}
	}
}

// ErrStreamClosed is returned when operations are attempted on a closed stream.
var ErrStreamClosed = errors.New("stream is closed")
