// Ported from: packages/ai/src/generate-text/content-part.ts
package generatetext

// ContentPart is a union type representing different kinds of content in a generation result.
// In TypeScript this is a discriminated union; in Go we use a struct with a Type discriminator.
type ContentPart struct {
	// Type discriminates the content part kind:
	// "text", "reasoning", "source", "file", "tool-call", "tool-result", "tool-error", "tool-approval-request"
	Type string

	// For "text" parts
	Text string

	// For "reasoning" parts (uses Text field)

	// For "source" parts
	Source *Source

	// For "file" parts
	File GeneratedFile

	// For "tool-call" parts
	ToolCall *ToolCall

	// For "tool-result" parts
	ToolResult *ToolResult

	// For "tool-error" parts
	ToolError *ToolError

	// For "tool-approval-request" parts
	ToolApprovalRequest *ToolApprovalRequestOutput

	// Common metadata
	ProviderMetadata ProviderMetadata
}

// NewTextContentPart creates a text content part.
func NewTextContentPart(text string, providerMetadata ProviderMetadata) ContentPart {
	return ContentPart{
		Type:             "text",
		Text:             text,
		ProviderMetadata: providerMetadata,
	}
}

// NewReasoningContentPart creates a reasoning content part.
func NewReasoningContentPart(text string, providerMetadata ProviderMetadata) ContentPart {
	return ContentPart{
		Type:             "reasoning",
		Text:             text,
		ProviderMetadata: providerMetadata,
	}
}

// NewSourceContentPart creates a source content part.
func NewSourceContentPart(source Source) ContentPart {
	return ContentPart{
		Type:   "source",
		Source: &source,
	}
}

// NewFileContentPart creates a file content part.
func NewFileContentPart(file GeneratedFile, providerMetadata ProviderMetadata) ContentPart {
	return ContentPart{
		Type:             "file",
		File:             file,
		ProviderMetadata: providerMetadata,
	}
}

// NewToolCallContentPart creates a tool-call content part.
func NewToolCallContentPart(tc ToolCall) ContentPart {
	return ContentPart{
		Type:             "tool-call",
		ToolCall:         &tc,
		ProviderMetadata: tc.ProviderMetadata,
	}
}

// NewToolResultContentPart creates a tool-result content part.
func NewToolResultContentPart(tr ToolResult) ContentPart {
	return ContentPart{
		Type:             "tool-result",
		ToolResult:       &tr,
		ProviderMetadata: tr.ProviderMetadata,
	}
}

// NewToolErrorContentPart creates a tool-error content part.
func NewToolErrorContentPart(te ToolError) ContentPart {
	return ContentPart{
		Type:             "tool-error",
		ToolError:        &te,
		ProviderMetadata: te.ProviderMetadata,
	}
}

// NewToolApprovalRequestContentPart creates a tool-approval-request content part.
func NewToolApprovalRequestContentPart(tar ToolApprovalRequestOutput) ContentPart {
	return ContentPart{
		Type:                "tool-approval-request",
		ToolApprovalRequest: &tar,
	}
}
