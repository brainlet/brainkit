// Ported from: packages/core/src/events/event-emitter.ts
package events

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// subID is a monotonically increasing counter for subscription identity.
var subIDCounter atomic.Uint64

// EventEmitterPubSub is an in-process PubSub implementation backed by
// callback maps protected with a sync.RWMutex. It is the Go equivalent
// of the TypeScript EventEmitterPubSub which wraps Node's EventEmitter.
type EventEmitterPubSub struct {
	mu          sync.RWMutex
	subscribers map[string][]subscriberEntry
}

// subscriberEntry pairs a callback with a unique id so that Unsubscribe
// can locate and remove entries (Go func values are not comparable).
type subscriberEntry struct {
	id uint64
	cb SubscribeCallback
}

// NewEventEmitterPubSub creates a new EventEmitterPubSub.
func NewEventEmitterPubSub() *EventEmitterPubSub {
	return &EventEmitterPubSub{
		subscribers: make(map[string][]subscriberEntry),
	}
}

// Publish sends an event to all subscribers of the given topic.
// It assigns a new UUID as the event ID and the current time as CreatedAt.
func (e *EventEmitterPubSub) Publish(topic string, pe PublishEvent) error {
	evt := Event{
		Type:      pe.Type,
		ID:        uuid.New().String(),
		Data:      pe.Data,
		RunID:     pe.RunID,
		CreatedAt: time.Now(),
	}

	e.mu.RLock()
	// Copy the slice under the read lock so callbacks run without holding it.
	subs := make([]subscriberEntry, len(e.subscribers[topic]))
	copy(subs, e.subscribers[topic])
	e.mu.RUnlock()

	for _, s := range subs {
		s.cb(evt, nil)
	}
	return nil
}

// Subscribe registers a callback for events on the given topic and returns
// its subscription ID. The ID can be passed to UnsubscribeByID for removal.
func (e *EventEmitterPubSub) Subscribe(topic string, cb SubscribeCallback) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers[topic] = append(e.subscribers[topic], subscriberEntry{
		id: subIDCounter.Add(1),
		cb: cb,
	})
	return nil
}

// SubscribeWithID is like Subscribe but also returns the subscription ID
// so the caller can later call UnsubscribeByID to remove this specific
// subscription. This is the recommended way to manage subscriptions in Go
// since function values are not comparable.
func (e *EventEmitterPubSub) SubscribeWithID(topic string, cb SubscribeCallback) (uint64, error) {
	id := subIDCounter.Add(1)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers[topic] = append(e.subscribers[topic], subscriberEntry{
		id: id,
		cb: cb,
	})
	return id, nil
}

// Unsubscribe removes the first subscriber for the given topic.
// Because Go function values are not comparable, this removes the oldest
// subscription on that topic. For precise removal, use SubscribeWithID +
// UnsubscribeByID instead. This method exists to satisfy the PubSub interface.
func (e *EventEmitterPubSub) Unsubscribe(topic string, _ SubscribeCallback) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	subs := e.subscribers[topic]
	if len(subs) > 0 {
		e.subscribers[topic] = subs[1:]
	}
	return nil
}

// UnsubscribeByID removes the subscriber with the given ID from the topic.
func (e *EventEmitterPubSub) UnsubscribeByID(topic string, id uint64) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	subs := e.subscribers[topic]
	for i, s := range subs {
		if s.id == id {
			e.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			return nil
		}
	}
	return nil
}

// Flush is a no-op for the in-process emitter.
func (e *EventEmitterPubSub) Flush() error {
	return nil
}

// Close removes all subscribers, cleaning up resources for graceful shutdown.
func (e *EventEmitterPubSub) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.subscribers = make(map[string][]subscriberEntry)
	return nil
}
