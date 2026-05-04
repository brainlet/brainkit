package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// TransportConfig configures the transport backend.
type TransportConfig struct {
	Type      string // "memory", "embedded", "nats", "amqp", "redis"
	Namespace string // consumer group name; default: "brainkit". Replicas with same namespace compete.

	// NATS (external or embedded)
	NATSURL      string
	NATSName     string
	NATSStoreDir string // JetStream store directory for embedded NATS. Empty = ephemeral.

	// AMQP (RabbitMQ)
	AMQPURL string // e.g. "amqp://guest:guest@localhost:5672/"

	// Redis Streams
	RedisURL string // e.g. "redis://localhost:6379/0"
}

// Transport bundles the concrete publisher/subscriber pair plus a shared closer.
type Transport struct {
	Publisher        Publisher
	Subscriber       Subscriber // consumer group = Namespace (competing consumers)
	FanOutSubscriber Subscriber // unique group per instance (all replicas receive)
	closeFns         []func() error

	// Kind is the normalized transport type ("memory", "embedded", "nats",
	// "amqp", "redis"). Modules use this to reject configurations the
	// transport can't support (e.g. plugins need real networking).
	Kind string

	// TopicSanitizer transforms logical topic names into transport-safe names.
	// Applied automatically by RemoteClient and Host.
	TopicSanitizer func(string) string
}

// NewManagedTransport builds a managed transport from backend-owned pieces.
// Backend factories live outside this core package so importing brainkit does
// not also import every network backend.
func NewManagedTransport(kind string, pub Publisher, sub, fanOut Subscriber, sanitizer func(string) string, closeFns ...func() error) *Transport {
	return &Transport{
		Publisher:        pub,
		Subscriber:       sub,
		FanOutSubscriber: fanOut,
		closeFns:         closeFns,
		Kind:             kind,
		TopicSanitizer:   sanitizer,
	}
}

// AppendClose adds backend-owned cleanup functions to the transport.
func (t *Transport) AppendClose(closeFns ...func() error) {
	if t == nil {
		return
	}
	t.closeFns = append(t.closeFns, closeFns...)
}

// SanitizeTopic applies the transport's topic sanitizer if set.
func (t *Transport) SanitizeTopic(topic string) string {
	if t == nil || t.TopicSanitizer == nil {
		return topic
	}
	return t.TopicSanitizer(topic)
}

// OnceCloser wraps a Close function with sync.Once to prevent double-close
// panics when a router and transport close the same backend resource through
// different lifecycle paths.
func OnceCloser(fn func() error) func() error {
	var once sync.Once
	return func() error {
		var err error
		once.Do(func() { err = fn() })
		return err
	}
}

// PublishResolved publishes to an already-resolved transport topic. It is used
// for low-level reply paths that intentionally bypass logical namespace
// resolution.
func (t *Transport) PublishResolved(ctx context.Context, topic, correlationID string, payload json.RawMessage, done bool, envelope bool) error {
	wmsg := NewMessage([]byte(payload))
	wmsg.SetContext(ctx)
	if correlationID != "" {
		wmsg.Metadata.Set("correlationId", correlationID)
	}
	if done {
		wmsg.Metadata.Set("done", "true")
	}
	if envelope {
		wmsg.Metadata.Set("envelope", "true")
	}
	return t.Publisher.Publish(topic, wmsg)
}

// Close shuts down all transport resources.
func (t *Transport) Close() error {
	if t == nil {
		return nil
	}
	var firstErr error
	for _, closeFn := range t.closeFns {
		if closeFn == nil {
			continue
		}
		if err := closeFn(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// NewTransportSet creates the core in-process transport. Network backends are
// built by github.com/brainlet/brainkit/transports so the root package stays
// light unless callers opt into those backends.
func NewTransportSet(cfg TransportConfig) (*Transport, error) {
	switch cfg.Type {
	case "", "memory":
		broker := newMemoryBroker()
		return NewManagedTransport("memory", broker, broker, broker, nil, broker.Close), nil
	default:
		return nil, fmt.Errorf("transport %q is not linked; import github.com/brainlet/brainkit/transports or use a backend builder", cfg.Type)
	}
}

// NewTransport preserves the old pub/sub factory signature for tests and helpers.
func NewTransport(cfg TransportConfig) (Publisher, Subscriber, error) {
	transport, err := NewTransportSet(cfg)
	if err != nil {
		return nil, nil, err
	}
	return transport.Publisher, transport.Subscriber, nil
}

// NamespacedTopic derives the concrete subject from a logical topic.
func NamespacedTopic(namespace, logicalTopic string) string {
	namespace = strings.TrimSpace(namespace)
	logicalTopic = strings.TrimSpace(logicalTopic)
	if namespace == "" {
		return logicalTopic
	}
	if logicalTopic == "" {
		return namespace
	}
	return namespace + "." + logicalTopic
}

func sanitizeDurable(value string) string {
	replacer := strings.NewReplacer(".", "_", "/", "_", "@", "_", "-", "_", " ", "_", ":", "_")
	return replacer.Replace(value)
}
