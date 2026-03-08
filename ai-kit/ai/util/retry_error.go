// Ported from: packages/ai/src/util/retry-error.ts
package util

import "fmt"

// RetryErrorReason describes why a retry operation failed.
type RetryErrorReason string

const (
	RetryReasonMaxRetriesExceeded RetryErrorReason = "maxRetriesExceeded"
	RetryReasonErrorNotRetryable  RetryErrorReason = "errorNotRetryable"
	RetryReasonAbort              RetryErrorReason = "abort"
)

// RetryError is returned when a retried operation ultimately fails.
type RetryError struct {
	// Reason is why the retry sequence ended.
	Reason RetryErrorReason
	// Errors contains all errors accumulated during retry attempts.
	Errors []error
	// LastError is the final error (convenience accessor).
	LastError error
	// Message is the human-readable error message.
	Message string
}

// NewRetryError creates a new RetryError.
func NewRetryError(message string, reason RetryErrorReason, errors []error) *RetryError {
	var lastErr error
	if len(errors) > 0 {
		lastErr = errors[len(errors)-1]
	}
	return &RetryError{
		Message:   message,
		Reason:    reason,
		Errors:    errors,
		LastError: lastErr,
	}
}

func (e *RetryError) Error() string {
	return fmt.Sprintf("AI_RetryError: %s", e.Message)
}

func (e *RetryError) Unwrap() error {
	return e.LastError
}

// IsRetryError checks whether the given error is a RetryError.
func IsRetryError(err error) bool {
	_, ok := err.(*RetryError)
	return ok
}
