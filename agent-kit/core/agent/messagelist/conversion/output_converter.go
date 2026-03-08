// Ported from: packages/core/src/agent/message-list/conversion/output-converter.ts
package conversion

import (
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/detection"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// SanitizeAIV4UIMessages sanitizes AIV4 UI messages by filtering out incomplete tool calls.
// Removes messages with empty parts arrays after sanitization.
func SanitizeAIV4UIMessages(messages []*state.UIMessageWithMetadata) []*state.UIMessageWithMetadata {
	var result []*state.UIMessageWithMetadata
	for _, m := range messages {
		if len(m.Parts) == 0 {
			continue
		}
		var safeParts []state.MastraMessagePart
		for _, p := range m.Parts {
			if p.Type != "tool-invocation" ||
				(p.ToolInvocation != nil && p.ToolInvocation.State != "call" && p.ToolInvocation.State != "partial-call") {
				safeParts = append(safeParts, p)
			}
		}
		if len(safeParts) == 0 {
			continue
		}

		sanitized := *m
		sanitized.Parts = safeParts

		// Ensure toolInvocations are also updated to only show results
		if len(m.ToolInvocations) > 0 {
			var resultInvocations []state.ToolInvocation
			for _, t := range m.ToolInvocations {
				if t.State == "result" {
					resultInvocations = append(resultInvocations, t)
				}
			}
			sanitized.ToolInvocations = resultInvocations
		}

		result = append(result, &sanitized)
	}
	return result
}

// SanitizeV5UIMessages sanitizes AIV5 UI messages by filtering out streaming states, data-* parts,
// empty text parts, and optionally incomplete tool calls.
func SanitizeV5UIMessages(messages []*adapters.AIV5UIMessage, filterIncompleteToolCalls bool) []*adapters.AIV5UIMessage {
	var result []*adapters.AIV5UIMessage
	for _, m := range messages {
		if len(m.Parts) == 0 {
			continue
		}
		var safeParts []adapters.AIV5UIPart
		for _, p := range m.Parts {
			// Filter out data-* parts
			if strings.HasPrefix(p.Type, "data-") {
				continue
			}

			// Filter out empty text parts (but preserve them if they are the only parts)
			if p.Type == "text" && (p.Text == "" || strings.TrimSpace(p.Text) == "") {
				hasNonEmpty := false
				for _, other := range m.Parts {
					if !(other.Type == "text" && (other.Text == "" || strings.TrimSpace(other.Text) == "")) {
						hasNonEmpty = true
						break
					}
				}
				if hasNonEmpty {
					continue
				}
			}

			if !adapters.IsToolUIPart(p) {
				safeParts = append(safeParts, p)
				continue
			}

			// Handle tool parts
			if filterIncompleteToolCalls {
				if p.State == "output-available" || p.State == "output-error" {
					// Strip completed provider-executed tools
					if p.ProviderExecuted != nil && *p.ProviderExecuted {
						continue
					}
					safeParts = append(safeParts, p)
				} else if p.State == "input-available" && p.ProviderExecuted != nil && *p.ProviderExecuted {
					safeParts = append(safeParts, p)
				}
				// else: skip incomplete tool calls
			} else {
				// When processing response messages FROM the LLM: keep input-available states
				// but filter out input-streaming
				if p.State != "input-streaming" {
					safeParts = append(safeParts, p)
				}
			}
		}

		if len(safeParts) == 0 {
			continue
		}

		// Unwrap output value wrappers for output-available tool parts
		processedParts := make([]adapters.AIV5UIPart, len(safeParts))
		for i, part := range safeParts {
			if adapters.IsToolUIPart(part) && part.State == "output-available" {
				output := part.Output
				if m, ok := output.(map[string]any); ok {
					if v, exists := m["value"]; exists {
						part.Output = v
					}
				}
			}
			processedParts[i] = part
		}

		sanitized := &adapters.AIV5UIMessage{
			ID:       m.ID,
			Role:     m.Role,
			Parts:    processedParts,
			Metadata: m.Metadata,
		}

		result = append(result, sanitized)
	}
	return result
}

// AddStartStepPartsForAIV5 adds step-start parts between tool parts and non-tool parts
// for proper AIV5 message conversion.
func AddStartStepPartsForAIV5(messages []*adapters.AIV5UIMessage) []*adapters.AIV5UIMessage {
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		// Iterate by index since we may be inserting elements
		for i := 0; i < len(message.Parts); i++ {
			part := message.Parts[i]
			if !adapters.IsToolUIPart(part) {
				continue
			}
			if i+1 >= len(message.Parts) {
				break
			}
			nextPart := message.Parts[i+1]
			// Don't add step-start between consecutive tool parts (parallel tool calls)
			if nextPart.Type != "step-start" && !adapters.IsToolUIPart(nextPart) {
				// Insert step-start
				newParts := make([]adapters.AIV5UIPart, len(message.Parts)+1)
				copy(newParts, message.Parts[:i+1])
				newParts[i+1] = adapters.AIV5UIPart{Type: "step-start"}
				copy(newParts[i+2:], message.Parts[i+1:])
				message.Parts = newParts
				i++ // skip the inserted step-start
			}
		}
	}
	return messages
}

