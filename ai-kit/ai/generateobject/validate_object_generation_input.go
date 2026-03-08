// Ported from: packages/ai/src/generate-object/validate-object-generation-input.ts
package generateobject

import "fmt"

// ValidateObjectGenerationInputOptions are the options for ValidateObjectGenerationInput.
type ValidateObjectGenerationInputOptions struct {
	Output            string // "object", "array", "enum", "no-schema"
	Schema            any
	SchemaName        string
	SchemaDescription string
	EnumValues        []string
}

// ValidateObjectGenerationInput validates the input parameters for object generation.
// Returns an error if the input is invalid.
func ValidateObjectGenerationInput(opts ValidateObjectGenerationInputOptions) error {
	output := opts.Output

	if output != "" && output != "object" && output != "array" && output != "enum" && output != "no-schema" {
		return fmt.Errorf("invalid argument: output=%q: invalid output type", output)
	}

	if output == "no-schema" {
		if opts.Schema != nil {
			return fmt.Errorf("invalid argument: schema is not supported for no-schema output")
		}
		if opts.SchemaDescription != "" {
			return fmt.Errorf("invalid argument: schema description is not supported for no-schema output")
		}
		if opts.SchemaName != "" {
			return fmt.Errorf("invalid argument: schema name is not supported for no-schema output")
		}
		if opts.EnumValues != nil {
			return fmt.Errorf("invalid argument: enum values are not supported for no-schema output")
		}
	}

	if output == "object" {
		if opts.Schema == nil {
			return fmt.Errorf("invalid argument: schema is required for object output")
		}
		if opts.EnumValues != nil {
			return fmt.Errorf("invalid argument: enum values are not supported for object output")
		}
	}

	if output == "array" {
		if opts.Schema == nil {
			return fmt.Errorf("invalid argument: element schema is required for array output")
		}
		if opts.EnumValues != nil {
			return fmt.Errorf("invalid argument: enum values are not supported for array output")
		}
	}

	if output == "enum" {
		if opts.Schema != nil {
			return fmt.Errorf("invalid argument: schema is not supported for enum output")
		}
		if opts.SchemaDescription != "" {
			return fmt.Errorf("invalid argument: schema description is not supported for enum output")
		}
		if opts.SchemaName != "" {
			return fmt.Errorf("invalid argument: schema name is not supported for enum output")
		}
		if opts.EnumValues == nil {
			return fmt.Errorf("invalid argument: enum values are required for enum output")
		}
		for _, v := range opts.EnumValues {
			if v == "" {
				return fmt.Errorf("invalid argument: enum values must be non-empty strings")
			}
		}
	}

	return nil
}
