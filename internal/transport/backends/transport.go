package backends

import (
	"fmt"

	core "github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/transport/backends/amqp"
	"github.com/brainlet/brainkit/internal/transport/backends/backutil"
	"github.com/brainlet/brainkit/internal/transport/backends/embeddednats"
	"github.com/brainlet/brainkit/internal/transport/backends/nats"
	"github.com/brainlet/brainkit/internal/transport/backends/redis"
)

// EmbeddedNATS manages an in-process NATS server with JetStream.
type EmbeddedNATS = embeddednats.EmbeddedNATS

// EmbeddedNATSConfig configures the embedded NATS server.
type EmbeddedNATSConfig = embeddednats.EmbeddedNATSConfig

// NewEmbeddedNATS starts an in-process NATS server with JetStream enabled.
func NewEmbeddedNATS(cfg EmbeddedNATSConfig) (*EmbeddedNATS, error) {
	return embeddednats.NewEmbeddedNATS(cfg)
}

// NewTransportSet creates a fully managed transport bundle.
func NewTransportSet(cfg core.TransportConfig) (*core.Transport, error) {
	var (
		t    *core.Transport
		err  error
		kind string
	)
	switch cfg.Type {
	case "memory":
		t, err = core.NewTransportSet(cfg)
		kind = "memory"

	case "", "embedded":
		t, err = embeddednats.NewTransport(cfg)
		kind = "embedded"

	case "nats":
		t, err = nats.NewTransport(cfg)
		kind = "nats"

	case "amqp":
		t, err = amqp.NewTransport(cfg)
		kind = "amqp"

	case "redis":
		t, err = redis.NewTransport(cfg)
		kind = "redis"

	default:
		return nil, fmt.Errorf("unknown transport type: %q (supported: memory, embedded, nats, amqp, redis)", cfg.Type)
	}
	if err != nil {
		return nil, err
	}
	t.Kind = kind
	return t, nil
}

// NewTransport preserves the old pub/sub factory signature for tests and helpers.
func NewTransport(cfg core.TransportConfig) (core.Publisher, core.Subscriber, error) {
	transport, err := NewTransportSet(cfg)
	if err != nil {
		return nil, nil, err
	}
	return transport.Publisher, transport.Subscriber, nil
}

// NamespacedTopic derives the concrete subject from a logical topic.
func NamespacedTopic(namespace, logicalTopic string) string {
	return backutil.NamespacedTopic(namespace, logicalTopic)
}
