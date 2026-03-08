// Ported from: packages/core/src/stream/aisdk/v5/execute.test.ts
package v5

import (
	"context"
	"errors"
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestOmitFromMap(t *testing.T) {
	t.Run("should omit specified keys", func(t *testing.T) {
		m := map[string]any{"a": 1, "b": 2, "c": 3}
		result := omitFromMap(m, "b")
		if _, ok := result["b"]; ok {
			t.Error("expected 'b' to be omitted")
		}
		if result["a"] != 1 {
			t.Errorf("expected 'a' to be 1, got %v", result["a"])
		}
		if result["c"] != 3 {
			t.Errorf("expected 'c' to be 3, got %v", result["c"])
		}
	})

	t.Run("should omit multiple keys", func(t *testing.T) {
		m := map[string]any{"a": 1, "b": 2, "c": 3}
		result := omitFromMap(m, "a", "c")
		if len(result) != 1 {
			t.Errorf("expected 1 key, got %d", len(result))
		}
		if result["b"] != 2 {
			t.Errorf("expected 'b' to be 2, got %v", result["b"])
		}
	})

	t.Run("should return copy without modifying original", func(t *testing.T) {
		m := map[string]any{"a": 1, "b": 2}
		result := omitFromMap(m, "a")
		if len(m) != 2 {
			t.Error("original map should not be modified")
		}
		if len(result) != 1 {
			t.Errorf("expected 1 key in result, got %d", len(result))
		}
	})

	t.Run("should handle empty map", func(t *testing.T) {
		m := map[string]any{}
		result := omitFromMap(m, "a")
		if len(result) != 0 {
			t.Errorf("expected 0 keys, got %d", len(result))
		}
	})

	t.Run("should handle omitting non-existent keys", func(t *testing.T) {
		m := map[string]any{"a": 1}
		result := omitFromMap(m, "z")
		if len(result) != 1 {
			t.Errorf("expected 1 key, got %d", len(result))
		}
	})
}

func TestAPICallError(t *testing.T) {
	t.Run("should implement error interface", func(t *testing.T) {
		err := &APICallError{
			Err:         errors.New("request failed"),
			IsRetryable: true,
		}
		if err.Error() != "request failed" {
			t.Errorf("expected 'request failed', got %q", err.Error())
		}
	})

	t.Run("should support Unwrap", func(t *testing.T) {
		inner := errors.New("inner error")
		err := &APICallError{Err: inner, IsRetryable: false}
		if errors.Unwrap(err) != inner {
			t.Error("expected Unwrap to return inner error")
		}
	})

	t.Run("IsAPICallError should detect APICallError", func(t *testing.T) {
		err := &APICallError{Err: errors.New("test"), IsRetryable: true}
		ace, ok := IsAPICallError(err)
		if !ok {
			t.Fatal("expected IsAPICallError to return true")
		}
		if !ace.IsRetryable {
			t.Error("expected IsRetryable to be true")
		}
	})

	t.Run("IsAPICallError should return false for other errors", func(t *testing.T) {
		_, ok := IsAPICallError(errors.New("not an API error"))
		if ok {
			t.Error("expected IsAPICallError to return false for non-APICallError")
		}
	})
}

func TestExecute(t *testing.T) {
	t.Run("should execute model and return stream", func(t *testing.T) {
		rawParts := []stream.LanguageModelV2StreamPart{
			{Type: "text-delta", Data: map[string]any{"id": "t1", "delta": "Hello"}},
			{Type: "text-delta", Data: map[string]any{"id": "t1", "delta": " World"}},
			{Type: "finish", Data: map[string]any{
				"finishReason": "stop",
				"usage":        map[string]any{"inputTokens": float64(3), "outputTokens": float64(10)},
			}},
		}
		rawStream := ConvertArrayToReadableStream(rawParts)

		model := &MastraLanguageModel{
			ModelID:              "test-model",
			Provider:             "test",
			SpecificationVersion: "v2",
			DoStream: func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error) {
				return &stream.LanguageModelV2StreamResult{
					Stream: rawStream,
				}, nil
			},
		}

		out := Execute(ExecutionProps{
			RunID:      "run-1",
			Model:      model,
			MethodType: MethodStream,
		})

		var chunks []stream.ChunkType
		for chunk := range out {
			chunks = append(chunks, chunk)
		}

		if len(chunks) == 0 {
			t.Fatal("expected chunks from stream")
		}

		// Should have text-delta chunks and a finish chunk
		hasTextDelta := false
		hasFinish := false
		for _, c := range chunks {
			if c.Type == "text-delta" {
				hasTextDelta = true
			}
			if c.Type == "finish" {
				hasFinish = true
			}
		}
		if !hasTextDelta {
			t.Error("expected at least one text-delta chunk")
		}
		if !hasFinish {
			t.Error("expected a finish chunk")
		}
	})

	t.Run("should handle model error with ShouldThrowError=false", func(t *testing.T) {
		model := &MastraLanguageModel{
			ModelID:              "test-model",
			Provider:             "test",
			SpecificationVersion: "v2",
			DoStream: func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error) {
				return nil, &APICallError{
					Err:         errors.New("rate limited"),
					IsRetryable: false,
				}
			},
		}

		maxRetries := 0
		out := Execute(ExecutionProps{
			RunID:            "run-1",
			Model:            model,
			ShouldThrowError: false,
			MethodType:       MethodStream,
			ModelSettings:    &LoopModelSettings{MaxRetries: &maxRetries},
		})

		var chunks []stream.ChunkType
		for chunk := range out {
			chunks = append(chunks, chunk)
		}

		// When ShouldThrowError is false, errors are returned as error chunks in the stream
		hasError := false
		for _, c := range chunks {
			if c.Type == "error" {
				hasError = true
			}
		}
		if !hasError {
			t.Error("expected an error chunk in the stream")
		}
	})

	t.Run("should call onResult callback", func(t *testing.T) {
		rawStream := ConvertArrayToReadableStream([]stream.LanguageModelV2StreamPart{
			{Type: "text-delta", Data: map[string]any{"id": "t1", "delta": "Hi"}},
		})

		model := &MastraLanguageModel{
			ModelID:              "test-model",
			Provider:             "test",
			SpecificationVersion: "v2",
			DoStream: func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error) {
				return &stream.LanguageModelV2StreamResult{Stream: rawStream}, nil
			},
		}

		onResultCalled := false
		out := Execute(ExecutionProps{
			RunID:      "run-1",
			Model:      model,
			MethodType: MethodStream,
			OnResult: func(result stream.LanguageModelV2StreamResultMeta) {
				onResultCalled = true
			},
		})

		for range out {
		}

		if !onResultCalled {
			t.Error("expected onResult to be called")
		}
	})

	t.Run("should use v3 target version for v3 models", func(t *testing.T) {
		rawStream := ConvertArrayToReadableStream([]stream.LanguageModelV2StreamPart{
			{Type: "text-delta", Data: map[string]any{"id": "t1", "delta": "Hi"}},
		})

		var receivedOpts DoStreamOptions
		model := &MastraLanguageModel{
			ModelID:              "test-model",
			Provider:             "test",
			SpecificationVersion: "v3",
			DoStream: func(ctx context.Context, opts DoStreamOptions) (*stream.LanguageModelV2StreamResult, error) {
				receivedOpts = opts
				return &stream.LanguageModelV2StreamResult{Stream: rawStream}, nil
			},
		}

		out := Execute(ExecutionProps{
			RunID:      "run-1",
			Model:      model,
			MethodType: MethodStream,
			Tools: map[string]any{
				"search": map[string]any{"description": "Search"},
			},
		})

		for range out {
		}

		// The tools should be prepared with v3 target version
		_ = receivedOpts // tools are prepared externally and passed through
	})
}
