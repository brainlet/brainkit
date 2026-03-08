// Ported from: packages/ai/src/util/retry-with-exponential-backoff.test.ts
package util

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestRetryWithExponentialBackoff_SuccessOnFirstAttempt(t *testing.T) {
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     2,
		InitialDelayMs: 1, // use 1ms for tests
	})

	result, err := retryFn(context.Background(), func() (interface{}, error) {
		return "success", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Fatalf("expected success, got %v", result)
	}
}

func TestRetryWithExponentialBackoff_RetryOnRetryable(t *testing.T) {
	attempt := 0
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     2,
		InitialDelayMs: 1,
	})

	result, err := retryFn(context.Background(), func() (interface{}, error) {
		attempt++
		if attempt == 1 {
			return nil, &APICallError{
				Message:     "Rate limited",
				URL:         "https://api.example.com",
				IsRetryable: true,
				ResponseHeaders: http.Header{
					"Retry-After-Ms": []string{"1"},
				},
			}
		}
		return "success", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Fatalf("expected success, got %v", result)
	}
	if attempt != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempt)
	}
}

func TestRetryWithExponentialBackoff_MaxRetriesExceeded(t *testing.T) {
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     2,
		InitialDelayMs: 1,
	})

	_, err := retryFn(context.Background(), func() (interface{}, error) {
		return nil, &APICallError{
			Message:         "Rate limited",
			URL:             "https://api.example.com",
			IsRetryable:     true,
			ResponseHeaders: http.Header{},
		}
	})

	if err == nil {
		t.Fatal("expected error")
	}

	var retryErr *RetryError
	if !errors.As(err, &retryErr) {
		t.Fatalf("expected RetryError, got %T: %v", err, err)
	}
	if retryErr.Reason != RetryReasonMaxRetriesExceeded {
		t.Fatalf("expected maxRetriesExceeded, got %s", retryErr.Reason)
	}
}

func TestRetryWithExponentialBackoff_NonRetryableError(t *testing.T) {
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     2,
		InitialDelayMs: 1,
	})

	_, err := retryFn(context.Background(), func() (interface{}, error) {
		return nil, errors.New("non-retryable error")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	// On first try with non-retryable, the error is returned unwrapped
	if err.Error() != "non-retryable error" {
		t.Fatalf("expected non-retryable error, got %v", err)
	}
}

func TestRetryWithExponentialBackoff_FallbackToExponentialBackoff(t *testing.T) {
	attempt := 0
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     2,
		InitialDelayMs: 1,
	})

	result, err := retryFn(context.Background(), func() (interface{}, error) {
		attempt++
		if attempt == 1 {
			return nil, &APICallError{
				Message:         "Temporary error",
				URL:             "https://api.example.com",
				IsRetryable:     true,
				ResponseHeaders: http.Header{}, // no retry headers
			}
		}
		return "success", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Fatalf("expected success, got %v", result)
	}
}

func TestRetryWithExponentialBackoff_ZeroRetries(t *testing.T) {
	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries:     0,
		InitialDelayMs: 1,
	})

	_, err := retryFn(context.Background(), func() (interface{}, error) {
		return nil, errors.New("some error")
	})

	if err == nil {
		t.Fatal("expected error")
	}
	// With 0 retries, should return the error directly
	if err.Error() != "some error" {
		t.Fatalf("expected original error, got %v", err)
	}
}
