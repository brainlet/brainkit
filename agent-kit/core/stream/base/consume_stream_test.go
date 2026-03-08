// Ported from: packages/core/src/stream/base/consume-stream.test.ts
package base

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestConsumeStream(t *testing.T) {
	t.Run("should drain all chunks from channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType, 3)
		ch <- stream.ChunkType{Type: "text-delta", Payload: map[string]any{"text": "a"}}
		ch <- stream.ChunkType{Type: "text-delta", Payload: map[string]any{"text": "b"}}
		ch <- stream.ChunkType{Type: "finish"}
		close(ch)

		// Should not panic and should drain all chunks
		ConsumeStream(ch, nil)
	})

	t.Run("should work with onError option", func(t *testing.T) {
		ch := make(chan stream.ChunkType, 1)
		ch <- stream.ChunkType{Type: "text-delta"}
		close(ch)

		errorCalled := false
		ConsumeStream(ch, &ConsumeStreamOptions{
			OnError: func(err error) {
				errorCalled = true
			},
		})

		// onError should not be called for normal stream processing
		if errorCalled {
			t.Error("expected onError not to be called for normal stream")
		}
	})

	t.Run("should handle empty channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType)
		close(ch)

		// Should not block or panic
		ConsumeStream(ch, nil)
	})
}
