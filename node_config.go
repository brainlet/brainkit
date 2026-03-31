package brainkit

import (
	"github.com/brainlet/brainkit/internal/discovery"
	"github.com/brainlet/brainkit/internal/messaging"
)

// MessagingConfig configures the transport-backed runtime host.
type MessagingConfig struct {
	Transport   string
	NATSURL     string
	NATSName    string
	AMQPURL     string
	RedisURL    string
	PostgresURL string
	SQLitePath  string
}

func (cfg MessagingConfig) transportConfig() messaging.TransportConfig {
	return messaging.TransportConfig{
		Type:        cfg.Transport,
		NATSURL:     cfg.NATSURL,
		NATSName:    cfg.NATSName,
		AMQPURL:     cfg.AMQPURL,
		RedisURL:    cfg.RedisURL,
		PostgresURL: cfg.PostgresURL,
		SQLitePath:  cfg.SQLitePath,
	}
}

// NodeConfig configures a transport-connected runtime node.
type NodeConfig struct {
	Kernel    KernelConfig
	Messaging MessagingConfig
	NodeID    string
	Namespace string
	Plugins   []PluginConfig
	Discovery discovery.Config // optional — peer discovery for cross-Kit
}
