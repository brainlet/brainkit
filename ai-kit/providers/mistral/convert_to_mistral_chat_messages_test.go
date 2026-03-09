// Ported from: packages/mistral/src/convert-to-mistral-chat-messages.test.ts
package mistral

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToMistralChatMessages_UserMessages(t *testing.T) {
	t.Run("should convert messages with image parts", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
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

		userMsg, ok := result[0].(MistralUserMessage)
		if !ok {
			t.Fatalf("expected MistralUserMessage, got %T", result[0])
		}
		if userMsg.Role != "user" {
			t.Errorf("expected role 'user', got %q", userMsg.Role)
		}
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}

		textPart, ok := userMsg.Content[0].(MistralUserContentText)
		if !ok {
			t.Fatalf("expected MistralUserContentText, got %T", userMsg.Content[0])
		}
		if textPart.Text != "Hello" {
			t.Errorf("expected text 'Hello', got %q", textPart.Text)
		}

		imgPart, ok := userMsg.Content[1].(MistralUserContentImageURL)
		if !ok {
			t.Fatalf("expected MistralUserContentImageURL, got %T", userMsg.Content[1])
		}
		if imgPart.ImageURL != "data:image/png;base64,AAECAw==" {
			t.Errorf("expected image URL 'data:image/png;base64,AAECAw==', got %q", imgPart.ImageURL)
		}
	})

	t.Run("should convert messages with image parts from Uint8Array", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
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

		userMsg, ok := result[0].(MistralUserMessage)
		if !ok {
			t.Fatalf("expected MistralUserMessage, got %T", result[0])
		}
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}

		textPart, ok := userMsg.Content[0].(MistralUserContentText)
		if !ok {
			t.Fatalf("expected MistralUserContentText, got %T", userMsg.Content[0])
		}
		if textPart.Text != "Hi" {
			t.Errorf("expected text 'Hi', got %q", textPart.Text)
		}

		imgPart, ok := userMsg.Content[1].(MistralUserContentImageURL)
		if !ok {
			t.Fatalf("expected MistralUserContentImageURL, got %T", userMsg.Content[1])
		}
		if imgPart.Type != "image_url" {
			t.Errorf("expected type 'image_url', got %q", imgPart.Type)
		}
		if imgPart.ImageURL != "data:image/png;base64,AAECAw==" {
			t.Errorf("expected image URL 'data:image/png;base64,AAECAw==', got %q", imgPart.ImageURL)
		}
	})

	t.Run("should convert messages with PDF file parts using URL", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Please analyze this document"},
					languagemodel.FilePart{
						Data:      languagemodel.DataContentString{Value: "https://example.com/document.pdf"},
						MediaType: "application/pdf",
					},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		userMsg, ok := result[0].(MistralUserMessage)
		if !ok {
			t.Fatalf("expected MistralUserMessage, got %T", result[0])
		}
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}

		docPart, ok := userMsg.Content[1].(MistralUserContentDocumentURL)
		if !ok {
			t.Fatalf("expected MistralUserContentDocumentURL, got %T", userMsg.Content[1])
		}
		if docPart.Type != "document_url" {
			t.Errorf("expected type 'document_url', got %q", docPart.Type)
		}
		if docPart.DocumentURL != "https://example.com/document.pdf" {
			t.Errorf("expected document URL 'https://example.com/document.pdf', got %q", docPart.DocumentURL)
		}
	})

	t.Run("should convert messages with PDF file parts from Uint8Array", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Analyze this PDF"},
					languagemodel.FilePart{
						Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
						MediaType: "application/pdf",
					},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		userMsg, ok := result[0].(MistralUserMessage)
		if !ok {
			t.Fatalf("expected MistralUserMessage, got %T", result[0])
		}
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}

		textPart, ok := userMsg.Content[0].(MistralUserContentText)
		if !ok {
			t.Fatalf("expected MistralUserContentText, got %T", userMsg.Content[0])
		}
		if textPart.Text != "Analyze this PDF" {
			t.Errorf("expected text 'Analyze this PDF', got %q", textPart.Text)
		}

		docPart, ok := userMsg.Content[1].(MistralUserContentDocumentURL)
		if !ok {
			t.Fatalf("expected MistralUserContentDocumentURL, got %T", userMsg.Content[1])
		}
		if docPart.Type != "document_url" {
			t.Errorf("expected type 'document_url', got %q", docPart.Type)
		}
		if docPart.DocumentURL != "data:application/pdf;base64,AAECAw==" {
			t.Errorf("expected document URL 'data:application/pdf;base64,AAECAw==', got %q", docPart.DocumentURL)
		}
	})

	t.Run("should convert messages with reasoning content", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{Text: "Let me think about this..."},
					languagemodel.TextPart{Text: "The answer is 42."},
				},
			},
		})

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		assistantMsg, ok := result[0].(MistralAssistantMessage)
		if !ok {
			t.Fatalf("expected MistralAssistantMessage, got %T", result[0])
		}
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		// Reasoning and text should be concatenated
		if assistantMsg.Content != "Let me think about this...The answer is 42." {
			t.Errorf("expected content 'Let me think about this...The answer is 42.', got %q", assistantMsg.Content)
		}
		// This is the last message, so prefix should be true
		if assistantMsg.Prefix == nil || *assistantMsg.Prefix != true {
			t.Errorf("expected prefix true, got %v", assistantMsg.Prefix)
		}
	})
}

