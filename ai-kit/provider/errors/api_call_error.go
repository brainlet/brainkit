// Ported from: packages/provider/src/errors/api-call-error.ts
package errors

import "fmt"

// APICallError represents an error from an API call.
type APICallError struct {
	AISDKError

	URL              string
	RequestBodyValues any
	StatusCode       *int
	ResponseHeaders  map[string]string
	ResponseBody     *string
	IsRetryable      bool
	Data             any
}

// NewAPICallError creates a new APICallError.
// If isRetryable is nil, it defaults based on the status code.
func NewAPICallError(opts APICallErrorOptions) *APICallError {
	isRetryable := false
	if opts.IsRetryable != nil {
		isRetryable = *opts.IsRetryable
	} else if opts.StatusCode != nil {
		sc := *opts.StatusCode
		isRetryable = sc == 408 || sc == 409 || sc == 429 || sc >= 500
	}

	return &APICallError{
		AISDKError: AISDKError{
			Name:    "AI_APICallError",
			Message: opts.Message,
			Cause:   opts.Cause,
		},
		URL:              opts.URL,
		RequestBodyValues: opts.RequestBodyValues,
		StatusCode:       opts.StatusCode,
		ResponseHeaders:  opts.ResponseHeaders,
		ResponseBody:     opts.ResponseBody,
		IsRetryable:      isRetryable,
		Data:             opts.Data,
	}
}

// APICallErrorOptions are the options for creating an APICallError.
type APICallErrorOptions struct {
	Message          string
	URL              string
	RequestBodyValues any
	StatusCode       *int
	ResponseHeaders  map[string]string
	ResponseBody     *string
	Cause            error
	IsRetryable      *bool
	Data             any
}

// Error implements the error interface.
func (e *APICallError) Error() string {
	msg := fmt.Sprintf("%s: %s (url: %s)", e.Name, e.Message, e.URL)
	if e.StatusCode != nil {
		msg += fmt.Sprintf(" (status: %d)", *e.StatusCode)
	}
	if e.Cause != nil {
		msg += fmt.Sprintf(" (cause: %v)", e.Cause)
	}
	return msg
}

// Unwrap returns the underlying cause.
func (e *APICallError) Unwrap() error {
	return e.Cause
}

// IsAPICallError checks if an error is an APICallError.
func IsAPICallError(err error) bool {
	var target *APICallError
	return As(err, &target)
}
