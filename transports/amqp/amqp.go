// Package amqp links Brainkit's AMQP/RabbitMQ transport backend.
package amqp

import (
	"github.com/brainlet/brainkit"
	backend "github.com/brainlet/brainkit/internal/transport/backends/amqp"
	"github.com/brainlet/brainkit/transports/internal/transportcfg"
)

// New connects to a RabbitMQ server.
func New(url string) brainkit.TransportConfig {
	return brainkit.AMQP(url)
}

func init() {
	brainkit.RegisterTransportBuilder("amqp", func(ctx brainkit.TransportBuildContext) (any, error) {
		return backend.NewTransport(transportcfg.FromBuildContext(ctx))
	})
}
