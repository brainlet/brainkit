package harness

import (
	"encoding/json"
	"fmt"
)

// RespondToToolApproval responds to a tool approval request.
// Uses direct bridge eval to avoid nested async issues during agent stream.
func (h *Harness) RespondToToolApproval(decision ToolApprovalDecision) error {
	b, _ := json.Marshal(map[string]string{"decision": string(decision)})
	code := fmt.Sprintf(`__brainkit_harness.respondToToolApproval(JSON.parse(%s))`, quoteJSString(string(b)))
	_, err := h.rt.EvalBridgeDirect("harness-respond-approval.js", code)
	if err != nil {
		return fmt.Errorf("respondToToolApproval: %w", err)
	}
	return nil
}

// SetPermissionForCategory sets the default policy for a tool category.
func (h *Harness) SetPermissionForCategory(category, policy string) error {
	b, _ := json.Marshal(map[string]string{"category": category, "policy": policy})
	return h.callJSVoid("setPermissionForCategory", string(b))
}

// SetPermissionForTool overrides the category policy for a specific tool.
func (h *Harness) SetPermissionForTool(toolName, policy string) error {
	b, _ := json.Marshal(map[string]string{"toolName": toolName, "policy": policy})
	return h.callJSVoid("setPermissionForTool", string(b))
}

// GetPermissionRules returns the current permission rules.
func (h *Harness) GetPermissionRules() PermissionRules {
	r, _ := h.callJSSimple("getPermissionRules")
	var rules PermissionRules
	json.Unmarshal([]byte(r), &rules)
	return rules
}

// GrantSessionCategory auto-approves all tools in a category for this session.
func (h *Harness) GrantSessionCategory(category string) error {
	b, _ := json.Marshal(map[string]string{"category": category})
	return h.callJSVoid("grantSessionCategory", string(b))
}

// GrantSessionTool auto-approves a specific tool for this session.
func (h *Harness) GrantSessionTool(toolName string) error {
	b, _ := json.Marshal(map[string]string{"toolName": toolName})
	return h.callJSVoid("grantSessionTool", string(b))
}

// RespondToQuestion answers an ask_user tool invocation.
// Uses direct bridge eval (not EvalTS) to avoid nested async wrapper issues
// when called while SendMessage is awaiting the agent stream.
func (h *Harness) RespondToQuestion(questionID, answer string) error {
	b, _ := json.Marshal(map[string]string{"questionId": questionID, "answer": answer})
	code := fmt.Sprintf(`__brainkit_harness.respondToQuestion(JSON.parse(%s))`, quoteJSString(string(b)))
	_, err := h.rt.EvalBridgeDirect("harness-respond-question.js", code)
	if err != nil {
		return fmt.Errorf("respondToQuestion: %w", err)
	}
	return nil
}

// RespondToPlanApproval responds to a plan approval request.
// Uses direct bridge eval to avoid nested async issues during agent stream.
func (h *Harness) RespondToPlanApproval(planID string, resp PlanResponse) error {
	b, _ := json.Marshal(map[string]any{"planId": planID, "response": resp})
	code := fmt.Sprintf(`__brainkit_harness.respondToPlanApproval(JSON.parse(%s))`, quoteJSString(string(b)))
	_, err := h.rt.EvalBridgeDirect("harness-respond-plan.js", code)
	if err != nil {
		return fmt.Errorf("respondToPlanApproval: %w", err)
	}
	return nil
}
