// Ported from: packages/core/src/agent/message-list/utils/tool-name.ts
package utils

import (
	"regexp"
)

var toolNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// FallbackToolName is used when a tool name is invalid or not a string.
const FallbackToolName = "unknown_tool"

// SanitizeToolName validates and sanitizes a tool name string.
// Returns FallbackToolName if the input is not a valid tool name.
func SanitizeToolName(toolName string) string {
	if toolNamePattern.MatchString(toolName) {
		return toolName
	}
	return FallbackToolName
}
