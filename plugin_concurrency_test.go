package brainkit

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
)

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

	const n = 50
	results := make(chan error, n)

	for i := 0; i < n; i++ {
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
	for i := 0; i < n; i++ {
		if err := <-results; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		for _, err := range errors {
			t.Error(err)
		}
		t.Fatalf("%d/%d concurrent calls failed", len(errors), n)
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
