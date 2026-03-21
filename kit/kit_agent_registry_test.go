package kit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/bus"
)

func TestAgentRegistry_RegisterAndList(t *testing.T) {
	kit := newTestKitNoKey(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "agents.register",
		CallerID: kit.CallerID(),
		Payload:  json.RawMessage(`{"name":"coder","capabilities":["code-review","refactor"],"model":"openai/gpt-4o","kit":"test-kit-1"}`),
	})
	if err != nil {
		t.Fatalf("agents.register: %v", err)
	}
	var regResult struct {
		Registered string `json:"registered"`
	}
	json.Unmarshal(resp.Payload, &regResult)
	if regResult.Registered != "coder" {
		t.Fatalf("expected registered=coder, got %q", regResult.Registered)
	}

	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "agents.list",
		CallerID: kit.CallerID(),
		Payload:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("agents.list: %v", err)
	}
	var agents []AgentInfo
	json.Unmarshal(resp.Payload, &agents)
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "coder" {
		t.Fatalf("name=%q", agents[0].Name)
	}
	if agents[0].Status != "idle" {
		t.Fatalf("status=%q", agents[0].Status)
	}
	if agents[0].Model != "openai/gpt-4o" {
		t.Fatalf("model=%q", agents[0].Model)
	}
	if len(agents[0].Capabilities) != 2 {
		t.Fatalf("capabilities=%d", len(agents[0].Capabilities))
	}
}

func TestAgentRegistry_DiscoverByCapability(t *testing.T) {
	kit := newTestKitNoKey(t)

	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.register",
		Payload: json.RawMessage(`{"name":"coder","capabilities":["code-review","refactor"],"model":"openai/gpt-4o","kit":"k1"}`),
	})
	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.register",
		Payload: json.RawMessage(`{"name":"writer","capabilities":["docs","blog"],"model":"anthropic/claude-sonnet-4-20250514","kit":"k1"}`),
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.discover",
		Payload: json.RawMessage(`{"capability":"code-review"}`),
	})
	if err != nil {
		t.Fatalf("agents.discover: %v", err)
	}
	var agents []AgentInfo
	json.Unmarshal(resp.Payload, &agents)
	if len(agents) != 1 || agents[0].Name != "coder" {
		t.Fatalf("expected coder, got %+v", agents)
	}

	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.discover",
		Payload: json.RawMessage(`{"model":"anthropic/claude-sonnet-4-20250514"}`),
	})
	if err != nil {
		t.Fatalf("discover by model: %v", err)
	}
	json.Unmarshal(resp.Payload, &agents)
	if len(agents) != 1 || agents[0].Name != "writer" {
		t.Fatalf("expected writer, got %+v", agents)
	}
}

func TestAgentRegistry_SetGetStatus(t *testing.T) {
	kit := newTestKitNoKey(t)

	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.register",
		Payload: json.RawMessage(`{"name":"worker","capabilities":[],"model":"openai/gpt-4o-mini","kit":"k1"}`),
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.set-status",
		Payload: json.RawMessage(`{"name":"worker","status":"busy"}`),
	})
	if err != nil {
		t.Fatalf("set-status: %v", err)
	}
	var ok struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &ok)
	if !ok.OK {
		t.Fatal("expected ok=true")
	}

	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.get-status",
		Payload: json.RawMessage(`{"name":"worker"}`),
	})
	if err != nil {
		t.Fatalf("get-status: %v", err)
	}
	var status struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	json.Unmarshal(resp.Payload, &status)
	if status.Status != "busy" {
		t.Fatalf("expected busy, got %q", status.Status)
	}
}

func TestAgentRegistry_Unregister(t *testing.T) {
	kit := newTestKitNoKey(t)

	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.register",
		Payload: json.RawMessage(`{"name":"temp","capabilities":[],"model":"test","kit":"k1"}`),
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.unregister",
		Payload: json.RawMessage(`{"name":"temp"}`),
	})
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}
	var ok struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &ok)
	if !ok.OK {
		t.Fatal("expected ok=true")
	}

	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.list",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var agents []AgentInfo
	json.Unmarshal(resp.Payload, &agents)
	if len(agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(agents))
	}
}

func TestAgentRegistry_AutoUnregisterOnKitClose(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.agentReg.register(AgentInfo{Name: "ephemeral-1", Status: "idle", Kit: "test-kit"})
	kit.agentReg.register(AgentInfo{Name: "ephemeral-2", Status: "idle", Kit: "test-kit"})

	agents := kit.agentReg.list(nil)
	if len(agents) != 2 {
		t.Fatalf("expected 2 before unregisterAll, got %d", len(agents))
	}

	removed := kit.agentReg.unregisterAllForKit("test-kit")
	if removed != 2 {
		t.Fatalf("expected 2 removed, got %d", removed)
	}

	agents = kit.agentReg.list(nil)
	if len(agents) != 0 {
		t.Fatalf("expected 0 after unregisterAll, got %d", len(agents))
	}
}

func TestAgentRegistry_AgentRequestRoutesToGenerate(t *testing.T) {
	kit := newTestKitNoKey(t)

	_, err := kit.EvalTS(context.Background(), "__setup_test_agent.ts", `
		globalThis.__kit_agent_registry["test-bot"] = {
			generate: async function(prompt) {
				return { text: "echo: " + prompt };
			},
		};
		return JSON.stringify({ ok: true });
	`)
	if err != nil {
		t.Fatalf("setup test agent: %v", err)
	}

	kit.agentReg.register(AgentInfo{
		Name:   "test-bot",
		Status: "idle",
		Kit:    kit.agents.ID(),
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "agents.request",
		CallerID: kit.CallerID(),
		Payload:  json.RawMessage(`{"name":"test-bot","prompt":"hello world"}`),
	})
	if err != nil {
		t.Fatalf("agents.request: %v", err)
	}

	var result struct{ Text string `json:"text"` }
	json.Unmarshal(resp.Payload, &result)
	if result.Text != "echo: hello world" {
		t.Fatalf("expected 'echo: hello world', got %q", result.Text)
	}
}

func TestAgentRegistry_MessageFireAndForget(t *testing.T) {
	kit := newTestKitNoKey(t)

	bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "agents.register",
		Payload: json.RawMessage(`{"name":"listener","capabilities":[],"model":"test","kit":"k1"}`),
	})

	delivered := make(chan bool, 1)
	kit.Bus.On("agent.listener.message", func(msg bus.Message, _ bus.ReplyFunc) {
		delivered <- true
	})

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:    "agents.message",
		CallerID: kit.CallerID(),
		Payload:  json.RawMessage(`{"target":"listener","payload":{"task":"do something"}}`),
	})
	if err != nil {
		t.Fatalf("agents.message: %v", err)
	}

	var result struct{ Delivered bool `json:"delivered"` }
	json.Unmarshal(resp.Payload, &result)
	if !result.Delivered {
		t.Fatal("expected delivered=true")
	}

	select {
	case <-delivered:
	case <-time.After(2 * time.Second):
		t.Fatal("message was not delivered to agent topic")
	}
}
