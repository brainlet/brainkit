// Ported from: packages/ai/src/prompt/create-tool-model-output.test.ts
package prompt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateToolModelOutput(t *testing.T) {
	t.Run("error cases", func(t *testing.T) {
		t.Run("should return error-text with string value when errorMode is text and output is string", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, "Error message", nil, "text")
			assert.NoError(t, err)
			assert.Equal(t, "error-text", result.Type)
			assert.Equal(t, "Error message", result.Value)
		})

		t.Run("should return error-text with JSON stringified value when errorMode is text and output is not string", func(t *testing.T) {
			output := map[string]interface{}{"error": "Something went wrong", "code": float64(500)}
			result, err := CreateToolModelOutput("123", nil, output, nil, "text")
			assert.NoError(t, err)
			assert.Equal(t, "error-text", result.Type)
			// The JSON output includes the fields
			assert.Contains(t, result.Value, "Something went wrong")
		})

		t.Run("should handle undefined output in error text case", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, nil, nil, "text")
			assert.NoError(t, err)
			assert.Equal(t, "error-text", result.Type)
			assert.Equal(t, "unknown error", result.Value)
		})

		t.Run("should use nil for undefined output in error json case", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, nil, nil, "json")
			assert.NoError(t, err)
			assert.Equal(t, "error-json", result.Type)
			assert.Nil(t, result.Value)
		})
	})

	t.Run("tool with toModelOutput", func(t *testing.T) {
		t.Run("should use tool.toModelOutput when available", func(t *testing.T) {
			tool := &ToolWithModelOutput{
				ToModelOutput: func(args ToModelOutputArgs) (ToolResultOutput, error) {
					return ToolResultOutput{
						Type:  "text",
						Value: "Custom output: " + args.Output.(string),
					}, nil
				},
			}

			result, err := CreateToolModelOutput("123", nil, "test output", tool, "none")
			assert.NoError(t, err)
			assert.Equal(t, "text", result.Type)
			assert.Equal(t, "Custom output: test output", result.Value)
		})
	})

	t.Run("string output without toModelOutput", func(t *testing.T) {
		t.Run("should return text type for string output", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, "Simple string output", nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "text", result.Type)
			assert.Equal(t, "Simple string output", result.Value)
		})

		t.Run("should return text type for empty string", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, "", nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "text", result.Type)
			assert.Equal(t, "", result.Value)
		})
	})

	t.Run("non-string output without toModelOutput", func(t *testing.T) {
		t.Run("should return json type for object output", func(t *testing.T) {
			output := map[string]interface{}{
				"result": "success",
				"data":   []interface{}{1, 2, 3},
			}
			result, err := CreateToolModelOutput("123", nil, output, nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "json", result.Type)
			assert.Equal(t, output, result.Value)
		})

		t.Run("should return json type for number output", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, 42, nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "json", result.Type)
			assert.Equal(t, 42, result.Value)
		})

		t.Run("should return json type for boolean output", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, true, nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "json", result.Type)
			assert.Equal(t, true, result.Value)
		})

		t.Run("should return json type for nil output", func(t *testing.T) {
			result, err := CreateToolModelOutput("123", nil, nil, nil, "none")
			assert.NoError(t, err)
			assert.Equal(t, "json", result.Type)
			assert.Nil(t, result.Value)
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should prioritize errorMode over tool.toModelOutput", func(t *testing.T) {
			tool := &ToolWithModelOutput{
				ToModelOutput: func(args ToModelOutputArgs) (ToolResultOutput, error) {
					return ToolResultOutput{
						Type:  "text",
						Value: "This should not be called",
					}, nil
				},
			}

			result, err := CreateToolModelOutput("123", nil, "Error occurred", tool, "text")
			assert.NoError(t, err)
			assert.Equal(t, "error-text", result.Type)
			assert.Equal(t, "Error occurred", result.Value)
		})
	})

	t.Run("arguments", func(t *testing.T) {
		t.Run("should pass toolCallId to tool.toModelOutput", func(t *testing.T) {
			tool := &ToolWithModelOutput{
				ToModelOutput: func(args ToModelOutputArgs) (ToolResultOutput, error) {
					return ToolResultOutput{
						Type:  "text",
						Value: "Tool call ID: " + args.ToolCallID,
					}, nil
				},
			}

			result, err := CreateToolModelOutput("2344", nil, "test", tool, "none")
			assert.NoError(t, err)
			assert.Equal(t, "text", result.Type)
			assert.Equal(t, "Tool call ID: 2344", result.Value)
		})
	})
}
