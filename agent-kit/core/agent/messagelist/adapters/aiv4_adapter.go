// Ported from: packages/core/src/agent/message-list/adapters/AIV4Adapter.ts
package adapters

import (
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/detection"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/prompt"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// filterDataParts filters out data-* parts from MastraMessagePart slice to get V4-compatible parts.
func filterDataParts(parts []MastraMessagePart) []MastraMessagePart {
	var result []MastraMessagePart
	for _, part := range parts {
		if !strings.HasPrefix(part.Type, "data-") {
			result = append(result, part)
		}
	}
	return result
}

// filterEmptyTextParts filters out empty text parts from message parts array.
// However, if the only part is an empty text part, it is preserved as a legitimate placeholder.
func filterEmptyTextParts(parts []MastraMessagePart) []MastraMessagePart {
	hasNonEmpty := false
	for _, part := range parts {
		if !(part.Type == "text" && part.Text == "") {
			hasNonEmpty = true
			break
		}
	}
	if !hasNonEmpty {
		return parts
	}
	var result []MastraMessagePart
	for _, part := range parts {
		if part.Type == "text" && part.Text == "" {
			continue
		}
		result = append(result, part)
	}
	return result
}

// AIV4Adapter handles conversions between MastraDBMessage and AI SDK V4 formats.
type AIV4Adapter struct{}

// ToUIMessage converts MastraDBMessage to AI SDK V4 UIMessageWithMetadata.
func (a *AIV4Adapter) ToUIMessage(m *MastraDBMessage) *UIMessageWithMetadata {
	return AIV4ToUIMessage(m)
}

// AIV4ToUIMessage is the static version of ToUIMessage for use without an instance.
func AIV4ToUIMessage(m *MastraDBMessage) *UIMessageWithMetadata {
	var experimentalAttachments []state.ExperimentalAttachment
	if len(m.Content.ExperimentalAttachments) > 0 {
		experimentalAttachments = append(experimentalAttachments, m.Content.ExperimentalAttachments...)
	}

	// Compute content string
	contentString := ""
	if s, ok := m.Content.Content, m.Content.Content != ""; ok && s != "" {
		contentString = s
	} else {
		for _, part := range m.Content.Parts {
			if part.Type == "text" {
				contentString = part.Text // return only the last text part like AI SDK does
			}
		}
	}

	var parts []MastraMessagePart
	sourceParts := m.Content.Parts

	if len(sourceParts) > 0 {
		for _, part := range sourceParts {
			if part.Type == "file" {
				// Normalize file data
				normalizedURL := part.Data
				if normalizedURL != "" {
					categorized := prompt.CategorizeFileData(normalizedURL, part.MimeType)
					if categorized.Type == "raw" {
						mt := part.MimeType
						if mt == "" {
							mt = "application/octet-stream"
						}
						normalizedURL = prompt.CreateDataUri(normalizedURL, mt)
					}
				}
				experimentalAttachments = append(experimentalAttachments, state.ExperimentalAttachment{
					ContentType: part.MimeType,
					URL:         normalizedURL,
				})
			} else if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
				(part.ToolInvocation.State == "call" || part.ToolInvocation.State == "partial-call") {
				continue
			} else if part.Type == "tool-invocation" && part.ToolInvocation != nil {
				// Handle tool invocations with step number logic
				inv := *part.ToolInvocation
				currentStep := -1
				toolStep := -1
				for _, innerPart := range sourceParts {
					if innerPart.Type == "step-start" {
						currentStep++
					}
					if innerPart.Type == "tool-invocation" && innerPart.ToolInvocation != nil &&
						innerPart.ToolInvocation.ToolCallID == part.ToolInvocation.ToolCallID {
						toolStep = currentStep
						break
					}
				}
				if toolStep >= 0 {
					inv.Step = &toolStep
				}
				parts = append(parts, MastraMessagePart{
					Type:           "tool-invocation",
					ToolInvocation: &inv,
				})
			} else {
				parts = append(parts, part)
			}
		}
	}

	if len(parts) == 0 && len(experimentalAttachments) > 0 {
		parts = append(parts, MastraMessagePart{Type: "text", Text: ""})
	}

	v4Parts := filterDataParts(parts)

	uiMessage := &UIMessageWithMetadata{
		ID:                      m.ID,
		Role:                    m.Role,
		Content:                 contentString,
		CreatedAt:               m.CreatedAt,
		Parts:                   v4Parts,
		ExperimentalAttachments: experimentalAttachments,
	}

	if m.Role == "assistant" {
		// Filter toolInvocations to only results
		if len(m.Content.ToolInvocations) > 0 {
			var resultInvocations []ToolInvocation
			for _, ti := range m.Content.ToolInvocations {
				if ti.State == "result" {
					resultInvocations = append(resultInvocations, ti)
				}
			}
			uiMessage.ToolInvocations = resultInvocations
		}
	}

	if m.Content.Metadata != nil {
		uiMessage.Metadata = m.Content.Metadata
	}

	return uiMessage
}

// AIV4SystemToV4Core converts a MastraDBMessage system message directly to CoreMessage format.
func AIV4SystemToV4Core(message *MastraDBMessage) (map[string]any, error) {
	if message.Role != "system" || message.Content.Content == "" {
		return nil, fmt.Errorf("invalid system message format: must include 'role' and 'content'")
	}

	result := map[string]any{
		"role":    "system",
		"content": message.Content.Content,
	}

	if message.Content.ProviderMetadata != nil {
		result["experimental_providerMetadata"] = message.Content.ProviderMetadata
	}

	return result, nil
}

// AIV4FromUIMessage converts AI SDK V4 UIMessage to MastraDBMessage.
func AIV4FromUIMessage(
	message *UIMessageWithMetadata,
	ctx *AIV4AdapterContext,
	messageSource MessageSource,
) *MastraDBMessage {
	filteredParts := filterEmptyTextParts(message.Parts)

	content := MastraMessageContentV2{
		Format: 2,
		Parts:  filteredParts,
	}

	if len(message.ToolInvocations) > 0 {
		content.ToolInvocations = message.ToolInvocations
	}
	if len(message.ExperimentalAttachments) > 0 {
		content.ExperimentalAttachments = message.ExperimentalAttachments
	}
	if message.Metadata != nil {
		content.Metadata = message.Metadata
	}

	id := message.ID
	if id == "" {
		id = ctx.NewMessageID()
	}

	msgMap := map[string]any{"role": message.Role}
	role := detection.GetRole(msgMap)

	createdAt := ctx.GenerateCreatedAt(messageSource, message.CreatedAt)

	result := &MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:        id,
			Role:      role,
			CreatedAt: createdAt,
		},
		Content: content,
	}
	if ctx.MemoryInfo != nil {
		result.ThreadID = ctx.MemoryInfo.ThreadID
		result.ResourceID = ctx.MemoryInfo.ResourceID
	}

	return result
}

