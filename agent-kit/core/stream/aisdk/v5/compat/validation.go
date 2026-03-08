// Ported from: packages/core/src/stream/aisdk/v5/compat/validation.ts
package compat

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// ValidationResult
// ---------------------------------------------------------------------------

// ValidationResult represents the result of a type validation.
// It is a discriminated union: either success with a value, or failure with an error.
type ValidationResult[T any] struct {
	Success bool
	Value   T
	Error   error
}

// NewValidationSuccess creates a successful validation result.
func NewValidationSuccess[T any](value T) ValidationResult[T] {
	return ValidationResult[T]{
		Success: true,
		Value:   value,
	}
}

// NewValidationFailure creates a failed validation result.
func NewValidationFailure[T any](err error) ValidationResult[T] {
	return ValidationResult[T]{
		Success: false,
		Error:   err,
	}
}

// ---------------------------------------------------------------------------
// Schema interface
// ---------------------------------------------------------------------------

// Schema represents a validation schema that can validate values.
// This mirrors the TS Schema<T> from @internal/ai-sdk-v5.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type Schema[T any] interface {
	// Validate validates the value against the schema.
	// Returns a ValidationResult indicating success or failure.
	Validate(value any) (ValidationResult[T], error)
}

// ---------------------------------------------------------------------------
// TypeValidationError
// ---------------------------------------------------------------------------

// TypeValidationError mirrors the TS TypeValidationError from @ai-sdk/provider-v5.
// It is raised when type validation fails.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type TypeValidationError struct {
	Value any
	Cause string
}

// Error implements the error interface.
func (e *TypeValidationError) Error() string {
	return fmt.Sprintf("type validation error: %s (value: %v)", e.Cause, e.Value)
}

// ---------------------------------------------------------------------------
// SafeValidateTypes
// ---------------------------------------------------------------------------

// SafeValidateTypes safely validates the types of an unknown value using a schema.
// Based on @ai-sdk/provider-utils safeValidateTypes.
//
// If the schema does not have a Validate method, the value is passed through
// as-is (cast to T). If validation fails, returns a TypeValidationError.
//
// This is the Go equivalent of the async safeValidateTypes function in TS.
func SafeValidateTypes[T any](value any, schema Schema[T]) ValidationResult[T] {
	if schema == nil {
		// No schema means no validation — pass through
		v, ok := value.(T)
		if ok {
			return NewValidationSuccess(v)
		}
		// If the type assertion fails, still try to return it
		// (mirrors TS behavior where `as OBJECT` always succeeds)
		return NewValidationSuccess(v)
	}

	result, err := schema.Validate(value)
	if err != nil {
		return NewValidationFailure[T](err)
	}

	if !result.Success {
		return NewValidationFailure[T](&TypeValidationError{
			Value: value,
			Cause: "Validation failed",
		})
	}

	return NewValidationSuccess(result.Value)
}
