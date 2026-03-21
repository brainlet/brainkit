package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/bus"
	pluginv1 "github.com/brainlet/brainkit/proto/plugin/v1"
	"github.com/brainlet/brainkit/internal/registry"
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
			{Name: "echo", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Plugin should have registered its tool as brainlet/test-echo@1.0.0/echo
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

	// Call plugin tool via bus
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

	// Subscribe to ack events (plugin forwards events back as test.ack)
	received := make(chan bus.Message, 1)
	kit.Bus.On("test.ack", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	// Send an event the plugin subscribes to
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

// ═══════════════════════════════════════════════════════════════
// New tests — Plan 4 hardening coverage
// ═══════════════════════════════════════════════════════════════

func TestPlugin_ConcurrentToolCalls(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-concurrent",
		Namespace: "test",
		Plugins:   []PluginConfig{{Name: "echo", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	const N = 50
	results := make(chan error, N)

	for i := 0; i < N; i++ {
		go func(idx int) {
			input := fmt.Sprintf(`{"name":"echo","input":{"message":"msg-%d"}}`, idx)
			resp, err := bus.AskSync(kit.Bus, t.Context(), bus.Message{
				Topic:    "tools.call",
				CallerID: "test",
				Payload:  json.RawMessage(input),
			})
			if err != nil {
				results <- fmt.Errorf("call %d: %w", idx, err)
				return
			}

			var result map[string]any
			if err := json.Unmarshal(resp.Payload, &result); err != nil {
				results <- fmt.Errorf("call %d unmarshal: %w", idx, err)
				return
			}

			expected := fmt.Sprintf("msg-%d", idx)
			if result["message"] != expected {
				results <- fmt.Errorf("call %d: expected %q, got %v", idx, expected, result["message"])
				return
			}

			results <- nil
		}(i)
	}

	var errors []error
	for i := 0; i < N; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		for _, e := range errors {
			t.Error(e)
		}
		t.Fatalf("%d/%d concurrent calls failed", len(errors), N)
	}
}

func TestPlugin_TwoPluginsInteracting(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-two-plugins",
		Namespace: "test",
		Plugins: []PluginConfig{
			{Name: "alpha", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
			{Name: "beta", Binary: binary, ShutdownTimeout: 500 * time.Millisecond, SIGTERMTimeout: 500 * time.Millisecond},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	received := make(chan bus.Message, 10)
	kit.Bus.On("test.ack", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- msg
	})

	kit.Bus.Send(bus.Message{
		Topic:    "test.events.hello",
		CallerID: "test",
		Payload:  json.RawMessage(`{"data":"from-test"}`),
	})

	// Expect 2 acks (one from each plugin)
	timeout := time.After(5 * time.Second)
	acks := 0
	for acks < 2 {
		select {
		case <-received:
			acks++
		case <-timeout:
			t.Fatalf("timeout waiting for 2 acks, got %d", acks)
		}
	}
}

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

	// Verify initial tool call works
	_, err = bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"pre-crash"}}`),
	})
	if err != nil {
		t.Fatalf("pre-crash tool call failed: %v", err)
	}

	// Trigger crash — will timeout because plugin dies before responding
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	bus.AskSync(kit.Bus, ctx, bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"crash","input":{}}`),
	})

	// Wait for restart (healthLoop detects done, restarts with backoff)
	time.Sleep(4 * time.Second)

	// Verify tool call works again after restart
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

	// Get the pluginConn to test backpressure directly
	kit.plugins.mu.Lock()
	pc := kit.plugins.plugins["echo"]
	kit.plugins.mu.Unlock()

	if pc == nil {
		t.Fatal("plugin conn not found")
	}

	// Fill the semaphore manually
	for i := 0; i < maxPending; i++ {
		pc.eventSem <- struct{}{}
	}

	// Now try to send an event — should be dropped
	dropped := !pc.safeSendEvent(&pluginv1.PluginMessage{
		Id:    "test-bp",
		Type:  "event",
		Topic: "test.events.bp",
	})
	if !dropped {
		t.Error("expected event to be dropped when semaphore is full")
	}

	// Tool calls should still work (bypass backpressure)
	sent := pc.safeSendEvent(&pluginv1.PluginMessage{
		Id:    "test-tool",
		Type:  "tool.call",
		Topic: "echo",
	})
	if !sent {
		t.Error("tool.call should never be dropped by backpressure")
	}

	// Drain semaphore
	for i := 0; i < maxPending; i++ {
		<-pc.eventSem
	}

	// Now events should send again
	sent = pc.safeSendEvent(&pluginv1.PluginMessage{
		Id:    "test-after",
		Type:  "event",
		Topic: "test.events.after",
	})
	if !sent {
		t.Error("expected event to be sent after semaphore drained")
	}

	t.Log("backpressure: events correctly dropped when semaphore full, tool.call bypasses")
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

	// Register a Kit-side tool that the plugin will call
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

	// Call the plugin's ask-kit tool — it internally calls client.Ask to Kit tool
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

	// Verify initial tool call works
	_, err = bus.AskSync(kit.Bus, t.Context(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"before"}}`),
	})
	if err != nil {
		t.Fatalf("pre-break tool call failed: %v", err)
	}

	// Force-close the gRPC stream
	kit.plugins.mu.Lock()
	pc := kit.plugins.plugins["echo"]
	kit.plugins.mu.Unlock()

	pc.sendMu.Lock()
	pc.stream.CloseSend()
	pc.sendMu.Unlock()

	// Wait for recovery or restart
	time.Sleep(3 * time.Second)

	// Verify tool call works after recovery
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

// ═══════════════════════════════════════════════════════════════
// Interceptor Tests
// ═══════════════════════════════════════════════════════════════

func TestPlugin_InterceptorModifiesMetadata(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-intercept-meta",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "echo",
				Binary:          binary,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "intercept"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Tool call — interceptor modifies metadata but passes through
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"test"}}`),
	})
	if err != nil {
		t.Fatalf("tool call: %v", err)
	}

	var result map[string]any
	json.Unmarshal(resp.Payload, &result)

	if errMsg, ok := result["error"]; ok {
		t.Fatalf("tool call returned error: %v", errMsg)
	}

	if result["message"] != "test" {
		t.Errorf("expected echo result, got: %s", resp.Payload)
	}

	t.Log("interceptor pass-through succeeded")
}

