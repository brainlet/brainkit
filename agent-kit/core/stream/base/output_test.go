// Ported from: packages/core/src/stream/base/output.test.ts
package base

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func makeTestStream(chunks []stream.ChunkType) <-chan stream.ChunkType {
	ch := make(chan stream.ChunkType, len(chunks))
	for _, c := range chunks {
		ch <- c
	}
	close(ch)
	return ch
}

func TestNewEmptyLLMStepResult(t *testing.T) {
	t.Run("should create step result with initialized slices", func(t *testing.T) {
		result := NewEmptyLLMStepResult()
		if result.Reasoning == nil {
			t.Error("expected Reasoning to be initialized")
		}
		if result.Sources == nil {
			t.Error("expected Sources to be initialized")
		}
		if result.Files == nil {
			t.Error("expected Files to be initialized")
		}
		if result.ToolCalls == nil {
			t.Error("expected ToolCalls to be initialized")
		}
		if result.ToolResults == nil {
			t.Error("expected ToolResults to be initialized")
		}
		if result.Content == nil {
			t.Error("expected Content to be initialized")
		}
		if result.Warnings == nil {
			t.Error("expected Warnings to be initialized")
		}
		if result.Request == nil {
			t.Error("expected Request to be initialized")
		}
		if result.Response == nil {
			t.Error("expected Response to be initialized")
		}
	})
}

func TestMastraModelOutput(t *testing.T) {
	t.Run("should accumulate text from text-delta chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "text-delta",
				Payload:       map[string]any{"text": "Hello", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "text-delta",
				Payload:       map[string]any{"text": " world", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		text, err := output.AwaitText()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if text != "Hello world" {
			t.Errorf("expected 'Hello world', got %q", text)
		}
	})

	t.Run("should accumulate tool calls", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "tool-call",
				Payload: map[string]any{
					"toolCallId": "call-1",
					"toolName":   "get_weather",
					"args":       map[string]any{"location": "NYC"},
				},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "tool-calls"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		toolCalls, err := output.AwaitToolCalls()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}
		if toolCalls[0].Type != "tool-call" {
			t.Errorf("expected type 'tool-call', got %q", toolCalls[0].Type)
		}
	})

	t.Run("should handle error chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "error",
				Payload: map[string]any{
					"error": "something went wrong",
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		output.WaitForFinish()

		if output.Status() != "failed" {
			t.Errorf("expected status 'failed', got %q", output.Status())
		}
		if output.Error() == nil {
			t.Error("expected error to be set")
		}
	})

	t.Run("should handle tripwire chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "tripwire",
				Payload: map[string]any{
					"reason": "content blocked",
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		output.WaitForFinish()

		tripwire := output.Tripwire()
		if tripwire == nil {
			t.Fatal("expected tripwire to be set")
		}
		if tripwire.Reason != "content blocked" {
			t.Errorf("expected reason 'content blocked', got %q", tripwire.Reason)
		}
	})

	t.Run("should handle source chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "source",
				Payload: map[string]any{
					"id":    "source-1",
					"url":   "https://example.com",
					"title": "Example",
				},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		sources, err := output.AwaitSources()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(sources) != 1 {
			t.Fatalf("expected 1 source, got %d", len(sources))
		}
	})

	t.Run("should handle file chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "file",
				Payload:       map[string]any{"data": "base64data", "mimeType": "image/png"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		files, err := output.AwaitFiles()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(files) != 1 {
			t.Fatalf("expected 1 file, got %d", len(files))
		}
	})

	t.Run("should filter raw chunks when IncludeRawChunks is false", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "raw",
				Payload:       map[string]any{"data": "raw data"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "text-delta",
				Payload:       map[string]any{"text": "hello", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID:            "run-1",
				IncludeRawChunks: false,
			},
		})

		fullStream := output.FullStream()
		var chunks []stream.ChunkType
		for chunk := range fullStream {
			chunks = append(chunks, chunk)
		}

		for _, chunk := range chunks {
			if chunk.Type == "raw" {
				t.Error("raw chunks should be filtered when IncludeRawChunks is false")
			}
		}
	})

	t.Run("should handle suspended status on tool-call-suspended", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "tool-call-suspended",
				Payload:       map[string]any{"toolCallId": "call-1"},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		output.WaitForFinish()

		if output.Status() != "suspended" {
			t.Errorf("expected status 'suspended', got %q", output.Status())
		}
	})

	t.Run("FullStream should replay buffered chunks", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "text-delta",
				Payload:       map[string]any{"text": "hello", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		// Wait for the stream to finish processing
		output.WaitForFinish()

		// Now get the full stream — should replay all buffered chunks
		fullStream := output.FullStream()
		var chunks []stream.ChunkType
		for chunk := range fullStream {
			chunks = append(chunks, chunk)
		}

		if len(chunks) < 1 {
			t.Error("expected at least 1 replayed chunk from FullStream")
		}
	})

	t.Run("ImmediateText should return currently buffered text", func(t *testing.T) {
		inputStream := make(chan stream.ChunkType, 10)
		inputStream <- stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
			Type:          "text-delta",
			Payload:       map[string]any{"text": "hello", "id": "t1"},
		}

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		// Give time for the chunk to be processed
		time.Sleep(50 * time.Millisecond)

		text := output.ImmediateText()
		if text != "hello" {
			t.Errorf("expected 'hello', got %q", text)
		}

		// Close to finish
		close(inputStream)
		output.WaitForFinish()
	})

	t.Run("GetFullOutput should return complete output", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "text-delta",
				Payload:       map[string]any{"text": "Hello world", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "step-finish",
				Payload: &stream.StepFinishPayload{
					StepResult: stream.StepFinishPayloadStepResult{
						Reason: "stop",
					},
					Output: stream.StepFinishPayloadOutput{
						Usage: stream.LanguageModelUsage{
							InputTokens:  10,
							OutputTokens: 5,
							TotalTokens:  15,
						},
					},
				},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1", From: stream.ChunkFromAgent},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output": map[string]any{
						"usage": map[string]any{
							"inputTokens":  10,
							"outputTokens": 5,
							"totalTokens":  15,
						},
					},
					"metadata": map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		fullOutput, err := output.GetFullOutput()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fullOutput.Text != "Hello world" {
			t.Errorf("expected text 'Hello world', got %q", fullOutput.Text)
		}
		if fullOutput.RunID != "run-1" {
			t.Errorf("expected runID 'run-1', got %q", fullOutput.RunID)
		}
	})

	t.Run("SerializeState should return current state", func(t *testing.T) {
		inputStream := makeTestStream([]stream.ChunkType{
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "text-delta",
				Payload:       map[string]any{"text": "test", "id": "t1"},
			},
			{
				BaseChunkType: stream.BaseChunkType{RunID: "run-1"},
				Type:          "finish",
				Payload: map[string]any{
					"stepResult": map[string]any{"reason": "stop"},
					"output":     map[string]any{"usage": map[string]any{}},
					"metadata":   map[string]any{},
				},
			},
		})

		output := NewMastraModelOutput(MastraModelOutputParams{
			Stream: inputStream,
			Options: MastraModelOutputOptions{
				RunID: "run-1",
			},
		})

		output.WaitForFinish()

		state := output.SerializeState()
		if state == nil {
			t.Fatal("expected non-nil state")
		}
		if state["status"] != "success" {
			t.Errorf("expected status 'success', got %v", state["status"])
		}
	})
}

