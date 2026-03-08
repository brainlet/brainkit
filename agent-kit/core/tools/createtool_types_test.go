// Ported from: packages/core/src/tools/createtool-types.test.ts
package tools

import (
	"testing"
)

// NOTE: The TypeScript createtool-types.test.ts tests focus heavily on TypeScript
// type inference (expectTypeOf, type narrowing). Go does not have runtime type
// inference from schemas, so we port the runtime behavioral tests and skip
// the compile-time type assertion tests.

func TestCreateToolTypeImprovements(t *testing.T) {
	t.Run("should have execute function when provided", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "Test tool",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				name, _ := m["name"].(string)
				return map[string]any{"message": "Hello " + name}, nil
			},
		})

		if tool.Execute == nil {
			t.Error("expected execute to be defined")
		}
	})

	t.Run("should have properly typed return value based on output schema", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "typed-tool",
			Description: "Tool with typed output",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				name, _ := m["name"].(string)
				return map[string]any{
					"greeting":  "Hello " + name,
					"timestamp": 1234567890,
				}, nil
			},
		})

		result, err := tool.Execute(map[string]any{"name": "Alice"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if m["greeting"] != "Hello Alice" {
			t.Errorf("expected greeting='Hello Alice', got %v", m["greeting"])
		}
		if m["timestamp"] != 1234567890 {
			t.Errorf("expected timestamp=1234567890, got %v", m["timestamp"])
		}
	})

	t.Run("should return any when no output schema is provided", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "no-output-schema",
			Description: "Tool without output schema",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				return map[string]any{"anything": "goes", "nested": map[string]any{"value": 42}}, nil
			},
		})

		result, err := tool.Execute(map[string]any{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if m["anything"] != "goes" {
			t.Errorf("expected anything=goes, got %v", m["anything"])
		}
	})

	t.Run("should handle tools without execute function", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "no-execute",
			Description: "Tool without execute",
		})

		if tool.Execute != nil {
			t.Error("expected execute to be nil for tools without it")
		}
	})

	t.Run("should properly handle execute with calculator logic", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "fully-typed",
			Description: "Fully typed tool",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				operation, _ := m["operation"].(string)
				a, _ := m["a"].(float64)
				b, _ := m["b"].(float64)

				var result float64
				switch operation {
				case "add":
					result = a + b
				case "subtract":
					result = a - b
				case "multiply":
					result = a * b
				case "divide":
					result = a / b
				}

				return map[string]any{
					"result":    result,
					"operation": operation,
				}, nil
			},
		})

		output, err := tool.Execute(map[string]any{
			"operation": "add",
			"a":         float64(5),
			"b":         float64(3),
		}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m, ok := output.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", output)
		}
		if m["result"] != float64(8) {
			t.Errorf("expected result=8, got %v", m["result"])
		}
		if m["operation"] != "add" {
			t.Errorf("expected operation=add, got %v", m["operation"])
		}
	})
}

func TestIssue11381_ToolExecuteReturnTypeNarrowing(t *testing.T) {
	// Tests for the TypeScript issue where tool.execute return type narrowing
	// didn't work properly. In Go, we test the runtime behavior of validation
	// errors vs success results.

	t.Run("should allow inline narrowing with error check", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "full-name-finder",
			Description: "Finds a full name",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m, _ := inputData.(map[string]any)
				firstName, _ := m["firstName"].(string)
				return map[string]any{
					"fullName": firstName + " von der Burg",
				}, nil
			},
		})

		result, err := tool.Execute(map[string]any{"firstName": "Hans"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check if result is a validation error.
		if ve, ok := result.(*ValidationError); ok && ve.Error {
			t.Fatalf("unexpected validation error: %s", ve.Message)
		}

		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", result)
		}
		if m["fullName"] != "Hans von der Burg" {
			t.Errorf("expected fullName='Hans von der Burg', got %v", m["fullName"])
		}
	})

	t.Run("should correctly detect validation errors with inline check", func(t *testing.T) {
		// Create a tool with an input schema that requires minimum length.
		schema := newMockSchemaWithValidation(func(data any) SafeParseResult {
			m, ok := data.(map[string]any)
			if !ok {
				return SafeParseResult{
					Success: false,
					Error:   &SchemaError{Issues: []SchemaIssue{{Message: "Expected object"}}},
				}
			}
			name, _ := m["name"].(string)
			if len(name) < 5 {
				return SafeParseResult{
					Success: false,
					Error: &SchemaError{Issues: []SchemaIssue{
						{Path: []string{"name"}, Message: "String must contain at least 5 character(s)"},
					}},
				}
			}
			return SafeParseResult{Success: true, Data: data}
		})

		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "Test tool",
			InputSchema: schema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m := inputData.(map[string]any)
				return map[string]any{"result": m["name"]}, nil
			},
		})

		// Pass invalid input (too short).
		result, err := tool.Execute(map[string]any{"name": "ab"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ve, ok := result.(*ValidationError)
		if !ok {
			t.Fatalf("expected *ValidationError, got %T: %v", result, result)
		}
		if !ve.Error {
			t.Error("expected error=true")
		}
		if ve.Message == "" {
			t.Error("expected non-empty error message")
		}
	})
}
