package amqp

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/server/configfile"
	transport "github.com/brainlet/brainkit/transports/amqp"
)

func init() {
	configfile.RegisterTransport("amqp", func(y configfile.TransportYAML) (brainkit.TransportConfig, error) {
		return transport.New(y.URL), nil
	})
}
