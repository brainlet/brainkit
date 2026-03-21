package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
)

func TestPlugin_CrashRestart(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-crash",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "echo",
				Binary:          binary,
				AutoRestart:     true,
				MaxRestarts:     3,
				HealthInterval:  500 * time.Millisecond,
				StartTimeout:    5 * time.Second,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "crash"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	_, err = bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"pre-crash"}}`),
	})
	if err != nil {
		t.Fatalf("pre-crash tool call failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"crash","input":{}}`),
	})

	time.Sleep(4 * time.Second)

	resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"post-crash"}}`),
	})
	if err != nil {
		t.Fatalf("post-restart tool call failed: %v", err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)
	if result["message"] != "post-crash" {
		t.Errorf("result = %s", resp.Payload)
	}
}

func TestPlugin_Timeout(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-timeout",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "echo",
				Binary:          binary,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "timeout"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, err = bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"hang","input":{}}`),
	})

	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	if elapsed < 4*time.Second {
		t.Errorf("expected ~5s timeout, got %v", elapsed)
	}
}

func TestPlugin_Backpressure(t *testing.T) {
	binary := buildTestPlugin(t)

	maxPending := 3

	kit, err := New(Config{
		Name:      "test-kit-backpressure",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "echo",
				Binary:          binary,
				MaxPending:      maxPending,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	pc := kit.plugins.GetConn("echo")
	if pc == nil {
		t.Fatal("plugin conn not found")
	}

	pc.TestBackpressure(maxPending, func(sendEvent func(*pluginv1.PluginMessage) bool) {
		dropped := !sendEvent(&pluginv1.PluginMessage{
			Id:    "test-bp",
			Type:  "event",
			Topic: "test.events.bp",
		})
		if !dropped {
			t.Error("expected event to be dropped when semaphore is full")
		}

		sent := sendEvent(&pluginv1.PluginMessage{
			Id:    "test-tool",
			Type:  "tool.call",
			Topic: "echo",
		})
		if !sent {
			t.Error("tool.call should never be dropped by backpressure")
		}
	})

	pc.TestBackpressure(0, func(sendEvent func(*pluginv1.PluginMessage) bool) {
		sent := sendEvent(&pluginv1.PluginMessage{
			Id:    "test-after",
			Type:  "event",
			Topic: "test.events.after",
		})
		if !sent {
			t.Error("expected event to be sent after semaphore drained")
		}
	})

	t.Log("backpressure: events correctly dropped when semaphore full, tool.call bypasses")
}

func TestPlugin_StreamRecovery(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-stream-recovery",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "echo",
				Binary:          binary,
				AutoRestart:     true,
				HealthInterval:  1 * time.Second,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	_, err = bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"before"}}`),
	})
	if err != nil {
		t.Fatalf("pre-break tool call failed: %v", err)
	}

	pc := kit.plugins.GetConn("echo")
	if pc == nil {
		t.Fatal("plugin conn not found")
	}
	pc.TestForceCloseStream()

	time.Sleep(3 * time.Second)

	resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"after"}}`),
	})
	if err != nil {
		t.Fatalf("post-recovery tool call failed: %v", err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)
	if result["message"] != "after" {
		t.Errorf("result = %s", resp.Payload)
	}
}
