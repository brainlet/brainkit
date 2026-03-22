package sdk

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
)

// Client is a deprecated alias for Runtime. Use Runtime instead.
// Kept temporarily for plugin.go/serve.go compatibility during migration.
type Client = Runtime

// pluginClient implements Runtime for plugin processes connected via Watermill.
type pluginClient struct {
	remote    *messaging.RemoteClient
	namespace string
}

func (c *pluginClient) PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (string, error) {
	return c.remote.PublishRaw(ctx, topic, payload)
}

func (c *pluginClient) SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (func(), error) {
	return c.remote.SubscribeRaw(ctx, topic, handler)
}

func (c *pluginClient) Close() error { return nil }
