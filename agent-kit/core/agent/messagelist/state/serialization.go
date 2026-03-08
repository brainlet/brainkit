// Ported from: packages/core/src/agent/message-list/state/serialization.ts
package state

import (
	"time"
)

// SerializedMessage is a MastraDBMessage where CreatedAt is stored as a string.
type SerializedMessage struct {
	ID         string                 `json:"id"`
	Role       string                 `json:"role"`
	CreatedAt  string                 `json:"createdAt"`
	ThreadID   string                 `json:"threadId,omitempty"`
	ResourceID string                 `json:"resourceId,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Content    MastraMessageContentV2 `json:"content"`
}

// SerializeMessage converts a MastraDBMessage to SerializedMessage by converting Date to string.
func SerializeMessage(message MastraDBMessage) SerializedMessage {
	return SerializedMessage{
		ID:         message.ID,
		Role:       message.Role,
		CreatedAt:  message.CreatedAt.UTC().Format(time.RFC1123),
		ThreadID:   message.ThreadID,
		ResourceID: message.ResourceID,
		Type:       message.Type,
		Content:    message.Content,
	}
}

// DeserializeMessage converts a SerializedMessage back to MastraDBMessage.
func DeserializeMessage(message SerializedMessage) MastraDBMessage {
	createdAt, err := time.Parse(time.RFC1123, message.CreatedAt)
	if err != nil {
		createdAt = time.Now()
	}
	return MastraDBMessage{
		MastraMessageShared: MastraMessageShared{
			ID:         message.ID,
			Role:       message.Role,
			CreatedAt:  createdAt,
			ThreadID:   message.ThreadID,
			ResourceID: message.ResourceID,
			Type:       message.Type,
		},
		Content: message.Content,
	}
}

// SerializeMessages converts an array of MastraDBMessage to SerializedMessage.
func SerializeMessages(messages []MastraDBMessage) []SerializedMessage {
	result := make([]SerializedMessage, len(messages))
	for i, m := range messages {
		result[i] = SerializeMessage(m)
	}
	return result
}

// DeserializeMessages converts an array of SerializedMessage to MastraDBMessage.
func DeserializeMessages(messages []SerializedMessage) []MastraDBMessage {
	result := make([]MastraDBMessage, len(messages))
	for i, m := range messages {
		result[i] = DeserializeMessage(m)
	}
	return result
}
