package messages

// BusMessage is the ONLY interface that typed messages implement.
// Both request messages (for Ask) and event messages (for Send)
// implement it. The SDK uses BusTopic() to route messages.
type BusMessage interface {
	BusTopic() string
}

// Message is the internal platform envelope. Exposed in Client.Ask/On
// callbacks for advanced/raw access. Developers should prefer typed
// messages with sdk.Ask[Resp] for type-safe interactions.
type Message struct {
	Topic    string            `json:"topic"`
	Payload  []byte            `json:"payload"`
	CallerID string            `json:"callerId,omitempty"`
	TraceID  string            `json:"traceId,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// ReplyFunc sends a reply to the current inbound message.
// Used in On handlers when the sender used Ask.
type ReplyFunc func(payload any) error
