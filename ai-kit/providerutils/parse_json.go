// Ported from: packages/provider-utils/src/parse-json.ts
package providerutils

import "fmt"

// JSONParseError represents an error that occurred during JSON parsing.
type JSONParseError struct {
	Text    string
	Cause   error
	Message string
}

func (e *JSONParseError) Error() string {
	return e.Message
}

func (e *JSONParseError) Unwrap() error {
	return e.Cause
}

// NewJSONParseError creates a new JSONParseError.
func NewJSONParseError(text string, cause error) *JSONParseError {
	msg := "JSON parsing failed"
	if cause != nil {
		msg = fmt.Sprintf("JSON parsing failed: %v", cause)
	}
	return &JSONParseError{
		Text:    text,
		Cause:   cause,
		Message: msg,
	}
}

// IsJSONParseError checks whether the given error is a JSONParseError.
func IsJSONParseError(err error) bool {
	_, ok := err.(*JSONParseError)
	return ok
}

// ParseResult represents the result of a safe JSON parse operation.
type ParseResult[T any] struct {
	Success  bool
	Value    T
	RawValue interface{}
	Error    error
}

// ParseJSON parses a JSON string into a strongly-typed object using the provided schema.
// If no schema is provided, returns the raw parsed value.
func ParseJSON[T any](text string, schema *Schema[T]) (T, error) {
	var zero T

	value, err := SecureJsonParse(text)
	if err != nil {
		return zero, NewJSONParseError(text, err)
	}

	if schema == nil {
		if v, ok := value.(T); ok {
			return v, nil
		}
		return zero, nil
	}

	result, validateErr := ValidateTypes(value, schema)
	if validateErr != nil {
		if IsJSONParseError(validateErr) || IsTypeValidationError(validateErr) {
			return zero, validateErr
		}
		return zero, NewJSONParseError(text, validateErr)
	}

	return result, nil
}

// SafeParseJSON safely parses a JSON string and returns a ParseResult.
// If no schema is provided, returns the raw parsed value.
func SafeParseJSON[T any](text string, schema *Schema[T]) ParseResult[T] {
	var zero T

	value, err := SecureJsonParse(text)
	if err != nil {
		return ParseResult[T]{
			Success:  false,
			Error:    NewJSONParseError(text, err),
			RawValue: nil,
		}
	}

	if schema == nil {
		if v, ok := value.(T); ok {
			return ParseResult[T]{
				Success:  true,
				Value:    v,
				RawValue: value,
			}
		}
		return ParseResult[T]{
			Success:  true,
			Value:    zero,
			RawValue: value,
		}
	}

	result := SafeValidateTypes(value, schema)
	if !result.Success {
		return ParseResult[T]{
			Success:  false,
			Error:    result.Error,
			RawValue: result.RawValue,
		}
	}

	return ParseResult[T]{
		Success:  true,
		Value:    result.Value,
		RawValue: result.RawValue,
	}
}
