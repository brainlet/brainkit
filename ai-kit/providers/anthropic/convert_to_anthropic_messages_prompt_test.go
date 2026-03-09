// Ported from: packages/anthropic/src/convert-to-anthropic-messages-prompt.test.ts
package anthropic

import (
	"encoding/base64"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// --- System messages ---

func TestConvertPrompt_SystemMessages(t *testing.T) {
	t.Run("should convert a single system message into an anthropic system message", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "This is a system message"},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result.Prompt.Messages))
		}
		if len(result.Prompt.System) != 1 {
			t.Fatalf("expected 1 system block, got %d", len(result.Prompt.System))
		}
		if result.Prompt.System[0].Type != "text" {
			t.Errorf("expected type 'text', got %q", result.Prompt.System[0].Type)
		}
		if result.Prompt.System[0].Text != "This is a system message" {
			t.Errorf("expected text 'This is a system message', got %q", result.Prompt.System[0].Text)
		}
		if len(result.Betas) != 0 {
			t.Errorf("expected no betas, got %v", result.Betas)
		}
	})

	t.Run("should convert multiple system messages into an anthropic system message", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "This is a system message"},
				languagemodel.SystemMessage{Content: "This is another system message"},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result.Prompt.Messages))
		}
		if len(result.Prompt.System) != 2 {
			t.Fatalf("expected 2 system blocks, got %d", len(result.Prompt.System))
		}
		if result.Prompt.System[0].Text != "This is a system message" {
			t.Errorf("expected first system text 'This is a system message', got %q", result.Prompt.System[0].Text)
		}
		if result.Prompt.System[1].Text != "This is another system message" {
			t.Errorf("expected second system text 'This is another system message', got %q", result.Prompt.System[1].Text)
		}
	})
}

// --- User messages ---

