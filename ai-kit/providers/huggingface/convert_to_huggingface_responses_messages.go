// Ported from: packages/huggingface/src/responses/convert-to-huggingface-responses-messages.ts
package huggingface

import (
	"fmt"
	"net/url"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// convertToHuggingFaceResponsesMessagesResult holds the result of message conversion.
type convertToHuggingFaceResponsesMessagesResult struct {
	Input    []map[string]any
	Warnings []shared.Warning
}

// convertToHuggingFaceResponsesMessages converts a language model prompt
// to the HuggingFace responses message format.
func convertToHuggingFaceResponsesMessages(prompt languagemodel.Prompt) (convertToHuggingFaceResponsesMessagesResult, error) {
	messages := []map[string]any{}
	warnings := []shared.Warning{}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			messages = append(messages, map[string]any{
				"role":    "system",
				"content": m.Content,
			})

		case languagemodel.UserMessage:
			contentParts := []map[string]any{}
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					contentParts = append(contentParts, map[string]any{
						"type": "input_text",
						"text": p.Text,
					})
				case languagemodel.FilePart:
					if len(p.MediaType) >= 6 && p.MediaType[:6] == "image/" {
						mediaType := p.MediaType
						if mediaType == "image/*" {
							mediaType = "image/jpeg"
						}

						var imageURL string
						switch d := p.Data.(type) {
						case languagemodel.DataContentString:
							// Check if it's a URL
							if _, err := url.ParseRequestURI(d.Value); err == nil &&
								(len(d.Value) > 8 && (d.Value[:7] == "http://" || d.Value[:8] == "https://")) {
								imageURL = d.Value
							} else {
								// Base64 data
								imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, d.Value)
							}
						case languagemodel.DataContentBytes:
							// Not directly supported, would need base64 encoding
							return convertToHuggingFaceResponsesMessagesResult{}, errors.NewUnsupportedFunctionalityError(
								"binary file data in user message",
								"",
							)
						default:
							return convertToHuggingFaceResponsesMessagesResult{}, fmt.Errorf("unsupported data content type: %T", p.Data)
						}

						contentParts = append(contentParts, map[string]any{
							"type":      "input_image",
							"image_url": imageURL,
						})
					} else {
						return convertToHuggingFaceResponsesMessagesResult{}, errors.NewUnsupportedFunctionalityError(
							fmt.Sprintf("file part media type %s", p.MediaType),
							"",
						)
					}
				default:
					return convertToHuggingFaceResponsesMessagesResult{}, fmt.Errorf("unsupported part type: %T", part)
				}
			}
			messages = append(messages, map[string]any{
				"role":    "user",
				"content": contentParts,
			})

		case languagemodel.AssistantMessage:
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					messages = append(messages, map[string]any{
						"role": "assistant",
						"content": []map[string]any{
							{"type": "output_text", "text": p.Text},
						},
					})
				case languagemodel.ToolCallPart:
					// Tool calls are handled by the responses API.
				case languagemodel.ToolResultPart:
					// Tool results are handled by the responses API.
				case languagemodel.ReasoningPart:
					// Include reasoning content in the message text.
					messages = append(messages, map[string]any{
						"role": "assistant",
						"content": []map[string]any{
							{"type": "output_text", "text": p.Text},
						},
					})
				}
			}

		case languagemodel.ToolMessage:
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "tool messages",
			})

		default:
			return convertToHuggingFaceResponsesMessagesResult{}, fmt.Errorf("unsupported role: %T", msg)
		}
	}

	return convertToHuggingFaceResponsesMessagesResult{
		Input:    messages,
		Warnings: warnings,
	}, nil
}
