// Ported from: packages/provider-utils/src/delayed-promise.ts
package providerutils

import "sync"

// DelayedPromise is a value that is resolved or rejected lazily.
// It is only constructed once the value is accessed.
// This is useful to avoid unhandled errors when the promise is created
// but not accessed.
type DelayedPromise[T any] struct {
	mu       sync.Mutex
	status   delayedPromiseStatus
	value    T
	err      error
	ch       chan struct{} // closed when resolved or rejected
	chOnce   sync.Once
}

type delayedPromiseStatus int

const (
	delayedPromisePending  delayedPromiseStatus = 0
	delayedPromiseResolved delayedPromiseStatus = 1
	delayedPromiseRejected delayedPromiseStatus = 2
)

// NewDelayedPromise creates a new pending DelayedPromise.
func NewDelayedPromise[T any]() *DelayedPromise[T] {
	return &DelayedPromise[T]{
		ch: make(chan struct{}),
	}
}

// Resolve sets the value and unblocks any waiters.
func (dp *DelayedPromise[T]) Resolve(value T) {
	dp.mu.Lock()
	dp.status = delayedPromiseResolved
	dp.value = value
	dp.mu.Unlock()
	dp.chOnce.Do(func() { close(dp.ch) })
}

// Reject sets the error and unblocks any waiters.
func (dp *DelayedPromise[T]) Reject(err error) {
	dp.mu.Lock()
	dp.status = delayedPromiseRejected
	dp.err = err
	dp.mu.Unlock()
	dp.chOnce.Do(func() { close(dp.ch) })
}

// Await blocks until the promise is resolved or rejected.
// Returns the value and nil on success, or zero value and error on rejection.
func (dp *DelayedPromise[T]) Await() (T, error) {
	<-dp.ch
	dp.mu.Lock()
	defer dp.mu.Unlock()
	if dp.status == delayedPromiseRejected {
		var zero T
		return zero, dp.err
	}
	return dp.value, nil
}

// IsResolved returns true if the promise has been resolved.
func (dp *DelayedPromise[T]) IsResolved() bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.status == delayedPromiseResolved
}

// IsRejected returns true if the promise has been rejected.
func (dp *DelayedPromise[T]) IsRejected() bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.status == delayedPromiseRejected
}

// IsPending returns true if the promise is still pending.
func (dp *DelayedPromise[T]) IsPending() bool {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.status == delayedPromisePending
}
