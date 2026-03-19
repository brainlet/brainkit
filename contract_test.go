package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	agentembed "github.com/brainlet/brainkit/agent-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// ═══════════════════════════════════════════════════════════════
// LOCAL OPERATIONS — within the Kit's runtime
// ═══════════════════════════════════════════════════════════════

func TestContract_LocalAgentGenerate(t *testing.T) {
	kit := newTestKit(t)

	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "greeter",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: HELLO_CONTRACT",
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

	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_CONTRACT") {
		t.Errorf("unexpected: %q", result.Text)
	}
	t.Logf("Contract LOCAL agent.generate: %q", result.Text)
}

func TestContract_LocalAgentStream(t *testing.T) {
	kit := newTestKit(t)

	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "streamer",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Count from 1 to 3, one per line.",
	})
	if err != nil {
		t.Fatal(err)
	}

	var tokens []string
	result, err := agent.Stream(context.Background(), agentembed.StreamParams{
		Prompt: "Count",
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if len(tokens) == 0 {
		t.Error("expected real-time token callbacks (SSE streaming)")
	}
	t.Logf("Contract LOCAL agent.stream: %d tokens, text: %q", len(tokens), result.Text)
}

func TestContract_LocalAgentWithTools(t *testing.T) {
	kit := newTestKit(t)

	toolCalled := false
	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "tool-user",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always use the add tool when asked to compute.",
		Tools: map[string]agentembed.Tool{
			"add": {
				Description: "Adds two numbers",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"]}`),
				Execute: func(ctx agentembed.ToolContext, args json.RawMessage) (any, error) {
					toolCalled = true
					var input struct{ A, B float64 }
					json.Unmarshal(args, &input)
					return map[string]any{"result": input.A + input.B}, nil
				},
			},
		},
		MaxSteps: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Generate(context.Background(), agentembed.GenerateParams{
		Prompt: "What is 7 + 5? Use the add tool.",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !toolCalled {
		t.Log("Warning: model didn't call the tool")
	}
	if !strings.Contains(result.Text, "12") {
		t.Errorf("expected 12 in response: %q", result.Text)
	}
	t.Logf("Contract LOCAL agent+tools: %q, toolCalled=%v", result.Text, toolCalled)
}

func TestContract_LocalMultipleAgents(t *testing.T) {
	kit := newTestKit(t)

	a1, err := kit.CreateAgent(agentembed.AgentConfig{
		Name: "a1", Model: "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: ALPHA",
	})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := kit.CreateAgent(agentembed.AgentConfig{
		Name: "a2", Model: "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: BETA",
	})
	if err != nil {
		t.Fatal(err)
	}

	r1, err := a1.Generate(context.Background(), agentembed.GenerateParams{Prompt: "Go"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := a2.Generate(context.Background(), agentembed.GenerateParams{Prompt: "Go"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(strings.ToUpper(r1.Text), "ALPHA") {
		t.Errorf("a1: %q", r1.Text)
	}
	if !strings.Contains(strings.ToUpper(r2.Text), "BETA") {
		t.Errorf("a2: %q", r2.Text)
	}
	t.Logf("Contract LOCAL multi-agent: %q, %q", r1.Text, r2.Text)
}

// ═══════════════════════════════════════════════════════════════
// .ts EXECUTION — brainlet import surface
// ═══════════════════════════════════════════════════════════════

func TestContract_TSAgentGenerate(t *testing.T) {
	kit := newTestKit(t)

	result, err := kit.EvalTS(context.Background(), "test.ts", `
		const a = agent({
			model: "openai/gpt-4o-mini",
			instructions: "Reply with exactly: TS_WORKS",
		});
		const r = await a.generate("Say it");
		return JSON.stringify({ text: r.text });
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct{ Text string }
	json.Unmarshal([]byte(result), &out)
	if !strings.Contains(strings.ToUpper(out.Text), "TS_WORKS") {
		t.Errorf("unexpected: %q", out.Text)
	}
	t.Logf("Contract .ts agent.generate: %q", out.Text)
}

func TestContract_TSAIGenerate(t *testing.T) {
	kit := newTestKit(t)

	result, err := kit.EvalTS(context.Background(), "test-ai.ts", `
		try {
			const r = await ai.generate({
				model: "openai/gpt-4o-mini",
				prompt: "Reply with exactly: AI_LOCAL_WORKS",
			});
			return JSON.stringify({ text: r.text, hasUsage: !!r.usage });
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct {
		Text     string `json:"text"`
		HasUsage bool   `json:"hasUsage"`
		Error    string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Error != "" {
		t.Fatalf("ai.generate error: %s", out.Error)
	}
	if !strings.Contains(strings.ToUpper(out.Text), "AI_LOCAL_WORKS") {
		t.Errorf("unexpected: %q", out.Text)
	}
	t.Logf("Contract .ts ai.generate (LOCAL): %q, hasUsage=%v", out.Text, out.HasUsage)
}

func TestContract_TSSandboxContext(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalTS(context.Background(), "test-ctx.ts", `
		return JSON.stringify({
			id: sandbox.id,
			namespace: sandbox.namespace,
			callerID: sandbox.callerID,
		});
	`)
	if err != nil {
		t.Fatal(err)
	}

	var ctx struct {
		ID        string `json:"id"`
		Namespace string `json:"namespace"`
		CallerID  string `json:"callerID"`
	}
	json.Unmarshal([]byte(result), &ctx)

	if ctx.ID == "" {
		t.Error("expected non-empty sandbox.id")
	}
	if ctx.Namespace != "test" {
		t.Errorf("sandbox.namespace = %q, want test", ctx.Namespace)
	}
	t.Logf("Contract .ts sandbox context: %+v", ctx)
}

// ═══════════════════════════════════════════════════════════════
// PLATFORM OPERATIONS — tools through bus
// ═══════════════════════════════════════════════════════════════

func TestContract_TSToolsCall(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/multiply", ShortName: "multiply",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ A, B float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.A * args.B})
				return result, nil
			},
		},
	})

	result, err := kit.EvalTS(context.Background(), "test-tool.ts", `
		const r = await tools.call("multiply", {a: 6, b: 7});
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct{ Result float64 }
	json.Unmarshal([]byte(result), &out)
	if out.Result != 42 {
		t.Errorf("result = %v, want 42", out.Result)
	}
	t.Logf("Contract .ts tools.call: 6 * 7 = %v", out.Result)
}

func TestContract_TSToolsCallShortNameResolution(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/math@1.0.0/square", ShortName: "square",
		Owner: "brainlet", Package: "math", Version: "1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ N float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.N * args.N})
				return result, nil
			},
		},
	})

	result, err := kit.EvalTS(context.Background(), "test-ns.ts", `
		const r = await tools.call("square", {n: 9});
		return JSON.stringify(r);
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct{ Result float64 }
	json.Unmarshal([]byte(result), &out)
	if out.Result != 81 {
		t.Errorf("result = %v, want 81", out.Result)
	}
	t.Logf("Contract .ts tools.call (short name): square(9) = %v", out.Result)
}

func TestContract_TSAgentUsesRegisteredTool(t *testing.T) {
	kit := newTestKit(t)

	// Register a Go tool on the registry (simulates a plugin providing a tool)
	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/multiply", ShortName: "multiply",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Multiplies two numbers",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"number","description":"first number"},"b":{"type":"number","description":"second number"}},"required":["a","b"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ A, B float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.A * args.B})
				return result, nil
			},
		},
	})

	// Agent uses the registered tool via tool("multiply")
	result, err := kit.EvalTS(context.Background(), "test-agent-tool.ts", `
		try {
			const multiplyTool = tool("multiply");
			const a = agent({
				model: "openai/gpt-4o-mini",
				instructions: "Always use the multiply tool. Return just the number.",
				tools: { multiply: multiplyTool },
			});
			const r = await a.generate("What is 6 times 7? Use the multiply tool.");
			return JSON.stringify({ text: r.text, toolCalls: r.toolCalls?.length || 0 });
		} catch(e) {
			return JSON.stringify({ error: e.message });
		}
	`)
	if err != nil {
		t.Fatalf("EvalTS: %v", err)
	}

	var out struct {
		Text      string `json:"text"`
		ToolCalls int    `json:"toolCalls"`
		Error     string `json:"error"`
	}
	json.Unmarshal([]byte(result), &out)
	if out.Error != "" {
		t.Fatalf("error: %s", out.Error)
	}
	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42 in response: %q", out.Text)
	}
	t.Logf("Contract .ts agent + tool('multiply'): %q, toolCalls=%d", out.Text, out.ToolCalls)
}

// ═══════════════════════════════════════════════════════════════
// BUS + REGISTRY — infrastructure
// ═══════════════════════════════════════════════════════════════

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

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{Topic: "echo.test", CallerID: "test", Payload: json.RawMessage(`{"ping":true}`)})
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

// ═══════════════════════════════════════════════════════════════
// KIT ISOLATION
// ═══════════════════════════════════════════════════════════════

func TestContract_KitIsolation(t *testing.T) {
	kit1 := newTestKitNoKey(t)
	kit2 := newTestKitNoKey(t)

	// Different runtimes
	if kit1.agents.ID() == kit2.agents.ID() {
		t.Error("Kits should have different runtime IDs")
	}

	// Agents in kit1 NOT visible in kit2 (separate QuickJS runtimes)
	r1, _ := kit1.agents.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)
	r2, _ := kit2.agents.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)

	if r1 != "[]" || r2 != "[]" {
		t.Logf("kit1 agents: %s, kit2 agents: %s", r1, r2)
	}
}
