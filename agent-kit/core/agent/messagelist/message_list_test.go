// Ported from: packages/core/src/agent/message-list/tests/message-list-ordering.test.ts
// Ported from: packages/core/src/agent/message-list/tests/message-list.test.ts (subset)
// Ported from: packages/core/src/agent/message-list/tests/message-list-sealed.test.ts (subset)
package messagelist

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// ---------------------------------------------------------------------------
// Message ordering with identical timestamps (Issue #10683)
// ---------------------------------------------------------------------------

func TestMessageOrderingIdenticalTimestamps(t *testing.T) {
	t.Run("should preserve input order when messages have identical createdAt timestamps", func(t *testing.T) {
		timestamp := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: timestamp},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "First message"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: timestamp},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Second message"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-3", Role: "user", CreatedAt: timestamp},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Third message"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-4", Role: "assistant", CreatedAt: timestamp},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Fourth message"}}},
			},
		}

		ml := NewMessageList()
		ml.Add(messages, state.MessageSourceMemory)
		result := ml.AllAIV5UI()

		if len(result) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(result))
		}
		expected := []string{"msg-1", "msg-2", "msg-3", "msg-4"}
		for i, exp := range expected {
			if result[i].ID != exp {
				t.Errorf("message %d: expected ID %s, got %s", i, exp, result[i].ID)
			}
		}
	})

	t.Run("should handle mixed timestamps correctly while preserving order for equal timestamps", func(t *testing.T) {
		time1 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		time2 := time.Date(2024, 1, 1, 12, 1, 0, 0, time.UTC)
		time3 := time.Date(2024, 1, 1, 12, 2, 0, 0, time.UTC)

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "a1", Role: "user", CreatedAt: time1},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "A1"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "a2", Role: "assistant", CreatedAt: time1},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "A2"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "b1", Role: "user", CreatedAt: time2},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "B1"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "b2", Role: "assistant", CreatedAt: time2},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "B2"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "c1", Role: "user", CreatedAt: time3},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "C1"}}},
			},
		}

		ml := NewMessageList()
		ml.Add(messages, state.MessageSourceMemory)
		result := ml.AllAIV5UI()

		expected := []string{"a1", "a2", "b1", "b2", "c1"}
		if len(result) != len(expected) {
			t.Fatalf("expected %d messages, got %d", len(expected), len(result))
		}
		for i, exp := range expected {
			if result[i].ID != exp {
				t.Errorf("message %d: expected ID %s, got %s", i, exp, result[i].ID)
			}
		}
	})

	t.Run("should sort messages correctly when user provides out-of-order timestamps", func(t *testing.T) {
		early := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
		middle := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
		late := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "late", Role: "user", CreatedAt: late},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Late message"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "early", Role: "assistant", CreatedAt: early},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Early message"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "middle", Role: "user", CreatedAt: middle},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Middle message"}}},
			},
		}

		ml := NewMessageList()
		ml.Add(messages, state.MessageSourceMemory)
		result := ml.AllAIV5UI()

		// Messages should be sorted by their timestamps: early, middle, late
		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}

		expectedTexts := []string{"Early message", "Middle message", "Late message"}
		for i, exp := range expectedTexts {
			textPart := findTextPart(result[i].Parts)
			if textPart != exp {
				t.Errorf("message %d: expected text %q, got %q", i, exp, textPart)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// Core MessageList operations
// ---------------------------------------------------------------------------

func TestMessageListBasicOperations(t *testing.T) {
	t.Run("Add and retrieve messages", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hello"}},
					Content: "Hello",
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: time.Now().Add(time.Second)},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hi there!"}},
					Content: "Hi there!",
				},
			},
		}

		ml.Add(messages, state.MessageSourceMemory)

		allDB := ml.AllDB()
		if len(allDB) != 2 {
			t.Fatalf("expected 2 DB messages, got %d", len(allDB))
		}

		allV5UI := ml.AllAIV5UI()
		if len(allV5UI) != 2 {
			t.Fatalf("expected 2 V5UI messages, got %d", len(allV5UI))
		}
		if allV5UI[0].Role != "user" {
			t.Errorf("expected first message role user, got %s", allV5UI[0].Role)
		}
		if allV5UI[1].Role != "assistant" {
			t.Errorf("expected second message role assistant, got %s", allV5UI[1].Role)
		}

		allV4UI := ml.AllAIV4UI()
		if len(allV4UI) != 2 {
			t.Fatalf("expected 2 V4UI messages, got %d", len(allV4UI))
		}
	})

	t.Run("Add string message", func(t *testing.T) {
		ml := NewMessageList()
		ml.Add("Hello world", state.MessageSourceInput)

		allDB := ml.AllDB()
		if len(allDB) != 1 {
			t.Fatalf("expected 1 message, got %d", len(allDB))
		}
		if allDB[0].Role != "user" {
			t.Errorf("expected role user, got %s", allDB[0].Role)
		}
	})

	t.Run("RemoveByIds", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Hello"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: time.Now().Add(time.Second)},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Hi"}}},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-3", Role: "user", CreatedAt: time.Now().Add(2 * time.Second)},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Bye"}}},
			},
		}

		ml.Add(messages, state.MessageSourceMemory)

		removed := ml.RemoveByIds([]string{"msg-2"})
		if len(removed) != 1 {
			t.Fatalf("expected 1 removed, got %d", len(removed))
		}
		if removed[0].ID != "msg-2" {
			t.Errorf("expected removed ID msg-2, got %s", removed[0].ID)
		}

		remaining := ml.AllDB()
		if len(remaining) != 2 {
			t.Fatalf("expected 2 remaining, got %d", len(remaining))
		}
		if remaining[0].ID != "msg-1" || remaining[1].ID != "msg-3" {
			t.Errorf("unexpected remaining IDs: %s, %s", remaining[0].ID, remaining[1].ID)
		}
	})

	t.Run("ClearAllDB", func(t *testing.T) {
		ml := NewMessageList()
		ml.Add("Hello", state.MessageSourceInput)
		ml.Add("World", state.MessageSourceInput)

		cleared := ml.ClearAllDB()
		if len(cleared) != 2 {
			t.Fatalf("expected 2 cleared messages, got %d", len(cleared))
		}

		remaining := ml.AllDB()
		if len(remaining) != 0 {
			t.Errorf("expected 0 remaining messages, got %d", len(remaining))
		}
	})

	t.Run("GetLatestUserContent", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "First question"}},
					Content: "First question",
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: time.Now().Add(time.Second)},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Answer"}},
					Content: "Answer",
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-3", Role: "user", CreatedAt: time.Now().Add(2 * time.Second)},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Second question"}},
					Content: "Second question",
				},
			},
		}

		ml.Add(messages, state.MessageSourceMemory)

		latest := ml.GetLatestUserContent()
		if latest != "Second question" {
			t.Errorf("expected 'Second question', got %q", latest)
		}
	})
}

