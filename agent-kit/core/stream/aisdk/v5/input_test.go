// Ported from: packages/core/src/stream/aisdk/v5/input.test.ts
package v5

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
	"github.com/brainlet/brainkit/agent-kit/core/stream/base"
)

func TestIsNumericID(t *testing.T) {
	t.Run("should return true for numeric strings", func(t *testing.T) {
		if !isNumericID("0") {
			t.Error("expected true for '0'")
		}
		if !isNumericID("1") {
			t.Error("expected true for '1'")
		}
		if !isNumericID("123") {
			t.Error("expected true for '123'")
		}
		if !isNumericID("999") {
			t.Error("expected true for '999'")
		}
	})

	t.Run("should return false for non-numeric strings", func(t *testing.T) {
		if isNumericID("abc") {
			t.Error("expected false for 'abc'")
		}
		if isNumericID("1a2b") {
			t.Error("expected false for '1a2b'")
		}
		if isNumericID("") {
			t.Error("expected false for empty string")
		}
		if isNumericID("id-1") {
			t.Error("expected false for 'id-1'")
		}
	})

	t.Run("should return false for UUIDs", func(t *testing.T) {
		if isNumericID("550e8400-e29b-41d4-a716-446655440000") {
			t.Error("expected false for UUID")
		}
	})
}

func TestPayloadIDHelpers(t *testing.T) {
	t.Run("hasPayloadID should return true when ID exists", func(t *testing.T) {
		chunk := &stream.ChunkType{
			Type:    "text-delta",
			Payload: map[string]any{"id": "t1"},
		}
		if !hasPayloadID(chunk) {
			t.Error("expected true for chunk with id")
		}
	})

	t.Run("hasPayloadID should return false when payload is nil", func(t *testing.T) {
		chunk := &stream.ChunkType{Type: "text-delta"}
		if hasPayloadID(chunk) {
			t.Error("expected false for nil payload")
		}
	})

	t.Run("hasPayloadID should return false when no id in payload", func(t *testing.T) {
		chunk := &stream.ChunkType{
			Type:    "text-delta",
			Payload: map[string]any{"text": "hello"},
		}
		if hasPayloadID(chunk) {
			t.Error("expected false for payload without id")
		}
	})

	t.Run("getPayloadID should return the ID", func(t *testing.T) {
		chunk := &stream.ChunkType{
			Type:    "text-delta",
			Payload: map[string]any{"id": "t1"},
		}
		if getPayloadID(chunk) != "t1" {
			t.Errorf("expected 't1', got %q", getPayloadID(chunk))
		}
	})

	t.Run("getPayloadID should return empty for nil payload", func(t *testing.T) {
		chunk := &stream.ChunkType{Type: "text-delta"}
		if getPayloadID(chunk) != "" {
			t.Errorf("expected empty, got %q", getPayloadID(chunk))
		}
	})

	t.Run("setPayloadID should update the ID", func(t *testing.T) {
		chunk := &stream.ChunkType{
			Type:    "text-delta",
			Payload: map[string]any{"id": "old"},
		}
		setPayloadID(chunk, "new")
		if getPayloadID(chunk) != "new" {
			t.Errorf("expected 'new', got %q", getPayloadID(chunk))
		}
	})
}

