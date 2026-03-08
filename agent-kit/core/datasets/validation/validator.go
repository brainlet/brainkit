// Ported from: packages/core/src/datasets/validation/validator.ts
package validation

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// JSONSchema7 is a JSON Schema (draft-07 compatible).
// In TS this is the JSONSchema7 type from json-schema.
type JSONSchema7 = map[string]any

// ZodSchema is the Go equivalent of a compiled Zod schema.
// Wraps a compiled JSON Schema validator from santhosh-tekuri/jsonschema.
type ZodSchema interface {
	// Validate validates data against the schema.
	// Returns nil if valid, or a list of FieldErrors if invalid.
	Validate(data any) []FieldError
}

// ---------------------------------------------------------------------------
// jsonSchemaValidator implements ZodSchema using santhosh-tekuri/jsonschema
// ---------------------------------------------------------------------------

// jsonSchemaValidator wraps a compiled JSON Schema for validation.
// Corresponds to the TS pattern: jsonSchemaToZod(schema) → resolveZodSchema(zodString)
type jsonSchemaValidator struct {
	schema *jsonschema.Schema
}

// Validate validates data against the compiled JSON Schema.
// Converts jsonschema validation errors to FieldError slice.
func (v *jsonSchemaValidator) Validate(data any) []FieldError {
	err := v.schema.Validate(data)
	if err == nil {
		return nil
	}

	// Convert validation error to FieldError slice.
	// santhosh-tekuri/jsonschema returns *jsonschema.ValidationError with structured causes.
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		// Unexpected error type — wrap as a single generic error.
		return []FieldError{{
			Path:    "/",
			Code:    "validation_error",
			Message: err.Error(),
		}}
	}

	return extractFieldErrors(validationErr)
}

// extractFieldErrors recursively extracts FieldError entries from a jsonschema.ValidationError tree.
func extractFieldErrors(ve *jsonschema.ValidationError) []FieldError {
	if ve == nil {
		return nil
	}

	// If there are causes (child errors), recurse into them.
	if len(ve.Causes) > 0 {
		var errors []FieldError
		for _, cause := range ve.Causes {
			errors = append(errors, extractFieldErrors(cause)...)
		}
		return errors
	}

	// Leaf error — convert to FieldError.
	path := instanceLocationToJSONPointer(ve.InstanceLocation)
	code := errorKindToCode(ve.ErrorKind)
	msg := ve.Error()

	return []FieldError{{
		Path:    path,
		Code:    code,
		Message: msg,
	}}
}

// instanceLocationToJSONPointer converts a jsonschema instance location ([]string) to a JSON Pointer path.
func instanceLocationToJSONPointer(loc []string) string {
	if len(loc) == 0 {
		return "/"
	}
	return "/" + strings.Join(loc, "/")
}

// errorKindToCode derives a validation error code from the ErrorKind.
// Maps common JSON Schema error kinds to Zod-like error codes for consistency
// with the TS source's FieldError.code values.
func errorKindToCode(kind jsonschema.ErrorKind) string {
	// Use the kind's KeywordPath or type name to determine the code.
	// ErrorKind is an interface with concrete types like *InvalidTypeError, etc.
	switch kind.(type) {
	default:
		// Fall back to extracting from the string representation.
		s := fmt.Sprintf("%T", kind)
		s = strings.TrimPrefix(s, "*jsonschema.")
		switch {
		case strings.Contains(s, "Type"):
			return "invalid_type"
		case strings.Contains(s, "Required"):
			return "required"
		case strings.Contains(s, "MinLength"), strings.Contains(s, "MinItems"),
			strings.Contains(s, "Minimum"):
			return "too_small"
		case strings.Contains(s, "MaxLength"), strings.Contains(s, "MaxItems"),
			strings.Contains(s, "Maximum"):
			return "too_big"
		case strings.Contains(s, "Pattern"):
			return "invalid_string"
		case strings.Contains(s, "Format"):
			return "invalid_format"
		case strings.Contains(s, "Enum"):
			return "invalid_enum_value"
		case strings.Contains(s, "Const"):
			return "invalid_literal"
		case strings.Contains(s, "AdditionalProperties"):
			return "unrecognized_keys"
		default:
			return "custom"
		}
	}
}

// ============================================================================
// Schema Validator
// ============================================================================

// SchemaValidator provides schema validation with compilation caching.
// Corresponds to TS: SchemaValidator class in validator.ts.
type SchemaValidator struct {
	mu    sync.RWMutex
	cache map[string]ZodSchema
}

// NewSchemaValidator creates a new SchemaValidator.
func NewSchemaValidator() *SchemaValidator {
	return &SchemaValidator{
		cache: make(map[string]ZodSchema),
	}
}

