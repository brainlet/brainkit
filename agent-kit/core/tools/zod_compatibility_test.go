// Ported from: packages/core/src/tools/zod-compatibility.test.ts
package tools

import (
	"testing"
)

// NOTE: The TypeScript zod-compatibility.test.ts tests Zod v3/v4 schema
// compatibility. In Go, there is no Zod library — schemas use the
// SafeParser interface instead. These tests verify that tools work
// correctly with different schema implementations (mock SafeParser),
// mirroring the TS concept of "multiple schema library versions".

func TestZodV3AndV4Compatibility(t *testing.T) {
	t.Run("Type Compatibility", func(t *testing.T) {
		t.Run("should accept schema implementing SafeParser (v3 equivalent)", func(t *testing.T) {
			schema := alwaysPassSchema()
			tool := CreateTool(ToolAction{
				ID:          "v3-tool",
				Description: "Tool with SafeParser schema (v3 equivalent)",
				InputSchema: schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					name, _ := m["name"].(string)
					age, _ := m["age"].(float64)
					return map[string]any{
						"message": name + " is great",
						"age":    age,
					}, nil
				},
			})

			if tool == nil {
				t.Fatal("expected tool to be defined")
			}
			if tool.ID != "v3-tool" {
				t.Errorf("expected ID=v3-tool, got %s", tool.ID)
			}
			if tool.InputSchema == nil {
				t.Error("expected InputSchema to be defined")
			}
		})

		t.Run("should accept another schema implementing SafeParser (v4 equivalent)", func(t *testing.T) {
			schema := alwaysPassSchema()
			tool := CreateTool(ToolAction{
				ID:          "v4-tool",
				Description: "Tool with SafeParser schema (v4 equivalent)",
				InputSchema: schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					input, _ := m["input"].(string)
					// Reverse the string.
					runes := []rune(input)
					for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
						runes[i], runes[j] = runes[j], runes[i]
					}
					return map[string]any{"output": string(runes)}, nil
				},
			})

			if tool == nil {
				t.Fatal("expected tool to be defined")
			}
			if tool.ID != "v4-tool" {
				t.Errorf("expected ID=v4-tool, got %s", tool.ID)
			}
			if tool.InputSchema == nil {
				t.Error("expected InputSchema to be defined")
			}
		})

		t.Run("should validate that SafeParser schemas have required methods", func(t *testing.T) {
			schema := alwaysPassSchema()

			// Verify the schema implements SafeParser.
			parser, ok := schema.Schema.(SafeParser)
			if !ok {
				t.Fatal("expected schema to implement SafeParser")
			}

			result := parser.SafeParse(map[string]any{"test": "value"})
			if !result.Success {
				t.Error("expected SafeParse to succeed")
			}
		})
	})

	t.Run("Runtime Behavior", func(t *testing.T) {
		t.Run("should execute tools with schema correctly - sum", func(t *testing.T) {
			schema := alwaysPassSchema()
			tool := CreateTool(ToolAction{
				ID:          "runtime-v3",
				Description: "Runtime test with v3-like schema",
				InputSchema: schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					x, _ := m["x"].(float64)
					y, _ := m["y"].(float64)
					return map[string]any{"sum": x + y}, nil
				},
			})

			result, err := tool.Execute(map[string]any{"x": float64(5), "y": float64(3)}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			m, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T", result)
			}
			if m["sum"] != float64(8) {
				t.Errorf("expected sum=8, got %v", m["sum"])
			}
		})

		t.Run("should execute tools with schema correctly - string length", func(t *testing.T) {
			schema := alwaysPassSchema()
			tool := CreateTool(ToolAction{
				ID:          "runtime-v4",
				Description: "Runtime test with v4-like schema",
				InputSchema: schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					text, _ := m["text"].(string)
					return map[string]any{"length": len(text)}, nil
				},
			})

			result, err := tool.Execute(map[string]any{"text": "hello"}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			m, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("expected map, got %T", result)
			}
			if m["length"] != 5 {
				t.Errorf("expected length=5, got %v", m["length"])
			}
		})

		t.Run("should handle validation with failing schema", func(t *testing.T) {
			failSchema := alwaysFailSchema([]SchemaIssue{
				{Path: []string{"email"}, Message: "Invalid email"},
			})

			tool := CreateTool(ToolAction{
				ID:          "validation-fail",
				Description: "Validation test with failing schema",
				InputSchema: failSchema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					return map[string]any{"validated": true}, nil
				},
			})

			result, err := tool.Execute(map[string]any{"email": "not-an-email"}, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			ve, ok := result.(*ValidationError)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T: %v", result, result)
			}
			if !ve.Error {
				t.Error("expected validation error")
			}
		})
	})

	t.Run("Regression Tests for Issue 8060", func(t *testing.T) {
		t.Run("should create tools with any SafeParser schema without errors", func(t *testing.T) {
			schema := alwaysPassSchema()
			tool := CreateTool(ToolAction{
				ID:          "test-tool",
				Description: "Reverse the input string",
				InputSchema: schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					input, _ := m["input"].(string)
					runes := []rune(input)
					for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
						runes[i], runes[j] = runes[j], runes[i]
					}
					return map[string]any{"output": string(runes)}, nil
				},
			})

			if tool == nil {
				t.Fatal("expected tool to be defined")
			}
			if tool.ID != "test-tool" {
				t.Errorf("expected ID=test-tool, got %s", tool.ID)
			}
			if tool.Description != "Reverse the input string" {
				t.Errorf("expected correct description, got %s", tool.Description)
			}
		})

		t.Run("should handle mixed schema versions in the same codebase", func(t *testing.T) {
			v3Schema := alwaysPassSchema()
			v4Schema := alwaysPassSchema()

			v3Tool := CreateTool(ToolAction{
				ID:          "mixed-v3",
				Description: "Uses v3",
				InputSchema: v3Schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					return map[string]any{"result": m["v3Input"]}, nil
				},
			})

			v4Tool := CreateTool(ToolAction{
				ID:          "mixed-v4",
				Description: "Uses v4",
				InputSchema: v4Schema,
				Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
					m, _ := inputData.(map[string]any)
					return map[string]any{"result": m["v4Input"]}, nil
				},
			})

			if v3Tool == nil {
				t.Fatal("expected v3Tool to be defined")
			}
			if v4Tool == nil {
				t.Fatal("expected v4Tool to be defined")
			}
			if v3Tool.ID != "mixed-v3" {
				t.Errorf("expected v3 ID, got %s", v3Tool.ID)
			}
			if v4Tool.ID != "mixed-v4" {
				t.Errorf("expected v4 ID, got %s", v4Tool.ID)
			}
		})
	})
}
