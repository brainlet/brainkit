// Ported from: packages/openai-compatible/src/chat/convert-to-openai-compatible-chat-messages.ts
package openaicompatible

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// getOpenAIMetadata extracts the openaiCompatible metadata from provider options.
func getOpenAIMetadata(providerOptions shared.ProviderOptions) jsonvalue.JSONObject {
	if providerOptions == nil {
		return jsonvalue.JSONObject{}
	}
	md, ok := providerOptions["openaiCompatible"]
	if !ok || md == nil {
		return jsonvalue.JSONObject{}
	}
	return md
}

// getAudioFormat returns the OpenAI-compatible audio format string for a media type,
// or empty string if unsupported.
func getAudioFormat(mediaType string) string {
	switch mediaType {
	case "audio/wav":
		return "wav"
	case "audio/mp3", "audio/mpeg":
		return "mp3"
	default:
		return ""
	}
}

// mergeMetadata creates a new map from base fields and merges in metadata entries.
func mergeMetadata(base map[string]any, metadata jsonvalue.JSONObject) map[string]any {
	for k, v := range metadata {
		base[k] = v
	}
	return base
}

// convertDataContentToBase64 converts a DataContent value to a base64 string.
func convertDataContentToBase64(data languagemodel.DataContent) string {
	switch d := data.(type) {
	case languagemodel.DataContentBytes:
		return base64.StdEncoding.EncodeToString(d.Data)
	case languagemodel.DataContentString:
		// If it's already a base64 string, return as-is.
		return d.Value
	default:
		return ""
	}
}

// isURL checks if a DataContent is a URL (DataContentString starting with http:// or https://).
func isURL(data languagemodel.DataContent) (string, bool) {
	if d, ok := data.(languagemodel.DataContentString); ok {
		if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
			return d.Value, true
		}
	}
	return "", false
}

// dataContentToString decodes a DataContent value to a UTF-8 string.
func dataContentToString(data languagemodel.DataContent) string {
	switch d := data.(type) {
	case languagemodel.DataContentBytes:
		return string(d.Data)
	case languagemodel.DataContentString:
		// Check if it's a URL
		if strings.HasPrefix(d.Value, "http://") || strings.HasPrefix(d.Value, "https://") {
			return d.Value
		}
		// Assume base64, decode to string
		decoded, err := base64.StdEncoding.DecodeString(d.Value)
		if err != nil {
			return d.Value
		}
		return string(decoded)
	default:
		return ""
	}
}

