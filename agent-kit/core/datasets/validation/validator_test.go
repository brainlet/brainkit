// Ported from: packages/core/src/datasets/validation/validator.test.ts
package validation

import (
	"testing"
)

func TestSchemaValidator_ValidateValid(t *testing.T) {
	sv := NewSchemaValidator()
	schema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []any{"name"},
	}

	// Valid data should pass.
	err := sv.Validate(map[string]any{"name": "Alice", "age": 30.0}, schema, "input", "test:1")
	if err != nil {
		t.Fatalf("expected nil error for valid data, got: %v", err)
	}
}

func TestSchemaValidator_ValidateInvalid(t *testing.T) {
	sv := NewSchemaValidator()
	schema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []any{"name"},
	}

	// Missing required field "name" should fail.
	err := sv.Validate(map[string]any{"age": 30.0}, schema, "input", "test:2")
	if err == nil {
		t.Fatal("expected validation error for missing required field, got nil")
	}

	schemaErr, ok := err.(*SchemaValidationError)
	if !ok {
		t.Fatalf("expected *SchemaValidationError, got %T", err)
	}
	if schemaErr.Field != "input" {
		t.Errorf("expected field 'input', got %q", schemaErr.Field)
	}
	if len(schemaErr.Errors) == 0 {
		t.Fatal("expected at least one field error")
	}
}

func TestSchemaValidator_ValidateTypeError(t *testing.T) {
	sv := NewSchemaValidator()
	schema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
	}

	// Wrong type for "name" should fail.
	err := sv.Validate(map[string]any{"name": 42.0}, schema, "input", "test:3")
	if err == nil {
		t.Fatal("expected validation error for wrong type, got nil")
	}
}

func TestSchemaValidator_ClearCache(t *testing.T) {
	sv := NewSchemaValidator()
	schema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
	}

	// Compile and cache.
	_ = sv.Validate(map[string]any{"x": "hello"}, schema, "input", "cache-key")

	// Verify cache has the entry.
	sv.mu.RLock()
	_, cached := sv.cache["cache-key"]
	sv.mu.RUnlock()
	if !cached {
		t.Fatal("expected validator to be cached")
	}

	// Clear and verify.
	sv.ClearCache("cache-key")
	sv.mu.RLock()
	_, cached = sv.cache["cache-key"]
	sv.mu.RUnlock()
	if cached {
		t.Fatal("expected cache to be cleared")
	}
}

func TestSchemaValidator_ValidateBatch(t *testing.T) {
	sv := NewSchemaValidator()
	inputSchema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"prompt": map[string]any{"type": "string"},
		},
		"required": []any{"prompt"},
	}

	items := []BatchItem{
		{Input: map[string]any{"prompt": "hello"}},           // valid
		{Input: map[string]any{}},                             // invalid: missing "prompt"
		{Input: map[string]any{"prompt": "world"}},            // valid
		{Input: map[string]any{"prompt": 42.0}},               // invalid: wrong type
	}

	result := sv.ValidateBatch(items, inputSchema, nil, "batch", 10)

	if len(result.Valid) != 2 {
		t.Errorf("expected 2 valid items, got %d", len(result.Valid))
	}
	if len(result.Invalid) != 2 {
		t.Errorf("expected 2 invalid items, got %d", len(result.Invalid))
	}

	// Verify valid indices.
	if result.Valid[0].Index != 0 || result.Valid[1].Index != 2 {
		t.Errorf("expected valid indices [0, 2], got [%d, %d]", result.Valid[0].Index, result.Valid[1].Index)
	}

	// Verify invalid indices.
	if result.Invalid[0].Index != 1 || result.Invalid[1].Index != 3 {
		t.Errorf("expected invalid indices [1, 3], got [%d, %d]", result.Invalid[0].Index, result.Invalid[1].Index)
	}
}

func TestSchemaValidator_ValidateBatchMaxErrors(t *testing.T) {
	sv := NewSchemaValidator()
	inputSchema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
		"required": []any{"x"},
	}

	// All invalid items.
	items := []BatchItem{
		{Input: map[string]any{}},
		{Input: map[string]any{}},
		{Input: map[string]any{}},
		{Input: map[string]any{}},
		{Input: map[string]any{}},
	}

	// maxErrors = 2 should stop after 2 invalid items.
	result := sv.ValidateBatch(items, inputSchema, nil, "max-err", 2)
	if len(result.Invalid) != 2 {
		t.Errorf("expected 2 invalid items (maxErrors), got %d", len(result.Invalid))
	}
}

func TestSchemaValidator_NilSchema(t *testing.T) {
	sv := NewSchemaValidator()

	// nil schema should skip validation (return nil).
	err := sv.Validate(map[string]any{"anything": true}, nil, "input", "nil-schema")
	if err != nil {
		t.Fatalf("expected nil for nil schema, got: %v", err)
	}
}

func TestSchemaValidator_Singleton(t *testing.T) {
	// GetSchemaValidator should always return the same instance.
	v1 := GetSchemaValidator()
	v2 := GetSchemaValidator()
	if v1 != v2 {
		t.Fatal("expected singleton instances to be identical")
	}
}

func TestSchemaValidator_GroundTruthValidation(t *testing.T) {
	sv := NewSchemaValidator()
	outputSchema := JSONSchema7{
		"type": "object",
		"properties": map[string]any{
			"answer": map[string]any{"type": "string"},
		},
		"required": []any{"answer"},
	}

	items := []BatchItem{
		{Input: "x", GroundTruth: map[string]any{"answer": "yes"}}, // valid
		{Input: "y", GroundTruth: map[string]any{}},                 // invalid: missing "answer"
		{Input: "z", GroundTruth: nil},                              // nil groundTruth: skip validation
	}

	result := sv.ValidateBatch(items, nil, outputSchema, "gt", 10)
	if len(result.Valid) != 2 {
		t.Errorf("expected 2 valid (items 0 and 2), got %d", len(result.Valid))
	}
	if len(result.Invalid) != 1 {
		t.Errorf("expected 1 invalid (item 1), got %d", len(result.Invalid))
	}
	if len(result.Invalid) > 0 && result.Invalid[0].Field != "groundTruth" {
		t.Errorf("expected field 'groundTruth', got %q", result.Invalid[0].Field)
	}
}
