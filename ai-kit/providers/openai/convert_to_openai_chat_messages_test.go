// Ported from: packages/openai/src/chat/convert-to-openai-chat-messages.test.ts
package openai

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestConvertToOpenAIChatMessages_SystemMessages(t *testing.T) {
	t.Run("should forward system messages", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			},
			"system",
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg, ok := result.Messages[0].(ChatCompletionSystemMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionSystemMessage, got %T", result.Messages[0])
		}
		if msg.Role != "system" {
			t.Errorf("expected role 'system', got %q", msg.Role)
		}
		if msg.Content != "You are a helpful assistant." {
			t.Errorf("expected content 'You are a helpful assistant.', got %q", msg.Content)
		}
	})

	t.Run("should convert system messages to developer messages when requested", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			},
			"developer",
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg, ok := result.Messages[0].(ChatCompletionDeveloperMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionDeveloperMessage, got %T", result.Messages[0])
		}
		if msg.Role != "developer" {
			t.Errorf("expected role 'developer', got %q", msg.Role)
		}
		if msg.Content != "You are a helpful assistant." {
			t.Errorf("expected content 'You are a helpful assistant.', got %q", msg.Content)
		}
	})

	t.Run("should remove system messages when requested", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.SystemMessage{Content: "You are a helpful assistant."},
			},
			"remove",
		)

		if len(result.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(result.Messages))
		}
	})
}

func TestConvertToOpenAIChatMessages_UserMessages(t *testing.T) {
	t.Run("should convert messages with only a text part to string content", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
			},
			"system",
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg, ok := result.Messages[0].(ChatCompletionUserMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionUserMessage, got %T", result.Messages[0])
		}
		if msg.Role != "user" {
			t.Errorf("expected role 'user', got %q", msg.Role)
		}
		// Single text part should be optimized to string
		contentStr, ok := msg.Content.(string)
		if !ok {
			t.Fatalf("expected string content, got %T", msg.Content)
		}
		if contentStr != "Hello" {
			t.Errorf("expected 'Hello', got %q", contentStr)
		}
	})

	t.Run("should convert messages with image parts", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
						languagemodel.FilePart{
							MediaType: "image/png",
							Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
						},
					},
				},
			},
			"system",
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg, ok := result.Messages[0].(ChatCompletionUserMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionUserMessage, got %T", result.Messages[0])
		}
		parts, ok := msg.Content.([]ChatCompletionContentPart)
		if !ok {
			t.Fatalf("expected []ChatCompletionContentPart, got %T", msg.Content)
		}
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}

		textPart, ok := parts[0].(ChatCompletionContentPartText)
		if !ok {
			t.Fatalf("expected ChatCompletionContentPartText, got %T", parts[0])
		}
		if textPart.Text != "Hello" {
			t.Errorf("expected text 'Hello', got %q", textPart.Text)
		}

		imgPart, ok := parts[1].(ChatCompletionContentPartImage)
		if !ok {
			t.Fatalf("expected ChatCompletionContentPartImage, got %T", parts[1])
		}
		if imgPart.Type != "image_url" {
			t.Errorf("expected type 'image_url', got %q", imgPart.Type)
		}
		expectedURL := "data:image/png;base64,AAECAw=="
		if imgPart.ImageURL.URL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, imgPart.ImageURL.URL)
		}
	})

	t.Run("should add image detail when specified through providerOptions", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "image/png",
							Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
							ProviderOptions: shared.ProviderOptions{
								"openai": map[string]any{
									"imageDetail": "low",
								},
							},
						},
					},
				},
			},
			"system",
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}
		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		imgPart := parts[0].(ChatCompletionContentPartImage)
		if imgPart.ImageURL.Detail != "low" {
			t.Errorf("expected detail 'low', got %v", imgPart.ImageURL.Detail)
		}
	})
}

func TestConvertToOpenAIChatMessages_AudioParts(t *testing.T) {
	t.Run("should add audio content for audio/wav file parts", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/wav",
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		audioPart, ok := parts[0].(ChatCompletionContentPartInputAudio)
		if !ok {
			t.Fatalf("expected ChatCompletionContentPartInputAudio, got %T", parts[0])
		}
		if audioPart.Type != "input_audio" {
			t.Errorf("expected type 'input_audio', got %q", audioPart.Type)
		}
		if audioPart.InputAudio.Format != "wav" {
			t.Errorf("expected format 'wav', got %q", audioPart.InputAudio.Format)
		}
	})

	t.Run("should add audio content for audio/mpeg file parts", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/mpeg",
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		audioPart := parts[0].(ChatCompletionContentPartInputAudio)
		if audioPart.InputAudio.Format != "mp3" {
			t.Errorf("expected format 'mp3', got %q", audioPart.InputAudio.Format)
		}
	})

	t.Run("should add audio content for audio/mp3 file parts", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/mp3",
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		audioPart := parts[0].(ChatCompletionContentPartInputAudio)
		if audioPart.InputAudio.Format != "mp3" {
			t.Errorf("expected format 'mp3', got %q", audioPart.InputAudio.Format)
		}
	})
}

