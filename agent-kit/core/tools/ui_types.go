// Ported from: packages/core/src/tools/ui-types.ts
package tools

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// ToolRef is a stub for ./tool.Tool (the Mastra Tool class).
// Stub: real Tool class has generic schema parameters not expressible in Go;
// kept as any for ToolSet map values.
// In TypeScript: Tool<TSchemaIn, TSchemaOut, TSuspendSchema, TResumeSchema>
type ToolRef = any

// ---------------------------------------------------------------------------
// UI Tool Types
// ---------------------------------------------------------------------------

// UITool represents a UI tool type for use with AI SDK frontend components.
type UITool struct {
	Input  any `json:"input"`
	Output any `json:"output,omitempty"`
}

// ToolSet is a named set of tools (object with tool instances).
// In TypeScript: Record<string, Tool>
type ToolSet map[string]ToolRef

// UITools is a set of UI tool type definitions for frontend components.
// In TypeScript: Record<string, UITool>
type UITools map[string]UITool

// ---------------------------------------------------------------------------
// NOTE: The following TypeScript utility types are not directly portable to Go
// because Go does not have structural type inference generics:
//
//   InferToolInput<T>   — extracts input type from a Tool<I, O, ...>
//   InferToolOutput<T>  — extracts output type from a Tool<I, O, ...>
//   InferUITool<TOOL>   — maps a Tool to UITool{ input, output }
//   InferUITools<TOOLS> — maps a ToolSet to UITools
//
// In Go, callers should use type assertions or concrete types instead.
// ---------------------------------------------------------------------------
