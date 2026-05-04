// Package nats links Brainkit's external NATS JetStream transport backend.
package nats

import (
	"github.com/brainlet/brainkit"
	backend "github.com/brainlet/brainkit/internal/transport/backends/nats"
	"github.com/brainlet/brainkit/transports/internal/transportcfg"
)

// New connects to an external NATS server.
func New(url string, opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return brainkit.NATS(url, opts...)
}

// WithNATSName sets the durable consumer prefix for NATS JetStream.
func WithNATSName(name string) brainkit.TransportOption {
	return brainkit.WithNATSName(name)
}

func init() {
	brainkit.RegisterTransportBuilder("nats", func(ctx brainkit.TransportBuildContext) (any, error) {
		return backend.NewTransport(transportcfg.FromBuildContext(ctx))
	})
}