func TestConvertToMistralChatMessages_ToolCalls(t *testing.T) {
	t.Run("should stringify arguments to tool calls", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						Input:      map[string]any{"key": "arg-value"},
						ToolCallID: "tool-call-id-1",
						ToolName:   "tool-1",
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "tool-call-id-1",
						ToolName:   "tool-1",
						Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"key": "result-value"}},
					},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		// First message: assistant with tool calls
		assistantMsg, ok := result[0].(MistralAssistantMessage)
		if !ok {
			t.Fatalf("expected MistralAssistantMessage, got %T", result[0])
		}
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		if assistantMsg.Content != "" {
			t.Errorf("expected empty content, got %q", assistantMsg.Content)
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}
		tc := assistantMsg.ToolCalls[0]
		if tc.ID != "tool-call-id-1" {
			t.Errorf("expected tool call ID 'tool-call-id-1', got %q", tc.ID)
		}
		if tc.Type != "function" {
			t.Errorf("expected type 'function', got %q", tc.Type)
		}
		if tc.Function.Name != "tool-1" {
			t.Errorf("expected function name 'tool-1', got %q", tc.Function.Name)
		}
		expectedArgs := `{"key":"arg-value"}`
		if tc.Function.Arguments != expectedArgs {
			t.Errorf("expected arguments %q, got %q", expectedArgs, tc.Function.Arguments)
		}

		// Second message: tool result
		toolMsg, ok := result[1].(MistralToolMessage)
		if !ok {
			t.Fatalf("expected MistralToolMessage, got %T", result[1])
		}
		if toolMsg.Role != "tool" {
			t.Errorf("expected role 'tool', got %q", toolMsg.Role)
		}
		if toolMsg.Name != "tool-1" {
			t.Errorf("expected name 'tool-1', got %q", toolMsg.Name)
		}
		if toolMsg.ToolCallID != "tool-call-id-1" {
			t.Errorf("expected tool call ID 'tool-call-id-1', got %q", toolMsg.ToolCallID)
		}
		expectedContent := `{"key":"result-value"}`
		if toolMsg.Content != expectedContent {
			t.Errorf("expected content %q, got %q", expectedContent, toolMsg.Content)
		}
	})

	t.Run("should handle text output format", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						Input:      map[string]any{"query": "test"},
						ToolCallID: "tool-call-id-2",
						ToolName:   "text-tool",
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "tool-call-id-2",
						ToolName:   "text-tool",
						Output:     languagemodel.ToolResultOutputText{Value: "This is a text response"},
					},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		toolMsg, ok := result[1].(MistralToolMessage)
		if !ok {
			t.Fatalf("expected MistralToolMessage, got %T", result[1])
		}
		if toolMsg.Content != "This is a text response" {
			t.Errorf("expected content 'This is a text response', got %q", toolMsg.Content)
		}
	})

	t.Run("should handle content output format", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						Input:      map[string]any{"query": "generate image"},
						ToolCallID: "tool-call-id-3",
						ToolName:   "image-tool",
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "tool-call-id-3",
						ToolName:   "image-tool",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentText{Text: "Here is the result:"},
								languagemodel.ToolResultContentImageData{
									Data:      "base64data",
									MediaType: "image/png",
								},
							},
						},
					},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		toolMsg, ok := result[1].(MistralToolMessage)
		if !ok {
			t.Fatalf("expected MistralToolMessage, got %T", result[1])
		}

		// Content should be JSON-serialized content array
		var content []map[string]any
		if err := json.Unmarshal([]byte(toolMsg.Content), &content); err != nil {
			t.Fatalf("expected valid JSON content, got error: %v, content: %q", err, toolMsg.Content)
		}
		if len(content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(content))
		}
	})

	t.Run("should handle error output format", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						Input:      map[string]any{"query": "test"},
						ToolCallID: "tool-call-id-4",
						ToolName:   "error-tool",
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "tool-call-id-4",
						ToolName:   "error-tool",
						Output:     languagemodel.ToolResultOutputErrorText{Value: "Invalid input provided"},
					},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		toolMsg, ok := result[1].(MistralToolMessage)
		if !ok {
			t.Fatalf("expected MistralToolMessage, got %T", result[1])
		}
		if toolMsg.Content != "Invalid input provided" {
			t.Errorf("expected content 'Invalid input provided', got %q", toolMsg.Content)
		}
	})
}

func TestConvertToMistralChatMessages_AssistantMessages(t *testing.T) {
	t.Run("should add prefix true to trailing assistant messages", func(t *testing.T) {
		result := ConvertToMistralChatMessages(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hello!"},
				},
			},
		})

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}

		// First message: user
		userMsg, ok := result[0].(MistralUserMessage)
		if !ok {
			t.Fatalf("expected MistralUserMessage, got %T", result[0])
		}
		if userMsg.Role != "user" {
			t.Errorf("expected role 'user', got %q", userMsg.Role)
		}

		// Second message: assistant with prefix
		assistantMsg, ok := result[1].(MistralAssistantMessage)
		if !ok {
			t.Fatalf("expected MistralAssistantMessage, got %T", result[1])
		}
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		if assistantMsg.Content != "Hello!" {
			t.Errorf("expected content 'Hello!', got %q", assistantMsg.Content)
		}
		if assistantMsg.Prefix == nil || *assistantMsg.Prefix != true {
			t.Errorf("expected prefix true for trailing assistant message, got %v", assistantMsg.Prefix)
		}
	})
}
