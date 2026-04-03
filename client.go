package brainkit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

// BusClient is a lightweight client for sending bus commands to a running Node.
// No Kernel, no JS runtime — just transport + publish/subscribe.
// Use NewClient() to create one from a NodeConfig.
type BusClient struct {
	remote    *messaging.RemoteClient
	transport *messaging.Transport
}

// PublishRaw sends a message to a topic. Returns correlationID.
func (c *BusClient) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return c.remote.PublishRaw(ctx, topic, payload)
}

// SubscribeRaw subscribes to a topic.
func (c *BusClient) SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (func(), error) {
	return c.remote.SubscribeRaw(ctx, topic, handler)
}

// Close shuts down the transport.
func (c *BusClient) Close() error {
	return c.transport.Close()
}

// NewClient creates a BusClient that connects to a running Node via transport.
// The BusClient shares the same transport type and namespace as the Node,
// so bus commands are delivered to the Node's command handlers.
func NewClient(cfg NodeConfig) (*BusClient, error) {
	namespace := cfg.Kernel.Namespace
	if namespace == "" {
		if cfg.Namespace != "" {
			namespace = cfg.Namespace
		} else {
			namespace = "user"
		}
	}

	transport, err := messaging.NewTransportSet(cfg.Messaging.transportConfig())
	if err != nil {
		return nil, err
	}

	remote := messaging.NewRemoteClientWithTransport(namespace, "cli", transport)
	return &BusClient{
		remote:    remote,
		transport: transport,
	}, nil
}