func TestConvertPrompt_UserMessages(t *testing.T) {
	t.Run("should add image parts for base64 images", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
							MediaType: "image/png",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Prompt.Messages))
		}
		msg := result.Prompt.Messages[0]
		if msg.Role != "user" {
			t.Errorf("expected role 'user', got %q", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(msg.Content))
		}
		part := msg.Content[0].(map[string]any)
		if part["type"] != "image" {
			t.Errorf("expected type 'image', got %v", part["type"])
		}
		source := part["source"].(map[string]any)
		if source["type"] != "base64" {
			t.Errorf("expected source type 'base64', got %v", source["type"])
		}
		if source["media_type"] != "image/png" {
			t.Errorf("expected media_type 'image/png', got %v", source["media_type"])
		}
		if source["data"] != "AAECAw==" {
			t.Errorf("expected data 'AAECAw==', got %v", source["data"])
		}
	})

	t.Run("should add image parts for URL images", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "https://example.com/image.png"},
							MediaType: "image/*",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Prompt.Messages))
		}
		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "image" {
			t.Errorf("expected type 'image', got %v", part["type"])
		}
		source := part["source"].(map[string]any)
		if source["type"] != "url" {
			t.Errorf("expected source type 'url', got %v", source["type"])
		}
		if source["url"] != "https://example.com/image.png" {
			t.Errorf("expected url 'https://example.com/image.png', got %v", source["url"])
		}
	})

	t.Run("should treat URL strings in image file data as URLs not base64", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "https://example.com/image.png"},
							MediaType: "image/png",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		source := part["source"].(map[string]any)
		if source["type"] != "url" {
			t.Errorf("expected source type 'url', got %v", source["type"])
		}
		if source["url"] != "https://example.com/image.png" {
			t.Errorf("expected url, got %v", source["url"])
		}
	})

	t.Run("should treat URL strings in PDF file data as URLs not base64", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "https://example.com/document.pdf"},
							MediaType: "application/pdf",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "document" {
			t.Errorf("expected type 'document', got %v", part["type"])
		}
		source := part["source"].(map[string]any)
		if source["type"] != "url" {
			t.Errorf("expected source type 'url', got %v", source["type"])
		}
		if source["url"] != "https://example.com/document.pdf" {
			t.Errorf("expected url 'https://example.com/document.pdf', got %v", source["url"])
		}
	})

	t.Run("should add PDF file parts for base64 PDFs", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata"},
							MediaType: "application/pdf",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "document" {
			t.Errorf("expected type 'document', got %v", part["type"])
		}
		source := part["source"].(map[string]any)
		if source["type"] != "base64" {
			t.Errorf("expected source type 'base64', got %v", source["type"])
		}
		if source["media_type"] != "application/pdf" {
			t.Errorf("expected media_type 'application/pdf', got %v", source["media_type"])
		}
		if source["data"] != "base64PDFdata" {
			t.Errorf("expected data 'base64PDFdata', got %v", source["data"])
		}
	})

	t.Run("should add text file parts for text/plain documents", func(t *testing.T) {
		warnings := []shared.Warning{}
		textContent := "sample text content"
		encoded := base64.StdEncoding.EncodeToString([]byte(textContent))
		filename := "sample.txt"
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: encoded},
							MediaType: "text/plain",
							Filename:  &filename,
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "document" {
			t.Errorf("expected type 'document', got %v", part["type"])
		}
		source := part["source"].(map[string]any)
		if source["type"] != "text" {
			t.Errorf("expected source type 'text', got %v", source["type"])
		}
		if source["media_type"] != "text/plain" {
			t.Errorf("expected media_type 'text/plain', got %v", source["media_type"])
		}
		if source["data"] != "sample text content" {
			t.Errorf("expected decoded text content, got %v", source["data"])
		}
		if part["title"] != "sample.txt" {
			t.Errorf("expected title 'sample.txt', got %v", part["title"])
		}
	})

	t.Run("should add warning for unsupported file types", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64data"},
							MediaType: "video/mp4",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		// Unsupported file types produce a warning (not an error) in Go
		if len(warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(warnings))
		}
		w, ok := warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", warnings[0])
		}
		if w.Feature != "file media type video/mp4" {
			t.Errorf("expected feature 'file media type video/mp4', got %q", w.Feature)
		}
		// No content parts added for unsupported types
		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Prompt.Messages))
		}
		if len(result.Prompt.Messages[0].Content) != 0 {
			t.Errorf("expected 0 content parts for unsupported file, got %d", len(result.Prompt.Messages[0].Content))
		}
	})
}

// --- Tool messages ---

