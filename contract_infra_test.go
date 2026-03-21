package brainkit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

func TestContract_BusPubSub(t *testing.T) {
	kit := newTestKitNoKey(t)

	received := make(chan string, 1)
	kit.Bus.On("test.event", func(msg bus.Message, _ bus.ReplyFunc) {
		received <- string(msg.Payload)
	})

	kit.Bus.Send(bus.Message{
		Topic:    "test.event",
		CallerID: "test",
		Payload:  json.RawMessage(`"hello"`),
	})

	select {
	case data := <-received:
		if data != `"hello"` {
			t.Errorf("payload = %s", data)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestContract_BusRequestResponse(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Bus.On("echo.*", func(msg bus.Message, reply bus.ReplyFunc) {
		reply(msg.Payload)
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "echo.test",
		CallerID: "test",
		Payload:  json.RawMessage(`{"ping":true}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Payload) != `{"ping":true}` {
		t.Errorf("payload = %s", resp.Payload)
	}
}

func TestContract_ToolRegistryResolve(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/echo", ShortName: "echo",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Echoes input",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return input, nil
			},
		},
	})

	tool, err := kit.Tools.Resolve("echo")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Description != "Echoes input" {
		t.Errorf("description = %q", tool.Description)
	}

	result, err := tool.Executor.Call(context.Background(), "user", json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"x":1}` {
		t.Errorf("result = %s", result)
	}
}

func TestContract_KitIsolation(t *testing.T) {
	kit1 := newTestKitNoKey(t)
	kit2 := newTestKitNoKey(t)

	if kit1.agents.ID() == kit2.agents.ID() {
		t.Error("Kits should have different runtime IDs")
	}

	r1, _ := kit1.agents.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)
	r2, _ := kit2.agents.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)

	if r1 != "[]" || r2 != "[]" {
		t.Logf("kit1 agents: %s, kit2 agents: %s", r1, r2)
	}
}
