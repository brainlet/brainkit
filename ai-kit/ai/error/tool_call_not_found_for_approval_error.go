// Ported from: packages/ai/src/error/tool-call-not-found-for-approval-error.ts
package aierror

import "fmt"

const toolCallNotFoundForApprovalErrorName = "AI_ToolCallNotFoundForApprovalError"
const toolCallNotFoundForApprovalErrorMarker = "vercel.ai.error." + toolCallNotFoundForApprovalErrorName

// ToolCallNotFoundForApprovalError is returned when a tool call is not found
// for the given approval request.
type ToolCallNotFoundForApprovalError struct {
	AISDKError

	// ToolCallID is the ID of the tool call that was not found.
	ToolCallID string

	// ApprovalID is the ID of the approval request.
	ApprovalID string
}

// NewToolCallNotFoundForApprovalError creates a new ToolCallNotFoundForApprovalError.
func NewToolCallNotFoundForApprovalError(toolCallID, approvalID string) *ToolCallNotFoundForApprovalError {
	return &ToolCallNotFoundForApprovalError{
		AISDKError: AISDKError{
			Name: toolCallNotFoundForApprovalErrorName,
			Message: fmt.Sprintf(
				`Tool call "%s" not found for approval request "%s".`,
				toolCallID, approvalID,
			),
		},
		ToolCallID: toolCallID,
		ApprovalID: approvalID,
	}
}

// IsToolCallNotFoundForApprovalError checks whether the given error is a ToolCallNotFoundForApprovalError.
func IsToolCallNotFoundForApprovalError(err error) bool {
	_, ok := err.(*ToolCallNotFoundForApprovalError)
	return ok
}
