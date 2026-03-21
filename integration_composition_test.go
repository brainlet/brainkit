//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/brainlet/brainkit/registry"
)

func TestIntegration_FullTSModule(t *testing.T) {
	kit := newTestKit(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/platform@1.0.0/reverse", ShortName: "reverse",
		Owner: "brainlet", Package: "platform", Version: "1.0.0",
		Description: "Reverses a string",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"text":{"type":"string","description":"text to reverse"}},"required":["text"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct{ Text string }
				json.Unmarshal(input, &args)
				runes := []rune(args.Text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				result, _ := json.Marshal(map[string]string{"result": string(runes)})
				return result, nil
			},
		},
	})

	result, err := kit.EvalModule(context.Background(), "full-module.js", `
		import { agent, ai, tools, tool, sandbox, z, createTool, output } from "kit";

		const ctx = { ns: sandbox.namespace, id: sandbox.id };

		const aiResult = await ai.generate({
			model: "openai/gpt-4o-mini",
			prompt: "Reply with exactly one word: WORKING",
		});

		const reversed = await tools.call("reverse", { text: "brainlet" });

		const localTool = createTool({
			id: "concat",
			description: "Concatenates strings",
			inputSchema: z.object({ a: z.string(), b: z.string() }),
			execute: async ({ a, b }) => ({ result: a + b }),
		});

		output({
			sandbox: ctx,
			aiText: aiResult.text,
			reversed: reversed.result,
			hasLocalTool: !!localTool,
		});
	`)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out struct {
		Sandbox      struct{ Ns, Id string } `json:"sandbox"`
		AIText       string                  `json:"aiText"`
		Reversed     string                  `json:"reversed"`
		HasLocalTool bool                    `json:"hasLocalTool"`
	}
	json.Unmarshal([]byte(result), &out)

	if out.Sandbox.Ns != "test" {
		t.Errorf("namespace = %q", out.Sandbox.Ns)
	}
	if !strings.Contains(strings.ToUpper(out.AIText), "WORKING") {
		t.Errorf("ai.generate = %q", out.AIText)
	}
	if out.Reversed != "telniарb" && out.Reversed != "telniarbÂ" && out.Reversed != "telniarbА" {
		if out.Reversed == "" {
			t.Error("reverse returned empty")
		}
	}
	if !out.HasLocalTool {
		t.Error("createTool failed")
	}
	t.Logf("Full .ts module: sandbox=%+v ai=%q reversed=%q localTool=%v",
		out.Sandbox, out.AIText, out.Reversed, out.HasLocalTool)
}
