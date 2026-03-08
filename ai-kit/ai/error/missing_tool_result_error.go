// Ported from: packages/ai/src/error/missing-tool-result-error.ts
package aierror

import (
	"fmt"
	"strings"
)

const missingToolResultsErrorName = "AI_MissingToolResultsError"
const missingToolResultsErrorMarker = "vercel.ai.error." + missingToolResultsErrorName

// MissingToolResultsError is returned when tool results are missing for one or more tool calls.
type MissingToolResultsError struct {
	AISDKError

	// ToolCallIDs contains the IDs of the tool calls that are missing results.
	ToolCallIDs []string
}

// NewMissingToolResultsError creates a new MissingToolResultsError.
func NewMissingToolResultsError(toolCallIDs []string) *MissingToolResultsError {
	var verb, plural string
	if len(toolCallIDs) > 1 {
		verb = "s are"
		plural = "s"
	} else {
		verb = " is"
		plural = ""
	}

	message := fmt.Sprintf("Tool result%s missing for tool call%s %s.",
		verb, plural, strings.Join(toolCallIDs, ", "))

	return &MissingToolResultsError{
		AISDKError: AISDKError{
			Name:    missingToolResultsErrorName,
			Message: message,
		},
		ToolCallIDs: toolCallIDs,
	}
}

// IsMissingToolResultsError checks whether the given error is a MissingToolResultsError.
func IsMissingToolResultsError(err error) bool {
	_, ok := err.(*MissingToolResultsError)
	return ok
}