func TestConvertPrompt_ToolMessages(t *testing.T) {
	t.Run("should convert a single tool result into an anthropic user message", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-call-1",
							ToolName:   "tool-1",
							Output: languagemodel.ToolResultOutputJSON{
								Value: map[string]any{"test": "This is a tool message"},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Prompt.Messages))
		}
		msg := result.Prompt.Messages[0]
		if msg.Role != "user" {
			t.Errorf("expected role 'user', got %q", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(msg.Content))
		}
		part := msg.Content[0].(map[string]any)
		if part["type"] != "tool_result" {
			t.Errorf("expected type 'tool_result', got %v", part["type"])
		}
		if part["tool_use_id"] != "tool-call-1" {
			t.Errorf("expected tool_use_id 'tool-call-1', got %v", part["tool_use_id"])
		}
		content := part["content"].(string)
		if content != `{"test":"This is a tool message"}` {
			t.Errorf("expected JSON content, got %q", content)
		}
	})

	t.Run("should convert multiple tool results into an anthropic user message", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-call-1",
							ToolName:   "tool-1",
							Output: languagemodel.ToolResultOutputJSON{
								Value: map[string]any{"test": "This is a tool message"},
							},
						},
						languagemodel.ToolResultPart{
							ToolCallID: "tool-call-2",
							ToolName:   "tool-2",
							Output: languagemodel.ToolResultOutputJSON{
								Value: map[string]any{"something": "else"},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Prompt.Messages))
		}
		msg := result.Prompt.Messages[0]
		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
		}
		part1 := msg.Content[0].(map[string]any)
		if part1["tool_use_id"] != "tool-call-1" {
			t.Errorf("expected tool_use_id 'tool-call-1', got %v", part1["tool_use_id"])
		}
		part2 := msg.Content[1].(map[string]any)
		if part2["tool_use_id"] != "tool-call-2" {
			t.Errorf("expected tool_use_id 'tool-call-2', got %v", part2["tool_use_id"])
		}
	})

	t.Run("should combine user and tool messages", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-call-1",
							ToolName:   "tool-1",
							Output: languagemodel.ToolResultOutputJSON{
								Value: map[string]any{"test": "This is a tool message"},
							},
						},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "This is a user message"},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 1 {
			t.Fatalf("expected 1 message (combined), got %d", len(result.Prompt.Messages))
		}
		msg := result.Prompt.Messages[0]
		if msg.Role != "user" {
			t.Errorf("expected role 'user', got %q", msg.Role)
		}
		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
		}
		// First part should be tool result
		part1 := msg.Content[0].(map[string]any)
		if part1["type"] != "tool_result" {
			t.Errorf("expected first part type 'tool_result', got %v", part1["type"])
		}
		// Second part should be text
		part2 := msg.Content[1].(map[string]any)
		if part2["type"] != "text" {
			t.Errorf("expected second part type 'text', got %v", part2["type"])
		}
		if part2["text"] != "This is a user message" {
			t.Errorf("expected text 'This is a user message', got %v", part2["text"])
		}
	})

	t.Run("should handle tool result with content parts", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "image-gen-1",
							ToolName:   "image-generator",
							Output: languagemodel.ToolResultOutputContent{
								Value: []languagemodel.ToolResultContentPart{
									languagemodel.ToolResultContentText{
										Text: "Image generated successfully",
									},
									languagemodel.ToolResultContentImageData{
										Data:      "AAECAw==",
										MediaType: "image/png",
									},
								},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		part := msg.Content[0].(map[string]any)
		if part["type"] != "tool_result" {
			t.Errorf("expected type 'tool_result', got %v", part["type"])
		}
		if part["tool_use_id"] != "image-gen-1" {
			t.Errorf("expected tool_use_id 'image-gen-1', got %v", part["tool_use_id"])
		}
		contentParts := part["content"].([]any)
		if len(contentParts) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(contentParts))
		}

		textPart := contentParts[0].(map[string]any)
		if textPart["type"] != "text" {
			t.Errorf("expected text type, got %v", textPart["type"])
		}
		if textPart["text"] != "Image generated successfully" {
			t.Errorf("expected text content, got %v", textPart["text"])
		}

		imagePart := contentParts[1].(map[string]any)
		if imagePart["type"] != "image" {
			t.Errorf("expected image type, got %v", imagePart["type"])
		}
		imageSource := imagePart["source"].(map[string]any)
		if imageSource["type"] != "base64" {
			t.Errorf("expected source type 'base64', got %v", imageSource["type"])
		}
		if imageSource["data"] != "AAECAw==" {
			t.Errorf("expected data 'AAECAw==', got %v", imageSource["data"])
		}
	})

	t.Run("should handle tool result with URL-based image content", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "image-gen-1",
							ToolName:   "image-generator",
							Output: languagemodel.ToolResultOutputContent{
								Value: []languagemodel.ToolResultContentPart{
									languagemodel.ToolResultContentImageURL{
										URL: "https://example.com/image.png",
									},
								},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		part := msg.Content[0].(map[string]any)
		contentParts := part["content"].([]any)
		imagePart := contentParts[0].(map[string]any)
		if imagePart["type"] != "image" {
			t.Errorf("expected image type, got %v", imagePart["type"])
		}
		source := imagePart["source"].(map[string]any)
		if source["type"] != "url" {
			t.Errorf("expected source type 'url', got %v", source["type"])
		}
		if source["url"] != "https://example.com/image.png" {
			t.Errorf("expected url, got %v", source["url"])
		}
	})

	t.Run("should handle tool result with error output", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-1",
							ToolName:   "test-tool",
							Output: languagemodel.ToolResultOutputErrorText{
								Value: "Something went wrong",
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["is_error"] != true {
			t.Errorf("expected is_error true, got %v", part["is_error"])
		}
		if part["content"] != "Something went wrong" {
			t.Errorf("expected error content, got %v", part["content"])
		}
	})

	t.Run("should handle tool result with text output", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-1",
							ToolName:   "test-tool",
							Output: languagemodel.ToolResultOutputText{
								Value: "Tool output text",
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["is_error"] != false {
			t.Errorf("expected is_error false, got %v", part["is_error"])
		}
		if part["content"] != "Tool output text" {
			t.Errorf("expected text content, got %v", part["content"])
		}
	})

	t.Run("should handle tool result with execution denied", func(t *testing.T) {
		warnings := []shared.Warning{}
		reason := "User denied execution"
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "tool-1",
							ToolName:   "test-tool",
							Output: languagemodel.ToolResultOutputExecutionDenied{
								Reason: &reason,
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["is_error"] != true {
			t.Errorf("expected is_error true, got %v", part["is_error"])
		}
		if part["content"] != "User denied execution" {
			t.Errorf("expected denial reason, got %v", part["content"])
		}
	})
}

