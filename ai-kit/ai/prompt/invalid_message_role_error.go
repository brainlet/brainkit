// Ported from: packages/ai/src/prompt/invalid-message-role-error.ts
package prompt

import "fmt"

const invalidMessageRoleErrorName = "AI_InvalidMessageRoleError"
const invalidMessageRoleErrorMarker = "vercel.ai.error." + invalidMessageRoleErrorName

// InvalidMessageRoleError is returned when a message has an invalid role.
type InvalidMessageRoleError struct {
	// Name is the error classification.
	Name string
	// Message is the human-readable error message.
	Message string
	// Role is the invalid role that was provided.
	Role string
}

func (e *InvalidMessageRoleError) Error() string {
	return e.Message
}

// NewInvalidMessageRoleError creates a new InvalidMessageRoleError.
func NewInvalidMessageRoleError(role string, message string) *InvalidMessageRoleError {
	if message == "" {
		message = fmt.Sprintf(
			`Invalid message role: '%s'. Must be one of: "system", "user", "assistant", "tool".`,
			role,
		)
	}
	return &InvalidMessageRoleError{
		Name:    invalidMessageRoleErrorName,
		Message: message,
		Role:    role,
	}
}

// IsInvalidMessageRoleError checks whether the given error is an InvalidMessageRoleError.
func IsInvalidMessageRoleError(err error) bool {
	_, ok := err.(*InvalidMessageRoleError)
	return ok
}
