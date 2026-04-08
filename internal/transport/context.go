package transport

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/internal/ctxkeys"
)

func withInboundMetadata(ctx context.Context, wmsg *message.Message, logicalTopic string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if callerID := wmsg.Metadata.Get("callerId"); callerID != "" {
		ctx = context.WithValue(ctx, ctxkeys.CallerID, callerID)
	}
	if correlationID := wmsg.Metadata.Get("correlationId"); correlationID != "" {
		ctx = context.WithValue(ctx, ctxkeys.CorrelationID, correlationID)
	}
	if replyTo := wmsg.Metadata.Get("replyTo"); replyTo != "" {
		ctx = context.WithValue(ctx, ctxkeys.ReplyTo, replyTo)
	}
	if logicalTopic != "" {
		ctx = context.WithValue(ctx, ctxkeys.Topic, logicalTopic)
	}
	if traceID := wmsg.Metadata.Get("traceId"); traceID != "" {
		ctx = context.WithValue(ctx, ctxkeys.TraceID, traceID)
	}
	if spanID := wmsg.Metadata.Get("parentSpanId"); spanID != "" {
		ctx = context.WithValue(ctx, ctxkeys.ParentSpanID, spanID)
	}
	if sampled := wmsg.Metadata.Get("traceSampled"); sampled != "" {
		ctx = context.WithValue(ctx, ctxkeys.Sampled, sampled)
	}
	return ctx
}

// WithPublishMeta stamps correlationID and replyTo into context for PublishRaw.
func WithPublishMeta(ctx context.Context, correlationID, replyTo string) context.Context {
	return ctxkeys.WithPublishMeta(ctx, correlationID, replyTo)
}

func ContextWithCorrelationID(ctx context.Context, correlationID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if correlationID == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxkeys.CorrelationID, correlationID)
}

func CallerIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.CallerID).(string)
	return v
}

func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.CorrelationID).(string)
	return v
}

func ReplyToFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.ReplyTo).(string)
	return v
}

func TopicFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.Topic).(string)
	return v
}

func WithTraceIDs(ctx context.Context, traceID, spanID, parentSpanID string) context.Context {
	if traceID != "" {
		ctx = context.WithValue(ctx, ctxkeys.TraceID, traceID)
	}
	if spanID != "" {
		ctx = context.WithValue(ctx, ctxkeys.SpanID, spanID)
	}
	if parentSpanID != "" {
		ctx = context.WithValue(ctx, ctxkeys.ParentSpanID, parentSpanID)
	}
	return ctx
}

func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.TraceID).(string)
	return v
}

func SpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.SpanID).(string)
	return v
}

func ParentSpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.ParentSpanID).(string)
	return v
}

func SampledFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxkeys.Sampled).(string)
	return v
}

func WithSampled(ctx context.Context, sampled string) context.Context {
	return context.WithValue(ctx, ctxkeys.Sampled, sampled)
}
