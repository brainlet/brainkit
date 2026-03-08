// Ported from: packages/ai/src/generate-text/collect-tool-approvals.test.ts
package generatetext

import (
	"testing"
)

func TestCollectToolApprovals_NoApprovals_WhenLastMessageNotTool(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{Role: "user", Content: "Hello, world!"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 0 {
		t.Errorf("expected 0 approved, got %d", len(result.ApprovedToolApprovals))
	}
	if len(result.DeniedToolApprovals) != 0 {
		t.Errorf("expected 0 denied, got %d", len(result.DeniedToolApprovals))
	}
}

func TestCollectToolApprovals_IgnoreApprovalRequestWithoutResponse(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role:    "tool",
			Content: []ModelMessageContent{},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 0 {
		t.Errorf("expected 0 approved, got %d", len(result.ApprovedToolApprovals))
	}
	if len(result.DeniedToolApprovals) != 0 {
		t.Errorf("expected 0 denied, got %d", len(result.DeniedToolApprovals))
	}
}

func TestCollectToolApprovals_ApprovedResponse(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: true},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 1 {
		t.Fatalf("expected 1 approved, got %d", len(result.ApprovedToolApprovals))
	}
	approval := result.ApprovedToolApprovals[0]
	if approval.ApprovalRequest.ApprovalID != "approval-id-1" {
		t.Errorf("expected approval-id-1, got %q", approval.ApprovalRequest.ApprovalID)
	}
	if approval.ToolCall.ToolName != "tool1" {
		t.Errorf("expected tool1, got %q", approval.ToolCall.ToolName)
	}
	if len(result.DeniedToolApprovals) != 0 {
		t.Errorf("expected 0 denied, got %d", len(result.DeniedToolApprovals))
	}
}

func TestCollectToolApprovals_ApprovedWithToolResult(t *testing.T) {
	// If there's already a tool result, the approval should not be returned
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: true},
				{Type: "tool-result", ToolCallID: "call-1", ToolName: "tool1", Output: map[string]interface{}{"type": "text", "value": "test-output"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 0 {
		t.Errorf("expected 0 approved (already has result), got %d", len(result.ApprovedToolApprovals))
	}
	if len(result.DeniedToolApprovals) != 0 {
		t.Errorf("expected 0 denied, got %d", len(result.DeniedToolApprovals))
	}
}

func TestCollectToolApprovals_DeniedResponse(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: false, Reason: "test-reason"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 0 {
		t.Errorf("expected 0 approved, got %d", len(result.ApprovedToolApprovals))
	}
	if len(result.DeniedToolApprovals) != 1 {
		t.Fatalf("expected 1 denied, got %d", len(result.DeniedToolApprovals))
	}
	denial := result.DeniedToolApprovals[0]
	if denial.ApprovalResponse.Reason != "test-reason" {
		t.Errorf("expected reason 'test-reason', got %q", denial.ApprovalResponse.Reason)
	}
	if denial.ToolCall.ToolCallID != "call-1" {
		t.Errorf("expected call-1, got %q", denial.ToolCall.ToolCallID)
	}
}

func TestCollectToolApprovals_DeniedWithToolResult(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: false, Reason: "test-reason"},
				{Type: "tool-result", ToolCallID: "call-1", ToolName: "tool1", Output: map[string]interface{}{"type": "execution-denied", "reason": "test-reason"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ApprovedToolApprovals) != 0 {
		t.Errorf("expected 0 approved, got %d", len(result.ApprovedToolApprovals))
	}
	if len(result.DeniedToolApprovals) != 0 {
		t.Errorf("expected 0 denied (already has result), got %d", len(result.DeniedToolApprovals))
	}
}

func TestCollectToolApprovals_UnknownApprovalID(t *testing.T) {
	_, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-1"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "unknown-approval-id", Approved: true},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown approval ID")
	}
	if _, ok := err.(*InvalidToolApprovalError); !ok {
		t.Errorf("expected InvalidToolApprovalError, got %T: %v", err, err)
	}
}

