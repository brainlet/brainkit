package nats

import (
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/server/configfile"
	transport "github.com/brainlet/brainkit/transports/nats"
)

func init() {
	configfile.RegisterTransport("nats", func(y configfile.TransportYAML) (brainkit.TransportConfig, error) {
		var opts []brainkit.TransportOption
		if y.NATSName != "" {
			opts = append(opts, transport.WithNATSName(y.NATSName))
		}
		return transport.New(y.URL, opts...), nil
	})
}
