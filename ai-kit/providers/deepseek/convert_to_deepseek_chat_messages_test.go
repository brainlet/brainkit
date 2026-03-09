// Ported from: packages/deepseek/src/chat/convert-to-deepseek-chat-messages.test.ts
package deepseek

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

func TestConvertToDeepSeekChatMessages_UserMessages(t *testing.T) {
	t.Run("should convert messages with only a text part to a string content", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
			},
			nil, // responseFormat undefined
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}

		userMsg, ok := result.Messages[0].(DeepSeekUserMessage)
		if !ok {
			t.Fatalf("expected DeepSeekUserMessage, got %T", result.Messages[0])
		}
		if userMsg.Role != "user" {
			t.Errorf("expected role 'user', got %q", userMsg.Role)
		}
		if userMsg.Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", userMsg.Content)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should warn about unsupported file parts", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
						languagemodel.FilePart{
							Data:      languagemodel.DataContentString{Value: "AAECAw=="},
							MediaType: "image/png",
						},
					},
				},
			},
			nil,
		)

		if len(result.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result.Messages))
		}

		userMsg, ok := result.Messages[0].(DeepSeekUserMessage)
		if !ok {
			t.Fatalf("expected DeepSeekUserMessage, got %T", result.Messages[0])
		}
		if userMsg.Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", userMsg.Content)
		}

		if len(result.Warnings) != 1 {
			t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
		}

		unsupported, ok := result.Warnings[0].(shared.UnsupportedWarning)
		if !ok {
			t.Fatalf("expected UnsupportedWarning, got %T", result.Warnings[0])
		}
		if unsupported.Feature == "" {
			t.Error("expected non-empty feature in unsupported warning")
		}
	})
}

