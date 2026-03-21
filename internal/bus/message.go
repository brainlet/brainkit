package bus

import (
	"encoding/json"
	"strings"
)

// Message is the bus message envelope.
type Message struct {
	// Protocol
	Version string `json:"v"` // "v1"

	// Routing
	Topic   string `json:"topic"`
	Address string `json:"addr,omitempty"` // "" = local, "kit:X", "host:X/kit:Y/agent:Z"
	ReplyTo string `json:"replyTo,omitempty"`

	// Identity
	CallerID string `json:"caller"`

	// Chain Tracking
	ID       string `json:"id"`
	ParentID string `json:"parent,omitempty"`
	TraceID  string `json:"trace"`
	Depth    int    `json:"depth,omitempty"`

	// Content
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"meta,omitempty"`
}

// MaxDepth is the maximum cascade depth before cycle detection triggers.
const MaxDepth = 16

// SubscriptionID identifies a subscription.
type SubscriptionID string

// ReplyFunc sends a response back to the Ask caller.
// If the message was a Send (no ReplyTo), calling reply is a no-op.
// Can only be called once — second call is no-op.
type ReplyFunc func(payload json.RawMessage)

// TopicMatches checks if a topic matches a pattern.
// "test.*" matches "test.foo", "test.foo.bar".
// "test.foo" matches only "test.foo".
func TopicMatches(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}
	return false
}
