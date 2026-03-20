package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestKitCreate(t *testing.T) {
	kit := newTestKit(t)
	if kit.Namespace() != "test" {
		t.Errorf("namespace = %q, want test", kit.Namespace())
	}
	if kit.CallerID() == "" {
		t.Error("empty callerID")
	}
}

func TestKitAgentGenerate(t *testing.T) {
	kit := newTestKit(t)

	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "test",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: KIT_WORKS",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Generate(context.Background(), agentembed.GenerateParams{
		Prompt: "Say it",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Response: %q", result.Text)
	if result.Text == "" {
		t.Error("expected non-empty response")
	}
}

func TestKitBusPubSub(t *testing.T) {
	kit := newTestKitNoKey(t)

	received := make(chan bool, 1)
	kit.Bus.On("test.event", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- true
	})

	kit.Bus.Send(bus.Message{
		Topic:    "test.event",
		CallerID: "test",
	})

	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestKitToolRegistry(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name:      "brainlet/platform@1.0.0/echo",
		ShortName: "echo",
		Owner:     "brainlet",
		Package:   "platform",
		Version:   "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"echoed":true}`), nil
			},
		},
	})

	tool, err := kit.Tools.Resolve("echo")
	if err != nil {
		t.Fatal(err)
	}

	result, err := tool.Executor.Call(context.Background(), "user", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"echoed":true}` {
		t.Errorf("result = %s", result)
	}
}