// ---------------------------------------------------------------------------
// System messages
// ---------------------------------------------------------------------------

func TestMessageListSystemMessages(t *testing.T) {
	t.Run("AddSystem and GetSystemMessages", func(t *testing.T) {
		ml := NewMessageList()

		ml.AddSystem("You are a helpful assistant.")

		sysMessages := ml.GetSystemMessages()
		if len(sysMessages) != 1 {
			t.Fatalf("expected 1 system message, got %d", len(sysMessages))
		}
		if sysMessages[0].Content != "You are a helpful assistant." {
			t.Errorf("unexpected system message content: %v", sysMessages[0].Content)
		}
	})

	t.Run("Tagged system messages", func(t *testing.T) {
		ml := NewMessageList()

		ml.AddSystem("Untagged system prompt")
		ml.AddSystem("Memory instruction", "memory")
		ml.AddSystem("Tool instruction", "tools")

		untagged := ml.GetSystemMessages()
		if len(untagged) != 1 {
			t.Fatalf("expected 1 untagged system message, got %d", len(untagged))
		}

		memoryMsgs := ml.GetSystemMessages("memory")
		if len(memoryMsgs) != 1 {
			t.Fatalf("expected 1 memory-tagged system message, got %d", len(memoryMsgs))
		}

		toolMsgs := ml.GetSystemMessages("tools")
		if len(toolMsgs) != 1 {
			t.Fatalf("expected 1 tools-tagged system message, got %d", len(toolMsgs))
		}

		allSystem := ml.GetAllSystemMessages()
		if len(allSystem) != 3 {
			t.Fatalf("expected 3 total system messages, got %d", len(allSystem))
		}
	})

	t.Run("ClearSystemMessages by tag", func(t *testing.T) {
		ml := NewMessageList()

		ml.AddSystem("Untagged")
		ml.AddSystem("Tagged 1", "tag-a")
		ml.AddSystem("Tagged 2", "tag-a")

		ml.ClearSystemMessages("tag-a")

		untagged := ml.GetSystemMessages()
		if len(untagged) != 1 {
			t.Fatalf("expected 1 untagged after clear, got %d", len(untagged))
		}

		tagA := ml.GetSystemMessages("tag-a")
		if len(tagA) != 0 {
			t.Fatalf("expected 0 tag-a messages after clear, got %d", len(tagA))
		}
	})

	t.Run("ReplaceAllSystemMessages", func(t *testing.T) {
		ml := NewMessageList()

		ml.AddSystem("First")
		ml.AddSystem("Second")
		ml.AddSystem("Tagged", "tag")

		ml.ReplaceAllSystemMessages([]state.CoreSystemMessage{
			{Role: "system", Content: "Replacement"},
		})

		allSystem := ml.GetAllSystemMessages()
		if len(allSystem) != 1 {
			t.Fatalf("expected 1 system message after replace, got %d", len(allSystem))
		}
		if allSystem[0].Content != "Replacement" {
			t.Errorf("expected content Replacement, got %v", allSystem[0].Content)
		}
	})

	t.Run("Deduplicate system messages", func(t *testing.T) {
		ml := NewMessageList()

		ml.AddSystem("Same prompt")
		ml.AddSystem("Same prompt")

		sysMessages := ml.GetSystemMessages()
		if len(sysMessages) != 1 {
			t.Fatalf("expected 1 system message (deduplicated), got %d", len(sysMessages))
		}
	})
}

// ---------------------------------------------------------------------------
// Recording events
// ---------------------------------------------------------------------------

