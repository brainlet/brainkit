package redis

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/server/configfile"
	transport "github.com/brainlet/brainkit/transports/redis"
)

func init() {
	configfile.RegisterTransport("redis", func(y configfile.TransportYAML) (brainkit.TransportConfig, error) {
		return transport.New(y.URL), nil
	})
}
