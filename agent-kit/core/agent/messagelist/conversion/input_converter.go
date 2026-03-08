// Ported from: packages/core/src/agent/message-list/conversion/input-converter.ts
package conversion

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/adapters"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/detection"
	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

// InputConversionContext provides context required for input conversion functions.
// This is passed from MessageList to provide access to instance-specific utilities.
type InputConversionContext struct {
	MemoryInfo      *state.MemoryInfo
	NewMessageID    func() string
	GenerateCreatedAt func(messageSource state.MessageSource, start ...any) time.Time
	// DBMessages array for looking up tool call args
	DBMessages []*state.MastraDBMessage
}

// InputToMastraDBMessage converts any supported message input format to MastraDBMessage.
// Routes to the appropriate converter based on message type detection.
func InputToMastraDBMessage(
	message map[string]any,
	messageSource state.MessageSource,
	context *InputConversionContext,
) (*state.MastraDBMessage, error) {
	// Validate threadId matches (except for memory messages which can come from other threads)
	if messageSource != state.MessageSourceMemory {
		if threadID, ok := message["threadId"].(string); ok && threadID != "" && context.MemoryInfo != nil {
			if threadID != context.MemoryInfo.ThreadID {
				return nil, fmt.Errorf("received input message with wrong threadId. Input %s, expected %s", threadID, context.MemoryInfo.ThreadID)
			}
		}
	}

	// Validate resourceId matches
	if resourceID, ok := message["resourceId"].(string); ok && resourceID != "" {
		if context.MemoryInfo != nil && context.MemoryInfo.ResourceID != "" {
			if resourceID != context.MemoryInfo.ResourceID {
				return nil, fmt.Errorf("received input message with wrong resourceId. Input %s, expected %s", resourceID, context.MemoryInfo.ResourceID)
			}
		}
	}

	if detection.IsMastraMessageV1(message) {
		return MastraMessageV1ToMastraDBMessage(message, messageSource, context), nil
	}
	if detection.IsMastraDBMessage(message) {
		return HydrateMastraDBMessageFields(mapToMastraDBMessage(message), context), nil
	}
	if detection.IsAIV4CoreMessage(message) {
		adapterCtx := &adapters.AIV4AdapterContext{
			MemoryInfo:      context.MemoryInfo,
			NewMessageID:    context.NewMessageID,
			GenerateCreatedAt: func(ms state.MessageSource, start ...any) time.Time {
				return context.GenerateCreatedAt(ms, start...)
			},
			DBMessages: context.DBMessages,
		}
		return adapters.AIV4FromCoreMessage(message, adapterCtx, messageSource), nil
	}
	if detection.IsAIV4UIMessage(message) {
		adapterCtx := &adapters.AIV4AdapterContext{
			MemoryInfo:      context.MemoryInfo,
			NewMessageID:    context.NewMessageID,
			GenerateCreatedAt: func(ms state.MessageSource, start ...any) time.Time {
				return context.GenerateCreatedAt(ms, start...)
			},
			DBMessages: context.DBMessages,
		}
		uiMsg := mapToUIMessageWithMetadata(message)
		return adapters.AIV4FromUIMessage(uiMsg, adapterCtx, messageSource), nil
	}

	// Use custom ID generator if message doesn't have an ID, otherwise keep the original
	hasOriginalID := false
	id := ""
	if v, ok := message["id"].(string); ok && v != "" {
		hasOriginalID = true
		id = v
	}
	if !hasOriginalID {
		id = context.NewMessageID()
	}

	if detection.IsAIV5CoreMessage(message) {
		dbMsg := adapters.AIV5FromModelMessage(message, string(messageSource))
		// Only use the original createdAt from input message metadata, not the generated one from the static method
		var rawCreatedAt any
		if metadata, ok := message["metadata"].(map[string]any); ok {
			rawCreatedAt = metadata["createdAt"]
		}
		dbMsg.ID = id
		dbMsg.CreatedAt = context.GenerateCreatedAt(messageSource, rawCreatedAt)
		if context.MemoryInfo != nil {
			dbMsg.ThreadID = context.MemoryInfo.ThreadID
			dbMsg.ResourceID = context.MemoryInfo.ResourceID
		}
		return dbMsg, nil
	}

	if detection.IsAIV5UIMessage(message) {
		v5UIMsg := mapToAIV5UIMessage(message)
		dbMsg := adapters.AIV5FromUIMessage(v5UIMsg)
		// Only use the original createdAt from input message, not the generated one from the static method
		var rawCreatedAt any
		if v, ok := message["createdAt"]; ok {
			rawCreatedAt = v
		}
		dbMsg.ID = id
		dbMsg.CreatedAt = context.GenerateCreatedAt(messageSource, rawCreatedAt)
		if context.MemoryInfo != nil {
			dbMsg.ThreadID = context.MemoryInfo.ThreadID
			dbMsg.ResourceID = context.MemoryInfo.ResourceID
		}
		return dbMsg, nil
	}

	msgJSON, _ := json.Marshal(message)
	return nil, fmt.Errorf("found unhandled message %s", string(msgJSON))
}

