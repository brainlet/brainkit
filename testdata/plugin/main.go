package main

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/sdk"
)

type TestPlugin struct {
	client sdk.BrainletClient
}

func (p *TestPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "test-echo",
		Version:     "1.0.0",
		Description: "Test plugin that echoes tool calls",
		Tools: []sdk.ToolDefinition{
			{Name: "echo", Description: "Echo input", InputSchema: `{"type":"object"}`},
		},
		Subscriptions: []sdk.SubscriptionDefinition{
			{Topic: "test.events.*"},
		},
	}
}

func (p *TestPlugin) OnStart(client sdk.BrainletClient) error {
	p.client = client
	return nil
}

func (p *TestPlugin) OnStop() error { return nil }

func (p *TestPlugin) HandleToolCall(_ context.Context, tool string, input json.RawMessage) (json.RawMessage, error) {
	return input, nil // echo
}

func (p *TestPlugin) HandleEvent(ctx context.Context, event sdk.Event) error {
	if p.client != nil {
		p.client.Send(ctx, "test.ack", event.Payload)
	}
	return nil
}

func (p *TestPlugin) HandleIntercept(_ context.Context, msg sdk.InterceptMessage) (*sdk.InterceptMessage, error) {
	return &msg, nil
}

func main() {
	sdk.Serve(&TestPlugin{})
}
