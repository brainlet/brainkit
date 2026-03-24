package messages

import "encoding/json"

// BrainkitMessage is the interface all typed messages implement.
// The BusTopic() return value is the Watermill routing key.
//
// The spec describes this as a union constraint (AiGenerateMsg | AiStreamMsg | ...).
// Go does not support union constraints with methods across 100+ unrelated structs.
// In practice this is an interface — restriction is by convention (only sdk/messages/ types).
type BrainkitMessage interface {
	BusTopic() string
}

// ResultMeta carries the common typed failure shape for transport-visible
// command results.
type ResultMeta struct {
	Error string `json:"error,omitempty"`
}

func (m *ResultMeta) SetError(err string) {
	m.Error = err
}

func (m ResultMeta) ResultError() string {
	return m.Error
}

func ResultErrorOf(v any) string {
	carrier, ok := v.(interface{ ResultError() string })
	if !ok {
		return ""
	}
	return carrier.ResultError()
}

// Message is the internal platform envelope used by transport-backed publish
// and subscribe helpers for advanced raw access.
type Message struct {
	Topic    string            `json:"topic"`
	Payload  []byte            `json:"payload"`
	CallerID string            `json:"callerId,omitempty"`
	TraceID  string            `json:"traceId,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CustomMsg is a generic message for sending to any topic.
// Used by Go Direct/Plugin to send messages to .ts deployments.
// MarshalJSON serializes only the Payload (not the Topic wrapper),
// so the .ts subscriber receives the inner payload directly as msg.payload.
type CustomMsg struct {
	Topic   string          `json:"-"`
	Payload json.RawMessage `json:"payload"`
}

func (m CustomMsg) BusTopic() string { return m.Topic }

func (m CustomMsg) MarshalJSON() ([]byte, error) {
	if len(m.Payload) > 0 {
		return []byte(m.Payload), nil
	}
	return []byte("null"), nil
}