// getValidator retrieves or compiles a validator for the given schema.
//
// In the TypeScript source, this converts JSON Schema to Zod via jsonSchemaToZod
// and evaluates the generated code with Function(). In Go, we use santhosh-tekuri/jsonschema
// to compile the JSON Schema directly.
func (sv *SchemaValidator) getValidator(schema JSONSchema7, cacheKey string) ZodSchema {
	sv.mu.RLock()
	if v, ok := sv.cache[cacheKey]; ok {
		sv.mu.RUnlock()
		return v
	}
	sv.mu.RUnlock()

	// Compile JSON Schema to a validator.
	// The TS source does: jsonSchemaToZod(schema) → resolveZodSchema(zodString)
	// In Go we compile directly using santhosh-tekuri/jsonschema.
	compiled, err := compileJSONSchema(schema)
	if err != nil {
		// Schema compilation failed — return nil (skip validation).
		// This matches the TS behavior where malformed schemas are silently skipped.
		return nil
	}

	validator := &jsonSchemaValidator{schema: compiled}

	// Cache the compiled validator.
	sv.mu.Lock()
	sv.cache[cacheKey] = validator
	sv.mu.Unlock()

	return validator
}

// compileJSONSchema compiles a JSONSchema7 map into a *jsonschema.Schema.
func compileJSONSchema(schema JSONSchema7) (*jsonschema.Schema, error) {
	if schema == nil {
		return nil, fmt.Errorf("schema is nil")
	}

	// Marshal schema to JSON bytes for the compiler.
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Unmarshal into the compiler's expected format.
	var schemaDoc any
	if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	// Create a compiler and compile the schema.
	// Use an in-memory URL since the library requires a URL for the schema.
	c := jsonschema.NewCompiler()
	if err := c.AddResource("schema.json", schemaDoc); err != nil {
		return nil, fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiled, err := c.Compile("schema.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema: %w", err)
	}

	return compiled, nil
}

// ClearCache clears a cached validator (call when schema changes).
func (sv *SchemaValidator) ClearCache(cacheKey string) {
	sv.mu.Lock()
	delete(sv.cache, cacheKey)
	sv.mu.Unlock()
}

// Validate validates data against a schema.
// Returns nil on success, or a *SchemaValidationError on failure.
func (sv *SchemaValidator) Validate(data any, schema JSONSchema7, field string, cacheKey string) error {
	validator := sv.getValidator(schema, cacheKey)
	if validator == nil {
		// No validator compiled — skip validation.
		return nil
	}

	fieldErrors := validator.Validate(data)
	if len(fieldErrors) > 0 {
		return NewSchemaValidationError(field, fieldErrors)
	}
	return nil
}

// ValidateBatch validates multiple items, returning a valid/invalid split.
func (sv *SchemaValidator) ValidateBatch(
	items []BatchItem,
	inputSchema JSONSchema7,
	outputSchema JSONSchema7,
	cacheKeyPrefix string,
	maxErrors int,
) BatchValidationResult {
	if maxErrors <= 0 {
		maxErrors = 10
	}

	result := BatchValidationResult{
		Valid:   make([]ValidItem, 0),
		Invalid: make([]InvalidItem, 0),
	}

	// Pre-compile schemas for performance.
	var inputValidator ZodSchema
	var outputValidator ZodSchema
	if inputSchema != nil {
		inputValidator = sv.getValidator(inputSchema, cacheKeyPrefix+":input")
	}
	if outputSchema != nil {
		outputValidator = sv.getValidator(outputSchema, cacheKeyPrefix+":output")
	}

	for i, item := range items {
		hasError := false

		// Validate input if schema enabled.
		if inputValidator != nil {
			fieldErrors := inputValidator.Validate(item.Input)
			if len(fieldErrors) > 0 {
				result.Invalid = append(result.Invalid, InvalidItem{
					Index:  i,
					Data:   item,
					Field:  "input",
					Errors: fieldErrors,
				})
				hasError = true
				if len(result.Invalid) >= maxErrors {
					break
				}
			}
		}

		// Validate groundTruth if schema enabled and value provided.
		if !hasError && outputValidator != nil && item.GroundTruth != nil {
			fieldErrors := outputValidator.Validate(item.GroundTruth)
			if len(fieldErrors) > 0 {
				result.Invalid = append(result.Invalid, InvalidItem{
					Index:  i,
					Data:   item,
					Field:  "groundTruth",
					Errors: fieldErrors,
				})
				hasError = true
				if len(result.Invalid) >= maxErrors {
					break
				}
			}
		}

		if !hasError {
			result.Valid = append(result.Valid, ValidItem{
				Index: i,
				Data:  item,
			})
		}
	}

	return result
}

// BatchItem is the input shape for batch validation.
type BatchItem struct {
	Input       any `json:"input"`
	GroundTruth any `json:"groundTruth,omitempty"`
}

// ============================================================================
// Singleton
// ============================================================================

var (
	validatorOnce     sync.Once
	validatorInstance  *SchemaValidator
)

// GetSchemaValidator returns the singleton validator instance.
func GetSchemaValidator() *SchemaValidator {
	validatorOnce.Do(func() {
		validatorInstance = NewSchemaValidator()
	})
	return validatorInstance
}

// CreateValidator creates a new validator (for testing).
func CreateValidator() *SchemaValidator {
	return NewSchemaValidator()
}
