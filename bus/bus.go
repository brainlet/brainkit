package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Handler processes a message and optionally returns a reply.
type Handler func(ctx context.Context, msg Message) (*Message, error)

// SubscriptionID identifies a subscription.
type SubscriptionID string

type subscription struct {
	id      SubscriptionID
	pattern string
	handler func(Message)
}

// Bus handles topic routing, pub/sub, and the interceptor pipeline.
type Bus struct {
	mu           sync.RWMutex
	handlers     map[string]Handler
	subs         map[SubscriptionID]*subscription
	interceptors []interceptorEntry
	closed       bool
}

// New creates a new Bus.
func New() *Bus {
	return &Bus{
		handlers: make(map[string]Handler),
		subs:     make(map[SubscriptionID]*subscription),
	}
}

// Close shuts down the bus.
func (b *Bus) Close() {
	b.mu.Lock()
	b.closed = true
	b.mu.Unlock()
}

// Handle registers a handler for a topic prefix.
func (b *Bus) Handle(topicPrefix string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topicPrefix] = h
}

// AddInterceptor registers an interceptor sorted by priority.
func (b *Bus) AddInterceptor(i Interceptor) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.interceptors = append(b.interceptors, interceptorEntry{
		interceptor: i,
		priority:    i.Priority(),
	})
	sort.Slice(b.interceptors, func(a, c int) bool {
		return b.interceptors[a].priority < b.interceptors[c].priority
	})
}

// Subscribe registers a callback for messages matching a topic pattern.
// Pattern "foo.*" matches "foo.bar", "foo.baz.qux", etc.
func (b *Bus) Subscribe(pattern string, handler func(Message)) (SubscriptionID, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	id := SubscriptionID(uuid.NewString())
	b.subs[id] = &subscription{id: id, pattern: pattern, handler: handler}
	return id, nil
}

// Unsubscribe removes a subscription.
func (b *Bus) Unsubscribe(id SubscriptionID) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subs, id)
}

// Send publishes a message. Runs interceptors, then delivers to subscribers and handlers.
func (b *Bus) Send(ctx context.Context, msg Message) error {
	if msg.Depth >= MaxDepth {
		return fmt.Errorf("bus: cycle detected (depth %d >= max %d)", msg.Depth, MaxDepth)
	}
	if msg.ID == "" {
		msg.ID = uuid.NewString()
	}
	if msg.TraceID == "" {
		msg.TraceID = msg.ID
	}

	b.mu.RLock()
	interceptors := make([]interceptorEntry, len(b.interceptors))
	copy(interceptors, b.interceptors)
	b.mu.RUnlock()

	if err := runInterceptors(ctx, interceptors, &msg); err != nil {
		return fmt.Errorf("bus: interceptor rejected: %w", err)
	}

	// Deliver to matching subscribers
	b.mu.RLock()
	var matchedSubs []func(Message)
	for _, sub := range b.subs {
		if TopicMatches(sub.pattern, msg.Topic) {
			matchedSubs = append(matchedSubs, sub.handler)
		}
	}
	b.mu.RUnlock()

	for _, handler := range matchedSubs {
		handler(msg) // synchronous delivery for deterministic tests
	}

	// Deliver to matching handlers (for Request pattern — only when ReplyTo is set)
	if msg.ReplyTo != "" {
		b.mu.RLock()
		var matchedHandler Handler
		for prefix, handler := range b.handlers {
			if TopicMatches(prefix, msg.Topic) {
				matchedHandler = handler
				break
			}
		}
		b.mu.RUnlock()

		if matchedHandler != nil {
			go func() {
				reply, err := matchedHandler(ctx, msg)
				if err != nil {
					errPayload, _ := json.Marshal(map[string]string{"error": err.Error()})
					b.Send(ctx, Message{
						Topic:    msg.ReplyTo,
						CallerID: "bus",
						Payload:  errPayload,
						TraceID:  msg.TraceID,
						ParentID: msg.ID,
						Depth:    msg.Depth + 1,
					})
					return
				}
				if reply != nil {
					reply.TraceID = msg.TraceID
					reply.ParentID = msg.ID
					reply.Depth = msg.Depth + 1
					if reply.Topic == "" {
						reply.Topic = msg.ReplyTo
					}
					b.Send(ctx, *reply)
				}
			}()
		}
	}

	return nil
}

// Request sends a message and waits for a response.
func (b *Bus) Request(ctx context.Context, topic, callerID string, payload json.RawMessage) (*Message, error) {
	replyTo := "reply." + uuid.NewString()
	replyCh := make(chan Message, 1)

	sub, _ := b.Subscribe(replyTo, func(msg Message) {
		replyCh <- msg
	})
	defer b.Unsubscribe(sub)

	err := b.Send(ctx, Message{
		Topic:    topic,
		CallerID: callerID,
		Payload:  payload,
		ReplyTo:  replyTo,
	})
	if err != nil {
		return nil, err
	}

	select {
	case reply := <-replyCh:
		return &reply, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// TopicMatches checks if a topic matches a pattern.
// "test.*" matches "test.foo", "test.foo.bar".
// "test.foo" matches only "test.foo".
func TopicMatches(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}
	return false
}
