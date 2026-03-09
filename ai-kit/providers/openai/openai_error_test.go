// Ported from: packages/openai/src/openai-error.test.ts
package openai

import (
	"encoding/json"
	"testing"
)

func TestOpenAIErrorDataSchema(t *testing.T) {
	t.Run("should parse OpenRouter resource exhausted error", func(t *testing.T) {
		errorJSON := `{"error":{"message":"{\n  \"error\": {\n    \"code\": 429,\n    \"message\": \"Resource has been exhausted (e.g. check quota).\",\n    \"status\": \"RESOURCE_EXHAUSTED\"\n  }\n}\n","code":429}}`

		var parsed OpenAIErrorData
		err := json.Unmarshal([]byte(errorJSON), &parsed)
		if err != nil {
			t.Fatalf("failed to parse error data: %v", err)
		}

		expectedMsg := "{\n  \"error\": {\n    \"code\": 429,\n    \"message\": \"Resource has been exhausted (e.g. check quota).\",\n    \"status\": \"RESOURCE_EXHAUSTED\"\n  }\n}\n"
		if parsed.Error.Message != expectedMsg {
			t.Errorf("unexpected message:\ngot:  %q\nwant: %q", parsed.Error.Message, expectedMsg)
		}

		// Code should be 429 (parsed as a number)
		codeFloat, ok := parsed.Error.Code.(float64)
		if !ok {
			t.Fatalf("expected code to be float64, got %T", parsed.Error.Code)
		}
		if codeFloat != 429 {
			t.Errorf("expected code 429, got %v", codeFloat)
		}
	})
}
