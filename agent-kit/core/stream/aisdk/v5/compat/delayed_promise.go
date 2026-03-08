// Ported from: packages/core/src/stream/aisdk/v5/compat/delayed-promise.ts
package compat

import "sync"

// DelayedPromiseStatusType represents the state of a delayed promise.
type DelayedPromiseStatusType string

const (
	DelayedPromiseStatusPending  DelayedPromiseStatusType = "pending"
	DelayedPromiseStatusResolved DelayedPromiseStatusType = "resolved"
	DelayedPromiseStatusRejected DelayedPromiseStatusType = "rejected"
)

// DelayedPromiseStatus holds the status information for a DelayedPromise.
// In TS this is a discriminated union; in Go we use a struct with a Type field.
type DelayedPromiseStatus[T any] struct {
	Type  DelayedPromiseStatusType
	Value T
	Error error
}

// DelayedPromise is a promise that is only constructed once the value is accessed.
// This is useful to avoid unhandled promise rejections when the promise is created
// but not accessed.
//
// In Go, this is implemented with a channel-based future that lazily creates
// the notification channel when Await is called.
type DelayedPromise[T any] struct {
	mu      sync.Mutex
	status  DelayedPromiseStatus[T]
	ch      chan struct{} // closed when resolved/rejected; lazily created
	created bool         // whether ch has been created
}

// NewDelayedPromise creates a new pending DelayedPromise.
func NewDelayedPromise[T any]() *DelayedPromise[T] {
	return &DelayedPromise[T]{
		status: DelayedPromiseStatus[T]{Type: DelayedPromiseStatusPending},
	}
}

// ensureCh lazily creates the notification channel. Must be called with mu held.
func (dp *DelayedPromise[T]) ensureCh() {
	if !dp.created {
		dp.ch = make(chan struct{})
		dp.created = true

		// If already resolved/rejected before anyone called Await, close immediately.
		if dp.status.Type == DelayedPromiseStatusResolved || dp.status.Type == DelayedPromiseStatusRejected {
			close(dp.ch)
		}
	}
}

// Status returns the current status of the promise.
func (dp *DelayedPromise[T]) Status() DelayedPromiseStatus[T] {
	dp.mu.Lock()
	defer dp.mu.Unlock()
	return dp.status
}

// Resolve resolves the promise with a value.
func (dp *DelayedPromise[T]) Resolve(value T) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.status = DelayedPromiseStatus[T]{
		Type:  DelayedPromiseStatusResolved,
		Value: value,
	}

	if dp.created {
		// Channel exists, close it to wake waiters.
		select {
		case <-dp.ch:
			// already closed
		default:
			close(dp.ch)
		}
	}
}

// Reject rejects the promise with an error.
func (dp *DelayedPromise[T]) Reject(err error) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	dp.status = DelayedPromiseStatus[T]{
		Type:  DelayedPromiseStatusRejected,
		Error: err,
	}

	if dp.created {
		select {
		case <-dp.ch:
			// already closed
		default:
			close(dp.ch)
		}
	}
}

// Await blocks until the promise is resolved or rejected.
// Returns (value, nil) on resolve, or (zero, error) on reject.
func (dp *DelayedPromise[T]) Await() (T, error) {
	dp.mu.Lock()
	dp.ensureCh()
	dp.mu.Unlock()

	<-dp.ch

	dp.mu.Lock()
	defer dp.mu.Unlock()

	if dp.status.Type == DelayedPromiseStatusRejected {
		var zero T
		return zero, dp.status.Error
	}
	return dp.status.Value, nil
}
