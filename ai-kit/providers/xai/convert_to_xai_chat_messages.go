// Ported from: packages/xai/src/convert-to-xai-chat-messages.ts
package xai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// convertToXaiChatMessagesResult is the result of converting prompts to xAI format.
type convertToXaiChatMessagesResult struct {
	Messages []interface{}
	Warnings []shared.Warning
}

// convertToXaiChatMessages converts a standard prompt to xAI chat message format.
func convertToXaiChatMessages(prompt languagemodel.Prompt) convertToXaiChatMessagesResult {
	var messages []interface{}
	var warnings []shared.Warning

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, map[string]interface{}{
				"role":    "system",
				"content": m.Content,
			})

		case languagemodel.UserMessage:
			if len(m.Content) == 1 {
				if textPart, ok := m.Content[0].(languagemodel.TextPart); ok {
					messages = append(messages, map[string]interface{}{
						"role":    "user",
						"content": textPart.Text,
					})
					continue
				}
			}

			var contentParts []interface{}
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					contentParts = append(contentParts, map[string]interface{}{
						"type": "text",
						"text": p.Text,
					})
				case languagemodel.FilePart:
					if strings.HasPrefix(p.MediaType, "image/") {
						mediaType := p.MediaType
						if mediaType == "image/*" {
							mediaType = "image/jpeg"
						}

						var imageURL string
						switch d := p.Data.(type) {
						case languagemodel.DataContentString:
							// Could be a URL or base64 string
							if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
								imageURL = d.Value
							} else {
								imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, providerutils.ConvertToBase64String(d.Value))
							}
						case languagemodel.DataContentBytes:
							imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, providerutils.ConvertToBase64Bytes(d.Data))
						}

						contentParts = append(contentParts, map[string]interface{}{
							"type": "image_url",
							"image_url": map[string]interface{}{
								"url": imageURL,
							},
						})
					} else {
						panic(errors.NewUnsupportedFunctionalityError(
							fmt.Sprintf("file part media type %s", p.MediaType), "",
						))
					}
				}
			}
			messages = append(messages, map[string]interface{}{
				"role":    "user",
				"content": contentParts,
			})

		case languagemodel.AssistantMessage:
			text := ""
			var toolCalls []interface{}

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ToolCallPart:
					inputJSON, _ := json.Marshal(p.Input)
					toolCalls = append(toolCalls, map[string]interface{}{
						"id":   p.ToolCallID,
						"type": "function",
						"function": map[string]interface{}{
							"name":      p.ToolName,
							"arguments": string(inputJSON),
						},
					})
				}
			}

			msg := map[string]interface{}{
				"role":    "assistant",
				"content": text,
			}
			if len(toolCalls) > 0 {
				msg["tool_calls"] = toolCalls
			}
			messages = append(messages, msg)

		case languagemodel.ToolMessage:
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					// skip approval responses
					continue
				case languagemodel.ToolResultPart:
					var contentValue string
					switch o := p.Output.(type) {
					case languagemodel.ToolResultOutputText:
						contentValue = o.Value
					case languagemodel.ToolResultOutputErrorText:
						contentValue = o.Value
					case languagemodel.ToolResultOutputExecutionDenied:
						if o.Reason != nil {
							contentValue = *o.Reason
						} else {
							contentValue = "Tool execution denied."
						}
					case languagemodel.ToolResultOutputContent:
						var parts []string
						for _, item := range o.Value {
							if t, ok := item.(languagemodel.ToolResultContentText); ok {
								parts = append(parts, t.Text)
							}
						}
						contentValue = strings.Join(parts, "")
					case languagemodel.ToolResultOutputJSON:
						jsonBytes, _ := json.Marshal(o.Value)
						contentValue = string(jsonBytes)
					case languagemodel.ToolResultOutputErrorJSON:
						jsonBytes, _ := json.Marshal(o.Value)
						contentValue = string(jsonBytes)
					}

					messages = append(messages, map[string]interface{}{
						"role":         "tool",
						"tool_call_id": p.ToolCallID,
						"content":      contentValue,
					})
				default:
					_ = p
				}
			}

		default:
			panic(fmt.Sprintf("Unsupported role: %T", msg))
		}
	}

	return convertToXaiChatMessagesResult{
		Messages: messages,
		Warnings: warnings,
	}
}