func TestMessageListRecording(t *testing.T) {
	t.Run("records add events", func(t *testing.T) {
		ml := NewMessageList()
		ml.StartRecording()

		ml.Add("Hello", state.MessageSourceInput)
		ml.Add("World", state.MessageSourceInput)

		events := ml.StopRecording()
		if len(events) != 2 {
			t.Fatalf("expected 2 recorded events, got %d", len(events))
		}
		if events[0].Type != "add" {
			t.Errorf("expected event type add, got %s", events[0].Type)
		}
		if events[0].Source != state.MessageSourceInput {
			t.Errorf("expected source input, got %s", events[0].Source)
		}
	})

	t.Run("records removeByIds events", func(t *testing.T) {
		ml := NewMessageList()

		msg := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
			Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "Hello"}}},
		}
		ml.Add(msg, state.MessageSourceMemory)

		ml.StartRecording()
		ml.RemoveByIds([]string{"msg-1"})

		events := ml.StopRecording()
		if len(events) != 1 {
			t.Fatalf("expected 1 recorded event, got %d", len(events))
		}
		if events[0].Type != "removeByIds" {
			t.Errorf("expected event type removeByIds, got %s", events[0].Type)
		}
	})

	t.Run("HasRecordedEvents", func(t *testing.T) {
		ml := NewMessageList()
		ml.StartRecording()

		if ml.HasRecordedEvents() {
			t.Error("expected no recorded events before adding")
		}

		ml.Add("Hello", state.MessageSourceInput)

		if !ml.HasRecordedEvents() {
			t.Error("expected recorded events after adding")
		}
	})
}

// ---------------------------------------------------------------------------
// V1 message conversion
// ---------------------------------------------------------------------------

func TestMessageListV1Conversion(t *testing.T) {
	t.Run("AllV1 returns V1 formatted messages", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hello"}},
					Content: "Hello",
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: time.Now().Add(time.Second)},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hi there!"}},
					Content: "Hi there!",
				},
			},
		}

		ml.Add(messages, state.MessageSourceMemory)

		v1Messages := ml.AllV1()
		if len(v1Messages) != 2 {
			t.Fatalf("expected 2 V1 messages, got %d", len(v1Messages))
		}
		if v1Messages[0].Role != "user" {
			t.Errorf("expected first V1 message role user, got %s", v1Messages[0].Role)
		}
		if v1Messages[1].Role != "assistant" {
			t.Errorf("expected second V1 message role assistant, got %s", v1Messages[1].Role)
		}
	})
}

// ---------------------------------------------------------------------------
// Tool invocation messages
// ---------------------------------------------------------------------------

func TestMessageListToolInvocations(t *testing.T) {
	t.Run("should handle messages with tool invocations in parts", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "What is the weather?"}},
					Content: "What is the weather?",
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-2", Role: "assistant", CreatedAt: time.Now().Add(time.Second)},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Let me check:"},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State:      "result",
								ToolCallID: "call-1",
								ToolName:   "get_weather",
								Args:       map[string]any{"location": "SF"},
								Result:     map[string]any{"temp": 72, "condition": "sunny"},
							},
						},
						{Type: "text", Text: "The weather in SF is 72F and sunny."},
					},
				},
			},
		}

		ml.Add(messages, state.MessageSourceMemory)

		v5UI := ml.AllAIV5UI()
		if len(v5UI) != 2 {
			t.Fatalf("expected 2 V5UI messages, got %d", len(v5UI))
		}

		assistantMsg := v5UI[1]
		if assistantMsg.Role != "assistant" {
			t.Errorf("expected assistant role, got %s", assistantMsg.Role)
		}

		// Should have text, tool, text parts
		hasToolPart := false
		for _, p := range assistantMsg.Parts {
			if p.ToolCallID == "call-1" {
				hasToolPart = true
				if p.State != "output-available" {
					t.Errorf("expected tool state output-available, got %s", p.State)
				}
			}
		}
		if !hasToolPart {
			t.Error("expected to find tool part with call-1")
		}
	})
}

// ---------------------------------------------------------------------------
// MessageList with options
// ---------------------------------------------------------------------------

func TestMessageListOptions(t *testing.T) {
	t.Run("thread and resource IDs", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{
			ThreadID:   "thread-123",
			ResourceID: "resource-456",
		})

		ml.Add("Hello", state.MessageSourceInput)

		allDB := ml.AllDB()
		if len(allDB) != 1 {
			t.Fatalf("expected 1 message, got %d", len(allDB))
		}
		if allDB[0].ThreadID != "thread-123" {
			t.Errorf("expected threadId thread-123, got %s", allDB[0].ThreadID)
		}
		if allDB[0].ResourceID != "resource-456" {
			t.Errorf("expected resourceId resource-456, got %s", allDB[0].ResourceID)
		}
	})

	t.Run("custom message ID generator", func(t *testing.T) {
		counter := 0
		ml := NewMessageList(MessageListOptions{
			GenerateMessageID: func(ctx *IdGeneratorContext) string {
				counter++
				return "custom-" + string(rune('0'+counter))
			},
		})

		ml.Add("First", state.MessageSourceInput)
		ml.Add("Second", state.MessageSourceInput)

		allDB := ml.AllDB()
		if len(allDB) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(allDB))
		}
		if allDB[0].ID != "custom-1" {
			t.Errorf("expected custom-1, got %s", allDB[0].ID)
		}
		if allDB[1].ID != "custom-2" {
			t.Errorf("expected custom-2, got %s", allDB[1].ID)
		}
	})
}

