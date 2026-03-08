// Ported from: packages/core/src/agent/message-list/detection/TypeDetector.ts
package detection

import (
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// TypeDetector provides centralized type detection for different message formats.
// The detection order is important because some formats share similar properties.
type TypeDetector struct{}

// IsMastraDBMessage checks if a message map represents a MastraDBMessage (format 2).
func IsMastraDBMessage(msg map[string]any) bool {
	content, ok := msg["content"]
	if !ok {
		return false
	}
	contentMap, ok := content.(map[string]any)
	if !ok {
		return false
	}
	format, ok := contentMap["format"]
	if !ok {
		return false
	}
	formatNum, ok := format.(float64)
	if !ok {
		if formatInt, ok := format.(int); ok {
			return formatInt == 2
		}
		return false
	}
	return formatNum == 2
}

// IsMastraDBMessageTyped checks if a typed message is a MastraDBMessage.
func IsMastraDBMessageTyped(msg *state.MastraDBMessage) bool {
	return msg != nil && msg.Content.Format == 2
}

// IsMastraMessageV1 checks if a message map is a MastraMessageV1 (legacy format).
func IsMastraMessageV1(msg map[string]any) bool {
	if IsMastraDBMessage(msg) {
		return false
	}
	_, hasThread := msg["threadId"]
	_, hasResource := msg["resourceId"]
	return hasThread || hasResource
}

// IsMastraMessage checks if a message is either Mastra format (V1 or V2/DB).
func IsMastraMessage(msg map[string]any) bool {
	return IsMastraDBMessage(msg) || IsMastraMessageV1(msg)
}

// IsAIV4UIMessage checks if a message map is an AIV4 UIMessage.
func IsAIV4UIMessage(msg map[string]any) bool {
	if IsMastraMessage(msg) {
		return false
	}
	if IsAIV4CoreMessage(msg) {
		return false
	}
	_, hasParts := msg["parts"]
	if !hasParts {
		return false
	}
	return !HasAIV5UIMessageCharacteristics(msg)
}

// IsAIV5UIMessage checks if a message map is an AIV5 UIMessage.
func IsAIV5UIMessage(msg map[string]any) bool {
	if IsMastraMessage(msg) {
		return false
	}
	if IsAIV5CoreMessage(msg) {
		return false
	}
	_, hasParts := msg["parts"]
	if !hasParts {
		return false
	}
	return HasAIV5UIMessageCharacteristics(msg)
}

// IsAIV4CoreMessage checks if a message map is an AIV4 CoreMessage.
func IsAIV4CoreMessage(msg map[string]any) bool {
	if IsMastraMessage(msg) {
		return false
	}
	_, hasParts := msg["parts"]
	if hasParts {
		return false
	}
	_, hasContent := msg["content"]
	if !hasContent {
		return false
	}
	return !HasAIV5CoreMessageCharacteristics(msg)
}

// IsAIV5CoreMessage checks if a message map is an AIV5 ModelMessage.
func IsAIV5CoreMessage(msg map[string]any) bool {
	if IsMastraMessage(msg) {
		return false
	}
	_, hasParts := msg["parts"]
	if hasParts {
		return false
	}
	_, hasContent := msg["content"]
	if !hasContent {
		return false
	}
	return HasAIV5CoreMessageCharacteristics(msg)
}

// HasAIV5UIMessageCharacteristics checks if a message has AIV5 UIMessage characteristics.
func HasAIV5UIMessageCharacteristics(msg map[string]any) bool {
	// V4 has these separated arrays that don't record overall order
	for _, key := range []string{"toolInvocations", "reasoning", "experimental_attachments", "data", "annotations"} {
		if _, ok := msg[key]; ok {
			return false
		}
	}

	partsRaw, ok := msg["parts"]
	if !ok {
		return false
	}
	parts, ok := partsRaw.([]any)
	if !ok {
		return false
	}

	for _, partRaw := range parts {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		if _, ok := part["metadata"]; ok {
			return true
		}

		// V4 tool: has toolInvocation; V5 tool: has toolCallId
		if _, ok := part["toolInvocation"]; ok {
			return false
		}
		if _, ok := part["toolCallId"]; ok {
			return true
		}

		partType, _ := part["type"].(string)
		if partType == "source" {
			return false
		}
		if partType == "source-url" {
			return true
		}

		if partType == "reasoning" {
			if _, ok := part["state"]; ok {
				return true
			}
			if _, ok := part["text"]; ok {
				return true
			}
			if _, ok := part["reasoning"]; ok {
				return false
			}
			if _, ok := part["details"]; ok {
				return false
			}
		}

		if partType == "file" {
			if _, ok := part["mediaType"]; ok {
				return true
			}
		}
	}

	return false // default to v4 for backwards compat
}

// HasAIV5CoreMessageCharacteristics checks if a message has AIV5 CoreMessage characteristics.
func HasAIV5CoreMessageCharacteristics(msg map[string]any) bool {
	if _, ok := msg["experimental_providerMetadata"]; ok {
		return false
	}

	content := msg["content"]
	if _, ok := content.(string); ok {
		return true
	}

	contentArr, ok := content.([]any)
	if !ok {
		return true
	}

	for _, partRaw := range contentArr {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		partType, _ := part["type"].(string)

		if partType == "tool-result" {
			if _, ok := part["output"]; ok {
				return true
			}
			if _, ok := part["result"]; ok {
				return false
			}
		}
		if partType == "tool-call" {
			if _, ok := part["input"]; ok {
				return true
			}
			if _, ok := part["args"]; ok {
				return false
			}
		}
		if _, ok := part["mediaType"]; ok {
			return true
		}
		if _, ok := part["mimeType"]; ok {
			return false
		}
		if _, ok := part["experimental_providerMetadata"]; ok {
			return false
		}
		if partType == "reasoning" {
			if _, ok := part["signature"]; ok {
				return false
			}
		}
		if partType == "redacted-reasoning" {
			return false
		}
	}

	return true
}

// GetRole returns the normalized role for a message.
// Maps "tool" role to "assistant".
func GetRole(msg map[string]any) string {
	role, _ := msg["role"].(string)
	switch role {
	case "assistant", "tool":
		return "assistant"
	case "user":
		return "user"
	case "system":
		return "system"
	default:
		return role
	}
}

// GetRoleFromDBMessage returns the normalized role for a MastraDBMessage.
func GetRoleFromDBMessage(msg *state.MastraDBMessage) string {
	switch msg.Role {
	case "assistant", "tool":
		return "assistant"
	case "user":
		return "user"
	case "system":
		return "system"
	default:
		return msg.Role
	}
}