func TestAISDKV5InputStream(t *testing.T) {
	t.Run("should create with default ID generator", func(t *testing.T) {
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})
		if s.Component != "LLM" {
			t.Errorf("expected component 'LLM', got %q", s.Component)
		}
		if s.Name != "test" {
			t.Errorf("expected name 'test', got %q", s.Name)
		}
		if s.generateID == nil {
			t.Error("expected generateID to be set")
		}
	})

	t.Run("should create with custom ID generator", func(t *testing.T) {
		counter := 0
		gen := func() string {
			counter++
			return "custom-" + string(rune('0'+counter))
		}
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component:  "LLM",
			Name:       "test",
			GenerateID: gen,
		})
		if s.generateID == nil {
			t.Error("expected generateID to be set")
		}
	})

	t.Run("should transform stream chunks", func(t *testing.T) {
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		rawStream := make(chan stream.LanguageModelV2StreamPart, 3)
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{"id": "uuid-1", "delta": "Hello"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{"id": "uuid-1", "delta": " World"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-end",
			Data: map[string]any{"id": "uuid-1"},
		}
		close(rawStream)

		controller := make(chan stream.ChunkType, 10)

		err := s.Transform(base.TransformParams{
			RunID:      "run-1",
			Stream:     rawStream,
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
	})

	t.Run("should replace numeric IDs with unique IDs", func(t *testing.T) {
		idCounter := 0
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
			GenerateID: func() string {
				idCounter++
				return "unique-" + string(rune('0'+idCounter))
			},
		})

		rawStream := make(chan stream.LanguageModelV2StreamPart, 3)
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-start",
			Data: map[string]any{"id": "0"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{"id": "0", "delta": "Hello"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-end",
			Data: map[string]any{"id": "0"},
		}
		close(rawStream)

		controller := make(chan stream.ChunkType, 10)
		err := s.Transform(base.TransformParams{
			RunID:      "run-1",
			Stream:     rawStream,
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

		// All three chunks should have the same unique ID (not "0")
		for i, chunk := range chunks {
			id := getPayloadID(&chunk)
			if id == "0" {
				t.Errorf("chunk %d: expected numeric ID to be replaced", i)
			}
			if id == "" {
				t.Errorf("chunk %d: expected non-empty ID", i)
			}
		}

		// All chunks should share the same generated ID
		id0 := getPayloadID(&chunks[0])
		id1 := getPayloadID(&chunks[1])
		id2 := getPayloadID(&chunks[2])
		if id0 != id1 || id1 != id2 {
			t.Errorf("expected all chunks to share same ID, got %q, %q, %q", id0, id1, id2)
		}
	})

	t.Run("should reset ID map on stream-start", func(t *testing.T) {
		idCounter := 0
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
			GenerateID: func() string {
				idCounter++
				return "gen-" + string(rune('0'+idCounter))
			},
		})

		rawStream := make(chan stream.LanguageModelV2StreamPart, 5)
		// First step
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-start",
			Data: map[string]any{"id": "0"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-end",
			Data: map[string]any{"id": "0"},
		}
		// New step with stream-start
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "stream-start",
			Data: map[string]any{},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-start",
			Data: map[string]any{"id": "0"},
		}
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-end",
			Data: map[string]any{"id": "0"},
		}
		close(rawStream)

		controller := make(chan stream.ChunkType, 10)
		err := s.Transform(base.TransformParams{
			RunID:      "run-1",
			Stream:     rawStream,
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

		// First step's "0" gets one generated ID, second step gets a different one
		firstID := getPayloadID(&chunks[0])
		// Find the first text-start after stream-start (skip stream-start which returns nil from convert)
		var secondID string
		for _, chunk := range chunks[2:] {
			if chunk.Type == "text-start" {
				secondID = getPayloadID(&chunk)
				break
			}
		}
		if firstID == secondID && firstID != "" {
			t.Errorf("expected different IDs after stream-start reset, both got %q", firstID)
		}
	})

	t.Run("should not replace non-numeric IDs", func(t *testing.T) {
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
			GenerateID: func() string {
				return "should-not-be-used"
			},
		})

		rawStream := make(chan stream.LanguageModelV2StreamPart, 1)
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{"id": "uuid-123", "delta": "Hi"},
		}
		close(rawStream)

		controller := make(chan stream.ChunkType, 10)
		err := s.Transform(base.TransformParams{
			RunID:      "run-1",
			Stream:     rawStream,
			Controller: controller,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		close(controller)

		chunk := <-controller
		id := getPayloadID(&chunk)
		if id != "uuid-123" {
			t.Errorf("expected UUID to be preserved, got %q", id)
		}
	})
}

func TestAISDKV5InputStreamInitialize(t *testing.T) {
	t.Run("should wire up stream creation pipeline", func(t *testing.T) {
		s := NewAISDKV5InputStream(AISDKV5InputStreamOptions{
			Component: "LLM",
			Name:      "test",
		})

		rawStream := make(chan stream.LanguageModelV2StreamPart, 2)
		rawStream <- stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{"id": "t1", "delta": "Hello"},
		}
		close(rawStream)

		onResultCalled := false
		out := s.Initialize(InitializeParams{
			RunID: "run-1",
			CreateStream: func() (*stream.LanguageModelV2StreamResult, error) {
				return &stream.LanguageModelV2StreamResult{
					Stream: rawStream,
				}, nil
			},
			OnResult: func(result stream.LanguageModelV2StreamResultMeta) {
				onResultCalled = true
			},
		})

		var chunks []stream.ChunkType
		for chunk := range out {
			chunks = append(chunks, chunk)
		}

		if len(chunks) == 0 {
			t.Error("expected at least 1 chunk")
		}
		if !onResultCalled {
			t.Error("expected onResult to be called")
		}
	})
}