func TestPlugin_InterceptorBlocks(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-intercept-block",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "blocker",
				Binary:          binary,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "intercept-block"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Tool call — interceptor rejects it.
	// AskSync returns a reply with {"error":"..."} payload, not a Go error.
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"should-be-blocked"}}`),
	})
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	var errResult struct {
		Error string `json:"error"`
	}
	json.Unmarshal(resp.Payload, &errResult)

	if errResult.Error == "" {
		t.Fatalf("expected interceptor rejection error in payload, got: %s", resp.Payload)
	}

	if !strings.Contains(errResult.Error, "blocked") && !strings.Contains(errResult.Error, "interceptor") {
		t.Errorf("expected interceptor rejection, got: %q", errResult.Error)
	}
}

func TestPlugin_InterceptorTimeout(t *testing.T) {
	binary := buildTestPlugin(t)

	kit, err := New(Config{
		Name:      "test-kit-intercept-timeout",
		Namespace: "test",
		Plugins: []PluginConfig{
			{
				Name:            "slow-interceptor",
				Binary:          binary,
				ShutdownTimeout: 500 * time.Millisecond,
				SIGTERMTimeout:  500 * time.Millisecond,
				Env:             map[string]string{"TEST_PLUGIN_MODE": "intercept-slow"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	start := time.Now()

	// Tool call — interceptor sleeps 10s, timeout is 5s
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "tools.call",
		CallerID: "test",
		Payload:  json.RawMessage(`{"name":"echo","input":{"message":"should-timeout"}}`),
	})

	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}

	var errResult struct {
		Error string `json:"error"`
	}
	json.Unmarshal(resp.Payload, &errResult)

	if errResult.Error == "" {
		t.Fatalf("expected timeout error in payload, got: %s", resp.Payload)
	}

	// Should timeout around 5s, not immediately and not at 10s
	if elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Errorf("expected ~5s timeout, got %v", elapsed)
	}

	if !strings.Contains(errResult.Error, "timeout") {
		t.Errorf("expected timeout error, got: %q", errResult.Error)
	}
}
