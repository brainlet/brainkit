// Ported from: packages/perplexity/src/convert-to-perplexity-messages.test.ts
package perplexity

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToPerplexityMessages_SystemMessages(t *testing.T) {
	t.Run("should convert a system message with text content", func(t *testing.T) {
		result, err := ConvertToPerplexityMessages(languagemodel.Prompt{
			languagemodel.SystemMessage{
				Content: "System initialization",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "system" {
			t.Errorf("expected role 'system', got %q", result[0].Role)
		}
		contentStr, ok := result[0].Content.(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T", result[0].Content)
		}
		if contentStr != "System initialization" {
			t.Errorf("expected content 'System initialization', got %q", contentStr)
		}
	})
}

func TestConvertToPerplexityMessages_UserMessages(t *testing.T) {
	t.Run("should convert a user message with text parts", func(t *testing.T) {
		result, err := ConvertToPerplexityMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello "},
					languagemodel.TextPart{Text: "World"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "user" {
			t.Errorf("expected role 'user', got %q", result[0].Role)
		}
		// Text-only messages should be joined into a single string
		contentStr, ok := result[0].Content.(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T", result[0].Content)
		}
		if contentStr != "Hello World" {
			t.Errorf("expected content 'Hello World', got %q", contentStr)
		}
	})

	t.Run("should convert a user message with image parts", func(t *testing.T) {
		result, err := ConvertToPerplexityMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello "},
					languagemodel.FilePart{
						Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
						MediaType: "image/png",
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "user" {
			t.Errorf("expected role 'user', got %q", result[0].Role)
		}

		// With image parts, content should be multipart ([]any)
		parts, ok := result[0].Content.([]any)
		if !ok {
			t.Fatalf("expected content to be []any, got %T", result[0].Content)
		}
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}

		// First part should be text
		textPart, ok := parts[0].(PerplexityTextContent)
		if !ok {
			t.Fatalf("expected first part to be PerplexityTextContent, got %T", parts[0])
		}
		if textPart.Type != "text" {
			t.Errorf("expected type 'text', got %q", textPart.Type)
		}
		if textPart.Text != "Hello " {
			t.Errorf("expected text 'Hello ', got %q", textPart.Text)
		}

		// Second part should be image_url
		imgPart, ok := parts[1].(PerplexityImageURLContent)
		if !ok {
			t.Fatalf("expected second part to be PerplexityImageURLContent, got %T", parts[1])
		}
		if imgPart.Type != "image_url" {
			t.Errorf("expected type 'image_url', got %q", imgPart.Type)
		}
	})
}

func TestConvertToPerplexityMessages_AssistantMessages(t *testing.T) {
	t.Run("should convert an assistant message with text content", func(t *testing.T) {
		result, err := ConvertToPerplexityMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Assistant reply"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", result[0].Role)
		}
		contentStr, ok := result[0].Content.(string)
		if !ok {
			t.Fatalf("expected content to be string, got %T", result[0].Content)
		}
		if contentStr != "Assistant reply" {
			t.Errorf("expected content 'Assistant reply', got %q", contentStr)
		}
	})
}

func TestConvertToPerplexityMessages_ToolMessages(t *testing.T) {
	t.Run("should throw an error for tool messages", func(t *testing.T) {
		_, err := ConvertToPerplexityMessages(languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "dummy-tool-call-id",
						ToolName:   "dummy-tool-name",
						Output: languagemodel.ToolResultOutputText{
							Value: "This should fail",
						},
					},
				},
			},
		})
		if err == nil {
			t.Fatal("expected error for tool messages, got nil")
		}
		if !errors.IsUnsupportedFunctionalityError(err) {
			t.Errorf("expected UnsupportedFunctionalityError, got %T: %v", err, err)
		}
	})
}