// ---------------------------------------------------------------------------
// Serialize / Deserialize
// ---------------------------------------------------------------------------

func TestMessageListSerialization(t *testing.T) {
	t.Run("round-trip serialization preserves messages", func(t *testing.T) {
		ml := NewMessageList()

		messages := []*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "msg-1", Role: "user", CreatedAt: time.Now()},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hello"}},
					Content: "Hello",
				},
			},
		}
		ml.Add(messages, state.MessageSourceMemory)
		ml.AddSystem("You are helpful.")

		serialized := ml.Serialize()

		ml2 := NewMessageList()
		ml2.Deserialize(serialized)

		allDB := ml2.AllDB()
		if len(allDB) != 1 {
			t.Fatalf("expected 1 message after deserialization, got %d", len(allDB))
		}
		if allDB[0].ID != "msg-1" {
			t.Errorf("expected msg-1, got %s", allDB[0].ID)
		}

		sysMessages := ml2.GetSystemMessages()
		if len(sysMessages) != 1 {
			t.Fatalf("expected 1 system message after deserialization, got %d", len(sysMessages))
		}
	})
}

// ---------------------------------------------------------------------------
// DrainUnsavedMessages
// ---------------------------------------------------------------------------

func TestMessageListDrainUnsaved(t *testing.T) {
	t.Run("drains input and response messages", func(t *testing.T) {
		ml := NewMessageList()

		ml.Add([]*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{ID: "mem-1", Role: "user", CreatedAt: time.Now()},
				Content:             state.MastraMessageContentV2{Format: 2, Parts: []state.MastraMessagePart{{Type: "text", Text: "From memory"}}},
			},
		}, state.MessageSourceMemory)

		ml.Add("New input", state.MessageSourceInput)

		drained := ml.DrainUnsavedMessages()
		if len(drained) != 1 {
			t.Fatalf("expected 1 drained message (input only), got %d", len(drained))
		}
	})
}

// ---------------------------------------------------------------------------
// Sealed message handling
// Ported from: packages/core/src/agent/message-list/tests/message-list-sealed.test.ts
// ---------------------------------------------------------------------------

