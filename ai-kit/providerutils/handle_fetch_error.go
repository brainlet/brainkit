// Ported from: packages/provider-utils/src/handle-fetch-error.ts
package providerutils

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
)

// networkErrorCodes contains error codes that indicate network connectivity issues.
var networkErrorCodes = []syscall.Errno{
	syscall.ECONNREFUSED,
	syscall.ECONNRESET,
	syscall.ETIMEDOUT,
	syscall.EPIPE,
}

// fetchFailedMessages contains error messages that indicate a fetch failure.
var fetchFailedMessages = []string{"fetch failed", "failed to fetch"}

// HandleFetchError handles errors from HTTP fetch operations, wrapping network
// errors in APICallError with isRetryable=true, and passing through abort errors.
func HandleFetchError(opts HandleFetchErrorOptions) error {
	if opts.Error == nil {
		return nil
	}

	if IsAbortError(opts.Error) {
		return opts.Error
	}

	// Check for network errors (connection refused, reset, timeout, etc.)
	var errno syscall.Errno
	if errors.As(opts.Error, &errno) {
		for _, code := range networkErrorCodes {
			if errno == code {
				return NewAPICallError(APICallErrorOptions{
					Message:         fmt.Sprintf("Cannot connect to API: %v", opts.Error),
					Cause:           opts.Error,
					URL:             opts.URL,
					RequestBodyValues: opts.RequestBodyValues,
					IsRetryable:     true,
				})
			}
		}
	}

	// Check for "fetch failed" type messages
	errMsg := strings.ToLower(opts.Error.Error())
	for _, msg := range fetchFailedMessages {
		if errMsg == msg {
			cause := errors.Unwrap(opts.Error)
			if cause != nil {
				return NewAPICallError(APICallErrorOptions{
					Message:         fmt.Sprintf("Cannot connect to API: %v", cause),
					Cause:           cause,
					URL:             opts.URL,
					RequestBodyValues: opts.RequestBodyValues,
					IsRetryable:     true,
				})
			}
			break
		}
	}

	return opts.Error
}

// HandleFetchErrorOptions are the options for HandleFetchError.
type HandleFetchErrorOptions struct {
	Error             error
	URL               string
	RequestBodyValues interface{}
}

// APICallError represents an error from an API call.
type APICallError struct {
	Message           string
	Cause             error
	URL               string
	RequestBodyValues interface{}
	StatusCode        int
	ResponseHeaders   map[string]string
	ResponseBody      string
	Data              interface{}
	IsRetryable       bool
}

// APICallErrorOptions are the options for creating an APICallError.
type APICallErrorOptions struct {
	Message           string
	Cause             error
	URL               string
	RequestBodyValues interface{}
	StatusCode        int
	ResponseHeaders   map[string]string
	ResponseBody      string
	Data              interface{}
	IsRetryable       bool
}

// NewAPICallError creates a new APICallError.
func NewAPICallError(opts APICallErrorOptions) *APICallError {
	return &APICallError{
		Message:           opts.Message,
		Cause:             opts.Cause,
		URL:               opts.URL,
		RequestBodyValues: opts.RequestBodyValues,
		StatusCode:        opts.StatusCode,
		ResponseHeaders:   opts.ResponseHeaders,
		ResponseBody:      opts.ResponseBody,
		Data:              opts.Data,
		IsRetryable:       opts.IsRetryable,
	}
}

func (e *APICallError) Error() string {
	return e.Message
}

func (e *APICallError) Unwrap() error {
	return e.Cause
}

// IsAPICallError checks whether the given error is an APICallError.
func IsAPICallError(err error) bool {
	var apiErr *APICallError
	return errors.As(err, &apiErr)
}

// EmptyResponseBodyError represents an error when the response body is empty.
type EmptyResponseBodyError struct {
	Message string
}

func (e *EmptyResponseBodyError) Error() string {
	if e.Message == "" {
		return "Empty response body"
	}
	return e.Message
}
