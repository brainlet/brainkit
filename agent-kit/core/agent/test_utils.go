// Ported from: packages/core/src/agent/test-utils.ts
package agent

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test tool data
// ---------------------------------------------------------------------------

var toolArgs = map[string]map[string]any{
	"weather":    {"location": "New York"},
	"calculator": {"expression": "2+2"},
	"search":     {"query": "latest AI developments"},
}

var toolResults = map[string]string{
	"weather":    "Pretty hot",
	"calculator": "4",
	"search":     "Anthropic blah blah blah",
}

// ---------------------------------------------------------------------------
// ConversationHistoryResult
// ---------------------------------------------------------------------------

// ConversationHistoryResult holds the output of GenerateConversationHistory.
type ConversationHistoryResult struct {
	// Messages is the v1 representation (CoreMessage format).
	Messages []CoreMessage `json:"messages"`
	// MessagesV2 is the v2 representation (MastraDBMessage format).
	MessagesV2 []MastraDBMessage `json:"messagesV2"`
	// FakeCore is the v1 representation cast as CoreMessage slice.
	FakeCore []CoreMessage `json:"fakeCore"`
	// Counts tracks how many messages, tool calls, and tool results were generated.
	Counts ConversationCounts `json:"counts"`
}

// ConversationCounts tracks message/tool counts.
type ConversationCounts struct {
	Messages    int `json:"messages"`
	ToolCalls   int `json:"toolCalls"`
	ToolResults int `json:"toolResults"`
}

// ---------------------------------------------------------------------------
// GenerateConversationHistoryParams
// ---------------------------------------------------------------------------

// GenerateConversationHistoryParams configures conversation history generation.
type GenerateConversationHistoryParams struct {
	// ThreadID is the thread ID for the messages.
	ThreadID string
	// ResourceID defaults to "test-resource".
	ResourceID string
	// MessageCount is the number of turn pairs (user + assistant) to generate. Default: 5.
	MessageCount int
	// ToolFrequency controls how often to include tool calls.
	// E.g., 3 means every 3rd assistant message. Default: 3.
	ToolFrequency int
	// ToolNames lists which tools to cycle through. Default: ["weather", "calculator", "search"].
	ToolNames []string
}

// ---------------------------------------------------------------------------
// GenerateConversationHistory
// ---------------------------------------------------------------------------

