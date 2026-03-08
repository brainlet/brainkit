// Ported from: packages/ai/src/ui-message-stream/ui-message-stream-on-step-finish-callback.ts
package uimessagestream

// UIMessageStreamOnStepFinishEvent contains the data passed to the OnStepFinish callback
// when a step finishes during streaming.
type UIMessageStreamOnStepFinishEvent struct {
	// Messages is the updated list of UI messages at the end of this step.
	Messages []UIMessage

	// IsContinuation indicates whether the response message is a continuation
	// of the last original message, or if a new message was created.
	IsContinuation bool

	// ResponseMessage is the message that was sent to the client as a response
	// (including the original message if it was extended).
	ResponseMessage UIMessage
}

// UIMessageStreamOnStepFinishCallback is called when a step finishes during streaming.
// This is useful for persisting intermediate UI messages during multi-step agent runs.
type UIMessageStreamOnStepFinishCallback func(event UIMessageStreamOnStepFinishEvent) error
