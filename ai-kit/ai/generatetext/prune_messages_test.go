// Ported from: packages/ai/src/generate-text/prune-messages.test.ts
package generatetext

import (
	"testing"
)

var messagesFixture1 = []ModelMessage{
	{
		Role: "user",
		Content: []ModelMessageContent{
			{Type: "text", Text: "Weather in Tokyo and Busan?"},
		},
	},
	{
		Role: "assistant",
		Content: []ModelMessageContent{
			{Type: "reasoning", Text: "I need to get the weather in Tokyo and Busan."},
			{Type: "tool-call", ToolCallID: "call-1", ToolName: "get-weather-tool-1", Input: `{"city": "Tokyo"}`},
			{Type: "tool-call", ToolCallID: "call-2", ToolName: "get-weather-tool-2", Input: `{"city": "Busan"}`},
			{Type: "tool-approval-request", ToolCallID: "call-2", ApprovalID: "approval-1"},
		},
	},
	{
		Role: "tool",
		Content: []ModelMessageContent{
			{Type: "tool-approval-response", ApprovalID: "approval-1", Approved: true},
			{Type: "tool-result", ToolCallID: "call-1", ToolName: "get-weather-tool-1", Output: map[string]interface{}{"type": "text", "value": "sunny"}},
			{Type: "tool-result", ToolCallID: "call-2", ToolName: "get-weather-tool-2", Output: map[string]interface{}{"type": "error-text", "value": "Error: Fetching weather data failed"}},
		},
	},
	{
		Role: "assistant",
		Content: []ModelMessageContent{
			{Type: "reasoning", Text: "I have got the weather in Tokyo and Busan."},
			{Type: "text", Text: "The weather in Tokyo is sunny. I could not get the weather in Busan."},
		},
	},
}

var messagesFixture2 = []ModelMessage{
	{
		Role: "user",
		Content: []ModelMessageContent{
			{Type: "text", Text: "Weather in Tokyo and Busan?"},
		},
	},
	{
		Role: "assistant",
		Content: []ModelMessageContent{
			{Type: "reasoning", Text: "I need to get the weather in Tokyo and Busan."},
			{Type: "tool-call", ToolCallID: "call-1", ToolName: "get-weather-tool-1", Input: `{"city": "Tokyo"}`},
			{Type: "tool-call", ToolCallID: "call-2", ToolName: "get-weather-tool-2", Input: `{"city": "Busan"}`},
			{Type: "tool-approval-request", ToolCallID: "call-1", ApprovalID: "approval-1"},
		},
	},
}

var multiTurnToolCallMessagesFixture = []ModelMessage{
	{Role: "user", Content: []ModelMessageContent{{Type: "text", Text: "ask me a question"}}},
	{
		Role: "assistant",
		Content: []ModelMessageContent{
			{Type: "text", Text: "What can i help you with"},
			{Type: "tool-call", ToolCallID: "toolu_01P9s4havAQSjDmS4eWT1N2V", ToolName: "AskUserQuestion",
				Input: map[string]interface{}{"question": "What would you like help with today?"}},
		},
	},
	{
		Role: "tool",
		Content: []ModelMessageContent{
			{Type: "tool-result", ToolCallID: "toolu_01P9s4havAQSjDmS4eWT1N2V", ToolName: "AskUserQuestion",
				Output: map[string]interface{}{"type": "text", "value": "Something else"}},
		},
	},
	{
		Role: "assistant",
		Content: []ModelMessageContent{
			{Type: "tool-call", ToolCallID: "toolu_01TMAuwWKLmBoQtx7K88dxsQ", ToolName: "AskUserQuestion",
				Input: map[string]interface{}{"question": "Ok what else?"}},
		},
	},
	{
		Role: "tool",
		Content: []ModelMessageContent{
			{Type: "tool-result", ToolCallID: "toolu_01TMAuwWKLmBoQtx7K88dxsQ", ToolName: "AskUserQuestion",
				Output: map[string]interface{}{"type": "text", "value": "Other - I'll describe it"}},
		},
	},
	{Role: "assistant", Content: []ModelMessageContent{{Type: "text", Text: "What would you like to discuss or work on?"}}},
	{Role: "user", Content: []ModelMessageContent{{Type: "text", Text: "never mind. lets end this conversation"}}},
	{Role: "assistant", Content: []ModelMessageContent{{Type: "text", Text: "ok, have a nice day"}}},
	{Role: "user", Content: []ModelMessageContent{{Type: "text", Text: "thank you"}}},
}

func getPartTypes(msg ModelMessage) []string {
	parts, ok := msg.Content.([]ModelMessageContent)
	if !ok {
		return nil
	}
	types := make([]string, len(parts))
	for i, p := range parts {
		types[i] = p.Type
	}
	return types
}

