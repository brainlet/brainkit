// Ported from: packages/ai/src/generate-text/tool-approval-request-output.ts
package generatetext

// ToolApprovalRequestOutput indicates that a tool approval request has been made.
// The tool approval request can be approved or denied in the next tool message.
type ToolApprovalRequestOutput struct {
	// Type is always "tool-approval-request".
	Type string

	// ApprovalID is the ID of the tool approval request.
	ApprovalID string

	// ToolCall is the tool call that the approval request is for.
	ToolCall ToolCall
}
