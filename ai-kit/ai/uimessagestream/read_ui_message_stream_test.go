// Ported from: packages/ai/src/ui-message-stream/read-ui-message-stream.test.ts
package uimessagestream

import (
	"testing"

	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

func TestReadUIMessageStream(t *testing.T) {
	t.Run("should return a ui message object stream for a basic input stream", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "start", MessageID: "msg-123"},
			{Type: "start-step"},
			{Type: "text-start", ID: "text-1"},
			{Type: "text-delta", ID: "text-1", Delta: "Hello, "},
			{Type: "text-delta", ID: "text-1", Delta: "world!"},
			{Type: "text-end", ID: "text-1"},
			{Type: "finish-step"},
			{Type: "finish"},
		}
		stream := sliceToChan(chunks)

		uiMessages := ReadUIMessageStream(ReadUIMessageStreamOptions{
			Stream: stream,
		})

		result, err := util.CollectStream(uiMessages)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// We expect multiple UIMessage emissions as the state progresses.
		// The first emission sets the message ID.
		if len(result) == 0 {
			t.Fatal("expected at least one UIMessage emission")
		}

		// The first emitted message should have the message ID
		if result[0].ID != "msg-123" {
			t.Errorf("expected first message id=msg-123, got %q", result[0].ID)
		}

		// All emitted messages should be assistant role
		for i, msg := range result {
			if msg.Role != "assistant" {
				t.Errorf("message[%d]: expected role assistant, got %q", i, msg.Role)
			}
		}

		// The last emitted message should have the completed text
		lastMsg := result[len(result)-1]
		foundText := false
		for _, part := range lastMsg.Parts {
			if part.Type == "text" && part.Text == "Hello, world!" && part.State == "done" {
				foundText = true
			}
		}
		if !foundText {
			t.Errorf("expected final message to have text 'Hello, world!' with state=done, parts: %+v", lastMsg.Parts)
		}
	})

	t.Run("should call onError when encountering an error UI stream part", func(t *testing.T) {
		chunks := []UIMessageChunk{
			{Type: "start", MessageID: "msg-123"},
			{Type: "text-start", ID: "text-1"},
			{Type: "text-delta", ID: "text-1", Delta: "Hello"},
			{Type: "error", ErrorText: "Test error message"},
		}
		stream := sliceToChan(chunks)

		var receivedError error
		uiMessages := ReadUIMessageStream(ReadUIMessageStreamOptions{
			Stream: stream,
			OnError: func(err error) {
				receivedError = err
			},
			TerminateOnError: true,
		})

		// Consume the stream - it should terminate due to error
		_, _ = util.CollectStream(uiMessages)

		if receivedError == nil {
			t.Fatal("expected an error to be reported via onError")
		}
		if receivedError.Error() != "Test error message" {
			t.Errorf("expected error 'Test error message', got %q", receivedError.Error())
		}
	})
}
