// Package embeddednats links Brainkit's embedded NATS transport backend.
package embeddednats

import (
	"github.com/brainlet/brainkit"
	backend "github.com/brainlet/brainkit/internal/transport/backends/embeddednats"
	"github.com/brainlet/brainkit/transports/internal/transportcfg"
)

// New creates a zero-config in-process NATS server with JetStream.
func New(opts ...brainkit.TransportOption) brainkit.TransportConfig {
	return brainkit.EmbeddedNATS(opts...)
}

// WithNATSName sets the durable consumer prefix for NATS JetStream.
func WithNATSName(name string) brainkit.TransportOption {
	return brainkit.WithNATSName(name)
}

func init() {
	brainkit.RegisterTransportBuilder("embedded", func(ctx brainkit.TransportBuildContext) (any, error) {
		return backend.NewTransport(transportcfg.FromBuildContext(ctx))
	})
}
