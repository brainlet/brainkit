// Ported from: packages/core/src/loop/workflows/errors.ts
package workflows

// ToolNotFoundError is raised when a tool name cannot be resolved in the
// available ToolSet. It mirrors the TS class ToolNotFoundError which extends
// Error and sets name = 'ToolNotFoundError'.
type ToolNotFoundError struct {
	Message string
}

func (e *ToolNotFoundError) Error() string {
	return e.Message
}

// NewToolNotFoundError creates a new ToolNotFoundError with the given message.
func NewToolNotFoundError(message string) *ToolNotFoundError {
	return &ToolNotFoundError{Message: message}
}
