package transportcfg

import (
	"github.com/brainlet/brainkit"
	core "github.com/brainlet/brainkit/internal/transport"
)

// FromBuildContext projects public transport configuration onto the internal
// transport config used by backend constructors.
func FromBuildContext(ctx brainkit.TransportBuildContext) core.TransportConfig {
	return core.TransportConfig{
		Type:         ctx.Config.Kind(),
		Namespace:    ctx.Namespace,
		NATSURL:      ctx.Config.NATSURL(),
		NATSName:     ctx.Config.NATSName(),
		AMQPURL:      ctx.Config.AMQPURL(),
		RedisURL:     ctx.Config.RedisURL(),
		NATSStoreDir: ctx.NATSStoreDir,
	}
}
