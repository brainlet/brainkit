// Ported from: packages/provider/src/language-model/v3/language-model-v3-tool-approval-request.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// ToolApprovalRequest is emitted by a provider for a provider-executed tool call.
//
// This is used for flows where the provider executes the tool (e.g. MCP tools)
// but requires an explicit user approval before continuing.
type ToolApprovalRequest struct {
	// ApprovalID is the ID of the approval request. Referenced by the subsequent
	// tool-approval-response (tool message) to approve or deny execution.
	ApprovalID string

	// ToolCallID is the tool call ID that this approval request is for.
	ToolCallID string

	// ProviderMetadata is additional provider-specific metadata for the approval request.
	ProviderMetadata shared.ProviderMetadata
}

func (ToolApprovalRequest) isContent()    {}
func (ToolApprovalRequest) isStreamPart() {}
