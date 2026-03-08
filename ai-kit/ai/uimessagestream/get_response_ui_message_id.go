// Ported from: packages/ai/src/ui-message-stream/get-response-ui-message-id.ts
package uimessagestream

// ResponseMessageID holds either a static string ID or an IdGenerator function.
// This mirrors the TypeScript union: string | IdGenerator.
type ResponseMessageID struct {
	// Static is a fixed message ID. If non-empty, it takes precedence.
	Static string

	// Generator is a function that generates a message ID.
	Generator IdGenerator
}

// GetResponseUIMessageId determines the message ID to use for a response message.
//
// If the last message is an assistant message, its ID is reused (continuation).
// Otherwise, a new ID is generated or the provided ID is used.
//
// When originalMessages is nil (no persistence mode), an empty string is
// returned along with ok=false to indicate client-side ID generation should
// be used.
func GetResponseUIMessageId(originalMessages []UIMessage, responseMessageID ResponseMessageID) (string, bool) {
	// When there are no original messages (i.e. no persistence),
	// the assistant message id generation is handled on the client side.
	if originalMessages == nil {
		return "", false
	}

	if len(originalMessages) > 0 {
		lastMessage := originalMessages[len(originalMessages)-1]
		if lastMessage.Role == "assistant" {
			return lastMessage.ID, true
		}
	}

	if responseMessageID.Static != "" {
		return responseMessageID.Static, true
	}

	if responseMessageID.Generator != nil {
		return responseMessageID.Generator(), true
	}

	return "", true
}
