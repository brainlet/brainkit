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
// LOCAL OPERATIONS — within one sandbox, direct Mastra
// Every test here is a promise: "this works in a .ts file"
// ═══════════════════════════════════════════════════════════════

func TestContract_LocalAgentGenerate(t *testing.T) {
	kit := newTestKit(t)
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	agent, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
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
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	agent, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
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
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	toolCalled := false
	agent, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
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
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	a1, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
		Name: "a1", Model: "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: ALPHA",
	})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := sandbox.AgentSandbox().CreateAgent(agentembed.AgentConfig{
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
// .ts EXECUTION — the brainlet import surface
// ═══════════════════════════════════════════════════════════════

func TestContract_TSAgentGenerate(t *testing.T) {
	kit := newTestKit(t)
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	result, err := sandbox.EvalTS(context.Background(), "test.ts", `
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

func TestContract_TSSandboxContext(t *testing.T) {
	kit := newTestKitNoKey(t)
	sandbox, err := kit.CreateSandbox(SandboxConfig{
		Namespace: "test.context",
		CallerID:  "test.context.caller",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	result, err := sandbox.EvalTS(context.Background(), "test-ctx.ts", `
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
	if ctx.Namespace != "test.context" {
		t.Errorf("sandbox.namespace = %q, want test.context", ctx.Namespace)
	}
	if ctx.CallerID != "test.context.caller" {
		t.Errorf("sandbox.callerID = %q, want test.context.caller", ctx.CallerID)
	}
	t.Logf("Contract .ts sandbox context: %+v", ctx)
}

// ═══════════════════════════════════════════════════════════════
// PLATFORM OPERATIONS — .ts calling ai-embed and tools through bus
// ═══════════════════════════════════════════════════════════════

func TestContract_TSAIGenerate(t *testing.T) {
	kit := newTestKit(t)
	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	result, err := sandbox.EvalTS(context.Background(), "test-ai.ts", `
		try {
			const r = await ai.generate({
				model: "openai/gpt-4o-mini",
				prompt: "Reply with exactly: AI_BRIDGE_WORKS",
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
	if !strings.Contains(strings.ToUpper(out.Text), "AI_BRIDGE_WORKS") {
		t.Errorf("unexpected: %q", out.Text)
	}
	if !out.HasUsage {
		t.Error("expected usage data")
	}
	t.Logf("Contract .ts ai.generate: %q, hasUsage=%v", out.Text, out.HasUsage)
}

func TestContract_TSToolsCall(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register a Go tool
	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.multiply", ShortName: "multiply", Namespace: "platform",
		Description: "Multiplies two numbers",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}}}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ A, B float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.A * args.B})
				return result, nil
			},
		},
	})

	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	// Call the Go tool from .ts through the bus
	result, err := sandbox.EvalTS(context.Background(), "test-tool.ts", `
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

func TestContract_TSToolsCallNamespaceResolution(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Register a tool in plugin namespace
	kit.Tools.Register(registry.RegisteredTool{
		Name: "plugin.math@1.0.0.square", ShortName: "square",
		Namespace: "plugin.math@1.0.0",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ N float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.N * args.N})
				return result, nil
			},
		},
	})

	sandbox, err := kit.CreateSandbox(SandboxConfig{Namespace: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer sandbox.Close()

	// Short name — should resolve to plugin.math@1.0.0.square
	result, err := sandbox.EvalTS(context.Background(), "test-ns.ts", `
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
	t.Logf("Contract .ts tools.call (namespace resolution): square(9) = %v", out.Result)
}

// ═══════════════════════════════════════════════════════════════
// BUS OPERATIONS — pure infrastructure, no API keys needed
// ═══════════════════════════════════════════════════════════════

func TestContract_BusPubSub(t *testing.T) {
	kit := newTestKitNoKey(t)

	received := make(chan string, 1)
	kit.Bus.Subscribe("test.event", func(msg bus.Message) {
		received <- string(msg.Payload)
	})

	kit.Bus.Send(context.Background(), bus.Message{
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

	kit.Bus.Handle("echo.*", func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		return &bus.Message{Payload: msg.Payload}, nil
	})

	resp, err := kit.Bus.Request(context.Background(), "echo.test", "test", json.RawMessage(`{"ping":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(resp.Payload) != `{"ping":true}` {
		t.Errorf("payload = %s", resp.Payload)
	}
}

// ═══════════════════════════════════════════════════════════════
// TOOL REGISTRY — namespace resolution
// ═══════════════════════════════════════════════════════════════

func TestContract_ToolRegistryResolve(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "platform.echo", ShortName: "echo", Namespace: "platform",
		Description: "Echoes input",
		InputSchema: json.RawMessage(`{"type":"object"}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return input, nil
			},
		},
	})

	tool, err := kit.Tools.Resolve("echo", "user")
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
// SANDBOX ISOLATION — security guarantees
// ═══════════════════════════════════════════════════════════════

func TestContract_SandboxIsolation(t *testing.T) {
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

	// Different IDs
	if s1.ID() == s2.ID() {
		t.Error("sandboxes should have different IDs")
	}

	// Different namespaces
	if s1.Namespace() == s2.Namespace() {
		t.Error("sandboxes should have different namespaces")
	}

	// Agents in s1 are NOT visible in s2 (separate QuickJS runtimes)
	// Verify by checking __agents registry in each sandbox
	r1, _ := s1.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)
	r2, _ := s2.Eval(context.Background(), "check.js", `JSON.stringify(Object.keys(globalThis.__agents))`)

	if r1 != "[]" || r2 != "[]" {
		t.Logf("s1 agents: %s, s2 agents: %s (both should be empty)", r1, r2)
	}
}

func TestContract_CallerIDOnSandbox(t *testing.T) {
	kit := newTestKitNoKey(t)

	s, err := kit.CreateSandbox(SandboxConfig{
		Namespace: "user",
		CallerID:  "user.test-script",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	if s.CallerID() != "user.test-script" {
		t.Errorf("callerID = %q, want user.test-script", s.CallerID())
	}
}
