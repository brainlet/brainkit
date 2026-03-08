// Ported from: packages/ai/src/generate-text/tool-result.ts
package generatetext

// ToolResult represents the result of a tool execution.
// In TypeScript this is a union of StaticToolResult | DynamicToolResult.
type ToolResult struct {
	// Type is always "tool-result".
	Type string

	// ToolCallID is the ID of the tool call that produced this result.
	ToolCallID string

	// ToolName is the name of the tool.
	ToolName string

	// Input is the input that was provided to the tool.
	Input interface{}

	// Output is the result produced by the tool.
	Output interface{}

	// ProviderExecuted indicates whether the tool was executed by the provider.
	ProviderExecuted bool

	// ProviderMetadata contains provider-specific metadata.
	ProviderMetadata ProviderMetadata

	// Dynamic indicates whether this is a dynamic tool result.
	Dynamic bool

	// Preliminary indicates whether this is a preliminary (streaming) result.
	Preliminary bool

	// Title is an optional human-readable title.
	Title string
}

// IsStatic returns true if this result is from a static tool.
func (tr *ToolResult) IsStatic() bool {
	return !tr.Dynamic
}

// IsDynamic returns true if this result is from a dynamic tool.
func (tr *ToolResult) IsDynamic() bool {
	return tr.Dynamic
}
