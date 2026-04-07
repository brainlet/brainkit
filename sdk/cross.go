package sdk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/google/uuid"
)

// PublishTo sends a typed command to a specific Kit's namespace.
// ReplyTo defaults to caller's namespace: <topic>.reply.<uuid>
func PublishTo[T messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error) {
	xrt, ok := rt.(CrossNamespaceRuntime)
	if !ok {
		return PublishResult{}, ErrNotCrossNamespace
	}

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

	ctx = transport.WithPublishMeta(ctx, correlationID, replyTo)
	msgID, err := xrt.PublishRawTo(ctx, targetNamespace, topic, payload)
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
