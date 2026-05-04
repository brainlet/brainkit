// Package redis links Brainkit's Redis Streams transport backend.
package redis

import (
	"github.com/brainlet/brainkit"
	backend "github.com/brainlet/brainkit/internal/transport/backends/redis"
	"github.com/brainlet/brainkit/transports/internal/transportcfg"
)

// New connects to a Redis Streams server.
func New(url string) brainkit.TransportConfig {
	return brainkit.Redis(url)
}

func init() {
	brainkit.RegisterTransportBuilder("redis", func(ctx brainkit.TransportBuildContext) (any, error) {
		return backend.NewTransport(transportcfg.FromBuildContext(ctx))
	})
}
