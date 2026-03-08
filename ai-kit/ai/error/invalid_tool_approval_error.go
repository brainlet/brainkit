// Ported from: packages/ai/src/error/invalid-tool-approval-error.ts
package aierror

import "fmt"

const invalidToolApprovalErrorName = "AI_InvalidToolApprovalError"
const invalidToolApprovalErrorMarker = "vercel.ai.error." + invalidToolApprovalErrorName

// InvalidToolApprovalError is returned when a tool approval response references
// an unknown approvalId with no matching tool-approval-request in message history.
type InvalidToolApprovalError struct {
	AISDKError

	// ApprovalID is the approval ID that was not found.
	ApprovalID string
}

// NewInvalidToolApprovalError creates a new InvalidToolApprovalError.
func NewInvalidToolApprovalError(approvalID string) *InvalidToolApprovalError {
	return &InvalidToolApprovalError{
		AISDKError: AISDKError{
			Name: invalidToolApprovalErrorName,
			Message: fmt.Sprintf(
				`Tool approval response references unknown approvalId: "%s". `+
					`No matching tool-approval-request found in message history.`,
				approvalID,
			),
		},
		ApprovalID: approvalID,
	}
}

// IsInvalidToolApprovalError checks whether the given error is an InvalidToolApprovalError.
func IsInvalidToolApprovalError(err error) bool {
	_, ok := err.(*InvalidToolApprovalError)
	return ok
}
