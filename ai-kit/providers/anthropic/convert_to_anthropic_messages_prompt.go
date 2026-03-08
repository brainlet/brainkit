// Ported from: packages/anthropic/src/convert-to-anthropic-messages-prompt.ts
package anthropic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// ConvertToAnthropicMessagesPromptResult is the result of prompt conversion.
type ConvertToAnthropicMessagesPromptResult struct {
	Prompt AnthropicMessagesPrompt
	Betas  map[string]bool
}

// convertToAnthropicMessagesPrompt converts an SDK prompt to the Anthropic Messages API format.
func convertToAnthropicMessagesPrompt(
	prompt languagemodel.Prompt,
	sendReasoning bool,
	warnings *[]shared.Warning,
	cacheControlValidator *CacheControlValidator,
) ConvertToAnthropicMessagesPromptResult {
	betas := map[string]bool{}
	blocks := groupIntoBlocks(prompt)
	validator := cacheControlValidator
	if validator == nil {
		validator = NewCacheControlValidator()
	}

	var system []AnthropicTextContent
	var messages []AnthropicMessage

	for i, block := range blocks {
		_ = i // used for isLastBlock checks if needed

		switch block.blockType {
		case "system":
			if system != nil {
				// Multiple system messages separated by user/assistant messages
				// is not supported; just append to the first system
			}

			system = []AnthropicTextContent{}
			for _, msg := range block.systemMessages {
				cc := validator.GetCacheControl(msg.ProviderOptions, CacheControlContext{
					Type:     "system message",
					CanCache: true,
				})
				system = append(system, AnthropicTextContent{
					Type:         "text",
					Text:         msg.Content,
					CacheControl: cc,
				})
			}

		case "user":
			content := []any{}

			for _, msg := range block.userMessages {
				switch m := msg.(type) {
				case languagemodel.UserMessage:
					for j, part := range m.Content {
						isLastPart := j == len(m.Content)-1

						cc := validator.GetCacheControl(getPartProviderOptions(part), CacheControlContext{
							Type:     "user message part",
							CanCache: true,
						})
						if cc == nil && isLastPart {
							cc = validator.GetCacheControl(m.ProviderOptions, CacheControlContext{
								Type:     "user message",
								CanCache: true,
							})
						}

						switch p := part.(type) {
						case languagemodel.TextPart:
							content = append(content, map[string]any{
								"type":          "text",
								"text":          p.Text,
								"cache_control": cacheControlToMap(cc),
							})

						case languagemodel.FilePart:
							if strings.HasPrefix(p.MediaType, "image/") {
								source := buildContentSource(p, "image")
								content = append(content, map[string]any{
									"type":          "image",
									"source":        source,
									"cache_control": cacheControlToMap(cc),
								})
							} else if p.MediaType == "application/pdf" || strings.HasPrefix(p.MediaType, "text/") {
								source := buildContentSource(p, "document")
								doc := map[string]any{
									"type":          "document",
									"source":        source,
									"cache_control": cacheControlToMap(cc),
								}

								// Check for citations and metadata from provider options
								if p.ProviderOptions != nil {
									if antOpts, ok := p.ProviderOptions["anthropic"]; ok && antOpts != nil {
										if cit, ok := antOpts["citations"]; ok {
											doc["citations"] = cit
										}
										if t, ok := antOpts["title"].(string); ok {
											doc["title"] = t
										}
										if ctx, ok := antOpts["context"].(string); ok {
											doc["context"] = ctx
										}
									}
								}

								if p.Filename != nil {
									if _, ok := doc["title"]; !ok {
										doc["title"] = *p.Filename
									}
								}

								content = append(content, doc)
							} else {
								*warnings = append(*warnings, shared.UnsupportedWarning{
									Feature: fmt.Sprintf("file media type %s", p.MediaType),
								})
							}
						}
					}

				case languagemodel.ToolMessage:
					for _, part := range m.Content {
						switch p := part.(type) {
						case languagemodel.ToolResultPart:
							toolResult := convertToolResultToAnthropic(p, validator, m.ProviderOptions)
							content = append(content, toolResult)

						case languagemodel.ToolApprovalResponsePart:
							// Approval responses are not directly supported in prompt conversion
							*warnings = append(*warnings, shared.UnsupportedWarning{
								Feature: "tool approval response in prompt",
							})
						}
					}
				}
			}

			messages = append(messages, AnthropicMessage{
				Role:    "user",
				Content: content,
			})

		case "assistant":
			content := []any{}

			for _, msg := range block.assistantMessages {
				for j, part := range msg.Content {
					isLastPart := j == len(msg.Content)-1

					cc := validator.GetCacheControl(getAssistantPartProviderOptions(part), CacheControlContext{
						Type:     "assistant message part",
						CanCache: true,
					})
					if cc == nil && isLastPart {
						cc = validator.GetCacheControl(msg.ProviderOptions, CacheControlContext{
							Type:     "assistant message",
							CanCache: true,
						})
					}

					switch p := part.(type) {
					case languagemodel.TextPart:
						content = append(content, map[string]any{
							"type":          "text",
							"text":          p.Text,
							"cache_control": cacheControlToMap(cc),
						})

					case languagemodel.ReasoningPart:
						if !sendReasoning {
							continue
						}
						// Extract signature from provider metadata
						var signature string
						if p.ProviderOptions != nil {
							if am, ok := p.ProviderOptions["anthropic"]; ok && am != nil {
								if s, ok := am["signature"].(string); ok {
									signature = s
								}
							}
						}

						if signature != "" {
							content = append(content, map[string]any{
								"type":      "thinking",
								"thinking":  p.Text,
								"signature": signature,
							})
						} else {
							// Redacted thinking
							content = append(content, map[string]any{
								"type": "redacted_thinking",
								"data": p.Text,
							})
						}

					case languagemodel.ToolCallPart:
						inputJSON := parseToolCallInput(p.Input)
						content = append(content, map[string]any{
							"type":          "tool_use",
							"id":            p.ToolCallID,
							"name":          p.ToolName,
							"input":         inputJSON,
							"cache_control": cacheControlToMap(cc),
						})

					case languagemodel.ToolResultPart:
						toolResult := convertToolResultToAnthropic(p, validator, msg.ProviderOptions)
						content = append(content, toolResult)

					case languagemodel.FilePart:
						// Files in assistant messages are not typically supported
						*warnings = append(*warnings, shared.UnsupportedWarning{
							Feature: "file in assistant message",
						})
					}
				}
			}

			messages = append(messages, AnthropicMessage{
				Role:    "assistant",
				Content: content,
			})
		}
	}

	return ConvertToAnthropicMessagesPromptResult{
		Prompt: AnthropicMessagesPrompt{
			System:   system,
			Messages: messages,
		},
		Betas: betas,
	}
}

