// Ported from: packages/core/src/stream/base/schema.ts
package base

import "encoding/json"

// ---------------------------------------------------------------------------
// JSON Schema types
// ---------------------------------------------------------------------------

// JSONSchema7 is a simplified representation of JSON Schema Draft 7.
// In TS this is imported from @internal/ai-sdk-v5.
// Simplified representation — covers the subset used by Mastra's structured output.
type JSONSchema7 struct {
	Schema               string                 `json:"$schema,omitempty"`
	Type                 string                 `json:"type,omitempty"`
	Properties           map[string]*JSONSchema7 `json:"properties,omitempty"`
	Required             []string               `json:"required,omitempty"`
	AdditionalProperties *bool                  `json:"additionalProperties,omitempty"`
	Items                *JSONSchema7           `json:"items,omitempty"`
	Enum                 []any                  `json:"enum,omitempty"`
	// Extra catches any other JSON Schema fields not explicitly modeled.
	Extra map[string]any `json:"-"`
}

// MarshalJSON provides custom marshaling that includes Extra fields.
func (s JSONSchema7) MarshalJSON() ([]byte, error) {
	type alias JSONSchema7
	data, err := json.Marshal(alias(s))
	if err != nil {
		return nil, err
	}
	if len(s.Extra) == 0 {
		return data, nil
	}
	// Merge extra fields
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	for k, v := range s.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// ---------------------------------------------------------------------------
// Schema types — Go equivalents of TS conditional/union types
// ---------------------------------------------------------------------------

// OutputSchema represents the union of schema types that can be provided
// for structured output. In TS this is:
//
//	z4.ZodType | z3.Schema | Schema<OBJECT> | JSONSchema7 | undefined
//
// In Go without Zod or AI SDK Schema types, we use any.
// Callers should pass *JSONSchema7 or nil.
// Zod-like validation not applicable in Go; callers pass *JSONSchema7 or nil.
type OutputSchema = any

// PartialSchemaOutput is the Go equivalent of TS PartialSchemaOutput<OUTPUT>.
// Since Go doesn't support conditional types, this is simply any.
type PartialSchemaOutput = any

// InferSchemaOutput infers the output type from a schema.
// In Go without generics on type aliases, this is any.
type InferSchemaOutput = any

// SchemaWithValidation wraps a schema that can validate data.
// Mirrors TS ZodLikeSchema which has safeParse(data) method.
// Zod-compat not applicable in Go; schemas implement SafeParser interface.
type SchemaWithValidation struct {
	Schema any
}

// ZodLikePartialSchema mirrors the TS type for partial schema validation.
// Zod partial validation not applicable in Go; struct wraps SafeParse func.
type ZodLikePartialSchema struct {
	SafeParse func(value any) SafeParseResult
}

// SafeParseResult is the result of a safeParse validation call.
type SafeParseResult struct {
	Success bool
	Data    any
	Error   error
}

// ---------------------------------------------------------------------------
// TransformedSchemaResult
// ---------------------------------------------------------------------------

// TransformedSchemaResult is returned by GetTransformedSchema.
// It contains the potentially-wrapped JSON schema and the detected output format.
type TransformedSchemaResult struct {
	// JSONSchema is the (possibly wrapped) schema for LLM generation.
	JSONSchema *JSONSchema7
	// OutputFormat is "array", "enum", or the schema's type (usually "object").
	OutputFormat string
}

// ---------------------------------------------------------------------------
// AsJsonSchema — convert an OutputSchema to JSONSchema7
// ---------------------------------------------------------------------------

// AsJsonSchema converts an OutputSchema to a *JSONSchema7.
// In TS this handles Zod schemas, AI SDK Schema types, and plain JSONSchema7.
// In Go we only handle *JSONSchema7 directly; other schema types would need
// adapters once ported.
//
// Returns nil if schema is nil or not a recognized type.
func AsJsonSchema(schema OutputSchema) *JSONSchema7 {
	if schema == nil {
		return nil
	}

	// Direct *JSONSchema7
	if js, ok := schema.(*JSONSchema7); ok {
		return js
	}

	// Zod-like schemas and AI SDK Schema types are TS-specific;
	// in Go, callers should pass *JSONSchema7 directly.

	return nil
}

// ---------------------------------------------------------------------------
// GetTransformedSchema — wrap array/enum schemas for better LLM generation
// ---------------------------------------------------------------------------

// GetTransformedSchema analyzes a schema and potentially wraps it for LLM generation.
//
// - Array schemas are wrapped in {elements: [...]} for reliable generation.
// - Enum schemas are wrapped in {result: ""} for reliable generation.
// - Object schemas are passed through unchanged.
//
// Returns nil if the schema is nil or cannot be converted to JSON Schema.
func GetTransformedSchema(schema OutputSchema) *TransformedSchemaResult {
	jsonSchema := AsJsonSchema(schema)
	if jsonSchema == nil {
		return nil
	}

	// Strip $schema and work with the item schema
	dollarSchema := jsonSchema.Schema
	itemSchema := &JSONSchema7{
		Type:                 jsonSchema.Type,
		Properties:           jsonSchema.Properties,
		Required:             jsonSchema.Required,
		AdditionalProperties: jsonSchema.AdditionalProperties,
		Items:                jsonSchema.Items,
		Enum:                 jsonSchema.Enum,
		Extra:                jsonSchema.Extra,
	}

	// Handle array type: wrap in {elements: [...]}
	if itemSchema.Type == "array" {
		innerElement := itemSchema.Items
		falseVal := false
		arrayOutputSchema := &JSONSchema7{
			Schema: dollarSchema,
			Type:   "object",
			Properties: map[string]*JSONSchema7{
				"elements": {
					Type:  "array",
					Items: innerElement,
				},
			},
			Required:             []string{"elements"},
			AdditionalProperties: &falseVal,
		}
		return &TransformedSchemaResult{
			JSONSchema:   arrayOutputSchema,
			OutputFormat: "array",
		}
	}

	// Handle enum type: wrap in {result: ""}
	if len(itemSchema.Enum) > 0 {
		enumType := itemSchema.Type
		if enumType == "" {
			enumType = "string"
		}
		falseVal := false
		enumOutputSchema := &JSONSchema7{
			Schema: dollarSchema,
			Type:   "object",
			Properties: map[string]*JSONSchema7{
				"result": {
					Type: enumType,
					Enum: itemSchema.Enum,
				},
			},
			Required:             []string{"result"},
			AdditionalProperties: &falseVal,
		}
		return &TransformedSchemaResult{
			JSONSchema:   enumOutputSchema,
			OutputFormat: "enum",
		}
	}

	// Default: return as-is (usually "object")
	return &TransformedSchemaResult{
		JSONSchema:   jsonSchema,
		OutputFormat: jsonSchema.Type,
	}
}

// ---------------------------------------------------------------------------
// GetResponseFormat — determine response format from schema
// ---------------------------------------------------------------------------

// ResponseFormatType is either "text" or "json".
type ResponseFormatType string

const (
	ResponseFormatText ResponseFormatType = "text"
	ResponseFormatJSON ResponseFormatType = "json"
)

// ResponseFormat describes the desired response format for an LLM call.
type ResponseFormat struct {
	Type   ResponseFormatType `json:"type"`
	Schema *JSONSchema7       `json:"schema,omitempty"`
}

// GetResponseFormat returns the response format configuration based on the schema.
// If a schema is provided, returns JSON format with the transformed schema.
// Otherwise returns text format.
func GetResponseFormat(schema OutputSchema) ResponseFormat {
	if schema != nil {
		transformedSchema := GetTransformedSchema(schema)
		var js *JSONSchema7
		if transformedSchema != nil {
			js = transformedSchema.JSONSchema
		}
		return ResponseFormat{
			Type:   ResponseFormatJSON,
			Schema: js,
		}
	}

	return ResponseFormat{
		Type: ResponseFormatText,
	}
}