// ConvertToChatMessages converts a languagemodel.Prompt into an OpenAI-compatible chat prompt.
// Returns a slice of map[string]any, where each map represents a message with arbitrary
// provider metadata extensions (matching the TS JsonRecord pattern).
func ConvertToChatMessages(prompt languagemodel.Prompt) []map[string]any {
	messages := []map[string]any{}

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			metadata := getOpenAIMetadata(m.ProviderOptions)
			sysMsg := mergeMetadata(map[string]any{
				"role":    "system",
				"content": m.Content,
			}, metadata)
			messages = append(messages, sysMsg)

		case languagemodel.UserMessage:
			metadata := getOpenAIMetadata(m.ProviderOptions)

			// Optimization: if single text part, send as string content
			if len(m.Content) == 1 {
				if tp, ok := m.Content[0].(languagemodel.TextPart); ok {
					partMetadata := getOpenAIMetadata(tp.ProviderOptions)
					userMsg := mergeMetadata(map[string]any{
						"role":    "user",
						"content": tp.Text,
					}, partMetadata)
					messages = append(messages, userMsg)
					continue
				}
			}

			// Multi-part content
			contentParts := []any{}
			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					partMetadata := getOpenAIMetadata(p.ProviderOptions)
					contentPart := mergeMetadata(map[string]any{
						"type": "text",
						"text": p.Text,
					}, partMetadata)
					contentParts = append(contentParts, contentPart)

				case languagemodel.FilePart:
					partMetadata := getOpenAIMetadata(p.ProviderOptions)

					if strings.HasPrefix(p.MediaType, "image/") {
						mediaType := p.MediaType
						if mediaType == "image/*" {
							mediaType = "image/jpeg"
						}

						var url string
						if u, isURL := isURL(p.Data); isURL {
							url = u
						} else {
							b64 := convertDataContentToBase64(p.Data)
							url = fmt.Sprintf("data:%s;base64,%s", mediaType, b64)
						}

						contentPart := mergeMetadata(map[string]any{
							"type": "image_url",
							"image_url": map[string]any{
								"url": url,
							},
						}, partMetadata)
						contentParts = append(contentParts, contentPart)

					} else if strings.HasPrefix(p.MediaType, "audio/") {
						if _, isURLData := isURL(p.Data); isURLData {
							panic(errors.NewUnsupportedFunctionalityError("audio file parts with URLs", ""))
						}

						format := getAudioFormat(p.MediaType)
						if format == "" {
							panic(errors.NewUnsupportedFunctionalityError(
								fmt.Sprintf("audio media type %s", p.MediaType), "",
							))
						}

						b64 := convertDataContentToBase64(p.Data)
						contentPart := mergeMetadata(map[string]any{
							"type": "input_audio",
							"input_audio": map[string]any{
								"data":   b64,
								"format": format,
							},
						}, partMetadata)
						contentParts = append(contentParts, contentPart)

					} else if p.MediaType == "application/pdf" {
						if _, isURLData := isURL(p.Data); isURLData {
							panic(errors.NewUnsupportedFunctionalityError("PDF file parts with URLs", ""))
						}

						filename := "document.pdf"
						if p.Filename != nil {
							filename = *p.Filename
						}

						b64 := convertDataContentToBase64(p.Data)
						contentPart := mergeMetadata(map[string]any{
							"type": "file",
							"file": map[string]any{
								"filename":  filename,
								"file_data": fmt.Sprintf("data:application/pdf;base64,%s", b64),
							},
						}, partMetadata)
						contentParts = append(contentParts, contentPart)

					} else if strings.HasPrefix(p.MediaType, "text/") {
						textContent := dataContentToString(p.Data)
						contentPart := mergeMetadata(map[string]any{
							"type": "text",
							"text": textContent,
						}, partMetadata)
						contentParts = append(contentParts, contentPart)

					} else {
						panic(errors.NewUnsupportedFunctionalityError(
							fmt.Sprintf("file part media type %s", p.MediaType), "",
						))
					}
				}
			}

			userMsg := mergeMetadata(map[string]any{
				"role":    "user",
				"content": contentParts,
			}, metadata)
			messages = append(messages, userMsg)

		case languagemodel.AssistantMessage:
			metadata := getOpenAIMetadata(m.ProviderOptions)
			text := ""
			reasoning := ""
			var toolCalls []map[string]any

			for _, part := range m.Content {
				switch p := part.(type) {
				case languagemodel.TextPart:
					text += p.Text
				case languagemodel.ReasoningPart:
					reasoning += p.Text
				case languagemodel.ToolCallPart:
					partMetadata := getOpenAIMetadata(p.ProviderOptions)

					inputJSON, _ := json.Marshal(p.Input)

					tc := mergeMetadata(map[string]any{
						"id":   p.ToolCallID,
						"type": "function",
						"function": map[string]any{
							"name":      p.ToolName,
							"arguments": string(inputJSON),
						},
					}, partMetadata)

					// Handle Google Gemini thought signatures
					if p.ProviderOptions != nil {
						if google, ok := p.ProviderOptions["google"]; ok && google != nil {
							if sig, ok := google["thoughtSignature"]; ok {
								tc["extra_content"] = map[string]any{
									"google": map[string]any{
										"thought_signature": fmt.Sprintf("%v", sig),
									},
								}
							}
						}
					}

					toolCalls = append(toolCalls, tc)
				}
			}

			assistantMsg := map[string]any{
				"role":    "assistant",
				"content": text,
			}
			if reasoning != "" {
				assistantMsg["reasoning_content"] = reasoning
			}
			if len(toolCalls) > 0 {
				assistantMsg["tool_calls"] = toolCalls
			}
			assistantMsg = mergeMetadata(assistantMsg, metadata)
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

				toolResponseMetadata := getOpenAIMetadata(tr.ProviderOptions)
				toolMsg := mergeMetadata(map[string]any{
					"role":         "tool",
					"tool_call_id": tr.ToolCallID,
					"content":      contentValue,
				}, toolResponseMetadata)
				messages = append(messages, toolMsg)
			}

		default:
			panic(fmt.Sprintf("Unsupported role: %T", msg))
		}
	}

	return messages
}