func TestMessageListSealedMessages(t *testing.T) {
	t.Run("should not replace a sealed message, but create a new message with only new parts", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{ThreadID: "test-thread"})

		ml.Add(map[string]any{"role": "user", "content": "Hello"}, state.MessageSourceInput)

		assistantMessageID := "assistant-msg-1"
		sealedPart := state.MastraMessagePart{
			Type: "text",
			Text: "Hello! How can I help?",
			Metadata: map[string]any{
				"mastra": map[string]any{"sealedAt": time.Now().UnixMilli()},
			},
		}

		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format:   2,
				Parts:    []state.MastraMessagePart{sealedPart},
				Metadata: map[string]any{"mastra": map[string]any{"sealed": true}},
			},
		}, state.MessageSourceResponse)

		// Streaming continues - accumulated message with same ID (old parts + new parts)
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					sealedPart,
					{Type: "text", Text: "Here is more content after observation."},
				},
			},
		}, state.MessageSourceResponse)

		allMessages := ml.AllDB()
		if len(allMessages) != 3 {
			t.Fatalf("expected 3 messages (user, sealed assistant, new assistant), got %d", len(allMessages))
		}

		var sealedMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID == assistantMessageID {
				sealedMessage = m
				break
			}
		}
		if sealedMessage == nil {
			t.Fatal("expected to find sealed message")
		}
		if len(sealedMessage.Content.Parts) != 1 {
			t.Fatalf("expected sealed message to have 1 part, got %d", len(sealedMessage.Content.Parts))
		}
		if sealedMessage.Content.Parts[0].Text != "Hello! How can I help?" {
			t.Errorf("expected sealed text, got %s", sealedMessage.Content.Parts[0].Text)
		}

		var newMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID != assistantMessageID && m.Role == "assistant" {
				newMessage = m
				break
			}
		}
		if newMessage == nil {
			t.Fatal("expected to find new message")
		}
		if len(newMessage.Content.Parts) != 1 {
			t.Fatalf("expected new message to have 1 part, got %d", len(newMessage.Content.Parts))
		}
		if newMessage.Content.Parts[0].Text != "Here is more content after observation." {
			t.Errorf("expected new content text, got %s", newMessage.Content.Parts[0].Text)
		}
		if newMessage.ID == assistantMessageID {
			t.Error("expected new message to have a different ID")
		}
	})

	t.Run("should not create a new message if incoming message has no new parts after sealedAt", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{ThreadID: "test-thread"})

		ml.Add(map[string]any{"role": "user", "content": "Hello"}, state.MessageSourceInput)

		assistantMessageID := "assistant-msg-1"
		sealedPart := state.MastraMessagePart{
			Type: "text",
			Text: "Response",
			Metadata: map[string]any{
				"mastra": map[string]any{"sealedAt": time.Now().UnixMilli()},
			},
		}

		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format:   2,
				Parts:    []state.MastraMessagePart{sealedPart},
				Metadata: map[string]any{"mastra": map[string]any{"sealed": true}},
			},
		}, state.MessageSourceResponse)

		// Try to add same content again (no new parts)
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts:  []state.MastraMessagePart{sealedPart},
			},
		}, state.MessageSourceResponse)

		allMessages := ml.AllDB()
		if len(allMessages) != 2 {
			t.Fatalf("expected 2 messages (no new message created), got %d", len(allMessages))
		}
	})

	t.Run("should still merge into non-sealed messages normally", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{ThreadID: "test-thread"})

		ml.Add(map[string]any{"role": "user", "content": "Hello"}, state.MessageSourceInput)

		assistantMessageID := "assistant-msg-1"
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts:  []state.MastraMessagePart{{Type: "text", Text: "Part 1"}},
			},
		}, state.MessageSourceResponse)

		// Add more parts (should merge since not sealed)
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts:  []state.MastraMessagePart{{Type: "text", Text: "Part 2"}},
			},
		}, state.MessageSourceResponse)

		allMessages := ml.AllDB()
		if len(allMessages) != 2 {
			t.Fatalf("expected 2 messages (user + merged assistant), got %d", len(allMessages))
		}

		var assistantMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.Role == "assistant" {
				assistantMessage = m
				break
			}
		}
		if assistantMessage == nil {
			t.Fatal("expected to find assistant message")
		}
		if len(assistantMessage.Content.Parts) < 2 {
			t.Fatalf("expected at least 2 parts (merged), got %d", len(assistantMessage.Content.Parts))
		}
	})

	t.Run("should add text flushed independently to a sealed message as a new message", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{ThreadID: "test-thread"})

		ml.Add(map[string]any{"role": "user", "content": "Hello"}, state.MessageSourceInput)

		assistantMessageID := "assistant-msg-1"
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "data-om-status", DataPayload: map[string]any{"windows": map[string]any{}}},
					{
						Type: "tool-invocation",
						ToolInvocation: &state.ToolInvocation{
							ToolCallID: "call-1",
							ToolName:   "view",
							State:      "result",
							Args:       map[string]any{},
							Result:     "ok",
						},
						Metadata: map[string]any{
							"mastra": map[string]any{"sealedAt": time.Now().UnixMilli()},
						},
					},
				},
				Metadata: map[string]any{"mastra": map[string]any{"sealed": true}},
			},
		}, state.MessageSourceResponse)

		// Text deltas flushed independently with SAME messageId
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts:  []state.MastraMessagePart{{Type: "text", Text: "Here is my analysis of the codebase..."}},
			},
		}, state.MessageSourceResponse)

		allMessages := ml.AllDB()
		if len(allMessages) != 3 {
			t.Fatalf("expected 3 messages (user, sealed assistant, new text), got %d", len(allMessages))
		}

		var sealedMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID == assistantMessageID {
				sealedMessage = m
				break
			}
		}
		if sealedMessage == nil {
			t.Fatal("expected sealed message")
		}
		if len(sealedMessage.Content.Parts) != 2 {
			t.Fatalf("expected sealed message to have 2 parts, got %d", len(sealedMessage.Content.Parts))
		}

		var textMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID != assistantMessageID && m.Role == "assistant" {
				textMessage = m
				break
			}
		}
		if textMessage == nil {
			t.Fatal("expected to find new text message")
		}
		if len(textMessage.Content.Parts) != 1 {
			t.Fatalf("expected 1 part in new message, got %d", len(textMessage.Content.Parts))
		}
		if textMessage.Content.Parts[0].Type != "text" {
			t.Errorf("expected text type, got %s", textMessage.Content.Parts[0].Type)
		}
		if textMessage.Content.Parts[0].Text != "Here is my analysis of the codebase..." {
			t.Errorf("expected analysis text, got %s", textMessage.Content.Parts[0].Text)
		}

		responseMsgs := ml.ResponseDB()
		hasTextResponse := false
		for _, m := range responseMsgs {
			for _, p := range m.Content.Parts {
				if p.Type == "text" {
					hasTextResponse = true
					break
				}
			}
		}
		if !hasTextResponse {
			t.Error("expected response messages to include the text message")
		}
	})

	t.Run("should preserve observation markers in sealed messages", func(t *testing.T) {
		ml := NewMessageList(MessageListOptions{ThreadID: "test-thread"})

		ml.Add(map[string]any{"role": "user", "content": "Hello"}, state.MessageSourceInput)

		assistantMessageID := "assistant-msg-1"
		observationMarkerPart := state.MastraMessagePart{
			Type: "data-om-observation-end",
			DataPayload: map[string]any{
				"cycleId":           "cycle-1",
				"recordId":         "record-1",
				"observedAt":       time.Now().Format(time.RFC3339),
				"tokensObserved":   100,
				"observationTokens": 50,
			},
			Metadata: map[string]any{
				"mastra": map[string]any{"sealedAt": time.Now().UnixMilli()},
			},
		}

		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "text", Text: "Response text"},
					observationMarkerPart,
				},
				Metadata: map[string]any{"mastra": map[string]any{"sealed": true}},
			},
		}, state.MessageSourceResponse)

		// Streaming continues with old parts + new parts
		ml.Add(&state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        assistantMessageID,
				Role:      "assistant",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "text", Text: "Response text"},
					observationMarkerPart,
					{Type: "text", Text: "New content after observation"},
				},
			},
		}, state.MessageSourceResponse)

		allMessages := ml.AllDB()
		if len(allMessages) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(allMessages))
		}

		var sealedMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID == assistantMessageID {
				sealedMessage = m
				break
			}
		}
		if sealedMessage == nil {
			t.Fatal("expected sealed message")
		}
		hasObservationMarker := false
		for _, p := range sealedMessage.Content.Parts {
			if p.Type == "data-om-observation-end" {
				hasObservationMarker = true
				break
			}
		}
		if !hasObservationMarker {
			t.Error("expected sealed message to still have observation marker")
		}

		var newMessage *state.MastraDBMessage
		for _, m := range allMessages {
			if m.ID != assistantMessageID && m.Role == "assistant" {
				newMessage = m
				break
			}
		}
		if newMessage == nil {
			t.Fatal("expected new message")
		}
		if len(newMessage.Content.Parts) != 1 {
			t.Fatalf("expected 1 part in new message, got %d", len(newMessage.Content.Parts))
		}
		if newMessage.Content.Parts[0].Text != "New content after observation" {
			t.Errorf("expected 'New content after observation', got %s", newMessage.Content.Parts[0].Text)
		}
	})
}

