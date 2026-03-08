// Ported from: packages/core/src/tools/validation.ts
package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ---------------------------------------------------------------------------
// Stub types for schema validation
// ---------------------------------------------------------------------------

// SafeParseResult represents the result of a schema's SafeParse call.
// Zod-specific; in Go, schemas implement SafeParser interface directly.
type SafeParseResult struct {
	Success bool
	Data    any
	Error   *SchemaError
}

// SchemaIssue represents a single validation issue from schema parsing.
type SchemaIssue struct {
	Path    []string `json:"path,omitempty"`
	Message string   `json:"message"`
}

// SchemaError represents the error object returned from a failed schema parse.
type SchemaError struct {
	Issues []SchemaIssue `json:"issues"`
}

// Format returns a formatted error representation, matching Zod's .format() output.
func (e *SchemaError) Format() any {
	if e == nil {
		return nil
	}
	result := map[string]any{}
	for _, issue := range e.Issues {
		key := "root"
		if len(issue.Path) > 0 {
			key = strings.Join(issue.Path, ".")
		}
		result[key] = map[string]any{
			"_errors": []string{issue.Message},
		}
	}
	return result
}

// SafeParser is the interface that schemas must implement for validation.
// This corresponds to the TypeScript pattern: schema.safeParse(data).
type SafeParser interface {
	SafeParse(data any) SafeParseResult
}

// SchemaTypeChecker provides methods to inspect schema structure.
type SchemaTypeChecker interface {
	// IsArray returns true if the schema expects an array type.
	IsArray() bool
	// IsObject returns true if the schema expects an object type.
	IsObject() bool
}

// SchemaShapeProvider provides access to object schema field definitions.
type SchemaShapeProvider interface {
	// Shape returns a map of field names to their sub-schemas.
	Shape() map[string]any
}

// SchemaUnwrapper unwraps wrapper types (optional, nullable, default, etc.)
// to find the base schema type.
type SchemaUnwrapper interface {
	// Unwrap returns the innermost base schema.
	Unwrap() any
}

// SchemaTypeNameProvider returns the schema's type name (e.g., "ZodString").
type SchemaTypeNameProvider interface {
	TypeName() string
}

// ---------------------------------------------------------------------------
// Sensitive key redaction
// ---------------------------------------------------------------------------

// sensitiveKeys is the set of keys that should be redacted from error messages
// to prevent sensitive data leakage.
var sensitiveKeys = map[string]bool{
	requestcontext.MastraResourceIDKey: true,
	requestcontext.MastraThreadIDKey:   true,
	"apiKey":                           true,
	"api_key":                          true,
	"token":                            true,
	"secret":                           true,
	"password":                         true,
	"credential":                       true,
	"authorization":                    true,
}

// redactSensitiveKeys returns a copy of data with sensitive key values replaced
// with "[REDACTED]".
func redactSensitiveKeys(data map[string]any) map[string]any {
	result := make(map[string]any, len(data))
	for key, value := range data {
		lower := strings.ToLower(key)
		if sensitiveKeys[key] || strings.Contains(lower, "secret") || strings.Contains(lower, "password") {
			result[key] = "[REDACTED]"
		} else {
			result[key] = value
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Logging helpers
// ---------------------------------------------------------------------------

// truncateForLogging safely truncates data for error messages to avoid
// exposing sensitive information.
func truncateForLogging(data any, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 200
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "[Unable to serialize data]"
	}
	s := string(b)
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength] + "... (truncated)"
}

// ---------------------------------------------------------------------------
// Input normalization helpers
// ---------------------------------------------------------------------------

// normalizeNullishInput normalises nil input to an appropriate default value
// based on schema type. This handles LLMs that send nil instead of {} or []
// when all parameters are optional.
func normalizeNullishInput(schema *SchemaWithValidation, input any) any {
	if input != nil {
		return input
	}

	if schema == nil {
		return input
	}

	// Check if schema implements SchemaTypeChecker.
	if checker, ok := schema.Schema.(SchemaTypeChecker); ok {
		if checker.IsArray() {
			return []any{}
		}
		if checker.IsObject() {
			return map[string]any{}
		}
	}

	return input
}

// isPlainObject checks if a value is a map[string]any (Go equivalent of JS plain object).
func isPlainObject(value any) bool {
	if value == nil {
		return false
	}
	_, ok := value.(map[string]any)
	return ok
}

// stripNullishValues recursively strips nil values from map properties.
// This handles LLMs (e.g. Gemini) that send null for optional fields,
// since schema validation for .optional() only accepts absence, not null.
//
// NOTE: This function should NOT be called unconditionally because it breaks
// schemas that use .nullable() (where null is a valid value). It is used as
// a fallback when initial validation fails. See ValidateToolInput for usage.
func stripNullishValues(input any) any {
	if input == nil {
		return nil
	}

	// Check for slice.
	if slice, ok := input.([]any); ok {
		result := make([]any, len(slice))
		for i, item := range slice {
			if item == nil {
				result[i] = nil // keep nulls in arrays (may be intentional)
			} else {
				result[i] = stripNullishValues(item)
			}
		}
		return result
	}

	// Check for plain object (map).
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}

	result := make(map[string]any, len(m))
	for key, value := range m {
		if value == nil {
			// Omit nil values - equivalent to "not provided" for optional fields.
			continue
		}
		result[key] = stripNullishValues(value)
	}
	return result
}

