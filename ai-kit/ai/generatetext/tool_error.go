// Ported from: packages/ai/src/generate-text/tool-error.ts
package generatetext

// ToolError represents an error that occurred during tool execution.
// In TypeScript this is a union of StaticToolError | DynamicToolError.
type ToolError struct {
	// Type is always "tool-error".
	Type string

	// ToolCallID is the ID of the tool call that produced this error.
	ToolCallID string

	// ToolName is the name of the tool.
	ToolName string

	// Input is the input that was provided to the tool.
	Input interface{}

	// Error is the error that occurred.
	Error interface{}

	// ProviderExecuted indicates whether the tool was executed by the provider.
	ProviderExecuted bool

	// ProviderMetadata contains provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Dynamic indicates whether this is a dynamic tool error.
	Dynamic bool

	// Title is an optional human-readable title.
	Title string
}

// IsStatic returns true if this error is from a static tool.
func (te *ToolError) IsStatic() bool {
	return !te.Dynamic
}

// IsDynamic returns true if this error is from a dynamic tool.
func (te *ToolError) IsDynamic() bool {
	return te.Dynamic
}
