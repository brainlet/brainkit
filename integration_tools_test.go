//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

func TestIntegration_ZodViaImport(t *testing.T) {
	kit := newTestKitNoKey(t)

	result, err := kit.EvalModule(context.Background(), "test-zod.js", `
		import { z, createTool, output } from "kit";

		const schema = z.object({
			name: z.string().describe("person name"),
			age: z.number().describe("person age"),
			active: z.boolean().optional(),
		});

		const greetTool = createTool({
			id: "greet",
			description: "Greets a person",
			inputSchema: schema,
			execute: async (input) => {
				return { greeting: "Hello " + input.name + ", age " + input.age };
			},
		});

		output({
			hasSchema: !!schema,
			hasTool: !!greetTool,
			zType: typeof z.string,
		});
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		HasSchema bool   `json:"hasSchema"`
		HasTool   bool   `json:"hasTool"`
		ZType     string `json:"zType"`
	}
	json.Unmarshal([]byte(result), &out)

	if !out.HasSchema {
		t.Error("z.object() failed")
	}
	if !out.HasTool {
		t.Error("createTool with z schema failed")
	}
	if out.ZType != "function" {
		t.Errorf("z.string type = %q, want function", out.ZType)
	}
	t.Logf("Zod via import: schema=%v tool=%v z.string=%s", out.HasSchema, out.HasTool, out.ZType)
}

func TestIntegration_GoToolFromTSModule(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/uppercase", ShortName: "uppercase",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Converts text to uppercase",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"text to uppercase"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]string{"result": strings.ToUpper(args.Text)})
				return result, nil
			},
		},
	})

	result, err := kit.EvalModule(context.Background(), "test-go-tool.js", `
		import { tools, output } from "kit";

		const result = await tools.call("uppercase", { text: "hello brainlet" });
		output(result);
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct{ Result string }
	json.Unmarshal([]byte(result), &out)
	if out.Result != "HELLO BRAINLET" {
		t.Errorf("result = %q, want HELLO BRAINLET", out.Result)
	}
	t.Logf("Go tool from .ts module: %q", out.Result)
}

func TestIntegration_GoToolViaAgentFromTSModule(t *testing.T) {
	kit := newTestKit(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/double", ShortName: "double",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Doubles a number",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"n":{"type":"number","description":"number to double"}},"required":["n"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ N float64 }
				json.Unmarshal(input, &args)
				result, _ := json.Marshal(map[string]float64{"result": args.N * 2})
				return result, nil
			},
		},
	})

	result, err := kit.EvalModule(context.Background(), "test-agent-go-tool.js", `
		import { agent, tool, output } from "kit";

		const doubleTool = tool("double");
		const a = agent({
			model: "openai/gpt-4o-mini",
			instructions: "Always use the double tool. Return only the number.",
			tools: { double: doubleTool },
		});
		const r = await a.generate("What is 21 doubled? Use the double tool.");
		output({ text: r.text });
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct{ Text string }
	json.Unmarshal([]byte(result), &out)
	if !strings.Contains(out.Text, "42") {
		t.Errorf("expected 42: %q", out.Text)
	}
	t.Logf("Agent + Go tool via import: %q", out.Text)
}

func TestIntegration_TypedGoTool(t *testing.T) {
	kit := newTestKitNoKey(t)

	type AddInput struct {
		A float64 `json:"a" desc:"First number"`
		B float64 `json:"b" desc:"Second number"`
	}

	RegisterTool(kit, "brainlet/platform@1.0.0/add", registry.TypedTool[AddInput]{
		Description: "Adds two numbers",
		Execute: func(ctx context.Context, input AddInput) (any, error) {
			return map[string]any{"result": input.A + input.B}, nil
		},
	})

	result, err := kit.EvalModule(context.Background(), "typed-tool-test.js", `
		import { tools, output } from "kit";
		const result = await tools.call("add", { a: 17, b: 25 });
		output(result);
	`)
	if err != nil {
		t.Fatal(err)
	}

	var out struct{ Result float64 }
	json.Unmarshal([]byte(result), &out)

	if out.Result != 42 {
		t.Errorf("expected 42, got %v", out.Result)
	}
	t.Logf("typed Go tool: math.add(17, 25) = %v", out.Result)
}
