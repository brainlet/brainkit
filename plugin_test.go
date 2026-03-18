package brainkit

import (
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
)

func buildTestPlugin(t *testing.T) string {
	t.Helper()
	binary := t.TempDir() + "/test-plugin"
	cmd := exec.Command("go", "build", "-o", binary, "./testdata/plugin/")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build test plugin: %s\n%s", err, out)
	}
	return binary
}

func TestPlugin_Lifecycle(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-plugin",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "echo", Binary: binary},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Plugin should have registered its tool
	tool, err := kit.Tools.Resolve("plugin.echo.echo", "test")
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
			{Name: "echo", Binary: binary},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Call plugin tool via bus
	resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"plugin.echo.echo","input":{"hello":"world"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)
	if result["hello"] != "world" {
		t.Errorf("result = %s", resp.Payload)
	}
}

func TestPlugin_EventForwarding(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-plugin-events",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "echo", Binary: binary},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Subscribe to ack events (plugin forwards events back as test.ack)
	received := make(chan bus.Message, 1)
	kit.Bus.On("test.ack", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	// Send an event the plugin subscribes to
	kit.Bus.Send(bus.Message{
		Topic:    "test.events.hello",
		CallerID: "test",
		Payload:  json.RawMessage(`{"msg":"ping"}`),
	})

	select {
	case msg := <-received:
		if string(msg.Payload) != `{"msg":"ping"}` {
			t.Errorf("ack payload = %s", msg.Payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for plugin ack")
	}
}
