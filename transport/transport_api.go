package transport

// Public type aliases and constructors for discovery and transports.
// Implementations live in internal/transport.

import (
	itransport "github.com/brainlet/brainkit/internal/transport"
	"github.com/nats-io/nats.go"
)

// Discovery resolves peer addresses for Kit-to-Kit networking.
type Discovery = itransport.Discovery

// Peer represents a discoverable Kit instance.
type Peer = itransport.Peer

// DiscoveryConfig configures the discovery mechanism.
type DiscoveryConfig = itransport.DiscoveryConfig

// MulticastDiscovery uses UDP multicast for LAN peer finding.
type MulticastDiscovery = itransport.MulticastDiscovery

// StaticDiscovery resolves peers from a fixed configuration map.
type StaticDiscovery = itransport.StaticDiscovery

// NewMulticastDiscovery creates a UDP multicast discovery for LAN peer finding.
func NewMulticastDiscovery(service string) (*MulticastDiscovery, error) {
	return itransport.NewMulticastDiscovery(service)
}

// NewStaticDiscovery creates a discovery backed by a fixed peer map.
func NewStaticDiscovery(peers map[string]string) *StaticDiscovery {
	return itransport.NewStaticDiscovery(peers)
}

// GRPCTransport implements bus.Transport with gRPC-based forwarding to remote Kits.
type GRPCTransport = itransport.GRPCTransport

// NewGRPCTransport creates a GRPCTransport with an in-process local transport.
func NewGRPCTransport() *GRPCTransport {
	return itransport.NewGRPCTransport()
}

// NATSTransport implements bus.Transport using NATS as the message broker.
type NATSTransport = itransport.NATSTransport

// NewNATSTransport connects to a NATS server and returns a transport.
func NewNATSTransport(url string, opts ...nats.Option) (*NATSTransport, error) {
	return itransport.NewNATSTransport(url, opts...)
}
