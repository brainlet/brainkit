// Package ctxkeys defines shared context key types for message metadata.
// Both sdk and internal/transport use these keys to pass correlationID
// and replyTo through context without creating an import cycle.
package ctxkeys

import "context"

type Key string

const (
	CallerID      Key = "brainkit.messaging.caller_id"
	CorrelationID Key = "brainkit.messaging.correlation_id"
	ReplyTo       Key = "brainkit.messaging.reply_to"
	Topic         Key = "brainkit.messaging.topic"
	TraceID       Key = "brainkit.messaging.trace_id"
	SpanID        Key = "brainkit.messaging.span_id"
	ParentSpanID  Key = "brainkit.messaging.parent_span_id"
	Sampled       Key = "brainkit.messaging.sampled"
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
