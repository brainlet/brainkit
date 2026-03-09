// Ported from: packages/xai/src/convert-to-xai-chat-messages.test.ts
package xai

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToXaiChatMessages_SimpleTextMessage(t *testing.T) {
	t.Run("should convert a simple text user message", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "hello"},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		if msg["role"] != "user" {
			t.Errorf("expected role 'user', got %v", msg["role"])
		}
		if msg["content"] != "hello" {
			t.Errorf("expected content 'hello', got %v", msg["content"])
		}
	})
}

func TestConvertToXaiChatMessages_SystemMessage(t *testing.T) {
	t.Run("should convert a system message", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "you are helpful"},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		if msg["role"] != "system" {
			t.Errorf("expected role 'system', got %v", msg["role"])
		}
		if msg["content"] != "you are helpful" {
			t.Errorf("expected content 'you are helpful', got %v", msg["content"])
		}
	})
}

func TestConvertToXaiChatMessages_AssistantMessage(t *testing.T) {
	t.Run("should convert an assistant message with text", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hi there"},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		if msg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", msg["role"])
		}
		if msg["content"] != "Hi there" {
			t.Errorf("expected content 'Hi there', got %v", msg["content"])
		}
	})
}

func TestConvertToXaiChatMessages_ImagePartURL(t *testing.T) {
	t.Run("should convert user message with image URL", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "describe this image"},
					languagemodel.FilePart{
						MediaType: "image/jpeg",
						Data:      languagemodel.DataContentString{Value: "https://example.com/image.jpg"},
					},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		parts := msg["content"].([]interface{})
		if len(parts) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(parts))
		}

		textPart := parts[0].(map[string]interface{})
		if textPart["type"] != "text" {
			t.Errorf("expected type 'text', got %v", textPart["type"])
		}
		if textPart["text"] != "describe this image" {
			t.Errorf("expected text 'describe this image', got %v", textPart["text"])
		}

		imgPart := parts[1].(map[string]interface{})
		if imgPart["type"] != "image_url" {
			t.Errorf("expected type 'image_url', got %v", imgPart["type"])
		}
		imgURL := imgPart["image_url"].(map[string]interface{})
		if imgURL["url"] != "https://example.com/image.jpg" {
			t.Errorf("expected URL 'https://example.com/image.jpg', got %v", imgURL["url"])
		}
	})
}

func TestConvertToXaiChatMessages_ImagePartBase64(t *testing.T) {
	t.Run("should convert user message with base64 image data", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentBytes{Data: []byte{0x89, 0x50, 0x4E, 0x47}},
					},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		parts := msg["content"].([]interface{})
		if len(parts) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(parts))
		}

		imgPart := parts[0].(map[string]interface{})
		if imgPart["type"] != "image_url" {
			t.Errorf("expected type 'image_url', got %v", imgPart["type"])
		}
		imgURL := imgPart["image_url"].(map[string]interface{})
		url := imgURL["url"].(string)
		if len(url) == 0 {
			t.Error("expected non-empty image URL")
		}
		// Should start with data:image/png;base64,
		expectedPrefix := "data:image/png;base64,"
		if url[:len(expectedPrefix)] != expectedPrefix {
			t.Errorf("expected URL to start with %q, got %q", expectedPrefix, url[:len(expectedPrefix)])
		}
	})
}

func TestConvertToXaiChatMessages_UnsupportedFileType(t *testing.T) {
	t.Run("should panic for unsupported file types", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic for unsupported file type")
			}
		}()

		convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "test"},
					},
				},
			},
		})
	})
}

func TestConvertToXaiChatMessages_ToolCalls(t *testing.T) {
	t.Run("should convert assistant message with tool calls", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Input:      map[string]interface{}{"location": "sf"},
					},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		if msg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", msg["role"])
		}
		toolCalls := msg["tool_calls"].([]interface{})
		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
		tc := toolCalls[0].(map[string]interface{})
		if tc["id"] != "call_123" {
			t.Errorf("expected id 'call_123', got %v", tc["id"])
		}
		if tc["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tc["type"])
		}
		fn := tc["function"].(map[string]interface{})
		if fn["name"] != "weather" {
			t.Errorf("expected function name 'weather', got %v", fn["name"])
		}
	})
}

func TestConvertToXaiChatMessages_ToolResponse(t *testing.T) {
	t.Run("should convert tool result message", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Output:     languagemodel.ToolResultOutputText{Value: "72°F in San Francisco"},
					},
				},
			},
		})

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(map[string]interface{})
		if msg["role"] != "tool" {
			t.Errorf("expected role 'tool', got %v", msg["role"])
		}
		if msg["tool_call_id"] != "call_123" {
			t.Errorf("expected tool_call_id 'call_123', got %v", msg["tool_call_id"])
		}
		if msg["content"] != "72°F in San Francisco" {
			t.Errorf("expected content '72°F in San Francisco', got %v", msg["content"])
		}
	})
}

func TestConvertToXaiChatMessages_MultiTurnConversation(t *testing.T) {
	t.Run("should handle a multi-turn conversation", func(t *testing.T) {
		result := convertToXaiChatMessages(languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "you are helpful"},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "hello"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hi!"},
				},
			},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "how are you?"},
				},
			},
		})

		if len(result.Messages) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(result.Messages))
		}
	})
}
