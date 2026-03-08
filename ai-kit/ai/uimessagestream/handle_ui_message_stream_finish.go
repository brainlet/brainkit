// Ported from: packages/ai/src/ui-message-stream/handle-ui-message-stream-finish.ts
package uimessagestream

import "sync"

// HandleUIMessageStreamFinishOptions configures HandleUIMessageStreamFinish.
type HandleUIMessageStreamFinishOptions struct {
	// Stream is the input channel of UIMessageChunk values.
	Stream <-chan UIMessageChunk

	// MessageID is the message ID to use for the response message.
	// If empty, no id will be set for the response message.
	MessageID string

	// OriginalMessages is the original messages. Defaults to empty slice.
	OriginalMessages []UIMessage

	// OnError is called when an error occurs.
	OnError func(err error)

	// OnStepFinish is called when each step finishes during multi-step agent runs.
	OnStepFinish UIMessageStreamOnStepFinishCallback

	// OnFinish is called when the stream finishes.
	OnFinish UIMessageStreamOnFinishCallback
}

// HandleUIMessageStreamFinish processes a UI message stream, injecting message IDs
// and invoking onFinish/onStepFinish callbacks as appropriate.
//
// It returns a new channel of UIMessageChunk values. The returned channel is the
// processed stream that should be consumed by the caller.
func HandleUIMessageStreamFinish(opts HandleUIMessageStreamFinishOptions) <-chan UIMessageChunk {
	originalMessages := opts.OriginalMessages
	if originalMessages == nil {
		originalMessages = []UIMessage{}
	}

	messageID := opts.MessageID

	// Determine last assistant message for continuation detection
	var lastMessage *UIMessage
	if len(originalMessages) > 0 {
		last := originalMessages[len(originalMessages)-1]
		if last.Role == "assistant" {
			lastMessage = &last
			messageID = last.ID // use existing assistant message ID
		}
	}

	isAborted := false

	// Phase 1: Inject messageId into start chunks and track abort
	idInjected := make(chan UIMessageChunk)
	go func() {
		defer close(idInjected)
		for chunk := range opts.Stream {
			if chunk.Type == "start" {
				if chunk.MessageID == "" && messageID != "" {
					chunk.MessageID = messageID
				}
			}
			if chunk.Type == "abort" {
				isAborted = true
			}
			idInjected <- chunk
		}
	}()

	// If no callbacks, just return the id-injected stream
	if opts.OnFinish == nil && opts.OnStepFinish == nil {
		return idInjected
	}

	// Phase 2: Process the stream through the state machine for callbacks
	var lastMessageCopy *UIMessage
	if lastMessage != nil {
		cp := lastMessage.DeepCopy()
		lastMessageCopy = &cp
	}

	state := CreateStreamingUIMessageState(messageID, lastMessageCopy)

	var mu sync.Mutex
	runUpdateMessageJob := func(job func(state *StreamingUIMessageState, write func()) error) error {
		mu.Lock()
		defer mu.Unlock()
		return job(state, func() {})
	}

	processed := ProcessUIMessageStream(ProcessUIMessageStreamOptions{
		Input:               idInjected,
		RunUpdateMessageJob: runUpdateMessageJob,
		OnError:             opts.OnError,
	})

	finishCalled := false

	callOnFinish := func() {
		if finishCalled || opts.OnFinish == nil {
			return
		}
		finishCalled = true

		mu.Lock()
		isContinuation := lastMessage != nil && state.Message.ID == lastMessage.ID
		msgCopy := state.Message.DeepCopy()
		finishReason := state.FinishReason
		mu.Unlock()

		var msgs []UIMessage
		if isContinuation {
			msgs = append(msgs, originalMessages[:len(originalMessages)-1]...)
		} else {
			msgs = append(msgs, originalMessages...)
		}
		msgs = append(msgs, msgCopy)

		_ = opts.OnFinish(UIMessageStreamOnFinishEvent{
			IsAborted:       isAborted,
			IsContinuation:  isContinuation,
			ResponseMessage: msgCopy,
			Messages:        msgs,
			FinishReason:    finishReason,
		})
	}

	callOnStepFinish := func() {
		if opts.OnStepFinish == nil {
			return
		}

		mu.Lock()
		isContinuation := lastMessage != nil && state.Message.ID == lastMessage.ID
		responseMsg := state.Message.DeepCopy()
		mu.Unlock()

		var msgs []UIMessage
		if isContinuation {
			msgs = append(msgs, originalMessages[:len(originalMessages)-1]...)
		} else {
			msgs = append(msgs, originalMessages...)
		}
		msgs = append(msgs, responseMsg)

		err := opts.OnStepFinish(UIMessageStreamOnStepFinishEvent{
			IsContinuation:  isContinuation,
			ResponseMessage: responseMsg,
			Messages:        msgs,
		})
		if err != nil && opts.OnError != nil {
			opts.OnError(err)
		}
	}

	// Phase 3: Intercept finish-step chunks to call onStepFinish,
	// and call onFinish when the stream ends.
	output := make(chan UIMessageChunk)
	go func() {
		defer close(output)
		defer callOnFinish()
		for chunk := range processed {
			if chunk.Type == "finish-step" {
				callOnStepFinish()
			}
			output <- chunk
		}
	}()

	return output
}
