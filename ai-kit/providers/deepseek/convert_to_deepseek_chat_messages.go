// Ported from: packages/deepseek/src/chat/convert-to-deepseek-chat-messages.ts
package deepseek

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// ConvertToDeepSeekChatMessagesResult holds the converted messages and any warnings.
type ConvertToDeepSeekChatMessagesResult struct {
	Messages DeepSeekChatPrompt
	Warnings []shared.Warning
}

// ConvertToDeepSeekChatMessages converts a language model prompt to DeepSeek chat messages.
func ConvertToDeepSeekChatMessages(
	prompt languagemodel.Prompt,
	responseFormat languagemodel.ResponseFormat,
) ConvertToDeepSeekChatMessagesResult {
	messages := DeepSeekChatPrompt{}
	warnings := []shared.Warning{}

	// Inject system message if response format is JSON
	if jsonFormat, ok := responseFormat.(languagemodel.ResponseFormatJSON); ok {
		if jsonFormat.Schema == nil {
			messages = append(messages, DeepSeekSystemMessage{
				Role:    "system",
				Content: "Return JSON.",
			})
		} else {
			schemaJSON, _ := json.Marshal(jsonFormat.Schema)
			messages = append(messages, DeepSeekSystemMessage{
				Role:    "system",
				Content: "Return JSON that conforms to the following schema: " + string(schemaJSON),
			})
			detail := "JSON response schema is injected into the system message."
			warnings = append(warnings, shared.CompatibilityWarning{
				Feature: "responseFormat JSON schema",
				Details: &detail,
			})
		}
	}

	// Find last user message index
	lastUserMessageIndex := -1
	for i := len(prompt) - 1; i >= 0; i-- {
		if _, ok := prompt[i].(languagemodel.UserMessage); ok {
			lastUserMessageIndex = i
			break
		}
	}

	for index, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, DeepSeekSystemMessage{
				Role:    "system",
				Content: m.Content,
			})

		case languagemodel.UserMessage:
			userContent := ""
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					userContent += p.Text
				default:
					warnings = append(warnings, shared.UnsupportedWarning{
						Feature: fmt.Sprintf("user message part type: %T", p),
					})
				}
			}
			messages = append(messages, DeepSeekUserMessage{
				Role:    "user",
				Content: userContent,
			})

		case languagemodel.AssistantMessage:
			text := ""
			var reasoning *string

			var toolCalls []DeepSeekMessageToolCall

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ReasoningPart:
					// Only include reasoning from messages after the last user message
					if index <= lastUserMessageIndex {
						break
					}
					if reasoning == nil {
						s := p.Text
						reasoning = &s
					} else {
						*reasoning += p.Text
					}
				case languagemodel.ToolCallPart:
					inputJSON, _ := json.Marshal(p.Input)
					toolCalls = append(toolCalls, DeepSeekMessageToolCall{
						ID:   p.ToolCallID,
						Type: "function",
						Function: DeepSeekMessageToolCallFunction{
							Name:      p.ToolName,
							Arguments: string(inputJSON),
						},
					})
				}
			}

			assistantMsg := DeepSeekAssistantMessage{
				Role:             "assistant",
				ReasoningContent: reasoning,
			}
			if text != "" {
				assistantMsg.Content = &text
			}
			if len(toolCalls) > 0 {
				assistantMsg.ToolCalls = toolCalls
			}
			messages = append(messages, assistantMsg)

		case languagemodel.ToolMessage:
			for _, toolResponse := range m.Content {
				// Skip tool approval responses
				if _, isApproval := toolResponse.(languagemodel.ToolApprovalResponsePart); isApproval {
					continue
				}

				tr, ok := toolResponse.(languagemodel.ToolResultPart)
				if !ok {
					continue
				}

				var contentValue string
				switch output := tr.Output.(type) {
				case languagemodel.ToolResultOutputText:
					contentValue = output.Value
				case languagemodel.ToolResultOutputErrorText:
					contentValue = output.Value
				case languagemodel.ToolResultOutputExecutionDenied:
					if output.Reason != nil {
						contentValue = *output.Reason
					} else {
						contentValue = "Tool execution denied."
					}
				case languagemodel.ToolResultOutputContent:
					jsonBytes, _ := json.Marshal(output.Value)
					contentValue = string(jsonBytes)
				case languagemodel.ToolResultOutputJSON:
					jsonBytes, _ := json.Marshal(output.Value)
					contentValue = string(jsonBytes)
				case languagemodel.ToolResultOutputErrorJSON:
					jsonBytes, _ := json.Marshal(output.Value)
					contentValue = string(jsonBytes)
				}

				messages = append(messages, DeepSeekToolMessage{
					Role:       "tool",
					ToolCallID: tr.ToolCallID,
					Content:    contentValue,
				})
			}

		default:
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: fmt.Sprintf("message role: %T", msg),
			})
		}
	}

	return ConvertToDeepSeekChatMessagesResult{
		Messages: messages,
		Warnings: warnings,
	}
}