// ---------------------------------------------------------------------------
// File URL Handling
// Ported from: packages/core/src/agent/message-list/tests/message-list-url-handling.test.ts (subset)
// ---------------------------------------------------------------------------

func TestMessageListURLHandling(t *testing.T) {
	t.Run("should preserve external URLs through V2->V5->V2 message conversion", func(t *testing.T) {
		ml := NewMessageList()
		imageURL := "https://placehold.co/10.png"

		v2Message := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:         "test-msg-1",
				Role:       "user",
				CreatedAt:  time.Now(),
				ResourceID: "test-resource",
				ThreadID:   "test-thread",
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "text", Text: "Describe this image"},
					{Type: "file", MimeType: "image/png", Data: imageURL},
				},
			},
		}

		ml.Add(v2Message, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		var v5FilePart *adapters.AIV5UIPart
		for i := range v5Messages[0].Parts {
			if v5Messages[0].Parts[i].Type == "file" {
				v5FilePart = &v5Messages[0].Parts[i]
				break
			}
		}
		if v5FilePart == nil {
			t.Fatal("expected to find file part in V5 message")
		}
		if v5FilePart.URL != imageURL {
			t.Errorf("expected V5 URL %s, got %s", imageURL, v5FilePart.URL)
		}
		if strings.Contains(v5FilePart.URL, "data:image/png;base64,https://") {
			t.Error("V5 URL should NOT contain malformed data URI")
		}

		v2MessagesBack := ml.AllDB()
		var v2FilePartBack *state.MastraMessagePart
		for i := range v2MessagesBack[0].Content.Parts {
			if v2MessagesBack[0].Content.Parts[i].Type == "file" {
				v2FilePartBack = &v2MessagesBack[0].Content.Parts[i]
				break
			}
		}
		if v2FilePartBack == nil {
			t.Fatal("expected to find file part in V2 message")
		}
		if v2FilePartBack.Data != imageURL {
			t.Errorf("expected V2 data %s, got %s", imageURL, v2FilePartBack.Data)
		}
		if strings.Contains(v2FilePartBack.Data, "data:image/png;base64,https://") {
			t.Error("V2 data should NOT contain malformed data URI")
		}
	})

	t.Run("should provide clean URLs to InputProcessors without data URI corruption", func(t *testing.T) {
		ml := NewMessageList()
		imageURL := "https://placehold.co/10.png"

		inputMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:         "input-msg",
				Role:       "user",
				CreatedAt:  time.Now(),
				ResourceID: "resource-1",
				ThreadID:   "thread-1",
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/png", Data: imageURL},
					{Type: "text", Text: "What is this?"},
				},
			},
		}

		ml.Add(inputMessage, state.MessageSourceInput)

		v2Messages := ml.AllDB()
		var filePart *state.MastraMessagePart
		for i := range v2Messages[0].Content.Parts {
			if v2Messages[0].Content.Parts[i].Type == "file" {
				filePart = &v2Messages[0].Content.Parts[i]
				break
			}
		}
		if filePart == nil {
			t.Fatal("expected file part")
		}
		if filePart.Data != imageURL {
			t.Errorf("expected clean URL %s, got %s", imageURL, filePart.Data)
		}
		if strings.Contains(filePart.Data, "data:image/png;base64,") {
			t.Error("data should not contain data URI prefix")
		}
		if strings.HasPrefix(filePart.Data, "data:") {
			t.Error("data should not be a data URI")
		}
	})

	t.Run("should correctly differentiate between URLs and base64 data", func(t *testing.T) {
		imageURL := "https://placehold.co/10.png"
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
		dataURI := "data:image/png;base64," + base64Data

		tests := []struct {
			name     string
			id       string
			data     string
			v5URL    string
			v2Data   string
			v5Check  func(t *testing.T, url string)
			v2Check  func(t *testing.T, data string)
		}{
			{
				name:   "URL should be preserved as-is",
				id:     "url-msg",
				data:   imageURL,
				v5URL:  imageURL,
				v2Data: imageURL,
			},
			{
				name: "base64 gets wrapped in data URI for V5",
				id:   "base64-msg",
				data: base64Data,
				v5Check: func(t *testing.T, url string) {
					if !strings.Contains(url, "data:image/png;base64,") {
						t.Errorf("expected V5 URL to contain data URI prefix, got %s", url)
					}
				},
				v2Data: base64Data,
			},
			{
				name:   "data URI should be preserved throughout",
				id:     "datauri-msg",
				data:   dataURI,
				v5URL:  dataURI,
				v2Data: dataURI,
			},
		}

		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				ml := NewMessageList()
				ml.Add(&state.MastraDBMessage{
					MastraMessageShared: state.MastraMessageShared{
						ID:         tc.id,
						Role:       "user",
						CreatedAt:  time.Now(),
						ResourceID: "r1",
						ThreadID:   "t1",
					},
					Content: state.MastraMessageContentV2{
						Format: 2,
						Parts:  []state.MastraMessagePart{{Type: "file", MimeType: "image/png", Data: tc.data}},
					},
				}, state.MessageSourceInput)

				v5Messages := ml.AllAIV5UI()
				var v5FilePart *adapters.AIV5UIPart
				for i := range v5Messages[0].Parts {
					if v5Messages[0].Parts[i].Type == "file" {
						v5FilePart = &v5Messages[0].Parts[i]
						break
					}
				}
				if v5FilePart == nil {
					t.Fatal("expected file part in V5")
				}
				if tc.v5URL != "" && v5FilePart.URL != tc.v5URL {
					t.Errorf("expected V5 URL %s, got %s", tc.v5URL, v5FilePart.URL)
				}
				if tc.v5Check != nil {
					tc.v5Check(t, v5FilePart.URL)
				}

				v2Messages := ml.AllDB()
				var v2FilePart *state.MastraMessagePart
				for i := range v2Messages[0].Content.Parts {
					if v2Messages[0].Content.Parts[i].Type == "file" {
						v2FilePart = &v2Messages[0].Content.Parts[i]
						break
					}
				}
				if v2FilePart == nil {
					t.Fatal("expected file part in V2")
				}
				if tc.v2Data != "" && v2FilePart.Data != tc.v2Data {
					t.Errorf("expected V2 data %s, got %s", tc.v2Data, v2FilePart.Data)
				}
				if tc.v2Check != nil {
					tc.v2Check(t, v2FilePart.Data)
				}
			})
		}
	})
}

