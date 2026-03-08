// Ported from: packages/core/src/stream/base/input.test.ts
package base

import (
	"sync"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestSafeEnqueue(t *testing.T) {
	t.Run("should enqueue a chunk to an open channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType, 1)
		chunk := stream.ChunkType{Type: "text-delta"}
		ok := SafeEnqueue(ch, chunk)
		if !ok {
			t.Fatal("expected SafeEnqueue to return true for open channel")
		}
		got := <-ch
		if got.Type != "text-delta" {
			t.Errorf("expected chunk type 'text-delta', got %q", got.Type)
		}
	})

	t.Run("should return false for a closed channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType, 1)
		close(ch)
		chunk := stream.ChunkType{Type: "text-delta"}
		ok := SafeEnqueue(ch, chunk)
		if ok {
			t.Fatal("expected SafeEnqueue to return false for closed channel")
		}
	})

	t.Run("should be safe to call concurrently", func(t *testing.T) {
		ch := make(chan stream.ChunkType, 100)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				SafeEnqueue(ch, stream.ChunkType{Type: "text-delta"})
			}()
		}
		wg.Wait()
		close(ch)
		count := 0
		for range ch {
			count++
		}
		if count != 100 {
			t.Errorf("expected 100 chunks enqueued, got %d", count)
		}
	})
}

func TestSafeClose(t *testing.T) {
	t.Run("should close an open channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType)
		ok := SafeClose(ch)
		if !ok {
			t.Fatal("expected SafeClose to return true for open channel")
		}
		// Verify channel is closed by receiving zero value
		_, open := <-ch
		if open {
			t.Fatal("expected channel to be closed")
		}
	})

	t.Run("should return false for an already closed channel", func(t *testing.T) {
		ch := make(chan stream.ChunkType)
		close(ch)
		ok := SafeClose(ch)
		if ok {
			t.Fatal("expected SafeClose to return false for already closed channel")
		}
	})
}

func TestSafeError(t *testing.T) {
	t.Run("should always return false (no-op in Go)", func(t *testing.T) {
		ch := make(chan stream.ChunkType)
		ok := SafeError(ch, nil)
		if ok {
			t.Fatal("expected SafeError to always return false")
		}
	})
}

func TestInitialize(t *testing.T) {
	t.Run("should create a stream that runs transform and closes", func(t *testing.T) {
		transformCalled := false
		onResultCalled := false

		mockInput := &mockMastraModelInput{
			transformFn: func(params TransformParams) error {
				transformCalled = true
				// Enqueue some chunks
				SafeEnqueue(params.Controller, stream.ChunkType{
					Type:    "text-delta",
					Payload: map[string]any{"text": "hello"},
				})
				return nil
			},
		}

		rawStream := make(chan stream.LanguageModelV2StreamPart, 1)
		close(rawStream) // close immediately for this test

		createStream := func() (*stream.LanguageModelV2StreamResult, error) {
			return &stream.LanguageModelV2StreamResult{
				Stream:   rawStream,
				Warnings: nil,
			}, nil
		}

		onResult := func(result stream.LanguageModelV2StreamResultMeta) {
			onResultCalled = true
		}

		out := Initialize(mockInput, "run-1", createStream, onResult)

		var chunks []stream.ChunkType
		for chunk := range out {
			chunks = append(chunks, chunk)
		}

		if !transformCalled {
			t.Error("expected transform to be called")
		}
		if !onResultCalled {
			t.Error("expected onResult to be called")
		}
		if len(chunks) != 1 {
			t.Fatalf("expected 1 chunk, got %d", len(chunks))
		}
		if chunks[0].Type != "text-delta" {
			t.Errorf("expected chunk type 'text-delta', got %q", chunks[0].Type)
		}
	})

	t.Run("should close channel if createStream returns error", func(t *testing.T) {
		mockInput := &mockMastraModelInput{
			transformFn: func(params TransformParams) error {
				t.Error("transform should not be called when createStream fails")
				return nil
			},
		}

		createStream := func() (*stream.LanguageModelV2StreamResult, error) {
			return nil, &testError{"create stream failed"}
		}

		onResult := func(result stream.LanguageModelV2StreamResultMeta) {
			t.Error("onResult should not be called when createStream fails")
		}

		out := Initialize(mockInput, "run-1", createStream, onResult)

		var chunks []stream.ChunkType
		for chunk := range out {
			chunks = append(chunks, chunk)
		}
		if len(chunks) != 0 {
			t.Errorf("expected 0 chunks when createStream fails, got %d", len(chunks))
		}
	})
}

// mockMastraModelInput is a mock implementation of MastraModelInput.
type mockMastraModelInput struct {
	transformFn func(params TransformParams) error
}

func (m *mockMastraModelInput) Transform(params TransformParams) error {
	if m.transformFn != nil {
		return m.transformFn(params)
	}
	return nil
}

// testError is a simple error type for tests.
type testError struct {
	msg string
}

func (e *testError) Error() string { return e.msg }
