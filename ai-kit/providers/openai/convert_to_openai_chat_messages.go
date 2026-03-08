// Ported from: packages/openai/src/chat/convert-to-openai-chat-messages.ts
package openai

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ConvertToOpenAIChatMessagesResult holds the result of converting a prompt.
type ConvertToOpenAIChatMessagesResult struct {
	Messages OpenAIChatPrompt
	Warnings []shared.Warning
}

// ConvertToOpenAIChatMessages converts a standard prompt to OpenAI chat messages.
func ConvertToOpenAIChatMessages(
	prompt languagemodel.Prompt,
	systemMessageMode string, // "system" | "developer" | "remove"
) ConvertToOpenAIChatMessagesResult {
	if systemMessageMode == "" {
		systemMessageMode = "system"
	}

	messages := OpenAIChatPrompt{}
	warnings := []shared.Warning{}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			switch systemMessageMode {
			case "system":
				messages = append(messages, ChatCompletionSystemMessage{
					Role:    "system",
					Content: m.Content,
				})
			case "developer":
				messages = append(messages, ChatCompletionDeveloperMessage{
					Role:    "developer",
					Content: m.Content,
				})
			case "remove":
				warnings = append(warnings, shared.OtherWarning{
					Message: "system messages are removed for this model",
				})
			default:
				panic(fmt.Sprintf("Unsupported system message mode: %s", systemMessageMode))
			}

		case languagemodel.UserMessage:
			// Optimization: if single text part, send as string content
			if len(m.Content) == 1 {
				if tp, ok := m.Content[0].(languagemodel.TextPart); ok {
					messages = append(messages, ChatCompletionUserMessage{
						Role:    "user",
						Content: tp.Text,
					})
					continue
				}
			}

			parts := []ChatCompletionContentPart{}
			for i, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					parts = append(parts, ChatCompletionContentPartText{
						Type: "text",
						Text: p.Text,
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
							// Check if it's a URL
							if _, err := url.ParseRequestURI(d.Value); err == nil &&
								(strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://")) {
								imageURL = d.Value
							} else {
								imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, providerutils.ConvertToBase64String(d.Value))
							}
						case languagemodel.DataContentBytes:
							imageURL = fmt.Sprintf("data:%s;base64,%s", mediaType, providerutils.ConvertToBase64Bytes(d.Data))
						}

						// Extract OpenAI-specific imageDetail from providerOptions
						var detail any
						if p.ProviderOptions != nil {
							if openaiOpts, ok := p.ProviderOptions["openai"]; ok {
								if d, ok := openaiOpts["imageDetail"]; ok {
									detail = d
								}
							}
						}

						parts = append(parts, ChatCompletionContentPartImage{
							Type: "image_url",
							ImageURL: ChatCompletionContentPartImageURL{
								URL:    imageURL,
								Detail: detail,
							},
						})

					} else if strings.HasPrefix(p.MediaType, "audio/") {
						// Audio parts don't support URLs
						switch d := p.Data.(type) {
						case languagemodel.DataContentString:
							if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
								panic(errors.NewUnsupportedFunctionalityError("audio file parts with URLs", ""))
							}

							switch p.MediaType {
							case "audio/wav":
								parts = append(parts, ChatCompletionContentPartInputAudio{
									Type: "input_audio",
									InputAudio: ChatCompletionContentPartInputAudioData{
										Data:   providerutils.ConvertToBase64String(d.Value),
										Format: "wav",
									},
								})
							case "audio/mp3", "audio/mpeg":
								parts = append(parts, ChatCompletionContentPartInputAudio{
									Type: "input_audio",
									InputAudio: ChatCompletionContentPartInputAudioData{
										Data:   providerutils.ConvertToBase64String(d.Value),
										Format: "mp3",
									},
								})
							default:
								panic(errors.NewUnsupportedFunctionalityError(
									fmt.Sprintf("audio content parts with media type %s", p.MediaType), "",
								))
							}

						case languagemodel.DataContentBytes:
							switch p.MediaType {
							case "audio/wav":
								parts = append(parts, ChatCompletionContentPartInputAudio{
									Type: "input_audio",
									InputAudio: ChatCompletionContentPartInputAudioData{
										Data:   providerutils.ConvertToBase64Bytes(d.Data),
										Format: "wav",
									},
								})
							case "audio/mp3", "audio/mpeg":
								parts = append(parts, ChatCompletionContentPartInputAudio{
									Type: "input_audio",
									InputAudio: ChatCompletionContentPartInputAudioData{
										Data:   providerutils.ConvertToBase64Bytes(d.Data),
										Format: "mp3",
									},
								})
							default:
								panic(errors.NewUnsupportedFunctionalityError(
									fmt.Sprintf("audio content parts with media type %s", p.MediaType), "",
								))
							}
						}

					} else if p.MediaType == "application/pdf" {
						// PDF parts don't support URLs
						switch d := p.Data.(type) {
						case languagemodel.DataContentString:
							if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
								panic(errors.NewUnsupportedFunctionalityError("PDF file parts with URLs", ""))
							}

							// Check if it's a file ID reference
							if strings.HasPrefix(d.Value, "file-") {
								parts = append(parts, ChatCompletionContentPartFile{
									Type: "file",
									File: ChatCompletionFileByID{
										FileID: d.Value,
									},
								})
							} else {
								filename := fmt.Sprintf("part-%d.pdf", i)
								if p.Filename != nil {
									filename = *p.Filename
								}
								parts = append(parts, ChatCompletionContentPartFile{
									Type: "file",
									File: ChatCompletionFileByData{
										Filename: filename,
										FileData: fmt.Sprintf("data:application/pdf;base64,%s", providerutils.ConvertToBase64String(d.Value)),
									},
								})
							}

						case languagemodel.DataContentBytes:
							filename := fmt.Sprintf("part-%d.pdf", i)
							if p.Filename != nil {
								filename = *p.Filename
							}
							parts = append(parts, ChatCompletionContentPartFile{
								Type: "file",
								File: ChatCompletionFileByData{
									Filename: filename,
									FileData: fmt.Sprintf("data:application/pdf;base64,%s", providerutils.ConvertToBase64Bytes(d.Data)),
								},
							})
						}

					} else {
						panic(errors.NewUnsupportedFunctionalityError(
							fmt.Sprintf("file part media type %s", p.MediaType), "",
						))
					}
				}
			}

			messages = append(messages, ChatCompletionUserMessage{
				Role:    "user",
				Content: parts,
			})

		case languagemodel.AssistantMessage:
			text := ""
			toolCalls := []ChatCompletionMessageToolCall{}

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ToolCallPart:
					inputStr := ""
					if p.Input != nil {
						b, err := json.Marshal(p.Input)
						if err == nil {
							inputStr = string(b)
						}
					}
					toolCalls = append(toolCalls, ChatCompletionMessageToolCall{
						ID:   p.ToolCallID,
						Type: "function",
						Function: ChatCompletionMessageToolCallFunction{
							Name:      p.ToolName,
							Arguments: inputStr,
						},
					})
				}
			}

			msg := ChatCompletionAssistantMessage{
				Role:    "assistant",
				Content: text,
			}
			if len(toolCalls) > 0 {
				msg.ToolCalls = toolCalls
			}
			messages = append(messages, msg)

		case languagemodel.ToolMessage:
			for _, part := range m.Content {
				switch tp := part.(type) {
				case languagemodel.ToolApprovalResponsePart:
					// skip tool approval responses
					continue
				case languagemodel.ToolResultPart:
					var contentValue string

					switch output := tp.Output.(type) {
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
						b, err := json.Marshal(output.Value)
						if err == nil {
							contentValue = string(b)
						}
					case languagemodel.ToolResultOutputJSON:
						b, err := json.Marshal(output.Value)
						if err == nil {
							contentValue = string(b)
						}
					case languagemodel.ToolResultOutputErrorJSON:
						b, err := json.Marshal(output.Value)
						if err == nil {
							contentValue = string(b)
						}
					}

					messages = append(messages, ChatCompletionToolMessage{
						Role:       "tool",
						ToolCallID: tp.ToolCallID,
						Content:    contentValue,
					})
				}
			}

		default:
			panic(fmt.Sprintf("Unsupported role: %T", m))
		}
	}

	return ConvertToOpenAIChatMessagesResult{
		Messages: messages,
		Warnings: warnings,
	}
}
