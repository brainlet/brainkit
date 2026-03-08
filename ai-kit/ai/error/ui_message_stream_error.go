// Ported from: packages/ai/src/error/ui-message-stream-error.ts
package aierror

const uiMessageStreamErrorName = "AI_UIMessageStreamError"
const uiMessageStreamErrorMarker = "vercel.ai.error." + uiMessageStreamErrorName

// UIMessageStreamError is returned when a UI message stream contains invalid
// or out-of-sequence chunks.
//
// This typically occurs when:
//   - A delta chunk is received without a corresponding start chunk
//   - An end chunk is received without a corresponding start chunk
//   - A tool invocation is not found for the given toolCallId
type UIMessageStreamError struct {
	AISDKError

	// ChunkType is the type of chunk that caused the error (e.g., "text-delta", "reasoning-end").
	ChunkType string

	// ChunkID is the ID associated with the failing chunk (part ID or toolCallId).
	ChunkID string
}

// NewUIMessageStreamError creates a new UIMessageStreamError.
func NewUIMessageStreamError(chunkType, chunkID, message string) *UIMessageStreamError {
	return &UIMessageStreamError{
		AISDKError: AISDKError{
			Name:    uiMessageStreamErrorName,
			Message: message,
		},
		ChunkType: chunkType,
		ChunkID:   chunkID,
	}
}

// IsUIMessageStreamError checks whether the given error is a UIMessageStreamError.
func IsUIMessageStreamError(err error) bool {
	_, ok := err.(*UIMessageStreamError)
	return ok
}
