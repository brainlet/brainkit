// Ported from: packages/core/src/stream/aisdk/v5/compat/consume-stream.test.ts
package compat

import (
	"testing"
)

func TestConsumeStreamCompat(t *testing.T) {
	t.Run("should drain all values from channel", func(t *testing.T) {
		ch := make(chan any, 3)
		ch <- "a"
		ch <- "b"
		ch <- "c"
		close(ch)

		// Should not panic or block
		ConsumeStream(ch, nil)
	})

	t.Run("should handle empty channel", func(t *testing.T) {
		ch := make(chan any)
		close(ch)

		ConsumeStream(ch, nil)
	})

	t.Run("should work with onError option", func(t *testing.T) {
		ch := make(chan any, 1)
		ch <- "value"
		close(ch)

		errorCalled := false
		ConsumeStream(ch, &ConsumeStreamOptions{
			OnError: func(err error) {
				errorCalled = true
			},
		})

		if errorCalled {
			t.Error("onError should not be called for normal stream")
		}
	})

	t.Run("should handle nil options", func(t *testing.T) {
		ch := make(chan any, 2)
		ch <- 1
		ch <- 2
		close(ch)

		ConsumeStream(ch, nil)
	})
}
