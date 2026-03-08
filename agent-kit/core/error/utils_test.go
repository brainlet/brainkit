// Ported from: packages/core/src/error/utils.test.ts
package mastraerror

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestGetErrorFromUnknown(t *testing.T) {
	t.Run("basic error conversion", func(t *testing.T) {
		t.Run("should return a SerializableError when passed an error", func(t *testing.T) {
			err := errors.New("test error")
			result := GetErrorFromUnknown(err)
			if result.Message() != "test error" {
				t.Errorf("expected message 'test error', got %q", result.Message())
			}
		})

		t.Run("should create an Error from a string", func(t *testing.T) {
			result := GetErrorFromUnknown("test error")
			if result.Message() != "test error" {
				t.Errorf("expected message 'test error', got %q", result.Message())
			}
		})

		t.Run("should create an Error with fallback message for unknown types", func(t *testing.T) {
			result := GetErrorFromUnknown(nil, &GetErrorOptions{FallbackMessage: "Unknown error occurred"})
			if result.Message() != "Unknown error occurred" {
				t.Errorf("expected message 'Unknown error occurred', got %q", result.Message())
			}
		})

		t.Run("should preserve custom properties on error objects", func(t *testing.T) {
			// In Go, we pass a map to simulate an object with custom properties.
			errObj := map[string]any{
				"message":         "test error",
				"statusCode":      float64(500),
				"responseHeaders": map[string]any{"retry-after": "60"},
			}
			result := GetErrorFromUnknown(errObj)
			if result.Message() != "test error" {
				t.Errorf("expected message 'test error', got %q", result.Message())
			}
			if result.Extra()["statusCode"] != float64(500) {
				t.Errorf("expected statusCode 500, got %v", result.Extra()["statusCode"])
			}
			responseHeaders, ok := result.Extra()["responseHeaders"].(map[string]any)
			if !ok {
				t.Fatal("expected responseHeaders to be a map")
			}
			if responseHeaders["retry-after"] != "60" {
				t.Errorf("expected retry-after '60', got %v", responseHeaders["retry-after"])
			}
		})
	})

	t.Run("serializeStack option", func(t *testing.T) {
		t.Run("should always preserve stack on instance regardless of serializeStack option", func(t *testing.T) {
			// Create a SerializableError with a stack.
			sErr := &SerializableError{
				message:        "test error",
				name:           "Error",
				stack:          "Error: test error\n    at something",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			result := GetErrorFromUnknown(sErr, &GetErrorOptions{SerializeStack: boolPtr(false)})

			// Stack should still be on the instance.
			if result.Stack() != "Error: test error\n    at something" {
				t.Errorf("expected stack to be preserved, got %q", result.Stack())
			}
		})

		t.Run("should include stack in JSON when serializeStack is true", func(t *testing.T) {
			sErr := &SerializableError{
				message:        "test error",
				name:           "Error",
				stack:          "Error: test error\n    at something",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			result := GetErrorFromUnknown(sErr, &GetErrorOptions{SerializeStack: boolPtr(true)})

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if _, exists := parsed["stack"]; !exists {
				t.Error("expected stack in JSON output")
			}
		})

		t.Run("should exclude stack from JSON when serializeStack is false", func(t *testing.T) {
			sErr := &SerializableError{
				message:        "test error",
				name:           "Error",
				stack:          "Error: test error\n    at something",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			result := GetErrorFromUnknown(sErr, &GetErrorOptions{SerializeStack: boolPtr(false)})

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if _, exists := parsed["stack"]; exists {
				t.Error("expected no stack in JSON output")
			}
		})
	})

	t.Run("cause chain serialization", func(t *testing.T) {
		t.Run("should add toJSON to cause chain", func(t *testing.T) {
			rootCause := &SerializableError{
				message:        "root cause",
				name:           "Error",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			middleCause := &SerializableError{
				message:        "middle cause",
				name:           "Error",
				cause:          rootCause,
				serializeStack: true,
				extra:          make(map[string]any),
			}
			topError := &SerializableError{
				message:        "top error",
				name:           "Error",
				cause:          middleCause,
				serializeStack: true,
				extra:          make(map[string]any),
			}

			result := GetErrorFromUnknown(topError)

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if parsed["message"] != "top error" {
				t.Errorf("expected top message 'top error', got %v", parsed["message"])
			}

			causeMap, ok := parsed["cause"].(map[string]any)
			if !ok {
				t.Fatal("expected cause to be an object")
			}
			if causeMap["message"] != "middle cause" {
				t.Errorf("expected middle message 'middle cause', got %v", causeMap["message"])
			}

			innerCauseMap, ok := causeMap["cause"].(map[string]any)
			if !ok {
				t.Fatal("expected inner cause to be an object")
			}
			if innerCauseMap["message"] != "root cause" {
				t.Errorf("expected root message 'root cause', got %v", innerCauseMap["message"])
			}
		})

		t.Run("should respect serializeStack for entire cause chain", func(t *testing.T) {
			rootCause := &SerializableError{
				message:        "root cause",
				name:           "Error",
				stack:          "root stack",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			topError := &SerializableError{
				message:        "top error",
				name:           "Error",
				stack:          "top stack",
				cause:          rootCause,
				serializeStack: true,
				extra:          make(map[string]any),
			}

			result := GetErrorFromUnknown(topError, &GetErrorOptions{SerializeStack: boolPtr(false)})

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if _, exists := parsed["stack"]; exists {
				t.Error("expected no stack in top-level JSON")
			}

			causeMap, ok := parsed["cause"].(map[string]any)
			if !ok {
				t.Fatal("expected cause to be an object")
			}
			if _, exists := causeMap["stack"]; exists {
				t.Error("expected no stack in cause JSON")
			}
		})

		t.Run("should preserve custom properties on cause errors", func(t *testing.T) {
			rootCause := &SerializableError{
				message:        "root cause",
				name:           "Error",
				serializeStack: true,
				extra:          map[string]any{"code": "ECONNREFUSED"},
			}
			topError := &SerializableError{
				message:        "top error",
				name:           "Error",
				cause:          rootCause,
				serializeStack: true,
				extra:          map[string]any{"statusCode": float64(500)},
			}

			result := GetErrorFromUnknown(topError)

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			if parsed["statusCode"] != float64(500) {
				t.Errorf("expected statusCode 500, got %v", parsed["statusCode"])
			}

			causeMap, ok := parsed["cause"].(map[string]any)
			if !ok {
				t.Fatal("expected cause to be an object")
			}
			if causeMap["code"] != "ECONNREFUSED" {
				t.Errorf("expected cause code 'ECONNREFUSED', got %v", causeMap["code"])
			}
		})
	})

	t.Run("maxDepth protection", func(t *testing.T) {
		t.Run("should limit cause chain processing to maxDepth", func(t *testing.T) {
			// Create a chain of 10 errors.
			var current *SerializableError
			for i := 0; i < 10; i++ {
				e := &SerializableError{
					message:        fmt.Sprintf("error-%d", i),
					name:           "Error",
					serializeStack: true,
					extra:          make(map[string]any),
				}
				if current != nil {
					e.cause = current
				}
				current = e
			}

			// Process with maxDepth of 3.
			result := GetErrorFromUnknown(current, &GetErrorOptions{MaxDepth: intPtr(3)})

			// The top-level error should be a *SerializableError.
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			// Traverse the chain and count how many SerializableErrors we can follow.
			var c any = result
			count := 0
			for c != nil {
				count++
				if sErr, ok := c.(*SerializableError); ok {
					c = sErr.cause
				} else {
					break
				}
			}

			// With maxDepth=3, we should be able to follow 4 levels (0, 1, 2, 3).
			if count != 4 {
				t.Errorf("expected 4 levels of cause chain, got %d", count)
			}
		})

		t.Run("should handle deeply nested causes without stack overflow", func(t *testing.T) {
			// Create a very deep chain (100 errors).
			var current *SerializableError
			for i := 0; i < 100; i++ {
				e := &SerializableError{
					message:        fmt.Sprintf("error-%d", i),
					name:           "Error",
					serializeStack: true,
					extra:          make(map[string]any),
				}
				if current != nil {
					e.cause = current
				}
				current = e
			}

			// Should not panic due to depth protection.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("unexpected panic: %v", r)
					}
				}()
				GetErrorFromUnknown(current)
			}()
		})

		t.Run("should use default maxDepth when not specified", func(t *testing.T) {
			// Create a chain that exceeds default depth (5).
			var current *SerializableError
			for i := 0; i < 20; i++ {
				e := &SerializableError{
					message:        fmt.Sprintf("error-%d", i),
					name:           "Error",
					serializeStack: true,
					extra:          make(map[string]any),
				}
				if current != nil {
					e.cause = current
				}
				current = e
			}

			// Should process without error.
			result := GetErrorFromUnknown(current)
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Message() != "error-19" {
				t.Errorf("expected message 'error-19', got %q", result.Message())
			}
		})
	})

	t.Run("object to Error conversion", func(t *testing.T) {
		t.Run("should convert plain objects with message property to Error", func(t *testing.T) {
			obj := map[string]any{"message": "error from object", "code": "ERR_TEST"}
			result := GetErrorFromUnknown(obj)

			if result.Message() != "error from object" {
				t.Errorf("expected message 'error from object', got %q", result.Message())
			}
			if result.Extra()["code"] != "ERR_TEST" {
				t.Errorf("expected code 'ERR_TEST', got %v", result.Extra()["code"])
			}
		})

		t.Run("should preserve cause from plain objects", func(t *testing.T) {
			cause := &SerializableError{
				message:        "original cause",
				name:           "Error",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			obj := map[string]any{"message": "wrapper error", "cause": cause}

			result := GetErrorFromUnknown(obj)

			if result.Message() != "wrapper error" {
				t.Errorf("expected message 'wrapper error', got %q", result.Message())
			}
			causeErr, ok := result.CauseValue().(*SerializableError)
			if !ok {
				t.Fatal("expected cause to be a *SerializableError")
			}
			if causeErr.Message() != "original cause" {
				t.Errorf("expected cause message 'original cause', got %q", causeErr.Message())
			}
		})
	})

	t.Run("toJSON serialization", func(t *testing.T) {
		t.Run("should include message and name in JSON", func(t *testing.T) {
			err := errors.New("test error")
			result := GetErrorFromUnknown(err)

			b, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				t.Fatalf("failed to marshal: %v", marshalErr)
			}
			var parsed map[string]any
			if marshalErr := json.Unmarshal(b, &parsed); marshalErr != nil {
				t.Fatalf("failed to unmarshal: %v", marshalErr)
			}
			if parsed["message"] != "test error" {
				t.Errorf("expected message 'test error', got %v", parsed["message"])
			}
			if parsed["name"] != "Error" {
				t.Errorf("expected name 'Error', got %v", parsed["name"])
			}
		})

		t.Run("should not overwrite existing toJSON method", func(t *testing.T) {
			// In Go, we test that a *SerializableError with custom ToJSON behavior
			// preserves its existing serialization. We use the extra map to verify
			// that custom data is preserved.
			sErr := &SerializableError{
				message:        "test error",
				name:           "Error",
				serializeStack: true,
				extra:          map[string]any{"custom": true},
			}

			result := GetErrorFromUnknown(sErr)

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if parsed["custom"] != true {
				t.Errorf("expected custom=true, got %v", parsed["custom"])
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should handle mixed cause types (string cause)", func(t *testing.T) {
			// An error with a string cause: in Go, we build a SerializableError
			// with a string cause value.
			sErr := &SerializableError{
				message:        "top error",
				name:           "Error",
				cause:          "string cause",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			result := GetErrorFromUnknown(sErr)

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if parsed["message"] != "top error" {
				t.Errorf("expected message 'top error', got %v", parsed["message"])
			}
			// The string cause gets wrapped into a SerializableError with message "string cause".
			causeMap, ok := parsed["cause"].(map[string]any)
			if !ok {
				// Could also be a direct string depending on depth processing.
				if causeStr, ok := parsed["cause"].(string); ok {
					if causeStr != "string cause" {
						t.Errorf("expected cause 'string cause', got %v", causeStr)
					}
				} else {
					t.Fatalf("expected cause to be a string or map, got %T", parsed["cause"])
				}
			} else {
				if causeMap["message"] != "string cause" {
					t.Errorf("expected cause message 'string cause', got %v", causeMap["message"])
				}
			}
		})

		t.Run("should handle mixed cause types (plain object cause)", func(t *testing.T) {
			plainCause := map[string]any{"code": "ERR_PLAIN", "details": "some details"}
			sErr := &SerializableError{
				message:        "top error",
				name:           "Error",
				cause:          plainCause,
				serializeStack: true,
				extra:          make(map[string]any),
			}
			result := GetErrorFromUnknown(sErr)

			b, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(b, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if parsed["message"] != "top error" {
				t.Errorf("expected message 'top error', got %v", parsed["message"])
			}

			// The plain object cause gets processed by GetErrorFromUnknown into a SerializableError.
			causeMap, ok := parsed["cause"].(map[string]any)
			if !ok {
				t.Fatalf("expected cause to be an object, got %T", parsed["cause"])
			}
			if causeMap["code"] != "ERR_PLAIN" {
				t.Errorf("expected cause code 'ERR_PLAIN', got %v", causeMap["code"])
			}
			if causeMap["details"] != "some details" {
				t.Errorf("expected cause details 'some details', got %v", causeMap["details"])
			}
		})

		t.Run("should handle number as unknown input", func(t *testing.T) {
			result := GetErrorFromUnknown(42, &GetErrorOptions{FallbackMessage: "Unexpected error"})
			if result.Message() != "Unexpected error" {
				t.Errorf("expected message 'Unexpected error', got %q", result.Message())
			}
		})

		t.Run("should handle array as unknown input", func(t *testing.T) {
			result := GetErrorFromUnknown([]string{"error1", "error2"}, &GetErrorOptions{FallbackMessage: "Unexpected error"})
			// Arrays get JSON stringified as the message.
			// Note: In the TS version, arrays are objects, so they get safeParseErrorObject'd.
			// In Go, []string is not a map, so it falls through to the fallback.
			// However, to match TS behavior, we need to handle slices as objects too.
			// The TS test expects: '["error1","error2"]'
			if result.Message() != `["error1","error2"]` {
				t.Errorf("expected message '[\"error1\",\"error2\"]', got %q", result.Message())
			}
		})

		t.Run("should handle symbol as unknown input", func(t *testing.T) {
			// Go doesn't have symbols. Use a custom type to simulate a non-standard value.
			type customType struct{}
			result := GetErrorFromUnknown(customType{}, &GetErrorOptions{FallbackMessage: "Unexpected error"})
			if result.Message() != "Unexpected error" {
				t.Errorf("expected message 'Unexpected error', got %q", result.Message())
			}
		})

		t.Run("should handle undefined as unknown input", func(t *testing.T) {
			result := GetErrorFromUnknown(nil, &GetErrorOptions{FallbackMessage: "Unexpected error"})
			if result.Message() != "Unexpected error" {
				t.Errorf("expected message 'Unexpected error', got %q", result.Message())
			}
		})

		t.Run("should handle circular references in cause chains gracefully via maxDepth", func(t *testing.T) {
			error1 := &SerializableError{
				message:        "error 1",
				name:           "Error",
				serializeStack: true,
				extra:          make(map[string]any),
			}
			error2 := &SerializableError{
				message:        "error 2",
				name:           "Error",
				cause:          error1,
				serializeStack: true,
				extra:          make(map[string]any),
			}
			// Create a circular reference.
			error1.cause = error2

			// Should not panic due to maxDepth protection.
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("unexpected panic: %v", r)
					}
				}()
				result := GetErrorFromUnknown(error2, &GetErrorOptions{MaxDepth: intPtr(3)})
				if result.Message() != "error 2" {
					t.Errorf("expected message 'error 2', got %q", result.Message())
				}
			}()
		})
	})
}
