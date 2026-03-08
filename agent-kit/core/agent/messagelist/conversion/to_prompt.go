// Ported from: packages/core/src/agent/message-list/conversion/to-prompt.ts
package conversion

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
	utilspkg "github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/utils"
)

// LanguageModelV1Message represents a V1 language model prompt message.
// TODO: In TS this comes from @internal/ai-sdk-v4 LanguageModelV1Prompt[0].
type LanguageModelV1Message struct {
	Role            string         `json:"role"`
	Content         any            `json:"content"` // string or []map[string]any
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
	ExperimentalProviderMetadata map[string]any `json:"experimental_providerMetadata,omitempty"`
}

// LanguageModelV2Message represents a V2 language model prompt message.
// TODO: In TS this comes from @ai-sdk/provider-v5 LanguageModelV2Prompt[0].
type LanguageModelV2Message struct {
	Role            string         `json:"role"`
	Content         any            `json:"content"` // string or []map[string]any
	ProviderOptions map[string]any `json:"providerOptions,omitempty"`
}

// AIV4CoreMessageToV1PromptMessage converts an AI SDK V4 CoreMessage to a V1 LanguageModel prompt message.
// Used for creating LLM prompt messages without AI SDK streamText/generateText.
func AIV4CoreMessageToV1PromptMessage(coreMessage map[string]any) (map[string]any, error) {
	role, _ := coreMessage["role"].(string)
	content := coreMessage["content"]

	if role == "system" {
		return coreMessage, nil
	}

	if contentStr, ok := content.(string); ok {
		if role == "assistant" || role == "user" {
			result := make(map[string]any)
			for k, v := range coreMessage {
				result[k] = v
			}
			result["content"] = []map[string]any{
				{"type": "text", "text": contentStr},
			}
			return result, nil
		}
		return nil, fmt.Errorf("saw text content for input CoreMessage, but the role is %s. This is only allowed for \"system\", \"assistant\", and \"user\" roles", role)
	}

	roleContent := map[string][]map[string]any{
		"user":      {},
		"assistant": {},
		"tool":      {},
	}

	contentArr, ok := content.([]any)
	if !ok {
		return nil, fmt.Errorf("invalid content type for CoreMessage")
	}

	for _, partRaw := range contentArr {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		partType, _ := part["type"].(string)
		incompatibleMsg := fmt.Sprintf("saw incompatible message content part type %s for message role %s", partType, role)

		switch partType {
		case "text":
			if role == "tool" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			roleContent[role] = append(roleContent[role], part)

		case "redacted-reasoning", "reasoning":
			if role != "assistant" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			roleContent[role] = append(roleContent[role], part)

		case "tool-call":
			if role == "tool" || role == "user" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			if toolName, ok := part["toolName"].(string); ok {
				newPart["toolName"] = utilspkg.SanitizeToolName(toolName)
			}
			roleContent[role] = append(roleContent[role], newPart)

		case "tool-result":
			if role == "assistant" || role == "user" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			if toolName, ok := part["toolName"].(string); ok {
				newPart["toolName"] = utilspkg.SanitizeToolName(toolName)
			}
			roleContent[role] = append(roleContent[role], newPart)

		case "image":
			if role == "tool" || role == "assistant" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			imageData := part["image"]
			mimeType, _ := part["mimeType"].(string)

			if imageStr, ok := imageData.(string); ok {
				categorized := prompt.CategorizeFileData(imageStr, mimeType)
				if categorized.Type == "raw" {
					if mimeType == "" {
						mimeType = "image/png"
					}
					dataUri := prompt.CreateDataUri(imageStr, mimeType)
					newPart := make(map[string]any)
					for k, v := range part {
						newPart[k] = v
					}
					newPart["image"] = dataUri
					roleContent[role] = append(roleContent[role], newPart)
				} else {
					roleContent[role] = append(roleContent[role], part)
				}
			} else {
				roleContent[role] = append(roleContent[role], part)
			}

		case "file":
			if role == "tool" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			if dataStr, ok := part["data"].(string); ok {
				newPart["data"] = dataStr
			} else if dataBytes, ok := part["data"].([]byte); ok {
				newPart["data"] = prompt.ConvertDataContentToBase64String(dataBytes)
			}
			roleContent[role] = append(roleContent[role], newPart)
		}
	}

	result := make(map[string]any)
	for k, v := range coreMessage {
		result[k] = v
	}
	result["content"] = roleContent[role]

	return result, nil
}

// AIV5ModelMessageToV2PromptMessage converts an AI SDK V5 ModelMessage to a V2 LanguageModel prompt message.
// Used for creating LLM prompt messages without AI SDK streamText/generateText.
func AIV5ModelMessageToV2PromptMessage(modelMessage map[string]any) (map[string]any, error) {
	role, _ := modelMessage["role"].(string)
	content := modelMessage["content"]

	if role == "system" {
		return modelMessage, nil
	}

	if contentStr, ok := content.(string); ok {
		if role == "assistant" || role == "user" {
			return map[string]any{
				"role":            role,
				"content":         []map[string]any{{"type": "text", "text": contentStr}},
				"providerOptions": modelMessage["providerOptions"],
			}, nil
		}
		return nil, fmt.Errorf("saw text content for input ModelMessage, but the role is %s. This is only allowed for \"system\", \"assistant\", and \"user\" roles", role)
	}

	roleContent := map[string][]map[string]any{
		"user":      {},
		"assistant": {},
		"tool":      {},
	}

	contentArr, ok := content.([]any)
	if !ok {
		if typedArr, ok := content.([]map[string]any); ok {
			for _, item := range typedArr {
				contentArr = append(contentArr, item)
			}
		} else {
			return nil, fmt.Errorf("invalid content type for ModelMessage: %T", content)
		}
	}

	for _, partRaw := range contentArr {
		part, ok := partRaw.(map[string]any)
		if !ok {
			continue
		}

		partType, _ := part["type"].(string)
		incompatibleMsg := fmt.Sprintf("saw incompatible message content part type %s for message role %s", partType, role)

		switch partType {
		case "text":
			if role == "tool" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			roleContent[role] = append(roleContent[role], part)

		case "reasoning":
			if role == "tool" || role == "user" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			roleContent[role] = append(roleContent[role], part)

		case "tool-call":
			if role != "assistant" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			if toolName, ok := part["toolName"].(string); ok {
				newPart["toolName"] = utilspkg.SanitizeToolName(toolName)
			}
			roleContent[role] = append(roleContent[role], newPart)

		case "tool-result":
			if role == "assistant" || role == "user" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			newPart := make(map[string]any)
			for k, v := range part {
				newPart[k] = v
			}
			if toolName, ok := part["toolName"].(string); ok {
				newPart["toolName"] = utilspkg.SanitizeToolName(toolName)
			}
			roleContent[role] = append(roleContent[role], newPart)

		case "file":
			if role == "tool" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			roleContent[role] = append(roleContent[role], part)

		case "image":
			if role == "tool" {
				return nil, fmt.Errorf("%s", incompatibleMsg)
			}
			mediaType, _ := part["mediaType"].(string)
			if mediaType == "" {
				mediaType = "image/unknown"
			}
			newPart := map[string]any{
				"type":      "file",
				"mediaType": mediaType,
			}
			if d, ok := part["image"]; ok {
				newPart["data"] = d
			}
			roleContent[role] = append(roleContent[role], newPart)
		}
	}

	result := make(map[string]any)
	for k, v := range modelMessage {
		result[k] = v
	}
	result["content"] = roleContent[role]

	return result, nil
}

// ensure json import is used
var _ = json.Marshal
