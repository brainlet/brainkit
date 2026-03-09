// Ported from: packages/xai/src/responses/convert-to-xai-responses-input.test.ts
package xai

import (
	"strings"
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

func TestConvertToXaiResponsesInput_SystemMessage(t *testing.T) {
	t.Run("should convert system messages", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.SystemMessage{Content: "you are a helpful assistant"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		sysMsg, ok := result.Input[0].(XaiResponsesSystemMessage)
		if !ok {
			t.Fatalf("expected XaiResponsesSystemMessage, got %T", result.Input[0])
		}
		if sysMsg.Role != "system" {
			t.Errorf("expected role 'system', got %q", sysMsg.Role)
		}
		if sysMsg.Content != "you are a helpful assistant" {
			t.Errorf("expected content 'you are a helpful assistant', got %q", sysMsg.Content)
		}
		if len(result.InputWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.InputWarnings))
		}
	})
}

func TestConvertToXaiResponsesInput_UserMessage(t *testing.T) {
	t.Run("should convert single text part", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "hello"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		userMsg, ok := result.Input[0].(XaiResponsesUserMessage)
		if !ok {
			t.Fatalf("expected XaiResponsesUserMessage, got %T", result.Input[0])
		}
		if userMsg.Role != "user" {
			t.Errorf("expected role 'user', got %q", userMsg.Role)
		}
		if len(userMsg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(userMsg.Content))
		}
		if userMsg.Content[0].Type != "input_text" {
			t.Errorf("expected type 'input_text', got %q", userMsg.Content[0].Type)
		}
		if userMsg.Content[0].Text != "hello" {
			t.Errorf("expected text 'hello', got %q", userMsg.Content[0].Text)
		}
	})

	t.Run("should handle multiple text parts", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "hello "},
					languagemodel.TextPart{Text: "world"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		userMsg := result.Input[0].(XaiResponsesUserMessage)
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}
		if userMsg.Content[0].Text != "hello " {
			t.Errorf("expected 'hello ', got %q", userMsg.Content[0].Text)
		}
		if userMsg.Content[1].Text != "world" {
			t.Errorf("expected 'world', got %q", userMsg.Content[1].Text)
		}
	})

	t.Run("should convert image file parts with URL", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "what is in this image"},
					languagemodel.FilePart{
						MediaType: "image/jpeg",
						Data:      languagemodel.DataContentString{Value: "https://example.com/image.jpg"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		userMsg := result.Input[0].(XaiResponsesUserMessage)
		if len(userMsg.Content) != 2 {
			t.Fatalf("expected 2 content parts, got %d", len(userMsg.Content))
		}
		if userMsg.Content[0].Type != "input_text" {
			t.Errorf("expected type 'input_text', got %q", userMsg.Content[0].Type)
		}
		if userMsg.Content[1].Type != "input_image" {
			t.Errorf("expected type 'input_image', got %q", userMsg.Content[1].Type)
		}
		if userMsg.Content[1].ImageURL != "https://example.com/image.jpg" {
			t.Errorf("expected image URL, got %q", userMsg.Content[1].ImageURL)
		}
		if len(result.InputWarnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.InputWarnings))
		}
	})

	t.Run("should convert image file parts with base64 data", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "describe this"},
					languagemodel.FilePart{
						MediaType: "image/png",
						Data:      languagemodel.DataContentBytes{Data: []byte{1, 2, 3}},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		userMsg := result.Input[0].(XaiResponsesUserMessage)
		if userMsg.Content[1].Type != "input_image" {
			t.Errorf("expected type 'input_image', got %q", userMsg.Content[1].Type)
		}
		if !strings.HasPrefix(userMsg.Content[1].ImageURL, "data:image/png;base64,") {
			t.Errorf("expected data URI, got %q", userMsg.Content[1].ImageURL)
		}
	})

	t.Run("should error for unsupported file types", func(t *testing.T) {
		_, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "check this file"},
					languagemodel.FilePart{
						MediaType: "application/pdf",
						Data:      languagemodel.DataContentBytes{Data: []byte{1, 2, 3}},
					},
				},
			},
		})
		if err == nil {
			t.Fatal("expected error for unsupported file type")
		}
		if !strings.Contains(err.Error(), "application/pdf") {
			t.Errorf("expected error to mention 'application/pdf', got %q", err.Error())
		}
	})
}

func TestConvertToXaiResponsesInput_AssistantMessage(t *testing.T) {
	t.Run("should convert text content", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "hi there"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		asstMsg, ok := result.Input[0].(XaiResponsesAssistantMessage)
		if !ok {
			t.Fatalf("expected XaiResponsesAssistantMessage, got %T", result.Input[0])
		}
		if asstMsg.Content != "hi there" {
			t.Errorf("expected content 'hi there', got %q", asstMsg.Content)
		}
		if asstMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", asstMsg.Role)
		}
	})

	t.Run("should handle client-side tool-call parts", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
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
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		toolCall, ok := result.Input[0].(XaiResponsesToolCallInput)
		if !ok {
			t.Fatalf("expected XaiResponsesToolCallInput, got %T", result.Input[0])
		}
		if toolCall.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", toolCall.Type)
		}
		if toolCall.ID != "call_123" {
			t.Errorf("expected id 'call_123', got %q", toolCall.ID)
		}
		if toolCall.CallID == nil || *toolCall.CallID != "call_123" {
			t.Errorf("expected call_id 'call_123', got %v", toolCall.CallID)
		}
		if toolCall.Name == nil || *toolCall.Name != "weather" {
			t.Errorf("expected name 'weather', got %v", toolCall.Name)
		}
		if toolCall.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", toolCall.Status)
		}
		if toolCall.Arguments == nil || !strings.Contains(*toolCall.Arguments, "location") {
			t.Error("expected arguments to contain location")
		}
	})

	t.Run("should handle client-side tool-call parts named like server-side tools", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_ws",
						ToolName:   "web_search",
						Input:      map[string]interface{}{"query": "latest news"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		toolCall := result.Input[0].(XaiResponsesToolCallInput)
		if toolCall.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", toolCall.Type)
		}
		if toolCall.Name == nil || *toolCall.Name != "web_search" {
			t.Errorf("expected name 'web_search', got %v", toolCall.Name)
		}
	})

	t.Run("should skip server-side tool-call parts", func(t *testing.T) {
		providerExecuted := true
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID:       "ws_123",
						ToolName:         "web_search",
						Input:            map[string]interface{}{},
						ProviderExecuted: &providerExecuted,
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 0 {
			t.Errorf("expected 0 inputs for server-side tool calls, got %d", len(result.Input))
		}
	})
}

