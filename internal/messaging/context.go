package messaging

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type contextKey string

const (
	callerIDContextKey      contextKey = "brainkit.messaging.caller_id"
	correlationIDContextKey contextKey = "brainkit.messaging.correlation_id"
	topicContextKey         contextKey = "brainkit.messaging.topic"
)

func withInboundMetadata(ctx context.Context, wmsg *message.Message, logicalTopic string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if callerID := wmsg.Metadata.Get("callerId"); callerID != "" {
		ctx = context.WithValue(ctx, callerIDContextKey, callerID)
	}
	if correlationID := wmsg.Metadata.Get("correlationId"); correlationID != "" {
		ctx = context.WithValue(ctx, correlationIDContextKey, correlationID)
	}
	if logicalTopic != "" {
		ctx = context.WithValue(ctx, topicContextKey, logicalTopic)
	}
	return ctx
}

func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if correlationID == "" {
		return ctx
	}
	return context.WithValue(ctx, correlationIDContextKey, correlationID)
}

// CallerIDFromContext returns the inbound Watermill caller identity, if any.
func CallerIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	callerID, _ := ctx.Value(callerIDContextKey).(string)
	return callerID
}

// CorrelationIDFromContext returns the inbound request correlation id, if any.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	correlationID, _ := ctx.Value(correlationIDContextKey).(string)
	return correlationID
}

// TopicFromContext returns the logical topic currently being handled.
func TopicFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	topic, _ := ctx.Value(topicContextKey).(string)
	return topic
}
