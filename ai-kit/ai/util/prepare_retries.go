// Ported from: packages/ai/src/util/prepare-retries.ts
package util

import (
	"context"
	"fmt"
)

// PrepareRetriesResult holds the validated maxRetries and the retry function.
type PrepareRetriesResult struct {
	MaxRetries int
	Retry      RetryFunction
}

// PrepareRetries validates and prepares a retry function with the given options.
func PrepareRetries(maxRetries *int, ctx context.Context) (PrepareRetriesResult, error) {
	if maxRetries != nil {
		mr := *maxRetries

		// Check if integer (Go ints are always integers, so just check range)
		if mr < 0 {
			return PrepareRetriesResult{}, fmt.Errorf(
				"invalid argument for parameter maxRetries: maxRetries must be >= 0 (got %d)", mr,
			)
		}
	}

	mr := 2
	if maxRetries != nil {
		mr = *maxRetries
	}

	retryFn := RetryWithExponentialBackoffRespectingRetryHeaders(&RetryWithExponentialBackoffOptions{
		MaxRetries: mr,
	})

	return PrepareRetriesResult{
		MaxRetries: mr,
		Retry:      retryFn,
	}, nil
}
