// Ported from: packages/core/src/stream/aisdk/v5/compat/ui-message.test.ts
package compat

import (
	"strings"
	"testing"
)

func TestGetResponseUIMessageId(t *testing.T) {
	t.Run("should return empty when no original messages", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages: nil,
		})
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should return last assistant message ID", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages: []UIMessage{
				{ID: "user-1", Role: "user"},
				{ID: "assistant-1", Role: "assistant"},
			},
		})
		if result != "assistant-1" {
			t.Errorf("expected 'assistant-1', got %q", result)
		}
	})

	t.Run("should use string responseMessageId when last message is not assistant", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages:  []UIMessage{{ID: "user-1", Role: "user"}},
			ResponseMessageID: "custom-id",
		})
		if result != "custom-id" {
			t.Errorf("expected 'custom-id', got %q", result)
		}
	})

	t.Run("should use generator function for responseMessageId", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages:  []UIMessage{{ID: "user-1", Role: "user"}},
			ResponseMessageID: IdGeneratorFn(func() string { return "generated-id" }),
		})
		if result != "generated-id" {
			t.Errorf("expected 'generated-id', got %q", result)
		}
	})

	t.Run("should use func() string for responseMessageId", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages:  []UIMessage{{ID: "user-1", Role: "user"}},
			ResponseMessageID: func() string { return "func-id" },
		})
		if result != "func-id" {
			t.Errorf("expected 'func-id', got %q", result)
		}
	})

	t.Run("should return empty for empty messages slice", func(t *testing.T) {
		result := GetResponseUIMessageId(GetResponseUIMessageIdParams{
			OriginalMessages: []UIMessage{},
		})
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestConvertFullStreamChunkToUIMessageStream(t *testing.T) {
	t.Run("text-start should produce text-start chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "text-start", ID: "t1"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-start" {
			t.Errorf("expected type 'text-start', got %q", result.Type)
		}
		if result.ID != "t1" {
			t.Errorf("expected ID 't1', got %q", result.ID)
		}
	})

	t.Run("text-delta should produce text-delta chunk with delta", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "text-delta", ID: "t1", Text: "Hello"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", result.Type)
		}
		if result.Delta != "Hello" {
			t.Errorf("expected delta 'Hello', got %q", result.Delta)
		}
	})

	t.Run("text-end should produce text-end chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "text-end", ID: "t1"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-end" {
			t.Errorf("expected type 'text-end', got %q", result.Type)
		}
	})

	t.Run("reasoning-delta should return nil when sendReasoning is false", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:          TextStreamPart{Type: "reasoning-delta", Text: "thinking..."},
			SendReasoning: false,
		})
		if result != nil {
			t.Error("expected nil when sendReasoning is false")
		}
	})

	t.Run("reasoning-delta should return chunk when sendReasoning is true", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:          TextStreamPart{Type: "reasoning-delta", ID: "r1", Text: "thinking..."},
			SendReasoning: true,
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "reasoning-delta" {
			t.Errorf("expected type 'reasoning-delta', got %q", result.Type)
		}
		if result.Delta != "thinking..." {
			t.Errorf("expected delta 'thinking...', got %q", result.Delta)
		}
	})

	t.Run("file should produce file chunk with data URL", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type: "file",
				File: &TextStreamPartFile{MediaType: "image/png", Base64: "iVBOR"},
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "file" {
			t.Errorf("expected type 'file', got %q", result.Type)
		}
		expectedURL := "data:image/png;base64,iVBOR"
		if result.URL != expectedURL {
			t.Errorf("expected URL %q, got %q", expectedURL, result.URL)
		}
	})

	t.Run("file should return nil when file data is nil", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "file"},
		})
		if result != nil {
			t.Error("expected nil for file without data")
		}
	})

	t.Run("source url should produce source-url chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:       "source",
				SourceType: "url",
				ID:         "s1",
				URL:        "https://example.com",
				Title:      "Example",
			},
			SendSources: true,
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "source-url" {
			t.Errorf("expected type 'source-url', got %q", result.Type)
		}
		if result.SourceID != "s1" {
			t.Errorf("expected sourceId 's1', got %q", result.SourceID)
		}
		if result.URL != "https://example.com" {
			t.Errorf("expected URL 'https://example.com', got %q", result.URL)
		}
	})

	t.Run("source should return nil when sendSources is false", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:        TextStreamPart{Type: "source", SourceType: "url"},
			SendSources: false,
		})
		if result != nil {
			t.Error("expected nil when sendSources is false")
		}
	})

	t.Run("source document should produce source-document chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:       "source",
				SourceType: "document",
				ID:         "d1",
				Title:      "Doc",
				MediaType:  "application/pdf",
				Filename:   "file.pdf",
			},
			SendSources: true,
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "source-document" {
			t.Errorf("expected type 'source-document', got %q", result.Type)
		}
		if result.Filename != "file.pdf" {
			t.Errorf("expected filename 'file.pdf', got %q", result.Filename)
		}
	})

	t.Run("tool-call should produce tool-input-available chunk", func(t *testing.T) {
		pExec := true
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:             "tool-call",
				ToolCallID:       "tc1",
				ToolName:         "search",
				Input:            map[string]any{"query": "test"},
				ProviderExecuted: &pExec,
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-input-available" {
			t.Errorf("expected type 'tool-input-available', got %q", result.Type)
		}
		if result.ToolCallID != "tc1" {
			t.Errorf("expected toolCallId 'tc1', got %q", result.ToolCallID)
		}
		if result.ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", result.ToolName)
		}
	})

	t.Run("tool-result should produce tool-output-available chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:       "tool-result",
				ToolCallID: "tc1",
				Output:     map[string]any{"data": "result"},
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-output-available" {
			t.Errorf("expected type 'tool-output-available', got %q", result.Type)
		}
	})

	t.Run("tool-error should produce tool-output-error chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:       "tool-error",
				ToolCallID: "tc1",
				Error:      "something went wrong",
			},
			OnError: func(err any) string {
				return "formatted: " + err.(string)
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-output-error" {
			t.Errorf("expected type 'tool-output-error', got %q", result.Type)
		}
		if !strings.Contains(result.ErrorText, "formatted: something went wrong") {
			t.Errorf("expected formatted error text, got %q", result.ErrorText)
		}
	})

	t.Run("error should produce error chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "error", Error: "big error"},
			OnError: func(err any) string {
				return err.(string)
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "error" {
			t.Errorf("expected type 'error', got %q", result.Type)
		}
		if result.ErrorText != "big error" {
			t.Errorf("expected errorText 'big error', got %q", result.ErrorText)
		}
	})

	t.Run("start should return nil when sendStart is false", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:      TextStreamPart{Type: "start"},
			SendStart: false,
		})
		if result != nil {
			t.Error("expected nil when sendStart is false")
		}
	})

	t.Run("start should return chunk when sendStart is true", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:              TextStreamPart{Type: "start"},
			SendStart:         true,
			ResponseMessageID: "msg-1",
			MessageMetadata:   map[string]any{"key": "value"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "start" {
			t.Errorf("expected type 'start', got %q", result.Type)
		}
		if result.MessageID != "msg-1" {
			t.Errorf("expected messageId 'msg-1', got %q", result.MessageID)
		}
	})

	t.Run("finish should return nil when sendFinish is false", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:       TextStreamPart{Type: "finish"},
			SendFinish: false,
		})
		if result != nil {
			t.Error("expected nil when sendFinish is false")
		}
	})

	t.Run("finish should return chunk when sendFinish is true", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part:       TextStreamPart{Type: "finish"},
			SendFinish: true,
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "finish" {
			t.Errorf("expected type 'finish', got %q", result.Type)
		}
	})

	t.Run("abort should produce abort chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "abort"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "abort" {
			t.Errorf("expected type 'abort', got %q", result.Type)
		}
	})

	t.Run("tool-input-end should return nil", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "tool-input-end"},
		})
		if result != nil {
			t.Error("expected nil for tool-input-end")
		}
	})

	t.Run("raw should return nil", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "raw"},
		})
		if result != nil {
			t.Error("expected nil for raw")
		}
	})

	t.Run("unknown type should return nil", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "unknown-type"},
		})
		if result != nil {
			t.Error("expected nil for unknown type")
		}
	})

	t.Run("tool-output should map output fields", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type: "tool-output",
				Output: map[string]any{
					"type":  "text-delta",
					"id":    "out-1",
					"delta": "output text",
				},
			},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", result.Type)
		}
		if result.ID != "out-1" {
			t.Errorf("expected id 'out-1', got %q", result.ID)
		}
		if result.Delta != "output text" {
			t.Errorf("expected delta 'output text', got %q", result.Delta)
		}
	})

	t.Run("tool-output should return nil for non-map output", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{
				Type:   "tool-output",
				Output: "not a map",
			},
		})
		if result != nil {
			t.Error("expected nil for non-map tool-output")
		}
	})

	t.Run("tool-input-start should produce tool-input-start chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "tool-input-start", ID: "tc2", ToolName: "calc"},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-input-start" {
			t.Errorf("expected type 'tool-input-start', got %q", result.Type)
		}
		if result.ToolCallID != "tc2" {
			t.Errorf("expected toolCallId 'tc2', got %q", result.ToolCallID)
		}
	})

	t.Run("tool-input-delta should produce tool-input-delta chunk", func(t *testing.T) {
		result := ConvertFullStreamChunkToUIMessageStream(ConvertFullStreamChunkToUIMessageStreamParams{
			Part: TextStreamPart{Type: "tool-input-delta", ID: "tc2", Delta: `{"a":1`},
		})
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-input-delta" {
			t.Errorf("expected type 'tool-input-delta', got %q", result.Type)
		}
		if result.InputTextDelta != `{"a":1` {
			t.Errorf("expected input text delta, got %q", result.InputTextDelta)
		}
	})
}
