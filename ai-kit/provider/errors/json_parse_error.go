// Ported from: packages/provider/src/errors/json-parse-error.ts
package errors

import "fmt"

// JSONParseError indicates a JSON parsing failure.
type JSONParseError struct {
	AISDKError

	// Text is the text that failed to parse.
	Text string
}

// NewJSONParseError creates a new JSONParseError.
func NewJSONParseError(text string, cause error) *JSONParseError {
	return &JSONParseError{
		AISDKError: AISDKError{
			Name: "AI_JSONParseError",
			Message: fmt.Sprintf("JSON parsing failed: Text: %s.\nError message: %s",
				text, GetErrorMessage(cause)),
			Cause: cause,
		},
		Text: text,
	}
}

// Error implements the error interface.
func (e *JSONParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

// Unwrap returns the underlying cause.
func (e *JSONParseError) Unwrap() error {
	return e.Cause
}

// IsJSONParseError checks if an error is a JSONParseError.
func IsJSONParseError(err error) bool {
	var target *JSONParseError
	return As(err, &target)
}
