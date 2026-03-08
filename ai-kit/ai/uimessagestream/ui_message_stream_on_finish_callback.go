// Ported from: packages/ai/src/ui-message-stream/ui-message-stream-on-finish-callback.ts
package uimessagestream

// UIMessageStreamOnFinishEvent contains the data passed to the OnFinish callback
// when a UI message stream finishes.
type UIMessageStreamOnFinishEvent struct {
	// Messages is the updated list of UI messages.
	Messages []UIMessage

	// IsContinuation indicates whether the response message is a continuation
	// of the last original message, or if a new message was created.
	IsContinuation bool

	// IsAborted indicates whether the stream was aborted.
	IsAborted bool

	// ResponseMessage is the message that was sent to the client as a response
	// (including the original message if it was extended).
	ResponseMessage UIMessage

	// FinishReason is the reason why the generation finished.
	FinishReason string
}

// UIMessageStreamOnFinishCallback is called when a UI message stream finishes.
type UIMessageStreamOnFinishCallback func(event UIMessageStreamOnFinishEvent) error
