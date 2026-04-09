package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// RemoteClient performs namespaced publish/request operations over Watermill.
type RemoteClient struct {
	namespace      string
	callerID       string
	clusterID      string
	runtimeID      string
	pub            message.Publisher
	sub            message.Subscriber
	fanOutSub      message.Subscriber
	topicSanitizer func(string) string
}

func NewRemoteClient(namespace, callerID string, pub message.Publisher, sub message.Subscriber) *RemoteClient {
	return &RemoteClient{
		namespace: namespace,
		callerID:  callerID,
		pub:       pub,
		sub:       sub,
	}
}

// NewRemoteClientWithTransport creates a RemoteClient that uses the transport's topic sanitizer.
func NewRemoteClientWithTransport(namespace, callerID string, transport *Transport) *RemoteClient {
	return &RemoteClient{
		namespace:      namespace,
		callerID:       callerID,
		pub:            transport.Publisher,
		sub:            transport.Subscriber,
		fanOutSub:      transport.FanOutSubscriber,
		topicSanitizer: transport.TopicSanitizer,
	}
}

// SetIdentity configures cluster and runtime identity for message metadata.
func (c *RemoteClient) SetIdentity(clusterID, runtimeID string) {
	c.clusterID = clusterID
	c.runtimeID = runtimeID
}

// ResolvedTopic returns the wire-level topic for a logical topic (namespaced + sanitized).
func (c *RemoteClient) ResolvedTopic(logicalTopic string) string {
	return c.resolvedTopic(logicalTopic)
}

func (c *RemoteClient) resolvedTopic(logicalTopic string) string {
	topic := NamespacedTopic(c.namespace, logicalTopic)
	if c.topicSanitizer != nil {
		topic = c.topicSanitizer(topic)
	}
	return topic
}

func (c *RemoteClient) resolvedTopicForNamespace(targetNamespace, logicalTopic string) string {
	topic := NamespacedTopic(targetNamespace, logicalTopic)
	if c.topicSanitizer != nil {
		topic = c.topicSanitizer(topic)
	}
	return topic
}

// stampIdentity writes all identity metadata onto a Watermill message.
func (c *RemoteClient) stampIdentity(wmsg *message.Message) {
	if c.callerID != "" {
		wmsg.Metadata.Set("callerId", c.callerID)
	}
	if c.namespace != "" {
		wmsg.Metadata.Set("namespace", c.namespace)
	}
	if c.clusterID != "" {
		wmsg.Metadata.Set("clusterID", c.clusterID)
	}
	if c.runtimeID != "" {
		wmsg.Metadata.Set("runtimeID", c.runtimeID)
	}
}

// PublishRawToNamespace publishes to a specific namespace, bypassing the client's own namespace.
func (c *RemoteClient) PublishRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, payload json.RawMessage) (string, error) {
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.SetContext(ctx)
	c.stampIdentity(wmsg)
	correlationID := CorrelationIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	wmsg.Metadata.Set("correlationId", correlationID)

	// Stamp trace context for cross-service propagation (same as PublishRaw)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		wmsg.Metadata.Set("traceId", traceID)
	}
	if spanID := SpanIDFromContext(ctx); spanID != "" {
		wmsg.Metadata.Set("parentSpanId", spanID)
	}
	if sampled := SampledFromContext(ctx); sampled != "" {
		wmsg.Metadata.Set("traceSampled", sampled)
	}

	// Stamp replyTo — resolve with publisher's namespace so the handler publishes to the right topic
	if replyTo := ReplyToFromContext(ctx); replyTo != "" {
		resolvedReplyTo := c.resolvedTopic(replyTo)
		wmsg.Metadata.Set("replyTo", resolvedReplyTo)

		// Pre-declare the replyTo queue/exchange BEFORE publishing (same as PublishRaw).
		// Without this, AMQP fanout exchanges discard the response because no queue is bound.
		// This was the root cause of cross-Kit failures on AMQP/Redis/Postgres (bug #5).
		preCtx, preCancel := context.WithCancel(ctx)
		_, subErr := c.sub.Subscribe(preCtx, resolvedReplyTo)
		preCancel()
		_ = subErr
	}

	if err := c.pub.Publish(c.resolvedTopicForNamespace(targetNamespace, logicalTopic), wmsg); err != nil {
		return "", err
	}
	return correlationID, nil
}