func TestCollectToolApprovals_MissingToolCall(t *testing.T) {
	_, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-that-does-not-exist"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: true},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing tool call")
	}
	if _, ok := err.(*ToolCallNotFoundForApprovalError); !ok {
		t.Errorf("expected ToolCallNotFoundForApprovalError, got %T: %v", err, err)
	}
}

func TestCollectToolApprovals_Complex_MultipleApprovalsAndDenials(t *testing.T) {
	result, err := CollectToolApprovals([]ModelMessage{
		{
			Role: "assistant",
			Content: []ModelMessageContent{
				{Type: "tool-call", ToolCallID: "call-approval-1", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-1"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-1", ToolCallID: "call-approval-1"},
				{Type: "tool-call", ToolCallID: "call-approval-2", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-2"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-2", ToolCallID: "call-approval-2"},
				{Type: "tool-call", ToolCallID: "call-approval-3", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-3"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-3", ToolCallID: "call-approval-3"},
				{Type: "tool-call", ToolCallID: "call-approval-4", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-4"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-4", ToolCallID: "call-approval-4"},
				{Type: "tool-call", ToolCallID: "call-approval-5", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-5"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-5", ToolCallID: "call-approval-5"},
				{Type: "tool-call", ToolCallID: "call-approval-6", ToolName: "tool1", Input: map[string]interface{}{"value": "test-input-6"}},
				{Type: "tool-approval-request", ApprovalID: "approval-id-6", ToolCallID: "call-approval-6"},
			},
		},
		{
			Role: "tool",
			Content: []ModelMessageContent{
				{Type: "tool-approval-response", ApprovalID: "approval-id-1", Approved: true},
				{Type: "tool-approval-response", ApprovalID: "approval-id-2", Approved: true},
				{Type: "tool-approval-response", ApprovalID: "approval-id-3", Approved: false, Reason: "test-reason"},
				{Type: "tool-approval-response", ApprovalID: "approval-id-4", Approved: false},
				{Type: "tool-approval-response", ApprovalID: "approval-id-5", Approved: true},
				{Type: "tool-result", ToolCallID: "call-approval-5", ToolName: "tool1", Output: map[string]interface{}{"type": "text", "value": "test-output-5"}},
				{Type: "tool-approval-response", ApprovalID: "approval-id-6", Approved: false},
				{Type: "tool-result", ToolCallID: "call-approval-6", ToolName: "tool1", Output: map[string]interface{}{"type": "execution-denied"}},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 approvals (1 and 2 — 5 has a result already)
	if len(result.ApprovedToolApprovals) != 2 {
		t.Errorf("expected 2 approved, got %d", len(result.ApprovedToolApprovals))
	}
	// 2 denials (3 and 4 — 6 has a result already)
	if len(result.DeniedToolApprovals) != 2 {
		t.Errorf("expected 2 denied, got %d", len(result.DeniedToolApprovals))
	}

	if len(result.ApprovedToolApprovals) >= 2 {
		if result.ApprovedToolApprovals[0].ToolCall.ToolCallID != "call-approval-1" {
			t.Errorf("expected first approved to be call-approval-1, got %q", result.ApprovedToolApprovals[0].ToolCall.ToolCallID)
		}
		if result.ApprovedToolApprovals[1].ToolCall.ToolCallID != "call-approval-2" {
			t.Errorf("expected second approved to be call-approval-2, got %q", result.ApprovedToolApprovals[1].ToolCall.ToolCallID)
		}
	}

	if len(result.DeniedToolApprovals) >= 2 {
		if result.DeniedToolApprovals[0].ApprovalResponse.Reason != "test-reason" {
			t.Errorf("expected first denied reason 'test-reason', got %q", result.DeniedToolApprovals[0].ApprovalResponse.Reason)
		}
		if result.DeniedToolApprovals[1].ToolCall.ToolCallID != "call-approval-4" {
			t.Errorf("expected second denied to be call-approval-4, got %q", result.DeniedToolApprovals[1].ToolCall.ToolCallID)
		}
	}
}
