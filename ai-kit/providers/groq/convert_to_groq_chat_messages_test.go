// Ported from: packages/groq/src/convert-to-groq-chat-messages.test.ts
package groq

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToGroqChatMessages_UserMessages(t *testing.T) {
	t.Run("should convert messages with image parts", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
					languagemodel.FilePart{
						Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						MediaType: "image/png",
					},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0]["role"] != "user" {
			t.Errorf("expected role 'user', got %v", result[0]["role"])
		}

		content, ok := result[0]["content"].([]any)
		if !ok {
			t.Fatalf("expected content to be []any, got %T", result[0]["content"])
		}
		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}

		// First part: text
		textPart := content[0].(map[string]any)
		if textPart["type"] != "text" {
			t.Errorf("expected type 'text', got %v", textPart["type"])
		}
		if textPart["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", textPart["text"])
		}

		// Second part: image_url
		imagePart := content[1].(map[string]any)
		if imagePart["type"] != "image_url" {
			t.Errorf("expected type 'image_url', got %v", imagePart["type"])
		}
		imageURL := imagePart["image_url"].(map[string]any)
		if imageURL["url"] != "data:image/png;base64,AAECAw==" {
			t.Errorf("expected url 'data:image/png;base64,AAECAw==', got %v", imageURL["url"])
		}
	})

	t.Run("should convert messages with image parts from byte data", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hi"},
					languagemodel.FilePart{
						Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
						MediaType: "image/png",
					},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		content, ok := result[0]["content"].([]any)
		if !ok {
			t.Fatalf("expected content to be []any, got %T", result[0]["content"])
		}
		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}

		imagePart := content[1].(map[string]any)
		imageURL := imagePart["image_url"].(map[string]any)
		if imageURL["url"] != "data:image/png;base64,AAECAw==" {
			t.Errorf("expected url 'data:image/png;base64,AAECAw==', got %v", imageURL["url"])
		}
	})

	t.Run("should convert messages with only a text part to a string content", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0]["role"] != "user" {
			t.Errorf("expected role 'user', got %v", result[0]["role"])
		}
		if result[0]["content"] != "Hello" {
			t.Errorf("expected content 'Hello', got %v", result[0]["content"])
		}
	})
}

func TestConvertToGroqChatMessages_ToolCalls(t *testing.T) {
	t.Run("should stringify arguments to tool calls", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						Input:      map[string]any{"foo": "bar123"},
						ToolCallID: "quux",
						ToolName:   "thwomp",
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "quux",
						ToolName:   "thwomp",
						Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"oof": "321rab"}},
					},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		// Assistant message
		assistantMsg := result[0]
		if assistantMsg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", assistantMsg["role"])
		}
		if assistantMsg["content"] != "" {
			t.Errorf("expected empty content, got %v", assistantMsg["content"])
		}

		toolCalls, ok := assistantMsg["tool_calls"].([]map[string]any)
		if !ok {
			t.Fatalf("expected tool_calls to be []map[string]any, got %T", assistantMsg["tool_calls"])
		}
		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
		tc := toolCalls[0]
		if tc["id"] != "quux" {
			t.Errorf("expected id 'quux', got %v", tc["id"])
		}
		if tc["type"] != "function" {
			t.Errorf("expected type 'function', got %v", tc["type"])
		}
		fn := tc["function"].(map[string]any)
		if fn["name"] != "thwomp" {
			t.Errorf("expected function name 'thwomp', got %v", fn["name"])
		}
		// Verify arguments is serialized JSON
		var args map[string]any
		if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
			t.Fatalf("failed to parse arguments: %v", err)
		}
		if args["foo"] != "bar123" {
			t.Errorf("expected foo 'bar123', got %v", args["foo"])
		}

		// Tool result message
		toolMsg := result[1]
		if toolMsg["role"] != "tool" {
			t.Errorf("expected role 'tool', got %v", toolMsg["role"])
		}
		if toolMsg["tool_call_id"] != "quux" {
			t.Errorf("expected tool_call_id 'quux', got %v", toolMsg["tool_call_id"])
		}
		// Content should be JSON-serialized tool result
		var toolContent map[string]any
		if err := json.Unmarshal([]byte(toolMsg["content"].(string)), &toolContent); err != nil {
			t.Fatalf("failed to parse tool content: %v", err)
		}
		if toolContent["oof"] != "321rab" {
			t.Errorf("expected oof '321rab', got %v", toolContent["oof"])
		}
	})

	t.Run("should send reasoning if present", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "I think the tool will return the correct value.",
					},
					languagemodel.ToolCallPart{
						Input:      map[string]any{"foo": "bar123"},
						ToolCallID: "quux",
						ToolName:   "thwomp",
					},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		msg := result[0]
		if msg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", msg["role"])
		}
		if msg["reasoning"] != "I think the tool will return the correct value." {
			t.Errorf("expected reasoning text, got %v", msg["reasoning"])
		}

		toolCalls, ok := msg["tool_calls"].([]map[string]any)
		if !ok {
			t.Fatalf("expected tool_calls to be []map[string]any, got %T", msg["tool_calls"])
		}
		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
	})

	t.Run("should not include reasoning field when no reasoning content is present", func(t *testing.T) {
		result := ConvertToGroqChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hello, how can I help you?"},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		msg := result[0]
		if msg["role"] != "assistant" {
			t.Errorf("expected role 'assistant', got %v", msg["role"])
		}
		if msg["content"] != "Hello, how can I help you?" {
			t.Errorf("expected content 'Hello, how can I help you?', got %v", msg["content"])
		}
		if _, hasReasoning := msg["reasoning"]; hasReasoning {
			t.Error("expected no reasoning field in message")
		}
	})
}