// coerceStringifiedJsonValues coerces stringified JSON values in object
// properties when the schema expects an array or object but the LLM returned
// a JSON string.
//
// Some LLMs (e.g., GLM4.7) return stringified JSON for array/object parameters:
//
//	{ "args": "[\"parse_excel.py\"]" }
//
// instead of:
//
//	{ "args": ["parse_excel.py"] }
//
// This function walks the top-level properties of a map and attempts to
// json.Unmarshal string values when the schema expects a non-string type.
func coerceStringifiedJsonValues(schema *SchemaWithValidation, input any) any {
	m, ok := input.(map[string]any)
	if !ok {
		return input
	}

	// Try to unwrap the schema to find the base object type.
	unwrapped := unwrapSchema(schema.Schema)

	// Check if unwrapped schema is an object type.
	objChecker, ok := unwrapped.(SchemaTypeChecker)
	if !ok || !objChecker.IsObject() {
		return input
	}

	// Get shape (field schemas).
	shapeProvider, ok := unwrapped.(SchemaShapeProvider)
	if !ok {
		return input
	}
	shape := shapeProvider.Shape()
	if shape == nil {
		return input
	}

	changed := false
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v
	}

	for key, value := range m {
		strVal, isStr := value.(string)
		if !isStr {
			continue
		}

		fieldSchema, hasField := shape[key]
		if !hasField {
			continue
		}

		// Unwrap the field schema to find the base type.
		baseFieldSchema := unwrapSchema(fieldSchema)

		// Check type name - skip if schema expects a string.
		if namer, ok := baseFieldSchema.(SchemaTypeNameProvider); ok {
			if namer.TypeName() == "ZodString" {
				continue
			}
		}

		trimmed := strings.TrimSpace(strVal)

		if checker, ok := baseFieldSchema.(SchemaTypeChecker); ok {
			if checker.IsArray() && strings.HasPrefix(trimmed, "[") {
				var parsed any
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					if _, isSlice := parsed.([]any); isSlice {
						result[key] = parsed
						changed = true
					}
				}
			} else if checker.IsObject() && strings.HasPrefix(trimmed, "{") {
				var parsed any
				if err := json.Unmarshal([]byte(strVal), &parsed); err == nil {
					if _, isMap := parsed.(map[string]any); isMap {
						result[key] = parsed
						changed = true
					}
				}
			}
		}
	}

	if changed {
		return result
	}
	return input
}

// unwrapSchema unwraps wrapper schema types (optional, nullable, default, etc.)
// to find the base schema type. This is the Go equivalent of unwrapZodType.
func unwrapSchema(schema any) any {
	current := schema
	for {
		if unwrapper, ok := current.(SchemaUnwrapper); ok {
			inner := unwrapper.Unwrap()
			if inner == nil || inner == current {
				break
			}
			current = inner
		} else {
			break
		}
	}
	return current
}

// ---------------------------------------------------------------------------
// Public validation functions
// ---------------------------------------------------------------------------

// ValidateToolSuspendData validates raw suspend data against a schema.
//
// If no schema is provided, the suspend data is returned as-is.
// If validation fails, a ValidationError is returned.
func ValidateToolSuspendData(schema *SchemaWithValidation, suspendData any, toolID string) (data any, validationErr *ValidationError) {
	if schema == nil {
		return suspendData, nil
	}

	parser, ok := schema.Schema.(SafeParser)
	if !ok {
		return suspendData, nil
	}

	result := parser.SafeParse(suspendData)
	if result.Success {
		return result.Data, nil
	}

	errorMessages := formatSchemaErrors(result.Error)

	toolSuffix := ""
	if toolID != "" {
		toolSuffix = fmt.Sprintf(" for %s", toolID)
	}

	return suspendData, &ValidationError{
		Error:   true,
		Message: fmt.Sprintf("Tool suspension data validation failed%s. Please fix the following errors and try again:\n%s\n\nProvided arguments: %s", toolSuffix, errorMessages, truncateForLogging(suspendData, 200)),
		ValidationErrors: func() any {
			if result.Error != nil {
				return result.Error.Format()
			}
			return nil
		}(),
	}
}

