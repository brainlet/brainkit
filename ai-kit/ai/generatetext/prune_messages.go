// Ported from: packages/ai/src/generate-text/prune-messages.ts
package generatetext

import (
	"strconv"
	"strings"
)

// PruneMessagesOptions contains options for pruning model messages.
type PruneMessagesOptions struct {
	// Messages is the list of model messages to prune.
	Messages []ModelMessage

	// Reasoning controls how to remove reasoning content from assistant messages.
	// Valid values: "all", "before-last-message", "none" (default: "none").
	Reasoning string

	// ToolCalls controls how to prune tool call/results/approval content.
	// Can be: "all", "before-last-message", "before-last-N-messages", "none",
	// or a slice of ToolCallPruneRule.
	// Default: nil (no pruning).
	ToolCalls interface{} // string | []ToolCallPruneRule

	// EmptyMessages controls whether to keep or remove messages whose content is empty after pruning.
	// Valid values: "keep", "remove" (default: "remove").
	EmptyMessages string
}

// ToolCallPruneRule specifies a rule for pruning tool calls.
type ToolCallPruneRule struct {
	// Type is "all", "before-last-message", or "before-last-N-messages".
	Type string
	// Tools optionally restricts pruning to specific tool names.
	Tools []string
}

// PruneMessages prunes model messages according to the given options.
func PruneMessages(opts PruneMessagesOptions) []ModelMessage {
	messages := opts.Messages
	reasoning := opts.Reasoning
	if reasoning == "" {
		reasoning = "none"
	}
	emptyMessages := opts.EmptyMessages
	if emptyMessages == "" {
		emptyMessages = "remove"
	}

	// Filter reasoning parts
	if reasoning == "all" || reasoning == "before-last-message" {
		newMessages := make([]ModelMessage, len(messages))
		for i, msg := range messages {
			if msg.Role != "assistant" {
				newMessages[i] = msg
				continue
			}
			parts, ok := msg.Content.([]ModelMessageContent)
			if !ok {
				newMessages[i] = msg
				continue
			}
			if reasoning == "before-last-message" && i == len(messages)-1 {
				newMessages[i] = msg
				continue
			}
			filtered := make([]ModelMessageContent, 0, len(parts))
			for _, part := range parts {
				if part.Type != "reasoning" {
					filtered = append(filtered, part)
				}
			}
			newMessages[i] = ModelMessage{
				Role:    msg.Role,
				Content: filtered,
			}
		}
		messages = newMessages
	}

	// Convert tool call rules
	var toolCallRules []ToolCallPruneRule
	switch tc := opts.ToolCalls.(type) {
	case nil:
		// no pruning
	case string:
		switch tc {
		case "none", "":
			// no pruning
		case "all":
			toolCallRules = []ToolCallPruneRule{{Type: "all"}}
		case "before-last-message":
			toolCallRules = []ToolCallPruneRule{{Type: "before-last-message"}}
		default:
			toolCallRules = []ToolCallPruneRule{{Type: tc}}
		}
	case []ToolCallPruneRule:
		toolCallRules = tc
	}

	// Apply tool call pruning rules
	for _, rule := range toolCallRules {
		var keepLastMessagesCount *int
		switch {
		case rule.Type == "all":
			// keepLastMessagesCount stays nil (prune all)
		case rule.Type == "before-last-message":
			one := 1
			keepLastMessagesCount = &one
		default:
			// Parse "before-last-N-messages"
			s := rule.Type
			s = strings.TrimPrefix(s, "before-last-")
			s = strings.TrimSuffix(s, "-messages")
			if n, err := strconv.Atoi(s); err == nil {
				keepLastMessagesCount = &n
			}
		}

		// Scan kept messages to identify tool calls and approvals that need to be kept
		keptToolCallIDs := map[string]bool{}
		keptApprovalIDs := map[string]bool{}

		if keepLastMessagesCount != nil {
			start := len(messages) - *keepLastMessagesCount
			if start < 0 {
				start = 0
			}
			for _, msg := range messages[start:] {
				parts, ok := msg.Content.([]ModelMessageContent)
				if !ok {
					continue
				}
				if msg.Role != "assistant" && msg.Role != "tool" {
					continue
				}
				for _, part := range parts {
					switch part.Type {
					case "tool-call", "tool-result":
						keptToolCallIDs[part.ToolCallID] = true
					case "tool-approval-request", "tool-approval-response":
						keptApprovalIDs[part.ApprovalID] = true
					}
				}
			}
		}

		newMessages := make([]ModelMessage, len(messages))
		for i, msg := range messages {
			if (msg.Role != "assistant" && msg.Role != "tool") {
				newMessages[i] = msg
				continue
			}
			parts, ok := msg.Content.([]ModelMessageContent)
			if !ok {
				newMessages[i] = msg
				continue
			}
			if keepLastMessagesCount != nil && i >= len(messages)-*keepLastMessagesCount {
				newMessages[i] = msg
				continue
			}

			toolCallIDToToolName := map[string]string{}
			approvalIDToToolName := map[string]string{}

			filtered := make([]ModelMessageContent, 0, len(parts))
			for _, part := range parts {
				// Keep non-tool parts
				if part.Type != "tool-call" && part.Type != "tool-result" &&
					part.Type != "tool-approval-request" && part.Type != "tool-approval-response" {
					filtered = append(filtered, part)
					continue
				}

				// Track tool calls and approvals
				if part.Type == "tool-call" {
					toolCallIDToToolName[part.ToolCallID] = part.ToolName
				} else if part.Type == "tool-approval-request" {
					approvalIDToToolName[part.ApprovalID] = toolCallIDToToolName[part.ToolCallID]
				}

				// Keep parts associated with kept tool calls or approvals
				if (part.Type == "tool-call" || part.Type == "tool-result") && keptToolCallIDs[part.ToolCallID] {
					filtered = append(filtered, part)
					continue
				}
				if (part.Type == "tool-approval-request" || part.Type == "tool-approval-response") && keptApprovalIDs[part.ApprovalID] {
					filtered = append(filtered, part)
					continue
				}

				// Keep parts not associated with a tool that should be removed
				if rule.Tools != nil {
					toolName := ""
					if part.Type == "tool-call" || part.Type == "tool-result" {
						toolName = part.ToolName
					} else {
						toolName = approvalIDToToolName[part.ApprovalID]
					}
					if !containsString(rule.Tools, toolName) {
						filtered = append(filtered, part)
					}
				}
			}

			newMessages[i] = ModelMessage{
				Role:    msg.Role,
				Content: filtered,
			}
		}
		messages = newMessages
	}

	// Remove empty messages
	if emptyMessages == "remove" {
		var filtered []ModelMessage
		for _, msg := range messages {
			switch content := msg.Content.(type) {
			case []ModelMessageContent:
				if len(content) > 0 {
					filtered = append(filtered, msg)
				}
			case string:
				if len(content) > 0 {
					filtered = append(filtered, msg)
				}
			default:
				filtered = append(filtered, msg)
			}
		}
		messages = filtered
	}

	return messages
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
