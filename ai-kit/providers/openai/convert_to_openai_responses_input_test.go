// Ported from: packages/openai/src/responses/convert-to-openai-responses-input.test.ts
package openai

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// testToolNameMapping is a passthrough mapping that returns names unchanged.
var testToolNameMapping = providerutils.ToolNameMapping{
	ToProviderToolName: func(customToolName string) string { return customToolName },
	ToCustomToolName:   func(providerToolName string) string { return providerToolName },
}

// Helper to convert input items to JSON for comparison.
func inputToJSON(t *testing.T, input OpenAIResponsesInput) []map[string]any {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}
	var result []map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal input: %v", err)
	}
	return result
}

// Helper to get a single input item as JSON map.
func inputItemJSON(t *testing.T, item OpenAIResponsesInputItem) map[string]any {
	t.Helper()
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal item: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal item: %v", err)
	}
	return result
}

func defaultOpts() ConvertToOpenAIResponsesInputOptions {
	return ConvertToOpenAIResponsesInputOptions{
		ToolNameMapping:     testToolNameMapping,
		SystemMessageMode:   "system",
		ProviderOptionsName: "openai",
		Store:               true,
	}
}

func TestConvertToOpenAIResponsesInput_SystemMessages(t *testing.T) {
	t.Run("should convert system messages to system role", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "Hello"},
		}
		opts.SystemMessageMode = "system"

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		msg, ok := result.Input[0].(OpenAIResponsesSystemMessage)
		if !ok {
			t.Fatalf("expected OpenAIResponsesSystemMessage, got %T", result.Input[0])
		}
		if msg.Role != "system" {
			t.Errorf("expected role 'system', got %q", msg.Role)
		}
		if msg.Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", msg.Content)
		}
	})

	t.Run("should convert system messages to developer role", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "Hello"},
		}
		opts.SystemMessageMode = "developer"

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		msg := result.Input[0].(OpenAIResponsesSystemMessage)
		if msg.Role != "developer" {
			t.Errorf("expected role 'developer', got %q", msg.Role)
		}
	})

	t.Run("should remove system messages", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "Hello"},
		}
		opts.SystemMessageMode = "remove"

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 0 {
			t.Errorf("expected 0 input items, got %d", len(result.Input))
		}
	})
}

