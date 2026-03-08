// Ported from: packages/ai/src/generate-text/tool-output.ts
package generatetext

// ToolOutput is a union type representing the result of a tool execution.
// It can be either a ToolResult or a ToolError.
//
// In Go, we represent this as an interface with a discriminator method.
type ToolOutput interface {
	// ToolOutputType returns "tool-result" or "tool-error".
	ToolOutputType() string
}

// Ensure ToolResult and ToolError implement ToolOutput.
func (tr *ToolResult) ToolOutputType() string { return tr.Type }
func (te *ToolError) ToolOutputType() string  { return te.Type }
