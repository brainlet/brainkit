// Ported from: packages/core/src/stream/aisdk/v4/input.test.ts
package v4

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestAISDKV4InputStream(t *testing.T) {
	t.Run("should create with options", func(t *testing.T) {
		s := NewAISDKV4InputStream(AISDKV4InputStreamOptions{
			Component: "LLM",
			Name:      "test-model",
		})
		if s.Component != "LLM" {
			t.Errorf("expected component 'LLM', got %q", s.Component)
		}
		if s.Name != "test-model" {
			t.Errorf("expected name 'test-model', got %q", s.Name)
		}
	})

	t.Run("TransformV4 should convert V4 stream parts to Mastra chunks", func(t *testing.T) {
		s := NewAISDKV4InputStream(AISDKV4InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		v4Stream := make(chan LanguageModelV1StreamPart, 3)
		v4Stream <- LanguageModelV1StreamPart{
			Type:      "text-delta",
			ID:        "t1",
			TextDelta: "Hello",
		}
		v4Stream <- LanguageModelV1StreamPart{
			Type:      "text-delta",
			ID:        "t1",
			TextDelta: " World",
		}
		v4Stream <- LanguageModelV1StreamPart{
			Type:         "step-finish",
			FinishReason: "stop",
		}
		close(v4Stream)

		controller := make(chan stream.ChunkType, 10)

		err := s.TransformV4(V4TransformParams{
			RunID:      "run-1",
			Stream:     v4Stream,
			Controller: controller,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		close(controller)

		var chunks []stream.ChunkType
		for c := range controller {
			chunks = append(chunks, c)
		}

		if len(chunks) != 3 {
			t.Fatalf("expected 3 chunks, got %d", len(chunks))
		}
		if chunks[0].Type != "text-delta" {
			t.Errorf("expected first chunk type 'text-delta', got %q", chunks[0].Type)
		}
		if chunks[1].Type != "text-delta" {
			t.Errorf("expected second chunk type 'text-delta', got %q", chunks[1].Type)
		}
		if chunks[2].Type != "step-finish" {
			t.Errorf("expected third chunk type 'step-finish', got %q", chunks[2].Type)
		}
	})

	t.Run("TransformV4 should skip unknown types", func(t *testing.T) {
		s := NewAISDKV4InputStream(AISDKV4InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		v4Stream := make(chan LanguageModelV1StreamPart, 2)
		v4Stream <- LanguageModelV1StreamPart{Type: "unknown-type"}
		v4Stream <- LanguageModelV1StreamPart{
			Type:      "text-delta",
			TextDelta: "Hello",
		}
		close(v4Stream)

		controller := make(chan stream.ChunkType, 10)
		err := s.TransformV4(V4TransformParams{
			RunID:      "run-1",
			Stream:     v4Stream,
			Controller: controller,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		close(controller)

		var chunks []stream.ChunkType
		for c := range controller {
			chunks = append(chunks, c)
		}

		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk (unknown skipped), got %d", len(chunks))
		}
		if chunks[0].Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", chunks[0].Type)
		}
	})

	t.Run("TransformV4 should handle empty stream", func(t *testing.T) {
		s := NewAISDKV4InputStream(AISDKV4InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		v4Stream := make(chan LanguageModelV1StreamPart)
		close(v4Stream)

		controller := make(chan stream.ChunkType, 10)
		err := s.TransformV4(V4TransformParams{
			RunID:      "run-1",
			Stream:     v4Stream,
			Controller: controller,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		close(controller)

		count := 0
		for range controller {
			count++
		}
		if count != 0 {
			t.Errorf("expected 0 chunks, got %d", count)
		}
	})

	t.Run("TransformV4 should set runID on all chunks", func(t *testing.T) {
		s := NewAISDKV4InputStream(AISDKV4InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		v4Stream := make(chan LanguageModelV1StreamPart, 2)
		v4Stream <- LanguageModelV1StreamPart{Type: "text-delta", TextDelta: "a"}
		v4Stream <- LanguageModelV1StreamPart{Type: "text-delta", TextDelta: "b"}
		close(v4Stream)

		controller := make(chan stream.ChunkType, 10)
		err := s.TransformV4(V4TransformParams{
			RunID:      "run-42",
			Stream:     v4Stream,
			Controller: controller,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		close(controller)

		for c := range controller {
			if c.RunID != "run-42" {
				t.Errorf("expected runID 'run-42', got %q", c.RunID)
			}
		}
	})
}
