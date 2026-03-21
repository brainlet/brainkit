package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestPlugin_Lifecycle(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-plugin",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "echo", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	tool, err := kit.Tools.Resolve("brainlet/test-echo@1.0.0/echo")
	if err != nil {
		t.Fatalf("tool not registered: %v", err)
	}
	if tool.Description != "Echo input" {
		t.Errorf("description = %q", tool.Description)
	}
}

func TestPlugin_ToolCall(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-plugin-tool",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "echo", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"world"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)
	if result["message"] != "world" {
		t.Errorf("result = %s", resp.Payload)
	}
}

func TestPlugin_EventForwarding(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-plugin-events",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "echo", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	received := make(chan bus.Message, 1)
	kit.Bus.On("test.ack", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	kit.Bus.Send(bus.Message{
		Topic:    "test.events.hello",
		CallerID: "test",
		Payload:  json.RawMessage(`{"data":"ping"}`),
	})

	select {
	case msg := <-received:
		if string(msg.Payload) != `{"data":"ping"}` {
			t.Errorf("ack payload = %s", msg.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for plugin ack")
	}
}

func TestPlugin_KitAPIRoundTrip(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-askkit",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "asker",
				Binary:          binary,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "ask-kit"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/add", ShortName: "add",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Add two numbers",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var in struct{ A, B float64 }
				json.Unmarshal(input, &in)
				return json.Marshal(map[string]float64{"result": in.A + in.B})
			},
		},
	})

	resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"ask-kit","input":{"tool":"add","input":{"a":5,"b":3}}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)
	if result["result"] != float64(8) {
		t.Errorf("expected 8, got %v (payload: %s)", result["result"], resp.Payload)
	}
}
