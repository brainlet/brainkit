// Ported from: packages/core/src/zod-to-json.ts
package core

// In TypeScript, this file re-exports zodToJsonSchema from @mastra/schema-compat/zod-to-json.
// In Go, there is no Zod schema system. Schema conversion is handled through
// the JSON Schema types defined in the types package (types/zod_compat.go).
//
// This file exists to maintain 1:1 structural parity with the TypeScript source.
// See types.JSONSchema and related types for Go JSON Schema representation.

// SchemaToJSON converts a Go struct schema description to a JSON Schema map.
// This is the Go equivalent of zodToJsonSchema.
//
// TODO: implement full JSON Schema conversion once the schema system is ported.
// For now, callers should use encoding/json with struct tags or construct
// map[string]any representations directly.
func SchemaToJSON(schema any) map[string]any {
	if schema == nil {
		return map[string]any{"type": "object"}
	}

	// If it's already a map, return it directly
	if m, ok := schema.(map[string]any); ok {
		return m
	}

	// Fallback: return a generic object schema
	return map[string]any{"type": "object"}
}
