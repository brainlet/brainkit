package sdk

import (
	"context"
	"encoding/json"
	"fmt"
)

// Emit sends a fire-and-forget event. No replyTo, no response expected.
func Emit[T BrainkitMessage](rt Runtime, ctx context.Context, msg T) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %T: %w", msg, err)
	}
	_, err = rt.PublishRaw(ctx, msg.BusTopic(), payload)
	return err
}

// SubscribeTo listens for typed messages on a specific topic. Wire envelopes
// are unwrapped: success envelopes decode their `data` field into T, error
// envelopes invoke the handler with a zero T so callers can inspect the
// failure via suite/helper functions that read the raw `msg.Metadata` +
// `msg.Payload` (e.g. `sdk.DecodeEnvelope(msg.Payload)` or the test
// `suite.ResponseErrorMessage`).
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, Message)) (func(), error) {
	return rt.SubscribeRaw(ctx, topic, func(msg Message) {
		payload := msg.Payload
		if msg.Metadata["envelope"] == "true" {
			env, err := DecodeEnvelope(payload)
			if err != nil {
				return
			}
			if !env.Ok {
				var zero T
				handler(zero, msg)
				return
			}
			payload = env.Data
		}
		var typed T
		if err := json.Unmarshal(payload, &typed); err != nil {
			return
		}
		handler(typed, msg)
	})
}
