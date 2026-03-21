package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

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