// ---------------------------------------------------------------------------
// AI SDK V5 URL handling
// Ported from: packages/core/src/agent/message-list/tests/message-list-aisdk-v5-url.test.ts
// ---------------------------------------------------------------------------

func TestMessageListAISDKV5URLHandling(t *testing.T) {
	t.Run("should preserve remote URLs when converting messages for AI SDK v5", func(t *testing.T) {
		ml := NewMessageList()

		userMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        "msg-1",
				Role:      "user",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/png", Data: "https://storage.easyquiz.cc/ai-chat/20250905cdacd4dff092.png"},
					{Type: "text", Text: "Describe it"},
				},
			},
		}

		ml.Add([]*state.MastraDBMessage{userMessage}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}
		if v5Messages[0].Role != "user" {
			t.Errorf("expected user role, got %s", v5Messages[0].Role)
		}
		if len(v5Messages[0].Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(v5Messages[0].Parts))
		}

		filePart := v5Messages[0].Parts[0]
		if filePart.Type != "file" {
			t.Errorf("expected file type, got %s", filePart.Type)
		}
		if filePart.URL != "https://storage.easyquiz.cc/ai-chat/20250905cdacd4dff092.png" {
			t.Errorf("expected URL to be preserved, got %s", filePart.URL)
		}
		if strings.Contains(filePart.URL, "data:image/png;base64,https://") {
			t.Error("URL should NOT be wrapped as a malformed data URI")
		}
		if strings.Contains(filePart.URL, "base64,https://") {
			t.Error("URL should NOT contain base64,https://")
		}
	})

	t.Run("should handle multiple image URLs in the same message", func(t *testing.T) {
		ml := NewMessageList()

		userMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        "msg-2",
				Role:      "user",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/jpeg", Data: "https://example.com/image1.jpg"},
					{Type: "text", Text: "Compare these images"},
					{Type: "file", MimeType: "image/png", Data: "https://example.com/image2.png"},
				},
			},
		}

		ml.Add([]*state.MastraDBMessage{userMessage}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}
		if len(v5Messages[0].Parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(v5Messages[0].Parts))
		}

		firstFile := v5Messages[0].Parts[0]
		if firstFile.Type == "file" {
			if firstFile.URL != "https://example.com/image1.jpg" {
				t.Errorf("expected first file URL, got %s", firstFile.URL)
			}
			if firstFile.MediaType != "image/jpeg" {
				t.Errorf("expected image/jpeg, got %s", firstFile.MediaType)
			}
		}

		secondFile := v5Messages[0].Parts[2]
		if secondFile.Type == "file" {
			if secondFile.URL != "https://example.com/image2.png" {
				t.Errorf("expected second file URL, got %s", secondFile.URL)
			}
			if secondFile.MediaType != "image/png" {
				t.Errorf("expected image/png, got %s", secondFile.MediaType)
			}
		}
	})

	t.Run("should handle base64 data URIs correctly", func(t *testing.T) {
		ml := NewMessageList()

		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
		dataURI := "data:image/png;base64," + base64Data

		userMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        "msg-3",
				Role:      "user",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/png", Data: dataURI},
					{Type: "text", Text: "What is this?"},
				},
			},
		}

		ml.Add([]*state.MastraDBMessage{userMessage}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}
		if len(v5Messages[0].Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(v5Messages[0].Parts))
		}

		filePart := v5Messages[0].Parts[0]
		if filePart.Type == "file" {
			if filePart.URL != dataURI {
				t.Errorf("expected data URI preserved, got %s", filePart.URL)
			}
			if filePart.MediaType != "image/png" {
				t.Errorf("expected image/png, got %s", filePart.MediaType)
			}
		}
	})

	t.Run("should handle plain base64 strings (no data URI prefix)", func(t *testing.T) {
		ml := NewMessageList()

		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

		userMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        "msg-4",
				Role:      "user",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/png", Data: base64Data},
					{Type: "text", Text: "What is this?"},
				},
			},
		}

		ml.Add([]*state.MastraDBMessage{userMessage}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}
		if len(v5Messages[0].Parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(v5Messages[0].Parts))
		}

		filePart := v5Messages[0].Parts[0]
		if filePart.Type == "file" {
			expectedDataURI := "data:image/png;base64," + base64Data
			if filePart.URL != expectedDataURI {
				t.Errorf("expected data URI %s, got %s", expectedDataURI, filePart.URL)
			}
			if filePart.MediaType != "image/png" {
				t.Errorf("expected image/png, got %s", filePart.MediaType)
			}
		}
	})

	t.Run("should NOT wrap non-http URLs as data URIs when they are actual URLs", func(t *testing.T) {
		ml := NewMessageList()

		userMessage := &state.MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        "msg-edge-1",
				Role:      "user",
				CreatedAt: time.Now(),
			},
			Content: state.MastraMessageContentV2{
				Format: 2,
				Parts: []state.MastraMessagePart{
					{Type: "file", MimeType: "image/png", Data: "//storage.example.com/image.png"},
					{Type: "text", Text: "What is this?"},
				},
			},
		}

		ml.Add([]*state.MastraDBMessage{userMessage}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}

		filePart := v5Messages[0].Parts[0]
		if filePart.Type == "file" {
			matched, _ := regexp.MatchString(`^data:.*base64,//`, filePart.URL)
			if matched {
				t.Error("protocol-relative URL should NOT be wrapped as a data URI")
			}
		}
	})

	t.Run("should handle remote URL in file part from user report", func(t *testing.T) {
		ml := NewMessageList()

		ml.Add([]*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:        "msg-report-1",
					Role:      "user",
					CreatedAt: time.Now(),
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "file", MimeType: "image/png", Data: "https://storage.easyquiz.cc/ai-chat/20250905cdacd4dff092.png"},
						{Type: "text", Text: "Describe it"},
					},
				},
			},
		}, state.MessageSourceInput)

		v2Messages := ml.AllDB()
		if len(v2Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v2Messages))
		}
		if v2Messages[0].Role != "user" {
			t.Errorf("expected user role, got %s", v2Messages[0].Role)
		}

		var filePart *state.MastraMessagePart
		for i := range v2Messages[0].Content.Parts {
			if v2Messages[0].Content.Parts[i].Type == "file" {
				filePart = &v2Messages[0].Content.Parts[i]
				break
			}
		}
		if filePart == nil {
			t.Fatal("expected file part")
		}
		if filePart.Data != "https://storage.easyquiz.cc/ai-chat/20250905cdacd4dff092.png" {
			t.Errorf("expected URL preserved in V2 data, got %s", filePart.Data)
		}
		if filePart.MimeType != "image/png" {
			t.Errorf("expected image/png, got %s", filePart.MimeType)
		}

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 V5 message, got %d", len(v5Messages))
		}
		if len(v5Messages[0].Parts) != 2 {
			t.Fatalf("expected 2 V5 parts, got %d", len(v5Messages[0].Parts))
		}

		v5FilePart := v5Messages[0].Parts[0]
		if v5FilePart.Type != "file" {
			t.Errorf("expected file type, got %s", v5FilePart.Type)
		}
		if v5FilePart.URL != "https://storage.easyquiz.cc/ai-chat/20250905cdacd4dff092.png" {
			t.Errorf("expected URL preserved in V5, got %s", v5FilePart.URL)
		}
		if strings.Contains(v5FilePart.URL, "data:image/png;base64,https://") {
			t.Error("URL should NOT be wrapped as malformed data URI in V5")
		}
	})

	t.Run("should handle base64 data correctly from user report", func(t *testing.T) {
		ml := NewMessageList()

		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="

		ml.Add([]*state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:        "msg-base64-report",
					Role:      "user",
					CreatedAt: time.Now(),
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "file", MimeType: "image/png", Data: base64Data},
						{Type: "text", Text: "Describe it"},
					},
				},
			},
		}, state.MessageSourceInput)

		v5Messages := ml.AllAIV5UI()
		if len(v5Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(v5Messages))
		}

		v5FilePart := v5Messages[0].Parts[0]
		if v5FilePart.Type == "file" {
			expectedDataURI := "data:image/png;base64," + base64Data
			if v5FilePart.URL != expectedDataURI {
				t.Errorf("expected proper data URI, got %s", v5FilePart.URL)
			}
			if v5FilePart.MediaType != "image/png" {
				t.Errorf("expected image/png, got %s", v5FilePart.MediaType)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func findTextPart(parts []adapters.AIV5UIPart) string {
	for _, p := range parts {
		if p.Type == "text" {
			return p.Text
		}
	}
	return ""
}
