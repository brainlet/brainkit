package brainkit

import (
	"github.com/brainlet/brainkit/bus"
)

// GRPCTransport implements bus.Transport by combining local delivery
// with gRPC-based forwarding to plugins and remote Kits.
// Kit-to-Kit peer forwarding is deferred — this handles plugin connections only.
type GRPCTransport struct {
	local *bus.InProcessTransport
}

// NewGRPCTransport creates a transport that delegates to InProcessTransport.
// Plugin communication is handled by pluginManager directly via gRPC streams,
// not through the transport layer — plugins register tools/subscriptions on the
// local bus, and the plugin manager forwards events via the stream.
func NewGRPCTransport() *GRPCTransport {
	return &GRPCTransport{
		local: bus.NewInProcessTransport(),
	}
}

func (t *GRPCTransport) Publish(msg bus.Message) error {
	return t.local.Publish(msg)
}

func (t *GRPCTransport) Forward(msg bus.Message, target string) error {
	// For v1, plugin forwarding is handled by pluginManager directly.
	// Remote Kit forwarding (host:X/kit:Y) is deferred.
	// Fall through to local publish for addressed messages.
	return t.local.Publish(msg)
}

func (t *GRPCTransport) Subscribe(info bus.SubscriberInfo) error {
	return t.local.Subscribe(info)
}

func (t *GRPCTransport) Unsubscribe(id bus.SubscriptionID) error {
	return t.local.Unsubscribe(id)
}

func (t *GRPCTransport) Metrics() bus.TransportMetrics {
	return t.local.Metrics()
}

func (t *GRPCTransport) SubscriberCount() int {
	return t.local.SubscriberCount()
}

func (t *GRPCTransport) Close() error {
	return t.local.Close()
}
