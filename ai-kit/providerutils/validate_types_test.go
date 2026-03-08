// Ported from: packages/provider-utils/src/validate-types.test.ts
package providerutils

import (
	"errors"
	"testing"
)

func TestValidateTypes_ValidInput(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	schema := &Schema[Person]{
		Validate: func(value interface{}) (*ValidationResult[Person], error) {
			m, ok := value.(map[string]interface{})
			if !ok {
				return &ValidationResult[Person]{Success: false, Error: errors.New("not a map")}, nil
			}
			name, _ := m["name"].(string)
			ageFloat, _ := m["age"].(float64)
			if name == "" {
				return &ValidationResult[Person]{Success: false, Error: errors.New("missing name")}, nil
			}
			return &ValidationResult[Person]{
				Success: true,
				Value:   Person{Name: name, Age: int(ageFloat)},
			}, nil
		},
	}

	input := map[string]interface{}{"name": "John", "age": float64(30)}
	result, err := ValidateTypes(input, schema)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != "John" || result.Age != 30 {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestValidateTypes_InvalidInput(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	schema := &Schema[Person]{
		Validate: func(value interface{}) (*ValidationResult[Person], error) {
			m, ok := value.(map[string]interface{})
			if !ok {
				return &ValidationResult[Person]{Success: false, Error: errors.New("not a map")}, nil
			}
			_, nameOk := m["name"].(string)
			_, ageOk := m["age"].(float64)
			if !nameOk || !ageOk {
				return &ValidationResult[Person]{Success: false, Error: errors.New("invalid input")}, nil
			}
			return &ValidationResult[Person]{
				Success: true,
				Value:   Person{Name: m["name"].(string), Age: int(m["age"].(float64))},
			}, nil
		},
	}

	input := map[string]interface{}{"name": "John", "age": "30"}
	_, err := ValidateTypes(input, schema)
	if err == nil {
		t.Fatal("expected TypeValidationError")
	}
	if !IsTypeValidationError(err) {
		t.Errorf("expected TypeValidationError, got %T", err)
	}
}

func TestSafeValidateTypes_ValidInput(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	schema := &Schema[Person]{
		Validate: func(value interface{}) (*ValidationResult[Person], error) {
			m, ok := value.(map[string]interface{})
			if !ok {
				return &ValidationResult[Person]{Success: false, Error: errors.New("not a map")}, nil
			}
			name, _ := m["name"].(string)
			ageFloat, _ := m["age"].(float64)
			return &ValidationResult[Person]{
				Success: true,
				Value:   Person{Name: name, Age: int(ageFloat)},
			}, nil
		},
	}

	input := map[string]interface{}{"name": "John", "age": float64(30)}
	result := SafeValidateTypes(input, schema)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	if result.Value.Name != "John" || result.Value.Age != 30 {
		t.Errorf("unexpected value: %+v", result.Value)
	}
}

func TestSafeValidateTypes_InvalidInput(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	schema := &Schema[Person]{
		Validate: func(value interface{}) (*ValidationResult[Person], error) {
			return &ValidationResult[Person]{Success: false, Error: errors.New("invalid")}, nil
		},
	}

	input := map[string]interface{}{"name": "John", "age": "30"}
	result := SafeValidateTypes(input, schema)
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.Error == nil {
		t.Fatal("expected error to be set")
	}
}
