package messaging

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/sdk/messages"
)

// Envelope wraps a Watermill message with a typed payload.
// Developers work with the typed Value. Raw carries internal metadata and Ack/Nack.
type Envelope[T messages.BrainkitMessage] struct {
	Raw   *message.Message // Watermill: UUID, Metadata, Ack(), Nack()
	Value T                // Typed payload
}

// DecodeEnvelope deserializes a Watermill message into a typed Envelope.
func DecodeEnvelope[T messages.BrainkitMessage](msg *message.Message) (Envelope[T], error) {
	var v T
	if err := json.Unmarshal(msg.Payload, &v); err != nil {
		return Envelope[T]{}, fmt.Errorf("decode %T: %w", v, err)
	}
	return Envelope[T]{Raw: msg, Value: v}, nil
}
