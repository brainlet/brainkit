// Ported from: packages/core/src/tools/validation.test.ts
package tools

import (
	"reflect"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// mockSafeParser is a test implementation of SafeParser for use in validation tests.
// It uses a validate function to determine success/failure.
type mockSafeParser struct {
	validateFn func(data any) SafeParseResult
}

func (m *mockSafeParser) SafeParse(data any) SafeParseResult {
	return m.validateFn(data)
}

// newMockSchemaWithValidation creates a SchemaWithValidation that wraps a mockSafeParser.
func newMockSchemaWithValidation(validateFn func(data any) SafeParseResult) *SchemaWithValidation {
	return &SchemaWithValidation{
		Schema: &mockSafeParser{validateFn: validateFn},
	}
}

// alwaysPassSchema creates a schema that always passes validation, returning data as-is.
func alwaysPassSchema() *SchemaWithValidation {
	return newMockSchemaWithValidation(func(data any) SafeParseResult {
		return SafeParseResult{Success: true, Data: data}
	})
}

// alwaysFailSchema creates a schema that always fails validation with the given issues.
func alwaysFailSchema(issues []SchemaIssue) *SchemaWithValidation {
	return newMockSchemaWithValidation(func(data any) SafeParseResult {
		return SafeParseResult{
			Success: false,
			Error:   &SchemaError{Issues: issues},
		}
	})
}

// requiredFieldSchema creates a schema that requires specific fields to be present.
func requiredFieldSchema(requiredFields ...string) *SchemaWithValidation {
	return newMockSchemaWithValidation(func(data any) SafeParseResult {
		m, ok := data.(map[string]any)
		if !ok {
			return SafeParseResult{
				Success: false,
				Error: &SchemaError{Issues: []SchemaIssue{
					{Message: "Expected object"},
				}},
			}
		}
		var issues []SchemaIssue
		for _, field := range requiredFields {
			if _, exists := m[field]; !exists {
				issues = append(issues, SchemaIssue{
					Path:    []string{field},
					Message: "Required",
				})
			}
		}
		if len(issues) > 0 {
			return SafeParseResult{
				Success: false,
				Error:   &SchemaError{Issues: issues},
			}
		}
		return SafeParseResult{Success: true, Data: data}
	})
}

func TestValidateToolInput(t *testing.T) {
	t.Run("should return data as-is when schema is nil", func(t *testing.T) {
		input := map[string]any{"name": "test"}
		data, validationErr := ValidateToolInput(nil, input, "test-tool")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["name"] != "test" {
			t.Errorf("expected input to pass through, got %v", data)
		}
	})

	t.Run("should pass validation with valid data", func(t *testing.T) {
		schema := alwaysPassSchema()
		input := map[string]any{"name": "John Doe", "age": 30}

		data, validationErr := ValidateToolInput(schema, input, "valid-test")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", data)
		}
		if m["name"] != "John Doe" {
			t.Errorf("expected name=John Doe, got %v", m["name"])
		}
	})

	t.Run("should fail validation when required fields are missing", func(t *testing.T) {
		schema := requiredFieldSchema("name", "age")
		input := map[string]any{}

		_, validationErr := ValidateToolInput(schema, input, "test-tool")
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !validationErr.Error {
			t.Error("expected error=true")
		}
		if !strings.Contains(validationErr.Message, "Tool input validation failed") {
			t.Errorf("expected message to contain 'Tool input validation failed', got: %s", validationErr.Message)
		}
		if !strings.Contains(validationErr.Message, "name: Required") {
			t.Errorf("expected message to contain 'name: Required', got: %s", validationErr.Message)
		}
		if !strings.Contains(validationErr.Message, "age: Required") {
			t.Errorf("expected message to contain 'age: Required', got: %s", validationErr.Message)
		}
	})

	t.Run("should include tool ID in validation error messages", func(t *testing.T) {
		schema := requiredFieldSchema("username")
		input := map[string]any{}

		_, validationErr := ValidateToolInput(schema, input, "user-registration")
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(validationErr.Message, "Tool input validation failed for user-registration") {
			t.Errorf("expected message to contain tool ID, got: %s", validationErr.Message)
		}
	})

	t.Run("should handle tools without input schema", func(t *testing.T) {
		input := map[string]any{"anything": "goes"}
		data, validationErr := ValidateToolInput(nil, input, "no-schema")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["anything"] != "goes" {
			t.Errorf("expected data to pass through, got %v", data)
		}
	})
}

