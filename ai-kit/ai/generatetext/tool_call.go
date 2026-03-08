// Ported from: packages/ai/src/generate-text/tool-call.ts
package generatetext

// ToolCall represents a tool call made by the model.
// In TypeScript this is a union of StaticToolCall | DynamicToolCall, discriminated by the Dynamic field.
type ToolCall struct {
	// Type is always "tool-call".
	Type string

	// ToolCallID is the unique identifier for this tool call.
	ToolCallID string

	// ToolName is the name of the tool being called.
	ToolName string

	// Input is the parsed input for the tool.
	Input interface{}

	// Dynamic indicates whether this is a dynamic tool call.
	Dynamic bool

	// Invalid is true if the tool call could not be parsed or the tool does not exist.
	Invalid bool

	// Error is the error that caused the tool call to be invalid.
	Error interface{}

	// Title is an optional human-readable title.
	Title string

	// ProviderExecuted indicates whether the tool was executed by the provider.
	ProviderExecuted bool

	// ProviderMetadata contains provider-specific metadata.
	ProviderMetadata ProviderMetadata
}

// IsStatic returns true if this tool call is static (not dynamic).
func (tc *ToolCall) IsStatic() bool {
	return !tc.Dynamic
}

// IsDynamic returns true if this tool call is dynamic.
func (tc *ToolCall) IsDynamic() bool {
	return tc.Dynamic
}