// SubscribeRawToNamespace subscribes to a topic in a specific namespace.
func (c *RemoteClient) SubscribeRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, handler func(sdk.Message)) (func(), error) {
	subCtx, cancel := context.WithCancel(ctx)
	ch, err := c.sub.Subscribe(subCtx, c.resolvedTopicForNamespace(targetNamespace, logicalTopic))
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			select {
			case <-subCtx.Done():
				return
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				handler(sdk.Message{
					Topic:    logicalTopic,
					Payload:  append([]byte(nil), wmsg.Payload...),
					CallerID: wmsg.Metadata.Get("callerId"),
					Metadata: cloneMetadata(wmsg.Metadata),
				})
				wmsg.Ack()
			}
		}
	}()

	return cancel, nil
}

// PublishRaw sends a message to a namespaced topic.
// Always generates a correlationID (or reuses one from ctx) and returns it.
// The correlationID is stamped in the Watermill message metadata as "correlationId".
func (c *RemoteClient) PublishRaw(ctx context.Context, logicalTopic string, payload json.RawMessage) (string, error) {
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.SetContext(ctx)
	c.stampIdentity(wmsg)

	// Always generate or reuse correlationID
	correlationID := CorrelationIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	wmsg.Metadata.Set("correlationId", correlationID)

	// Stamp trace context for cross-service propagation
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		wmsg.Metadata.Set("traceId", traceID)
	}
	if spanID := SpanIDFromContext(ctx); spanID != "" {
		wmsg.Metadata.Set("parentSpanId", spanID)
	}
	if sampled := SampledFromContext(ctx); sampled != "" {
		wmsg.Metadata.Set("traceSampled", sampled)
	}

	// Stamp replyTo if present in context (set by sdk.Publish).
	// Namespace it so the handler can publish to the absolute topic.
	if replyTo := ReplyToFromContext(ctx); replyTo != "" {
		resolvedReplyTo := c.resolvedTopic(replyTo)
		wmsg.Metadata.Set("replyTo", resolvedReplyTo)

		// Pre-declare the replyTo queue/exchange BEFORE publishing, so the
		// handler's response isn't dropped. AMQP fanout exchanges discard
		// messages when no queue is bound. The subscribe creates a durable
		// exchange+queue, then we cancel immediately — the durable queue
		// retains messages until the caller's SubscribeTo consumes them.
		// For GoChannel/NATS/Redis/SQL this is harmless (they persist).
		preCtx, preCancel := context.WithCancel(ctx)
		_, subErr := c.sub.Subscribe(preCtx, resolvedReplyTo)
		preCancel() // Cancel consumer — durable queue persists
		_ = subErr
	}

	if err := c.pub.Publish(c.resolvedTopic(logicalTopic), wmsg); err != nil {
		return "", err
	}
	return correlationID, nil
}

// PublishRawWithMeta sends a message with extra metadata (e.g., retryCount for failure handling).
// Extra metadata keys are stamped directly — replyTo in extra is NOT namespace-resolved
// (used for retry re-publishes where replyTo is already resolved).
func (c *RemoteClient) PublishRawWithMeta(ctx context.Context, logicalTopic string, payload json.RawMessage, extra map[string]string) (string, error) {
	wmsg := message.NewMessage(watermill.NewUUID(), []byte(payload))
	wmsg.SetContext(ctx)
	c.stampIdentity(wmsg)

	correlationID := CorrelationIDFromContext(ctx)
	if correlationID == "" {
		correlationID = uuid.NewString()
	}
	wmsg.Metadata.Set("correlationId", correlationID)

	// Stamp extra metadata AFTER defaults — extras can override correlationId/replyTo
	// (used by retry to pass already-resolved values without re-namespacing)
	for k, v := range extra {
		wmsg.Metadata.Set(k, v)
	}

	if err := c.pub.Publish(c.resolvedTopic(logicalTopic), wmsg); err != nil {
		return "", err
	}
	return wmsg.Metadata.Get("correlationId"), nil
}

