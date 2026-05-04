// Package transports links all optional Brainkit network transport backends.
//
// Import backend-specific packages such as transports/embeddednats,
// transports/nats, transports/amqp, or transports/redis when a binary should
// link only one backend. This aggregate package intentionally links all of
// them for convenience.
package transports

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/transports/amqp"
	"github.com/brainlet/brainkit/transports/embeddednats"
	"github.com/brainlet/brainkit/transports/nats"
	"github.com/brainlet/brainkit/transports/redis"
)

// EmbeddedNATS creates a zero-config in-process NATS server with JetStream.
func EmbeddedNATS(opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return embeddednats.New(opts...)
}

// NATS connects to an external NATS server.
func NATS(url string, opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return nats.New(url, opts...)
}

// WithNATSName sets the durable consumer prefix for NATS JetStream.
func WithNATSName(name string) brainkit.TransportOption {
	return brainkit.WithNATSName(name)
}

// AMQP connects to a RabbitMQ server.
func AMQP(url string) brainkit.TransportConfig {
	return amqp.New(url)
}

// Redis connects to a Redis Streams server.
func Redis(url string) brainkit.TransportConfig {
	return redis.New(url)
}