func TestConvertToOpenAIChatMessages_FileParts(t *testing.T) {
	t.Run("should throw for unsupported mime types", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic for unsupported mime type")
			}
		}()

		ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/something",
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						},
					},
				},
			},
			"system",
		)
	})

	t.Run("should throw for audio file URLs", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic for audio URL")
			}
		}()

		ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/wav",
							Data:      languagemodel.DataContentString{Value: "https://example.com/foo.wav"},
						},
					},
				},
			},
			"system",
		)
	})

	t.Run("should convert messages with PDF file parts", func(t *testing.T) {
		filename := "document.pdf"
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
							Filename:  &filename,
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		filePart, ok := parts[0].(ChatCompletionContentPartFile)
		if !ok {
			t.Fatalf("expected ChatCompletionContentPartFile, got %T", parts[0])
		}
		if filePart.Type != "file" {
			t.Errorf("expected type 'file', got %q", filePart.Type)
		}
		fileByData, ok := filePart.File.(ChatCompletionFileByData)
		if !ok {
			t.Fatalf("expected ChatCompletionFileByData, got %T", filePart.File)
		}
		if fileByData.Filename != "document.pdf" {
			t.Errorf("expected filename 'document.pdf', got %q", fileByData.Filename)
		}
		expectedData := "data:application/pdf;base64,AQIDBAU="
		if fileByData.FileData != expectedData {
			t.Errorf("expected file_data %q, got %q", expectedData, fileByData.FileData)
		}
	})

	t.Run("should convert messages with PDF file parts using file_id", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "file-pdf-12345"},
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		filePart := parts[0].(ChatCompletionContentPartFile)
		fileByID, ok := filePart.File.(ChatCompletionFileByID)
		if !ok {
			t.Fatalf("expected ChatCompletionFileByID, got %T", filePart.File)
		}
		if fileByID.FileID != "file-pdf-12345" {
			t.Errorf("expected file_id 'file-pdf-12345', got %q", fileByID.FileID)
		}
	})

	t.Run("should use default filename for PDF file parts when not provided", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
						},
					},
				},
			},
			"system",
		)

		msg := result.Messages[0].(ChatCompletionUserMessage)
		parts := msg.Content.([]ChatCompletionContentPart)
		filePart := parts[0].(ChatCompletionContentPartFile)
		fileByData := filePart.File.(ChatCompletionFileByData)
		if fileByData.Filename != "part-0.pdf" {
			t.Errorf("expected default filename 'part-0.pdf', got %q", fileByData.Filename)
		}
	})

	t.Run("should throw error for unsupported file types", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic for unsupported file type")
			}
		}()

		ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "text/plain",
							Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
						},
					},
				},
			},
			"system",
		)
	})

	t.Run("should throw error for PDF file URLs", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Error("expected panic for PDF URL")
			}
		}()

		ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "https://example.com/document.pdf"},
						},
					},
				},
			},
			"system",
		)
	})
}

func TestConvertToOpenAIChatMessages_ToolCalls(t *testing.T) {
	t.Run("should stringify arguments to tool calls", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
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
			},
			"system",
		)

		if len(result.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result.Messages))
		}

		// Assistant message with tool calls
		assistantMsg, ok := result.Messages[0].(ChatCompletionAssistantMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionAssistantMessage, got %T", result.Messages[0])
		}
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}
		tc := assistantMsg.ToolCalls[0]
		if tc.Type != "function" {
			t.Errorf("expected type 'function', got %q", tc.Type)
		}
		if tc.ID != "quux" {
			t.Errorf("expected id 'quux', got %q", tc.ID)
		}
		if tc.Function.Name != "thwomp" {
			t.Errorf("expected function name 'thwomp', got %q", tc.Function.Name)
		}
		expectedArgs, _ := json.Marshal(map[string]any{"foo": "bar123"})
		if tc.Function.Arguments != string(expectedArgs) {
			t.Errorf("expected arguments %q, got %q", string(expectedArgs), tc.Function.Arguments)
		}

		// Tool message with result
		toolMsg, ok := result.Messages[1].(ChatCompletionToolMessage)
		if !ok {
			t.Fatalf("expected ChatCompletionToolMessage, got %T", result.Messages[1])
		}
		if toolMsg.Role != "tool" {
			t.Errorf("expected role 'tool', got %q", toolMsg.Role)
		}
		if toolMsg.ToolCallID != "quux" {
			t.Errorf("expected tool_call_id 'quux', got %q", toolMsg.ToolCallID)
		}
		expectedContent, _ := json.Marshal(map[string]any{"oof": "321rab"})
		if toolMsg.Content != string(expectedContent) {
			t.Errorf("expected content %q, got %q", string(expectedContent), toolMsg.Content)
		}
	})

	t.Run("should handle different tool output types", func(t *testing.T) {
		result := ConvertToOpenAIChatMessages(
			languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "text-tool",
							ToolName:   "text-tool",
							Output:     languagemodel.ToolResultOutputText{Value: "Hello world"},
						},
						languagemodel.ToolResultPart{
							ToolCallID: "error-tool",
							ToolName:   "error-tool",
							Output:     languagemodel.ToolResultOutputErrorText{Value: "Something went wrong"},
						},
					},
				},
			},
			"system",
		)

		if len(result.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result.Messages))
		}

		textToolMsg := result.Messages[0].(ChatCompletionToolMessage)
		if textToolMsg.Content != "Hello world" {
			t.Errorf("expected 'Hello world', got %q", textToolMsg.Content)
		}
		if textToolMsg.ToolCallID != "text-tool" {
			t.Errorf("expected tool_call_id 'text-tool', got %q", textToolMsg.ToolCallID)
		}

		errorToolMsg := result.Messages[1].(ChatCompletionToolMessage)
		if errorToolMsg.Content != "Something went wrong" {
			t.Errorf("expected 'Something went wrong', got %q", errorToolMsg.Content)
		}
		if errorToolMsg.ToolCallID != "error-tool" {
			t.Errorf("expected tool_call_id 'error-tool', got %q", errorToolMsg.ToolCallID)
		}
	})
}