// ValidateToolInput validates raw input data against a schema.
//
// The validation pipeline:
//  1. normalizeNullishInput: Convert top-level nil to {} or [] based on schema type.
//  2. First validation attempt with values preserved.
//  3. If validation fails, retry with stringified JSON values coerced.
//  4. If validation still fails, retry with nil values stripped from object properties.
func ValidateToolInput(schema *SchemaWithValidation, input any, toolID string) (data any, validationErr *ValidationError) {
	if schema == nil {
		return input, nil
	}

	parser, ok := schema.Schema.(SafeParser)
	if !ok {
		return input, nil
	}

	// Step 1: Normalize top-level nil to appropriate default.
	normalizedInput := normalizeNullishInput(schema, input)

	// Step 2: Try validation with values preserved.
	validation := parser.SafeParse(normalizedInput)
	if validation.Success {
		return validation.Data, nil
	}

	// Step 3: Retry with stringified JSON values coerced.
	coercedInput := coerceStringifiedJsonValues(schema, normalizedInput)
	if !reflect.DeepEqual(coercedInput, normalizedInput) {
		coercedValidation := parser.SafeParse(coercedInput)
		if coercedValidation.Success {
			return coercedValidation.Data, nil
		}
	}

	// Step 4: Retry with nil values stripped.
	strippedInput := stripNullishValues(input)
	normalizedStripped := normalizeNullishInput(schema, strippedInput)
	retryValidation := parser.SafeParse(normalizedStripped)
	if retryValidation.Success {
		return retryValidation.Data, nil
	}

	// All attempts failed - return the original (non-stripped) error since it's
	// more informative about what the schema actually expects.
	errorMessages := formatSchemaErrors(validation.Error)

	toolSuffix := ""
	if toolID != "" {
		toolSuffix = fmt.Sprintf(" for %s", toolID)
	}

	return input, &ValidationError{
		Error:   true,
		Message: fmt.Sprintf("Tool input validation failed%s. Please fix the following errors and try again:\n%s\n\nProvided arguments: %s", toolSuffix, errorMessages, truncateForLogging(input, 200)),
		ValidationErrors: func() any {
			if validation.Error != nil {
				return validation.Error.Format()
			}
			return nil
		}(),
	}
}

// ValidateToolOutput validates tool output data against a schema.
//
// If no schema is provided or suspendCalled is true, the output is returned as-is.
func ValidateToolOutput(schema *SchemaWithValidation, output any, toolID string, suspendCalled bool) (data any, validationErr *ValidationError) {
	if schema == nil || suspendCalled {
		return output, nil
	}

	parser, ok := schema.Schema.(SafeParser)
	if !ok {
		return output, nil
	}

	result := parser.SafeParse(output)
	if result.Success {
		return result.Data, nil
	}

	errorMessages := formatSchemaErrors(result.Error)

	toolSuffix := ""
	if toolID != "" {
		toolSuffix = fmt.Sprintf(" for %s", toolID)
	}

	return output, &ValidationError{
		Error:   true,
		Message: fmt.Sprintf("Tool output validation failed%s. The tool returned invalid output:\n%s\n\nReturned output: %s", toolSuffix, errorMessages, truncateForLogging(output, 200)),
		ValidationErrors: func() any {
			if result.Error != nil {
				return result.Error.Format()
			}
			return nil
		}(),
	}
}

// ValidateRequestContext validates request context values against a schema.
//
// If no schema is provided, the context values are returned as-is.
func ValidateRequestContext(schema *SchemaWithValidation, rc *requestcontext.RequestContext, identifier string) (data any, validationErr *ValidationError) {
	var contextValues map[string]any
	if rc != nil {
		contextValues = rc.All()
	} else {
		contextValues = map[string]any{}
	}

	if schema == nil {
		return contextValues, nil
	}

	parser, ok := schema.Schema.(SafeParser)
	if !ok {
		return contextValues, nil
	}

	result := parser.SafeParse(contextValues)
	if result.Success {
		return result.Data, nil
	}

	errorMessages := formatSchemaErrors(result.Error)

	identSuffix := ""
	if identifier != "" {
		identSuffix = fmt.Sprintf(" for %s", identifier)
	}

	// Redact sensitive keys before including in error message.
	redactedContextValues := redactSensitiveKeys(contextValues)

	return contextValues, &ValidationError{
		Error:   true,
		Message: fmt.Sprintf("Request context validation failed%s. Please fix the following errors and try again:\n%s\n\nProvided context: %s", identSuffix, errorMessages, truncateForLogging(redactedContextValues, 200)),
		ValidationErrors: func() any {
			if result.Error != nil {
				return result.Error.Format()
			}
			return nil
		}(),
	}
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// formatSchemaErrors formats schema validation issues into a human-readable string.
func formatSchemaErrors(err *SchemaError) string {
	if err == nil || len(err.Issues) == 0 {
		return "(no error details)"
	}
	var lines []string
	for _, issue := range err.Issues {
		path := "root"
		if len(issue.Path) > 0 {
			path = strings.Join(issue.Path, ".")
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", path, issue.Message))
	}
	return strings.Join(lines, "\n")
}
