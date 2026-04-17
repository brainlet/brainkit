package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/google/uuid"
)

// PublishResult contains metadata about a published command.
type PublishResult struct {
	MessageID     string // Watermill message UUID
	CorrelationID string // for response filtering
	ReplyTo       string // where responses will be sent (always populated for commands)
	Topic         string // where the message was published
}

type publishConfig struct {
	replyTo string
}

// PublishOption configures a Publish call.
type PublishOption func(*publishConfig)

// WithReplyTo overrides the auto-generated reply topic.
func WithReplyTo(topic string) PublishOption {
	return func(c *publishConfig) { c.replyTo = topic }
}

// Publish sends a typed command. Always generates a replyTo for response routing.
// Default convention: <topic>.reply.<uuid>
func Publish[T BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error) {
	cfg := publishConfig{}
	for _, opt := range opts {
		opt(&cfg)
	}

	topic := msg.BusTopic()
	correlationID := uuid.NewString()
	replyTo := cfg.replyTo
	if replyTo == "" {
		replyTo = topic + ".reply." + correlationID
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return PublishResult{}, fmt.Errorf("marshal %T: %w", msg, err)
	}

	ctx = ctxkeys.WithPublishMeta(ctx, correlationID, replyTo)
	msgID, err := rt.PublishRaw(ctx, topic, payload)
	if err != nil {
		return PublishResult{}, err
	}

	return PublishResult{
		MessageID:     msgID,
		CorrelationID: correlationID,
		ReplyTo:       replyTo,
		Topic:         topic,
	}, nil
}

// Emit sends a fire-and-forget event. No replyTo, no response expected.
func Emit[T BrainkitMessage](rt Runtime, ctx context.Context, msg T) error {
	payload, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal %T: %w", msg, err)
	}
	_, err = rt.PublishRaw(ctx, msg.BusTopic(), payload)
	return err
}

// SubscribeTo listens for typed messages on a specific topic. If the message
// is a wire envelope (metadata["envelope"]="true"), success data is decoded
// into T; error envelopes are flattened into legacy {error, code, details}
// shape before decoding so response types that still embed ResultMeta keep
// working during the migration.
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, Message)) (func(), error) {
	return rt.SubscribeRaw(ctx, topic, func(msg Message) {
		payload := msg.Payload
		if msg.Metadata["envelope"] == "true" {
			env, err := DecodeEnvelope(payload)
			if err != nil {
				return
			}
			if env.Ok {
				payload = env.Data
			} else if env.Error != nil {
				flat, err := json.Marshal(map[string]any{
					"error":   env.Error.Message,
					"code":    env.Error.Code,
					"details": env.Error.Details,
				})
				if err != nil {
					return
				}
				payload = flat
			} else {
				return
			}
		}
		var typed T
		if err := json.Unmarshal(payload, &typed); err != nil {
			return
		}
		handler(typed, msg)
	})
}
