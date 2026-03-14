package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestKitCreateSandbox(t *testing.T) {
	kit := newTestKit(t)

	sandbox, err := kit.CreateSandbox(SandboxConfig{
		Namespace: "test",
		CallerID:  "test.kit",
	})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	if sandbox.ID() == "" {
		t.Error("expected non-empty sandbox ID")
	}
	if sandbox.Namespace() != "test" {
		t.Errorf("namespace = %q, want test", sandbox.Namespace())
	}
	if sandbox.CallerID() != "test.kit" {
		t.Errorf("callerID = %q, want test.kit", sandbox.CallerID())
	}
}

func TestKitSandboxAgentGenerate(t *testing.T) {
	kit := newTestKit(t)

	sandbox, err := kit.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	agent, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
		Name:         "test",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: KIT_WORKS",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Generate(context.Background(), agentembed.GenerateParams{
		Prompt: "Say the magic word",
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Response: %q", result.Text)
	if result.Text == "" {
		t.Error("expected non-empty response")
	}
}

func TestKitMultipleSandboxes(t *testing.T) {
	kit := newTestKitNoKey(t)

	s1, err := kit.CreateSandbox(SandboxConfig{Namespace: "team-a"})
	if err != nil {
		t.Fatal(err)
	}
	s2, err := kit.CreateSandbox(SandboxConfig{Namespace: "team-b"})
	if err != nil {
		t.Fatal(err)
	}
	defer s1.Close()
	defer s2.Close()

	if s1.ID() == s2.ID() {
		t.Error("sandboxes should have different IDs")
	}
	if s1.Namespace() == s2.Namespace() {
		t.Error("sandboxes should have different namespaces")
	}
}

func TestKitBusPubSub(t *testing.T) {
	kit := newTestKitNoKey(t)

	received := make(chan bool, 1)
	kit.Bus.Subscribe("test.event", func(msg bus.Message) {
		received <- true
	})

	kit.Bus.Send(context.Background(), bus.Message{
		Topic:    "test.event",
		CallerID: "test",
	})

	select {
	case <-received:
		// ok
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestKitToolRegistry(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name:      "platform.echo",
		ShortName: "echo",
		Namespace: "platform",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{"echoed":true}`), nil
			},
		},
	})

	tool, err := kit.Tools.Resolve("echo", "user")
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