func TestConvertToDeepSeekChatMessages_ToolCalls(t *testing.T) {
	t.Run("should stringify arguments to tool calls", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
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
			nil,
		)

		if len(result.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result.Messages))
		}

		// Check assistant message
		assistantMsg, ok := result.Messages[0].(DeepSeekAssistantMessage)
		if !ok {
			t.Fatalf("expected DeepSeekAssistantMessage, got %T", result.Messages[0])
		}
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected role 'assistant', got %q", assistantMsg.Role)
		}
		if assistantMsg.ReasoningContent != nil {
			t.Errorf("expected nil reasoning_content, got %v", assistantMsg.ReasoningContent)
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}
		tc := assistantMsg.ToolCalls[0]
		if tc.ID != "quux" {
			t.Errorf("expected tool call id 'quux', got %q", tc.ID)
		}
		if tc.Type != "function" {
			t.Errorf("expected tool call type 'function', got %q", tc.Type)
		}
		if tc.Function.Name != "thwomp" {
			t.Errorf("expected function name 'thwomp', got %q", tc.Function.Name)
		}
		if tc.Function.Arguments != `{"foo":"bar123"}` {
			t.Errorf("expected arguments '{\"foo\":\"bar123\"}', got %q", tc.Function.Arguments)
		}

		// Check tool message
		toolMsg, ok := result.Messages[1].(DeepSeekToolMessage)
		if !ok {
			t.Fatalf("expected DeepSeekToolMessage, got %T", result.Messages[1])
		}
		if toolMsg.Role != "tool" {
			t.Errorf("expected role 'tool', got %q", toolMsg.Role)
		}
		if toolMsg.ToolCallID != "quux" {
			t.Errorf("expected tool_call_id 'quux', got %q", toolMsg.ToolCallID)
		}
		if toolMsg.Content != `{"oof":"321rab"}` {
			t.Errorf("expected content '{\"oof\":\"321rab\"}', got %q", toolMsg.Content)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should handle text output type in tool results", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
			languagemodel.Prompt{
				languagemodel.AssistantMessage{
					Content: []languagemodel.AssistantMessagePart{
						languagemodel.ToolCallPart{
							Input:      map[string]any{"query": "weather"},
							ToolCallID: "call-1",
							ToolName:   "getWeather",
						},
					},
				},
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "call-1",
							ToolName:   "getWeather",
							Output:     languagemodel.ToolResultOutputText{Value: "It is sunny today"},
						},
					},
				},
			},
			nil,
		)

		if len(result.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result.Messages))
		}

		// Check assistant message
		assistantMsg, ok := result.Messages[0].(DeepSeekAssistantMessage)
		if !ok {
			t.Fatalf("expected DeepSeekAssistantMessage, got %T", result.Messages[0])
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}
		tc := assistantMsg.ToolCalls[0]
		if tc.Function.Name != "getWeather" {
			t.Errorf("expected function name 'getWeather', got %q", tc.Function.Name)
		}
		if tc.Function.Arguments != `{"query":"weather"}` {
			t.Errorf("expected arguments '{\"query\":\"weather\"}', got %q", tc.Function.Arguments)
		}

		// Check tool message
		toolMsg, ok := result.Messages[1].(DeepSeekToolMessage)
		if !ok {
			t.Fatalf("expected DeepSeekToolMessage, got %T", result.Messages[1])
		}
		if toolMsg.Content != "It is sunny today" {
			t.Errorf("expected content 'It is sunny today', got %q", toolMsg.Content)
		}
		if toolMsg.ToolCallID != "call-1" {
			t.Errorf("expected tool_call_id 'call-1', got %q", toolMsg.ToolCallID)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should support reasoning content in tool calls", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
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
			nil,
		)

		if len(result.Messages) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result.Messages))
		}

		// Check user message
		userMsg, ok := result.Messages[0].(DeepSeekUserMessage)
		if !ok {
			t.Fatalf("expected DeepSeekUserMessage, got %T", result.Messages[0])
		}
		if userMsg.Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", userMsg.Content)
		}

		// Check assistant message with reasoning (after last user message)
		assistantMsg, ok := result.Messages[1].(DeepSeekAssistantMessage)
		if !ok {
			t.Fatalf("expected DeepSeekAssistantMessage, got %T", result.Messages[1])
		}
		if assistantMsg.ReasoningContent == nil {
			t.Fatal("expected non-nil reasoning_content")
		}
		if *assistantMsg.ReasoningContent != "I think the tool will return the correct value." {
			t.Errorf("expected reasoning content, got %q", *assistantMsg.ReasoningContent)
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}

		// Check tool message
		toolMsg, ok := result.Messages[2].(DeepSeekToolMessage)
		if !ok {
			t.Fatalf("expected DeepSeekToolMessage, got %T", result.Messages[2])
		}
		if toolMsg.Content != `{"oof":"321rab"}` {
			t.Errorf("expected content '{\"oof\":\"321rab\"}', got %q", toolMsg.Content)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})

	t.Run("should filter out reasoning content from turns before the last user message", func(t *testing.T) {
		result := ConvertToDeepSeekChatMessages(
			languagemodel.Prompt{
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Hello"},
					},
				},
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
				languagemodel.ToolMessage{
					Content: []languagemodel.ToolMessagePart{
						languagemodel.ToolResultPart{
							ToolCallID: "quux",
							ToolName:   "thwomp",
							Output:     languagemodel.ToolResultOutputJSON{Value: map[string]any{"oof": "321rab"}},
						},
					},
				},
				languagemodel.UserMessage{
					Content: []languagemodel.UserMessagePart{
						languagemodel.TextPart{Text: "Goodbye"},
					},
				},
			},
			nil,
		)

		if len(result.Messages) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(result.Messages))
		}

		// Check user message
		userMsg, ok := result.Messages[0].(DeepSeekUserMessage)
		if !ok {
			t.Fatalf("expected DeepSeekUserMessage, got %T", result.Messages[0])
		}
		if userMsg.Content != "Hello" {
			t.Errorf("expected content 'Hello', got %q", userMsg.Content)
		}

		// Check assistant message - reasoning should be filtered out (before last user message)
		assistantMsg, ok := result.Messages[1].(DeepSeekAssistantMessage)
		if !ok {
			t.Fatalf("expected DeepSeekAssistantMessage, got %T", result.Messages[1])
		}
		if assistantMsg.ReasoningContent != nil {
			t.Errorf("expected nil reasoning_content (should be filtered), got %q", *assistantMsg.ReasoningContent)
		}
		if len(assistantMsg.ToolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(assistantMsg.ToolCalls))
		}

		// Check tool message
		toolMsg, ok := result.Messages[2].(DeepSeekToolMessage)
		if !ok {
			t.Fatalf("expected DeepSeekToolMessage, got %T", result.Messages[2])
		}
		if toolMsg.Content != `{"oof":"321rab"}` {
			t.Errorf("expected content '{\"oof\":\"321rab\"}', got %q", toolMsg.Content)
		}

		// Check last user message
		lastUserMsg, ok := result.Messages[3].(DeepSeekUserMessage)
		if !ok {
			t.Fatalf("expected DeepSeekUserMessage, got %T", result.Messages[3])
		}
		if lastUserMsg.Content != "Goodbye" {
			t.Errorf("expected content 'Goodbye', got %q", lastUserMsg.Content)
		}

		if len(result.Warnings) != 0 {
			t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		}
	})
}