// --- Assistant messages ---

func TestConvertPrompt_AssistantMessages(t *testing.T) {
	t.Run("should convert assistant message with text content", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "user content"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "assistant content"},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result.Prompt.Messages))
		}
		assistantMsg := result.Prompt.Messages[1]
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		part := assistantMsg.Content[0].(map[string]any)
		if part["text"] != "assistant content" {
			t.Errorf("expected text 'assistant content', got %v", part["text"])
		}
	})

	t.Run("should combine multiple sequential assistant messages into a single message", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hi!"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "World"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "!"},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		if len(result.Prompt.Messages) != 2 {
			t.Fatalf("expected 2 messages (user + combined assistant), got %d", len(result.Prompt.Messages))
		}
		assistantMsg := result.Prompt.Messages[1]
		if len(assistantMsg.Content) != 3 {
			t.Fatalf("expected 3 content parts in combined assistant, got %d", len(assistantMsg.Content))
		}
		texts := []string{"Hello", "World", "!"}
		for i, expected := range texts {
			part := assistantMsg.Content[i].(map[string]any)
			if part["text"] != expected {
				t.Errorf("expected text %q at index %d, got %v", expected, i, part["text"])
			}
		}
	})

	t.Run("should convert reasoning parts with signature into thinking parts when sendReasoning is true", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ReasoningPart{
							Text: `I need to count the number of "r"s in the word "strawberry".`,
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"signature": "test-signature",
								},
							},
						},
						languagemodel.TextPart{
							Text: `The word "strawberry" has 2 "r"s.`,
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
		}

		thinkingPart := msg.Content[0].(map[string]any)
		if thinkingPart["type"] != "thinking" {
			t.Errorf("expected type 'thinking', got %v", thinkingPart["type"])
		}
		if thinkingPart["thinking"] != `I need to count the number of "r"s in the word "strawberry".` {
			t.Errorf("expected thinking text, got %v", thinkingPart["thinking"])
		}
		if thinkingPart["signature"] != "test-signature" {
			t.Errorf("expected signature 'test-signature', got %v", thinkingPart["signature"])
		}

		textPart := msg.Content[1].(map[string]any)
		if textPart["type"] != "text" {
			t.Errorf("expected type 'text', got %v", textPart["type"])
		}
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %v", warnings)
		}
	})

	t.Run("should convert reasoning parts without signature into redacted thinking when sendReasoning is true", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ReasoningPart{
							Text: `I need to count the number of "r"s in the word "strawberry".`,
						},
						languagemodel.TextPart{
							Text: `The word "strawberry" has 2 "r"s.`,
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		// Without signature, reasoning becomes redacted_thinking
		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
		}
		redactedPart := msg.Content[0].(map[string]any)
		if redactedPart["type"] != "redacted_thinking" {
			t.Errorf("expected type 'redacted_thinking', got %v", redactedPart["type"])
		}
	})

	t.Run("should omit reasoning parts when sendReasoning is false", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ReasoningPart{
							Text: `I need to count the number of "r"s in the word "strawberry".`,
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"signature": "test-signature",
								},
							},
						},
						languagemodel.TextPart{
							Text: `The word "strawberry" has 2 "r"s.`,
						},
					},
				},
			},
			false,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part (reasoning omitted), got %d", len(msg.Content))
		}
		textPart := msg.Content[0].(map[string]any)
		if textPart["type"] != "text" {
			t.Errorf("expected type 'text', got %v", textPart["type"])
		}
		if textPart["text"] != `The word "strawberry" has 2 "r"s.` {
			t.Errorf("expected text content, got %v", textPart["text"])
		}
	})

	t.Run("should convert assistant message with tool call parts", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "user-content"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							ToolCallID: "test-id",
							ToolName:   "test-tool",
							Input:      map[string]any{"some": "arg"},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[1]
		part := msg.Content[0].(map[string]any)
		if part["type"] != "tool_use" {
			t.Errorf("expected type 'tool_use', got %v", part["type"])
		}
		if part["id"] != "test-id" {
			t.Errorf("expected id 'test-id', got %v", part["id"])
		}
		if part["name"] != "test-tool" {
			t.Errorf("expected name 'test-tool', got %v", part["name"])
		}
		input := part["input"].(map[string]any)
		if input["some"] != "arg" {
			t.Errorf("expected input arg, got %v", input)
		}
	})
}

