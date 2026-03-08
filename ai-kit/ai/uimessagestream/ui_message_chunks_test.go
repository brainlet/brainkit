// Ported from: packages/ai/src/ui-message-stream/ui-message-chunks.test-d.ts
//
// The original TypeScript file is a type-level test using expectTypeOf.
// In Go, we verify that UIMessageChunk serialization/deserialization works
// correctly and that IsDataUIMessageChunk functions properly.
package uimessagestream

import (
	"encoding/json"
	"testing"
)

func TestUIMessageChunks(t *testing.T) {
	t.Run("should round-trip a text-delta chunk through JSON", func(t *testing.T) {
		chunk := UIMessageChunk{
			Type:  "text-delta",
			ID:    "123",
			Delta: "Hello, world!",
		}

		data, err := json.Marshal(chunk)
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}

		var parsed UIMessageChunk
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if parsed.Type != "text-delta" {
			t.Errorf("expected type text-delta, got %q", parsed.Type)
		}
		if parsed.ID != "123" {
			t.Errorf("expected id 123, got %q", parsed.ID)
		}
		if parsed.Delta != "Hello, world!" {
			t.Errorf("expected delta 'Hello, world!', got %q", parsed.Delta)
		}
	})

	t.Run("IsDataUIMessageChunk should detect data- prefixed types", func(t *testing.T) {
		dataChunk := UIMessageChunk{Type: "data-custom", Data: map[string]any{"key": "value"}}
		if !IsDataUIMessageChunk(dataChunk) {
			t.Error("expected data chunk to be detected")
		}

		textChunk := UIMessageChunk{Type: "text-delta", Delta: "hello"}
		if IsDataUIMessageChunk(textChunk) {
			t.Error("expected text chunk NOT to be detected as data chunk")
		}
	})

	t.Run("should marshal all chunk types without error", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "text-start", ID: "1"},
			{Type: "text-delta", ID: "1", Delta: "hello"},
			{Type: "text-end", ID: "1"},
			{Type: "error", ErrorText: "an error"},
			{Type: "tool-input-start", ToolCallID: "tc-1", ToolName: "myTool"},
			{Type: "tool-input-delta", ToolCallID: "tc-1", InputTextDelta: `{"key":`},
			{Type: "tool-input-available", ToolCallID: "tc-1", ToolName: "myTool", Input: map[string]any{"key": "val"}},
			{Type: "tool-output-available", ToolCallID: "tc-1", Output: "result"},
			{Type: "tool-output-error", ToolCallID: "tc-1", ErrorText: "tool error"},
			{Type: "tool-output-denied", ToolCallID: "tc-1"},
			{Type: "tool-approval-request", ToolCallID: "tc-1", ApprovalID: "ap-1"},
			{Type: "reasoning-start", ID: "r-1"},
			{Type: "reasoning-delta", ID: "r-1", Delta: "thinking..."},
			{Type: "reasoning-end", ID: "r-1"},
			{Type: "source-url", SourceID: "s-1", URL: "https://example.com"},
			{Type: "source-document", SourceID: "s-2", MediaType: "text/plain", Title: "doc"},
			{Type: "file", URL: "https://example.com/file.png", MediaType: "image/png"},
			{Type: "data-custom", Data: map[string]any{"x": 1}},
			{Type: "start-step"},
			{Type: "finish-step"},
			{Type: "start", MessageID: "msg-1"},
			{Type: "finish", FinishReason: "stop"},
			{Type: "abort", Reason: "user cancelled"},
			{Type: "message-metadata", MessageMetadata: map[string]any{"foo": "bar"}},
		}

		for _, chunk := range chunks {
			data, err := json.Marshal(chunk)
			if err != nil {
				t.Errorf("marshal error for type %q: %v", chunk.Type, err)
				continue
			}
			var parsed UIMessageChunk
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Errorf("unmarshal error for type %q: %v", chunk.Type, err)
			}
			if parsed.Type != chunk.Type {
				t.Errorf("expected type %q after round-trip, got %q", chunk.Type, parsed.Type)
			}
		}
	})
}
