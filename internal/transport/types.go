package transport

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk"
)

// Envelope wraps an internal transport message with a typed payload.
// Developers work with the typed Value. Raw carries adapter metadata and Ack/Nack.
type Envelope[T sdk.BrainkitMessage] struct {
	Raw   *Message // transport message ID, metadata, Ack(), Nack()
	Value T        // Typed payload
}

// DecodeEnvelope deserializes an internal transport message into a typed Envelope.
func DecodeEnvelope[T sdk.BrainkitMessage](msg *Message) (Envelope[T], error) {
	var v T
	if err := json.Unmarshal(msg.Payload, &v); err != nil {
		return Envelope[T]{}, fmt.Errorf("decode %T: %w", v, err)
	}
	return Envelope[T]{Raw: msg, Value: v}, nil
}
