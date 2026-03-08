// Ported from: packages/provider-utils/src/inject-json-instruction.ts
package providerutils

import (
	"encoding/json"
	"strings"
)

const defaultSchemaPrefix = "JSON schema:"
const defaultSchemaSuffix = "You MUST answer with a JSON object that matches the JSON schema above."
const defaultGenericSuffix = "You MUST answer with JSON."

// InjectJsonInstructionOptions are the options for InjectJsonInstruction.
type InjectJsonInstructionOptions struct {
	// Prompt is the original prompt text.
	Prompt *string
	// Schema is the JSON schema to include. Can be nil.
	Schema interface{}
	// SchemaPrefix is the prefix before the schema. Defaults based on whether schema is provided.
	SchemaPrefix *string
	// SchemaSuffix is the suffix after the schema. Defaults based on whether schema is provided.
	SchemaSuffix *string
}

// InjectJsonInstruction creates a prompt string with JSON instruction injected.
func InjectJsonInstruction(opts InjectJsonInstructionOptions) string {
	var schemaPrefix *string
	var schemaSuffix *string

	if opts.SchemaPrefix != nil {
		schemaPrefix = opts.SchemaPrefix
	} else if opts.Schema != nil {
		p := defaultSchemaPrefix
		schemaPrefix = &p
	}

	if opts.SchemaSuffix != nil {
		schemaSuffix = opts.SchemaSuffix
	} else if opts.Schema != nil {
		s := defaultSchemaSuffix
		schemaSuffix = &s
	} else {
		s := defaultGenericSuffix
		schemaSuffix = &s
	}

	var lines []string

	if opts.Prompt != nil && len(*opts.Prompt) > 0 {
		lines = append(lines, *opts.Prompt)
		lines = append(lines, "") // add a newline after prompt
	}

	if schemaPrefix != nil {
		lines = append(lines, *schemaPrefix)
	}

	if opts.Schema != nil {
		schemaJSON, err := json.Marshal(opts.Schema)
		if err == nil {
			lines = append(lines, string(schemaJSON))
		}
	}

	if schemaSuffix != nil {
		lines = append(lines, *schemaSuffix)
	}

	return strings.Join(lines, "\n")
}
