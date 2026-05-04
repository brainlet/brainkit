// Package ctxkeys defines shared context key types for message metadata.
// Both sdk and internal/transport use these keys to pass correlationID
// and replyTo through context without creating an import cycle.
package ctxkeys

import "context"

type Key string

const (
	CallerID      Key = "brainkit.messaging.caller_id"
	CorrelationID Key = "brainkit.messaging.correlation_id"
	Metadata      Key = "brainkit.messaging.metadata"
	ReplyTo       Key = "brainkit.messaging.reply_to"
	Topic         Key = "brainkit.messaging.topic"
	TraceID       Key = "brainkit.messaging.trace_id"
	SpanID        Key = "brainkit.messaging.span_id"
	ParentSpanID  Key = "brainkit.messaging.parent_span_id"
	Sampled       Key = "brainkit.messaging.sampled"
	RuntimeID     Key = "brainkit.messaging.runtime_id"
)

// WithPublishMeta stamps correlationID and replyTo into context.
func WithPublishMeta(ctx context.Context, correlationID, replyTo string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if correlationID != "" {
		ctx = context.WithValue(ctx, CorrelationID, correlationID)
	}
	if replyTo != "" {
		ctx = context.WithValue(ctx, ReplyTo, replyTo)
	}
	return ctx
}

// WithCallerID stamps an explicit caller identity into publish context.
func WithCallerID(ctx context.Context, callerID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if callerID == "" {
		return ctx
	}
	return context.WithValue(ctx, CallerID, callerID)
}

// WithMetadata appends caller-provided metadata to a publish context.
func WithMetadata(ctx context.Context, metadata map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(metadata) == 0 {
		return ctx
	}
	merged := map[string]string{}
	if existing, ok := ctx.Value(Metadata).(map[string]string); ok {
		for key, value := range existing {
			merged[key] = value
		}
	}
	for key, value := range metadata {
		if key == "" {
			continue
		}
		merged[key] = value
	}
	return context.WithValue(ctx, Metadata, merged)
}

// MetadataFromContext returns a detached copy of publish metadata.
func MetadataFromContext(ctx context.Context) map[string]string {
	if ctx == nil {
		return nil
	}
	metadata, _ := ctx.Value(Metadata).(map[string]string)
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
