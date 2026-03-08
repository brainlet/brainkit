// Ported from: packages/ai/src/ui-message-stream/handle-ui-message-stream-finish.test.ts
package uimessagestream

import (
	"fmt"
	"testing"
)

func TestHandleUIMessageStreamFinish(t *testing.T) {
	t.Run("stream pass-through without onFinish", func(t *testing.T) {
		t.Run("should pass through stream chunks without processing when onFinish is not provided", func(t *testing.T) {
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-123"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Hello"},
				{Type: "text-delta", ID: "text-1", Delta: " World"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			errorHandlerCalled := false
			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-123",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) { errorHandlerCalled = true },
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			for i, expected := range inputChunks {
				if result[i].Type != expected.Type {
					t.Errorf("chunk[%d]: expected type %q, got %q", i, expected.Type, result[i].Type)
				}
			}
			if errorHandlerCalled {
				t.Error("error handler should not have been called")
			}
		})

		t.Run("should inject messageId when not present in start chunk", func(t *testing.T) {
			inputChunks := []UIMessageChunk{
				{Type: "start"}, // no messageId
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Test"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "injected-123",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) {},
			})

			result := collectChan(resultStream)

			if result[0].MessageID != "injected-123" {
				t.Errorf("expected messageId injected-123, got %q", result[0].MessageID)
			}
		})
	})

	t.Run("stream processing with onFinish callback", func(t *testing.T) {
		t.Run("should process stream and call onFinish with correct parameters", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-456"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Hello"},
				{Type: "text-delta", ID: "text-1", Delta: " World"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			originalMessages := []UIMessage{
				{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Hello"}}},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-456",
				OriginalMessages: originalMessages,
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if finishEvent.IsContinuation {
				t.Error("expected isContinuation=false")
			}
			if finishEvent.ResponseMessage.ID != "msg-456" {
				t.Errorf("expected responseMessage.id=msg-456, got %q", finishEvent.ResponseMessage.ID)
			}
			if finishEvent.ResponseMessage.Role != "assistant" {
				t.Errorf("expected role assistant, got %q", finishEvent.ResponseMessage.Role)
			}
			if len(finishEvent.Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(finishEvent.Messages))
			}
		})

		t.Run("should handle empty original messages array", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-789"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Response"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-789",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			collectChan(resultStream)

			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if finishEvent.IsContinuation {
				t.Error("expected isContinuation=false")
			}
			if len(finishEvent.Messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(finishEvent.Messages))
			}
		})
	})

	t.Run("continuation scenario", func(t *testing.T) {
		t.Run("should handle continuation when last message is assistant", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "assistant-msg-1"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: " continued"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			originalMessages := []UIMessage{
				{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Continue this"}}},
				{ID: "assistant-msg-1", Role: "assistant", Parts: []UIMessagePart{{Type: "text", Text: "This is"}}},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-999",
				OriginalMessages: originalMessages,
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			collectChan(resultStream)

			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if !finishEvent.IsContinuation {
				t.Error("expected isContinuation=true")
			}
			if finishEvent.ResponseMessage.ID != "assistant-msg-1" {
				t.Errorf("expected id=assistant-msg-1, got %q", finishEvent.ResponseMessage.ID)
			}
			if len(finishEvent.Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(finishEvent.Messages))
			}
		})

		t.Run("should not treat user message as continuation", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-001"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "New response"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			originalMessages := []UIMessage{
				{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Question"}}},
				{ID: "user-msg-2", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Another question"}}},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-001",
				OriginalMessages: originalMessages,
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			collectChan(resultStream)

			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if finishEvent.IsContinuation {
				t.Error("expected isContinuation=false")
			}
			if finishEvent.ResponseMessage.ID != "msg-001" {
				t.Errorf("expected id=msg-001, got %q", finishEvent.ResponseMessage.ID)
			}
			if len(finishEvent.Messages) != 3 {
				t.Fatalf("expected 3 messages, got %d", len(finishEvent.Messages))
			}
		})
	})

	t.Run("abort scenarios", func(t *testing.T) {
		t.Run("should set isAborted to true when abort chunk is encountered", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-abort-1"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Starting text"},
				{Type: "abort"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-abort-1",
				OriginalMessages: []UIMessage{{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Test request"}}}},
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if !finishEvent.IsAborted {
				t.Error("expected isAborted=true")
			}
			if finishEvent.IsContinuation {
				t.Error("expected isContinuation=false")
			}
			if finishEvent.ResponseMessage.ID != "msg-abort-1" {
				t.Errorf("expected id=msg-abort-1, got %q", finishEvent.ResponseMessage.ID)
			}
			if len(finishEvent.Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(finishEvent.Messages))
			}
		})

		t.Run("should set isAborted to false when no abort chunk is encountered", func(t *testing.T) {
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-normal"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Complete text"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-normal",
				OriginalMessages: []UIMessage{{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Test request"}}}},
				OnError:          func(err error) {},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			collectChan(resultStream)

			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			if finishEvent.IsAborted {
				t.Error("expected isAborted=false")
			}
			if finishEvent.ResponseMessage.ID != "msg-normal" {
				t.Errorf("expected id=msg-normal, got %q", finishEvent.ResponseMessage.ID)
			}
		})

		t.Run("should handle abort chunk in pass-through mode without onFinish", func(t *testing.T) {
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-abort-passthrough"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Text before abort"},
				{Type: "abort"},
				{Type: "finish"},
			}

			errorHandlerCalled := false
			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-abort-passthrough",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) { errorHandlerCalled = true },
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if errorHandlerCalled {
				t.Error("error handler should not have been called")
			}
		})
	})

	t.Run("onStepFinish callback", func(t *testing.T) {
		t.Run("should call onStepFinish when finish-step chunk is encountered", func(t *testing.T) {
			var stepEvents []UIMessageStreamOnStepFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-step-1"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Step 1 text"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			originalMessages := []UIMessage{
				{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Hello"}}},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-step-1",
				OriginalMessages: originalMessages,
				OnError:          func(err error) {},
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					stepEvents = append(stepEvents, event)
					return nil
				},
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if len(stepEvents) != 1 {
				t.Fatalf("expected 1 onStepFinish call, got %d", len(stepEvents))
			}
			if stepEvents[0].IsContinuation {
				t.Error("expected isContinuation=false")
			}
			if stepEvents[0].ResponseMessage.ID != "msg-step-1" {
				t.Errorf("expected id=msg-step-1, got %q", stepEvents[0].ResponseMessage.ID)
			}
			if stepEvents[0].ResponseMessage.Role != "assistant" {
				t.Errorf("expected role assistant, got %q", stepEvents[0].ResponseMessage.Role)
			}
			if len(stepEvents[0].Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(stepEvents[0].Messages))
			}
		})

		t.Run("should call onStepFinish multiple times for multiple steps", func(t *testing.T) {
			var stepEvents []UIMessageStreamOnStepFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-multi-step"},
				// Step 1
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Step 1"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				// Step 2
				{Type: "start-step"},
				{Type: "text-start", ID: "text-2"},
				{Type: "text-delta", ID: "text-2", Delta: "Step 2"},
				{Type: "text-end", ID: "text-2"},
				{Type: "finish-step"},
				// Step 3
				{Type: "start-step"},
				{Type: "text-start", ID: "text-3"},
				{Type: "text-delta", ID: "text-3", Delta: "Step 3"},
				{Type: "text-end", ID: "text-3"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-multi-step",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) {},
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					stepEvents = append(stepEvents, event)
					return nil
				},
			})

			collectChan(resultStream)

			if len(stepEvents) != 3 {
				t.Fatalf("expected 3 onStepFinish calls, got %d", len(stepEvents))
			}

			// Step 1 should have 1 text part
			if len(stepEvents[0].ResponseMessage.Parts) < 1 {
				t.Errorf("step 1: expected at least 1 part, got %d", len(stepEvents[0].ResponseMessage.Parts))
			}
		})

		t.Run("should call both onStepFinish and onFinish when both are provided", func(t *testing.T) {
			stepCalls := 0
			finishCalls := 0
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-both"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Hello"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-both",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) {},
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					stepCalls++
					return nil
				},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishCalls++
					return nil
				},
			})

			collectChan(resultStream)

			if stepCalls != 1 {
				t.Errorf("expected 1 onStepFinish call, got %d", stepCalls)
			}
			if finishCalls != 1 {
				t.Errorf("expected 1 onFinish call, got %d", finishCalls)
			}
		})

		t.Run("should handle onStepFinish errors by logging and continuing", func(t *testing.T) {
			errorHandlerCalls := 0
			stepCalls := 0
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-error"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Step 1"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "start-step"},
				{Type: "text-start", ID: "text-2"},
				{Type: "text-delta", ID: "text-2", Delta: "Step 2"},
				{Type: "text-end", ID: "text-2"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-error",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) { errorHandlerCalls++ },
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					stepCalls++
					return fmt.Errorf("DB error")
				},
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if stepCalls != 2 {
				t.Errorf("expected 2 onStepFinish calls, got %d", stepCalls)
			}
			if errorHandlerCalls != 2 {
				t.Errorf("expected 2 error handler calls, got %d", errorHandlerCalls)
			}
		})

		t.Run("should handle continuation scenario with onStepFinish", func(t *testing.T) {
			var stepEvents []UIMessageStreamOnStepFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "assistant-msg-1"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: " continued"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			originalMessages := []UIMessage{
				{ID: "user-msg-1", Role: "user", Parts: []UIMessagePart{{Type: "text", Text: "Continue this"}}},
				{ID: "assistant-msg-1", Role: "assistant", Parts: []UIMessagePart{{Type: "text", Text: "This is"}}},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-999",
				OriginalMessages: originalMessages,
				OnError:          func(err error) {},
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					stepEvents = append(stepEvents, event)
					return nil
				},
			})

			collectChan(resultStream)

			if len(stepEvents) != 1 {
				t.Fatalf("expected 1 onStepFinish call, got %d", len(stepEvents))
			}
			if !stepEvents[0].IsContinuation {
				t.Error("expected isContinuation=true")
			}
			if stepEvents[0].ResponseMessage.ID != "assistant-msg-1" {
				t.Errorf("expected id=assistant-msg-1, got %q", stepEvents[0].ResponseMessage.ID)
			}
			if len(stepEvents[0].Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(stepEvents[0].Messages))
			}
		})

		t.Run("should provide deep-cloned messages in onStepFinish to prevent mutation", func(t *testing.T) {
			var stepEvent *UIMessageStreamOnStepFinishEvent
			var finishEvent *UIMessageStreamOnFinishEvent
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-clone"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Hello"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-clone",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) {},
				OnStepFinish: func(event UIMessageStreamOnStepFinishEvent) error {
					// Mutate the message in the callback
					event.ResponseMessage.Parts = append(event.ResponseMessage.Parts, UIMessagePart{Type: "text", Text: "MUTATION!"})
					stepEvent = &event
					return nil
				},
				OnFinish: func(event UIMessageStreamOnFinishEvent) error {
					finishEvent = &event
					return nil
				},
			})

			collectChan(resultStream)

			if stepEvent == nil {
				t.Fatal("onStepFinish was not called")
			}
			// The step event should have the mutated message
			if len(stepEvent.ResponseMessage.Parts) != 2 { // Original + mutation
				t.Errorf("step: expected 2 parts, got %d", len(stepEvent.ResponseMessage.Parts))
			}

			if finishEvent == nil {
				t.Fatal("onFinish was not called")
			}
			// The finish event should NOT see the mutation
			if len(finishEvent.ResponseMessage.Parts) != 1 {
				t.Errorf("finish: expected 1 part (no mutation), got %d", len(finishEvent.ResponseMessage.Parts))
			}
		})

		t.Run("should not process stream when neither onFinish nor onStepFinish is provided", func(t *testing.T) {
			inputChunks := []UIMessageChunk{
				{Type: "start", MessageID: "msg-passthrough"},
				{Type: "text-start", ID: "text-1"},
				{Type: "text-delta", ID: "text-1", Delta: "Test"},
				{Type: "text-end", ID: "text-1"},
				{Type: "finish-step"},
				{Type: "finish"},
			}

			errorHandlerCalled := false
			resultStream := HandleUIMessageStreamFinish(HandleUIMessageStreamFinishOptions{
				Stream:           sliceToChan(inputChunks),
				MessageID:        "msg-passthrough",
				OriginalMessages: []UIMessage{},
				OnError:          func(err error) { errorHandlerCalled = true },
			})

			result := collectChan(resultStream)

			if len(result) != len(inputChunks) {
				t.Fatalf("expected %d chunks, got %d", len(inputChunks), len(result))
			}
			if errorHandlerCalled {
				t.Error("error handler should not have been called")
			}
		})
	})
}
