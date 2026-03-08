// Ported from: packages/ai/src/prompt/message-conversion-error.ts
package prompt

const messageConversionErrorName = "AI_MessageConversionError"
const messageConversionErrorMarker = "vercel.ai.error." + messageConversionErrorName

// UIMessage is a stub for the UI message type.
// TODO: import from brainlink/experiments/ai-kit/ui once it exists
type UIMessage struct {
	ID      string
	Role    string
	Content string
}

// MessageConversionError is returned when a message cannot be converted.
type MessageConversionError struct {
	// Name is the error classification.
	Name string
	// Message is the human-readable error message.
	Message string
	// OriginalMessage is the message that could not be converted.
	OriginalMessage UIMessage
}

func (e *MessageConversionError) Error() string {
	return e.Message
}

// NewMessageConversionError creates a new MessageConversionError.
func NewMessageConversionError(originalMessage UIMessage, message string) *MessageConversionError {
	return &MessageConversionError{
		Name:            messageConversionErrorName,
		Message:         message,
		OriginalMessage: originalMessage,
	}
}

// IsMessageConversionError checks whether the given error is a MessageConversionError.
func IsMessageConversionError(err error) bool {
	_, ok := err.(*MessageConversionError)
	return ok
}