// InputToMastraDBMessageTyped converts a typed message input to MastraDBMessage.
// This is a convenience wrapper for typed MastraDBMessage inputs.
func InputToMastraDBMessageTyped(
	message *state.MastraDBMessage,
	context *InputConversionContext,
) *state.MastraDBMessage {
	return HydrateMastraDBMessageFields(message, context)
}

// MastraMessageV1ToMastraDBMessage converts MastraMessageV1 format to MastraDBMessage.
func MastraMessageV1ToMastraDBMessage(
	message map[string]any,
	messageSource state.MessageSource,
	context *InputConversionContext,
) *state.MastraDBMessage {
	content := message["content"]
	role, _ := message["role"].(string)

	adapterCtx := &adapters.AIV4AdapterContext{
		MemoryInfo:      context.MemoryInfo,
		NewMessageID:    context.NewMessageID,
		GenerateCreatedAt: func(ms state.MessageSource, start ...any) time.Time {
			return context.GenerateCreatedAt(ms, start...)
		},
		DBMessages: context.DBMessages,
	}

	coreMsg := map[string]any{
		"content": content,
		"role":    role,
	}
	coreV2 := adapters.AIV4FromCoreMessage(coreMsg, adapterCtx, messageSource)

	id, _ := message["id"].(string)
	threadID, _ := message["threadId"].(string)
	resourceID, _ := message["resourceId"].(string)

	var rawCreatedAt any
	if v, ok := message["createdAt"]; ok {
		rawCreatedAt = v
	}

	return &state.MastraDBMessage{
		MastraMessageShared: state.MastraMessageShared{
			ID:         id,
			Role:       coreV2.Role,
			CreatedAt:  context.GenerateCreatedAt(messageSource, rawCreatedAt),
			ThreadID:   threadID,
			ResourceID: resourceID,
		},
		Content: coreV2.Content,
	}
}

// HydrateMastraDBMessageFields hydrates a MastraDBMessage with missing fields (id, createdAt, threadId, resourceId).
// Also fixes toolInvocations with empty args by looking in the parts array.
func HydrateMastraDBMessageFields(
	message *state.MastraDBMessage,
	context *InputConversionContext,
) *state.MastraDBMessage {
	// Generate ID if missing
	if message.ID == "" {
		message.ID = context.NewMessageID()
	}

	// Fix toolInvocations with empty args by looking in the parts array
	if len(message.Content.ToolInvocations) > 0 && len(message.Content.Parts) > 0 {
		for i := range message.Content.ToolInvocations {
			ti := &message.Content.ToolInvocations[i]
			if ti.Args == nil || len(ti.Args) == 0 {
				for _, part := range message.Content.Parts {
					if part.Type == "tool-invocation" && part.ToolInvocation != nil &&
						part.ToolInvocation.ToolCallID == ti.ToolCallID &&
						part.ToolInvocation.Args != nil && len(part.ToolInvocation.Args) > 0 {
						ti.Args = part.ToolInvocation.Args
						break
					}
				}
			}
		}
	}

	if message.ThreadID == "" && context.MemoryInfo != nil && context.MemoryInfo.ThreadID != "" {
		message.ThreadID = context.MemoryInfo.ThreadID
		if message.ResourceID == "" && context.MemoryInfo.ResourceID != "" {
			message.ResourceID = context.MemoryInfo.ResourceID
		}
	}

	return message
}