// block types for grouping messages
type messageBlock struct {
	blockType         string // "system", "user", "assistant"
	systemMessages    []languagemodel.SystemMessage
	userMessages      []languagemodel.Message // UserMessage or ToolMessage
	assistantMessages []languagemodel.AssistantMessage
}

// groupIntoBlocks groups consecutive messages by role.
func groupIntoBlocks(prompt languagemodel.Prompt) []messageBlock {
	var blocks []messageBlock

	for _, msg := range prompt {
		switch m := msg.(type) {
		case languagemodel.SystemMessage:
			if len(blocks) > 0 && blocks[len(blocks)-1].blockType == "system" {
				blocks[len(blocks)-1].systemMessages = append(blocks[len(blocks)-1].systemMessages, m)
			} else {
				blocks = append(blocks, messageBlock{
					blockType:      "system",
					systemMessages: []languagemodel.SystemMessage{m},
				})
			}

		case languagemodel.UserMessage:
			if len(blocks) > 0 && blocks[len(blocks)-1].blockType == "user" {
				blocks[len(blocks)-1].userMessages = append(blocks[len(blocks)-1].userMessages, m)
			} else {
				blocks = append(blocks, messageBlock{
					blockType:    "user",
					userMessages: []languagemodel.Message{m},
				})
			}

		case languagemodel.ToolMessage:
			if len(blocks) > 0 && blocks[len(blocks)-1].blockType == "user" {
				blocks[len(blocks)-1].userMessages = append(blocks[len(blocks)-1].userMessages, m)
			} else {
				blocks = append(blocks, messageBlock{
					blockType:    "user",
					userMessages: []languagemodel.Message{m},
				})
			}

		case languagemodel.AssistantMessage:
			if len(blocks) > 0 && blocks[len(blocks)-1].blockType == "assistant" {
				blocks[len(blocks)-1].assistantMessages = append(blocks[len(blocks)-1].assistantMessages, m)
			} else {
				blocks = append(blocks, messageBlock{
					blockType:         "assistant",
					assistantMessages: []languagemodel.AssistantMessage{m},
				})
			}
		}
	}

	return blocks
}