func (c *RemoteClient) AwaitRaw(ctx context.Context, logicalTopic, correlationID string) (sdk.Message, error) {
	resultCh, err := c.sub.Subscribe(ctx, c.resolvedTopic(logicalTopic))
	if err != nil {
		return sdk.Message{}, fmt.Errorf("subscribe %s: %w", logicalTopic, err)
	}

	for {
		select {
		case <-ctx.Done():
			return sdk.Message{}, ctx.Err()
		case wmsg, ok := <-resultCh:
			if !ok {
				return sdk.Message{}, fmt.Errorf("subscription closed for %s", logicalTopic)
			}

			if correlationID != "" && wmsg.Metadata.Get("correlationId") != correlationID {
				wmsg.Ack()
				continue
			}

			msg := sdk.Message{
				Topic:    logicalTopic,
				Payload:  append([]byte(nil), wmsg.Payload...),
				CallerID: wmsg.Metadata.Get("callerId"),
				Metadata: cloneMetadata(wmsg.Metadata),
			}
			wmsg.Ack()
			return msg, nil
		}
	}
}

// SubscribeRaw subscribes to a namespaced topic.
// Contract: the subscription is active and ready to receive messages before this method returns.
// This is guaranteed because sub.Subscribe() returns the channel synchronously; the consumer
// goroutine is started afterward. Combined with GoChannel's Persistent mode, messages are
// buffered even before the goroutine starts reading.
func (c *RemoteClient) SubscribeRaw(ctx context.Context, logicalTopic string, handler func(sdk.Message)) (func(), error) {
	subCtx, cancel := context.WithCancel(ctx)
	ch, err := c.sub.Subscribe(subCtx, c.resolvedTopic(logicalTopic))
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			select {
			case <-subCtx.Done():
				return
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				handler(sdk.Message{
					Topic:    logicalTopic,
					Payload:  append([]byte(nil), wmsg.Payload...),
					CallerID: wmsg.Metadata.Get("callerId"),
					Metadata: cloneMetadata(wmsg.Metadata),
				})
				wmsg.Ack()
			}
		}
	}()

	return cancel, nil
}

// SubscribeRawFanOut subscribes using the fan-out subscriber (all replicas receive).
// Used for events like deployment propagation where every replica needs the message.
func (c *RemoteClient) SubscribeRawFanOut(ctx context.Context, logicalTopic string, handler func(sdk.Message)) (func(), error) {
	if c.fanOutSub == nil {
		// Fallback to regular subscriber if no fan-out subscriber configured
		return c.SubscribeRaw(ctx, logicalTopic, handler)
	}
	subCtx, cancel := context.WithCancel(ctx)
	ch, err := c.fanOutSub.Subscribe(subCtx, c.resolvedTopic(logicalTopic))
	if err != nil {
		cancel()
		return nil, err
	}

	go func() {
		for {
			select {
			case <-subCtx.Done():
				return
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				handler(sdk.Message{
					Topic:    logicalTopic,
					Payload:  append([]byte(nil), wmsg.Payload...),
					CallerID: wmsg.Metadata.Get("callerId"),
					Metadata: cloneMetadata(wmsg.Metadata),
				})
				wmsg.Ack()
			}
		}
	}()

	return cancel, nil
}

func cloneMetadata(metadata message.Metadata) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
