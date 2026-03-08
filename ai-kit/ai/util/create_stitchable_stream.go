// Ported from: packages/ai/src/util/create-stitchable-stream.ts
package util

import (
	"context"
	"errors"
	"sync"
)

// StitchableStream is a stream that can pipe one inner stream at a time.
// Inner streams are consumed sequentially in the order they are added.
type StitchableStream[T any] struct {
	out       chan T
	addCh     chan *Stream[T]
	closeCh   chan struct{}
	termCh    chan struct{}
	done      chan struct{}
	mu        sync.Mutex
	closed    bool
	ctx       context.Context
	cancel    context.CancelFunc
}

// StitchableStreamResult holds the stitchable stream and its control methods.
type StitchableStreamResult[T any] struct {
	// Stream is the output stream that yields values from all added inner streams.
	Stream *Stream[T]
	// AddStream adds an inner stream. Panics if the outer stream is closed.
	AddStream func(inner *Stream[T])
	// Close gracefully closes the outer stream. Inner streams finish processing first.
	Close func()
	// Terminate immediately closes the outer stream and cancels all inner streams.
	Terminate func()
}

// CreateStitchableStream creates a stitchable stream that can pipe one stream at a time.
func CreateStitchableStream[T any](ctx context.Context) StitchableStreamResult[T] {
	ctx, cancel := context.WithCancel(ctx)

	ss := &StitchableStream[T]{
		out:     make(chan T),
		addCh:   make(chan *Stream[T], 16),
		closeCh: make(chan struct{}, 1),
		termCh:  make(chan struct{}, 1),
		done:    make(chan struct{}),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Background goroutine that processes inner streams sequentially.
	go ss.run()

	outStream := NewStream[T](ctx, func(w *StreamWriter[T]) {
		for {
			select {
			case val, ok := <-ss.out:
				if !ok {
					return
				}
				if !w.Enqueue(val) {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	})

	return StitchableStreamResult[T]{
		Stream: outStream,
		AddStream: func(inner *Stream[T]) {
			ss.mu.Lock()
			if ss.closed {
				ss.mu.Unlock()
				panic("Cannot add inner stream: outer stream is closed")
			}
			ss.mu.Unlock()
			ss.addCh <- inner
		},
		Close: func() {
			ss.mu.Lock()
			ss.closed = true
			ss.mu.Unlock()
			select {
			case ss.closeCh <- struct{}{}:
			default:
			}
		},
		Terminate: func() {
			ss.mu.Lock()
			ss.closed = true
			ss.mu.Unlock()
			select {
			case ss.termCh <- struct{}{}:
			default:
			}
		},
	}
}

func (ss *StitchableStream[T]) run() {
	defer close(ss.out)
	defer close(ss.done)

	var pending []*Stream[T]
	closedOuter := false

	// checkClose non-blockingly checks if close or terminate has been signaled.
	checkClose := func() (closed, terminated bool) {
		select {
		case <-ss.termCh:
			return false, true
		default:
		}
		select {
		case <-ss.closeCh:
			return true, false
		default:
		}
		return false, false
	}

	// drainAddCh non-blockingly drains all pending addCh items.
	drainAddCh := func() {
		for {
			select {
			case inner := <-ss.addCh:
				pending = append(pending, inner)
			default:
				return
			}
		}
	}

	for {
		// Check for terminate/close signals at each iteration.
		if closed, terminated := checkClose(); terminated {
			for _, s := range pending {
				s.Cancel()
			}
			return
		} else if closed {
			closedOuter = true
		}

		// Always drain addCh to pick up any queued streams.
		drainAddCh()

		// If we have pending streams, drain the first one.
		if len(pending) > 0 {
			current := pending[0]
			if ss.drainStream(current) {
				return
			}
			pending = pending[1:]

			// After draining, pick up any newly added streams.
			drainAddCh()

			if closedOuter && len(pending) == 0 {
				return
			}
			continue
		}

		// If outer is closed and no pending, we're done.
		if closedOuter {
			return
		}

		// No pending streams. Wait for a new one, close signal, or terminate.
		select {
		case inner := <-ss.addCh:
			pending = append(pending, inner)
		case <-ss.closeCh:
			closedOuter = true
			drainAddCh()
			if len(pending) == 0 {
				return
			}
		case <-ss.termCh:
			for _, s := range pending {
				s.Cancel()
			}
			return
		case <-ss.ctx.Done():
			return
		}
	}
}

// drainStream drains a single inner stream into the output channel.
// Returns true if an error occurred and the stitchable stream should terminate.
func (ss *StitchableStream[T]) drainStream(s *Stream[T]) bool {
	for val, err := range s.Iter() {
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				// Propagate error by closing output.
				return true
			}
			return false
		}
		select {
		case ss.out <- val:
		case <-ss.ctx.Done():
			return false
		}
	}
	return false
}
