// Ported from: packages/ai/src/generate-text/tool-output-denied.ts
package generatetext

// ToolOutputDenied represents a tool output when execution has been denied.
type ToolOutputDenied struct {
	// Type is always "tool-output-denied".
	Type string

	// ToolCallID is the ID of the denied tool call.
	ToolCallID string

	// ToolName is the name of the tool.
	ToolName string

	// ProviderExecuted indicates whether the tool was provider-executed.
	ProviderExecuted bool

	// Dynamic indicates whether this is a dynamic tool denial.
	Dynamic bool
}
