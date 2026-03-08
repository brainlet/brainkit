// Ported from: packages/core/src/agent/message-list/utils/provider-compat.ts
package utils

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// ToolResultWithInput is a tool result part with an input field (Anthropic requirement).
type ToolResultWithInput struct {
	Type       string         `json:"type"`
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Output     any            `json:"output,omitempty"`
	Input      map[string]any `json:"input"`
}

// EnsureGeminiCompatibleMessages ensures message array is compatible with Gemini API requirements.
// Gemini API requires:
// 1. The first non-system message must be from the user role
// 2. Cannot have only system messages - at least one user/assistant is required
func EnsureGeminiCompatibleMessages(messages []map[string]any) []map[string]any {
	result := make([]map[string]any, len(messages))
	copy(result, messages)

	// Ensure first non-system message is user
	firstNonSystemIndex := -1
	for i, m := range result {
		role, _ := m["role"].(string)
		if role != "system" {
			firstNonSystemIndex = i
			break
		}
	}

	if firstNonSystemIndex == -1 {
		// Only system messages or empty — pass through unchanged.
		return result
	}

	role, _ := result[firstNonSystemIndex]["role"].(string)
	if role == "assistant" {
		// First non-system is assistant, insert user message before it
		userMsg := map[string]any{
			"role":    "user",
			"content": ".",
		}
		newResult := make([]map[string]any, 0, len(result)+1)
		newResult = append(newResult, result[:firstNonSystemIndex]...)
		newResult = append(newResult, userMsg)
		newResult = append(newResult, result[firstNonSystemIndex:]...)
		return newResult
	}

	return result
}

// EnsureGeminiCompatibleDBMessages ensures MastraDBMessage array is compatible with Gemini API requirements.
func EnsureGeminiCompatibleDBMessages(messages []*state.MastraDBMessage) []*state.MastraDBMessage {
	result := make([]*state.MastraDBMessage, len(messages))
	copy(result, messages)

	firstNonSystemIndex := -1
	for i, m := range result {
		if m.Role != "system" {
			firstNonSystemIndex = i
			break
		}
	}

	if firstNonSystemIndex == -1 {
		return result
	}

	if result[firstNonSystemIndex].Role == "assistant" {
		userMsg := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				Role: "user",
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "text", Text: "."},
				},
			},
		}
		newResult := make([]*state.MastraDBMessage, 0, len(result)+1)
		newResult = append(newResult, result[:firstNonSystemIndex]...)
		newResult = append(newResult, userMsg)
		newResult = append(newResult, result[firstNonSystemIndex:]...)
		return newResult
	}

	return result
}

// EnsureAnthropicCompatibleMessages ensures model messages are compatible with Anthropic API requirements.
// Anthropic API requires tool-result parts to include an 'input' field.
func EnsureAnthropicCompatibleMessages(messages []map[string]any, dbMessages []*state.MastraDBMessage) []map[string]any {
	result := make([]map[string]any, len(messages))
	for i, msg := range messages {
		result[i] = enrichToolResultsWithInput(msg, dbMessages)
	}
	return result
}

// enrichToolResultsWithInput enriches a single message's tool-result parts with input field.
func enrichToolResultsWithInput(message map[string]any, dbMessages []*state.MastraDBMessage) map[string]any {
	role, _ := message["role"].(string)
	if role != "tool" {
		return message
	}

	contentArr, ok := message["content"].([]map[string]any)
	if !ok {
		return message
	}

	newContent := make([]map[string]any, len(contentArr))
	for i, part := range contentArr {
		partType, _ := part["type"].(string)
		if partType == "tool-result" {
			toolCallID, _ := part["toolCallId"].(string)
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			newPart["input"] = FindToolCallArgs(dbMessages, toolCallID)
			newContent[i] = newPart
		} else {
			newContent[i] = part
		}
	}

	result := make(map[string]any)
	for k, v := range message {
		result[k] = v
	}
	result["content"] = newContent
	return result
}

// HasOpenAIReasoningItemId checks if a message part has OpenAI reasoning itemId.
func HasOpenAIReasoningItemId(part map[string]any) bool {
	if part == nil {
		return false
	}
	pmRaw, ok := part["providerMetadata"]
	if !ok {
		return false
	}
	pm, ok := pmRaw.(map[string]any)
	if !ok {
		return false
	}
	openaiRaw, ok := pm["openai"]
	if !ok {
		return false
	}
	openai, ok := openaiRaw.(map[string]any)
	if !ok {
		return false
	}
	_, ok = openai["itemId"].(string)
	return ok
}

// GetOpenAIReasoningItemId extracts the OpenAI itemId from a message part if present.
func GetOpenAIReasoningItemId(part map[string]any) string {
	if !HasOpenAIReasoningItemId(part) {
		return ""
	}
	pm := part["providerMetadata"].(map[string]any)
	openai := pm["openai"].(map[string]any)
	return openai["itemId"].(string)
}

// FindToolCallArgs finds the tool call args for a given toolCallId by searching through messages.
func FindToolCallArgs(messages []*state.MastraDBMessage, toolCallID string) map[string]any {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil || msg.Role != "assistant" {
			continue
		}

		for _, part := range msg.Content.Parts {
			if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
				part.ToolInvocation.ToolCallID == toolCallID {
				if part.ToolInvocation.Args != nil {
					return part.ToolInvocation.Args
				}
				return map[string]any{}
			}
		}

		for _, ti := range msg.Content.ToolInvocations {
			if ti.ToolCallID == toolCallID {
				if ti.Args != nil {
					return ti.Args
				}
				return map[string]any{}
			}
		}
	}

	return map[string]any{}
}
