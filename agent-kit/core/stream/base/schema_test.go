// Ported from: packages/core/src/stream/base/schema.test.ts
package base

import (
	"encoding/json"
	"testing"
)

func TestJSONSchema7MarshalJSON(t *testing.T) {
	t.Run("should marshal basic schema", func(t *testing.T) {
		schema := JSONSchema7{
			Type: "object",
			Properties: map[string]*JSONSchema7{
				"name": {Type: "string"},
				"age":  {Type: "number"},
			},
			Required: []string{"name"},
		}

		data, err := json.Marshal(schema)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if result["type"] != "object" {
			t.Errorf("expected type 'object', got %v", result["type"])
		}
		props, ok := result["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties to be a map")
		}
		if len(props) != 2 {
			t.Errorf("expected 2 properties, got %d", len(props))
		}
	})

	t.Run("should include Extra fields in marshaled output", func(t *testing.T) {
		schema := JSONSchema7{
			Type: "object",
			Extra: map[string]any{
				"description": "A custom schema",
				"minProperties": 1,
			},
		}

		data, err := json.Marshal(schema)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if result["description"] != "A custom schema" {
			t.Errorf("expected Extra field 'description', got %v", result["description"])
		}
		if result["type"] != "object" {
			t.Errorf("expected type 'object', got %v", result["type"])
		}
	})

	t.Run("should not override named fields with Extra", func(t *testing.T) {
		schema := JSONSchema7{
			Type: "object",
			Extra: map[string]any{
				"type": "string", // should not override
			},
		}

		data, err := json.Marshal(schema)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]any
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal result: %v", err)
		}

		if result["type"] != "object" {
			t.Errorf("Extra should not override named fields, got type=%v", result["type"])
		}
	})
}

func TestAsJsonSchema(t *testing.T) {
	t.Run("should return nil for nil schema", func(t *testing.T) {
		result := AsJsonSchema(nil)
		if result != nil {
			t.Error("expected nil for nil schema")
		}
	})

	t.Run("should return JSONSchema7 directly", func(t *testing.T) {
		schema := &JSONSchema7{Type: "object"}
		result := AsJsonSchema(schema)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "object" {
			t.Errorf("expected type 'object', got %q", result.Type)
		}
	})

	t.Run("should return nil for unrecognized type", func(t *testing.T) {
		result := AsJsonSchema("not a schema")
		if result != nil {
			t.Error("expected nil for unrecognized type")
		}
	})
}

func TestGetTransformedSchema(t *testing.T) {
	t.Run("should return nil for nil schema", func(t *testing.T) {
		result := GetTransformedSchema(nil)
		if result != nil {
			t.Error("expected nil for nil schema")
		}
	})

	t.Run("should wrap array schemas in elements wrapper", func(t *testing.T) {
		schema := &JSONSchema7{
			Type: "array",
			Items: &JSONSchema7{
				Type: "object",
				Properties: map[string]*JSONSchema7{
					"name": {Type: "string"},
				},
			},
		}

		result := GetTransformedSchema(schema)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OutputFormat != "array" {
			t.Errorf("expected output format 'array', got %q", result.OutputFormat)
		}
		if result.JSONSchema.Type != "object" {
			t.Errorf("expected wrapped schema type 'object', got %q", result.JSONSchema.Type)
		}

		elemProp, ok := result.JSONSchema.Properties["elements"]
		if !ok {
			t.Fatal("expected 'elements' property in wrapped schema")
		}
		if elemProp.Type != "array" {
			t.Errorf("expected elements type 'array', got %q", elemProp.Type)
		}
		if result.JSONSchema.AdditionalProperties == nil || *result.JSONSchema.AdditionalProperties != false {
			t.Error("expected additionalProperties to be false")
		}
	})

	t.Run("should wrap enum schemas in result wrapper", func(t *testing.T) {
		schema := &JSONSchema7{
			Type: "string",
			Enum: []any{"red", "green", "blue"},
		}

		result := GetTransformedSchema(schema)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OutputFormat != "enum" {
			t.Errorf("expected output format 'enum', got %q", result.OutputFormat)
		}
		if result.JSONSchema.Type != "object" {
			t.Errorf("expected wrapped schema type 'object', got %q", result.JSONSchema.Type)
		}

		resultProp, ok := result.JSONSchema.Properties["result"]
		if !ok {
			t.Fatal("expected 'result' property in wrapped schema")
		}
		if resultProp.Type != "string" {
			t.Errorf("expected result type 'string', got %q", resultProp.Type)
		}
		if len(resultProp.Enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(resultProp.Enum))
		}
	})

	t.Run("should pass object schemas through unchanged", func(t *testing.T) {
		schema := &JSONSchema7{
			Type: "object",
			Properties: map[string]*JSONSchema7{
				"name": {Type: "string"},
			},
		}

		result := GetTransformedSchema(schema)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.OutputFormat != "object" {
			t.Errorf("expected output format 'object', got %q", result.OutputFormat)
		}
		if result.JSONSchema.Type != "object" {
			t.Errorf("expected schema type 'object', got %q", result.JSONSchema.Type)
		}
	})

	t.Run("should default enum type to string if not specified", func(t *testing.T) {
		schema := &JSONSchema7{
			Enum: []any{"a", "b", "c"},
		}

		result := GetTransformedSchema(schema)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		resultProp := result.JSONSchema.Properties["result"]
		if resultProp.Type != "string" {
			t.Errorf("expected default enum type 'string', got %q", resultProp.Type)
		}
	})
}

func TestGetResponseFormat(t *testing.T) {
	t.Run("should return text format when schema is nil", func(t *testing.T) {
		result := GetResponseFormat(nil)
		if result.Type != ResponseFormatText {
			t.Errorf("expected type 'text', got %q", result.Type)
		}
		if result.Schema != nil {
			t.Error("expected nil schema for text format")
		}
	})

	t.Run("should return json format when schema is provided", func(t *testing.T) {
		schema := &JSONSchema7{
			Type: "object",
			Properties: map[string]*JSONSchema7{
				"name": {Type: "string"},
			},
		}

		result := GetResponseFormat(schema)
		if result.Type != ResponseFormatJSON {
			t.Errorf("expected type 'json', got %q", result.Type)
		}
		if result.Schema == nil {
			t.Fatal("expected non-nil schema for json format")
		}
	})
}

func TestSafeParseResult(t *testing.T) {
	t.Run("should represent successful validation", func(t *testing.T) {
		result := SafeParseResult{
			Success: true,
			Data:    "test",
		}
		if !result.Success {
			t.Error("expected success to be true")
		}
		if result.Data != "test" {
			t.Errorf("expected data 'test', got %v", result.Data)
		}
	})
}