func TestConvertToXaiResponsesInput_ToolMessage(t *testing.T) {
	t.Run("should convert tool-result to function_call_output with json", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Output:     languagemodel.ToolResultOutputJSON{Value: map[string]interface{}{"temp": float64(72)}},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 1 {
			t.Fatalf("expected 1 input, got %d", len(result.Input))
		}
		fcOutput, ok := result.Input[0].(XaiResponsesFunctionCallOutput)
		if !ok {
			t.Fatalf("expected XaiResponsesFunctionCallOutput, got %T", result.Input[0])
		}
		if fcOutput.Type != "function_call_output" {
			t.Errorf("expected type 'function_call_output', got %q", fcOutput.Type)
		}
		if fcOutput.CallID != "call_123" {
			t.Errorf("expected call_id 'call_123', got %q", fcOutput.CallID)
		}
		if !strings.Contains(fcOutput.Output, "temp") {
			t.Errorf("expected output to contain 'temp', got %q", fcOutput.Output)
		}
	})

	t.Run("should handle text output", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Output:     languagemodel.ToolResultOutputText{Value: "sunny, 72 degrees"},
					},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		fcOutput := result.Input[0].(XaiResponsesFunctionCallOutput)
		if fcOutput.Output != "sunny, 72 degrees" {
			t.Errorf("expected output 'sunny, 72 degrees', got %q", fcOutput.Output)
		}
	})
}

func TestConvertToXaiResponsesInput_MultiTurn(t *testing.T) {
	t.Run("should handle full conversation with client-side tool calls", func(t *testing.T) {
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "whats the weather"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Input:      map[string]interface{}{"location": "sf"},
					},
				},
			},
			languagemodel.ToolMessage{
				Content: []languagemodel.ToolMessagePart{
					languagemodel.ToolResultPart{
						ToolCallID: "call_123",
						ToolName:   "weather",
						Output:     languagemodel.ToolResultOutputJSON{Value: map[string]interface{}{"temp": float64(72)}},
					},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.TextPart{Text: "its 72 degrees"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Input) != 4 {
			t.Fatalf("expected 4 inputs, got %d", len(result.Input))
		}

		// First should be user message
		_, ok := result.Input[0].(XaiResponsesUserMessage)
		if !ok {
			t.Errorf("expected XaiResponsesUserMessage, got %T", result.Input[0])
		}

		// Second should be function_call
		toolCall, ok := result.Input[1].(XaiResponsesToolCallInput)
		if !ok {
			t.Fatalf("expected XaiResponsesToolCallInput, got %T", result.Input[1])
		}
		if toolCall.Type != "function_call" {
			t.Errorf("expected type 'function_call', got %q", toolCall.Type)
		}

		// Third should be function_call_output
		fcOutput, ok := result.Input[2].(XaiResponsesFunctionCallOutput)
		if !ok {
			t.Fatalf("expected XaiResponsesFunctionCallOutput, got %T", result.Input[2])
		}
		if fcOutput.Type != "function_call_output" {
			t.Errorf("expected type 'function_call_output', got %q", fcOutput.Type)
		}

		// Fourth should be assistant message
		asstMsg, ok := result.Input[3].(XaiResponsesAssistantMessage)
		if !ok {
			t.Fatalf("expected XaiResponsesAssistantMessage, got %T", result.Input[3])
		}
		if asstMsg.Content != "its 72 degrees" {
			t.Errorf("expected content 'its 72 degrees', got %q", asstMsg.Content)
		}
	})

	t.Run("should handle conversation with server-side tool calls skipped", func(t *testing.T) {
		providerExecuted := true
		result, err := convertToXaiResponsesInput(languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "search for ai news"},
				},
			},
			languagemodel.AssistantMessage{
				Content: []languagemodel.AssistantMessagePart{
					languagemodel.ToolCallPart{
						ToolCallID:       "ws_123",
						ToolName:         "web_search",
						Input:            map[string]interface{}{},
						ProviderExecuted: &providerExecuted,
					},
					languagemodel.TextPart{Text: "here are the results"},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should only have 2 items: user message + assistant message (tool call skipped)
		if len(result.Input) != 2 {
			t.Fatalf("expected 2 inputs, got %d", len(result.Input))
		}

		_, ok := result.Input[0].(XaiResponsesUserMessage)
		if !ok {
			t.Errorf("expected XaiResponsesUserMessage, got %T", result.Input[0])
		}

		asstMsg, ok := result.Input[1].(XaiResponsesAssistantMessage)
		if !ok {
			t.Fatalf("expected XaiResponsesAssistantMessage, got %T", result.Input[1])
		}
		if asstMsg.Content != "here are the results" {
			t.Errorf("expected content 'here are the results', got %q", asstMsg.Content)
		}
	})
}