func TestConvertToOpenAIResponsesInput_UserMessages(t *testing.T) {
	t.Run("should convert text part to input_text", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		msg := result.Input[0].(OpenAIResponsesUserMessage)
		if msg.Role != "user" {
			t.Errorf("expected role 'user', got %q", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(msg.Content))
		}
		part := msg.Content[0].(map[string]any)
		if part["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", part["type"])
		}
		if part["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", part["text"])
		}
	})

	t.Run("should convert image URL to input_image", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
					languagemodel.FilePart{
						MediaType: "image/*",
						Data:      languagemodel.DataContentString{Value: "https://example.com/image.jpg"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		if len(msg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
		}
		imgPart := msg.Content[1].(map[string]any)
		if imgPart["type"] != "input_image" {
			t.Errorf("expected type 'input_image', got %v", imgPart["type"])
		}
		if imgPart["image_url"] != "https://example.com/image.jpg" {
			t.Errorf("expected image_url 'https://example.com/image.jpg', got %v", imgPart["image_url"])
		}
	})

	t.Run("should convert image base64 data to data URL", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "AAECAw=="},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		expected := "data:image/png;base64,AAECAw=="
		if imgPart["image_url"] != expected {
			t.Errorf("expected %q, got %v", expected, imgPart["image_url"])
		}
	})

	t.Run("should convert image bytes to data URL", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentBytes{Data: []byte{0, 1, 2, 3}},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		expected := "data:image/png;base64,AAECAw=="
		if imgPart["image_url"] != expected {
			t.Errorf("expected %q, got %v", expected, imgPart["image_url"])
		}
	})

	t.Run("should convert image with file_id prefix", func(t *testing.T) {
		opts := defaultOpts()
		opts.FileIDPrefixes = []string{"file-"}
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "file-12345"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["file_id"] != "file-12345" {
			t.Errorf("expected file_id 'file-12345', got %v", imgPart["file_id"])
		}
	})

	t.Run("should use default mime type for image/*", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/*",
						Data:      languagemodel.DataContentString{Value: "AAECAw=="},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		expected := "data:image/jpeg;base64,AAECAw=="
		if imgPart["image_url"] != expected {
			t.Errorf("expected %q, got %v", expected, imgPart["image_url"])
		}
	})

	t.Run("should add image detail from provider options", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"imageDetail": "low",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["detail"] != "low" {
			t.Errorf("expected detail 'low', got %v", imgPart["detail"])
		}
	})

	t.Run("should read image detail from azure provider options", func(t *testing.T) {
		opts := defaultOpts()
		opts.ProviderOptionsName = "azure"
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "AAECAw=="},
						ProviderOptions: shared.ProviderOptions{
							"azure": map[string]any{
								"imageDetail": "low",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["detail"] != "low" {
			t.Errorf("expected detail 'low', got %v", imgPart["detail"])
		}
	})

	t.Run("should convert PDF file parts", func(t *testing.T) {
		filename := "document.pdf"
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
						Filename:  &filename,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		filePart := msg.Content[0].(map[string]any)
		if filePart["type"] != "input_file" {
			t.Errorf("expected type 'input_file', got %v", filePart["type"])
		}
		if filePart["filename"] != "document.pdf" {
			t.Errorf("expected filename 'document.pdf', got %v", filePart["filename"])
		}
		if filePart["file_data"] != "data:application/pdf;base64,AQIDBAU=" {
			t.Errorf("expected file_data 'data:application/pdf;base64,AQIDBAU=', got %v", filePart["file_data"])
		}
	})

	t.Run("should convert PDF file parts with file_id", func(t *testing.T) {
		opts := defaultOpts()
		opts.FileIDPrefixes = []string{"file-"}
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "file-pdf-12345"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		filePart := msg.Content[0].(map[string]any)
		if filePart["file_id"] != "file-pdf-12345" {
			t.Errorf("expected file_id 'file-pdf-12345', got %v", filePart["file_id"])
		}
	})

	t.Run("should use default filename for PDF when not provided", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		filePart := msg.Content[0].(map[string]any)
		if filePart["filename"] != "part-0.pdf" {
			t.Errorf("expected filename 'part-0.pdf', got %v", filePart["filename"])
		}
	})

	t.Run("should error for unsupported file types", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "text/plain",
						Data:      languagemodel.DataContentString{Value: "AQIDBAU="},
					},
				},
			},
		}

		_, err := ConvertToOpenAIResponsesInput(opts)
		if err == nil {
			t.Fatal("expected error for unsupported file type")
		}
		if !strings.Contains(err.Error(), "text/plain") {
			t.Errorf("expected error to mention 'text/plain', got %q", err.Error())
		}
	})

	t.Run("should convert PDF URL to input_file with file_url", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "https://example.com/document.pdf"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		filePart := msg.Content[0].(map[string]any)
		if filePart["file_url"] != "https://example.com/document.pdf" {
			t.Errorf("expected file_url 'https://example.com/document.pdf', got %v", filePart["file_url"])
		}
	})

	t.Run("should support Azure file ID prefixes", func(t *testing.T) {
		opts := defaultOpts()
		opts.FileIDPrefixes = []string{"assistant-"}
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "assistant-img-abc123"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["file_id"] != "assistant-img-abc123" {
			t.Errorf("expected file_id 'assistant-img-abc123', got %v", imgPart["file_id"])
		}
	})

	t.Run("should support multiple file ID prefixes", func(t *testing.T) {
		opts := defaultOpts()
		opts.FileIDPrefixes = []string{"assistant-", "file-"}
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "assistant-img-abc123"},
					},
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "file-pdf-xyz789"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["file_id"] != "assistant-img-abc123" {
			t.Errorf("expected file_id 'assistant-img-abc123', got %v", imgPart["file_id"])
		}
		pdfPart := msg.Content[1].(map[string]any)
		if pdfPart["file_id"] != "file-pdf-xyz789" {
			t.Errorf("expected file_id 'file-pdf-xyz789', got %v", pdfPart["file_id"])
		}
	})

	t.Run("should treat data as base64 when fileIdPrefixes is nil", func(t *testing.T) {
		opts := defaultOpts()
		// FileIDPrefixes is nil by default
		filename := "test.pdf"
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "file-12345"},
					},
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentString{Value: "assistant-abc123"},
						Filename:  &filename,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["image_url"] != "data:image/png;base64,file-12345" {
			t.Errorf("expected data URL, got %v", imgPart["image_url"])
		}
		pdfPart := msg.Content[1].(map[string]any)
		if pdfPart["file_data"] != "data:application/pdf;base64,assistant-abc123" {
			t.Errorf("expected data URL, got %v", pdfPart["file_data"])
		}
	})

	t.Run("should handle empty fileIdPrefixes array", func(t *testing.T) {
		opts := defaultOpts()
		opts.FileIDPrefixes = []string{} // Empty array should disable file ID detection
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentString{Value: "file-12345"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesUserMessage)
		imgPart := msg.Content[0].(map[string]any)
		if imgPart["image_url"] != "data:image/png;base64,file-12345" {
			t.Errorf("expected data URL, got %v", imgPart["image_url"])
		}
	})
}

func TestConvertToOpenAIResponsesInput_AssistantMessages(t *testing.T) {
	t.Run("should convert text part to output_text", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		msg := result.Input[0].(OpenAIResponsesAssistantMessage)
		if msg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", msg.Role)
		}
		content := msg.Content[0].(map[string]any)
		if content["type"] != "output_text" {
			t.Errorf("expected type 'output_text', got %v", content["type"])
		}
		if content["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", content["text"])
		}
	})

	t.Run("should include phase from provider options", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{
						Text: "I will search for that",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "msg_001",
								"phase":  "commentary",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesAssistantMessage)
		if msg.ID != "msg_001" {
			t.Errorf("expected ID 'msg_001', got %q", msg.ID)
		}
		if msg.Phase != "commentary" {
			t.Errorf("expected phase 'commentary', got %q", msg.Phase)
		}
	})

	t.Run("should include final_answer phase", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{
						Text: "The capital of France is Paris.",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "msg_002",
								"phase":  "final_answer",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesAssistantMessage)
		if msg.Phase != "final_answer" {
			t.Errorf("expected phase 'final_answer', got %q", msg.Phase)
		}
	})

	t.Run("should not include phase when not set", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{
						Text: "Hello",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "msg_003",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		msg := result.Input[0].(OpenAIResponsesAssistantMessage)
		if msg.ID != "msg_003" {
			t.Errorf("expected ID 'msg_003', got %q", msg.ID)
		}
		if msg.Phase != "" {
			t.Errorf("expected empty phase, got %q", msg.Phase)
		}
	})

	t.Run("should convert tool call parts to function_call", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "I will search for that information."},
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Input:      map[string]any{"query": "weather in San Francisco"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}

		// First item: assistant message with text
		assistMsg := result.Input[0].(OpenAIResponsesAssistantMessage)
		content := assistMsg.Content[0].(map[string]any)
		if content["text"] != "I will search for that information." {
			t.Errorf("unexpected text: %v", content["text"])
		}

		// Second item: function call
		fc := result.Input[1].(OpenAIResponsesFunctionCall)
		if fc.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", fc.Type)
		}
		if fc.CallID != "call_123" {
			t.Errorf("expected call_id 'call_123', got %q", fc.CallID)
		}
		if fc.Name != "search" {
			t.Errorf("expected name 'search', got %q", fc.Name)
		}
		expectedArgs := `{"query":"weather in San Francisco"}`
		if fc.Arguments != expectedArgs {
			t.Errorf("expected arguments %q, got %q", expectedArgs, fc.Arguments)
		}
	})

	t.Run("should convert tool calls with IDs to item_reference when store is true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{
						Text: "I will search for that information.",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "id_123"},
						},
					},
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Input:      map[string]any{"query": "weather in San Francisco"},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "id_456"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		ref1 := result.Input[0].(OpenAIResponsesItemReference)
		if ref1.ID != "id_123" {
			t.Errorf("expected ID 'id_123', got %q", ref1.ID)
		}
		ref2 := result.Input[1].(OpenAIResponsesItemReference)
		if ref2.ID != "id_456" {
			t.Errorf("expected ID 'id_456', got %q", ref2.ID)
		}
	})

	t.Run("should convert multiple tool call parts", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Input:      map[string]any{"query": "weather in San Francisco"},
					},
					languagemodel.ToolCallPart{
						ToolCallID: "call_456",
						ToolName:   "calculator",
						Input:      map[string]any{"expression": "2 + 2"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		fc1 := result.Input[0].(OpenAIResponsesFunctionCall)
		if fc1.Name != "search" {
			t.Errorf("expected name 'search', got %q", fc1.Name)
		}
		fc2 := result.Input[1].(OpenAIResponsesFunctionCall)
		if fc2.Name != "calculator" {
			t.Errorf("expected name 'calculator', got %q", fc2.Name)
		}
	})
}

func TestConvertToOpenAIResponsesInput_ReasoningMessages(t *testing.T) {
	t.Run("should convert single reasoning part with text (store false)", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "Analyzing the problem step by step",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		reasoning := result.Input[0].(OpenAIResponsesReasoning)
		if reasoning.Type != "reasoning" {
			t.Errorf("expected type 'reasoning', got %q", reasoning.Type)
		}
		if reasoning.ID != "reasoning_001" {
			t.Errorf("expected ID 'reasoning_001', got %q", reasoning.ID)
		}
		if len(reasoning.Summary) != 1 {
			t.Fatalf("expected 1 summary part, got %d", len(reasoning.Summary))
		}
		if reasoning.Summary[0].Text != "Analyzing the problem step by step" {
			t.Errorf("expected summary text, got %q", reasoning.Summary[0].Text)
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should convert reasoning with encrypted content", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "Analyzing the problem step by step",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId":                     "reasoning_001",
								"reasoningEncryptedContent": "encrypted_content_001",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reasoning := result.Input[0].(OpenAIResponsesReasoning)
		if reasoning.EncryptedContent == nil || *reasoning.EncryptedContent != "encrypted_content_001" {
			t.Errorf("expected encrypted content 'encrypted_content_001', got %v", reasoning.EncryptedContent)
		}
	})

	t.Run("should create empty summary for initial empty text", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		reasoning := result.Input[0].(OpenAIResponsesReasoning)
		if len(reasoning.Summary) != 0 {
			t.Errorf("expected 0 summary parts, got %d", len(reasoning.Summary))
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should warn when appending empty text to existing sequence", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "First reasoning step",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
					languagemodel.ReasoningPart{
						Text: "",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		warning, ok := result.Warnings[0].(shared.OtherWarning)
		if !ok {
			t.Fatalf("expected OtherWarning, got %T", result.Warnings[0])
		}
		if !strings.Contains(warning.Message, "Cannot append empty reasoning part") {
			t.Errorf("unexpected warning message: %q", warning.Message)
		}
	})

	t.Run("should merge consecutive parts with same reasoning ID", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "First reasoning step",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
					languagemodel.ReasoningPart{
						Text: "Second reasoning step",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId":                     "reasoning_001",
								"reasoningEncryptedContent": "encrypted_content_001",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		// The Go implementation stores a value copy, so we check JSON output
		data, _ := json.Marshal(result.Input[0])
		var reasoning map[string]any
		json.Unmarshal(data, &reasoning)

		if reasoning["id"] != "reasoning_001" {
			t.Errorf("expected ID 'reasoning_001', got %v", reasoning["id"])
		}
		summaries := reasoning["summary"].([]any)
		if len(summaries) != 2 {
			t.Fatalf("expected 2 summary parts, got %d", len(summaries))
		}
		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should create separate messages for different reasoning IDs", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "First reasoning block",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
					languagemodel.ReasoningPart{
						Text: "Second reasoning block",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_002"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		r1 := result.Input[0].(OpenAIResponsesReasoning)
		if r1.ID != "reasoning_001" {
			t.Errorf("expected ID 'reasoning_001', got %q", r1.ID)
		}
		r2 := result.Input[1].(OpenAIResponsesReasoning)
		if r2.ID != "reasoning_002" {
			t.Errorf("expected ID 'reasoning_002', got %q", r2.ID)
		}
	})

	t.Run("should warn when reasoning part has no provider options", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "This is a reasoning part without any provider options",
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 0 {
			t.Errorf("expected 0 input items, got %d", len(result.Input))
		}
		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		warning := result.Warnings[0].(shared.OtherWarning)
		if !strings.Contains(warning.Message, "Non-OpenAI reasoning parts are not supported") {
			t.Errorf("unexpected warning: %q", warning.Message)
		}
	})

	t.Run("should include reasoning without id when encrypted_content is present", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "Thinking through the problem",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"reasoningEncryptedContent": "encrypted_reasoning_blob",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		reasoning := result.Input[0].(OpenAIResponsesReasoning)
		if reasoning.EncryptedContent == nil || *reasoning.EncryptedContent != "encrypted_reasoning_blob" {
			t.Errorf("expected encrypted content 'encrypted_reasoning_blob', got %v", reasoning.EncryptedContent)
		}
		if len(reasoning.Summary) != 1 {
			t.Fatalf("expected 1 summary, got %d", len(reasoning.Summary))
		}
		if reasoning.Summary[0].Text != "Thinking through the problem" {
			t.Errorf("unexpected summary text: %q", reasoning.Summary[0].Text)
		}
	})

	t.Run("should use item_reference for reasoning with store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "First reasoning step (message 1)",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_001"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "reasoning_001" {
			t.Errorf("expected ID 'reasoning_001', got %q", ref.ID)
		}
	})
}

func TestConvertToOpenAIResponsesInput_ToolMessages(t *testing.T) {
	t.Run("should convert tool result with JSON value", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"temperature": "72°F", "condition": "Sunny"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		if fco.Type != "function_call_output" {
			t.Errorf("expected type 'function_call_output', got %q", fco.Type)
		}
		if fco.CallID != "call_123" {
			t.Errorf("expected call_id 'call_123', got %q", fco.CallID)
		}
		// Output should be JSON string
		outputStr, ok := fco.Output.(string)
		if !ok {
			t.Fatalf("expected string output, got %T", fco.Output)
		}
		var parsed map[string]any
		json.Unmarshal([]byte(outputStr), &parsed)
		if parsed["temperature"] != "72°F" {
			t.Errorf("unexpected output: %v", outputStr)
		}
	})

	t.Run("should convert tool result with text value", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputText{
							Value: "The weather in San Francisco is 72°F",
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		if fco.Output != "The weather in San Francisco is 72°F" {
			t.Errorf("unexpected output: %v", fco.Output)
		}
	})

	t.Run("should convert tool result with content parts (text)", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentText{
									Text: "The weather in San Francisco is 72°F",
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		outputParts, ok := fco.Output.([]any)
		if !ok {
			t.Fatalf("expected []any output, got %T", fco.Output)
		}
		if len(outputParts) != 1 {
			t.Fatalf("expected 1 output part, got %d", len(outputParts))
		}
		part := outputParts[0].(map[string]any)
		if part["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", part["type"])
		}
	})

	t.Run("should convert tool result with content parts (image data)", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentImageData{
									MediaType: "image/png",
									Data:      "base64_data",
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		outputParts := fco.Output.([]any)
		part := outputParts[0].(map[string]any)
		if part["type"] != "input_image" {
			t.Errorf("expected type 'input_image', got %v", part["type"])
		}
		if part["image_url"] != "data:image/png;base64,base64_data" {
			t.Errorf("unexpected image_url: %v", part["image_url"])
		}
	})

	t.Run("should convert tool result with content parts (image URL)", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "screenshot",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentImageURL{
									URL: "https://example.com/screenshot.png",
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		outputParts := fco.Output.([]any)
		part := outputParts[0].(map[string]any)
		if part["image_url"] != "https://example.com/screenshot.png" {
			t.Errorf("unexpected image_url: %v", part["image_url"])
		}
	})

	t.Run("should convert tool result with file data (PDF)", func(t *testing.T) {
		filename := "document.pdf"
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentFileData{
									MediaType: "application/pdf",
									Data:      "AQIDBAU=",
									Filename:  &filename,
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		outputParts := fco.Output.([]any)
		part := outputParts[0].(map[string]any)
		if part["type"] != "input_file" {
			t.Errorf("expected type 'input_file', got %v", part["type"])
		}
		if part["filename"] != "document.pdf" {
			t.Errorf("unexpected filename: %v", part["filename"])
		}
		if part["file_data"] != "data:application/pdf;base64,AQIDBAU=" {
			t.Errorf("unexpected file_data: %v", part["file_data"])
		}
	})

	t.Run("should convert mixed content parts", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentText{
									Text: "The weather in San Francisco is 72°F",
								},
								languagemodel.ToolResultContentImageData{
									MediaType: "image/png",
									Data:      "base64_data",
								},
								languagemodel.ToolResultContentFileData{
									MediaType: "application/pdf",
									Data:      "AQIDBAU=",
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fco := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		outputParts := fco.Output.([]any)
		if len(outputParts) != 3 {
			t.Fatalf("expected 3 output parts, got %d", len(outputParts))
		}
		// Check text part
		textPart := outputParts[0].(map[string]any)
		if textPart["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", textPart["type"])
		}
		// Check image part
		imgPart := outputParts[1].(map[string]any)
		if imgPart["type"] != "input_image" {
			t.Errorf("expected type 'input_image', got %v", imgPart["type"])
		}
		// Check file part (no filename => "data")
		filePart := outputParts[2].(map[string]any)
		if filePart["type"] != "input_file" {
			t.Errorf("expected type 'input_file', got %v", filePart["type"])
		}
		if filePart["filename"] != "data" {
			t.Errorf("expected default filename 'data', got %v", filePart["filename"])
		}
	})

	t.Run("should convert multiple tool result parts", func(t *testing.T) {
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "search",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"temperature": "72°F", "condition": "Sunny"},
						},
					},
					languagemodel.ToolResultPart{
						ToolCallID: "call_456",
						ToolName:   "calculator",
						Output: languagemodel.ToolResultOutputJSON{
							Value: float64(4),
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		fco1 := result.Input[0].(OpenAIResponsesFunctionCallOutput)
		if fco1.CallID != "call_123" {
			t.Errorf("expected call_id 'call_123', got %q", fco1.CallID)
		}
		fco2 := result.Input[1].(OpenAIResponsesFunctionCallOutput)
		if fco2.CallID != "call_456" {
			t.Errorf("expected call_id 'call_456', got %q", fco2.CallID)
		}
	})
}

func TestConvertToOpenAIResponsesInput_ProviderDefinedTools(t *testing.T) {
	t.Run("should convert provider-executed tool call to item_reference with store true", func(t *testing.T) {
		providerExecuted := true
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID:       "ci_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f",
						ToolName:         "code_interpreter",
						Input:            map[string]any{"code": "example code", "containerId": "container_123"},
						ProviderExecuted: &providerExecuted,
					},
					languagemodel.ToolResultPart{
						ToolCallID: "ci_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f",
						ToolName:   "code_interpreter",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"outputs": []any{map[string]any{"type": "logs", "logs": "example logs"}},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "ci_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f" {
			t.Errorf("unexpected ID: %q", ref.ID)
		}
	})

	t.Run("should exclude provider-executed tool calls when store is false", func(t *testing.T) {
		providerExecuted := true
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Let me search for recent news from San Francisco."},
					languagemodel.ToolCallPart{
						ToolCallID:       "ws_67cf2b3051e88190b006770db6fdb13d",
						ToolName:         "web_search",
						Input:            map[string]any{"query": "San Francisco major news events June 22 2025"},
						ProviderExecuted: &providerExecuted,
					},
					languagemodel.ToolResultPart{
						ToolCallID: "ws_67cf2b3051e88190b006770db6fdb13d",
						ToolName:   "web_search",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"action": map[string]any{
									"type":  "search",
									"query": "San Francisco major news events June 22 2025",
								},
							},
						},
					},
					languagemodel.TextPart{Text: "Based on the search results, several significant events took place."},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 2 assistant text messages and a warning about the tool result
		assistCount := 0
		for _, item := range result.Input {
			if _, ok := item.(OpenAIResponsesAssistantMessage); ok {
				assistCount++
			}
		}
		if assistCount != 2 {
			t.Errorf("expected 2 assistant messages, got %d", assistCount)
		}
		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}
		warning := result.Warnings[0].(shared.OtherWarning)
		if !strings.Contains(warning.Message, "not sent to the API when store is false") {
			t.Errorf("unexpected warning: %q", warning.Message)
		}
	})
}

func TestConvertToOpenAIResponsesInput_LocalShell(t *testing.T) {
	t.Run("should convert local shell call to item_reference with store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasLocalShellTool = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_XWgeTylovOiS8xLNz2TONOgO",
						ToolName:   "local_shell",
						Input:      map[string]any{"action": map[string]any{"type": "exec", "command": []any{"ls"}}},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "lsh_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f",
							},
						},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_XWgeTylovOiS8xLNz2TONOgO",
						ToolName:   "local_shell",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"output": "example output"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "lsh_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f" {
			t.Errorf("unexpected ID: %q", ref.ID)
		}
		shellOutput := result.Input[1].(OpenAIResponsesLocalShellCallOutput)
		if shellOutput.Type != "local_shell_call_output" {
			t.Errorf("expected type 'local_shell_call_output', got %q", shellOutput.Type)
		}
		if shellOutput.Output != "example output" {
			t.Errorf("expected output 'example output', got %q", shellOutput.Output)
		}
	})

	t.Run("should convert local shell call with store false", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.HasLocalShellTool = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_XWgeTylovOiS8xLNz2TONOgO",
						ToolName:   "local_shell",
						Input:      map[string]any{"action": map[string]any{"type": "exec", "command": []any{"ls"}}},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "lsh_68c2e2cf522c81908f3e2c1bccd1493b0b24aae9c6c01e4f",
							},
						},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_XWgeTylovOiS8xLNz2TONOgO",
						ToolName:   "local_shell",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"output": "example output"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		shellCall := result.Input[0].(OpenAIResponsesLocalShellCall)
		if shellCall.Type != "local_shell_call" {
			t.Errorf("expected type 'local_shell_call', got %q", shellCall.Type)
		}
		shellOutput := result.Input[1].(OpenAIResponsesLocalShellCallOutput)
		if shellOutput.Output != "example output" {
			t.Errorf("expected output 'example output', got %q", shellOutput.Output)
		}
	})
}

func TestConvertToOpenAIResponsesInput_ApplyPatch(t *testing.T) {
	t.Run("should convert apply_patch to item_reference with store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasApplyPatchTool = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_INoksNAffcdh5UmRTWMLk1Ne",
						ToolName:   "apply_patch",
						Input: map[string]any{
							"callId": "call_INoksNAffcdh5UmRTWMLk1Ne",
							"operation": map[string]any{
								"type": "create_file",
								"path": "index.html",
								"diff": "+<!doctype html>\n+<html></html>",
							},
						},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "apc_0d5dfb28a009b1ee0169713022c3f88195a70b253d2a8cf798",
							},
						},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_INoksNAffcdh5UmRTWMLk1Ne",
						ToolName:   "apply_patch",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"status": "completed",
								"output": "Created index.html",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "apc_0d5dfb28a009b1ee0169713022c3f88195a70b253d2a8cf798" {
			t.Errorf("unexpected ID: %q", ref.ID)
		}
		output := result.Input[1].(OpenAIResponsesApplyPatchCallOutput)
		if output.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", output.Status)
		}
		if output.Output != "Created index.html" {
			t.Errorf("expected output 'Created index.html', got %q", output.Output)
		}
	})

	t.Run("should convert apply_patch call with store false", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.HasApplyPatchTool = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_INoksNAffcdh5UmRTWMLk1Ne",
						ToolName:   "apply_patch",
						Input: map[string]any{
							"callId": "call_INoksNAffcdh5UmRTWMLk1Ne",
							"operation": map[string]any{
								"type": "create_file",
								"path": "index.html",
								"diff": "+<!doctype html>\n+<html></html>",
							},
						},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{
								"itemId": "apc_0d5dfb28a009b1ee0169713022c3f88195a70b253d2a8cf798",
							},
						},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_INoksNAffcdh5UmRTWMLk1Ne",
						ToolName:   "apply_patch",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"status": "completed",
								"output": "Created index.html",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		applyPatch := result.Input[0].(OpenAIResponsesApplyPatchCall)
		if applyPatch.Type != "apply_patch_call" {
			t.Errorf("expected type 'apply_patch_call', got %q", applyPatch.Type)
		}
		if applyPatch.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", applyPatch.Status)
		}
		output := result.Input[1].(OpenAIResponsesApplyPatchCallOutput)
		if output.Output != "Created index.html" {
			t.Errorf("expected output 'Created index.html', got %q", output.Output)
		}
	})
}

func TestConvertToOpenAIResponsesInput_ShellToolOutputs(t *testing.T) {
	t.Run("should include shell and apply_patch outputs together", func(t *testing.T) {
		exitCode := 0
		opts := defaultOpts()
		opts.Store = true
		opts.HasShellTool = true
		opts.HasApplyPatchTool = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call-shell",
						ToolName:   "shell",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"output": []any{
									map[string]any{
										"stdout": "hi\n",
										"stderr": "",
										"outcome": map[string]any{
											"type":     "exit",
											"exitCode": float64(exitCode),
										},
									},
								},
							},
						},
					},
					languagemodel.ToolResultPart{
						ToolCallID: "call-apply",
						ToolName:   "apply_patch",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{
								"status": "completed",
								"output": "patched",
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}

		shellOutput := result.Input[0].(OpenAIResponsesShellCallOutput)
		if shellOutput.Type != "shell_call_output" {
			t.Errorf("expected type 'shell_call_output', got %q", shellOutput.Type)
		}
		if len(shellOutput.Output) != 1 {
			t.Fatalf("expected 1 output entry, got %d", len(shellOutput.Output))
		}
		if shellOutput.Output[0].Stdout != "hi\n" {
			t.Errorf("expected stdout 'hi\\n', got %q", shellOutput.Output[0].Stdout)
		}

		patchOutput := result.Input[1].(OpenAIResponsesApplyPatchCallOutput)
		if patchOutput.Output != "patched" {
			t.Errorf("expected output 'patched', got %q", patchOutput.Output)
		}
	})
}

func TestConvertToOpenAIResponsesInput_FunctionTools(t *testing.T) {
	t.Run("should include client-side tool calls in prompt", func(t *testing.T) {
		providerExecuted := false
		opts := defaultOpts()
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID:       "call-1",
						ToolName:         "calculator",
						Input:            map[string]any{"a": float64(1), "b": float64(2)},
						ProviderExecuted: &providerExecuted,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		fc := result.Input[0].(OpenAIResponsesFunctionCall)
		if fc.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", fc.Type)
		}
		if fc.Name != "calculator" {
			t.Errorf("expected name 'calculator', got %q", fc.Name)
		}
	})
}

func TestConvertToOpenAIResponsesInput_MCPApprovalResponses(t *testing.T) {
	t.Run("should convert approved approval response with store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "mcp-approval-123",
						Approved:   true,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "mcp-approval-123" {
			t.Errorf("expected ref ID 'mcp-approval-123', got %q", ref.ID)
		}
		approval := result.Input[1].(OpenAIResponsesMcpApprovalResponse)
		if approval.ApprovalRequestID != "mcp-approval-123" {
			t.Errorf("unexpected approval request ID: %q", approval.ApprovalRequestID)
		}
		if !approval.Approve {
			t.Error("expected approve to be true")
		}
	})

	t.Run("should convert denied approval response with store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "mcp-approval-456",
						Approved:   false,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		approval := result.Input[1].(OpenAIResponsesMcpApprovalResponse)
		if approval.Approve {
			t.Error("expected approve to be false")
		}
	})

	t.Run("should omit item_reference when store is false", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = false
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "mcp-approval-789",
						Approved:   true,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		approval := result.Input[0].(OpenAIResponsesMcpApprovalResponse)
		if approval.ApprovalRequestID != "mcp-approval-789" {
			t.Errorf("unexpected approval request ID: %q", approval.ApprovalRequestID)
		}
	})

	t.Run("should skip duplicate approval IDs", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "duplicate-approval",
						Approved:   true,
					},
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "duplicate-approval",
						Approved:   true,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have 1 item_reference + 1 mcp_approval_response (second duplicate skipped)
		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
	})

	t.Run("should handle multiple different approval responses", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "approval-1",
						Approved:   true,
					},
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "approval-2",
						Approved:   false,
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 2 item_references + 2 mcp_approval_responses
		if len(result.Input) != 4 {
			t.Fatalf("expected 4 input items, got %d", len(result.Input))
		}
	})

	t.Run("should skip execution-denied output with approvalId", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "denied-approval",
						Approved:   false,
					},
					languagemodel.ToolResultPart{
						ToolCallID: "call-123",
						ToolName:   "mcp_tool",
						Output: languagemodel.ToolResultOutputExecutionDenied{
							Reason: strPtr("User denied the tool execution"),
							ProviderOptions: shared.ProviderOptions{
								"openai": map[string]any{
									"approvalId": "denied-approval",
								},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only item_reference + mcp_approval_response, no function_call_output
		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "denied-approval" {
			t.Errorf("unexpected ref ID: %q", ref.ID)
		}
		approval := result.Input[1].(OpenAIResponsesMcpApprovalResponse)
		if approval.Approve {
			t.Error("expected approve to be false")
		}
	})

	t.Run("should handle approval response mixed with regular tool results", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolApprovalResponsePart{
						ApprovalID: "approval-for-mcp",
						Approved:   true,
					},
					languagemodel.ToolResultPart{
						ToolCallID: "regular-call-1",
						ToolName:   "calculator",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"result": float64(42)},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// item_reference + mcp_approval_response + function_call_output
		if len(result.Input) != 3 {
			t.Fatalf("expected 3 input items, got %d", len(result.Input))
		}
	})
}

func TestConvertToOpenAIResponsesInput_HasConversation(t *testing.T) {
	t.Run("should skip assistant text with item IDs when hasConversation is true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasConversation = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{
						Text: "Hi there!",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "msg_existing_123"},
						},
					},
				},
			},
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "What is the weather?"},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only user messages should be present (assistant message with itemId skipped)
		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		msg1 := result.Input[0].(OpenAIResponsesUserMessage)
		content1 := msg1.Content[0].(map[string]any)
		if content1["text"] != "Hello" {
			t.Errorf("expected 'Hello', got %v", content1["text"])
		}
		msg2 := result.Input[1].(OpenAIResponsesUserMessage)
		content2 := msg2.Content[0].(map[string]any)
		if content2["text"] != "What is the weather?" {
			t.Errorf("expected 'What is the weather?', got %v", content2["text"])
		}
	})

	t.Run("should skip tool-call with item ID when hasConversation is true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasConversation = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "What is the weather?"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "getWeather",
						Input:      map[string]any{"location": "San Francisco"},
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "fc_existing_456"},
						},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "getWeather",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"temp": float64(72)},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// User message + function_call_output (tool call with itemId skipped)
		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		_ = result.Input[0].(OpenAIResponsesUserMessage)        // user message
		_ = result.Input[1].(OpenAIResponsesFunctionCallOutput) // tool result stays
	})

	t.Run("should include assistant messages without item IDs when hasConversation is true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasConversation = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "Hi there!"},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 2 {
			t.Fatalf("expected 2 input items, got %d", len(result.Input))
		}
		_ = result.Input[1].(OpenAIResponsesAssistantMessage) // assistant message without itemId included
	})

	t.Run("should skip reasoning with item ID when hasConversation is true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.HasConversation = true
		opts.Prompt = languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Hello"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ReasoningPart{
						Text: "Let me think...",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "reasoning_existing_789"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only user message (reasoning skipped)
		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		_ = result.Input[0].(OpenAIResponsesUserMessage)
	})
}

func TestConvertToOpenAIResponsesInput_CustomToolCalls(t *testing.T) {
	customProviderToolNames := map[string]struct{}{
		"write_sql": {},
	}

	t.Run("should convert custom tool call with string input", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_custom_001",
						ToolName:   "write_sql",
						Input:      "SELECT * FROM users WHERE age > 25",
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input item, got %d", len(result.Input))
		}
		ct := result.Input[0].(OpenAIResponsesCustomToolCall)
		if ct.Type != "custom_tool_call" {
			t.Errorf("expected type 'custom_tool_call', got %q", ct.Type)
		}
		if ct.Name != "write_sql" {
			t.Errorf("expected name 'write_sql', got %q", ct.Name)
		}
		if ct.Input != "SELECT * FROM users WHERE age > 25" {
			t.Errorf("unexpected input: %q", ct.Input)
		}
	})

	t.Run("should JSON.stringify non-string custom tool call input", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_custom_002",
						ToolName:   "write_sql",
						Input:      map[string]any{"query": "test"},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ct := result.Input[0].(OpenAIResponsesCustomToolCall)
		if ct.Input != `{"query":"test"}` {
			t.Errorf("expected JSON string input, got %q", ct.Input)
		}
	})

	t.Run("should convert custom tool call with itemId to item_reference when store true", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_custom_003",
						ToolName:   "write_sql",
						Input:      "SELECT 1",
						ProviderOptions: shared.ProviderOptions{
							"openai": map[string]any{"itemId": "ct_ref_123"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ref := result.Input[0].(OpenAIResponsesItemReference)
		if ref.ID != "ct_ref_123" {
			t.Errorf("expected ID 'ct_ref_123', got %q", ref.ID)
		}
	})

	t.Run("should convert custom tool result with text value", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_custom_001",
						ToolName:   "write_sql",
						Output: languagemodel.ToolResultOutputText{
							Value: "Query executed successfully. 42 rows returned.",
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cto := result.Input[0].(OpenAIResponsesCustomToolCallOutput)
		if cto.Type != "custom_tool_call_output" {
			t.Errorf("expected type 'custom_tool_call_output', got %q", cto.Type)
		}
		if cto.Output != "Query executed successfully. 42 rows returned." {
			t.Errorf("unexpected output: %v", cto.Output)
		}
	})

	t.Run("should convert custom tool result with JSON value", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_custom_002",
						ToolName:   "write_sql",
						Output: languagemodel.ToolResultOutputJSON{
							Value: map[string]any{"rows": float64(42), "status": "ok"},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cto := result.Input[0].(OpenAIResponsesCustomToolCallOutput)
		outputStr, ok := cto.Output.(string)
		if !ok {
			t.Fatalf("expected string output, got %T", cto.Output)
		}
		if !strings.Contains(outputStr, `"rows":42`) {
			t.Errorf("expected JSON output, got %q", outputStr)
		}
	})

	t.Run("should convert aliased tool name to provider custom tool name", func(t *testing.T) {
		aliasMapping := providerutils.ToolNameMapping{
			ToProviderToolName: func(name string) string {
				if name == "alias_name" {
					return "write_sql"
				}
				return name
			},
			ToCustomToolName: func(name string) string {
				if name == "write_sql" {
					return "alias_name"
				}
				return name
			},
		}

		opts := defaultOpts()
		opts.Store = true
		opts.ToolNameMapping = aliasMapping
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_custom_004",
						ToolName:   "alias_name",
						Input:      "SELECT 1",
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ct := result.Input[0].(OpenAIResponsesCustomToolCall)
		if ct.Name != "write_sql" {
			t.Errorf("expected name 'write_sql', got %q", ct.Name)
		}
	})

	t.Run("should convert custom tool result content output", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		opts.CustomProviderToolNames = customProviderToolNames
		opts.Prompt = languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_custom_005",
						ToolName:   "write_sql",
						Output: languagemodel.ToolResultOutputContent{
							Value: []languagemodel.ToolResultContentPart{
								languagemodel.ToolResultContentText{Text: "hello"},
							},
						},
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cto := result.Input[0].(OpenAIResponsesCustomToolCallOutput)
		outputParts, ok := cto.Output.([]any)
		if !ok {
			t.Fatalf("expected []any output, got %T", cto.Output)
		}
		part := outputParts[0].(map[string]any)
		if part["type"] != "input_text" {
			t.Errorf("expected type 'input_text', got %v", part["type"])
		}
		if part["text"] != "hello" {
			t.Errorf("expected text 'hello', got %v", part["text"])
		}
	})

	t.Run("should not emit custom_tool_call when customProviderToolNames is not provided", func(t *testing.T) {
		opts := defaultOpts()
		opts.Store = true
		// No CustomProviderToolNames set
		opts.Prompt = languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_custom_001",
						ToolName:   "write_sql",
						Input:      "SELECT 1",
					},
				},
			},
		}

		result, err := ConvertToOpenAIResponsesInput(opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should be a function_call, not custom_tool_call
		fc := result.Input[0].(OpenAIResponsesFunctionCall)
		if fc.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", fc.Type)
		}
		if fc.Name != "write_sql" {
			t.Errorf("expected name 'write_sql', got %q", fc.Name)
		}
		// String input gets JSON.stringify'd: "SELECT 1" -> `"SELECT 1"`
		if fc.Arguments != `"SELECT 1"` {
			t.Errorf("expected arguments %q, got %q", `"SELECT 1"`, fc.Arguments)
		}
	})
}

// Helper function
func strPtr(s string) *string {
	return &s
}