// GenerateConversationHistory creates a simulated conversation history with
// alternating user/assistant messages and occasional tool calls.
func GenerateConversationHistory(params GenerateConversationHistoryParams) ConversationHistoryResult {
	// Apply defaults.
	if params.ResourceID == "" {
		params.ResourceID = "test-resource"
	}
	if params.MessageCount <= 0 {
		params.MessageCount = 5
	}
	if params.ToolFrequency <= 0 {
		params.ToolFrequency = 3
	}
	if len(params.ToolNames) == 0 {
		params.ToolNames = []string{"weather", "calculator", "search"}
	}

	counts := ConversationCounts{}
	words := []string{"apple", "banana", "orange", "grape"}

	var messages []MastraDBMessage
	startTime := time.Now()

	for i := 0; i < params.MessageCount; i++ {
		// User message content (~100 tokens).
		userContent := repeatAndJoin(words, 25)

		messages = append(messages, MastraDBMessage{
			ID:         fmt.Sprintf("message-%d", i*2),
			Role:       "user",
			Content:    MastraMessageContentV2{Format: 2, Parts: []MastraMessagePart{{Type: "text", Text: userContent}}},
			ThreadID:   params.ThreadID,
			ResourceID: params.ResourceID,
			CreatedAt:  startTime.Add(time.Duration(i*2) * time.Second),
		})
		counts.Messages++

		// Determine if this assistant message should include a tool call.
		includeTool := i > 0 && i%params.ToolFrequency == 0
		toolIndex := -1
		toolName := ""
		if includeTool {
			toolIndex = (i / params.ToolFrequency) % len(params.ToolNames)
			toolName = params.ToolNames[toolIndex]
		}

		if includeTool && toolName != "" {
			args := toolArgs[toolName]
			if args == nil {
				args = map[string]any{}
			}
			result := toolResults[toolName]

			messages = append(messages, MastraDBMessage{
				ID:   fmt.Sprintf("tool-call-%d", i*2+1),
				Role: "assistant",
				Content: MastraMessageContentV2{
					Format: 2,
					Parts: []MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &ToolInvocation{
								State:      "result",
								ToolCallID: fmt.Sprintf("tool-%d", i),
								ToolName:   toolName,
								Args:       args,
								Result:     result,
							},
						},
					},
				},
				ThreadID:   params.ThreadID,
				ResourceID: params.ResourceID,
				CreatedAt:  startTime.Add(time.Duration(i*2)*time.Second + time.Second),
			})
			counts.Messages++
			counts.ToolCalls++
			counts.ToolResults++
		} else {
			// Regular assistant text message (~60 tokens).
			assistantContent := repeatAndJoin(words, 15)
			messages = append(messages, MastraDBMessage{
				ID:         fmt.Sprintf("message-%d", i*2+1),
				Role:       "assistant",
				Content:    MastraMessageContentV2{Format: 2, Parts: []MastraMessagePart{{Type: "text", Text: assistantContent}}},
				ThreadID:   params.ThreadID,
				ResourceID: params.ResourceID,
				CreatedAt:  startTime.Add(time.Duration(i*2)*time.Second + time.Second),
			})
			counts.Messages++
		}
	}

	// If the last message is an assistant tool invocation, append one more user message
	// so that the conversation ends on a user turn.
	if len(messages) > 0 {
		last := messages[len(messages)-1]
		if last.Role == "assistant" && len(last.Content.Parts) > 0 &&
			last.Content.Parts[len(last.Content.Parts)-1].Type == "tool-invocation" {

			userContent := repeatAndJoin(words, 25)
			messages = append(messages, MastraDBMessage{
				ID:         fmt.Sprintf("message-%d", (len(messages)+1)*2),
				Role:       "user",
				Content:    MastraMessageContentV2{Format: 2, Parts: []MastraMessagePart{{Type: "text", Text: userContent}}},
				ThreadID:   params.ThreadID,
				ResourceID: params.ResourceID,
				CreatedAt:  startTime.Add(time.Duration((len(messages)+1)*2) * time.Second),
			})
			counts.Messages++
		}
	}

	// In the TypeScript version, messages are added to a MessageList and then
	// extracted in different formats. Here we return them directly.
	// TODO: Use real MessageList once ported to convert between formats.

	// Build v1 (CoreMessage) representation from the v2 messages.
	var coreMessages []CoreMessage
	for _, m := range messages {
		cm := CoreMessage{Role: m.Role}
		if len(m.Content.Parts) > 0 && m.Content.Parts[0].Type == "text" {
			cm.Content = m.Content.Parts[0].Text
		}
		coreMessages = append(coreMessages, cm)
	}

	return ConversationHistoryResult{
		Messages:   coreMessages,
		MessagesV2: messages,
		FakeCore:   coreMessages,
		Counts:     counts,
	}
}

// ---------------------------------------------------------------------------
// AssertNoDuplicateParts
// ---------------------------------------------------------------------------

// AssertNoDuplicateParts asserts that no duplicate tool-invocation results or
// text parts exist in the given slice.
func AssertNoDuplicateParts(t *testing.T, parts []MastraMessagePart) {
	t.Helper()

	seenToolResults := make(map[string]bool)
	for _, part := range parts {
		if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
			part.ToolInvocation.State == "result" {

			key := fmt.Sprintf("%s|%v", part.ToolInvocation.ToolCallID, part.ToolInvocation.Result)
			if seenToolResults[key] {
				t.Errorf("duplicate tool-invocation result found: %s", key)
			}
			seenToolResults[key] = true
		}
	}

	seenTexts := make(map[string]bool)
	for _, part := range parts {
		if part.Type == "text" {
			if seenTexts[part.Text] {
				t.Errorf("duplicate text part found: %q", part.Text)
			}
			seenTexts[part.Text] = true
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// repeatAndJoin repeats the words slice n times and joins them with spaces.
func repeatAndJoin(words []string, n int) string {
	total := n * len(words)
	parts := make([]string, 0, total)
	for i := 0; i < n; i++ {
		parts = append(parts, words...)
	}
	return strings.Join(parts, " ")
}