func TestExtractTextFromPayload(t *testing.T) {
	t.Run("should extract text from map payload", func(t *testing.T) {
		result := extractTextFromPayload(map[string]any{"text": "hello"})
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("should extract text from TextDeltaPayload pointer", func(t *testing.T) {
		result := extractTextFromPayload(&stream.TextDeltaPayload{Text: "hello"})
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("should extract text from TextDeltaPayload value", func(t *testing.T) {
		result := extractTextFromPayload(stream.TextDeltaPayload{Text: "hello"})
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("should return empty string for nil", func(t *testing.T) {
		result := extractTextFromPayload(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestExtractIDFromPayload(t *testing.T) {
	t.Run("should extract id from map payload", func(t *testing.T) {
		result := extractIDFromPayload(map[string]any{"id": "test-id"})
		if result != "test-id" {
			t.Errorf("expected 'test-id', got %q", result)
		}
	})

	t.Run("should return empty string for nil", func(t *testing.T) {
		result := extractIDFromPayload(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestIntFromAny(t *testing.T) {
	t.Run("should convert float64", func(t *testing.T) {
		result := intFromAny(float64(42))
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("should convert int", func(t *testing.T) {
		result := intFromAny(42)
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("should convert int64", func(t *testing.T) {
		result := intFromAny(int64(42))
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("should return 0 for nil", func(t *testing.T) {
		result := intFromAny(nil)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("should return 0 for string", func(t *testing.T) {
		result := intFromAny("not a number")
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})
}

func TestStringFromAny(t *testing.T) {
	t.Run("should return string directly", func(t *testing.T) {
		result := stringFromAny("hello")
		if result != "hello" {
			t.Errorf("expected 'hello', got %q", result)
		}
	})

	t.Run("should return empty string for nil", func(t *testing.T) {
		result := stringFromAny(nil)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should convert non-string to string", func(t *testing.T) {
		result := stringFromAny(42)
		if result != "42" {
			t.Errorf("expected '42', got %q", result)
		}
	})
}

func TestUsageFromMap(t *testing.T) {
	t.Run("should convert usage map to LanguageModelUsage", func(t *testing.T) {
		m := map[string]any{
			"inputTokens":      float64(10),
			"outputTokens":     float64(20),
			"totalTokens":      float64(30),
			"reasoningTokens":  float64(5),
			"cachedInputTokens": float64(3),
		}
		result := usageFromMap(m)
		if result.InputTokens != 10 {
			t.Errorf("expected InputTokens 10, got %d", result.InputTokens)
		}
		if result.OutputTokens != 20 {
			t.Errorf("expected OutputTokens 20, got %d", result.OutputTokens)
		}
		if result.TotalTokens != 30 {
			t.Errorf("expected TotalTokens 30, got %d", result.TotalTokens)
		}
		if result.ReasoningTokens != 5 {
			t.Errorf("expected ReasoningTokens 5, got %d", result.ReasoningTokens)
		}
		if result.CachedInputTokens != 3 {
			t.Errorf("expected CachedInputTokens 3, got %d", result.CachedInputTokens)
		}
	})
}
