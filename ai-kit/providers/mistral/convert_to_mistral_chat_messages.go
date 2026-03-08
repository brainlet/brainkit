// Ported from: packages/mistral/src/convert-to-mistral-chat-messages.ts
package mistral

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// formatFileURL formats a DataContent value into a URL string suitable for Mistral.
// For URL data, returns the URL string. For byte data, returns a data URI.
func formatFileURL(data languagemodel.DataContent, mediaType string) string {
	switch d := data.(type) {
	case languagemodel.DataContentString:
		// Check if it's a URL
		if _, err := url.ParseRequestURI(d.Value); err == nil &&
			(strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") || strings.HasPrefix(d.Value, "data:")) {
			return d.Value
		}
		// Treat as base64
		return fmt.Sprintf("data:%s;base64,%s", mediaType, d.Value)
	case languagemodel.DataContentBytes:
		return fmt.Sprintf("data:%s;base64,%s", mediaType, providerutils.ConvertBytesToBase64(d.Data))
	default:
		return ""
	}
}

// ConvertToMistralChatMessages converts a language model prompt to Mistral chat messages.
func ConvertToMistralChatMessages(prompt languagemodel.Prompt) MistralPrompt {
	var messages MistralPrompt

	for i, msg := range prompt {
		isLastMessage := i == len(prompt)-1

		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, MistralSystemMessage{
				Role:    "system",
				Content: m.Content,
			})

		case languagemodel.UserMessage:
			var content []MistralUserMessageContent
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					content = append(content, MistralUserContentText{
						Type: "text",
						Text: p.Text,
					})
				case languagemodel.FilePart:
					if strings.HasPrefix(p.MediaType, "image/") {
						mediaType := p.MediaType
						if mediaType == "image/*" {
							mediaType = "image/jpeg"
						}
						content = append(content, MistralUserContentImageURL{
							Type:     "image_url",
							ImageURL: formatFileURL(p.Data, mediaType),
						})
					} else if p.MediaType == "application/pdf" {
						content = append(content, MistralUserContentDocumentURL{
							Type:        "document_url",
							DocumentURL: formatFileURL(p.Data, "application/pdf"),
						})
					} else {
						panic(errors.NewUnsupportedFunctionalityError(
							"Only images and PDF file parts are supported", "",
						))
					}
				}
			}
			messages = append(messages, MistralUserMessage{
				Role:    "user",
				Content: content,
			})

		case languagemodel.AssistantMessage:
			text := ""
			var toolCalls []MistralAssistantToolCall

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ToolCallPart:
					inputJSON, err := json.Marshal(p.Input)
					if err != nil {
						inputJSON = []byte("{}")
					}
					toolCalls = append(toolCalls, MistralAssistantToolCall{
						ID:   p.ToolCallID,
						Type: "function",
						Function: MistralAssistantToolCallFunction{
							Name:      p.ToolName,
							Arguments: string(inputJSON),
						},
					})
				case languagemodel.ReasoningPart:
					text += p.Text
				default:
					panic(fmt.Sprintf("Unsupported content type in assistant message: %T", p))
				}
			}

			assistantMsg := MistralAssistantMessage{
				Role:    "assistant",
				Content: text,
			}
			if isLastMessage {
				prefix := true
				assistantMsg.Prefix = &prefix
			}
			if len(toolCalls) > 0 {
				assistantMsg.ToolCalls = toolCalls
			}
			messages = append(messages, assistantMsg)

		case languagemodel.ToolMessage:
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					// skip tool approval responses
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
						jsonBytes, err := json.Marshal(o.Value)
						if err != nil {
							contentValue = "[]"
						} else {
							contentValue = string(jsonBytes)
						}
					case languagemodel.ToolResultOutputJSON:
						jsonBytes, err := json.Marshal(o.Value)
						if err != nil {
							contentValue = "{}"
						} else {
							contentValue = string(jsonBytes)
						}
					case languagemodel.ToolResultOutputErrorJSON:
						jsonBytes, err := json.Marshal(o.Value)
						if err != nil {
							contentValue = "{}"
						} else {
							contentValue = string(jsonBytes)
						}
					}

					messages = append(messages, MistralToolMessage{
						Role:       "tool",
						Name:       p.ToolName,
						ToolCallID: p.ToolCallID,
						Content:    contentValue,
					})
				}
			}

		default:
			panic(fmt.Sprintf("Unsupported role: %T", m))
		}
	}

	return messages
}
