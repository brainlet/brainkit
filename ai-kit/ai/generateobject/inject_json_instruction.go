// Ported from: packages/ai/src/generate-object/inject-json-instruction.ts
package generateobject

import (
	"encoding/json"
	"strings"
)

const defaultSchemaPrefix = "JSON schema:"
const defaultSchemaSuffix = "You MUST answer with a JSON object that matches the JSON schema above."
const defaultGenericSuffix = "You MUST answer with JSON."

// InjectJsonInstructionOptions are the options for InjectJsonInstruction.
type InjectJsonInstructionOptions struct {
	Prompt       string
	Schema       any
	SchemaPrefix *string
	SchemaSuffix *string
}

// InjectJsonInstruction constructs a prompt with JSON schema instructions.
func InjectJsonInstruction(opts InjectJsonInstructionOptions) string {
	var schemaPrefix string
	var schemaSuffix string

	if opts.SchemaPrefix != nil {
		schemaPrefix = *opts.SchemaPrefix
	} else if opts.Schema != nil {
		schemaPrefix = defaultSchemaPrefix
	}

	if opts.SchemaSuffix != nil {
		schemaSuffix = *opts.SchemaSuffix
	} else if opts.Schema != nil {
		schemaSuffix = defaultSchemaSuffix
	} else {
		schemaSuffix = defaultGenericSuffix
	}

	var parts []string

	if len(opts.Prompt) > 0 {
		parts = append(parts, opts.Prompt)
		parts = append(parts, "") // add a newline
	}

	if schemaPrefix != "" {
		parts = append(parts, schemaPrefix)
	}

	if opts.Schema != nil {
		schemaJSON, err := json.Marshal(opts.Schema)
		if err == nil {
			parts = append(parts, string(schemaJSON))
		}
	}

	if schemaSuffix != "" {
		parts = append(parts, schemaSuffix)
	}

	return strings.Join(parts, "\n")
}
