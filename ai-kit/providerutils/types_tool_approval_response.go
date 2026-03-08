// Ported from: packages/provider-utils/src/types/tool-approval-response.ts
package providerutils

// ToolApprovalResponse represents a tool approval response prompt part.
type ToolApprovalResponse struct {
	Type             string `json:"type"` // "tool-approval-response"
	ApprovalID       string `json:"approvalId"`
	Approved         bool   `json:"approved"`
	Reason           string `json:"reason,omitempty"`
	ProviderExecuted bool   `json:"providerExecuted,omitempty"`
}
