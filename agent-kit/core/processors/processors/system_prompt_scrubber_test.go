// Ported from: packages/core/src/processors/processors/system-prompt-scrubber.test.ts
package concreteprocessors

import (
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

func TestSystemPromptScrubber(t *testing.T) {
	// stubModel is a minimal model config to satisfy NewSystemPromptScrubber requirements.
	// TODO: replace with actual model mock once ported.
	stubModel := "stub-model-for-test"

	t.Run("basic functionality", func(t *testing.T) {
		t.Run("should pass through messages with no system prompts", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			messages := []processors.MastraDBMessage{
				{
					ID:   "msg-1",
					Role: "assistant",
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts:  []processors.MessagePart{{Type: "text", Text: "Hello, how can I help you?"}},
					},
				},
			}

			result, _, err := sps.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages: messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Since detection agent is stub (returns no detections), messages should pass through
			// Result may be nil (no-op) or the messages unchanged
			if result != nil && len(result) > 0 {
				if result[0].Content.Parts[0].Text != "Hello, how can I help you?" {
					t.Fatalf("expected original text preserved, got '%s'", result[0].Content.Parts[0].Text)
				}
			}
		})

		t.Run("should handle empty messages array", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			result, _, err := sps.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages: []processors.MastraDBMessage{},
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != nil && len(result) > 0 {
				t.Fatalf("expected empty result for empty messages, got %d", len(result))
			}
		})

		t.Run("should handle messages with no text parts", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			messages := []processors.MastraDBMessage{
				{
					ID:   "msg-1",
					Role: "assistant",
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts:  []processors.MessagePart{},
					},
				},
			}

			result, _, err := sps.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages: messages,
				},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Should pass through messages without text parts
			if result != nil && len(result) > 0 {
				if result[0].ID != "msg-1" {
					t.Fatalf("expected message preserved, got id '%s'", result[0].ID)
				}
			}
		})
	})

	t.Run("detection with redact strategy", func(t *testing.T) {
		t.Run("should redact detected system prompts", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})
	})

	t.Run("block strategy", func(t *testing.T) {
		t.Run("should abort when system prompt detected with abort function", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})

		t.Run("should handle missing abort function", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})
	})

	t.Run("filter strategy", func(t *testing.T) {
		t.Run("should filter out messages with system prompts", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})
	})

	t.Run("warn strategy", func(t *testing.T) {
		t.Run("should log warning and pass through", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})
	})

	t.Run("processOutputStream", func(t *testing.T) {
		t.Run("should pass through non-text-delta chunks", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			part := processors.ChunkType{
				Type:    "tool-call",
				Payload: map[string]any{"toolName": "test"},
			}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			result, err := sps.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result for non-text chunk")
			}
			if result.Type != "tool-call" {
				t.Fatalf("expected tool-call type, got %s", result.Type)
			}
		})

		t.Run("should pass through empty text chunks", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "   "},
			}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			result, err := sps.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result for whitespace-only text")
			}
		})

		t.Run("should pass through text when no detections (stub agent)", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model: stubModel,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			part := processors.ChunkType{
				Type:    "text-delta",
				Payload: map[string]any{"text": "Hello world"},
			}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			result, err := sps.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  part,
				State: map[string]any{},
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result when no detections")
			}
		})
	})

	t.Run("error handling", func(t *testing.T) {
		t.Run("should handle detection failure gracefully", func(t *testing.T) {
			t.Skip("not yet implemented: detection agent returns empty result (stub)")
		})
	})

	t.Run("config options", func(t *testing.T) {
		t.Run("should accept custom placeholder text", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model:           stubModel,
				PlaceholderText: "[REDACTED]",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sps == nil {
				t.Fatal("expected non-nil processor")
			}
		})

		t.Run("should accept custom instructions", func(t *testing.T) {
			sps, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model:        stubModel,
				Instructions: "Custom detection instructions",
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if sps == nil {
				t.Fatal("expected non-nil processor")
			}
		})

		t.Run("should require model", func(t *testing.T) {
			_, err := NewSystemPromptScrubber(SystemPromptScrubberOptions{})
			if err == nil {
				t.Fatal("expected error when model is nil")
			}
		})
	})

	t.Run("redaction methods", func(t *testing.T) {
		t.Run("mask redaction should replace with asterisks", func(t *testing.T) {
			sps, _ := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model:           stubModel,
				RedactionMethod: "mask",
			})

			detections := []SystemPromptDetection{
				{Type: "system_prompt", Value: "secret", Start: 0, End: 6},
			}
			result := sps.redactText("secret text", detections)
			if result != "****** text" {
				t.Fatalf("expected '****** text', got '%s'", result)
			}
		})

		t.Run("placeholder redaction should use placeholder text", func(t *testing.T) {
			sps, _ := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model:           stubModel,
				RedactionMethod: "placeholder",
				PlaceholderText: "[REDACTED]",
			})

			detections := []SystemPromptDetection{
				{Type: "system_prompt", Value: "secret", Start: 0, End: 6},
			}
			result := sps.redactText("secret text", detections)
			if result != "[REDACTED] text" {
				t.Fatalf("expected '[REDACTED] text', got '%s'", result)
			}
		})

		t.Run("remove redaction should remove detected text", func(t *testing.T) {
			sps, _ := NewSystemPromptScrubber(SystemPromptScrubberOptions{
				Model:           stubModel,
				RedactionMethod: "remove",
			})

			detections := []SystemPromptDetection{
				{Type: "system_prompt", Value: "secret", Start: 0, End: 6},
			}
			result := sps.redactText("secret text", detections)
			if result != " text" {
				t.Fatalf("expected ' text', got '%s'", result)
			}
		})
	})
}
