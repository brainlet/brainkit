// Ported from: packages/ai/src/error/no-such-tool-error.ts
package aierror

import (
	"fmt"
	"strings"
)

const noSuchToolErrorName = "AI_NoSuchToolError"
const noSuchToolErrorMarker = "vercel.ai.error." + noSuchToolErrorName

// NoSuchToolError is returned when a model tries to call a tool that is not available.
type NoSuchToolError struct {
	AISDKError

	// ToolName is the name of the tool that was requested but not found.
	ToolName string

	// AvailableTools lists the tools that are available. Nil if no tools are available.
	AvailableTools []string
}

// NoSuchToolErrorOptions are the options for creating a NoSuchToolError.
type NoSuchToolErrorOptions struct {
	// ToolName is the name of the tool that was requested but not found.
	ToolName string
	// AvailableTools lists the tools that are available. Optional.
	AvailableTools []string
	// Message overrides the default error message. Optional.
	Message string
}

// NewNoSuchToolError creates a new NoSuchToolError.
func NewNoSuchToolError(opts NoSuchToolErrorOptions) *NoSuchToolError {
	message := opts.Message
	if message == "" {
		if opts.AvailableTools == nil {
			message = fmt.Sprintf("Model tried to call unavailable tool '%s'. No tools are available.", opts.ToolName)
		} else {
			message = fmt.Sprintf("Model tried to call unavailable tool '%s'. Available tools: %s.",
				opts.ToolName, strings.Join(opts.AvailableTools, ", "))
		}
	}

	return &NoSuchToolError{
		AISDKError: AISDKError{
			Name:    noSuchToolErrorName,
			Message: message,
		},
		ToolName:       opts.ToolName,
		AvailableTools: opts.AvailableTools,
	}
}

// IsNoSuchToolError checks whether the given error is a NoSuchToolError.
func IsNoSuchToolError(err error) bool {
	_, ok := err.(*NoSuchToolError)
	return ok
}
