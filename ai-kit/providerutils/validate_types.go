// Ported from: packages/provider-utils/src/validate-types.ts
package providerutils

import "fmt"

// TypeValidationError represents a type validation failure.
type TypeValidationError struct {
	Value   interface{}
	Cause   error
	Message string
}

func (e *TypeValidationError) Error() string {
	return e.Message
}

func (e *TypeValidationError) Unwrap() error {
	return e.Cause
}

// NewTypeValidationError creates a new TypeValidationError.
func NewTypeValidationError(value interface{}, cause error) *TypeValidationError {
	msg := "Type validation failed"
	if cause != nil {
		msg = fmt.Sprintf("Type validation failed: %v", cause)
	}
	return &TypeValidationError{
		Value:   value,
		Cause:   cause,
		Message: msg,
	}
}

// IsTypeValidationError checks whether the given error is a TypeValidationError.
func IsTypeValidationError(err error) bool {
	_, ok := err.(*TypeValidationError)
	return ok
}

// ValidateTypesResult is the result of SafeValidateTypes.
type ValidateTypesResult[T any] struct {
	Success  bool
	Value    T
	RawValue interface{}
	Error    *TypeValidationError
}

// ValidateTypes validates the types of an unknown object using a schema and
// returns a strongly-typed object.
func ValidateTypes[T any](value interface{}, schema *Schema[T]) (T, error) {
	result := SafeValidateTypes(value, schema)
	if !result.Success {
		return result.Value, result.Error
	}
	return result.Value, nil
}

// SafeValidateTypes safely validates the types of an unknown object using a schema.
func SafeValidateTypes[T any](value interface{}, schema *Schema[T]) ValidateTypesResult[T] {
	if schema == nil || schema.Validate == nil {
		// Cast directly if no validation function
		v, ok := value.(T)
		if ok {
			return ValidateTypesResult[T]{
				Success:  true,
				Value:    v,
				RawValue: value,
			}
		}
		// For interface{} types, just pass through
		var zero T
		return ValidateTypesResult[T]{
			Success:  true,
			Value:    zero,
			RawValue: value,
		}
	}

	result, err := schema.Validate(value)
	if err != nil {
		return ValidateTypesResult[T]{
			Success:  false,
			Error:    NewTypeValidationError(value, err),
			RawValue: value,
		}
	}

	if result.Success {
		return ValidateTypesResult[T]{
			Success:  true,
			Value:    result.Value,
			RawValue: value,
		}
	}

	return ValidateTypesResult[T]{
		Success:  false,
		Error:    NewTypeValidationError(value, result.Error),
		RawValue: value,
	}
}