// mapToMastraDBMessage converts a map[string]any to a MastraDBMessage.
// This is a best-effort conversion for the typed interface.
func mapToMastraDBMessage(msg map[string]any) *state.MastraDBMessage {
	// If we already have a typed message, return it directly
	result := &state.MastraDBMessage{}
	result.ID, _ = msg["id"].(string)
	result.Role, _ = msg["role"].(string)
	result.ThreadID, _ = msg["threadId"].(string)
	result.ResourceID, _ = msg["resourceId"].(string)

	if createdAt, ok := msg["createdAt"].(time.Time); ok {
		result.CreatedAt = createdAt
	} else if s, ok := msg["createdAt"].(string); ok {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			result.CreatedAt = t
		}
	}

	if contentMap, ok := msg["content"].(map[string]any); ok {
		if format, ok := contentMap["format"]; ok {
			switch v := format.(type) {
			case float64:
				result.Content.Format = int(v)
			case int:
				result.Content.Format = v
			}
		}
		if parts, ok := contentMap["parts"].([]state.MastraMessagePart); ok {
			result.Content.Parts = parts
		}
		if content, ok := contentMap["content"].(string); ok {
			result.Content.Content = content
		}
		if meta, ok := contentMap["metadata"].(map[string]any); ok {
			result.Content.Metadata = meta
		}
	}

	return result
}

// mapToUIMessageWithMetadata converts a map[string]any to a UIMessageWithMetadata.
func mapToUIMessageWithMetadata(msg map[string]any) *state.UIMessageWithMetadata {
	result := &state.UIMessageWithMetadata{}
	result.ID, _ = msg["id"].(string)
	result.Role, _ = msg["role"].(string)
	result.Content = msg["content"]

	if createdAt, ok := msg["createdAt"].(time.Time); ok {
		result.CreatedAt = createdAt
	}
	if parts, ok := msg["parts"].([]state.MastraMessagePart); ok {
		result.Parts = parts
	}
	if meta, ok := msg["metadata"].(map[string]any); ok {
		result.Metadata = meta
	}

	return result
}

// mapToAIV5UIMessage converts a map[string]any to an AIV5UIMessage.
func mapToAIV5UIMessage(msg map[string]any) *adapters.AIV5UIMessage {
	result := &adapters.AIV5UIMessage{}
	result.ID, _ = msg["id"].(string)
	result.Role, _ = msg["role"].(string)

	if meta, ok := msg["metadata"].(map[string]any); ok {
		result.Metadata = meta
	}

	if partsRaw, ok := msg["parts"].([]any); ok {
		for _, partRaw := range partsRaw {
			if partMap, ok := partRaw.(map[string]any); ok {
				part := adapters.AIV5UIPart{}
				part.Type, _ = partMap["type"].(string)
				part.Text, _ = partMap["text"].(string)
				part.ToolCallID, _ = partMap["toolCallId"].(string)
				part.State, _ = partMap["state"].(string)
				part.Input = partMap["input"]
				part.Output = partMap["output"]
				part.URL, _ = partMap["url"].(string)
				part.MediaType, _ = partMap["mediaType"].(string)
				part.Filename, _ = partMap["filename"].(string)
				part.SourceID, _ = partMap["sourceId"].(string)
				part.Title, _ = partMap["title"].(string)
				part.DataPayload = partMap["data"]
				result.Parts = append(result.Parts, part)
			}
		}
	}

	return result
}
