// Ported from: packages/groq/src/convert-to-groq-chat-messages.ts
package groq

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// convertDataContentToBase64 converts a DataContent value to a base64 string.
func convertDataContentToBase64(data languagemodel.DataContent) string {
	switch d := data.(type) {
	case languagemodel.DataContentBytes:
		return base64.StdEncoding.EncodeToString(d.Data)
	case languagemodel.DataContentString:
		return d.Value
	default:
		return ""
	}
}

// isURLData checks if a DataContent is a URL.
func isURLData(data languagemodel.DataContent) (string, bool) {
	if d, ok := data.(languagemodel.DataContentString); ok {
		if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
			return d.Value, true
		}
	}
	return "", false
}

// ConvertToGroqChatMessages converts a languagemodel.Prompt into a Groq chat prompt.
func ConvertToGroqChatMessages(prompt languagemodel.Prompt) []map[string]any {
	messages := []map[string]any{}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, map[string]any{
				"role":    "system",
				"content": m.Content,
			})

		case languagemodel.UserMessage:
			// Optimization: if single text part, send as string content
			if len(m.Content) == 1 {
				if tp, ok := m.Content[0].(languagemodel.TextPart); ok {
					messages = append(messages, map[string]any{
						"role":    "user",
						"content": tp.Text,
					})
					continue
				}
			}

			// Multi-part content
			contentParts := []any{}
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					contentParts = append(contentParts, map[string]any{
						"type": "text",
						"text": p.Text,
					})

				case languagemodel.FilePart:
					if !strings.HasPrefix(p.MediaType, "image/") {
						panic(errors.NewUnsupportedFunctionalityError("Non-image file content parts", ""))
					}

					mediaType := p.MediaType
					if mediaType == "image/*" {
						mediaType = "image/jpeg"
					}

					var url string
					if u, isURL := isURLData(p.Data); isURL {
						url = u
					} else {
						b64 := convertDataContentToBase64(p.Data)
						url = fmt.Sprintf("data:%s;base64,%s", mediaType, b64)
					}

					contentParts = append(contentParts, map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": url,
						},
					})
				}
			}

			messages = append(messages, map[string]any{
				"role":    "user",
				"content": contentParts,
			})

		case languagemodel.AssistantMessage:
			text := ""
			reasoning := ""
			var toolCalls []map[string]any

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.ReasoningPart:
					// groq supports reasoning for tool-calls in multi-turn conversations
					reasoning += p.Text
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ToolCallPart:
					inputJSON, _ := json.Marshal(p.Input)
					tc := map[string]any{
						"id":   p.ToolCallID,
						"type": "function",
						"function": map[string]any{
							"name":      p.ToolName,
							"arguments": string(inputJSON),
						},
					}
					toolCalls = append(toolCalls, tc)
				}
			}

			assistantMsg := map[string]any{
				"role":    "assistant",
				"content": text,
			}
			if reasoning != "" {
				assistantMsg["reasoning"] = reasoning
			}
			if len(toolCalls) > 0 {
				assistantMsg["tool_calls"] = toolCalls
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

				messages = append(messages, map[string]any{
					"role":         "tool",
					"tool_call_id": tr.ToolCallID,
					"content":      contentValue,
				})
			}

		default:
			panic(fmt.Sprintf("Unsupported role: %T", msg))
		}
	}

	return messages
}
