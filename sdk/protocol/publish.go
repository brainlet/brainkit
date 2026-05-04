// Package protocol contains low-level bus protocol helpers for diagnostics,
// transport tests, and bridge code that intentionally owns reply topics.
//
// Normal request/reply should use sdk.Call, sdk.CallStream, or generated
// module-owned CallXxx helpers.
package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/ctxkeys"
	"github.com/google/uuid"
)

// PublishResult contains metadata about a low-level protocol publish.
type PublishResult struct {
	MessageID     string // transport message ID
	CorrelationID string // for response filtering
	ReplyTo       string // where responses will be sent
	Topic         string // where the message was published
}

type publishConfig struct {
	replyTo string
}

// PublishOption configures a low-level protocol publish call.
type PublishOption func(*publishConfig)

// WithReplyTo overrides the auto-generated reply topic. It is for protocol
// bridges, tests, and transport-level diagnostics; normal request/reply
// callers should use sdk.Call or generated CallXxx helpers.
func WithReplyTo(topic string) PublishOption {
	return func(c *publishConfig) { c.replyTo = topic }
}

// Publish sends a typed message with reply metadata and returns the raw reply
// topic. It is the low-level command publish primitive for protocol bridges,
// tests, and diagnostics. Normal request/reply callers should use sdk.Call or
// generated CallXxx helpers so replies are routed through the shared caller
// inbox. Default reply topic convention: <topic>.reply.<uuid>.
func Publish[T sdk.BrainkitMessage](rt sdk.Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error) {
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

// PublishTo sends a low-level typed command to a specific Kit namespace.
// Normal cross-namespace request/reply callers should use sdk.Call with
// sdk.WithCallTo. ReplyTo defaults to the caller's namespace:
// <topic>.reply.<uuid>.
func PublishTo[T sdk.BrainkitMessage](rt sdk.Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error) {
	xrt, ok := rt.(sdk.CrossNamespaceRuntime)
	if !ok {
		return PublishResult{}, sdk.ErrNotCrossNamespace
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

	ctx = ctxkeys.WithPublishMeta(ctx, correlationID, replyTo)
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

// SendToService publishes at the low-level to a deployed .ts service's mailbox
// topic. Prefer sdk.Call with sdk.CustomMsg, generated helpers, or JS
// bus.callService for normal request/reply.
func SendToService(rt sdk.Runtime, ctx context.Context, service, topic string, payload any, opts ...PublishOption) (PublishResult, error) {
	resolved := ResolveServiceTopic(service, topic)
	data, err := json.Marshal(payload)
	if err != nil {
		return PublishResult{}, fmt.Errorf("sdk/protocol: marshal payload: %w", err)
	}
	return Publish(rt, ctx, sdk.CustomMsg{Topic: resolved, Payload: data}, opts...)
}

// ResolveServiceTopic converts a service name + local topic to the bus topic.
// Convention: ts.<name-without-ext>.<topic>
//
//	"my-agent.ts" + "ask" -> "ts.my-agent.ask"
//	"my-agent"    + "ask" -> "ts.my-agent.ask"
//	"nested/svc"  + "rpc" -> "ts.nested.svc.rpc"
func ResolveServiceTopic(service, topic string) string {
	name := strings.TrimSuffix(service, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
