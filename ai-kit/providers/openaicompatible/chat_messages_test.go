// Ported from: packages/openai-compatible/src/chat/convert-to-openai-compatible-chat-messages.test.ts
package openaicompatible

import (
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func strPtr(v string) *string { return &v }

func TestConvertToChatMessages(t *testing.T) {
	t.Run("system message", func(t *testing.T) {
		t.Run("should convert system message", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.SystemMessage{
					Content: "You are a helpful assistant.",
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["role"] != "system" {
				t.Errorf("expected role 'system', got %v", messages[0]["role"])
			}
			if messages[0]["content"] != "You are a helpful assistant." {
				t.Errorf("expected content 'You are a helpful assistant.', got %v", messages[0]["content"])
			}
		})

		t.Run("should merge system message metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.SystemMessage{
					Content: "System prompt",
					ProviderOptions: shared.ProviderOptions{
						"openaiCompatible": {
							"cacheControl": map[string]any{"type": "ephemeral"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["role"] != "system" {
				t.Errorf("expected role 'system', got %v", messages[0]["role"])
			}
			cacheControl, ok := messages[0]["cacheControl"].(map[string]any)
			if !ok {
				t.Fatalf("expected cacheControl to be map, got %T", messages[0]["cacheControl"])
			}
			if cacheControl["type"] != "ephemeral" {
				t.Errorf("expected cacheControl type 'ephemeral', got %v", cacheControl["type"])
			}
		})
	})

	t.Run("user message", func(t *testing.T) {
		t.Run("should flatten single text part to string content", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello, world!"},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["role"] != "user" {
				t.Errorf("expected role 'user', got %v", messages[0]["role"])
			}
			// Single text part should be flattened to string
			if messages[0]["content"] != "Hello, world!" {
				t.Errorf("expected content 'Hello, world!', got %v", messages[0]["content"])
			}
		})

		t.Run("should convert image part with base64 data", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Describe this image"},
						languagemodel.FilePart{
							MediaType: "image/png",
							Data:      languagemodel.DataContentString{Value: "base64-image-data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			contentParts, ok := messages[0]["content"].([]any)
			if !ok {
				t.Fatalf("expected content to be []any, got %T", messages[0]["content"])
			}
			if len(contentParts) != 2 {
				t.Fatalf("expected 2 content parts, got %d", len(contentParts))
			}

			// Text part
			textPart := contentParts[0].(map[string]any)
			if textPart["type"] != "text" {
				t.Errorf("expected type 'text', got %v", textPart["type"])
			}
			if textPart["text"] != "Describe this image" {
				t.Errorf("expected text 'Describe this image', got %v", textPart["text"])
			}

			// Image part
			imagePart := contentParts[1].(map[string]any)
			if imagePart["type"] != "image_url" {
				t.Errorf("expected type 'image_url', got %v", imagePart["type"])
			}
			imageURL := imagePart["image_url"].(map[string]any)
			expectedURL := "data:image/png;base64,base64-image-data"
			if imageURL["url"] != expectedURL {
				t.Errorf("expected URL %q, got %v", expectedURL, imageURL["url"])
			}
		})

		t.Run("should convert image part with URL", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "image/jpeg",
							Data:      languagemodel.DataContentString{Value: "https://example.com/image.jpg"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			imagePart := contentParts[0].(map[string]any)
			imageURL := imagePart["image_url"].(map[string]any)
			if imageURL["url"] != "https://example.com/image.jpg" {
				t.Errorf("expected URL 'https://example.com/image.jpg', got %v", imageURL["url"])
			}
		})

		t.Run("should convert image/* media type to image/jpeg", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "image/*",
							Data:      languagemodel.DataContentString{Value: "base64data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			imagePart := contentParts[0].(map[string]any)
			imageURL := imagePart["image_url"].(map[string]any)
			expected := "data:image/jpeg;base64,base64data"
			if imageURL["url"] != expected {
				t.Errorf("expected URL %q, got %v", expected, imageURL["url"])
			}
		})

		t.Run("should convert audio part with wav format", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/wav",
							Data:      languagemodel.DataContentString{Value: "base64-audio-data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			audioPart := contentParts[0].(map[string]any)
			if audioPart["type"] != "input_audio" {
				t.Errorf("expected type 'input_audio', got %v", audioPart["type"])
			}
			inputAudio := audioPart["input_audio"].(map[string]any)
			if inputAudio["data"] != "base64-audio-data" {
				t.Errorf("expected data 'base64-audio-data', got %v", inputAudio["data"])
			}
			if inputAudio["format"] != "wav" {
				t.Errorf("expected format 'wav', got %v", inputAudio["format"])
			}
		})

		t.Run("should convert audio part with mp3 format", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/mp3",
							Data:      languagemodel.DataContentString{Value: "base64-mp3-data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			audioPart := contentParts[0].(map[string]any)
			inputAudio := audioPart["input_audio"].(map[string]any)
			if inputAudio["format"] != "mp3" {
				t.Errorf("expected format 'mp3', got %v", inputAudio["format"])
			}
		})

		t.Run("should convert audio part with mpeg format to mp3", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "audio/mpeg",
							Data:      languagemodel.DataContentString{Value: "base64-mpeg-data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			audioPart := contentParts[0].(map[string]any)
			inputAudio := audioPart["input_audio"].(map[string]any)
			if inputAudio["format"] != "mp3" {
				t.Errorf("expected format 'mp3', got %v", inputAudio["format"])
			}
		})

		t.Run("should convert PDF file part", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "base64-pdf-data"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			pdfPart := contentParts[0].(map[string]any)
			if pdfPart["type"] != "file" {
				t.Errorf("expected type 'file', got %v", pdfPart["type"])
			}
			file := pdfPart["file"].(map[string]any)
			if file["filename"] != "document.pdf" {
				t.Errorf("expected filename 'document.pdf', got %v", file["filename"])
			}
			expectedData := "data:application/pdf;base64,base64-pdf-data"
			if file["file_data"] != expectedData {
				t.Errorf("expected file_data %q, got %v", expectedData, file["file_data"])
			}
		})

		t.Run("should convert PDF file part with custom filename", func(t *testing.T) {
			filename := "my-document.pdf"
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/pdf",
							Data:      languagemodel.DataContentString{Value: "base64-pdf-data"},
							Filename:  &filename,
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			pdfPart := contentParts[0].(map[string]any)
			file := pdfPart["file"].(map[string]any)
			if file["filename"] != "my-document.pdf" {
				t.Errorf("expected filename 'my-document.pdf', got %v", file["filename"])
			}
		})

		t.Run("should convert text file part to text content", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "text/plain",
							Data:      languagemodel.DataContentBytes{Data: []byte("Hello from a text file")},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			textPart := contentParts[0].(map[string]any)
			if textPart["type"] != "text" {
				t.Errorf("expected type 'text', got %v", textPart["type"])
			}
			if textPart["text"] != "Hello from a text file" {
				t.Errorf("expected text 'Hello from a text file', got %v", textPart["text"])
			}
		})

		t.Run("should panic for unsupported file media type", func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic for unsupported media type")
				}
			}()

			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "application/octet-stream",
							Data:      languagemodel.DataContentString{Value: "some-data"},
						},
					},
				},
			}

			ConvertToChatMessages(prompt)
		})

		t.Run("should merge user message metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
						languagemodel.TextPart{Text: "World"},
					},
					ProviderOptions: shared.ProviderOptions{
						"openaiCompatible": {
							"customField": "custom-value",
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["customField"] != "custom-value" {
				t.Errorf("expected customField 'custom-value', got %v", messages[0]["customField"])
			}
		})

		t.Run("should merge text part metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{
							Text: "Hello",
							ProviderOptions: shared.ProviderOptions{
								"openaiCompatible": {
									"cacheControl": map[string]any{"type": "ephemeral"},
								},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			// Single text part is flattened to string, but metadata is merged at the message level
			cacheControl, ok := messages[0]["cacheControl"].(map[string]any)
			if !ok {
				t.Fatalf("expected cacheControl to be map, got %T", messages[0]["cacheControl"])
			}
			if cacheControl["type"] != "ephemeral" {
				t.Errorf("expected cacheControl type 'ephemeral', got %v", cacheControl["type"])
			}
		})

		t.Run("should merge image part metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.FilePart{
							MediaType: "image/png",
							Data:      languagemodel.DataContentString{Value: "base64data"},
							ProviderOptions: shared.ProviderOptions{
								"openaiCompatible": {
									"imageDetail": "high",
								},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			contentParts := messages[0]["content"].([]any)
			imagePart := contentParts[0].(map[string]any)
			if imagePart["imageDetail"] != "high" {
				t.Errorf("expected imageDetail 'high', got %v", imagePart["imageDetail"])
			}
		})
	})

	t.Run("assistant message", func(t *testing.T) {
		t.Run("should convert assistant text message", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "I can help you with that."},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["role"] != "assistant" {
				t.Errorf("expected role 'assistant', got %v", messages[0]["role"])
			}
			if messages[0]["content"] != "I can help you with that." {
				t.Errorf("expected content 'I can help you with that.', got %v", messages[0]["content"])
			}
		})

		t.Run("should convert assistant message with tool calls", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							ToolCallID: "call-1",
							ToolName:   "get_weather",
							Input:      map[string]any{"city": "San Francisco"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			toolCalls, ok := messages[0]["tool_calls"].([]map[string]any)
			if !ok {
				t.Fatalf("expected tool_calls to be []map[string]any, got %T", messages[0]["tool_calls"])
			}
			if len(toolCalls) != 1 {
				t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
			}
			if toolCalls[0]["id"] != "call-1" {
				t.Errorf("expected id 'call-1', got %v", toolCalls[0]["id"])
			}
			if toolCalls[0]["type"] != "function" {
				t.Errorf("expected type 'function', got %v", toolCalls[0]["type"])
			}
			fn := toolCalls[0]["function"].(map[string]any)
			if fn["name"] != "get_weather" {
				t.Errorf("expected function name 'get_weather', got %v", fn["name"])
			}
			// Arguments should be JSON stringified
			var args map[string]any
			if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
				t.Fatalf("failed to parse arguments: %v", err)
			}
			if args["city"] != "San Francisco" {
				t.Errorf("expected city 'San Francisco', got %v", args["city"])
			}
		})

		t.Run("should include reasoning content", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ReasoningPart{Text: "Let me think about this..."},
						languagemodel.TextPart{Text: "The answer is 42."},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["reasoning_content"] != "Let me think about this..." {
				t.Errorf("expected reasoning_content, got %v", messages[0]["reasoning_content"])
			}
			if messages[0]["content"] != "The answer is 42." {
				t.Errorf("expected content 'The answer is 42.', got %v", messages[0]["content"])
			}
		})

		t.Run("should merge assistant message metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.TextPart{Text: "Response text"},
					},
					ProviderOptions: shared.ProviderOptions{
						"openaiCompatible": {
							"customField": "custom-value",
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["customField"] != "custom-value" {
				t.Errorf("expected customField 'custom-value', got %v", messages[0]["customField"])
			}
		})

		t.Run("should merge tool call metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							ToolCallID: "call-1",
							ToolName:   "test",
							Input:      map[string]any{},
							ProviderOptions: shared.ProviderOptions{
								"openaiCompatible": {
									"customField": "tool-metadata",
								},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			toolCalls := messages[0]["tool_calls"].([]map[string]any)
			if toolCalls[0]["customField"] != "tool-metadata" {
				t.Errorf("expected customField 'tool-metadata', got %v", toolCalls[0]["customField"])
			}
		})

		t.Run("should handle Google Gemini thought signatures", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							ToolCallID: "call-1",
							ToolName:   "test",
							Input:      map[string]any{},
							ProviderOptions: shared.ProviderOptions{
								"google": {
									"thoughtSignature": "test-signature",
								},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			toolCalls := messages[0]["tool_calls"].([]map[string]any)
			extraContent, ok := toolCalls[0]["extra_content"].(map[string]any)
			if !ok {
				t.Fatalf("expected extra_content to be map, got %T", toolCalls[0]["extra_content"])
			}
			google := extraContent["google"].(map[string]any)
			if google["thought_signature"] != "test-signature" {
				t.Errorf("expected thought_signature 'test-signature', got %v", google["thought_signature"])
			}
		})
	})

	t.Run("tool message", func(t *testing.T) {
		t.Run("should convert tool result with text output", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "get_weather",
							Output:     languagemodel.ToolResultOutputText{Value: "Sunny, 72F"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["role"] != "tool" {
				t.Errorf("expected role 'tool', got %v", messages[0]["role"])
			}
			if messages[0]["tool_call_id"] != "call-1" {
				t.Errorf("expected tool_call_id 'call-1', got %v", messages[0]["tool_call_id"])
			}
			if messages[0]["content"] != "Sunny, 72F" {
				t.Errorf("expected content 'Sunny, 72F', got %v", messages[0]["content"])
			}
		})

		t.Run("should convert tool result with JSON output", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "get_weather",
							Output: languagemodel.ToolResultOutputJSON{
								Value: map[string]any{"temp": 72, "condition": "sunny"},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			var parsed map[string]any
			if err := json.Unmarshal([]byte(messages[0]["content"].(string)), &parsed); err != nil {
				t.Fatalf("failed to parse JSON content: %v", err)
			}
			if parsed["condition"] != "sunny" {
				t.Errorf("expected condition 'sunny', got %v", parsed["condition"])
			}
		})

		t.Run("should convert tool result with error text output", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "get_weather",
							Output:     languagemodel.ToolResultOutputErrorText{Value: "City not found"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["content"] != "City not found" {
				t.Errorf("expected content 'City not found', got %v", messages[0]["content"])
			}
		})

		t.Run("should convert tool result with execution denied output", func(t *testing.T) {
			reason := "User denied access"
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "sensitive_tool",
							Output:     languagemodel.ToolResultOutputExecutionDenied{Reason: &reason},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["content"] != "User denied access" {
				t.Errorf("expected content 'User denied access', got %v", messages[0]["content"])
			}
		})

		t.Run("should convert tool result with execution denied without reason", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "sensitive_tool",
							Output:     languagemodel.ToolResultOutputExecutionDenied{},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["content"] != "Tool execution denied." {
				t.Errorf("expected content 'Tool execution denied.', got %v", messages[0]["content"])
			}
		})

		t.Run("should merge tool result metadata from provider options", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "test",
							Output:     languagemodel.ToolResultOutputText{Value: "result"},
							ProviderOptions: shared.ProviderOptions{
								"openaiCompatible": {
									"customField": "tool-result-metadata",
								},
							},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			if messages[0]["customField"] != "tool-result-metadata" {
				t.Errorf("expected customField 'tool-result-metadata', got %v", messages[0]["customField"])
			}
		})

		t.Run("should skip tool approval responses", func(t *testing.T) {
			prompt := languagemodel.Prompt{
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolApprovalResponsePart{},
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "test",
							Output:     languagemodel.ToolResultOutputText{Value: "result"},
						},
					},
				},
			}

			messages := ConvertToChatMessages(prompt)

			// Should only get one message (the tool result, not the approval)
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}
			if messages[0]["content"] != "result" {
				t.Errorf("expected content 'result', got %v", messages[0]["content"])
			}
		})
	})
}
