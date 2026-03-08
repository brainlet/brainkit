// Ported from: packages/core/src/events/pubsub.ts
package events

// AckFunc is an optional acknowledgment callback provided to subscribers.
type AckFunc func() error

// SubscribeCallback is the function signature for event subscribers.
type SubscribeCallback func(event Event, ack AckFunc)

// PubSub defines the interface for publish/subscribe implementations.
type PubSub interface {
	// Publish sends an event to all subscribers of the given topic.
	// The implementation is responsible for assigning ID and CreatedAt.
	Publish(topic string, event PublishEvent) error

	// Subscribe registers a callback for events on the given topic.
	Subscribe(topic string, cb SubscribeCallback) error

	// Unsubscribe removes a previously registered callback for the given topic.
	Unsubscribe(topic string, cb SubscribeCallback) error

	// Flush ensures all pending events have been delivered.
	Flush() error
}
