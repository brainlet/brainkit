// Ported from: packages/core/src/error/index.test.ts
package mastraerror

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMastraError_BaseClass(t *testing.T) {
	sampleContext := map[string]any{
		"fileName":   "test.ts",
		"lineNumber": float64(42),
	}
	sampleErrorDefinition := ErrorDefinition{
		ID:       "BASE_TEST_001",
		Domain:   ErrorDomainAgent,
		Category: ErrorCategoryUnknown,
		Details:  sampleContext,
	}

	t.Run("should create a base error with definition and context", func(t *testing.T) {
		err := NewMastraError(sampleErrorDefinition)

		// Check it implements error interface
		var _ error = err

		if err.ID() != "BASE_TEST_001" {
			t.Errorf("expected ID 'BASE_TEST_001', got %q", err.ID())
		}
		// Since there's no text field in the definition, message will be "Unknown error"
		if err.Message() != "Unknown error" {
			t.Errorf("expected message 'Unknown error', got %q", err.Message())
		}
		if err.Domain() != ErrorDomainAgent {
			t.Errorf("expected domain %q, got %q", ErrorDomainAgent, err.Domain())
		}
		if err.Category() != ErrorCategoryUnknown {
			t.Errorf("expected category %q, got %q", ErrorCategoryUnknown, err.Category())
		}
	})

	t.Run("should use error message from provided cause", func(t *testing.T) {
		cause := &SerializableError{
			message:        "Test error message",
			name:           "Error",
			serializeStack: false,
			extra:          make(map[string]any),
		}
		err := NewMastraError(sampleErrorDefinition, cause)
		if err.Message() != "Test error message" {
			t.Errorf("expected message 'Test error message', got %q", err.Message())
		}
	})

	t.Run("should handle object errors with details preserved", func(t *testing.T) {
		objectError := map[string]any{
			"type":            "error",
			"sequence_number": float64(2),
			"error": map[string]any{
				"type":    "invalid_request_error",
				"code":    "context_length_exceeded",
				"message": "Your input exceeds the context window of this model. Please adjust your input and try again.",
				"param":   "input",
			},
		}
		err := NewMastraError(sampleErrorDefinition, objectError)
		// The message should contain the stringified object details.
		msg := err.Message()
		if !strings.Contains(msg, "context_length_exceeded") {
			t.Errorf("expected message to contain 'context_length_exceeded', got %q", msg)
		}
		if !strings.Contains(msg, "Your input exceeds the context window") {
			t.Errorf("expected message to contain 'Your input exceeds the context window', got %q", msg)
		}
		if !strings.Contains(msg, "sequence_number") {
			t.Errorf("expected message to contain 'sequence_number', got %q", msg)
		}
		if !strings.Contains(msg, "invalid_request_error") {
			t.Errorf("expected message to contain 'invalid_request_error', got %q", msg)
		}
	})

	t.Run("should create a base error with a cause", func(t *testing.T) {
		cause := &SerializableError{
			message:        "Original cause",
			name:           "Error",
			serializeStack: false,
			extra:          make(map[string]any),
		}
		err := NewMastraError(sampleErrorDefinition, cause)
		if err.Cause() == nil {
			t.Fatal("expected cause to be set")
		}
		if err.Cause().Message() != "Original cause" {
			t.Errorf("expected cause message 'Original cause', got %q", err.Cause().Message())
		}
	})

	t.Run("toJSON methods for Base MastraError", func(t *testing.T) {
		t.Run("should correctly serialize to JSON with ToJSON and ToJSONDetails", func(t *testing.T) {
			cause := &SerializableError{
				message:        "Original cause",
				name:           "Error",
				serializeStack: false,
				extra:          make(map[string]any),
			}
			err := NewMastraError(sampleErrorDefinition, cause)

			// Since we have a cause, the message should be from the cause.
			if err.Message() != "Original cause" {
				t.Errorf("expected message 'Original cause', got %q", err.Message())
			}

			jsonDetails := err.ToJSONDetails()
			if jsonDetails.Message != "Original cause" {
				t.Errorf("expected jsonDetails.Message 'Original cause', got %q", jsonDetails.Message)
			}
			if jsonDetails.Domain != ErrorDomainAgent {
				t.Errorf("expected jsonDetails.Domain %q, got %q", ErrorDomainAgent, jsonDetails.Domain)
			}
			if jsonDetails.Category != ErrorCategoryUnknown {
				t.Errorf("expected jsonDetails.Category %q, got %q", ErrorCategoryUnknown, jsonDetails.Category)
			}
			assertMapEqual(t, sampleContext, jsonDetails.Details)

			jsonError := err.ToJSON()
			if jsonError.Code != "BASE_TEST_001" {
				t.Errorf("expected jsonError.Code 'BASE_TEST_001', got %q", jsonError.Code)
			}
			if jsonError.Message != "Original cause" {
				t.Errorf("expected jsonError.Message 'Original cause', got %q", jsonError.Message)
			}
			if jsonError.Domain != ErrorDomainAgent {
				t.Errorf("expected jsonError.Domain %q, got %q", ErrorDomainAgent, jsonError.Domain)
			}
			if jsonError.Category != ErrorCategoryUnknown {
				t.Errorf("expected jsonError.Category %q, got %q", ErrorCategoryUnknown, jsonError.Category)
			}
			assertMapEqual(t, sampleContext, jsonError.Details)

			// Cause should be serialized.
			if jsonError.Cause == nil {
				t.Fatal("expected cause to be serialized")
			}
			if jsonError.Cause.Message != "Original cause" {
				t.Errorf("expected cause message 'Original cause', got %q", jsonError.Cause.Message)
			}
		})

		t.Run("should serialize to JSON without a cause", func(t *testing.T) {
			err := NewMastraError(sampleErrorDefinition)

			jsonDetails := err.ToJSONDetails()
			if jsonDetails.Message != "Unknown error" {
				t.Errorf("expected jsonDetails.Message 'Unknown error', got %q", jsonDetails.Message)
			}
			if jsonDetails.Domain != ErrorDomainAgent {
				t.Errorf("expected jsonDetails.Domain %q, got %q", ErrorDomainAgent, jsonDetails.Domain)
			}
			if jsonDetails.Category != ErrorCategoryUnknown {
				t.Errorf("expected jsonDetails.Category %q, got %q", ErrorCategoryUnknown, jsonDetails.Category)
			}
			assertMapEqual(t, sampleContext, jsonDetails.Details)

			jsonError := err.ToJSON()
			if jsonError.Code != "BASE_TEST_001" {
				t.Errorf("expected jsonError.Code 'BASE_TEST_001', got %q", jsonError.Code)
			}
			if jsonError.Message != "Unknown error" {
				t.Errorf("expected jsonError.Message 'Unknown error', got %q", jsonError.Message)
			}
			if jsonError.Domain != ErrorDomainAgent {
				t.Errorf("expected jsonError.Domain %q, got %q", ErrorDomainAgent, jsonError.Domain)
			}
			if jsonError.Category != ErrorCategoryUnknown {
				t.Errorf("expected jsonError.Category %q, got %q", ErrorCategoryUnknown, jsonError.Category)
			}
			assertMapEqual(t, sampleContext, jsonError.Details)
			if jsonError.Cause != nil {
				t.Errorf("expected cause to be nil, got %v", jsonError.Cause)
			}
		})
	})
}

// assertMapEqual compares two maps for equality by marshaling to JSON and comparing.
func assertMapEqual(t *testing.T, expected, actual map[string]any) {
	t.Helper()
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("failed to marshal expected: %v", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("failed to marshal actual: %v", err)
	}
	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("maps not equal:\n  expected: %s\n  actual:   %s", expectedJSON, actualJSON)
	}
}
