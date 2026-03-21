package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
)

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
