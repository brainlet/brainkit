// Ported from: packages/core/src/stream/aisdk/v5/compat/validation.test.ts
package compat

import (
	"errors"
	"testing"
)

func TestNewValidationSuccess(t *testing.T) {
	t.Run("should create a successful result", func(t *testing.T) {
		result := NewValidationSuccess("hello")
		if !result.Success {
			t.Error("expected success to be true")
		}
		if result.Value != "hello" {
			t.Errorf("expected value 'hello', got %q", result.Value)
		}
		if result.Error != nil {
			t.Errorf("expected nil error, got %v", result.Error)
		}
	})

	t.Run("should work with int type", func(t *testing.T) {
		result := NewValidationSuccess(42)
		if !result.Success {
			t.Error("expected success to be true")
		}
		if result.Value != 42 {
			t.Errorf("expected value 42, got %d", result.Value)
		}
	})
}

func TestNewValidationFailure(t *testing.T) {
	t.Run("should create a failed result", func(t *testing.T) {
		err := errors.New("validation failed")
		result := NewValidationFailure[string](err)
		if result.Success {
			t.Error("expected success to be false")
		}
		if result.Error == nil {
			t.Fatal("expected non-nil error")
		}
		if result.Error.Error() != "validation failed" {
			t.Errorf("expected error message 'validation failed', got %q", result.Error.Error())
		}
	})
}

func TestTypeValidationError(t *testing.T) {
	t.Run("should format error message correctly", func(t *testing.T) {
		err := &TypeValidationError{
			Value: "test-value",
			Cause: "invalid type",
		}
		expected := "type validation error: invalid type (value: test-value)"
		if err.Error() != expected {
			t.Errorf("expected %q, got %q", expected, err.Error())
		}
	})

	t.Run("should implement error interface", func(t *testing.T) {
		var err error = &TypeValidationError{
			Value: 123,
			Cause: "expected string",
		}
		if err == nil {
			t.Fatal("expected non-nil error")
		}
	})
}

// mockSchema implements Schema[T] for testing.
type mockSchema[T any] struct {
	validateFn func(value any) (ValidationResult[T], error)
}

func (m *mockSchema[T]) Validate(value any) (ValidationResult[T], error) {
	return m.validateFn(value)
}

func TestSafeValidateTypes(t *testing.T) {
	t.Run("should pass through when schema is nil", func(t *testing.T) {
		result := SafeValidateTypes[string]("test", nil)
		if !result.Success {
			t.Error("expected success when schema is nil")
		}
	})

	t.Run("should return success when schema validates", func(t *testing.T) {
		schema := &mockSchema[string]{
			validateFn: func(value any) (ValidationResult[string], error) {
				return NewValidationSuccess(value.(string)), nil
			},
		}
		result := SafeValidateTypes("hello", schema)
		if !result.Success {
			t.Error("expected success")
		}
		if result.Value != "hello" {
			t.Errorf("expected 'hello', got %q", result.Value)
		}
	})

	t.Run("should return failure when schema Validate returns error", func(t *testing.T) {
		schema := &mockSchema[string]{
			validateFn: func(value any) (ValidationResult[string], error) {
				return ValidationResult[string]{}, errors.New("schema error")
			},
		}
		result := SafeValidateTypes("bad", schema)
		if result.Success {
			t.Error("expected failure")
		}
		if result.Error == nil {
			t.Fatal("expected non-nil error")
		}
	})

	t.Run("should return TypeValidationError when schema result is not success", func(t *testing.T) {
		schema := &mockSchema[string]{
			validateFn: func(value any) (ValidationResult[string], error) {
				return ValidationResult[string]{Success: false}, nil
			},
		}
		result := SafeValidateTypes("bad", schema)
		if result.Success {
			t.Error("expected failure")
		}
		var tve *TypeValidationError
		if !errors.As(result.Error, &tve) {
			t.Fatal("expected TypeValidationError")
		}
		if tve.Value != "bad" {
			t.Errorf("expected value 'bad', got %v", tve.Value)
		}
	})
}
