// Ported from: packages/core/src/datasets/validation/errors.ts
package validation

import (
	"fmt"
	"strings"
)

// ============================================================================
// Field Error
// ============================================================================

// FieldError represents a field-level validation error.
type FieldError struct {
	// Path is a JSON Pointer path, e.g., "/name" or "/address/city".
	Path string `json:"path"`
	// Code is the validation error code, e.g., "invalid_type", "too_small".
	Code string `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
}

// ============================================================================
// Schema Validation Error
// ============================================================================

// SchemaValidationError is a schema validation error with field details.
type SchemaValidationError struct {
	// Field is the field that failed validation ("input" or "groundTruth").
	Field string
	// Errors holds the individual field-level validation errors.
	Errors []FieldError
	// msg is the cached error message.
	msg string
}

// NewSchemaValidationError creates a new SchemaValidationError.
func NewSchemaValidationError(field string, errors []FieldError) *SchemaValidationError {
	// Build summary from first 3 errors
	limit := 3
	if len(errors) < limit {
		limit = len(errors)
	}
	msgs := make([]string, limit)
	for i := 0; i < limit; i++ {
		msgs[i] = errors[i].Message
	}
	summary := strings.Join(msgs, "; ")

	return &SchemaValidationError{
		Field:  field,
		Errors: errors,
		msg:    fmt.Sprintf("Validation failed for %s: %s", field, summary),
	}
}

// Error implements the error interface.
func (e *SchemaValidationError) Error() string {
	return e.msg
}

// ============================================================================
// Batch Validation Result
// ============================================================================

// ValidItem holds a validated item with its original index.
type ValidItem struct {
	// Index is the original index in the input slice.
	Index int `json:"index"`
	// Data is the validated data.
	Data any `json:"data"`
}

// InvalidItem holds an invalid item with its original index and errors.
type InvalidItem struct {
	// Index is the original index in the input slice.
	Index int `json:"index"`
	// Data is the invalid data.
	Data any `json:"data"`
	// Field is the field that failed validation ("input" or "groundTruth").
	Field string `json:"field"`
	// Errors holds the validation errors for this item.
	Errors []FieldError `json:"errors"`
}

// BatchValidationResult holds the result of validating multiple items.
type BatchValidationResult struct {
	// Valid holds items that passed validation.
	Valid []ValidItem `json:"valid"`
	// Invalid holds items that failed validation.
	Invalid []InvalidItem `json:"invalid"`
}

// ============================================================================
// Schema Update Validation Error
// ============================================================================

// SchemaUpdateValidationError is thrown when a schema update would invalidate existing items.
type SchemaUpdateValidationError struct {
	// FailingItems holds the items that would fail validation.
	FailingItems []InvalidItem
	// msg is the cached error message.
	msg string
}

// NewSchemaUpdateValidationError creates a new SchemaUpdateValidationError.
func NewSchemaUpdateValidationError(failingItems []InvalidItem) *SchemaUpdateValidationError {
	count := len(failingItems)
	return &SchemaUpdateValidationError{
		FailingItems: failingItems,
		msg:          fmt.Sprintf("Cannot update schema: %d existing item(s) would fail validation", count),
	}
}

// Error implements the error interface.
func (e *SchemaUpdateValidationError) Error() string {
	return e.msg
}
