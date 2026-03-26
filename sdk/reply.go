package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/sdk/messages"
)

// Replier is an optional interface for direct bus responses.
// Runtimes that support bus messaging implement this.
// The replyTo topic is already fully resolved — no namespace prefixing.
type Replier interface {
	ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error
}

// Reply sends a final response to a bus message.
// Equivalent to JS msg.reply(data). Sets done=true in metadata.
func Reply(rt Runtime, ctx context.Context, msg messages.Message, payload any) error {
	return replyInternal(rt, ctx, msg, payload, true)
}

// SendChunk sends an intermediate chunk to a bus message.
// Equivalent to JS msg.send(data). Sets done=false in metadata.
// Use for streaming patterns where multiple responses precede a final Reply.
func SendChunk(rt Runtime, ctx context.Context, msg messages.Message, payload any) error {
	return replyInternal(rt, ctx, msg, payload, false)
}

func replyInternal(rt Runtime, ctx context.Context, msg messages.Message, payload any, done bool) error {
	replier, ok := rt.(Replier)
	if !ok {
		return ErrNotReplier
	}
	replyTo := msg.Metadata["replyTo"]
	if replyTo == "" {
		return ErrNoReplyTo
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sdk: marshal reply payload: %w", err)
	}
	correlationID := msg.Metadata["correlationId"]
	return replier.ReplyRaw(ctx, replyTo, correlationID, data, done)
}
