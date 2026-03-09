// Ported from: packages/anthropic/src/anthropic-error.test.ts
package anthropic

import (
	"encoding/json"
	"testing"
)

func TestAnthropicErrorDataSchema(t *testing.T) {
	t.Run("should parse overloaded error", func(t *testing.T) {
		input := `{
			"type": "error",
			"error": {
				"details": null,
				"type": "overloaded_error",
				"message": "Overloaded"
			}
		}`

		var data AnthropicErrorData
		err := json.Unmarshal([]byte(input), &data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if data.Type != "error" {
			t.Errorf("expected type 'error', got %q", data.Type)
		}
		if data.Error.Type != "overloaded_error" {
			t.Errorf("expected error type 'overloaded_error', got %q", data.Error.Type)
		}
		if data.Error.Message != "Overloaded" {
			t.Errorf("expected error message 'Overloaded', got %q", data.Error.Message)
		}
	})
}