func TestPruneMessages_Reasoning_All(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  messagesFixture1,
		Reasoning: "all",
	})

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	// First assistant message should have no reasoning
	types := getPartTypes(result[1])
	for _, tp := range types {
		if tp == "reasoning" {
			t.Error("expected no reasoning parts in first assistant message")
		}
	}

	// Last assistant message should have no reasoning
	types = getPartTypes(result[3])
	for _, tp := range types {
		if tp == "reasoning" {
			t.Error("expected no reasoning parts in last assistant message")
		}
	}
	if len(types) != 1 || types[0] != "text" {
		t.Errorf("expected last assistant to have only text, got %v", types)
	}
}

func TestPruneMessages_Reasoning_BeforeLastMessage(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  messagesFixture1,
		Reasoning: "before-last-message",
	})

	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	// First assistant message should have no reasoning
	types := getPartTypes(result[1])
	for _, tp := range types {
		if tp == "reasoning" {
			t.Error("expected no reasoning parts in first assistant message")
		}
	}

	// Last assistant message SHOULD keep reasoning (it's the last message)
	types = getPartTypes(result[3])
	hasReasoning := false
	for _, tp := range types {
		if tp == "reasoning" {
			hasReasoning = true
		}
	}
	if !hasReasoning {
		t.Error("expected reasoning in last assistant message")
	}
}

func TestPruneMessages_ToolCalls_All(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  messagesFixture1,
		ToolCalls: "all",
	})

	// Tool message should be removed (empty after pruning)
	for _, msg := range result {
		if msg.Role == "tool" {
			t.Error("expected no tool messages after pruning all tool calls")
		}
	}

	// Assistant messages should have no tool-call or tool-approval parts
	for _, msg := range result {
		if msg.Role == "assistant" {
			types := getPartTypes(msg)
			for _, tp := range types {
				if tp == "tool-call" || tp == "tool-approval-request" {
					t.Errorf("expected no tool-call/approval parts, got %q", tp)
				}
			}
		}
	}
}

func TestPruneMessages_ToolCalls_BeforeLastMessage(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  messagesFixture2,
		ToolCalls: "before-last-message",
	})

	// The last message should be kept (it's the assistant message with tool calls and approval)
	if len(result) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(result))
	}

	// Last message should keep its content since it's the last message
	lastMsg := result[len(result)-1]
	if lastMsg.Role != "assistant" {
		t.Errorf("expected last message to be assistant, got %q", lastMsg.Role)
	}
	lastTypes := getPartTypes(lastMsg)
	if len(lastTypes) == 0 {
		t.Error("expected last message to have content")
	}
}

func TestPruneMessages_ToolCalls_BeforeLastMessage_MultiTurn(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  multiTurnToolCallMessagesFixture,
		ToolCalls: "before-last-message",
	})

	// All tool messages should be removed
	for _, msg := range result {
		if msg.Role == "tool" {
			t.Error("expected no tool messages after pruning")
		}
	}

	// Assistant messages before the last should have no tool-call parts
	for i, msg := range result {
		if msg.Role == "assistant" && i < len(result)-1 {
			types := getPartTypes(msg)
			for _, tp := range types {
				if tp == "tool-call" {
					t.Errorf("expected no tool-call parts in assistant message %d", i)
				}
			}
		}
	}
}

func TestPruneMessages_ToolCalls_BeforeLast2Messages(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages:  messagesFixture1,
		ToolCalls: "before-last-2-messages",
	})

	// With 4 messages, keeping last 2 means tool+last-assistant stay
	if len(result) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(result))
	}

	// The tool and last assistant messages should keep their content
	toolMsgFound := false
	for _, msg := range result {
		if msg.Role == "tool" {
			toolMsgFound = true
			types := getPartTypes(msg)
			if len(types) == 0 {
				t.Error("expected tool message to keep content")
			}
		}
	}
	if !toolMsgFound {
		t.Error("expected tool message to be present (within last 2)")
	}
}

func TestPruneMessages_ToolCalls_TwoRules(t *testing.T) {
	result := PruneMessages(PruneMessagesOptions{
		Messages: messagesFixture1,
		ToolCalls: []ToolCallPruneRule{
			{Type: "all", Tools: []string{"get-weather-tool-1"}},
			{Type: "before-last-2-messages", Tools: []string{"get-weather-tool-2"}},
		},
	})

	// get-weather-tool-1 should be completely removed
	for _, msg := range result {
		parts, ok := msg.Content.([]ModelMessageContent)
		if !ok {
			continue
		}
		for _, part := range parts {
			if part.ToolName == "get-weather-tool-1" {
				t.Error("expected get-weather-tool-1 to be removed")
			}
		}
	}
}
