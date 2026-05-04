package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/brainlet/brainkit/sdk"
	"github.com/google/uuid"
)

// RemoteClient performs namespaced publish/request operations over the bus.
type RemoteClient struct {
	namespace      string
	callerID       string
	clusterID      string
	runtimeID      string
	pub            Publisher
	sub            Subscriber
	fanOutSub      Subscriber
	topicSanitizer func(string) string
	metrics        *Metrics
	activeSubs     atomic.Int64
}

// SubscriptionHandle owns a raw subscription goroutine. Stop only requests
// cancellation; CloseContext waits until the subscription handler loop exits.
type SubscriptionHandle struct {
	cancel context.CancelFunc
	done   chan struct{}

	stopOnce   sync.Once
	finishOnce sync.Once
}

func newSubscriptionHandle(cancel context.CancelFunc) *SubscriptionHandle {
	return &SubscriptionHandle{
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

// Stop requests subscription cancellation without waiting for the handler
// goroutine to exit. This preserves the legacy SubscribeRaw cancel contract.
func (h *SubscriptionHandle) Stop() {
	if h == nil {
		return
	}
	h.stopOnce.Do(func() {
		if h.cancel != nil {
			h.cancel()
		}
	})
}

// Done closes once the subscription goroutine has exited.
func (h *SubscriptionHandle) Done() <-chan struct{} {
	if h == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return h.done
}

// CloseContext requests cancellation and waits for the subscription goroutine
// to exit or ctx to expire.
func (h *SubscriptionHandle) CloseContext(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	h.Stop()
	select {
	case <-h.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (h *SubscriptionHandle) finish() {
	if h == nil {
		return
	}
	h.finishOnce.Do(func() {
		close(h.done)
	})
}

func NewRemoteClient(namespace, callerID string, pub Publisher, sub Subscriber) *RemoteClient {
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

// SetMetrics attaches optional bus metrics to logical publish paths.
func (c *RemoteClient) SetMetrics(metrics *Metrics) {
	if c == nil {
		return
	}
	c.metrics = metrics
}

// ActiveSubscriptions returns the number of live raw subscription leases.
func (c *RemoteClient) ActiveSubscriptions() int64 {
	if c == nil {
		return 0
	}
	return c.activeSubs.Load()
}

func (c *RemoteClient) recordPublished(logicalTopic string) {
	if c == nil || c.metrics == nil {
		return
	}
	c.metrics.Published(logicalTopic)
}

func (c *RemoteClient) trackSubscription(cancel context.CancelFunc) (*SubscriptionHandle, func()) {
	c.activeSubs.Add(1)
	handle := newSubscriptionHandle(cancel)
	var finishOnce sync.Once
	finish := func() {
		finishOnce.Do(func() {
			c.activeSubs.Add(-1)
			handle.finish()
		})
	}
	return handle, finish
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

// globalTopic applies topic sanitization WITHOUT namespace prefixing.
// Used for cluster-wide system topics like presence announcements.
func (c *RemoteClient) globalTopic(topic string) string {
	if c.topicSanitizer != nil {
		return c.topicSanitizer(topic)
	}
	return topic
}

// stampIdentity writes all identity metadata onto a transport message.
func (c *RemoteClient) stampIdentity(wmsg *Message) {
	callerID := c.callerID
	if ctxCallerID := CallerIDFromContext(wmsg.Context()); ctxCallerID != "" {
		callerID = ctxCallerID
	}
	if callerID != "" {
		wmsg.Metadata.Set("callerId", callerID)
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
	for key, value := range MetadataFromContext(wmsg.Context()) {
		if isReservedPublishMetadata(key) {
			continue
		}
		wmsg.Metadata.Set(key, value)
	}
}

func isReservedPublishMetadata(key string) bool {
	switch key {
	case "callerId", "correlationId", "replyTo", "namespace", "clusterID", "runtimeID":
		return true
	default:
		return false
	}
}

// PublishRawToNamespace publishes to a specific namespace, bypassing the client's own namespace.
func (c *RemoteClient) PublishRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, payload json.RawMessage) (string, error) {
	wmsg := NewMessage([]byte(payload))
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
	c.recordPublished(logicalTopic)
	return correlationID, nil
}

// SubscribeRawToNamespace subscribes to a topic in a specific namespace.
func (c *RemoteClient) SubscribeRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, handler func(sdk.Message)) (func(), error) {
	handle, err := c.subscribeRawMessages(ctx, c.sub, c.resolvedTopicForNamespace(targetNamespace, logicalTopic), logicalTopic, handler)
	if err != nil {
		return nil, err
	}
	return handle.Stop, nil
}

// PublishRaw sends a message to a namespaced topic.
// Always generates a correlationID (or reuses one from ctx) and returns it.
// The correlationID is stamped in message metadata as "correlationId".
func (c *RemoteClient) PublishRaw(ctx context.Context, logicalTopic string, payload json.RawMessage) (string, error) {
	wmsg := NewMessage([]byte(payload))
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

	// Stamp replyTo if present in context (set by sdk/protocol.Publish).
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
	c.recordPublished(logicalTopic)
	return correlationID, nil
}

// PublishRawWithMeta sends a message with extra metadata (e.g., retryCount for failure handling).
// Extra metadata keys are stamped directly — replyTo in extra is NOT namespace-resolved
// (used for retry re-publishes where replyTo is already resolved).
func (c *RemoteClient) PublishRawWithMeta(ctx context.Context, logicalTopic string, payload json.RawMessage, extra map[string]string) (string, error) {
	wmsg := NewMessage([]byte(payload))
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
	c.recordPublished(logicalTopic)
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
	handle, err := c.SubscribeRawHandle(ctx, logicalTopic, handler)
	if err != nil {
		return nil, err
	}
	return handle.Stop, nil
}

// SubscribeRawHandle subscribes to a namespaced topic and returns a
// context-aware close handle for module-owned subscriptions.
func (c *RemoteClient) SubscribeRawHandle(ctx context.Context, logicalTopic string, handler func(sdk.Message)) (*SubscriptionHandle, error) {
	return c.subscribeRawMessages(ctx, c.sub, c.resolvedTopic(logicalTopic), logicalTopic, handler)
}

// SubscribeRawFanOut subscribes using the fan-out subscriber (all replicas receive).
// Used for events like deployment propagation where every replica needs the message.
func (c *RemoteClient) SubscribeRawFanOut(ctx context.Context, logicalTopic string, handler func(sdk.Message)) (func(), error) {
	handle, err := c.SubscribeRawFanOutHandle(ctx, logicalTopic, handler)
	if err != nil {
		return nil, err
	}
	return handle.Stop, nil
}

// SubscribeRawFanOutHandle is SubscribeRawFanOut with context-aware close.
func (c *RemoteClient) SubscribeRawFanOutHandle(ctx context.Context, logicalTopic string, handler func(sdk.Message)) (*SubscriptionHandle, error) {
	if c.fanOutSub == nil {
		// Fallback to regular subscriber if no fan-out subscriber configured
		return c.SubscribeRawHandle(ctx, logicalTopic, handler)
	}
	return c.subscribeRawMessages(ctx, c.fanOutSub, c.resolvedTopic(logicalTopic), logicalTopic, handler)
}

func (c *RemoteClient) subscribeRawMessages(ctx context.Context, sub Subscriber, wireTopic, logicalTopic string, handler func(sdk.Message)) (*SubscriptionHandle, error) {
	subCtx, cancel := context.WithCancel(ctx)
	ch, err := sub.Subscribe(subCtx, wireTopic)
	if err != nil {
		cancel()
		return nil, err
	}
	handle, finish := c.trackSubscription(cancel)

	go func() {
		defer finish()
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

	return handle, nil
}

// PublishRawGlobal sends a message to a global (non-namespaced) topic.
// Used for cluster-wide system messages (presence, etc.) where all namespaces
// must see the message. Topic sanitization still applies (dots→dashes for NATS).
func (c *RemoteClient) PublishRawGlobal(ctx context.Context, topic string, payload json.RawMessage) error {
	wmsg := NewMessage([]byte(payload))
	wmsg.SetContext(ctx)
	c.stampIdentity(wmsg)
	if err := c.pub.Publish(c.globalTopic(topic), wmsg); err != nil {
		return err
	}
	c.recordPublished(topic)
	return nil
}

// SubscribeRawFanOutGlobal subscribes to a global (non-namespaced) topic
// using the fan-out subscriber. Every instance receives every message.
func (c *RemoteClient) SubscribeRawFanOutGlobal(ctx context.Context, topic string, handler func(payload json.RawMessage)) (func(), error) {
	handle, err := c.SubscribeRawFanOutGlobalHandle(ctx, topic, handler)
	if err != nil {
		return nil, err
	}
	return handle.Stop, nil
}

// SubscribeRawFanOutGlobalHandle is SubscribeRawFanOutGlobal with context-aware close.
func (c *RemoteClient) SubscribeRawFanOutGlobalHandle(ctx context.Context, topic string, handler func(payload json.RawMessage)) (*SubscriptionHandle, error) {
	sub := c.fanOutSub
	if sub == nil {
		sub = c.sub // fallback to regular subscriber
	}
	subCtx, cancel := context.WithCancel(ctx)
	ch, err := sub.Subscribe(subCtx, c.globalTopic(topic))
	if err != nil {
		cancel()
		return nil, err
	}
	handle, finish := c.trackSubscription(cancel)
	go func() {
		defer finish()
		for {
			select {
			case <-subCtx.Done():
				return
			case wmsg, ok := <-ch:
				if !ok {
					return
				}
				handler(json.RawMessage(wmsg.Payload))
				wmsg.Ack()
			}
		}
	}()
	return handle, nil
}

func cloneMetadata(metadata Metadata) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	out := make(map[string]string, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
