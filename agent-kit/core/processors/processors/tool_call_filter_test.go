// Ported from: packages/core/src/processors/processors/tool-call-filter.test.ts
package concreteprocessors

import (
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeToolInvocationPart(toolName, toolCallID, st string) processors.MastraMessagePart {
	return processors.MastraMessagePart{
		Type: "tool-invocation",
		ToolInvocation: &processors.ToolInvocation{
			ToolCallID: toolCallID,
			ToolName:   toolName,
			State:      st,
		},
	}
}

func makeToolInvocationPartWithArgs(toolName, toolCallID, st string, args map[string]any) processors.MastraMessagePart {
	return processors.MastraMessagePart{
		Type: "tool-invocation",
		ToolInvocation: &processors.ToolInvocation{
			ToolCallID: toolCallID,
			ToolName:   toolName,
			State:      st,
			Args:       args,
		},
	}
}

func makeToolInvocationPartWithResult(toolName, toolCallID, st string, result any) processors.MastraMessagePart {
	return processors.MastraMessagePart{
		Type: "tool-invocation",
		ToolInvocation: &processors.ToolInvocation{
			ToolCallID: toolCallID,
			ToolName:   toolName,
			State:      st,
			Result:     result,
		},
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestToolCallFilter(t *testing.T) {

	t.Run("exclude all tool calls (default)", func(t *testing.T) {
		t.Run("should exclude all tool call messages", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeTextMessage("user", "hello"),
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
				makeTextMessage("assistant", "result"),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(result))
			}
			if result[0].Content.Parts[0].Text != "hello" {
				t.Fatal("expected first message to be 'hello'")
			}
			if result[1].Content.Parts[0].Text != "result" {
				t.Fatal("expected second message to be 'result'")
			}
		})

		t.Run("should handle messages with no tool calls", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeTextMessage("user", "hello"),
				makeTextMessage("assistant", "world"),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(result))
			}
		})

		t.Run("should handle empty messages", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			result, _, _, err := f.ProcessInput(defaultArgs([]processors.MastraDBMessage{}))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 0 {
				t.Fatalf("expected 0 messages, got %d", len(result))
			}
		})

		t.Run("should handle multiple tool calls in one message", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
					makeToolInvocationPart("calculate", "call-2", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// All tool invocations filtered, message should be dropped
			if len(result) != 0 {
				t.Fatalf("expected 0 messages (all parts filtered), got %d", len(result))
			}
		})

		t.Run("should keep text parts when tool invocations are filtered", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					{Type: "text", Text: "I'll search for that"},
					makeToolInvocationPart("search", "call-1", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			if len(result[0].Content.Parts) != 1 {
				t.Fatalf("expected 1 part, got %d", len(result[0].Content.Parts))
			}
			if result[0].Content.Parts[0].Text != "I'll search for that" {
				t.Fatal("expected text part to be preserved")
			}
		})
	})

	t.Run("exclude specific tools", func(t *testing.T) {
		t.Run("should exclude only specified tools", func(t *testing.T) {
			f := NewToolCallFilter(&ToolCallFilterOptions{
				Exclude: []string{"search"},
			})
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("calculate", "call-2", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message (calculate only), got %d", len(result))
			}
			if result[0].Content.Parts[0].ToolInvocation.ToolName != "calculate" {
				t.Fatal("expected 'calculate' tool to remain")
			}
		})

		t.Run("should exclude multiple specified tools", func(t *testing.T) {
			f := NewToolCallFilter(&ToolCallFilterOptions{
				Exclude: []string{"search", "calculate"},
			})
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("calculate", "call-2", "call"),
				}),
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("fetch", "call-3", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message (fetch only), got %d", len(result))
			}
			if result[0].Content.Parts[0].ToolInvocation.ToolName != "fetch" {
				t.Fatal("expected 'fetch' tool to remain")
			}
		})

		t.Run("should pass through all messages with empty exclude list", func(t *testing.T) {
			f := NewToolCallFilter(&ToolCallFilterOptions{
				Exclude: []string{},
			})
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
		})

		t.Run("should not filter when tool name does not match exclude", func(t *testing.T) {
			f := NewToolCallFilter(&ToolCallFilterOptions{
				Exclude: []string{"nonexistent"},
			})
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
		})

		t.Run("should exclude tool results for excluded tools", func(t *testing.T) {
			f := NewToolCallFilter(&ToolCallFilterOptions{
				Exclude: []string{"search"},
			})
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPart("search", "call-1", "call"),
				}),
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPartWithResult("search", "call-1", "result", "search result"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 0 {
				t.Fatalf("expected 0 messages (both call and result filtered), got %d", len(result))
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should handle messages with no parts", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeMessage("user", nil),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
		})

		t.Run("should handle result-only messages with excludeAll", func(t *testing.T) {
			f := NewToolCallFilter(nil)
			messages := []processors.MastraDBMessage{
				makeMessage("assistant", []processors.MastraMessagePart{
					makeToolInvocationPartWithResult("search", "call-1", "result", "search result"),
				}),
			}
			result, _, _, err := f.ProcessInput(defaultArgs(messages))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// tool-invocation parts should be filtered
			if len(result) != 0 {
				t.Fatalf("expected 0 messages, got %d", len(result))
			}
		})
	})
}