// --- Cache control ---

func TestConvertPrompt_CacheControl(t *testing.T) {
	t.Run("system message", func(t *testing.T) {
		t.Run("should set cache_control on system message with message cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.SystemMessage{
						Content: "system message",
						ProviderOptions: shared.ProviderOptions{
							"anthropic": map[string]any{
								"cacheControl": map[string]any{"type": "ephemeral"},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			if len(result.Prompt.System) != 1 {
				t.Fatalf("expected 1 system block, got %d", len(result.Prompt.System))
			}
			if result.Prompt.System[0].CacheControl == nil {
				t.Fatal("expected cache_control to be set")
			}
			if result.Prompt.System[0].CacheControl.Type != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %q", result.Prompt.System[0].CacheControl.Type)
			}
		})
	})

	t.Run("user message", func(t *testing.T) {
		t.Run("should set cache_control on user message part with part cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{
								Text: "test",
								ProviderOptions: shared.ProviderOptions{
									"anthropic": map[string]any{
										"cacheControl": map[string]any{"type": "ephemeral"},
									},
								},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			msg := result.Prompt.Messages[0]
			part := msg.Content[0].(map[string]any)
			cc := part["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control to be set")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %v", ccMap["type"])
			}
		})

		t.Run("should set cache_control on last user message part with message cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "part1"},
							languagemodel.TextPart{Text: "part2"},
						},
						ProviderOptions: shared.ProviderOptions{
							"anthropic": map[string]any{
								"cacheControl": map[string]any{"type": "ephemeral"},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			msg := result.Prompt.Messages[0]
			// First part should have nil cache_control
			part1 := msg.Content[0].(map[string]any)
			if part1["cache_control"] != nil {
				t.Errorf("expected nil cache_control on first part, got %v", part1["cache_control"])
			}
			// Last part should have cache_control
			part2 := msg.Content[1].(map[string]any)
			cc := part2["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control on last part")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %v", ccMap["type"])
			}
		})
	})

	t.Run("assistant message", func(t *testing.T) {
		t.Run("should set cache_control on assistant message text part with part cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "user-content"},
						},
					},
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.TextPart{
								Text: "test",
								ProviderOptions: shared.ProviderOptions{
									"anthropic": map[string]any{
										"cacheControl": map[string]any{"type": "ephemeral"},
									},
								},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			assistantMsg := result.Prompt.Messages[1]
			part := assistantMsg.Content[0].(map[string]any)
			cc := part["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control to be set")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %v", ccMap["type"])
			}
		})

		t.Run("should set cache_control on assistant tool call part with part cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "user-content"},
						},
					},
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.ToolCallPart{
								ToolCallID: "test-id",
								ToolName:   "test-tool",
								Input:      map[string]any{"some": "arg"},
								ProviderOptions: shared.ProviderOptions{
									"anthropic": map[string]any{
										"cacheControl": map[string]any{"type": "ephemeral"},
									},
								},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			assistantMsg := result.Prompt.Messages[1]
			part := assistantMsg.Content[0].(map[string]any)
			if part["type"] != "tool_use" {
				t.Errorf("expected type 'tool_use', got %v", part["type"])
			}
			cc := part["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control to be set")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %v", ccMap["type"])
			}
		})

		t.Run("should set cache_control on last assistant message part with message cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.UserMessage{
						Content: []languagemodel.UserMessagePart{
							languagemodel.TextPart{Text: "user-content"},
						},
					},
					languagemodel.AssistantMessage{
						Content: []languagemodel.AssistantMessagePart{
							languagemodel.TextPart{Text: "part1"},
							languagemodel.TextPart{Text: "part2"},
						},
						ProviderOptions: shared.ProviderOptions{
							"anthropic": map[string]any{
								"cacheControl": map[string]any{"type": "ephemeral"},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			assistantMsg := result.Prompt.Messages[1]
			part1 := assistantMsg.Content[0].(map[string]any)
			if part1["cache_control"] != nil {
				t.Errorf("expected nil cache_control on first part, got %v", part1["cache_control"])
			}
			part2 := assistantMsg.Content[1].(map[string]any)
			cc := part2["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control on last part")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected type 'ephemeral', got %v", ccMap["type"])
			}
		})
	})

	t.Run("tool message", func(t *testing.T) {
		t.Run("should set cache_control on tool result message part with part cache control", func(t *testing.T) {
			warnings := []shared.Warning{}
			result := convertToAnthropicMessagesPrompt(
				languagemodel.Prompt{
					languagemodel.ToolMessage{
						Content: []languagemodel.ToolMessagePart{
							languagemodel.ToolResultPart{
								ToolCallID: "test",
								ToolName:   "test",
								Output: languagemodel.ToolResultOutputJSON{
									Value: map[string]any{"test": "test"},
								},
								ProviderOptions: shared.ProviderOptions{
									"anthropic": map[string]any{
										"cacheControl": map[string]any{"type": "ephemeral"},
									},
								},
							},
						},
					},
				},
				true,
				&warnings,
				nil,
			)

			part := result.Prompt.Messages[0].Content[0].(map[string]any)
			cc := part["cache_control"]
			if cc == nil {
				t.Fatal("expected cache_control to be set")
			}
			ccMap := cc.(map[string]any)
			if ccMap["type"] != "ephemeral" {
				t.Errorf("expected cache_control type 'ephemeral', got %v", ccMap["type"])
			}
		})
	})

	t.Run("should limit cache breakpoints to 4", func(t *testing.T) {
		warnings := []shared.Warning{}
		cacheControlValidator := NewCacheControlValidator()
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.SystemMessage{
					Content: "system 1",
					ProviderOptions: shared.ProviderOptions{
						"anthropic": map[string]any{
							"cacheControl": map[string]any{"type": "ephemeral"},
						},
					},
				},
				languagemodel.SystemMessage{
					Content: "system 2",
					ProviderOptions: shared.ProviderOptions{
						"anthropic": map[string]any{
							"cacheControl": map[string]any{"type": "ephemeral"},
						},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{
							Text: "user 1",
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"cacheControl": map[string]any{"type": "ephemeral"},
								},
							},
						},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{
							Text: "assistant 1",
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"cacheControl": map[string]any{"type": "ephemeral"},
								},
							},
						},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{
							Text: "user 2 (should be rejected)",
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"cacheControl": map[string]any{"type": "ephemeral"},
								},
							},
						},
					},
				},
			},
			true,
			&warnings,
			cacheControlValidator,
		)

		// First 4 should have cache_control: system[0], system[1], user 1 message, assistant 1 message
		if result.Prompt.System[0].CacheControl == nil {
			t.Error("expected cache_control on system[0]")
		}
		if result.Prompt.System[1].CacheControl == nil {
			t.Error("expected cache_control on system[1]")
		}

		userMsg1 := result.Prompt.Messages[0].Content[0].(map[string]any)
		if userMsg1["cache_control"] == nil {
			t.Error("expected cache_control on user message 1")
		}

		assistantMsg := result.Prompt.Messages[1].Content[0].(map[string]any)
		if assistantMsg["cache_control"] == nil {
			t.Error("expected cache_control on assistant message")
		}

		// 5th should be rejected
		userMsg2 := result.Prompt.Messages[2].Content[0].(map[string]any)
		if userMsg2["cache_control"] != nil {
			t.Error("expected no cache_control on 5th breakpoint (exceeded limit)")
		}

		// Should have warning about exceeding limit
		validatorWarnings := cacheControlValidator.GetWarnings()
		if len(validatorWarnings) != 1 {
			t.Fatalf("expected 1 warning about limit, got %d", len(validatorWarnings))
		}
	})
}

