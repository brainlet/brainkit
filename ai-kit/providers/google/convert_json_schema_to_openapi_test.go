// Ported from: packages/google/src/convert-json-schema-to-openapi-schema.test.ts
package google

import (
	"reflect"
	"testing"
)

func TestConvertJSONSchemaToOpenAPISchema(t *testing.T) {
	t.Run("should remove additionalProperties and $schema", func(t *testing.T) {
		input := map[string]any{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "number"},
			},
			"additionalProperties": false,
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
				"age":  map[string]any{"type": "number"},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should remove additionalProperties object from nested object schemas", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keys": map[string]any{
					"type":                 "object",
					"additionalProperties": map[string]any{"type": "string"},
					"description":          "Description for the key",
				},
			},
			"additionalProperties": false,
			"$schema":              "http://json-schema.org/draft-07/schema#",
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keys": map[string]any{
					"type":        "object",
					"description": "Description for the key",
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle nested objects and arrays", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"users": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":   map[string]any{"type": "number"},
							"name": map[string]any{"type": "string"},
						},
						"additionalProperties": false,
					},
				},
			},
			"additionalProperties": false,
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"users": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":   map[string]any{"type": "number"},
							"name": map[string]any{"type": "string"},
						},
					},
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert const to enum with a single value", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"const": "active"},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"status": map[string]any{"enum": []any{"active"}},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle allOf, anyOf, and oneOf", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"allOfProp": map[string]any{
					"allOf": []any{
						map[string]any{"type": "string"},
						map[string]any{"minLength": 5},
					},
				},
				"anyOfProp": map[string]any{
					"anyOf": []any{
						map[string]any{"type": "string"},
						map[string]any{"type": "number"},
					},
				},
				"oneOfProp": map[string]any{
					"oneOf": []any{
						map[string]any{"type": "boolean"},
						map[string]any{"type": "null"},
					},
				},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"allOfProp": map[string]any{
					"allOf": []any{
						map[string]any{"type": "string"},
						map[string]any{"minLength": 5},
					},
				},
				"anyOfProp": map[string]any{
					"anyOf": []any{
						map[string]any{"type": "string"},
						map[string]any{"type": "number"},
					},
				},
				"oneOfProp": map[string]any{
					"oneOf": []any{
						map[string]any{"type": "boolean"},
						map[string]any{"type": "null"},
					},
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert format date-time", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timestamp": map[string]any{"type": "string", "format": "date-time"},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"timestamp": map[string]any{"type": "string", "format": "date-time"},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle required properties", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "number"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"id"},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "number"},
				"name": map[string]any{"type": "string"},
			},
			"required": []any{"id"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert deeply nested const to enum", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"deeplyNested": map[string]any{
							"anyOf": []any{
								map[string]any{
									"type": "object",
									"properties": map[string]any{
										"value": map[string]any{"const": "specific value"},
									},
								},
								map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"deeplyNested": map[string]any{
							"anyOf": []any{
								map[string]any{
									"type": "object",
									"properties": map[string]any{
										"value": map[string]any{"enum": []any{"specific value"}},
									},
								},
								map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle null type correctly", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nullableField": map[string]any{
					"type": []any{"string", "null"},
				},
				"explicitNullField": map[string]any{
					"type": "null",
				},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nullableField": map[string]any{
					"anyOf":    []any{map[string]any{"type": "string"}},
					"nullable": true,
				},
				"explicitNullField": map[string]any{
					"type": "null",
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle descriptions", func(t *testing.T) {
		input := map[string]any{
			"type":        "object",
			"description": "A user object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "number", "description": "The user ID"},
				"name": map[string]any{"type": "string", "description": "The user's full name"},
				"email": map[string]any{
					"type":        "string",
					"format":      "email",
					"description": "The user's email address",
				},
			},
			"required": []any{"id", "name"},
		}
		expected := map[string]any{
			"type":        "object",
			"description": "A user object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "number", "description": "The user ID"},
				"name": map[string]any{"type": "string", "description": "The user's full name"},
				"email": map[string]any{
					"type":        "string",
					"format":      "email",
					"description": "The user's email address",
				},
			},
			"required": []any{"id", "name"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should return nil for empty object schemas at root level", func(t *testing.T) {
		emptyObjectSchemas := []map[string]any{
			{"type": "object"},
			{"type": "object", "properties": map[string]any{}},
		}
		for _, schema := range emptyObjectSchemas {
			result := ConvertJSONSchemaToOpenAPISchema(schema, true)
			if result != nil {
				t.Errorf("expected nil for empty object schema at root, got %v", result)
			}
		}
	})

	t.Run("should preserve nested empty object schemas to avoid breaking required array validation", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":           map[string]any{"type": "string", "description": "URL to navigate to"},
				"launchOptions": map[string]any{"type": "object", "description": "PuppeteerJS LaunchOptions"},
				"allowDangerous": map[string]any{
					"type":        "boolean",
					"description": "Allow dangerous options",
				},
			},
			"required": []any{"url", "launchOptions"},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url":           map[string]any{"type": "string", "description": "URL to navigate to"},
				"launchOptions": map[string]any{"type": "object", "description": "PuppeteerJS LaunchOptions"},
				"allowDangerous": map[string]any{
					"type":        "boolean",
					"description": "Allow dangerous options",
				},
			},
			"required": []any{"url", "launchOptions"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should preserve nested empty object schemas without descriptions", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"options": map[string]any{"type": "object"},
			},
			"required": []any{"options"},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"options": map[string]any{"type": "object"},
			},
			"required": []any{"options"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle non-empty object schemas", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{"type": "string"},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert string enum properties", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{
					"type": "string",
					"enum": []any{"text", "code", "image"},
				},
			},
			"required":             []any{"kind"},
			"additionalProperties": false,
			"$schema":              "http://json-schema.org/draft-07/schema#",
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{
					"type": "string",
					"enum": []any{"text", "code", "image"},
				},
			},
			"required": []any{"kind"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert nullable string enum", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"fieldD": map[string]any{
					"anyOf": []any{
						map[string]any{
							"type": "string",
							"enum": []any{"a", "b", "c"},
						},
						map[string]any{
							"type": "null",
						},
					},
				},
			},
			"required":             []any{"fieldD"},
			"additionalProperties": false,
			"$schema":              "http://json-schema.org/draft-07/schema#",
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"fieldD": map[string]any{
					"nullable": true,
					"type":     "string",
					"enum":     []any{"a", "b", "c"},
				},
			},
			"required": []any{"fieldD"},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should handle type arrays with multiple non-null types plus null", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"multiTypeField": map[string]any{
					"type": []any{"string", "number", "null"},
				},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"multiTypeField": map[string]any{
					"anyOf":    []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}},
					"nullable": true,
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})

	t.Run("should convert type arrays without null to anyOf", func(t *testing.T) {
		input := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"multiTypeField": map[string]any{
					"type": []any{"string", "number"},
				},
			},
		}
		expected := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"multiTypeField": map[string]any{
					"anyOf": []any{map[string]any{"type": "string"}, map[string]any{"type": "number"}},
				},
			},
		}
		result := ConvertJSONSchemaToOpenAPISchema(input, false)
		assertDeepEqual(t, expected, result)
	})
}

// assertDeepEqual compares two map[string]any structures using reflect.DeepEqual
// and reports a helpful error message on mismatch.
func assertDeepEqual(t *testing.T, expected, actual any) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("mismatch:\nexpected: %#v\n  actual: %#v", expected, actual)
	}
}