func TestValidateToolOutput(t *testing.T) {
	t.Run("should return data as-is when schema is nil", func(t *testing.T) {
		output := map[string]any{"result": "success"}
		data, validationErr := ValidateToolOutput(nil, output, "test-tool", false)
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["result"] != "success" {
			t.Errorf("expected output to pass through, got %v", data)
		}
	})

	t.Run("should pass validation with valid output", func(t *testing.T) {
		schema := alwaysPassSchema()
		output := map[string]any{"id": "123", "name": "John"}

		data, validationErr := ValidateToolOutput(schema, output, "output-test", false)
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["id"] != "123" {
			t.Errorf("expected valid output, got %v", data)
		}
	})

	t.Run("should fail validation when output does not match schema", func(t *testing.T) {
		schema := alwaysFailSchema([]SchemaIssue{
			{Path: []string{"name"}, Message: "Required"},
			{Path: []string{"email"}, Message: "Required"},
		})
		output := map[string]any{"id": "123"}

		_, validationErr := ValidateToolOutput(schema, output, "invalid-output", false)
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !validationErr.Error {
			t.Error("expected error=true")
		}
		if !strings.Contains(validationErr.Message, "Tool output validation failed") {
			t.Errorf("expected message to contain 'Tool output validation failed', got: %s", validationErr.Message)
		}
		if !strings.Contains(validationErr.Message, "name: Required") {
			t.Errorf("expected message to contain 'name: Required', got: %s", validationErr.Message)
		}
		if !strings.Contains(validationErr.Message, "email: Required") {
			t.Errorf("expected message to contain 'email: Required', got: %s", validationErr.Message)
		}
	})

	t.Run("should include tool ID in output validation error messages", func(t *testing.T) {
		schema := alwaysFailSchema([]SchemaIssue{
			{Path: []string{"userId"}, Message: "Invalid uuid"},
		})
		output := map[string]any{"userId": "not-a-uuid"}

		_, validationErr := ValidateToolOutput(schema, output, "user-service", false)
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(validationErr.Message, "Tool output validation failed for user-service") {
			t.Errorf("expected message to contain tool ID, got: %s", validationErr.Message)
		}
	})

	t.Run("should skip validation when suspendCalled is true", func(t *testing.T) {
		schema := alwaysFailSchema([]SchemaIssue{
			{Message: "Should not matter"},
		})
		output := map[string]any{"anything": "goes"}

		data, validationErr := ValidateToolOutput(schema, output, "test", true)
		if validationErr != nil {
			t.Fatalf("expected no error when suspendCalled, got: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["anything"] != "goes" {
			t.Errorf("expected output to pass through, got %v", data)
		}
	})
}

func TestValidateToolSuspendData(t *testing.T) {
	t.Run("should return data as-is when schema is nil", func(t *testing.T) {
		suspendData := map[string]any{"reason": "pause"}
		data, validationErr := ValidateToolSuspendData(nil, suspendData, "test-tool")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["reason"] != "pause" {
			t.Errorf("expected suspend data to pass through, got %v", data)
		}
	})

	t.Run("should pass validation with valid suspend data", func(t *testing.T) {
		schema := alwaysPassSchema()
		suspendData := map[string]any{"step": "waiting"}

		data, validationErr := ValidateToolSuspendData(schema, suspendData, "suspend-test")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok || m["step"] != "waiting" {
			t.Errorf("expected valid suspend data, got %v", data)
		}
	})

	t.Run("should fail validation when suspend data is invalid", func(t *testing.T) {
		schema := alwaysFailSchema([]SchemaIssue{
			{Path: []string{"reason"}, Message: "Required"},
		})
		suspendData := map[string]any{}

		_, validationErr := ValidateToolSuspendData(schema, suspendData, "suspend-tool")
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(validationErr.Message, "Tool suspension data validation failed") {
			t.Errorf("expected suspend validation message, got: %s", validationErr.Message)
		}
	})
}

func TestTruncateForLogging(t *testing.T) {
	t.Run("should truncate large data", func(t *testing.T) {
		// Create data that will exceed 200 characters when serialized.
		largeData := map[string]any{}
		for i := 0; i < 50; i++ {
			largeData["key"+string(rune('A'+i%26))+string(rune('0'+i/26))] = "some-long-value-that-fills-space"
		}

		result := truncateForLogging(largeData, 200)
		if !strings.Contains(result, "... (truncated)") {
			t.Errorf("expected truncated output, got: %s", result)
		}
		if len(result) > 250 { // 200 + "... (truncated)" overhead
			t.Errorf("expected truncated length, got %d", len(result))
		}
	})

	t.Run("should not truncate small data", func(t *testing.T) {
		smallData := map[string]any{"key": "value"}
		result := truncateForLogging(smallData, 200)
		if strings.Contains(result, "... (truncated)") {
			t.Errorf("expected non-truncated output, got: %s", result)
		}
	})
}

func TestStripNullishValues(t *testing.T) {
	t.Run("should strip nil values from maps", func(t *testing.T) {
		input := map[string]any{
			"name":  "test",
			"value": nil,
			"count": 42,
		}
		result := stripNullishValues(input)
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}
		if _, exists := m["value"]; exists {
			t.Error("expected nil value to be stripped")
		}
		if m["name"] != "test" {
			t.Errorf("expected name=test, got %v", m["name"])
		}
		if m["count"] != 42 {
			t.Errorf("expected count=42, got %v", m["count"])
		}
	})

	t.Run("should handle nested maps", func(t *testing.T) {
		input := map[string]any{
			"level1": map[string]any{
				"level2": map[string]any{
					"value":    nil,
					"required": "present",
				},
			},
		}
		result := stripNullishValues(input)
		m := result.(map[string]any)
		l1 := m["level1"].(map[string]any)
		l2 := l1["level2"].(map[string]any)
		if _, exists := l2["value"]; exists {
			t.Error("expected nested nil value to be stripped")
		}
		if l2["required"] != "present" {
			t.Errorf("expected required=present, got %v", l2["required"])
		}
	})

	t.Run("should return nil for nil input", func(t *testing.T) {
		result := stripNullishValues(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("should return non-map non-slice values as-is", func(t *testing.T) {
		result := stripNullishValues("hello")
		if result != "hello" {
			t.Errorf("expected hello, got %v", result)
		}
	})

	t.Run("should handle slices preserving nils", func(t *testing.T) {
		input := []any{"a", nil, "b"}
		result := stripNullishValues(input)
		s, ok := result.([]any)
		if !ok {
			t.Fatalf("expected slice, got %T", result)
		}
		if len(s) != 3 {
			t.Fatalf("expected 3 elements, got %d", len(s))
		}
		if s[0] != "a" || s[1] != nil || s[2] != "b" {
			t.Errorf("unexpected slice contents: %v", s)
		}
	})
}

func TestNormalizeNullishInput(t *testing.T) {
	t.Run("should return input as-is when not nil", func(t *testing.T) {
		input := map[string]any{"key": "value"}
		result := normalizeNullishInput(nil, input)
		if !reflect.DeepEqual(result, input) {
			t.Errorf("expected same input reference, got %v", result)
		}
	})

	t.Run("should return nil input as-is when schema is nil", func(t *testing.T) {
		result := normalizeNullishInput(nil, nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestRedactSensitiveKeys(t *testing.T) {
	t.Run("should redact known sensitive keys", func(t *testing.T) {
		data := map[string]any{
			"apiKey":        "secret-key-123",
			"api_key":       "another-key",
			"token":         "bearer-token",
			"secret":        "shh",
			"password":      "pass123",
			"credential":    "cred",
			"authorization": "auth-header",
			"name":          "visible",
		}
		result := redactSensitiveKeys(data)
		for key, val := range result {
			if key == "name" {
				if val != "visible" {
					t.Errorf("expected name=visible, got %v", val)
				}
			} else {
				if val != "[REDACTED]" {
					t.Errorf("expected key %s to be [REDACTED], got %v", key, val)
				}
			}
		}
	})
}

func TestIsPlainObject(t *testing.T) {
	t.Run("should return true for map[string]any", func(t *testing.T) {
		if !isPlainObject(map[string]any{"key": "value"}) {
			t.Error("expected true for map[string]any")
		}
	})

	t.Run("should return false for nil", func(t *testing.T) {
		if isPlainObject(nil) {
			t.Error("expected false for nil")
		}
	})

	t.Run("should return false for string", func(t *testing.T) {
		if isPlainObject("hello") {
			t.Error("expected false for string")
		}
	})

	t.Run("should return false for slice", func(t *testing.T) {
		if isPlainObject([]any{1, 2, 3}) {
			t.Error("expected false for slice")
		}
	})
}

func TestFormatSchemaErrors(t *testing.T) {
	t.Run("should format schema errors into readable string", func(t *testing.T) {
		err := &SchemaError{
			Issues: []SchemaIssue{
				{Path: []string{"name"}, Message: "Required"},
				{Path: []string{"age"}, Message: "Expected number"},
				{Message: "General error"},
			},
		}
		result := formatSchemaErrors(err)
		if !strings.Contains(result, "- name: Required") {
			t.Errorf("expected 'name: Required' in output, got: %s", result)
		}
		if !strings.Contains(result, "- age: Expected number") {
			t.Errorf("expected 'age: Expected number' in output, got: %s", result)
		}
		if !strings.Contains(result, "- root: General error") {
			t.Errorf("expected 'root: General error' in output, got: %s", result)
		}
	})

	t.Run("should handle nil error", func(t *testing.T) {
		result := formatSchemaErrors(nil)
		if result != "(no error details)" {
			t.Errorf("expected '(no error details)', got: %s", result)
		}
	})

	t.Run("should handle empty issues", func(t *testing.T) {
		err := &SchemaError{Issues: []SchemaIssue{}}
		result := formatSchemaErrors(err)
		if result != "(no error details)" {
			t.Errorf("expected '(no error details)', got: %s", result)
		}
	})
}

func TestSchemaErrorFormat(t *testing.T) {
	t.Run("should format errors into map structure", func(t *testing.T) {
		err := &SchemaError{
			Issues: []SchemaIssue{
				{Path: []string{"name"}, Message: "Required"},
				{Message: "Root level error"},
			},
		}
		result := err.Format()
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}
		if _, exists := m["name"]; !exists {
			t.Error("expected 'name' key in formatted output")
		}
		if _, exists := m["root"]; !exists {
			t.Error("expected 'root' key in formatted output")
		}
	})

	t.Run("should return nil for nil error", func(t *testing.T) {
		var err *SchemaError
		result := err.Format()
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})
}

func TestValidateRequestContext(t *testing.T) {
	t.Run("should return context values as-is when schema is nil", func(t *testing.T) {
		rc := requestcontext.NewRequestContext()
		rc.Set("key", "value")

		data, validationErr := ValidateRequestContext(nil, rc, "test-tool")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", data)
		}
		if m["key"] != "value" {
			t.Errorf("expected key=value, got %v", m["key"])
		}
	})

	t.Run("should return empty map when request context is nil and schema is nil", func(t *testing.T) {
		data, validationErr := ValidateRequestContext(nil, nil, "test-tool")
		if validationErr != nil {
			t.Fatalf("unexpected validation error: %v", validationErr)
		}
		m, ok := data.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", data)
		}
		if len(m) != 0 {
			t.Errorf("expected empty map, got %v", m)
		}
	})

	t.Run("should fail validation with schema and invalid context", func(t *testing.T) {
		schema := alwaysFailSchema([]SchemaIssue{
			{Path: []string{"token"}, Message: "Required"},
		})
		rc := requestcontext.NewRequestContext()

		_, validationErr := ValidateRequestContext(schema, rc, "auth-tool")
		if validationErr == nil {
			t.Fatal("expected validation error, got nil")
		}
		if !strings.Contains(validationErr.Message, "Request context validation failed for auth-tool") {
			t.Errorf("expected context validation message, got: %s", validationErr.Message)
		}
	})
}

func TestToolInputValidationIntegration(t *testing.T) {
	// These tests verify the integration of validation within CreateTool's wrapped Execute.

	t.Run("should validate required fields via tool execute", func(t *testing.T) {
		schema := requiredFieldSchema("name", "age")
		tool := CreateTool(ToolAction{
			ID:          "test-tool",
			Description: "Test tool with validation",
			InputSchema: schema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				return map[string]any{"success": true, "data": inputData}, nil
			},
		})

		// Test with missing required fields.
		result, err := tool.Execute(map[string]any{}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// When validation fails, the ValidationError is returned as the result.
		ve, ok := result.(*ValidationError)
		if !ok {
			t.Fatalf("expected *ValidationError result, got %T: %v", result, result)
		}
		if !ve.Error {
			t.Error("expected error=true")
		}
		if !strings.Contains(ve.Message, "Tool input validation failed") {
			t.Errorf("expected validation failure message, got: %s", ve.Message)
		}
	})

	t.Run("should pass validation with valid data via tool execute", func(t *testing.T) {
		schema := requiredFieldSchema("name")
		tool := CreateTool(ToolAction{
			ID:          "valid-test",
			Description: "Test valid data",
			InputSchema: schema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				m := inputData.(map[string]any)
				return map[string]any{"success": true, "name": m["name"]}, nil
			},
		})

		result, err := tool.Execute(map[string]any{"name": "John Doe"}, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T: %v", result, result)
		}
		if m["success"] != true {
			t.Errorf("expected success=true, got %v", m["success"])
		}
		if m["name"] != "John Doe" {
			t.Errorf("expected name=John Doe, got %v", m["name"])
		}
	})
}

func TestToolOutputValidationIntegration(t *testing.T) {
	t.Run("should validate output via tool execute", func(t *testing.T) {
		outputSchema := alwaysFailSchema([]SchemaIssue{
			{Path: []string{"count"}, Message: "Required"},
		})
		tool := CreateTool(ToolAction{
			ID:           "output-fail",
			Description:  "Test invalid output",
			OutputSchema: outputSchema,
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				return map[string]any{"result": "success"}, nil // Missing "count"
			},
		})

		result, err := tool.Execute(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		ve, ok := result.(*ValidationError)
		if !ok {
			t.Fatalf("expected *ValidationError, got %T: %v", result, result)
		}
		if !strings.Contains(ve.Message, "Tool output validation failed") {
			t.Errorf("expected output validation failure, got: %s", ve.Message)
		}
	})

	t.Run("should allow tools without output schema", func(t *testing.T) {
		tool := CreateTool(ToolAction{
			ID:          "no-output-schema",
			Description: "Tool without output schema",
			Execute: func(inputData any, ctx *ToolExecutionContext) (any, error) {
				return map[string]any{"anything": "goes", "extra": 123}, nil
			},
		})

		result, err := tool.Execute(nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m, ok := result.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", result)
		}
		if m["anything"] != "goes" {
			t.Errorf("expected anything=goes, got %v", m["anything"])
		}
	})
}