// AIV4FromCoreMessage converts AI SDK V4 CoreMessage to MastraDBMessage.
// coreMessage is represented as map[string]any since CoreMessage is a complex union type.
func AIV4FromCoreMessage(
	coreMessage map[string]any,
	ctx *AIV4AdapterContext,
	messageSource MessageSource,
) *MastraDBMessage {
	id, _ := coreMessage["id"].(string)
	if id == "" {
		id = ctx.NewMessageID()
	}

	var parts []MastraMessagePart
	var experimentalAttachments []state.ExperimentalAttachment
	var toolInvocations []ToolInvocation
	role, _ := coreMessage["role"].(string)

	content := coreMessage["content"]

	if contentStr, ok := content.(string); ok {
		parts = append(parts, MastraMessagePart{
			Type: "text",
			Text: contentStr,
		})

		filteredParts := filterEmptyTextParts(parts)
		contentV2 := MastraMessageContentV2{
			Format:  2,
			Parts:   filteredParts,
			Content: contentStr,
		}

		msgRole := detection.GetRole(coreMessage)

		var rawCreatedAt any
		if metadata, ok := coreMessage["metadata"].(map[string]any); ok {
			rawCreatedAt = metadata["createdAt"]
		}

		createdAt := ctx.GenerateCreatedAt(messageSource, rawCreatedAt)

		result := &MastraDBMessage{
			MastraMessageShared: state.MastraMessageShared{
				ID:        id,
				Role:      msgRole,
				CreatedAt: createdAt,
			},
			Content: contentV2,
		}
		if ctx.MemoryInfo != nil {
			result.ThreadID = ctx.MemoryInfo.ThreadID
			result.ResourceID = ctx.MemoryInfo.ResourceID
		}
		return result
	}

	contentArr, ok := content.([]any)
	if ok {
		for _, partRaw := range contentArr {
			part, ok := partRaw.(map[string]any)
			if !ok {
				continue
			}
			partType, _ := part["type"].(string)
			switch partType {
			case "text":
				text, _ := part["text"].(string)
				prevPart := ""
				if len(parts) > 0 {
					prevPart = parts[len(parts)-1].Type
				}
				if role == "assistant" && prevPart == "tool-invocation" {
					parts = append(parts, MastraMessagePart{Type: "step-start"})
				}
				mp := MastraMessagePart{Type: "text", Text: text}
				if po, ok := part["providerOptions"].(map[string]map[string]any); ok {
					mp.ProviderMetadata = state.ProviderMetadata(po)
				}
				parts = append(parts, mp)

			case "tool-call":
				toolCallID, _ := part["toolCallId"].(string)
				toolName, _ := part["toolName"].(string)
				args, _ := part["args"].(map[string]any)
				mp := MastraMessagePart{
					Type: "tool-invocation",
					ToolInvocation: &ToolInvocation{
						State:      "call",
						ToolCallID: toolCallID,
						ToolName:   toolName,
						Args:       args,
					},
				}
				parts = append(parts, mp)

			case "tool-result":
				toolCallID, _ := part["toolCallId"].(string)
				toolName, _ := part["toolName"].(string)
				result := part["result"]
				if result == nil {
					result = ""
				}
				// Try to find args from corresponding tool-call
				toolArgs := make(map[string]any)
				if ctx.DBMessages != nil {
					toolArgs = findToolCallArgsFromDB(ctx.DBMessages, toolCallID)
				}

				inv := ToolInvocation{
					State:      "result",
					ToolCallID: toolCallID,
					ToolName:   toolName,
					Result:     result,
					Args:       toolArgs,
				}
				mp := MastraMessagePart{
					Type:           "tool-invocation",
					ToolInvocation: &inv,
				}
				parts = append(parts, mp)
				toolInvocations = append(toolInvocations, inv)

			case "reasoning":
				text, _ := part["text"].(string)
				signature, _ := part["signature"].(string)
				mp := MastraMessagePart{
					Type:      "reasoning",
					Reasoning: "",
					Details:   []state.ReasoningDetail{{Type: "text", Text: text, Signature: signature}},
				}
				parts = append(parts, mp)

			case "redacted-reasoning":
				data, _ := part["data"].(string)
				mp := MastraMessagePart{
					Type:      "reasoning",
					Reasoning: "",
					Details:   []state.ReasoningDetail{{Type: "redacted", Data: data}},
				}
				parts = append(parts, mp)

			case "image":
				image := part["image"]
				mimeType, _ := part["mimeType"].(string)
				mp := MastraMessagePart{
					Type:     "file",
					Data:     prompt.ImageContentToString(image),
					MimeType: mimeType,
				}
				parts = append(parts, mp)

			case "file":
				data := part["data"]
				mimeType, _ := part["mimeType"].(string)
				if dataStr, ok := data.(string); ok {
					mp := MastraMessagePart{
						Type:     "file",
						Data:     dataStr,
						MimeType: mimeType,
					}
					if fn, ok := part["filename"].(string); ok {
						mp.Filename = fn
					}
					parts = append(parts, mp)
				}
			}
		}
	}

	filteredParts := filterEmptyTextParts(parts)

	contentV2 := MastraMessageContentV2{
		Format: 2,
		Parts:  filteredParts,
	}
	if len(toolInvocations) > 0 {
		contentV2.ToolInvocations = toolInvocations
	}
	if len(experimentalAttachments) > 0 {
		contentV2.ExperimentalAttachments = experimentalAttachments
	}
	if po, ok := coreMessage["providerOptions"].(map[string]map[string]any); ok {
		contentV2.ProviderMetadata = state.ProviderMetadata(po)
	}

	msgRole := detection.GetRole(coreMessage)

	var rawCreatedAt any
	if metadata, ok := coreMessage["metadata"].(map[string]any); ok {
		rawCreatedAt = metadata["createdAt"]
	}

	createdAt := ctx.GenerateCreatedAt(messageSource, rawCreatedAt)

	result := &MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:        id,
			Role:      msgRole,
			CreatedAt: createdAt,
		},
		Content: contentV2,
	}
	if ctx.MemoryInfo != nil {
		result.ThreadID = ctx.MemoryInfo.ThreadID
		result.ResourceID = ctx.MemoryInfo.ResourceID
	}

	return result
}

// findToolCallArgsFromDB searches DB messages for tool call args matching a toolCallId.
func findToolCallArgsFromDB(dbMessages []*MastraDBMessage, toolCallID string) map[string]any {
	for i := len(dbMessages) - 1; i >= 0; i-- {
		msg := dbMessages[i]
		if msg.Role != "assistant" {
			continue
		}
		for _, part := range msg.Content.Parts {
			if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
				part.ToolInvocation.ToolCallID == toolCallID {
				if part.ToolInvocation.Args != nil {
					return part.ToolInvocation.Args
				}
				return map[string]any{}
			}
		}
		for _, ti := range msg.Content.ToolInvocations {
			if ti.ToolCallID == toolCallID {
				if ti.Args != nil {
					return ti.Args
				}
				return map[string]any{}
			}
		}
	}
	return map[string]any{}
}

// ensure contentString is used to suppress unused variable warning
var _ = func() time.Time { return time.Time{} }
