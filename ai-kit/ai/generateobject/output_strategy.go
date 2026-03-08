// Ported from: packages/ai/src/generate-object/output-strategy.ts
package generateobject

import (
	"encoding/json"
	"fmt"
)

// OutputType represents the type of output strategy.
type OutputType string

const (
	OutputTypeObject   OutputType = "object"
	OutputTypeArray    OutputType = "array"
	OutputTypeEnum     OutputType = "enum"
	OutputTypeNoSchema OutputType = "no-schema"
)

// ValidationResult represents the result of a validation operation.
type ValidationResult struct {
	Success bool
	Value   any
	Error   error
}

// OutputStrategy defines the interface for different output strategies.
type OutputStrategy interface {
	// Type returns the output strategy type.
	Type() OutputType
	// JSONSchema returns the JSON schema for this output strategy.
	JSONSchema() (any, error)
	// ValidateFinalResult validates the final parsed result.
	ValidateFinalResult(value any) ValidationResult
}

// NoSchemaOutputStrategy is the output strategy for no-schema mode.
type NoSchemaOutputStrategy struct{}

func (s *NoSchemaOutputStrategy) Type() OutputType { return OutputTypeNoSchema }

func (s *NoSchemaOutputStrategy) JSONSchema() (any, error) { return nil, nil }

func (s *NoSchemaOutputStrategy) ValidateFinalResult(value any) ValidationResult {
	if value == nil {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("no object generated: response did not match schema"),
		}
	}
	return ValidationResult{Success: true, Value: value}
}

// ObjectOutputStrategy is the output strategy for object mode.
type ObjectOutputStrategy struct {
	Schema any // JSON Schema
}

func (s *ObjectOutputStrategy) Type() OutputType { return OutputTypeObject }

func (s *ObjectOutputStrategy) JSONSchema() (any, error) { return s.Schema, nil }

func (s *ObjectOutputStrategy) ValidateFinalResult(value any) ValidationResult {
	// In Go, we accept any valid JSON value as the result.
	// Schema validation would be done by a separate validator.
	if value == nil {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("no object generated: value is nil"),
		}
	}
	return ValidationResult{Success: true, Value: value}
}

// ArrayOutputStrategy is the output strategy for array mode.
type ArrayOutputStrategy struct {
	ItemSchema any // JSON Schema for array items
}

func (s *ArrayOutputStrategy) Type() OutputType { return OutputTypeArray }

func (s *ArrayOutputStrategy) JSONSchema() (any, error) {
	return map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]any{
			"elements": map[string]any{
				"type":  "array",
				"items": s.ItemSchema,
			},
		},
		"required":             []string{"elements"},
		"additionalProperties": false,
	}, nil
}

func (s *ArrayOutputStrategy) ValidateFinalResult(value any) ValidationResult {
	obj, ok := value.(map[string]any)
	if !ok {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("value must be an object that contains an array of elements"),
		}
	}
	elements, ok := obj["elements"]
	if !ok {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("value must be an object that contains an array of elements"),
		}
	}
	arr, ok := elements.([]any)
	if !ok {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("elements must be an array"),
		}
	}
	return ValidationResult{Success: true, Value: arr}
}

// EnumOutputStrategy is the output strategy for enum mode.
type EnumOutputStrategy struct {
	EnumValues []string
}

func (s *EnumOutputStrategy) Type() OutputType { return OutputTypeEnum }

func (s *EnumOutputStrategy) JSONSchema() (any, error) {
	enumVals := make([]any, len(s.EnumValues))
	for i, v := range s.EnumValues {
		enumVals[i] = v
	}
	return map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"properties": map[string]any{
			"result": map[string]any{
				"type": "string",
				"enum": enumVals,
			},
		},
		"required":             []string{"result"},
		"additionalProperties": false,
	}, nil
}

func (s *EnumOutputStrategy) ValidateFinalResult(value any) ValidationResult {
	obj, ok := value.(map[string]any)
	if !ok {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("value must be an object that contains a string in the result property"),
		}
	}
	result, ok := obj["result"].(string)
	if !ok {
		return ValidationResult{
			Success: false,
			Error:   fmt.Errorf("value must be an object that contains a string in the result property"),
		}
	}
	for _, v := range s.EnumValues {
		if v == result {
			return ValidationResult{Success: true, Value: result}
		}
	}
	return ValidationResult{
		Success: false,
		Error:   fmt.Errorf("value must be a string in the enum"),
	}
}

// GetOutputStrategy returns the appropriate output strategy for the given output type.
func GetOutputStrategy(output OutputType, schema any, enumValues []string) (OutputStrategy, error) {
	switch output {
	case OutputTypeObject:
		return &ObjectOutputStrategy{Schema: schema}, nil
	case OutputTypeArray:
		return &ArrayOutputStrategy{ItemSchema: schema}, nil
	case OutputTypeEnum:
		return &EnumOutputStrategy{EnumValues: enumValues}, nil
	case OutputTypeNoSchema:
		return &NoSchemaOutputStrategy{}, nil
	default:
		return nil, fmt.Errorf("unsupported output type: %s", output)
	}
}

// ParseJSON parses a JSON string into a Go value.
func ParseJSON(text string) (any, error) {
	var result any
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, err
	}
	return result, nil
}
