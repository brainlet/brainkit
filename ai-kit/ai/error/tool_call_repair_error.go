// Ported from: packages/ai/src/error/tool-call-repair-error.ts
package aierror

const toolCallRepairErrorName = "AI_ToolCallRepairError"
const toolCallRepairErrorMarker = "vercel.ai.error." + toolCallRepairErrorName

// ToolCallRepairError is returned when an attempt to repair a tool call fails.
type ToolCallRepairError struct {
	AISDKError

	// OriginalError is the original error that triggered the repair attempt.
	// It is either a *NoSuchToolError or an *InvalidToolInputError.
	OriginalError error
}

// ToolCallRepairErrorOptions are the options for creating a ToolCallRepairError.
type ToolCallRepairErrorOptions struct {
	// Message overrides the default error message. Optional.
	Message string
	// Cause is the underlying error from the repair attempt.
	Cause error
	// OriginalError is the original NoSuchToolError or InvalidToolInputError.
	OriginalError error
}

// NewToolCallRepairError creates a new ToolCallRepairError.
func NewToolCallRepairError(opts ToolCallRepairErrorOptions) *ToolCallRepairError {
	message := opts.Message
	if message == "" {
		message = "Error repairing tool call: " + GetErrorMessage(opts.Cause)
	}

	return &ToolCallRepairError{
		AISDKError: AISDKError{
			Name:    toolCallRepairErrorName,
			Message: message,
			Cause:   opts.Cause,
		},
		OriginalError: opts.OriginalError,
	}
}

// IsToolCallRepairError checks whether the given error is a ToolCallRepairError.
func IsToolCallRepairError(err error) bool {
	_, ok := err.(*ToolCallRepairError)
	return ok
}
