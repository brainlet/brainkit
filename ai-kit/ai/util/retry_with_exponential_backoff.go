// Ported from: packages/ai/src/util/retry-with-exponential-backoff.ts
package util

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

// RetryFunction is a function that wraps another function with retry logic.
type RetryFunction func(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)

// APICallError represents an error from an API call that may contain
// retry headers and retryability information.
type APICallError struct {
	Message         string
	URL             string
	StatusCode      int
	IsRetryable     bool
	ResponseHeaders http.Header
	Data            interface{}
	Cause           error
}

func (e *APICallError) Error() string {
	return e.Message
}

func (e *APICallError) Unwrap() error {
	return e.Cause
}

// IsAPICallError checks if an error is an APICallError.
func IsAPICallError(err error) (*APICallError, bool) {
	var ace *APICallError
	if errors.As(err, &ace) {
		return ace, true
	}
	return nil, false
}

// RetryWithExponentialBackoffOptions configures the retry behavior.
type RetryWithExponentialBackoffOptions struct {
	MaxRetries     int
	InitialDelayMs int
	BackoffFactor  float64
}

// getRetryDelayInMs determines the delay before the next retry, respecting
// rate-limit headers when present and reasonable.
func getRetryDelayInMs(apiErr *APICallError, exponentialBackoffDelay int) int {
	headers := apiErr.ResponseHeaders
	if headers == nil {
		return exponentialBackoffDelay
	}

	var ms *float64

	// retry-after-ms is more precise than retry-after and used by e.g. OpenAI
	if retryAfterMs := headers.Get("retry-after-ms"); retryAfterMs != "" {
		if timeoutMs, err := strconv.ParseFloat(retryAfterMs, 64); err == nil {
			ms = &timeoutMs
		}
	}

	// About the Retry-After header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Retry-After
	if retryAfter := headers.Get("retry-after"); retryAfter != "" && ms == nil {
		if timeoutSeconds, err := strconv.ParseFloat(retryAfter, 64); err == nil {
			v := timeoutSeconds * 1000
			ms = &v
		} else {
			// Try parsing as HTTP date
			if t, err := http.ParseTime(retryAfter); err == nil {
				v := float64(time.Until(t).Milliseconds())
				ms = &v
			}
		}
	}

	// Check that the delay is reasonable
	if ms != nil && !math.IsNaN(*ms) && *ms >= 0 &&
		(*ms < 60000 || *ms < float64(exponentialBackoffDelay)) {
		return int(*ms)
	}

	return exponentialBackoffDelay
}

// RetryWithExponentialBackoffRespectingRetryHeaders creates a RetryFunction that
// retries a failed API call with exponential backoff, respecting rate limit headers.
func RetryWithExponentialBackoffRespectingRetryHeaders(opts *RetryWithExponentialBackoffOptions) RetryFunction {
	maxRetries := 2
	initialDelayMs := 2000
	backoffFactor := 2.0

	if opts != nil {
		if opts.MaxRetries > 0 || opts.MaxRetries == 0 {
			maxRetries = opts.MaxRetries
		}
		if opts.InitialDelayMs > 0 {
			initialDelayMs = opts.InitialDelayMs
		}
		if opts.BackoffFactor > 0 {
			backoffFactor = opts.BackoffFactor
		}
	}

	return func(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
		return retryWithExponentialBackoff(ctx, fn, maxRetries, initialDelayMs, backoffFactor, nil)
	}
}

func retryWithExponentialBackoff(
	ctx context.Context,
	fn func() (interface{}, error),
	maxRetries int,
	delayMs int,
	backoffFactor float64,
	prevErrors []error,
) (interface{}, error) {
	result, err := fn()
	if err == nil {
		return result, nil
	}

	// Don't retry on context cancellation (abort)
	if ctx.Err() != nil {
		return nil, err
	}

	if maxRetries == 0 {
		return nil, err // don't wrap the error when retries are disabled
	}

	newErrors := append(prevErrors, err)
	tryNumber := len(newErrors)

	if tryNumber > maxRetries {
		return nil, NewRetryError(
			fmt.Sprintf("Failed after %d attempts. Last error: %s", tryNumber, err.Error()),
			RetryReasonMaxRetriesExceeded,
			newErrors,
		)
	}

	if apiErr, ok := IsAPICallError(err); ok && apiErr.IsRetryable && tryNumber <= maxRetries {
		retryDelay := getRetryDelayInMs(apiErr, delayMs)

		select {
		case <-time.After(time.Duration(retryDelay) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}

		return retryWithExponentialBackoff(
			ctx,
			fn,
			maxRetries,
			int(backoffFactor*float64(delayMs)),
			backoffFactor,
			newErrors,
		)
	}

	if tryNumber == 1 {
		return nil, err // don't wrap the error on first try non-retryable
	}

	return nil, NewRetryError(
		fmt.Sprintf("Failed after %d attempts with non-retryable error: '%s'", tryNumber, err.Error()),
		RetryReasonErrorNotRetryable,
		newErrors,
	)
}