// convertToolResultToAnthropic converts a tool result part to an Anthropic API format.
func convertToolResultToAnthropic(
	p languagemodel.ToolResultPart,
	validator *CacheControlValidator,
	messageProviderOptions shared.ProviderOptions,
) map[string]any {
	cc := validator.GetCacheControl(p.ProviderOptions, CacheControlContext{
		Type:     "tool result",
		CanCache: true,
	})

	result := map[string]any{
		"type":          "tool_result",
		"tool_use_id":   p.ToolCallID,
		"cache_control": cacheControlToMap(cc),
	}

	// Convert the output
	switch out := p.Output.(type) {
	case languagemodel.ToolResultOutputText:
		result["content"] = out.Value
		result["is_error"] = false

	case languagemodel.ToolResultOutputJSON:
		jsonStr, err := json.Marshal(out.Value)
		if err != nil {
			result["content"] = fmt.Sprintf("%v", out.Value)
		} else {
			result["content"] = string(jsonStr)
		}
		result["is_error"] = false

	case languagemodel.ToolResultOutputErrorText:
		result["content"] = out.Value
		result["is_error"] = true

	case languagemodel.ToolResultOutputErrorJSON:
		jsonStr, err := json.Marshal(out.Value)
		if err != nil {
			result["content"] = fmt.Sprintf("%v", out.Value)
		} else {
			result["content"] = string(jsonStr)
		}
		result["is_error"] = true

	case languagemodel.ToolResultOutputExecutionDenied:
		reason := "Tool execution was denied"
		if out.Reason != nil {
			reason = *out.Reason
		}
		result["content"] = reason
		result["is_error"] = true

	case languagemodel.ToolResultOutputContent:
		contentParts := []any{}
		for _, cp := range out.Value {
			switch p := cp.(type) {
			case languagemodel.ToolResultContentText:
				contentParts = append(contentParts, map[string]any{
					"type": "text",
					"text": p.Text,
				})
			case languagemodel.ToolResultContentImageData:
				contentParts = append(contentParts, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type":       "base64",
						"media_type": p.MediaType,
						"data":       p.Data,
					},
				})
			case languagemodel.ToolResultContentImageURL:
				contentParts = append(contentParts, map[string]any{
					"type": "image",
					"source": map[string]any{
						"type": "url",
						"url":  p.URL,
					},
				})
			}
		}
		result["content"] = contentParts
		result["is_error"] = false

	default:
		result["content"] = ""
		result["is_error"] = false
	}

	return result
}

// buildContentSource builds an AnthropicContentSource from a FilePart.
func buildContentSource(p languagemodel.FilePart, contentType string) map[string]any {
	switch d := p.Data.(type) {
	case languagemodel.DataContentBytes:
		mediaType := p.MediaType
		if contentType == "image" && mediaType == "image/*" {
			mediaType = "image/jpeg"
		}
		encoded := base64.StdEncoding.EncodeToString(d.Data)
		if contentType == "document" && strings.HasPrefix(mediaType, "text/") {
			return map[string]any{
				"type":       "text",
				"media_type": mediaType,
				"data":       string(d.Data),
			}
		}
		return map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       encoded,
		}
	case languagemodel.DataContentString:
		s := d.Value
		if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
			return map[string]any{
				"type": "url",
				"url":  s,
			}
		}
		mediaType := p.MediaType
		if contentType == "image" && mediaType == "image/*" {
			mediaType = "image/jpeg"
		}
		if contentType == "document" && strings.HasPrefix(mediaType, "text/") {
			return map[string]any{
				"type":       "text",
				"media_type": mediaType,
				"data":       decodeBase64ToString(s),
			}
		}
		return map[string]any{
			"type":       "base64",
			"media_type": mediaType,
			"data":       s,
		}
	default:
		return map[string]any{
			"type":       "base64",
			"media_type": p.MediaType,
			"data":       "",
		}
	}
}

// cacheControlToMap converts an AnthropicCacheControl pointer to a map, or nil.
func cacheControlToMap(cc *AnthropicCacheControl) any {
	if cc == nil {
		return nil
	}
	m := map[string]any{"type": cc.Type}
	if cc.TTL != nil {
		m["ttl"] = *cc.TTL
	}
	return m
}

// parseToolCallInput parses tool call input from any to a JSON-compatible value.
func parseToolCallInput(input any) any {
	if input == nil {
		return map[string]any{}
	}
	if s, ok := input.(string); ok {
		var parsed any
		if err := json.Unmarshal([]byte(s), &parsed); err == nil {
			return parsed
		}
		return input
	}
	return input
}

// getPartProviderOptions extracts provider options from a user message part.
func getPartProviderOptions(part languagemodel.UserMessagePart) shared.ProviderOptions {
	switch p := part.(type) {
	case languagemodel.TextPart:
		return p.ProviderOptions
	case languagemodel.FilePart:
		return p.ProviderOptions
	default:
		return nil
	}
}

// getAssistantPartProviderOptions extracts provider options from an assistant message part.
func getAssistantPartProviderOptions(part languagemodel.AssistantMessagePart) shared.ProviderOptions {
	switch p := part.(type) {
	case languagemodel.TextPart:
		return p.ProviderOptions
	case languagemodel.ReasoningPart:
		return p.ProviderOptions
	case languagemodel.ToolCallPart:
		return p.ProviderOptions
	case languagemodel.ToolResultPart:
		return p.ProviderOptions
	case languagemodel.FilePart:
		return p.ProviderOptions
	default:
		return nil
	}
}

// decodeBase64ToString decodes a base64 string to a UTF-8 string.
func decodeBase64ToString(data string) string {
	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return data
	}
	return string(decoded)
}
