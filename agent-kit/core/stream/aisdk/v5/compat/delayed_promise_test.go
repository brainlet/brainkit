// Ported from: packages/core/src/stream/aisdk/v5/compat/delayed-promise.test.ts
package compat

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewDelayedPromise(t *testing.T) {
	t.Run("should create a pending promise", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		status := dp.Status()
		if status.Type != DelayedPromiseStatusPending {
			t.Errorf("expected status pending, got %q", status.Type)
		}
	})
}

func TestDelayedPromiseResolve(t *testing.T) {
	t.Run("should resolve with a value", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		dp.Resolve("hello")

		status := dp.Status()
		if status.Type != DelayedPromiseStatusResolved {
			t.Fatalf("expected status resolved, got %q", status.Type)
		}
		if status.Value != "hello" {
			t.Errorf("expected value 'hello', got %q", status.Value)
		}
	})

	t.Run("should resolve with an int value", func(t *testing.T) {
		dp := NewDelayedPromise[int]()
		dp.Resolve(42)

		status := dp.Status()
		if status.Type != DelayedPromiseStatusResolved {
			t.Fatalf("expected status resolved, got %q", status.Type)
		}
		if status.Value != 42 {
			t.Errorf("expected value 42, got %d", status.Value)
		}
	})

	t.Run("should allow Await after Resolve", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		dp.Resolve("world")

		val, err := dp.Await()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "world" {
			t.Errorf("expected 'world', got %q", val)
		}
	})
}

func TestDelayedPromiseReject(t *testing.T) {
	t.Run("should reject with an error", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		dp.Reject(errors.New("test error"))

		status := dp.Status()
		if status.Type != DelayedPromiseStatusRejected {
			t.Fatalf("expected status rejected, got %q", status.Type)
		}
		if status.Error == nil || status.Error.Error() != "test error" {
			t.Errorf("expected error 'test error', got %v", status.Error)
		}
	})

	t.Run("should return error from Await after Reject", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		dp.Reject(errors.New("rejected"))

		val, err := dp.Await()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "rejected" {
			t.Errorf("expected error 'rejected', got %q", err.Error())
		}
		if val != "" {
			t.Errorf("expected zero value, got %q", val)
		}
	})
}

func TestDelayedPromiseAwait(t *testing.T) {
	t.Run("should block until resolved", func(t *testing.T) {
		dp := NewDelayedPromise[string]()

		done := make(chan struct{})
		var result string
		var resultErr error

		go func() {
			result, resultErr = dp.Await()
			close(done)
		}()

		// Give the goroutine time to start waiting
		time.Sleep(10 * time.Millisecond)

		dp.Resolve("async result")

		select {
		case <-done:
			if resultErr != nil {
				t.Fatalf("unexpected error: %v", resultErr)
			}
			if result != "async result" {
				t.Errorf("expected 'async result', got %q", result)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Await timed out")
		}
	})

	t.Run("should block until rejected", func(t *testing.T) {
		dp := NewDelayedPromise[int]()

		done := make(chan struct{})
		var resultErr error

		go func() {
			_, resultErr = dp.Await()
			close(done)
		}()

		time.Sleep(10 * time.Millisecond)
		dp.Reject(errors.New("async error"))

		select {
		case <-done:
			if resultErr == nil {
				t.Fatal("expected error, got nil")
			}
			if resultErr.Error() != "async error" {
				t.Errorf("expected 'async error', got %q", resultErr.Error())
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Await timed out")
		}
	})

	t.Run("should return immediately if already resolved", func(t *testing.T) {
		dp := NewDelayedPromise[string]()
		dp.Resolve("already done")

		val, err := dp.Await()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if val != "already done" {
			t.Errorf("expected 'already done', got %q", val)
		}
	})
}

func TestDelayedPromiseConcurrent(t *testing.T) {
	t.Run("should handle multiple concurrent Await callers", func(t *testing.T) {
		dp := NewDelayedPromise[string]()

		var wg sync.WaitGroup
		results := make([]string, 10)
		errs := make([]error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errs[idx] = dp.Await()
			}(i)
		}

		time.Sleep(10 * time.Millisecond)
		dp.Resolve("shared result")

		wg.Wait()

		for i := 0; i < 10; i++ {
			if errs[i] != nil {
				t.Errorf("goroutine %d: unexpected error: %v", i, errs[i])
			}
			if results[i] != "shared result" {
				t.Errorf("goroutine %d: expected 'shared result', got %q", i, results[i])
			}
		}
	})

	t.Run("should be safe for concurrent Resolve calls", func(t *testing.T) {
		dp := NewDelayedPromise[int]()

		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(v int) {
				defer wg.Done()
				dp.Resolve(v)
			}(i)
		}
		wg.Wait()

		status := dp.Status()
		if status.Type != DelayedPromiseStatusResolved {
			t.Fatalf("expected resolved, got %q", status.Type)
		}
	})
}
