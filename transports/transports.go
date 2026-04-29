// Package transports links Brainkit's optional network transport backends.
//
// Import this package when a Kit uses EmbeddedNATS, NATS, AMQP, or Redis. The
// root brainkit package keeps only the in-process memory transport linked by
// default.
package transports

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/transport/backends"
)

// EmbeddedNATS creates a zero-config in-process NATS server with JetStream.
func EmbeddedNATS(opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return brainkit.EmbeddedNATS(opts...)
}

// NATS connects to an external NATS server.
func NATS(url string, opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return brainkit.NATS(url, opts...)
}

// WithNATSName sets the durable consumer prefix for NATS JetStream.
func WithNATSName(name string) brainkit.TransportOption {
	return brainkit.WithNATSName(name)
}

// AMQP connects to a RabbitMQ server.
func AMQP(url string) brainkit.TransportConfig {
	return brainkit.AMQP(url)
}

// Redis connects to a Redis Streams server.
func Redis(url string) brainkit.TransportConfig {
	return brainkit.Redis(url)
}

func init() {
	register("embedded")
	register("nats")
	register("amqp")
	register("redis")
}

func register(kind string) {
	brainkit.RegisterTransportBuilder(kind, func(ctx brainkit.TransportBuildContext) (any, error) {
		cfg := transport.TransportConfig{
			Type:         ctx.Config.Kind(),
			Namespace:    ctx.Namespace,
			NATSURL:      ctx.Config.NATSURL(),
			NATSName:     ctx.Config.NATSName(),
			AMQPURL:      ctx.Config.AMQPURL(),
			RedisURL:     ctx.Config.RedisURL(),
			NATSStoreDir: ctx.NATSStoreDir,
		}
		return backends.NewTransportSet(cfg)
	})
}
