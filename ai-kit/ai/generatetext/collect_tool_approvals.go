// Ported from: packages/ai/src/generate-text/collect-tool-approvals.ts
package generatetext

import "fmt"

// CollectedToolApproval holds a tool approval request, its response, and the associated tool call.
type CollectedToolApproval struct {
	ApprovalRequest  ToolApprovalRequest
	ApprovalResponse ToolApprovalResponse
	ToolCall         ToolCall
}

// CollectedToolApprovals holds the results of collecting tool approvals.
type CollectedToolApprovals struct {
	ApprovedToolApprovals []CollectedToolApproval
	DeniedToolApprovals   []CollectedToolApproval
}

// CollectToolApprovals collects tool approvals from the last tool message.
// If the last message is not a tool message, returns empty slices.
func CollectToolApprovals(messages []ModelMessage) (CollectedToolApprovals, error) {
	result := CollectedToolApprovals{}

	if len(messages) == 0 {
		return result, nil
	}

	lastMessage := messages[len(messages)-1]
	if lastMessage.Role != "tool" {
		return result, nil
	}

	// Gather tool calls from all messages
	toolCallsByToolCallID := map[string]ToolCall{}
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		parts, ok := msg.Content.([]ModelMessageContent)
		if !ok {
			continue
		}
		for _, part := range parts {
			if part.Type == "tool-call" {
				toolCallsByToolCallID[part.ToolCallID] = ToolCall{
					Type:       "tool-call",
					ToolCallID: part.ToolCallID,
					ToolName:   part.ToolName,
					Input:      part.Input,
				}
			}
		}
	}

	// Gather approval requests
	toolApprovalRequestsByApprovalID := map[string]ToolApprovalRequest{}
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		parts, ok := msg.Content.([]ModelMessageContent)
		if !ok {
			continue
		}
		for _, part := range parts {
			if part.Type == "tool-approval-request" {
				toolApprovalRequestsByApprovalID[part.ApprovalID] = ToolApprovalRequest{
					Type:       "tool-approval-request",
					ApprovalID: part.ApprovalID,
					ToolCallID: part.ToolCallID,
				}
			}
		}
	}

	// Gather tool results from the last tool message
	toolResults := map[string]bool{}
	lastParts, ok := lastMessage.Content.([]ModelMessageContent)
	if !ok {
		return result, nil
	}
	for _, part := range lastParts {
		if part.Type == "tool-result" {
			toolResults[part.ToolCallID] = true
		}
	}

	// Process approval responses
	for _, part := range lastParts {
		if part.Type != "tool-approval-response" {
			continue
		}

		approvalRequest, ok := toolApprovalRequestsByApprovalID[part.ApprovalID]
		if !ok {
			return result, &InvalidToolApprovalError{ApprovalID: part.ApprovalID}
		}

		// Skip if there's already a tool result for this tool call
		if toolResults[approvalRequest.ToolCallID] {
			continue
		}

		toolCall, ok := toolCallsByToolCallID[approvalRequest.ToolCallID]
		if !ok {
			return result, &ToolCallNotFoundForApprovalError{
				ToolCallID: approvalRequest.ToolCallID,
				ApprovalID: approvalRequest.ApprovalID,
			}
		}

		approval := CollectedToolApproval{
			ApprovalRequest: approvalRequest,
			ApprovalResponse: ToolApprovalResponse{
				Type:       "tool-approval-response",
				ApprovalID: part.ApprovalID,
				Approved:   part.Approved,
				Reason:     part.Reason,
			},
			ToolCall: toolCall,
		}

		if part.Approved {
			result.ApprovedToolApprovals = append(result.ApprovedToolApprovals, approval)
		} else {
			result.DeniedToolApprovals = append(result.DeniedToolApprovals, approval)
		}
	}

	return result, nil
}

// --- Error types ---

// InvalidToolApprovalError indicates an approval response references a non-existent approval request.
type InvalidToolApprovalError struct {
	ApprovalID string
}

func (e *InvalidToolApprovalError) Error() string {
	return fmt.Sprintf("invalid tool approval: approval ID %s not found", e.ApprovalID)
}

// ToolCallNotFoundForApprovalError indicates an approval references a non-existent tool call.
type ToolCallNotFoundForApprovalError struct {
	ToolCallID string
	ApprovalID string
}

func (e *ToolCallNotFoundForApprovalError) Error() string {
	return fmt.Sprintf("tool call %s not found for approval %s", e.ToolCallID, e.ApprovalID)
}