// --- Citations ---

func TestConvertPrompt_Citations(t *testing.T) {
	t.Run("should not include citations by default", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata"},
							MediaType: "application/pdf",
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "document" {
			t.Errorf("expected type 'document', got %v", part["type"])
		}
		// citations should not be present
		if _, ok := part["citations"]; ok {
			t.Error("expected no citations by default")
		}
	})

	t.Run("should include citations when enabled on file part", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata"},
							MediaType: "application/pdf",
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"citations": map[string]any{"enabled": true},
								},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["type"] != "document" {
			t.Errorf("expected type 'document', got %v", part["type"])
		}
		citations, ok := part["citations"]
		if !ok {
			t.Fatal("expected citations to be present")
		}
		citMap := citations.(map[string]any)
		if citMap["enabled"] != true {
			t.Errorf("expected citations enabled, got %v", citMap["enabled"])
		}
	})

	t.Run("should include custom title and context when provided", func(t *testing.T) {
		warnings := []shared.Warning{}
		filename := "original-name.pdf"
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata"},
							MediaType: "application/pdf",
							Filename:  &filename,
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"title":    "Custom Document Title",
									"context":  "This is metadata about the document",
									"citations": map[string]any{"enabled": true},
								},
							},
						},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		part := result.Prompt.Messages[0].Content[0].(map[string]any)
		if part["title"] != "Custom Document Title" {
			t.Errorf("expected title 'Custom Document Title', got %v", part["title"])
		}
		if part["context"] != "This is metadata about the document" {
			t.Errorf("expected context, got %v", part["context"])
		}
	})

	t.Run("should handle multiple documents with consistent citation settings", func(t *testing.T) {
		warnings := []shared.Warning{}
		filename1 := "doc1.pdf"
		filename2 := "doc2.pdf"
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata1"},
							MediaType: "application/pdf",
							Filename:  &filename1,
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"citations": map[string]any{"enabled": true},
									"title":     "Custom Title 1",
								},
							},
						},
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "base64PDFdata2"},
							MediaType: "application/pdf",
							Filename:  &filename2,
							ProviderOptions: shared.ProviderOptions{
								"anthropic": map[string]any{
									"citations": map[string]any{"enabled": true},
									"title":     "Custom Title 2",
									"context":   "Additional context for document 2",
								},
							},
						},
						languagemodel.TextPart{Text: "Analyze both documents"},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		msg := result.Prompt.Messages[0]
		if len(msg.Content) != 3 {
			t.Fatalf("expected 3 content parts, got %d", len(msg.Content))
		}

		doc1 := msg.Content[0].(map[string]any)
		if doc1["title"] != "Custom Title 1" {
			t.Errorf("expected title 'Custom Title 1', got %v", doc1["title"])
		}

		doc2 := msg.Content[1].(map[string]any)
		if doc2["title"] != "Custom Title 2" {
			t.Errorf("expected title 'Custom Title 2', got %v", doc2["title"])
		}
		if doc2["context"] != "Additional context for document 2" {
			t.Errorf("expected context, got %v", doc2["context"])
		}

		textPart := msg.Content[2].(map[string]any)
		if textPart["text"] != "Analyze both documents" {
			t.Errorf("expected text content, got %v", textPart["text"])
		}
	})
}

