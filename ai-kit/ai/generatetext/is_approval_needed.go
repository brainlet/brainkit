// Ported from: packages/ai/src/generate-text/is-approval-needed.ts
package generatetext

// IsApprovalNeeded checks if a tool call requires user approval before execution.
func IsApprovalNeeded(tool Tool, toolCall ToolCall, messages []ModelMessage, experimentalContext interface{}) (bool, error) {
	if tool.NeedsApproval == nil {
		return false, nil
	}

	// Static boolean check
	if b, ok := tool.NeedsApproval.(bool); ok {
		return b, nil
	}

	// Dynamic function check
	if fn, ok := tool.NeedsApproval.(func(input interface{}, options NeedsApprovalOptions) (bool, error)); ok {
		return fn(toolCall.Input, NeedsApprovalOptions{
			ToolCallID:          toolCall.ToolCallID,
			Messages:            messages,
			ExperimentalContext: experimentalContext,
		})
	}

	return false, nil
}
