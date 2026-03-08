// Ported from: packages/ai/src/util/create-resolvable-promise.ts
package util

import "sync"

// ResolvablePromise is a Go equivalent of a Promise with externally accessible
// resolve and reject functions. It is backed by channels.
type ResolvablePromise[T any] struct {
	value    T
	err      error
	done     chan struct{}
	once     sync.Once
}

// NewResolvablePromise creates a new ResolvablePromise.
func NewResolvablePromise[T any]() *ResolvablePromise[T] {
	return &ResolvablePromise[T]{
		done: make(chan struct{}),
	}
}

// Resolve resolves the promise with a value.
func (p *ResolvablePromise[T]) Resolve(value T) {
	p.once.Do(func() {
		p.value = value
		close(p.done)
	})
}

// Reject rejects the promise with an error.
func (p *ResolvablePromise[T]) Reject(err error) {
	p.once.Do(func() {
		p.err = err
		close(p.done)
	})
}

// Await blocks until the promise is resolved or rejected.
// Returns the value and any error.
func (p *ResolvablePromise[T]) Await() (T, error) {
	<-p.done
	return p.value, p.err
}

// Done returns a channel that is closed when the promise is resolved or rejected.
func (p *ResolvablePromise[T]) Done() <-chan struct{} {
	return p.done
}