// --- Message sequences ---

func TestConvertPrompt_MessageSequences(t *testing.T) {
	t.Run("should convert user-assistant-tool-assistant-user message sequence with multiple tool calls", func(t *testing.T) {
		warnings := []shared.Warning{}
		result := convertToAnthropicMessagesPrompt(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "weather for berlin, london and paris"},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{
							Text: "I will use the weather tool to get the weather for berlin, london and paris",
						},
						languagemodel.ToolCallPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-1",
							Input:      map[string]any{"location": "berlin"},
						},
						languagemodel.ToolCallPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-2",
							Input:      map[string]any{"location": "london"},
						},
						languagemodel.ToolCallPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-3",
							Input:      map[string]any{"location": "paris"},
						},
					},
				},
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-1",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"weather": "sunny"}},
						},
						languagemodel.ToolResultPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-2",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"weather": "cloudy"}},
						},
						languagemodel.ToolResultPart{
							ToolName:   "weather",
							ToolCallID: "weather-call-3",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"weather": "rainy"}},
						},
					},
				},
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{
							Text: "The weather for berlin is sunny, the weather for london is cloudy, and the weather for paris is rainy",
						},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "and for new york?"},
					},
				},
			},
			true,
			&warnings,
			nil,
		)

		// Expected: user, assistant (text+3 tool_use), user (3 tool_results), assistant (text), user (text)
		if len(result.Prompt.Messages) != 5 {
			t.Fatalf("expected 5 messages, got %d", len(result.Prompt.Messages))
		}

		// First message: user
		if result.Prompt.Messages[0].Role != "user" {
			t.Errorf("expected first message role 'user', got %q", result.Prompt.Messages[0].Role)
		}

		// Second message: assistant with text + 3 tool calls
		assistantMsg := result.Prompt.Messages[1]
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected second message role 'assistant', got %q", assistantMsg.Role)
		}
		if len(assistantMsg.Content) != 4 {
			t.Fatalf("expected 4 content parts in assistant, got %d", len(assistantMsg.Content))
		}

		// Third message: user (tool results)
		toolMsg := result.Prompt.Messages[2]
		if toolMsg.Role != "user" {
			t.Errorf("expected third message role 'user', got %q", toolMsg.Role)
		}
		if len(toolMsg.Content) != 3 {
			t.Fatalf("expected 3 tool results, got %d", len(toolMsg.Content))
		}
		for i, expectedID := range []string{"weather-call-1", "weather-call-2", "weather-call-3"} {
			part := toolMsg.Content[i].(map[string]any)
			if part["tool_use_id"] != expectedID {
				t.Errorf("expected tool_use_id %q at index %d, got %v", expectedID, i, part["tool_use_id"])
			}
		}

		// Fourth message: assistant
		if result.Prompt.Messages[3].Role != "assistant" {
			t.Errorf("expected fourth message role 'assistant', got %q", result.Prompt.Messages[3].Role)
		}

		// Fifth message: user
		if result.Prompt.Messages[4].Role != "user" {
			t.Errorf("expected fifth message role 'user', got %q", result.Prompt.Messages[4].Role)
		}
		lastPart := result.Prompt.Messages[4].Content[0].(map[string]any)
		if lastPart["text"] != "and for new york?" {
			t.Errorf("expected text 'and for new york?', got %v", lastPart["text"])
		}
	})
}
