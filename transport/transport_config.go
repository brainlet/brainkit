package transport

// NetworkConfig configures Kit-to-Kit networking.
type NetworkConfig struct {
	Listen    string            // ":9090" — listen for incoming connections
	Peers     map[string]string // name → address: {"server-2": "10.0.1.5:9090"}
	Discovery DiscoveryConfig   // optional discovery configuration
}

// NATSConfig configures the NATS transport.
type NATSConfig struct {
	URL  string // "nats://localhost:4222"
	Name string // client name (defaults to Kit name)
}
