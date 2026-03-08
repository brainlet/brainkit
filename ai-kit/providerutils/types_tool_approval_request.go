// Ported from: packages/provider-utils/src/types/tool-approval-request.ts
package providerutils

// ToolApprovalRequest represents a tool approval request prompt part.
type ToolApprovalRequest struct {
	Type       string `json:"type"` // "tool-approval-request"
	ApprovalID string `json:"approvalId"`
	ToolCallID string `json:"toolCallId"`
}
