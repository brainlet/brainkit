package bus

import (
	"encoding/json"
	"time"
)

// Message is a typed message on the platform bus.
type Message struct {
	ID       string            `json:"id"`
	Topic    string            `json:"topic"`
	CallerID string            `json:"callerId"`
	Payload  json.RawMessage   `json:"payload"`
	Metadata map[string]string `json:"metadata,omitempty"`

	// Causation chain for cycle detection
	TraceID  string `json:"traceId"`
	ParentID string `json:"parentId,omitempty"`
	Depth    int    `json:"depth"`

	// Request/response and streaming patterns
	ReplyTo  string `json:"replyTo,omitempty"`
	StreamTo string `json:"streamTo,omitempty"`

	// Internal
	CreatedAt time.Time `json:"-"`
}

// Response wraps a reply to a Request.
type Response struct {
	Message Message
	Err     error
}

// MaxDepth is the maximum causation chain depth before cycle detection triggers.
const MaxDepth = 16
