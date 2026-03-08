// Ported from: packages/ai/src/ui-message-stream/get-response-ui-message-id.test.ts
package uimessagestream

import "testing"

func TestGetResponseUIMessageId(t *testing.T) {
	mockGenerateId := func() string { return "new-id" }

	t.Run("should return empty with ok=false when originalMessages is nil", func(t *testing.T) {
		result, ok := GetResponseUIMessageId(nil, ResponseMessageID{Generator: mockGenerateId})
		if ok {
			t.Error("expected ok=false for nil originalMessages")
		}
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should return the last assistant message id when present", func(t *testing.T) {
		messages := []UIMessage{
			{ID: "msg-1", Role: "user", Parts: []UIMessagePart{}},
			{ID: "msg-2", Role: "assistant", Parts: []UIMessagePart{}},
		}
		result, ok := GetResponseUIMessageId(messages, ResponseMessageID{Generator: mockGenerateId})
		if !ok {
			t.Error("expected ok=true")
		}
		if result != "msg-2" {
			t.Errorf("expected msg-2, got %q", result)
		}
	})

	t.Run("should generate new id when last message is not from assistant", func(t *testing.T) {
		messages := []UIMessage{
			{ID: "msg-1", Role: "assistant", Parts: []UIMessagePart{}},
			{ID: "msg-2", Role: "user", Parts: []UIMessagePart{}},
		}
		result, ok := GetResponseUIMessageId(messages, ResponseMessageID{Generator: mockGenerateId})
		if !ok {
			t.Error("expected ok=true")
		}
		if result != "new-id" {
			t.Errorf("expected new-id, got %q", result)
		}
	})

	t.Run("should generate new id when messages array is empty", func(t *testing.T) {
		result, ok := GetResponseUIMessageId([]UIMessage{}, ResponseMessageID{Generator: mockGenerateId})
		if !ok {
			t.Error("expected ok=true")
		}
		if result != "new-id" {
			t.Errorf("expected new-id, got %q", result)
		}
	})

	t.Run("should use the responseMessageId when it is a string", func(t *testing.T) {
		result, ok := GetResponseUIMessageId([]UIMessage{}, ResponseMessageID{Static: "response-id"})
		if !ok {
			t.Error("expected ok=true")
		}
		if result != "response-id" {
			t.Errorf("expected response-id, got %q", result)
		}
	})
}
