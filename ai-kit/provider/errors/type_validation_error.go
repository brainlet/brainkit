// Ported from: packages/provider/src/errors/type-validation-error.ts
package errors

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TypeValidationContext provides context about what is being validated.
type TypeValidationContext struct {
	// Field is the field path in dot notation (e.g., "message.metadata").
	Field *string

	// EntityName is the entity name (e.g., tool name, data type name).
	EntityName *string

	// EntityID is the entity identifier (e.g., message ID, tool call ID).
	EntityID *string
}

// TypeValidationError indicates a type validation failure.
type TypeValidationError struct {
	AISDKError

	// Value is the value that failed validation.
	Value any

	// Context provides additional context about the validation.
	Context *TypeValidationContext
}

// NewTypeValidationError creates a new TypeValidationError.
func NewTypeValidationError(value any, cause error, context *TypeValidationContext) *TypeValidationError {
	contextPrefix := "Type validation failed"

	if context != nil && context.Field != nil {
		contextPrefix += fmt.Sprintf(" for %s", *context.Field)
	}

	if context != nil && (context.EntityName != nil || context.EntityID != nil) {
		contextPrefix += " ("
		var parts []string
		if context.EntityName != nil {
			parts = append(parts, *context.EntityName)
		}
		if context.EntityID != nil {
			parts = append(parts, fmt.Sprintf("id: %q", *context.EntityID))
		}
		contextPrefix += strings.Join(parts, ", ")
		contextPrefix += ")"
	}

	b, _ := json.Marshal(value)
	message := fmt.Sprintf("%s: Value: %s.\nError message: %s",
		contextPrefix, string(b), GetErrorMessage(cause))

	return &TypeValidationError{
		AISDKError: AISDKError{
			Name:    "AI_TypeValidationError",
			Message: message,
			Cause:   cause,
		},
		Value:   value,
		Context: context,
	}
}

// Error implements the error interface.
func (e *TypeValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Name, e.Message)
}

// Unwrap returns the underlying cause.
func (e *TypeValidationError) Unwrap() error {
	return e.Cause
}

// IsTypeValidationError checks if an error is a TypeValidationError.
func IsTypeValidationError(err error) bool {
	var target *TypeValidationError
	return As(err, &target)
}

// WrapTypeValidationError wraps an error into a TypeValidationError.
// If the cause is already a TypeValidationError with the same value and context, it returns the cause.
func WrapTypeValidationError(value any, cause error, context *TypeValidationContext) *TypeValidationError {
	var existing *TypeValidationError
	if As(cause, &existing) && existing.Value == value {
		// Check context equality.
		if contextEqual(existing.Context, context) {
			return existing
		}
	}
	return NewTypeValidationError(value, cause, context)
}

func contextEqual(a, b *TypeValidationContext) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return ptrStringEqual(a.Field, b.Field) &&
		ptrStringEqual(a.EntityName, b.EntityName) &&
		ptrStringEqual(a.EntityID, b.EntityID)
}

func ptrStringEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