// AIV4UIMessagesToAIV4CoreMessages converts AIV4 UI messages to AIV4 Core messages.
// TODO: In TS this calls convertToCoreMessages from @internal/ai-sdk-v4.
// For now, this is a stub that converts through the adapter layer.
func AIV4UIMessagesToAIV4CoreMessages(messages []*state.UIMessageWithMetadata) []map[string]any {
	sanitized := SanitizeAIV4UIMessages(messages)
	// TODO: Call actual AI SDK V4 convertToCoreMessages equivalent.
	// For now, convert UIMessages to CoreMessage-like maps.
	var result []map[string]any
	for _, msg := range sanitized {
		coreMsg := map[string]any{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.ID != "" {
			coreMsg["id"] = msg.ID
		}
		if len(msg.Parts) > 0 {
			// Build content array from parts
			var contentParts []map[string]any
			for _, part := range msg.Parts {
				switch part.Type {
				case "text":
					contentParts = append(contentParts, map[string]any{
						"type": "text",
						"text": part.Text,
					})
				case "tool-invocation":
					if part.ToolInvocation != nil && part.ToolInvocation.State == "result" {
						contentParts = append(contentParts, map[string]any{
							"type":       "tool-call",
							"toolCallId": part.ToolInvocation.ToolCallID,
							"toolName":   part.ToolInvocation.ToolName,
							"args":       part.ToolInvocation.Args,
						})
					}
				}
			}
			if len(contentParts) > 0 {
				coreMsg["content"] = contentParts
			}
		}
		result = append(result, coreMsg)
	}
	return result
}

// AIV5UIMessagesToAIV5ModelMessages converts AIV5 UI messages to AIV5 Model messages.
// TODO: In TS this calls AIV5.convertToModelMessages.
// For now returns a stub conversion through the DB layer.
func AIV5UIMessagesToAIV5ModelMessages(
	messages []*adapters.AIV5UIMessage,
	dbMessages []*state.MastraDBMessage,
	filterIncompleteToolCalls ...bool,
) []map[string]any {
	filter := false
	if len(filterIncompleteToolCalls) > 0 {
		filter = filterIncompleteToolCalls[0]
	}

	sanitized := SanitizeV5UIMessages(messages, filter)
	preprocessed := AddStartStepPartsForAIV5(sanitized)

	// TODO: Call actual AIV5.convertToModelMessages equivalent.
	// For now, convert through a basic transformation.
	var result []map[string]any
	for _, msg := range preprocessed {
		modelMsg := map[string]any{
			"role": msg.Role,
		}

		var contentParts []map[string]any
		for _, part := range msg.Parts {
			switch {
			case part.Type == "text":
				contentParts = append(contentParts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			case adapters.IsToolUIPart(part):
				toolName := part.Type
				if strings.HasPrefix(toolName, "tool-") {
					toolName = toolName[5:]
				}
				if part.State == "output-available" || part.State == "output-error" {
					contentParts = append(contentParts, map[string]any{
						"type":       "tool-result",
						"toolCallId": part.ToolCallID,
						"toolName":   toolName,
						"output":     part.Output,
					})
				} else {
					contentParts = append(contentParts, map[string]any{
						"type":       "tool-call",
						"toolCallId": part.ToolCallID,
						"toolName":   toolName,
						"input":      part.Input,
					})
				}
			case part.Type == "reasoning":
				contentParts = append(contentParts, map[string]any{
					"type": "reasoning",
					"text": part.Text,
				})
			case part.Type == "file":
				contentParts = append(contentParts, map[string]any{
					"type":      "file",
					"url":       part.URL,
					"mediaType": part.MediaType,
				})
			}
		}

		if len(contentParts) > 0 {
			modelMsg["content"] = contentParts
		}
		if msg.Metadata != nil {
			modelMsg["metadata"] = msg.Metadata
			// Restore providerOptions from metadata.providerMetadata
			if pm, ok := msg.Metadata["providerMetadata"]; ok {
				modelMsg["providerOptions"] = pm
			}
		}

		result = append(result, modelMsg)
	}

	// Apply stored modelOutput from dbMessages
	storedModelOutputs := make(map[string]any)
	for _, dbMsg := range dbMessages {
		if dbMsg.Content.Format == 2 {
			for _, part := range dbMsg.Content.Parts {
				if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
					part.ToolInvocation.State == "result" && part.ProviderMetadata != nil {
					if mastra, ok := part.ProviderMetadata["mastra"]; ok {
						if mo, ok := mastra["modelOutput"]; ok {
							storedModelOutputs[part.ToolInvocation.ToolCallID] = mo
						}
					}
				}
			}
		}
	}

	if len(storedModelOutputs) > 0 {
		for _, modelMsg := range result {
			role, _ := modelMsg["role"].(string)
			if role == "tool" {
				if contentArr, ok := modelMsg["content"].([]map[string]any); ok {
					for i, part := range contentArr {
						partType, _ := part["type"].(string)
						if partType == "tool-result" {
							if toolCallID, ok := part["toolCallId"].(string); ok {
								if mo, ok := storedModelOutputs[toolCallID]; ok {
									contentArr[i]["output"] = mo
								}
							}
						}
					}
				}
			}
		}
	}

	// Add input field to tool-result parts for Anthropic API compatibility
	return EnsureAnthropicCompatibleMessages(result, dbMessages)
}

// AIV4CoreMessagesToAIV5ModelMessages converts AIV4 Core messages to AIV5 Model messages.
func AIV4CoreMessagesToAIV5ModelMessages(
	messages []map[string]any,
	source state.MessageSource,
	adapterContext *adapters.AdapterContext,
	dbMessages []*state.MastraDBMessage,
) []map[string]any {
	// Convert core messages to DB messages then to V5 UI messages then to model messages
	var v5UIMessages []*adapters.AIV5UIMessage
	for _, m := range messages {
		dbMsg := adapters.AIV4FromCoreMessage(m, adapterContext, source)
		v5UI := adapters.AIV5ToUIMessage(dbMsg)
		v5UIMessages = append(v5UIMessages, v5UI)
	}
	return AIV5UIMessagesToAIV5ModelMessages(v5UIMessages, dbMessages)
}

// SystemMessageToAIV4Core converts various message formats to AIV4 CoreMessage format for system messages.
func SystemMessageToAIV4Core(message map[string]any) map[string]any {
	if detection.IsAIV5CoreMessage(message) {
		dbMsg := adapters.AIV5FromModelMessage(message, "system")
		result, _ := adapters.AIV4SystemToV4Core(dbMsg)
		return result
	}
	if detection.IsMastraDBMessage(message) {
		dbMsg := mapToMastraDBMessage(message)
		result, _ := adapters.AIV4SystemToV4Core(dbMsg)
		return result
	}
	return message
}

// SystemMessageToAIV4CoreString converts a string to an AIV4 system CoreMessage.
func SystemMessageToAIV4CoreString(content string) map[string]any {
	return map[string]any{
		"role":    "system",
		"content": content,
	}
}

// EnsureAnthropicCompatibleMessages adds input field to tool-result parts for Anthropic API compatibility.
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
