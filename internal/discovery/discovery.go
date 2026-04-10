package discovery

import (
	"context"
	"encoding/json"
)

// Provider resolves peer addresses for Kit-to-Kit networking.
type Provider interface {
	Resolve(name string) (namespace string, err error)
	Browse() ([]Peer, error)
	BrowseNamespaces() ([]string, error)
	Register(self Peer) error
	Close() error
}

// Peer represents a discoverable Kit instance.
type Peer struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Address   string            `json:"address"`
	Meta      map[string]string `json:"meta,omitempty"`
}

// PresenceTransport is what the bus discovery provider needs from the transport layer.
// Defined here (consumer package), implemented by *transport.RemoteClient.
type PresenceTransport interface {
	PublishRawGlobal(ctx context.Context, topic string, payload json.RawMessage) error
	SubscribeRawFanOutGlobal(ctx context.Context, topic string, handler func(payload json.RawMessage)) (func(), error)
}
