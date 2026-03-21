package brainkit

import (
	transportpkg "github.com/brainlet/brainkit/transport"
	"github.com/nats-io/nats.go"
)

type NetworkConfig = transportpkg.NetworkConfig
type NATSConfig = transportpkg.NATSConfig

type Discovery = transportpkg.Discovery
type Peer = transportpkg.Peer
type DiscoveryConfig = transportpkg.DiscoveryConfig
type MulticastDiscovery = transportpkg.MulticastDiscovery
type StaticDiscovery = transportpkg.StaticDiscovery
type GRPCTransport = transportpkg.GRPCTransport
type NATSTransport = transportpkg.NATSTransport

func NewMulticastDiscovery(service string) (*MulticastDiscovery, error) {
	return transportpkg.NewMulticastDiscovery(service)
}

func NewStaticDiscovery(peers map[string]string) *StaticDiscovery {
	return transportpkg.NewStaticDiscovery(peers)
}

func NewGRPCTransport() *GRPCTransport {
	return transportpkg.NewGRPCTransport()
}

func NewNATSTransport(url string, opts ...nats.Option) (*NATSTransport, error) {
	return transportpkg.NewNATSTransport(url, opts...)
}
