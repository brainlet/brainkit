// Ported from: packages/core/src/processors/processors/token-limiter.test.ts
package concreteprocessors

import (
	"strings"
	"testing"
	"time"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

func createTLTestMessage(text string, role string, id string) processors.MastraDBMessage {
	return processors.MastraDBMessage{
		ID:   id,
		Role: role,
		Content: processors.MastraMessageContentV2{
			Format:  2,
			Parts:   []processors.MessagePart{{Type: "text", Text: text}},
			Content: text,
		},
		CreatedAt: time.Now(),
	}
}

func TestTokenLimiterProcessor(t *testing.T) {
	t.Run("basic functionality", func(t *testing.T) {
		t.Run("should allow chunks within token limit", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			state := map[string]any{}
			result, err := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result for text within limit")
			}
			tokens, ok := state["currentTokens"].(float64)
			if !ok || tokens <= 0 {
				t.Fatal("expected currentTokens > 0 in state")
			}
			if tokens > 10 {
				t.Fatalf("expected currentTokens <= 10, got %f", tokens)
			}
		})

		t.Run("should truncate when token limit is exceeded (default strategy)", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(5, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// First part should be allowed
			chunk1 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			result1, err := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk1,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result1 == nil {
				t.Fatal("expected first chunk to be allowed")
			}

			// Second part (long text) should be truncated (nil)
			chunk2 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": " world this is a very long message that will exceed the token limit"},
			}
			result2, err := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk2,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result2 != nil {
				t.Fatal("expected nil result when token limit exceeded with truncate strategy")
			}
		})

		t.Run("should accept simple number constructor", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			result, err := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if proc.GetMaxTokens() != 10 {
				t.Fatalf("expected max tokens 10, got %d", proc.GetMaxTokens())
			}
		})
	})

	t.Run("abort strategy", func(t *testing.T) {
		t.Run("should abort when token limit is exceeded", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(5, &TokenLimiterOptions{Strategy: "abort"})
			abortCalled := false
			abortMessage := ""
			mockAbort := func(reason string, opts *processors.TripWireOptions) error {
				abortCalled = true
				abortMessage = reason
				return nil
			}

			// First part should be allowed
			chunk1 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk1,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			// Second part should trigger abort
			chunk2 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": " world this is a very long message"},
			}
			proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk2,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			if !abortCalled {
				t.Fatal("expected abort to be called")
			}
			if !strings.Contains(abortMessage, "Token limit of 5 exceeded") {
				t.Fatalf("expected abort message to contain 'Token limit of 5 exceeded', got '%s'", abortMessage)
			}
		})
	})

	t.Run("count modes", func(t *testing.T) {
		t.Run("should use cumulative counting by default", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			state := map[string]any{}

			chunk1 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk1,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			tokensAfter1, _ := state["currentTokens"].(float64)

			chunk2 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": " world"},
			}
			proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk2,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			tokensAfter2, _ := state["currentTokens"].(float64)

			if tokensAfter2 <= tokensAfter1 {
				t.Fatalf("expected cumulative tokens to increase: after1=%f, after2=%f", tokensAfter1, tokensAfter2)
			}

			// Third part should be truncated
			chunk3 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": " this is a very long message that will definitely exceed the token limit"},
			}
			result3, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk3,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result3 != nil {
				t.Fatal("expected nil result for chunk exceeding cumulative limit")
			}
		})

		t.Run("should use part counting when specified", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(5, &TokenLimiterOptions{CountMode: "part"})
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// Short part should be allowed
			chunk1 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			state1 := map[string]any{}
			result1, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk1,
				State: state1,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result1 == nil {
				t.Fatal("expected first chunk to be allowed in part mode")
			}

			// Long part should be truncated
			chunk2 := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": " world this is a very long message"},
			}
			state2 := map[string]any{}
			result2, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  chunk2,
				State: state2,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result2 != nil {
				t.Fatal("expected nil result for long chunk in part mode")
			}

			// Token count should be reset
			tokens, _ := state2["currentTokens"].(float64)
			if tokens != 0 {
				t.Fatalf("expected token count reset to 0 in part mode, got %f", tokens)
			}
		})
	})

	t.Run("different part types", func(t *testing.T) {
		t.Run("should handle text-delta chunks", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello world"},
			}
			result, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result == nil {
				t.Fatal("expected non-nil result for text-delta")
			}
		})
	})

	t.Run("utility methods", func(t *testing.T) {
		t.Run("should return max tokens", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(42, nil)
			if proc.GetMaxTokens() != 42 {
				t.Fatalf("expected 42, got %d", proc.GetMaxTokens())
			}
		})

		t.Run("should track tokens in state", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			state := map[string]any{}
			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			tokens, ok := state["currentTokens"].(float64)
			if !ok || tokens <= 0 {
				t.Fatal("expected currentTokens > 0 after processing")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should handle empty text chunks", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(5, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": ""},
			}
			state := map[string]any{}
			result, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result == nil {
				t.Fatal("expected non-nil result for empty text")
			}
		})

		t.Run("should handle very large limits", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(1000000, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello world"},
			}
			result, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result == nil {
				t.Fatal("expected non-nil result with large limit")
			}
		})

		t.Run("should handle zero limit", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(0, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello"},
			}
			result, _ := proc.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if result != nil {
				t.Fatal("expected nil result with zero limit")
			}
		})
	})

	t.Run("processOutputResult", func(t *testing.T) {
		t.Run("should truncate text content that exceeds token limit", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			messages := []processors.MastraDBMessage{
				createTLTestMessage("This is a very long message that will definitely exceed the token limit of 10 tokens", "assistant", "test-id"),
			}

			result, _, err := proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			truncatedText := result[0].Content.Parts[0].Text
			originalText := messages[0].Content.Parts[0].Text
			if len(truncatedText) >= len(originalText) {
				t.Fatalf("expected truncated text to be shorter than original: truncated=%d, original=%d", len(truncatedText), len(originalText))
			}
			if len(truncatedText) == 0 {
				t.Fatal("expected truncated text to not be empty")
			}
		})

		t.Run("should not truncate text content within token limit", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(50, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			originalText := "This is a short message"
			messages := []processors.MastraDBMessage{
				createTLTestMessage(originalText, "assistant", "test-id"),
			}

			result, _, err := proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			if result[0].Content.Parts[0].Text != originalText {
				t.Fatalf("expected '%s', got '%s'", originalText, result[0].Content.Parts[0].Text)
			}
		})

		t.Run("should handle non-assistant messages", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			messages := []processors.MastraDBMessage{
				createTLTestMessage("This is a user message that should not be processed", "user", "test-id"),
			}

			result, _, err := proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}
			// User messages should not be processed
			if result[0].Content.Parts[0].Text != messages[0].Content.Parts[0].Text {
				t.Fatal("user message should not be modified")
			}
		})

		t.Run("should abort when token limit is exceeded with abort strategy", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(10, &TokenLimiterOptions{Strategy: "abort"})
			abortCalled := false
			mockAbort := func(reason string, opts *processors.TripWireOptions) error {
				abortCalled = true
				return nil
			}

			messages := []processors.MastraDBMessage{
				createTLTestMessage("This is a very long message that will definitely exceed the token limit of 10 tokens and should trigger an abort", "assistant", "test-id"),
			}

			proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if !abortCalled {
				t.Fatal("expected abort to be called")
			}
		})
	})

	t.Run("processInput", func(t *testing.T) {
		t.Run("should limit input messages to the specified token count", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(50, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			messages := []processors.MastraDBMessage{
				createTLTestMessage("This is the first message with some content", "user", "message-1"),
				createTLTestMessage("This is a response with more content", "assistant", "message-2"),
				createTLTestMessage("Another message here", "user", "message-3"),
				createTLTestMessage("Final response", "assistant", "message-4"),
				createTLTestMessage("Latest message", "user", "message-5"),
			}

			result, _, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Should prioritize newest messages
			if len(result) >= len(messages) {
				t.Fatalf("expected fewer messages, got %d (original %d)", len(result), len(messages))
			}

			// Newest message should be included
			hasMsg5 := false
			hasMsg4 := false
			hasMsg1 := false
			for _, m := range result {
				if m.ID == "message-5" {
					hasMsg5 = true
				}
				if m.ID == "message-4" {
					hasMsg4 = true
				}
				if m.ID == "message-1" {
					hasMsg1 = true
				}
			}
			if !hasMsg5 {
				t.Fatal("expected message-5 (newest) to be included")
			}
			if !hasMsg4 {
				t.Fatal("expected message-4 to be included")
			}
			if hasMsg1 {
				t.Fatal("expected message-1 (oldest) to be excluded")
			}
		})

		t.Run("should throw error for empty messages array", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(1000, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			_, _, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         []processors.MastraDBMessage{},
				},
			})
			if err == nil {
				t.Fatal("expected error for empty messages")
			}
			if !strings.Contains(err.Error(), "No messages to process") {
				t.Fatalf("expected 'No messages to process' error, got '%s'", err.Error())
			}
		})

		t.Run("should handle tool call messages", func(t *testing.T) {
			proc := NewTokenLimiterProcessor(300, nil)
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			messages := []processors.MastraDBMessage{
				{
					ID:   "tool-call-1",
					Role: "assistant",
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MessagePart{
							{
								Type: "tool-invocation",
								ToolInvocationData: &processors.ToolInvocation{
									State:      "call",
									ToolCallID: "call_1",
									ToolName:   "calculator",
									Args:       map[string]any{"expression": "2+2"},
								},
							},
						},
					},
					CreatedAt: time.Now(),
				},
				{
					ID:   "tool-result-1",
					Role: "assistant",
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MessagePart{
							{
								Type: "tool-invocation",
								ToolInvocationData: &processors.ToolInvocation{
									State:      "result",
									ToolCallID: "call_1",
									ToolName:   "calculator",
									Args:       map[string]any{},
									Result:     "The result is 4",
								},
							},
						},
					},
					CreatedAt: time.Now(),
				},
				createTLTestMessage("Calculate 2+2", "user", "user-1"),
			}

			result, _, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
					Messages:         messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != 3 {
				t.Fatalf("expected 3 messages, got %d", len(result))
			}
		})
	})
}
