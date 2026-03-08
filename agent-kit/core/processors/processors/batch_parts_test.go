// Ported from: packages/core/src/processors/processors/batch-parts.test.ts
package concreteprocessors

import (
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

func makeTextDeltaChunk(text string) processors.ChunkType {
	return processors.ChunkType{
		Type:    "text-delta",
		Payload: map[string]any{"text": text, "id": "test-id"},
	}
}

func makeNonTextChunk(chunkType string) processors.ChunkType {
	return processors.ChunkType{
		Type:    chunkType,
		Payload: map[string]any{"data": "test"},
	}
}

func TestBatchPartsProcessor(t *testing.T) {
	t.Run("basic batching", func(t *testing.T) {
		t.Run("should batch parts up to the configured batch size", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 3})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// First two parts should be batched (return nil)
			for i := 0; i < 2; i++ {
				result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
					Part:  makeTextDeltaChunk("chunk"),
					State: state,
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != nil {
					t.Fatalf("expected nil for batched part %d, got %v", i, result)
				}
			}

			// Third part should trigger emission
			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("chunk3"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result when batch size reached")
			}
			if result.Type != "text-delta" {
				t.Fatalf("expected text-delta, got %s", result.Type)
			}
		})

		t.Run("should use default batch size of 5", func(t *testing.T) {
			bp := NewBatchPartsProcessor(nil)
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// First 4 parts should be batched (return nil)
			for i := 0; i < 4; i++ {
				result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
					Part:  makeTextDeltaChunk("chunk"),
					State: state,
					ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result != nil {
					t.Fatalf("expected nil for batched part %d, got %v", i, result)
				}
			}

			// 5th part should trigger emission
			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("chunk5"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result when batch size reached")
			}
		})
	})

	t.Run("non-text chunks", func(t *testing.T) {
		t.Run("should emit immediately on non-text chunks when emitOnNonText is true", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 5, EmitOnNonText: true})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// Batch one text part
			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("hello"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != nil {
				t.Fatal("expected nil for first batched text part")
			}

			// Non-text part should trigger flush of batched text
			result, err = bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeNonTextChunk("tool-call"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result on non-text chunk with emitOnNonText=true")
			}
			// Should have flushed the batched text part
			if result.Type != "text-delta" {
				t.Fatalf("expected text-delta from flush, got %s", result.Type)
			}
		})

		t.Run("should not emit immediately on non-text when emitOnNonText is false", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 5, EmitOnNonText: false})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// Non-text part should be batched like text parts
			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeNonTextChunk("tool-call"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != nil {
				t.Fatal("expected nil when emitOnNonText is false")
			}
		})

		t.Run("should handle mixed text and non-text chunks", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 3, EmitOnNonText: true})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// Text parts
			bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("hello"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk(" world"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			// Non-text should flush the batched text parts
			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeNonTextChunk("tool-call"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.Type != "text-delta" {
				t.Fatalf("expected flushed text-delta, got %s", result.Type)
			}
			// Check combined text
			if payload, ok := result.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					if text != "hello world" {
						t.Fatalf("expected combined text 'hello world', got '%s'", text)
					}
				}
			}
		})
	})

	t.Run("flush", func(t *testing.T) {
		t.Run("should flush remaining chunks", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 5})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			// Add some parts
			bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("hello"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk(" world"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			// Flush should return the remaining batched parts
			result := bp.Flush(state)
			if result == nil {
				t.Fatal("expected non-nil result from flush")
			}
			if result.Type != "text-delta" {
				t.Fatalf("expected text-delta from flush, got %s", result.Type)
			}
		})

		t.Run("should return nil when flushing empty batch", func(t *testing.T) {
			bp := NewBatchPartsProcessor(nil)
			state := map[string]any{}

			result := bp.Flush(state)
			if result != nil {
				t.Fatal("expected nil when flushing empty batch")
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should handle single part", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 1})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk("single"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result with batch size 1")
			}
		})

		t.Run("should handle empty text deltas", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 2})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk(""),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})

			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeTextDeltaChunk(""),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result at batch size")
			}
		})

		t.Run("should handle only non-text chunks with emitOnNonText true", func(t *testing.T) {
			bp := NewBatchPartsProcessor(&BatchPartsOptions{BatchSize: 5, EmitOnNonText: true})
			state := map[string]any{}
			mockAbort := func(reason string, opts *processors.TripWireOptions) error { return nil }

			result, err := bp.ProcessOutputStream(processors.ProcessOutputStreamArgs{
				Part:  makeNonTextChunk("tool-call"),
				State: state,
				ProcessorContext: processors.ProcessorContext{Abort: mockAbort},
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// With empty batch and emitOnNonText, should return the part directly
			if result == nil {
				t.Fatal("expected non-nil result for non-text with empty batch")
			}
			if result.Type != "tool-call" {
				t.Fatalf("expected tool-call, got %s", result.Type)
			}
		})
	})
}
