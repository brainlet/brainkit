package transport

import (
	"context"

	"github.com/google/uuid"
)

// Metadata carries bus message metadata.
type Metadata map[string]string

// Get returns a metadata value.
func (m Metadata) Get(key string) string {
	if m == nil {
		return ""
	}
	return m[key]
}

// Set writes a metadata value.
func (m Metadata) Set(key, value string) {
	if m == nil {
		return
	}
	m[key] = value
}

// Clone returns a detached copy of metadata.
func (m Metadata) Clone() Metadata {
	if len(m) == 0 {
		return Metadata{}
	}
	out := make(Metadata, len(m))
	for key, value := range m {
		out[key] = value
	}
	return out
}

// Message is Brainkit's internal transport message.
type Message struct {
	UUID     string
	Payload  []byte
	Metadata Metadata

	ctx  context.Context
	ack  func() bool
	nack func() bool
}

// NewMessage creates a transport message with a generated id.
func NewMessage(payload []byte) *Message {
	return NewMessageWithID(uuid.NewString(), payload)
}

// NewMessageWithID creates a transport message with an explicit id.
func NewMessageWithID(id string, payload []byte) *Message {
	if id == "" {
		id = uuid.NewString()
	}
	return &Message{
		UUID:     id,
		Payload:  append([]byte(nil), payload...),
		Metadata: Metadata{},
		ctx:      context.Background(),
		ack:      func() bool { return true },
		nack:     func() bool { return true },
	}
}

// NewReceivedMessage wraps a message received from an adapter.
func NewReceivedMessage(id string, payload []byte, metadata map[string]string, ctx context.Context, ack, nack func() bool) *Message {
	msg := NewMessageWithID(id, payload)
	msg.Metadata = Metadata(metadata).Clone()
	if ctx != nil {
		msg.ctx = ctx
	}
	if ack != nil {
		msg.ack = ack
	}
	if nack != nil {
		msg.nack = nack
	}
	return msg
}

// Context returns the message context.
func (m *Message) Context() context.Context {
	if m == nil || m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

// SetContext sets the message context.
func (m *Message) SetContext(ctx context.Context) {
	if m == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx = ctx
}

// Ack acknowledges successful handling.
func (m *Message) Ack() bool {
	if m == nil || m.ack == nil {
		return true
	}
	return m.ack()
}

// Nack rejects handling.
func (m *Message) Nack() bool {
	if m == nil || m.nack == nil {
		return true
	}
	return m.nack()
}

func (m *Message) clone() *Message {
	if m == nil {
		return nil
	}
	out := NewMessageWithID(m.UUID, m.Payload)
	out.Metadata = m.Metadata.Clone()
	out.ctx = m.Context()
	return out
}

// Publisher publishes messages to a transport topic.
type Publisher interface {
	Publish(topic string, messages ...*Message) error
}

// Subscriber subscribes to a transport topic.
type Subscriber interface {
	Subscribe(ctx context.Context, topic string) (<-chan *Message, error)
}

// HandlerFunc processes a message and can emit follow-up messages.
type HandlerFunc func(*Message) ([]*Message, error)

// NoPublishHandlerFunc processes a message without emitted messages.
type NoPublishHandlerFunc func(*Message) error

// Middleware wraps a handler.
type Middleware func(HandlerFunc) HandlerFunc
