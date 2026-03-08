// Ported from: packages/ai/src/ui-message-stream/create-ui-message-stream.test.ts
package uimessagestream

import (
	"fmt"
	"testing"
)

// collectChan reads all values from a channel into a slice.
func collectChan(ch <-chan UIMessageChunk) []UIMessageChunk {
	var result []UIMessageChunk
	for c := range ch {
		result = append(result, c)
	}
	return result
}

func TestCreateUIMessageStream(t *testing.T) {
	t.Run("should send data stream part and close the stream", func(t *testing.T) {
		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				writer.Write(UIMessageChunk{Type: "text-start", ID: "1"})
				writer.Write(UIMessageChunk{Type: "text-delta", ID: "1", Delta: "1a"})
				writer.Write(UIMessageChunk{Type: "text-end", ID: "1"})
				return nil
			},
		})

		result := collectChan(stream)
		if len(result) != 3 {
			t.Fatalf("expected 3 chunks, got %d", len(result))
		}
		if result[0].Type != "text-start" || result[0].ID != "1" {
			t.Errorf("unexpected chunk[0]: %+v", result[0])
		}
		if result[1].Type != "text-delta" || result[1].Delta != "1a" {
			t.Errorf("unexpected chunk[1]: %+v", result[1])
		}
		if result[2].Type != "text-end" || result[2].ID != "1" {
			t.Errorf("unexpected chunk[2]: %+v", result[2])
		}
	})

	t.Run("should forward a single stream with 2 elements", func(t *testing.T) {
		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				ch := make(chan UIMessageChunk)
				go func() {
					defer close(ch)
					ch <- UIMessageChunk{Type: "text-delta", ID: "1", Delta: "1a"}
					ch <- UIMessageChunk{Type: "text-delta", ID: "1", Delta: "1b"}
				}()
				writer.Merge(ch)
				return nil
			},
		})

		result := collectChan(stream)
		if len(result) != 2 {
			t.Fatalf("expected 2 chunks, got %d", len(result))
		}
		if result[0].Delta != "1a" {
			t.Errorf("expected delta 1a, got %q", result[0].Delta)
		}
		if result[1].Delta != "1b" {
			t.Errorf("expected delta 1b, got %q", result[1].Delta)
		}
	})

	t.Run("should add error parts when execute throws", func(t *testing.T) {
		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				return fmt.Errorf("execute-error")
			},
			OnError: func(err error) string {
				return "error-message"
			},
		})

		result := collectChan(stream)
		found := false
		for _, c := range result {
			if c.Type == "error" && c.ErrorText == "error-message" {
				found = true
			}
		}
		if !found {
			t.Error("expected error chunk with 'error-message'")
		}
	})

	t.Run("should add error parts when execute panics", func(t *testing.T) {
		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				panic("execute-panic")
			},
			OnError: func(err error) string {
				return "error-message"
			},
		})

		result := collectChan(stream)
		found := false
		for _, c := range result {
			if c.Type == "error" && c.ErrorText == "error-message" {
				found = true
			}
		}
		if !found {
			t.Error("expected error chunk with 'error-message'")
		}
	})

	t.Run("should handle onFinish without original messages", func(t *testing.T) {
		var recorded []UIMessageStreamOnFinishEvent

		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				writer.Write(UIMessageChunk{Type: "text-start", ID: "1"})
				writer.Write(UIMessageChunk{Type: "text-delta", ID: "1", Delta: "1a"})
				writer.Write(UIMessageChunk{Type: "text-end", ID: "1"})
				return nil
			},
			OnFinish: func(event UIMessageStreamOnFinishEvent) error {
				recorded = append(recorded, event)
				return nil
			},
			GenerateId: func() string { return "response-message-id" },
		})

		// Consume the stream
		collectChan(stream)

		if len(recorded) != 1 {
			t.Fatalf("expected 1 onFinish call, got %d", len(recorded))
		}
		ev := recorded[0]
		if ev.IsAborted {
			t.Error("expected isAborted=false")
		}
		if ev.IsContinuation {
			t.Error("expected isContinuation=false")
		}
		if ev.ResponseMessage.ID != "response-message-id" {
			t.Errorf("expected response message ID response-message-id, got %q", ev.ResponseMessage.ID)
		}
		if ev.ResponseMessage.Role != "assistant" {
			t.Errorf("expected role assistant, got %q", ev.ResponseMessage.Role)
		}
		if len(ev.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(ev.Messages))
		}
		// The message should have text parts assembled
		if len(ev.ResponseMessage.Parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(ev.ResponseMessage.Parts))
		}
		if ev.ResponseMessage.Parts[0].Text != "1a" {
			t.Errorf("expected text '1a', got %q", ev.ResponseMessage.Parts[0].Text)
		}
	})

	t.Run("should handle onFinish with messages (continuation)", func(t *testing.T) {
		var recorded []UIMessageStreamOnFinishEvent

		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				writer.Write(UIMessageChunk{Type: "text-start", ID: "1"})
				writer.Write(UIMessageChunk{Type: "text-delta", ID: "1", Delta: "1b"})
				writer.Write(UIMessageChunk{Type: "text-end", ID: "1"})
				return nil
			},
			OriginalMessages: []UIMessage{
				{
					ID:   "0",
					Role: "user",
					Parts: []UIMessagePart{
						{Type: "text", Text: "0a"},
					},
				},
				{
					ID:   "1",
					Role: "assistant",
					Parts: []UIMessagePart{
						{Type: "text", Text: "1a", State: "done"},
					},
				},
			},
			OnFinish: func(event UIMessageStreamOnFinishEvent) error {
				recorded = append(recorded, event)
				return nil
			},
		})

		collectChan(stream)

		if len(recorded) != 1 {
			t.Fatalf("expected 1 onFinish call, got %d", len(recorded))
		}
		ev := recorded[0]
		if !ev.IsContinuation {
			t.Error("expected isContinuation=true")
		}
		if ev.IsAborted {
			t.Error("expected isAborted=false")
		}
		if ev.ResponseMessage.ID != "1" {
			t.Errorf("expected response message ID '1', got %q", ev.ResponseMessage.ID)
		}
		if len(ev.Messages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(ev.Messages))
		}
		if ev.Messages[0].Role != "user" {
			t.Errorf("expected first message role user, got %q", ev.Messages[0].Role)
		}
	})

	t.Run("should inject a messageId into the stream when originalMessages are provided", func(t *testing.T) {
		var recorded []UIMessageStreamOnFinishEvent

		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				writer.Write(UIMessageChunk{Type: "start"}) // no messageId
				return nil
			},
			OriginalMessages: []UIMessage{
				{ID: "0", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "0a"}}},
			},
			OnFinish: func(event UIMessageStreamOnFinishEvent) error {
				recorded = append(recorded, event)
				return nil
			},
			GenerateId: func() string { return "response-message-id" },
		})

		result := collectChan(stream)

		// The start chunk should have the injected messageId
		foundStart := false
		for _, c := range result {
			if c.Type == "start" {
				foundStart = true
				if c.MessageID != "response-message-id" {
					t.Errorf("expected messageId response-message-id, got %q", c.MessageID)
				}
			}
		}
		if !foundStart {
			t.Error("expected a start chunk in the output")
		}

		if len(recorded) != 1 {
			t.Fatalf("expected 1 onFinish call, got %d", len(recorded))
		}
		ev := recorded[0]
		if ev.IsContinuation {
			t.Error("expected isContinuation=false")
		}
		if ev.ResponseMessage.ID != "response-message-id" {
			t.Errorf("expected response message ID response-message-id, got %q", ev.ResponseMessage.ID)
		}
	})

	t.Run("should keep existing messageId from start chunk when originalMessages are provided", func(t *testing.T) {
		var recorded []UIMessageStreamOnFinishEvent

		stream := CreateUIMessageStream(CreateUIMessageStreamOptions{
			Execute: func(writer *UIMessageStreamWriter) error {
				writer.Write(UIMessageChunk{Type: "start", MessageID: "existing-message-id"})
				return nil
			},
			OriginalMessages: []UIMessage{
				{ID: "0", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "0a"}}},
			},
			OnFinish: func(event UIMessageStreamOnFinishEvent) error {
				recorded = append(recorded, event)
				return nil
			},
			GenerateId: func() string { return "response-message-id" },
		})

		result := collectChan(stream)

		// The start chunk should keep the existing messageId
		for _, c := range result {
			if c.Type == "start" {
				if c.MessageID != "existing-message-id" {
					t.Errorf("expected messageId existing-message-id, got %q", c.MessageID)
				}
			}
		}

		if len(recorded) != 1 {
			t.Fatalf("expected 1 onFinish call, got %d", len(recorded))
		}
		ev := recorded[0]
		if ev.ResponseMessage.ID != "existing-message-id" {
			t.Errorf("expected response message ID existing-message-id, got %q", ev.ResponseMessage.ID)
		}
	})
}
